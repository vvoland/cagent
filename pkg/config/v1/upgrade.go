package v1

import (
	"encoding/json"
	"fmt"

	v0 "github.com/docker/cagent/pkg/config/v0"
)

func UpgradeFrom(old v0.Config) Config {
	var config Config
	cloneThroughJSON(old, &config)

	for name, old_model := range old.Models {
		new_model := config.Models[name]
		new_model.Provider = old_model.Type
		config.Models[name] = new_model
	}

	return config
}

func cloneThroughJSON(oldConfig, newConfig any) {
	o, err := json.Marshal(oldConfig)
	if err != nil {
		panic(fmt.Sprintf("marshalling old: %v", err))
	}

	if err := json.Unmarshal(o, newConfig); err != nil {
		panic(fmt.Sprintf("unmarshalling new: %v", err))
	}
}
