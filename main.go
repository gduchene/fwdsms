// SPDX-FileCopyrightText: © 2020 Grégoire Duchêne <gduchene@awhk.org>
// SPDX-License-Identifier: ISC

package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"go.awhk.org/fwdsms/pkg/twilio"
	"go.awhk.org/gosdd"
)

var cfgFilename = flag.String("c", "/etc/fwdsms.yaml", "configuration file")

func main() {
	flag.Parse()
	log.SetFlags(0)
	fd, err := os.Open(*cfgFilename)
	if err != nil {
		log.Fatalf("Could not open the configuration file: %v.", err)
	}
	cfg, err := loadConfig(fd)
	if err != nil {
		log.Fatalf("Could not load the configuration: %v.", err)
	}
	if err := fd.Close(); err != nil {
		log.Printf("Failed to close configuration file: %v.", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	sms := make(chan twilio.SMS)

	r := mux.NewRouter()
	r.Path(cfg.Twilio.Endpoint).
		Methods(http.MethodPost).
		Handler(handlers.ProxyHeaders(&twilio.Filter{
			AuthToken: []byte(cfg.Twilio.AuthToken),
			Handler: &twilio.SMSTee{
				Chan:    sms,
				Handler: twilio.EmptyResponseHandler,
			},
		}))

	srv := http.Server{Handler: r}
	go func() {
		ln, err := listenSD()
		if err != nil {
			log.Fatalf("Failed to listen on systemd socket: %s.", err)
		}
		if ln == nil {
			ln = listenEnv(cfg)
		}
		if err = srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve HTTP: %s.", err)
		}
	}()

	mailer := newMailer(cfg, sms)
	ctx, cancel := context.WithCancel(context.Background())
	go mailer.start(ctx)

	<-done
	cancel()
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to properly shut down the HTTP server: %v.", err)
	}
}

func listenEnv(cfg *Config) net.Listener {
	if cfg.Twilio.Address != "" && cfg.Twilio.Address[0] == '/' {
		ln, err := net.Listen("unix", cfg.Twilio.Address)
		if err != nil {
			log.Fatalf("Could not set up UNIX listener: %s.", err)
		}
		if err := os.Chmod(cfg.Twilio.Address, 0666); err != nil {
			log.Fatalf("Could not set up permissions on UNIX socket: %s.", err)
		}
		return ln
	}
	ln, err := net.Listen("tcp", cfg.Twilio.Address)
	if err != nil {
		log.Fatalf("Could not set up TCP listener: %s.", err)
	}
	return ln
}

func listenSD() (net.Listener, error) {
	fds, err := gosdd.SDListenFDs(true)
	if err != nil {
		if err == gosdd.ErrNoSDSupport {
			return nil, nil
		}
		return nil, err
	}
	if len(fds) == 0 {
		return nil, nil
	}
	return net.FileListener(fds[0])
}
