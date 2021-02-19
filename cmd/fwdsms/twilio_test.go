// SPDX-FileCopyrightText: © 2020 Grégoire Duchêne <gduchene@awhk.org>
// SPDX-License-Identifier: ISC

package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var handler = newSMSHandler(&Config{
	Twilio: Twilio{
		AuthToken: "token",
		Endpoint:  "/endpoint",
	},
}, make(chan SMS))

func TestSMSHandler_checkRequestSignature_MissingHeader(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://example.com/endpoint", nil)
	assert.NoError(t, err)
	assert.EqualError(t, handler.checkRequestSignature(req), "missing X-Twilio-Signature header")
}

func TestSMSHandler_checkRequestSignature_SignatureBad(t *testing.T) {
	form := url.Values{}
	form.Set("foo", "bar")
	req, err := http.NewRequest(http.MethodPost, "https://example.com/endpoint", strings.NewReader(form.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Twilio-Signature", "a bad signature")
	assert.NoError(t, req.ParseForm())
	assert.EqualError(t, handler.checkRequestSignature(req), "bad X-Twilio-Signature header")
}

func TestSMSHandler_checkRequestSignature_SignatureMismatch(t *testing.T) {
	form := url.Values{}
	form.Set("foo", "bar")
	req, err := http.NewRequest(http.MethodPost, "https://example.com/endpoint", strings.NewReader(form.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Twilio-Signature", "DYIRnXpKIjrgAMxc0FD01B55+ag=")
	assert.NoError(t, req.ParseForm())
	assert.EqualError(t, handler.checkRequestSignature(req), "signature mismatch")
}

func TestSMSHandler_checkRequestSignature_SignatureGood(t *testing.T) {
	form := url.Values{}
	form.Set("foo", "bar")
	form.Set("bar", "baz")
	req, err := http.NewRequest(http.MethodPost, "/endpoint", strings.NewReader(form.Encode()))
	req.Host = "example.com"
	req.URL.Scheme = "https"
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Signature generated with:
	// % echo -n "https://example.com/endpointbarbazfoobar" | openssl dgst -binary -hmac "token" -sha1 | base64
	req.Header.Set("X-Twilio-Signature", "NpKVG88Z4y6ayJIxLJrzgEHeEwY=")
	assert.NoError(t, req.ParseForm())
	assert.NoError(t, handler.checkRequestSignature(req))
}
