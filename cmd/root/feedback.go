package root

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/feedback"
	"github.com/docker/cagent/pkg/telemetry"
)

// NewFeedbackCmd creates a new feedback command
func NewFeedbackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "feedback",
		Short: "Send feedback about cagent",
		Long:  `Submit feedback or report issues with cagent`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			telemetry.TrackCommand("feedback", args)
			fmt.Println("Feel free to give feedback:\n", feedback.FeedbackLink)
		},
	}
}
