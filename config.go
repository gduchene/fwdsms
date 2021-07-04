// SPDX-FileCopyrightText: © 2020 Grégoire Duchêne <gduchene@awhk.org>
// SPDX-License-Identifier: ISC

package main

import (
	"io"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Message Message `yaml:"message"`
	SMTP    SMTP    `yaml:"smtp"`
	Twilio  Twilio  `yaml:"twilio"`
}

type Message struct {
	From     string `yaml:"from"`
	To       string `yaml:"to"`
	Subject  string `yaml:"subject"`
	Template string `yaml:"template"`
}

type SMTP struct {
	Address  string `yaml:"hostname"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Twilio struct {
	Address   string `yaml:"address"`
	AuthToken string `yaml:"authToken"`
	Endpoint  string `yaml:"endpoint"`
}

func loadConfig(r io.Reader) (*Config, error) {
	dec := yaml.NewDecoder(r)
	cfg := &Config{}
	if err := dec.Decode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
