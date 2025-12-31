package rerank

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/docker/cagent/pkg/model/provider"
	"github.com/docker/cagent/pkg/rag/database"
	"github.com/docker/cagent/pkg/rag/types"
)

// Reranker re-scores search results using a reranking model
type Reranker interface {
	// Rerank re-scores the given search results and returns them sorted by new scores
	Rerank(ctx context.Context, query string, results []database.SearchResult) ([]database.SearchResult, error)
}

// NOTE: The Reranker interface doesn't expose criteria directly - it's configured
// during construction via Config and passed through to the underlying provider.

// Config holds reranking configuration
type Config struct {
	Model     provider.Provider // The reranking model provider
	TopK      int               // Optional: only rerank top K results (0 = rerank all)
	Threshold float64           // Optional: minimum score threshold after reranking
	Criteria  string            // Optional: domain-specific relevance criteria to guide scoring
}

// LLMReranker implements reranking using any LLM provider that supports the RerankingProvider interface.
// This includes OpenAI, Anthropic, Gemini, and DMR providers.
type LLMReranker struct {
	config Config
}

// NewLLMReranker creates a new LLM-based reranker
func NewLLMReranker(config Config) (*LLMReranker, error) {
	if config.Model == nil {
		return nil, fmt.Errorf("reranking model is required")
	}

	slog.Debug("[Reranker] Creating LLM-based reranker",
		"model_id", config.Model.ID(),
		"top_k", config.TopK,
		"threshold", config.Threshold)

	return &LLMReranker{
		config: config,
	}, nil
}

// Rerank re-scores results using the reranking model
func (r *LLMReranker) Rerank(ctx context.Context, query string, results []database.SearchResult) ([]database.SearchResult, error) {
	startTime := time.Now()

	if len(results) == 0 {
		return results, nil
	}

	slog.Debug("[Reranker] Starting reranking",
		"model_id", r.config.Model.ID(),
		"query_length", len(query),
		"num_results", len(results),
		"top_k", r.config.TopK,
		"threshold", r.config.Threshold)

	// If TopK is set, only rerank the top K results
	numToRerank := len(results)
	if r.config.TopK > 0 && r.config.TopK < len(results) {
		numToRerank = r.config.TopK
		slog.Debug("[Reranker] TopK configured, limiting reranking",
			"original_count", len(results),
			"will_rerank", numToRerank)
	}

	// Get reranking provider that supports the reranking operation
	rerankProvider, ok := r.config.Model.(provider.RerankingProvider)
	if !ok {
		slog.Error("[Reranker] Model does not support reranking",
			"model_id", r.config.Model.ID(),
			"model_type", fmt.Sprintf("%T", r.config.Model))
		return nil, fmt.Errorf("model %s does not support reranking operation", r.config.Model.ID())
	}

	// Prepare documents for reranking with metadata
	documents := make([]types.Document, numToRerank)
	totalContentLength := 0
	for i := range numToRerank {
		doc := results[i].Document

		// Convert database.Document to types.Document
		// Include source path and any other useful metadata for context
		documents[i] = types.Document{
			Content:    doc.Content,
			SourcePath: doc.SourcePath,
			ChunkIndex: doc.ChunkIndex,
			// Note: database.Document doesn't have Metadata map yet,
			// but when it does, we should pass it through here
			Metadata: map[string]string{
				"created_at": doc.CreatedAt,
			},
		}
		totalContentLength += len(doc.Content)
	}

	slog.Debug("[Reranker] Prepared documents for reranking",
		"num_documents", len(documents),
		"total_content_length", totalContentLength,
		"avg_doc_length", totalContentLength/len(documents))

	// Call the reranking model with criteria
	scores, err := rerankProvider.Rerank(ctx, query, documents, r.config.Criteria)
	if err != nil {
		slog.Error("[Reranker] Reranking call failed",
			"model_id", r.config.Model.ID(),
			"num_documents", len(documents),
			"error", err)
		return nil, fmt.Errorf("reranking failed: %w", err)
	}

	if len(scores) != numToRerank {
		slog.Error("[Reranker] Score count mismatch",
			"expected", numToRerank,
			"received", len(scores))
		return nil, fmt.Errorf("reranking returned %d scores but expected %d", len(scores), numToRerank)
	}

	// Log score statistics
	minScore, maxScore, avgScore := calculateScoreStats(scores)
	slog.Debug("[Reranker] Received reranking scores",
		"num_scores", len(scores),
		"min_score", minScore,
		"max_score", maxScore,
		"avg_score", avgScore)

	// Update results with new scores
	rerankedResults := make([]database.SearchResult, 0, len(results))
	filteredCount := 0
	for i := range numToRerank {
		// Apply minimum score threshold if configured
		if r.config.Threshold > 0 && scores[i] < r.config.Threshold {
			filteredCount++
			slog.Debug("[Reranker] Filtering result below threshold",
				"index", i,
				"original_score", results[i].Similarity,
				"rerank_score", scores[i],
				"threshold", r.config.Threshold,
				"source_path", results[i].Document.SourcePath)
			continue
		}

		// Log score changes for top results
		if i < 5 {
			slog.Debug("[Reranker] Score update",
				"index", i,
				"original_score", results[i].Similarity,
				"rerank_score", scores[i],
				"score_change", scores[i]-results[i].Similarity,
				"source_path", results[i].Document.SourcePath)
		}

		// Create new result with updated score
		newResult := results[i]
		newResult.Similarity = scores[i]
		rerankedResults = append(rerankedResults, newResult)
	}

	if filteredCount > 0 {
		slog.Debug("[Reranker] Filtered results below threshold",
			"filtered_count", filteredCount,
			"threshold", r.config.Threshold)
	}

	// Add any remaining results that weren't reranked (if TopK was used)
	if numToRerank < len(results) {
		notRerankedCount := len(results) - numToRerank
		slog.Debug("[Reranker] Adding unranked results",
			"count", notRerankedCount)
		rerankedResults = append(rerankedResults, results[numToRerank:]...)
	}

	// Sort by new scores (descending)
	slices.SortFunc(rerankedResults, func(a, b database.SearchResult) int {
		return cmp.Compare(b.Similarity, a.Similarity)
	})

	// Log final score statistics
	totalDuration := time.Since(startTime)

	if len(rerankedResults) > 0 {
		finalMinScore, finalMaxScore, finalAvgScore := calculateScoreStats(extractScores(rerankedResults))
		slog.Debug("[Reranker] Reranking complete",
			"input_count", len(results),
			"reranked_count", numToRerank,
			"filtered_count", numToRerank-len(rerankedResults)+(len(results)-numToRerank),
			"output_count", len(rerankedResults),
			"final_min_score", finalMinScore,
			"final_max_score", finalMaxScore,
			"final_avg_score", finalAvgScore,
			"duration_ms", totalDuration.Milliseconds())
	} else {
		slog.Debug("[Reranker] Reranking complete",
			"input_count", len(results),
			"reranked_count", numToRerank,
			"output_count", len(rerankedResults),
			"duration_ms", totalDuration.Milliseconds())
	}

	return rerankedResults, nil
}

// calculateScoreStats computes min, max, and average scores
func calculateScoreStats(scores []float64) (minScore, maxScore, avgScore float64) {
	if len(scores) == 0 {
		return 0, 0, 0
	}

	minScore = scores[0]
	maxScore = scores[0]
	sum := 0.0

	for _, score := range scores {
		minScore = min(minScore, score)
		maxScore = max(maxScore, score)
		sum += score
	}

	avgScore = sum / float64(len(scores))
	return minScore, maxScore, avgScore
}

// extractScores extracts similarity scores from search results
func extractScores(results []database.SearchResult) []float64 {
	scores := make([]float64, len(results))
	for i, result := range results {
		scores[i] = result.Similarity
	}
	return scores
}
