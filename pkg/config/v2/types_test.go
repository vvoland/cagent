package v2

import (
	"encoding/json"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/config/types"
)

func TestCommandsUnmarshal_Map(t *testing.T) {
	var c types.Commands
	input := []byte(`
df: "check disk"
ls: "list files"
`)
	err := yaml.Unmarshal(input, &c)
	require.NoError(t, err)
	require.Equal(t, "check disk", c["df"].Instruction)
	require.Equal(t, "list files", c["ls"].Instruction)
}

func TestCommandsUnmarshal_List(t *testing.T) {
	var c types.Commands
	input := []byte(`
- df: "check disk"
- ls: "list files"
`)
	err := yaml.Unmarshal(input, &c)
	require.NoError(t, err)
	require.Equal(t, "check disk", c["df"].Instruction)
	require.Equal(t, "list files", c["ls"].Instruction)
}

func TestCommandsUnmarshal_Advanced(t *testing.T) {
	var c types.Commands
	input := []byte(`
fix-lint:
  description: "Fix linting errors"
  instruction: "Fix the lint issues"
simple: "A simple command"
`)
	err := yaml.Unmarshal(input, &c)
	require.NoError(t, err)
	require.Equal(t, "Fix linting errors", c["fix-lint"].Description)
	require.Equal(t, "Fix the lint issues", c["fix-lint"].Instruction)
	require.Equal(t, "A simple command", c["simple"].Instruction)
	require.Empty(t, c["simple"].Description)
}

func TestCommandsDisplayText(t *testing.T) {
	simple := types.Command{Instruction: "simple instruction"}
	require.Equal(t, "simple instruction", simple.DisplayText())

	advanced := types.Command{Description: "my description", Instruction: "instruction"}
	require.Equal(t, "my description", advanced.DisplayText())
}

func TestRAGStrategyConfig_MarshalUnmarshal_FlattenedParams(t *testing.T) {
	t.Parallel()

	// Test that params are flattened during unmarshal and remain flattened after marshal
	input := []byte(`type: chunked-embeddings
model: embeddinggemma
database: ./rag/test.db
threshold: 0.5
vector_dimensions: 768
`)

	var strategy RAGStrategyConfig

	// Unmarshal
	err := yaml.Unmarshal(input, &strategy)
	require.NoError(t, err)
	require.Equal(t, "chunked-embeddings", strategy.Type)
	require.Equal(t, "./rag/test.db", mustGetDBString(t, strategy.Database))
	require.NotNil(t, strategy.Params)
	require.Equal(t, "embeddinggemma", strategy.Params["model"])
	require.InEpsilon(t, 0.5, strategy.Params["threshold"], 0.001)
	// YAML may unmarshal numbers as different numeric types (int, uint64, float64)
	require.InEpsilon(t, float64(768), toFloat64(strategy.Params["vector_dimensions"]), 0.001)

	// Marshal back
	output, err := yaml.Marshal(strategy)
	require.NoError(t, err)

	// Verify it's still flattened (no "params:" key)
	outputStr := string(output)
	require.Contains(t, outputStr, "type: chunked-embeddings")
	require.Contains(t, outputStr, "model: embeddinggemma")
	require.Contains(t, outputStr, "threshold: 0.5")
	require.Contains(t, outputStr, "vector_dimensions: 768")
	require.NotContains(t, outputStr, "params:")

	// Unmarshal again to verify round-trip
	var strategy2 RAGStrategyConfig
	err = yaml.Unmarshal(output, &strategy2)
	require.NoError(t, err)
	require.Equal(t, strategy.Type, strategy2.Type)
	require.Equal(t, strategy.Params["model"], strategy2.Params["model"])
	require.Equal(t, strategy.Params["threshold"], strategy2.Params["threshold"])
	// YAML may unmarshal numbers as different numeric types (int, uint64, float64)
	// Just verify the numeric value is correct
	require.InEpsilon(t, float64(768), toFloat64(strategy2.Params["vector_dimensions"]), 0.001)
}

func TestRAGStrategyConfig_MarshalUnmarshal_WithDatabase(t *testing.T) {
	t.Parallel()

	input := []byte(`type: chunked-embeddings
database: ./test.db
model: test-model
`)

	var strategy RAGStrategyConfig
	err := yaml.Unmarshal(input, &strategy)
	require.NoError(t, err)

	// Marshal back
	output, err := yaml.Marshal(strategy)
	require.NoError(t, err)

	// Should contain database as a simple string, not nested with sub-fields
	outputStr := string(output)
	require.Contains(t, outputStr, "database: ./test.db")
	require.NotContains(t, outputStr, "  value:") // Should not be nested with internal fields
	require.Contains(t, outputStr, "model: test-model")
	require.NotContains(t, outputStr, "params:") // Should be flattened
}

func TestRAGStrategyConfig_MarshalJSON(t *testing.T) {
	t.Parallel()

	// Create a strategy config with various fields
	strategy := RAGStrategyConfig{
		Type:  "chunked-embeddings",
		Docs:  []string{"doc1.md", "doc2.md"},
		Limit: 10,
		Params: map[string]any{
			"model":     "embedding-model",
			"threshold": 0.75,
		},
	}
	strategy.Database.value = "./test.db"

	// Marshal to JSON
	output, err := json.Marshal(strategy)
	require.NoError(t, err)

	// Verify JSON structure - params should be flattened
	outputStr := string(output)
	require.Contains(t, outputStr, `"type":"chunked-embeddings"`)
	require.Contains(t, outputStr, `"database":"./test.db"`)
	require.Contains(t, outputStr, `"model":"embedding-model"`)
	require.Contains(t, outputStr, `"threshold":0.75`)
	require.NotContains(t, outputStr, `"params"`)
}

func TestRAGStrategyConfig_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	input := []byte(`{
		"type": "bm25",
		"database": "./bm25.db",
		"limit": 20,
		"k1": 1.2,
		"b": 0.75
	}`)

	var strategy RAGStrategyConfig
	err := json.Unmarshal(input, &strategy)
	require.NoError(t, err)

	require.Equal(t, "bm25", strategy.Type)
	require.Equal(t, "./bm25.db", mustGetDBString(t, strategy.Database))
	require.Equal(t, 20, strategy.Limit)
	require.NotNil(t, strategy.Params)
	require.InEpsilon(t, 1.2, toFloat64(strategy.Params["k1"]), 0.001)
	require.InEpsilon(t, 0.75, toFloat64(strategy.Params["b"]), 0.001)
}

func TestRAGStrategyConfig_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	// Create original config
	original := RAGStrategyConfig{
		Type:  "chunked-embeddings",
		Docs:  []string{"readme.md"},
		Limit: 15,
		Params: map[string]any{
			"embedding_model":   "openai/text-embedding-3-small",
			"vector_dimensions": float64(1536),
			"threshold":         0.6,
		},
	}
	original.Database.value = "./vectors.db"

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal back
	var restored RAGStrategyConfig
	err = json.Unmarshal(jsonData, &restored)
	require.NoError(t, err)

	// Verify round-trip preserves data
	require.Equal(t, original.Type, restored.Type)
	require.Equal(t, original.Docs, restored.Docs)
	require.Equal(t, original.Limit, restored.Limit)
	require.Equal(t, mustGetDBString(t, original.Database), mustGetDBString(t, restored.Database))
	require.Equal(t, original.Params["embedding_model"], restored.Params["embedding_model"])
	require.Equal(t, original.Params["vector_dimensions"], restored.Params["vector_dimensions"])
	require.Equal(t, original.Params["threshold"], restored.Params["threshold"])
}

func mustGetDBString(t *testing.T, db RAGDatabaseConfig) string {
	t.Helper()
	str, err := db.AsString()
	require.NoError(t, err)
	return str
}

// toFloat64 converts various numeric types to float64 for comparison
func toFloat64(v any) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case uint64:
		return float64(val)
	case float64:
		return val
	case float32:
		return float64(val)
	default:
		return 0
	}
}
