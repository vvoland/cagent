package v2

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

func Load(data []byte) (Config, error) {
	var cfg Config

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	err := decoder.Decode(&cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}
