// SPDX-FileCopyrightText: © 2021 Grégoire Duchêne <gduchene@awhk.org>
// SPDX-License-Identifier: ISC

package twilio

import (
	"fmt"
	"net/http"
)

// EmptyResponseHandler writes an empty XML response so Twilio knows not
// to do anything after a webhook has been called.
var EmptyResponseHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/xml")
	fmt.Fprint(w, "<Response/>")
})
