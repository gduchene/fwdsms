// SPDX-FileCopyrightText: © 2020 Grégoire Duchêne <gduchene@awhk.org>
// SPDX-License-Identifier: ISC

package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

type SMS struct {
	Date              time.Time
	From, To, Message string
}

type smsHandler struct {
	hash hash.Hash
	sms  chan SMS
}

var _ http.Handler = &smsHandler{}

func (h *smsHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		resp.Header().Set("Allow", http.MethodPost)
		resp.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := h.checkRequestSignature(req); err != nil {
		log.Printf("Failed to check the request signature: %v.", err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	from, ok := req.PostForm["From"]
	if !ok || len(from) == 0 {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	to, ok := req.PostForm["To"]
	if !ok || len(to) == 0 {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	msg, ok := req.PostForm["Body"]
	if !ok || len(msg) == 0 {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	resp.Header().Set("Content-Type", "text/xml")
	fmt.Fprintf(resp, "<Response/>")
	h.sms <- SMS{time.Now(), from[0], to[0], msg[0]}
}

func (h *smsHandler) checkRequestSignature(req *http.Request) error {
	reqSig, err := func() ([]byte, error) {
		h := req.Header.Get("X-Twilio-Signature")
		if len(h) == 0 {
			return nil, errors.New("missing X-Twilio-Signature header")
		}
		b, err := base64.StdEncoding.DecodeString(h)
		if err != nil {
			return nil, errors.New("bad X-Twilio-Signature header")
		}
		return b, nil
	}()
	if err != nil {
		return err
	}

	if err := req.ParseForm(); err != nil {
		return err
	}
	ourSig := func() []byte {
		defer h.hash.Reset()
		parts := []string{}
		for k := range req.PostForm {
			parts = append(parts, k)
		}
		sort.Strings(parts)
		for i := range parts {
			parts[i] += req.PostForm[parts[i]][0]
		}
		blob := req.Host + req.URL.Path + strings.Join(parts, "")
		if req.URL.Scheme != "" {
			blob = fmt.Sprintf("%s://%s", req.URL.Scheme, blob)
		}
		h.hash.Write([]byte(blob))
		return h.hash.Sum(nil)
	}()

	if !hmac.Equal(ourSig, reqSig) {
		return errors.New("signature mismatch")
	}
	return nil
}

func newSMSHandler(cfg *Config, sms chan SMS) *smsHandler {
	if cfg.Twilio.AuthToken == "" {
		log.Fatal("Twilio auth token unspecified.")
	}
	if cfg.Twilio.Endpoint == "" {
		log.Fatal("Twilio endpoint unspecified.")
	}
	return &smsHandler{hmac.New(sha1.New, []byte(cfg.Twilio.AuthToken)), sms}
}
