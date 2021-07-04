// SPDX-FileCopyrightText: © 2020 Grégoire Duchêne <gduchene@awhk.org>
// SPDX-License-Identifier: ISC

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"text/template"
	"time"

	"go.awhk.org/fwdsms/pkg/twilio"
)

type email struct {
	from, to string
	body     []byte
}

type mailer struct {
	auth                      smtp.Auth
	hostname                  string
	sms                       <-chan twilio.SMS
	tmplFrom, tmplTo, tmplMsg *template.Template
}

func (m *mailer) sendEmail(e email) error {
	dialer := &net.Dialer{Timeout: time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", m.hostname, nil)
	if err != nil {
		return err
	}
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		log.Printf("Failed to set the SMTP connection deadline: %s.", err)
	}
	h, _, _ := net.SplitHostPort(m.hostname)
	c, err := smtp.NewClient(conn, h)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	if err := c.Auth(m.auth); err != nil {
		return err
	}

	if err := c.Mail(e.from); err != nil {
		return err
	}
	if err := c.Rcpt(e.to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(e.body); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return nil
	}
	return c.Quit()
}

func (m *mailer) newEmail(sms twilio.SMS) email {
	var from, to, msg bytes.Buffer
	if err := m.tmplFrom.Execute(&from, sms); err != nil {
		log.Printf("Failed to apply a template: %v.", err)
	}
	if err := m.tmplTo.Execute(&to, sms); err != nil {
		log.Printf("Failed to apply a template: %v.", err)
	}
	if err := m.tmplMsg.Execute(&msg, sms); err != nil {
		log.Printf("Failed to apply a template: %v.", err)
	}
	return email{from.String(), to.String(), msg.Bytes()}
}

func (m *mailer) start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case sms := <-m.sms:
			if err := m.sendEmail(m.newEmail(sms)); err != nil {
				log.Printf("Failed to send email: %v.", err)
			}
		}
	}
}

func newMailer(cfg *Config, sms <-chan twilio.SMS) *mailer {
	if cfg.Message.From == "" {
		log.Fatal("Missing From field.")
	}
	if cfg.Message.To == "" {
		log.Fatal("Missing To field.")
	}
	if cfg.Message.Subject == "" {
		log.Fatal("Missing Subject field.")
	}
	if cfg.Message.Template == "" {
		log.Fatal("Missing Template field.")
	}
	tmplMsg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n",
		cfg.Message.From, cfg.Message.To, cfg.Message.Subject, cfg.Message.Template)
	host, _, _ := net.SplitHostPort(cfg.SMTP.Address)
	return &mailer{
		auth:     smtp.PlainAuth("", cfg.SMTP.Username, cfg.SMTP.Password, host),
		hostname: cfg.SMTP.Address,
		sms:      sms,
		tmplFrom: template.Must(template.New("from").Parse(cfg.Message.From)),
		tmplTo:   template.Must(template.New("to").Parse(cfg.Message.To)),
		tmplMsg:  template.Must(template.New("message").Parse(tmplMsg)),
	}
}
