package v3

import (
	"github.com/docker/cagent/pkg/config/types"
	v2 "github.com/docker/cagent/pkg/config/v2"
)

func UpgradeFrom(old v2.Config) (Config, error) {
	var config Config
	types.CloneThroughJSON(old, &config)
	return config, nil
}
