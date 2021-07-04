// SPDX-FileCopyrightText: © 2021 Grégoire Duchêne <gduchene@awhk.org>
// SPDX-License-Identifier: ISC

package twilio

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
)

var (
	ErrBase64            = errors.New("failed to decode X-Twilio-Signature header")
	ErrMissingHeader     = errors.New("missing X-Twilio-Signature header")
	ErrSignatureMismatch = errors.New("signature mismatch")
)

type Filter struct {
	AuthToken []byte
	Handler   http.Handler
}

var _ http.Handler = &Filter{}

func (th *Filter) CheckRequestSignature(r *http.Request) error {
	hdr := r.Header.Get("X-Twilio-Signature")
	if len(hdr) == 0 {
		return ErrMissingHeader
	}
	reqSig, err := base64.StdEncoding.DecodeString(hdr)
	if err != nil {
		return ErrBase64
	}

	// See https://www.twilio.com/docs/usage/security#validating-requests
	// for more details.

	parts := []string{}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			return err
		}
		for k := range r.PostForm {
			parts = append(parts, k)
		}
		sort.Strings(parts)
		for i, k := range parts {
			parts[i] += r.PostForm[k][0]
		}
	}
	s := r.URL.String() + strings.Join(parts, "")
	h := hmac.New(sha1.New, th.AuthToken)
	if _, err := h.Write([]byte(s)); err != nil {
		return fmt.Errorf("failed to write bytes to calculate signature: %s", err)
	}
	ourSig := h.Sum(nil)

	if !hmac.Equal(reqSig, ourSig) {
		return ErrSignatureMismatch
	}
	return nil
}

func (th *Filter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := th.CheckRequestSignature(r); err != nil {
		log.Println("Failed to check Twilio signature:", err)
		if err == ErrSignatureMismatch {
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		return
	}
	th.Handler.ServeHTTP(w, r)
}
