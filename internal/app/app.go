package app

import (
	"context"

	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
)

type App struct {
	agentFilename string
	runtime       *runtime.Runtime
	session       *session.Session
	firstMessage  *string
	events        chan tea.Msg
}

func New(agentFilename string, rt *runtime.Runtime, sess *session.Session, firstMessage *string) *App {
	return &App{
		agentFilename: agentFilename,
		runtime:       rt,
		session:       sess,
		firstMessage:  firstMessage,
		events:        make(chan tea.Msg, 128),
	}
}

func (a *App) FirstMessage() *string {
	return a.firstMessage
}

// Run one agent loop
func (a *App) Run(ctx context.Context, message string) {
	go func() {
		a.session.AddMessage(session.UserMessage(a.agentFilename, message))
		for event := range a.runtime.RunStream(ctx, a.session) {
			a.events <- event
		}
	}()
}

func (a *App) Subscribe(ctx context.Context, program *tea.Program) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-a.events:
			if !ok {
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
