package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/docker/cagent/pkg/rag"
	"github.com/docker/cagent/pkg/tools"
)

const (
	ToolNameQueryRAG = "query_documents"
)

// RAGTool provides document querying capabilities
type RAGTool struct {
	tools.ElicitationTool
	managers map[string]*rag.Manager
}

var _ tools.ToolSet = (*RAGTool)(nil)

// NewRAGTool creates a new RAG tool with the given managers
func NewRAGTool(managers map[string]*rag.Manager) *RAGTool {
	return &RAGTool{
		managers: managers,
	}
}

type QueryRAGArgs struct {
	Query  string `json:"query"  jsonschema:"The search query to find relevant documents"`
	Source string `json:"source,omitempty" jsonschema:"Optional name of a specific knowledge base to query; if omitted, searches all knowledge bases"`
}

type QueryResult struct {
	SourcePath string  `json:"source_path" jsonschema:"Path to the source document"`
	Content    string  `json:"content" jsonschema:"Relevant document chunk content"`
	Similarity float64 `json:"similarity" jsonschema:"Similarity score (0-1)"`
	ChunkIndex int     `json:"chunk_index" jsonschema:"Index of the chunk within the source document"`
}

func (t *RAGTool) Instructions() string {
	if len(t.managers) == 0 {
		return ""
	}

	instruction := "## Document Knowledge Bases\n\nYou have access to the following document knowledge bases:\n\n"
	for name, mgr := range t.managers {
		instruction += fmt.Sprintf("- **%s**: %s\n", name, mgr.Description())
	}
	instruction += "\nUse the `query_documents` tool to search these knowledge bases for relevant information.\n"
	instruction += "You can optionally set the `source` argument to one of the names above to query a single knowledge base; "
	instruction += "leave `source` empty to search across all knowledge bases."

	return instruction
}

func (t *RAGTool) Tools(context.Context) ([]tools.Tool, error) {
	return []tools.Tool{
		{
			Name:         ToolNameQueryRAG,
			Category:     "knowledge",
			Description:  "Search document knowledge bases for relevant information. Returns the most relevant document chunks based on semantic similarity.",
			Parameters:   tools.MustSchemaFor[QueryRAGArgs](),
			OutputSchema: tools.MustSchemaFor[[]QueryResult](),
			Handler:      t.handleQueryRAG,
			Annotations: tools.ToolAnnotations{
				ReadOnlyHint: true,
				Title:        "Query Documents",
			},
		},
	}, nil
}

func (t *RAGTool) handleQueryRAG(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var args QueryRAGArgs
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// If a specific source is provided, restrict the query to that knowledge base
	if args.Source != "" {
		mgr, ok := t.managers[args.Source]
		if !ok {
			return nil, fmt.Errorf("unknown source %q; must be one of the configured RAG sources", args.Source)
		}

		results, err := mgr.Query(ctx, args.Query)
		if err != nil {
			return nil, fmt.Errorf("RAG query failed for source %q: %w", args.Source, err)
		}

		allResults := make([]QueryResult, 0, len(results))
		for _, result := range results {
			allResults = append(allResults, QueryResult{
				SourcePath: result.Document.SourcePath,
				Content:    result.Document.Content,
				Similarity: result.Similarity,
				ChunkIndex: result.Document.ChunkIndex,
			})
		}

		sortResults(allResults)

		maxResults := 10
		if len(allResults) > maxResults {
			allResults = allResults[:maxResults]
		}

		resultJSON, err := json.Marshal(allResults)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal results: %w", err)
		}

		return &tools.ToolCallResult{
			Output: string(resultJSON),
		}, nil
	}

	// No specific source: query all RAG managers and combine results
	var allResults []QueryResult
	for name, mgr := range t.managers {
		results, err := mgr.Query(ctx, args.Query)
		if err != nil {
			slog.Error("RAG query failed", "rag", name, "error", err)
			continue
		}

		for _, result := range results {
			allResults = append(allResults, QueryResult{
				SourcePath: result.Document.SourcePath,
				Content:    result.Document.Content,
				Similarity: result.Similarity,
				ChunkIndex: result.Document.ChunkIndex,
			})
		}
	}

	// Sort by similarity (already sorted per manager, but we need to sort combined results)
	sortResults(allResults)

	// Limit to top results across all managers
	maxResults := 10
	if len(allResults) > maxResults {
		allResults = allResults[:maxResults]
	}

	resultJSON, err := json.Marshal(allResults)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
	}

	return &tools.ToolCallResult{
		Output: string(resultJSON),
	}, nil
}

func (t *RAGTool) Start(context.Context) error {
	return nil
}

func (t *RAGTool) Stop(context.Context) error {
	return nil
}

// sortResults sorts query results by similarity in descending order
func sortResults(results []QueryResult) {
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Similarity > results[i].Similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}
