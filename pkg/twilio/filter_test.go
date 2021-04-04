// SPDX-FileCopyrightText: © 2021 Grégoire Duchêne <gduchene@awhk.org>
// SPDX-License-Identifier: ISC

package twilio

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilter_CheckRequestSignature(t *testing.T) {
	th := &Filter{[]byte("token"), EmptyResponseHandler}

	t.Run("Good Signature (POST)", func(t *testing.T) {
		assert.NoError(t, th.CheckRequestSignature(newRequest(Post)))
	})

	t.Run("Good Signature (GET)", func(t *testing.T) {
		assert.NoError(t, th.CheckRequestSignature(newRequest(Get)))
	})

	t.Run("Missing Header", func(t *testing.T) {
		r := newRequest(Post)
		r.Header.Del("X-Twilio-Signature")
		assert.ErrorIs(t, th.CheckRequestSignature(r), ErrMissingHeader)
	})

	t.Run("Bad Base64", func(t *testing.T) {
		r := newRequest(Post)
		r.Header.Set("X-Twilio-Signature", "Very suspicious Base64 header.")
		assert.ErrorIs(t, th.CheckRequestSignature(r), ErrBase64)
	})

	t.Run("Signature Mismatch", func(t *testing.T) {
		r := newRequest(Post)
		r.Header.Set("X-Twilio-Signature", "dpE7iSS3LEQo72hCT34eBRt3UEI=")
		assert.ErrorIs(t, th.CheckRequestSignature(r), ErrSignatureMismatch)
	})
}

func TestFilter_ServeHTTP(t *testing.T) {
	th := &Filter{[]byte("token"), EmptyResponseHandler}

	t.Run("Good Signature (POST)", func(t *testing.T) {
		w := httptest.NewRecorder()
		th.ServeHTTP(w, newRequest(Post))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/xml", w.HeaderMap.Get("Content-Type"))
		assert.Equal(t, "<Response/>", w.Body.String())
	})

	t.Run("Good Signature (GET)", func(t *testing.T) {
		w := httptest.NewRecorder()
		th.ServeHTTP(w, newRequest(Get))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/xml", w.HeaderMap.Get("Content-Type"))
		assert.Equal(t, "<Response/>", w.Body.String())
	})

	t.Run("Missing Header", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := newRequest(Post)
		r.Header.Del("X-Twilio-Signature")
		th.ServeHTTP(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Bad Base64", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := newRequest(Post)
		r.Header.Set("X-Twilio-Signature", "Very suspicious Base64 header.")
		th.ServeHTTP(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Signature Mismatch", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := newRequest(Post)
		r.Header.Set("X-Twilio-Signature", "dpE7iSS3LEQo72hCT34eBRt3UEI=")
		th.ServeHTTP(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

const (
	Get  = true
	Post = false
)

// X-Twilio-Signature can be manually generated with:
// % echo -n "${SOME_STRING}" | openssl dgst -binary -hmac ${AUTH_TOKEN} -sha1 | base64

func newRequest(get bool) *http.Request {
	vals := url.Values{
		"To":   {"Bob"},
		"From": {"Alice"},
		"Body": {"A random message."},
	}.Encode()
	if get {
		r := httptest.NewRequest(http.MethodGet, "https://example.test/endpoint?"+vals, nil)
		r.Header.Set("X-Twilio-Signature", "Hh0ReTk/+7Ea38qZ3Xt1/NQx4i4=")
		return r
	}
	rd := strings.NewReader(vals)
	r := httptest.NewRequest(http.MethodPost, "https://example.test/endpoint", rd)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("X-Twilio-Signature", "j61PPnnoUAAsfEnLuwUefOfylf4=")
	return r
}
