package memory

import (
	"context"

	"github.com/rumpl/cagent/pkg/chat"
	"github.com/rumpl/cagent/pkg/memory/database"
	"github.com/rumpl/cagent/pkg/memorymanager"
	"github.com/rumpl/cagent/pkg/model/provider"
)

type Manager struct {
	db  database.Database
	llm provider.Provider
}

var _ memorymanager.Manager = (*Manager)(nil)

func NewManager(db database.Database, llm provider.Provider) *Manager {
	return &Manager{db: db, llm: llm}
}

func (m *Manager) AddMemory(ctx context.Context, memory database.UserMemory) error {
	return m.db.AddMemory(ctx, memory)
}

func (m *Manager) GetMemories(ctx context.Context) ([]database.UserMemory, error) {
	return m.db.GetMemories(ctx)
}

func (m *Manager) DeleteMemory(ctx context.Context, memory database.UserMemory) error {
	return m.db.DeleteMemory(ctx, memory)
}

func (m *Manager) SummarizeMemory(ctx context.Context, memory database.UserMemory) (string, error) {
	return m.llm.CreateChatCompletion(ctx, []chat.Message{
		{
			Role:    "system",
			Content: "You are a helpful assistant that summarizes memories.",
		},
	})
}
