package runtime

import (
	"context"
	"log/slog"

	"github.com/docker/docker-agent/pkg/agent"
	"github.com/docker/docker-agent/pkg/chat"
	"github.com/docker/docker-agent/pkg/compaction"
	"github.com/docker/docker-agent/pkg/model/provider"
	"github.com/docker/docker-agent/pkg/model/provider/options"
	"github.com/docker/docker-agent/pkg/session"
	"github.com/docker/docker-agent/pkg/team"
)

// runSummarization sends the prepared messages through a one-shot runtime
// and returns the model's summary together with the output token count and
// cost. The runtime is created with compaction disabled so it cannot recurse.
func runSummarization(ctx context.Context, model provider.Provider, messages []chat.Message) (compaction.Result, error) {
	summaryModel := provider.CloneWithOptions(ctx, model, options.WithStructuredOutput(nil))
	root := agent.New("root", compaction.SystemPrompt, agent.WithModel(summaryModel))
	t := team.New(team.WithAgents(root))

	sess := session.New()
	sess.Title = "Generating summary..."
	for _, msg := range messages {
		sess.AddMessage(&session.Message{Message: msg})
	}

	rt, err := New(t, WithSessionCompaction(false))
	if err != nil {
		return compaction.Result{}, err
	}
	if _, err = rt.Run(ctx, sess); err != nil {
		return compaction.Result{}, err
	}

	return compaction.Result{
		Summary:     sess.GetLastAssistantMessageContent(),
		InputTokens: sess.OutputTokens,
		Cost:        sess.TotalCost(),
	}, nil
}

// doCompact runs compaction on a session and applies the result (events,
// persistence, token count updates). The agent is used to extract the
// conversation from the session and to obtain the model for summarization.
func (r *LocalRuntime) doCompact(ctx context.Context, sess *session.Session, a *agent.Agent, additionalPrompt string, events chan Event) {
	slog.Debug("Generating summary for session", "session_id", sess.ID)

	events <- SessionCompaction(sess.ID, "started", a.Name())
	defer func() {
		events <- SessionCompaction(sess.ID, "completed", a.Name())
	}()

	messages := sess.GetMessages(a)
	if !compaction.HasConversationMessages(messages) {
		if additionalPrompt == "" {
			events <- Warning("Session is empty. Start a conversation before compacting.", a.Name())
		}
		return
	}

	prepared := compaction.BuildPrompt(messages, additionalPrompt)

	result, err := runSummarization(ctx, a.Model(), prepared)
	if err != nil {
		slog.Error("Failed to generate session summary", "error", err)
		events <- Error(err.Error())
		return
	}
	if result.Summary == "" {
		return
	}

	sess.Messages = append(sess.Messages, session.Item{Summary: result.Summary, Cost: result.Cost})
	sess.InputTokens = result.InputTokens
	sess.OutputTokens = 0

	_ = r.sessionStore.UpdateSession(ctx, sess)

	slog.Debug("Generated session summary", "session_id", sess.ID, "summary_length", len(result.Summary), "compaction_cost", result.Cost)
	events <- SessionSummary(sess.ID, result.Summary, a.Name())
}
