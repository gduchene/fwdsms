// SPDX-FileCopyrightText: © 2020 Grégoire Duchêne <gduchene@awhk.org>
// SPDX-License-Identifier: ISC

package main

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.awhk.org/pipeln"
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

func TestSMSHandler_EndToEnd(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/endpoint", handlers.ProxyHeaders(handler))
	srv := http.Server{Handler: mux}
	ln := pipeln.New("localhost.test:80")
	go srv.Serve(ln)
	defer srv.Close()

	client := http.Client{Transport: &http.Transport{Dial: ln.Dial}}
	form := url.Values{}
	form.Set("From", "Foo")
	form.Set("To", "Bar")
	form.Set("Body", "Test")
	req, err := http.NewRequest(http.MethodPost, "http://localhost.test:80/endpoint", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-Scheme", "http")

	t.Run("Bad HTTP Method", func(t *testing.T) {
		resp, err := client.Head("http://localhost.test:80/endpoint")
		require.NoError(t, err)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("Bad Signature", func(t *testing.T) {
		req.Header.Set("X-Twilio-Signature", "DYIRnXpKIjrgAMxc0FD01B55+ag=")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Good Signature", func(t *testing.T) {
		done := make(chan struct{})

		go func() {
			defer close(done)
			select {
			case sms := <-handler.sms:
				assert.Equal(t, "Foo", sms.From)
				assert.Equal(t, "Bar", sms.To)
				assert.Equal(t, "Test", sms.Message)
			case <-time.After(time.Second):
				t.Error("Timed out while waiting on handler.sms.")
			}
		}()

		// Signature generated with:
		// % echo -n "http://localhost.test:80/endpointBodyTestFromFooToBar" | openssl dgst -binary -hmac "token" -sha1 | base64
		req.Header.Set("X-Twilio-Signature", "iiifXqv3dP5j8Oj5eB4RAOm/3tI=")
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/xml", resp.Header.Get("Content-Type"))
		assert.Equal(t, "<Response/>", string(body))
		<-done
	})
}
