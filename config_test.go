package main

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
)

//go:embed config_example.yaml
var cfg []byte

func TestConfig(t *testing.T) {
	_, err := loadConfig(bytes.NewReader(cfg))
	assert.NoError(t, err)
}
