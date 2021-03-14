// SPDX-FileCopyrightText: © 2021 Grégoire Duchêne <gduchene@awhk.org>
// SPDX-License-Identifier: ISC

package twilio

import (
	"net/http"
	"time"
)

type SMS struct {
	DateReceived   time.Time
	From, To, Body string
}

type SMSTee struct {
	Chan    chan<- SMS
	Handler http.Handler
}

var _ http.Handler = &SMSTee{}

func (th *SMSTee) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	select {
	case th.Chan <- SMS{
		DateReceived: time.Now(),
		From:         r.FormValue("From"),
		To:           r.FormValue("To"),
		Body:         r.FormValue("Body"),
	}:
		th.Handler.ServeHTTP(w, r)
	case <-r.Context().Done():
	}
}
