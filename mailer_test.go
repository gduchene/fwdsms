// SPDX-FileCopyrightText: © 2020 Grégoire Duchêne <gduchene@awhk.org>
// SPDX-License-Identifier: ISC

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"go.awhk.org/core"
	"go.awhk.org/fwdsms/pkg/twilio"
)

func TestMailer_newEmail(s *testing.T) {
	t := core.T{T: s, Options: []cmp.Option{cmp.AllowUnexported(email{})}}

	m := newMailer(&Config{
		Message: Message{
			From:    "fwdsms@example.com",
			To:      "sms{{.To}}@example.com",
			Subject: "New SMS From {{.From}}",
			Template: `From: {{.From}}
  To: {{.To}}
Date: {{.DateReceived.UTC}}

{{.Body}}`,
		}}, nil)
	// Reserved phone numbers, see Ofcom's website.
	sms := twilio.SMS{
		DateReceived: time.Unix(0, 0),
		From:         "+442079460123",
		To:           "+447700900123",
		Body:         "Hello World!",
	}
	wants := email{
		from: "fwdsms@example.com",
		to:   "sms+447700900123@example.com",
		body: []byte(strings.Join([]string{
			"From: fwdsms@example.com",
			"To: sms+447700900123@example.com",
			"Subject: New SMS From +442079460123",
			"",
			`From: +442079460123
  To: +447700900123
Date: 1970-01-01 00:00:00 +0000 UTC

Hello World!`,
			"",
		}, "\r\n")),
	}
	t.AssertEqual(wants, m.newEmail(sms))
}
