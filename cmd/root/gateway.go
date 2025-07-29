package root

import "github.com/spf13/cobra"

func addGatewayFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&runConfig.Gateway, "gateway", "", "Set the gateway address")
}
