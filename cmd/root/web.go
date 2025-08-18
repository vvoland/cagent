package root

import (
	"github.com/spf13/cobra"
)

// NewWebCmd creates a new web command
func NewWebCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "web <agent-file>|<agents-dir>",
		Short:   "Start a web server",
		Long:    `Start a web server that exposes the agents via an HTTP API`,
		Example: `cagent web /path/to/agents --listen :8080`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHttp(cmd, true, true, args)
		},
	}

	cmd.PersistentFlags().StringVarP(&listenAddr, "listen", "l", ":8080", "Address to listen on")
	cmd.PersistentFlags().StringVarP(&sessionDb, "session-db", "s", "session.db", "Path to the session database")
	cmd.PersistentFlags().StringSliceVar(&runConfig.EnvFiles, "env-from-file", nil, "Set environment variables from file")
	addGatewayFlags(cmd)

	return cmd
}
