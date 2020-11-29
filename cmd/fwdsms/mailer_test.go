// SPDX-FileCopyrightText: © 2020 Grégoire Duchêne <gduchene@awhk.org>
// SPDX-License-Identifier: ISC

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMailer_newEmail(t *testing.T) {
	m := newMailer(&Config{
		Message: Message{
			From:    "fwdsms@example.com",
			To:      "sms{{.To}}@example.com",
			Subject: "New SMS From {{.From}}",
			Template: `From: {{.From}}
  To: {{.To}}
Date: {{.Date.UTC}}

{{.Message}}`,
		}}, nil)
	// Reserved phone numbers, see Ofcom's website.
	sms := SMS{time.Unix(0, 0), "+442079460123", "+447700900123", "Hello World!"}
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
	assert.Equal(t, wants, m.newEmail(sms))
}
