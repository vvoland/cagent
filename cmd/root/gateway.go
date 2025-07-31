package root

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func addGatewayFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&runConfig.Gateway, "gateway", "", "Set the gateway address")

	persistentPreRunE := cmd.PersistentPreRunE
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Get gateway value from the environment.
		if gateway := os.Getenv("CAGENT_GATEWAY"); gateway != "" {
			runConfig.Gateway = gateway
		}

		// Ensure the gateway url is canonical.
		runConfig.Gateway = strings.TrimSpace(strings.TrimSuffix(runConfig.Gateway, "/"))

		if persistentPreRunE != nil {
			return persistentPreRunE(cmd, args)
		}
		return nil
	}
}
