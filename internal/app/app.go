package app

import (
	"context"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
)

type App struct {
	agentFilename string
	runtime       *runtime.Runtime
	session       *session.Session
	events        chan tea.Msg
}

func New(agentFilename string, rt *runtime.Runtime, sess *session.Session) *App {
	return &App{
		agentFilename: agentFilename,
		runtime:       rt,
		session:       sess,
		events:        make(chan tea.Msg),
	}
}

// Run one agent loop
func (a *App) Run(ctx context.Context, message string) {
	go func() {
		a.session.AddMessage(session.UserMessage(a.agentFilename, message))
		for event := range a.runtime.RunStream(ctx, a.session) {
			var msg tea.Msg = event
			a.events <- msg
		}
	}()
}

func (a *App) Subscribe(ctx context.Context, program *tea.Program) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("TUI message handler shutting down")
			return
		case msg, ok := <-a.events:
			if !ok {
				slog.Debug("TUI message channel closed")
				return
			}
			program.Send(msg)
		}
	}
}

// Resume resumes the runtime with the given confirmation type
func (a *App) Resume(confirmationType string) {
	if a.runtime != nil {
		a.runtime.Resume(context.Background(), confirmationType)
	}
}
