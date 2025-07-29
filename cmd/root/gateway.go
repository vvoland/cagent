package root

import (
	"strings"

	"github.com/spf13/cobra"
)

func addGatewayFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&runConfig.Gateway, "gateway", "", "Set the gateway address")

	persistentPreRunE := cmd.PersistentPreRunE
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Ensure the gateway url is canonical.
		runConfig.Gateway = strings.TrimSpace(strings.TrimSuffix(runConfig.Gateway, "/"))

		if persistentPreRunE != nil {
			return persistentPreRunE(cmd, args)
		}
		return nil
	}
}
