package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const cfg string = `
message:
  from: foo@example.com
  to: bar@example.com
  subject: New SMS From {{.From}} For {{.To}}
  template: |
    From: {{.From}}
      To: {{.To}}
    Date: {{.DateReceived.UTC}}

    {{.Message}}

smtp:
  hostname: example.com:465
  username: bar
  password: some password

twilio:
  address: /run/fwdsms/socket
  authToken: some token
  endpoint: /
`

func TestConfig(t *testing.T) {
	_, err := loadConfig(strings.NewReader(cfg))
	assert.NoError(t, err)
}
