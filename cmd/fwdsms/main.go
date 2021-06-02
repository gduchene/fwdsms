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
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"golang.org/x/sys/unix"

	"go.awhk.org/fwdsms/pkg/twilio"
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

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, unix.SIGTERM)

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
		var (
			l   net.Listener
			err error
		)
		if cfg.Twilio.Address != "" && cfg.Twilio.Address[0] == '/' {
			if l, err = net.Listen("unix", cfg.Twilio.Address); err != nil {
				log.Fatalf("Could not set up UNIX listener: %v.", err)
			}
			if err = os.Chmod(cfg.Twilio.Address, 0666); err != nil {
				log.Fatalf("Could not set up permissions on UNIX socket: %v.", err)
			}
		} else {
			if cfg.Twilio.Address == "" {
				cfg.Twilio.Address = ":8080"
			}
			if l, err = net.Listen("tcp", cfg.Twilio.Address); err != nil {
				log.Fatalf("Could not set up TCP listener: %v.", err)
			}
		}
		if err = srv.Serve(l); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve HTTP: %v.", err)
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
