package main

import (
	"bytes"
	_ "embed"
	"testing"

	"go.awhk.org/core"
)

//go:embed config_example.yaml
var cfg []byte

func TestConfig(s *testing.T) {
	t := core.T{T: s}

	_, err := loadConfig(bytes.NewReader(cfg))
	t.AssertErrorIs(nil, err)
}
