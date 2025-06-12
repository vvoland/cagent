package database

import (
	"context"
	"errors"
)

var (
	ErrEmptyID = errors.New("memory ID cannot be empty")
)

type UserMemory struct {
	ID        string
	CreatedAt string
	Memory    string
}

type Memory interface {
	AddMemory(ctx context.Context, memory UserMemory) error
	GetMemories(ctx context.Context) ([]UserMemory, error)
	DeleteMemory(ctx context.Context, memory UserMemory) error
}

type Database interface {
	AddMemory(ctx context.Context, memory UserMemory) error
	GetMemories(ctx context.Context) ([]UserMemory, error)
	DeleteMemory(ctx context.Context, memory UserMemory) error
}
