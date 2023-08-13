// SPDX-FileCopyrightText: © 2021 Grégoire Duchêne <gduchene@awhk.org>
// SPDX-License-Identifier: ISC

package twilio

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"go.awhk.org/core"
)

func TestFilter_CheckRequestSignature(s *testing.T) {
	t := core.T{T: s}

	th := &Filter{[]byte("token"), EmptyResponseHandler}

	t.Run("Good Signature (POST)", func(t *core.T) {
		t.AssertErrorIs(nil, th.CheckRequestSignature(newRequest(Post)))
	})

	t.Run("Good Signature (GET)", func(t *core.T) {
		t.AssertErrorIs(nil, th.CheckRequestSignature(newRequest(Get)))
	})

	t.Run("Missing Header", func(t *core.T) {
		r := newRequest(Post)
		r.Header.Del("X-Twilio-Signature")
		t.AssertErrorIs(ErrMissingHeader, th.CheckRequestSignature(r))
	})

	t.Run("Bad Base64", func(t *core.T) {
		r := newRequest(Post)
		r.Header.Set("X-Twilio-Signature", "Very suspicious Base64 header.")
		t.AssertErrorIs(ErrBase64, th.CheckRequestSignature(r))
	})

	t.Run("Signature Mismatch", func(t *core.T) {
		r := newRequest(Post)
		r.Header.Set("X-Twilio-Signature", "dpE7iSS3LEQo72hCT34eBRt3UEI=")
		t.AssertErrorIs(ErrSignatureMismatch, th.CheckRequestSignature(r))
	})
}

func TestFilter_ServeHTTP(s *testing.T) {
	t := core.T{T: s}

	th := &Filter{[]byte("token"), EmptyResponseHandler}

	t.Run("Good Signature (POST)", func(t *core.T) {
		w := httptest.NewRecorder()
		th.ServeHTTP(w, newRequest(Post))
		t.AssertEqual(http.StatusOK, w.Code)
		t.AssertEqual("text/xml", w.Result().Header.Get("Content-Type"))
		t.AssertEqual("<Response/>", w.Body.String())
	})

	t.Run("Good Signature (GET)", func(t *core.T) {
		w := httptest.NewRecorder()
		th.ServeHTTP(w, newRequest(Get))
		t.AssertEqual(http.StatusOK, w.Code)
		t.AssertEqual("text/xml", w.Result().Header.Get("Content-Type"))
		t.AssertEqual("<Response/>", w.Body.String())
	})

	t.Run("Missing Header", func(t *core.T) {
		w := httptest.NewRecorder()
		r := newRequest(Post)
		r.Header.Del("X-Twilio-Signature")
		th.ServeHTTP(w, r)
		t.AssertEqual(http.StatusBadRequest, w.Code)
	})

	t.Run("Bad Base64", func(t *core.T) {
		w := httptest.NewRecorder()
		r := newRequest(Post)
		r.Header.Set("X-Twilio-Signature", "Very suspicious Base64 header.")
		th.ServeHTTP(w, r)
		t.AssertEqual(http.StatusBadRequest, w.Code)
	})

	t.Run("Signature Mismatch", func(t *core.T) {
		w := httptest.NewRecorder()
		r := newRequest(Post)
		r.Header.Set("X-Twilio-Signature", "dpE7iSS3LEQo72hCT34eBRt3UEI=")
		th.ServeHTTP(w, r)
		t.AssertEqual(http.StatusForbidden, w.Code)
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
