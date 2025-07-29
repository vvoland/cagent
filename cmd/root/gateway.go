package root

import (
	"errors"
	"strings"

	"github.com/docker/cagent/pkg/desktop"
	"github.com/spf13/cobra"
)

func addGatewayFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&runConfig.Gateway, "gateway", "", "Set the gateway address")

	persistentPreRunE := cmd.PersistentPreRunE
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		runConfig.Gateway = strings.TrimSpace(runConfig.Gateway)

		if strings.HasSuffix(runConfig.Gateway, "dckr.io") {
			if !desktop.IsLoggedIn(cmd.Context()) {
				return errors.New("Sorry, you first need to sign in Docker Desktop to use the AI Gateway")
			}
		}

		if persistentPreRunE != nil {
			return persistentPreRunE(cmd, args)
		}
		return nil
	}
}
