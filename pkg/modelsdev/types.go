package modelsdev

import "time"

// Database represents the complete models.dev database
type Database struct {
	Providers map[string]Provider `json:"providers"`
	UpdatedAt time.Time           `json:"updated_at"`
}

// Provider represents an AI model provider
type Provider struct {
	ID     string           `json:"id"`
	Name   string           `json:"name"`
	Doc    string           `json:"doc,omitempty"`
	API    string           `json:"api,omitempty"`
	NPM    string           `json:"npm,omitempty"`
	Env    []string         `json:"env,omitempty"`
	Models map[string]Model `json:"models"`
}

// Model represents an AI model with its specifications and capabilities
type Model struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Family      string     `json:"family,omitempty"`
	Attachment  bool       `json:"attachment"`
	Reasoning   bool       `json:"reasoning"`
	Temperature bool       `json:"temperature"`
	ToolCall    bool       `json:"tool_call"`
	Knowledge   string     `json:"knowledge,omitempty"`
	ReleaseDate string     `json:"release_date"`
	LastUpdated string     `json:"last_updated"`
	OpenWeights bool       `json:"open_weights"`
	Cost        *Cost      `json:"cost,omitempty"`
	Limit       Limit      `json:"limit"`
	Modalities  Modalities `json:"modalities"`
}

// Cost represents the pricing information for a model
type Cost struct {
	Input      float64 `json:"input,omitempty"`
	Output     float64 `json:"output,omitempty"`
	CacheRead  float64 `json:"cache_read,omitempty"`
	CacheWrite float64 `json:"cache_write,omitempty"`
}

// Limit represents the context and output limitations of a model
type Limit struct {
	Context int   `json:"context"`
	Output  int64 `json:"output"`
}

// Modalities represents the supported input and output types
type Modalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

// CachedData represents the cached models.dev data with metadata
type CachedData struct {
	Database    Database  `json:"database"`
	CachedAt    time.Time `json:"cached_at"`
	LastRefresh time.Time `json:"last_refresh"`
}
