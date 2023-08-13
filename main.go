// SPDX-FileCopyrightText: © 2020 Grégoire Duchêne <gduchene@awhk.org>
// SPDX-License-Identifier: ISC

package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.awhk.org/core"
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
	if err := fd.Close(); err != nil {
		log.Printf("Failed to close configuration file: %v.", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	sms := make(chan twilio.SMS)

	h := core.FilteringHTTPHandler(&twilio.Filter{
		AuthToken: []byte(cfg.Twilio.AuthToken),
		Handler: &twilio.SMSTee{
			Chan:    sms,
			Handler: twilio.EmptyResponseHandler,
		},
	}, core.FilterHTTPMethod(http.MethodPost))
	m := http.NewServeMux()
	m.Handle(cfg.Twilio.Endpoint, h)

	srv := http.Server{Handler: m}
	go func() {
		if err = srv.Serve(core.Must(core.Listen(cfg.Twilio.Address))); err != nil && err != http.ErrServerClosed {
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
