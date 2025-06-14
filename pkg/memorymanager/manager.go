package memorymanager

import (
	"context"

	"github.com/rumpl/cagent/pkg/memory/database"
)

type Manager interface {
	AddMemory(ctx context.Context, memory database.UserMemory) error
	GetMemories(ctx context.Context) ([]database.UserMemory, error)
	DeleteMemory(ctx context.Context, memory database.UserMemory) error
}
