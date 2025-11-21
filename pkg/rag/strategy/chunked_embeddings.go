package strategy

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/docker/cagent/pkg/config"
	latest "github.com/docker/cagent/pkg/config/v3"
	"github.com/docker/cagent/pkg/model/provider"
	"github.com/docker/cagent/pkg/model/provider/options"
	"github.com/docker/cagent/pkg/modelsdev"
	"github.com/docker/cagent/pkg/rag/chunk"
	"github.com/docker/cagent/pkg/rag/database"
	"github.com/docker/cagent/pkg/rag/embed"
)

// NewChunkedEmbeddingsFromConfig creates a chunked-embeddings strategy from configuration
func NewChunkedEmbeddingsFromConfig(ctx context.Context, cfg latest.RAGStrategyConfig, buildCtx BuildContext, events chan<- Event) (*Config, error) {
	// Extract required parameters
	modelName := GetParam(cfg.Params, "model", "")
	if modelName == "" {
		return nil, fmt.Errorf("'model' parameter required for chunked-embeddings strategy")
	}

	// Get or create embedding model
	var embedModel provider.Provider
	var modelCfg latest.ModelConfig
	var err error

	if modelName == "auto" {
		// Auto-detect embedding model (try DMR first, fall back to OpenAI)
		embedModel, err = createAutoEmbeddingModel(ctx, buildCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to create auto embedding model: %w", err)
		}
		// For auto, model ID will be determined from embedModel.ID()
	} else {
		// Look up model in config
		modelCfg, exists := buildCtx.Models[modelName]
		if !exists {
			return nil, fmt.Errorf("model '%s' not found for chunked-embeddings strategy", modelName)
		}

		embedModel, err = provider.New(ctx, &modelCfg, buildCtx.Env,
			options.WithGateway(buildCtx.ModelsGateway))
		if err != nil {
			return nil, fmt.Errorf("failed to create embedding model: %w", err)
		}
	}

	// Get parameters (some optional, some required) with defaults where safe.
	similarityMetric := GetParam(cfg.Params, "similarity_metric", "cosine_similarity")
	threshold := GetParam(cfg.Params, "threshold", 0.5)

	// vector_dimensions is required because embedding dimensionality depends on
	// the chosen model, and using an incorrect default could corrupt or
	// invalidate the vector database.
	vectorDimensionsPtr := GetParamPtr[int](cfg.Params, "vector_dimensions")
	if vectorDimensionsPtr == nil {
		return nil, fmt.Errorf("vector_dimensions parameter is required for chunked-embeddings strategy")
	}
	vectorDimensions := *vectorDimensionsPtr

	// Handle threshold as pointer (might be float or *float)
	var thresholdVal float64
	if thresholdPtr := GetParamPtr[float64](cfg.Params, "threshold"); thresholdPtr != nil {
		thresholdVal = *thresholdPtr
	} else {
		thresholdVal = threshold
	}

	// Merge document paths
	docs := MergeDocPaths(buildCtx.SharedDocs, cfg.Docs, buildCtx.ParentDir)

	// Resolve database path
	dbPath, err := ResolveDatabasePath(cfg.Database, buildCtx.ParentDir,
		fmt.Sprintf("rag_%s_chunked_embeddings.db", buildCtx.RAGName))
	if err != nil {
		return nil, fmt.Errorf("invalid database config: %w", err)
	}

	// Create vector database with specified dimensions
	db, err := NewVectorDatabase(dbPath, vectorDimensions)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Set default limit if not provided
	limit := cfg.Limit
	if limit == 0 {
		limit = 5
	}

	// Extract chunking configuration with defaults
	chunkSize := cfg.Chunking.Size
	if chunkSize == 0 {
		chunkSize = 1000
	}
	chunkOverlap := cfg.Chunking.Overlap
	if chunkOverlap == 0 {
		chunkOverlap = 75
	}
	respectWordBoundaries := cfg.Chunking.RespectWordBoundaries

	batchSize := GetParam(cfg.Params, "batch_size", 50)
	maxConcurrency := GetParam(cfg.Params, "max_embedding_concurrency", 3)

	// Get full model ID for pricing lookup
	modelID := modelCfg.Provider + "/" + modelCfg.Model
	if modelName == "auto" {
		// For auto models, determine the ID from what was created
		modelID = embedModel.ID()
	}

	// Create models.dev store for pricing (separate from runtime's store)
	modelsStore, err := modelsdev.NewStore()
	if err != nil {
		slog.Debug("Failed to create models.dev store for RAG pricing; cost tracking disabled",
			"rag", buildCtx.RAGName,
			"error", err)
	}

	// Create strategy with model store for pricing
	strategy := NewVectorStrategy(
		"chunked-embeddings",
		embedModel,
		db,
		events,
		similarityMetric,
		batchSize,
		maxConcurrency,
		modelID,
		modelsStore,
	)

	return &Config{
		Name:                  "chunked-embeddings",
		Strategy:              strategy,
		Docs:                  docs,
		Limit:                 limit,
		Threshold:             thresholdVal,
		ChunkSize:             chunkSize,
		ChunkOverlap:          chunkOverlap,
		RespectWordBoundaries: respectWordBoundaries,
	}, nil
}

// createAutoEmbeddingModel creates an auto-detected embedding model
func createAutoEmbeddingModel(ctx context.Context, buildCtx BuildContext) (provider.Provider, error) {
	// Try embedding-capable models in the shared auto-embedding priority order.
	// The order is defined in config.AutoEmbeddingModelConfigs and currently
	// prefers OpenAI first, then falls back to DMR.
	var lastErr error

	for _, autoModelCfg := range config.AutoEmbeddingModelConfigs() {
		// Convert to v2.ModelConfig so we use the same type everywhere in RAG.
		modelCfg := latest.ModelConfig{
			Provider: autoModelCfg.Provider,
			Model:    autoModelCfg.Model,
		}

		model, err := provider.New(ctx, &modelCfg, buildCtx.Env,
			options.WithGateway(buildCtx.ModelsGateway))
		if err != nil {
			lastErr = err
			continue
		}

		return model, nil
	}

	if lastErr == nil {
		return nil, fmt.Errorf("failed to create auto embedding model: no candidates configured")
	}

	return nil, fmt.Errorf("failed to create auto embedding model: %w", lastErr)
}

// VectorStrategy implements retrieval using vector embeddings and similarity search
type VectorStrategy struct {
	name             string
	db               database.Database
	embedder         *embed.Embedder
	processor        *chunk.Processor
	fileHashes       map[string]string
	watcher          *fsnotify.Watcher
	watcherMu        sync.Mutex
	events           chan<- Event
	similarityMetric string
	indexingTokens   int     // Track tokens used during indexing
	indexingCost     float64 // Track cost during indexing
	modelID          string  // Full model ID (e.g., "openai/text-embedding-3-small") for pricing lookup
	modelsStore      modelStore
}

type modelStore interface {
	GetModel(ctx context.Context, modelID string) (*modelsdev.Model, error)
}

// NewVectorStrategy creates a new vector-based retrieval strategy with models.dev pricing
func NewVectorStrategy(name string, embedModel provider.Provider, db database.Database, events chan<- Event, similarityMetric string, batchSize, maxConcurrency int, modelID string, modelsStore modelStore) *VectorStrategy {
	emb := embed.New(embedModel, embed.WithBatchSize(batchSize), embed.WithMaxConcurrency(maxConcurrency))

	s := &VectorStrategy{
		name:             name,
		db:               db,
		embedder:         emb,
		processor:        chunk.New(),
		fileHashes:       make(map[string]string),
		events:           events,
		similarityMetric: similarityMetric,
		modelID:          modelID,
		modelsStore:      modelsStore,
	}

	// Set usage handler to calculate cost from models.dev and emit events with CUMULATIVE totals
	// This matches how chat completions calculate cost in runtime.go:691-694
	emb.SetUsageHandler(func(tokens int, providerCost float64) {
		// Calculate cost using models.dev pricing (same as chat completions)
		cost := s.calculateCost(context.Background(), tokens)

		s.indexingTokens += tokens
		s.indexingCost += cost

		// Emit usage event with CUMULATIVE totals for TUI
		s.emitEvent(Event{
			Type:        "usage",
			TotalTokens: s.indexingTokens, // Cumulative tokens
			Cost:        s.indexingCost,   // Cumulative cost
		})
	})

	return s
}

// calculateCost calculates embedding cost using models.dev pricing (same pattern as runtime.go:691-694)
func (s *VectorStrategy) calculateCost(ctx context.Context, tokens int) float64 {
	if s.modelsStore == nil {
		return 0
	}

	model, err := s.modelsStore.GetModel(ctx, s.modelID)
	if err != nil {
		slog.Debug("Failed to get model pricing from models.dev, cost will be 0",
			"model_id", s.modelID,
			"error", err)
		return 0
	}

	if model.Cost == nil {
		return 0
	}

	// Embeddings only have input tokens, no output
	// Cost is per 1M tokens, so divide by 1e6
	return (float64(tokens) * model.Cost.Input) / 1e6
}

// Initialize indexes all documents
func (s *VectorStrategy) Initialize(ctx context.Context, docPaths []string, chunkSize, chunkOverlap int, respectWordBoundaries bool) error {
	slog.Info("Starting chunked-embeddings strategy initialization",
		"name", s.name,
		"doc_paths", docPaths,
		"chunk_size", chunkSize,
		"chunk_overlap", chunkOverlap,
		"respect_word_boundaries", respectWordBoundaries)

	// Load existing file hashes from metadata
	slog.Debug("Loading existing file hashes", "strategy", s.name)
	if err := s.loadExistingHashes(ctx); err != nil {
		slog.Warn("Failed to load existing file hashes", "strategy", s.name, "error", err)
	}

	// Collect all files
	slog.Debug("Collecting files", "strategy", s.name, "paths", docPaths)
	files, err := s.processor.CollectFiles(docPaths)
	if err != nil {
		s.emitEvent(Event{Type: "error", Error: err})
		return fmt.Errorf("failed to collect files: %w", err)
	}

	// Track seen files for cleanup
	seenFilesForCleanup := make(map[string]bool)
	for _, f := range files {
		seenFilesForCleanup[f] = true
	}

	// Clean up orphaned documents
	if err := s.cleanupOrphanedDocuments(ctx, seenFilesForCleanup); err != nil {
		slog.Error("Failed to cleanup orphaned documents during initialization", "error", err)
	}

	if len(files) == 0 {
		slog.Warn("No files found for chunked-embeddings strategy", "name", s.name, "paths", docPaths)
		return nil
	}

	slog.Debug("Collected files for indexing check",
		"strategy", s.name,
		"file_count", len(files))

	// Determine which files need indexing
	type fileStatus struct {
		path          string
		needsIndexing bool
	}

	var fileStatuses []fileStatus
	seenFiles := make(map[string]bool)
	filesToIndex := 0

	for _, filePath := range files {
		seenFiles[filePath] = true

		needsIndexing, err := s.needsIndexing(ctx, filePath)
		if err != nil {
			slog.Error("Failed to check if file needs indexing",
				"path", filePath, "error", err)
			fileStatuses = append(fileStatuses, fileStatus{path: filePath, needsIndexing: false})
			continue
		}

		fileStatuses = append(fileStatuses, fileStatus{path: filePath, needsIndexing: needsIndexing})
		if needsIndexing {
			filesToIndex++
		}
	}

	if filesToIndex == 0 {
		slog.Info("All fi  # Split on whitespace, not in the middle of wordses up to date, no indexing needed",
			"name", s.name,
			"total_files", len(files))
		return nil
	}

	s.emitEvent(Event{Type: "indexing_started"})

	// Index files that need it
	indexed := 0
	for _, status := range fileStatuses {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !status.needsIndexing {
			slog.Debug("File unchanged, skipping", "path", status.path)
			continue
		}

		indexed++

		s.emitEvent(Event{
			Type: "indexing_progress",
			Progress: &Progress{
				Current: indexed,
				Total:   filesToIndex,
			},
		})

		if err := s.indexFile(ctx, status.path, chunkSize, chunkOverlap, respectWordBoundaries); err != nil {
			slog.Error("Failed to index file", "path", status.path, "error", err)
			continue
		}
	}

	if err := s.cleanupOrphanedDocuments(ctx, seenFiles); err != nil {
		slog.Error("Failed to cleanup orphaned documents", "error", err)
	}

	s.emitEvent(Event{Type: "indexing_completed"})

	slog.Info("Chunked-embeddings strategy initialization completed",
		"name", s.name,
		"total_files", len(files),
		"indexed", indexed,
		"total_tokens", s.indexingTokens,
		"total_cost", s.indexingCost)

	return nil
}

// Query searches for relevant documents using vector similarity
func (s *VectorStrategy) Query(ctx context.Context, query string, numResults int, threshold float64) ([]database.SearchResult, error) {
	queryEmbedding, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	results, err := s.db.SearchSimilar(ctx, queryEmbedding, numResults)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	var filtered []database.SearchResult
	for _, result := range results {
		if result.Similarity >= threshold {
			filtered = append(filtered, result)
		}
	}

	return filtered, nil
}

// CheckAndReindexChangedFiles checks for file changes and re-indexes if needed
func (s *VectorStrategy) CheckAndReindexChangedFiles(ctx context.Context, docPaths []string, chunkSize, chunkOverlap int, respectWordBoundaries bool) error {
	files, err := s.processor.CollectFiles(docPaths)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}

	seenFiles := make(map[string]bool)

	for _, filePath := range files {
		seenFiles[filePath] = true

		needsIndexing, err := s.needsIndexing(ctx, filePath)
		if err != nil {
			slog.Error("Failed to check if file needs indexing", "path", filePath, "error", err)
			continue
		}

		if needsIndexing {
			slog.Info("File changed, re-indexing", "path", filePath)
			if err := s.indexFile(ctx, filePath, chunkSize, chunkOverlap, respectWordBoundaries); err != nil {
				slog.Error("Failed to re-index file", "path", filePath, "error", err)
			}
		}
	}

	if err := s.cleanupOrphanedDocuments(ctx, seenFiles); err != nil {
		slog.Error("Failed to cleanup orphaned documents during file watch", "error", err)
	}

	return nil
}

// StartFileWatcher starts monitoring files for changes
func (s *VectorStrategy) StartFileWatcher(ctx context.Context, docPaths []string, chunkSize, chunkOverlap int, respectWordBoundaries bool) error {
	s.watcherMu.Lock()
	defer s.watcherMu.Unlock()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	s.watcher = watcher

	for _, docPath := range docPaths {
		if err := s.addPathToWatcher(docPath); err != nil {
			slog.Warn("Failed to watch path", "strategy", s.name, "path", docPath, "error", err)
			continue
		}
		slog.Debug("Watching path for changes", "strategy", s.name, "path", docPath)
	}

	go s.watchLoop(ctx, docPaths, chunkSize, chunkOverlap, respectWordBoundaries)

	slog.Info("File watcher started", "strategy", s.name, "paths", docPaths)
	return nil
}

// Close releases resources
func (s *VectorStrategy) Close() error {
	s.watcherMu.Lock()
	defer s.watcherMu.Unlock()

	var firstErr error

	// Close file watcher
	if s.watcher != nil {
		if err := s.watcher.Close(); err != nil {
			slog.Warn("Failed to close file watcher", "strategy", s.name, "error", err)
			firstErr = err
		}
		s.watcher = nil
	}

	// Close database connection
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			slog.Error("Failed to close database", "strategy", s.name, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

// Helper methods

func (s *VectorStrategy) loadExistingHashes(ctx context.Context) error {
	metadata, err := s.db.GetAllFileMetadata(ctx)
	if err != nil {
		return fmt.Errorf("failed to get file metadata: %w", err)
	}

	for _, meta := range metadata {
		s.fileHashes[meta.SourcePath] = meta.FileHash
		slog.Debug("Loaded file hash from metadata",
			"path", meta.SourcePath,
			"hash", meta.FileHash)
	}

	slog.Debug("Loaded existing file hashes from metadata",
		"strategy", s.name,
		"count", len(s.fileHashes))

	return nil
}

func (s *VectorStrategy) needsIndexing(_ context.Context, filePath string) (bool, error) {
	currentHash, err := s.processor.FileHash(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to hash file: %w", err)
	}

	storedHash, exists := s.fileHashes[filePath]
	if !exists {
		slog.Debug("File not in metadata, needs indexing", "path", filePath)
		return true, nil
	}

	needsIndexing := storedHash != currentHash
	if needsIndexing {
		slog.Debug("File hash changed, needs re-indexing", "path", filePath)
	}
	return needsIndexing, nil
}

func (s *VectorStrategy) indexFile(ctx context.Context, filePath string, chunkSize, chunkOverlap int, respectWordBoundaries bool) error {
	fileHash, err := s.processor.FileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to hash file: %w", err)
	}

	if err := s.db.DeleteDocumentsByPath(ctx, filePath); err != nil {
		return fmt.Errorf("failed to delete old documents: %w", err)
	}

	chunks, err := s.processor.ProcessFile(filePath, chunkSize, chunkOverlap, respectWordBoundaries)
	if err != nil {
		return fmt.Errorf("failed to process file: %w", err)
	}

	// Filter out empty chunks and collect chunk contents
	var validChunks []chunk.Chunk
	var chunkContents []string
	for _, chunk := range chunks {
		if chunk.Content != "" {
			validChunks = append(validChunks, chunk)
			chunkContents = append(chunkContents, chunk.Content)
		}
	}

	if len(validChunks) == 0 {
		slog.Debug("No valid chunks in file", "path", filePath)
		return nil
	}

	// Generate embeddings for all chunks in batch
	slog.Debug("Generating embeddings for file",
		"path", filePath,
		"chunk_count", len(validChunks))

	embeddings, err := s.embedder.EmbedBatch(ctx, chunkContents)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	if len(embeddings) != len(validChunks) {
		return fmt.Errorf("embedding count mismatch: got %d embeddings for %d chunks", len(embeddings), len(validChunks))
	}

	// Store all documents
	storedChunks := 0
	for i, chunk := range validChunks {
		doc := database.Document{
			ID:         fmt.Sprintf("%s_%d_%d", filePath, chunk.Index, time.Now().UnixNano()),
			SourcePath: filePath,
			ChunkIndex: chunk.Index,
			Content:    chunk.Content,
			Embedding:  embeddings[i],
			FileHash:   fileHash,
		}

		if err := s.db.AddDocument(ctx, doc); err != nil {
			return fmt.Errorf("failed to add document: %w", err)
		}

		storedChunks++
	}

	metadata := database.FileMetadata{
		SourcePath: filePath,
		FileHash:   fileHash,
		ChunkCount: storedChunks,
	}
	if err := s.db.SetFileMetadata(ctx, metadata); err != nil {
		return fmt.Errorf("failed to update file metadata: %w", err)
	}

	s.fileHashes[filePath] = fileHash
	slog.Debug("Indexed file", "path", filePath, "chunks", storedChunks)
	return nil
}

func (s *VectorStrategy) cleanupOrphanedDocuments(ctx context.Context, seenFiles map[string]bool) error {
	metadata, err := s.db.GetAllFileMetadata(ctx)
	if err != nil {
		return fmt.Errorf("failed to get file metadata: %w", err)
	}

	deletedCount := 0
	for _, meta := range metadata {
		if seenFiles[meta.SourcePath] {
			continue
		}

		slog.Info("Removing embeddings of orphaned documents", "path", meta.SourcePath)

		if err := s.db.DeleteDocumentsByPath(ctx, meta.SourcePath); err != nil {
			slog.Error("Failed to delete orphaned documents",
				"path", meta.SourcePath, "error", err)
			continue
		}

		if err := s.db.DeleteFileMetadata(ctx, meta.SourcePath); err != nil {
			slog.Error("Failed to delete orphaned metadata",
				"path", meta.SourcePath, "error", err)
			continue
		}

		delete(s.fileHashes, meta.SourcePath)
		deletedCount++
	}

	if deletedCount > 0 {
		slog.Info("Cleaned up orphaned documents", "strategy", s.name, "count", deletedCount)
	}

	return nil
}

func (s *VectorStrategy) addPathToWatcher(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	path = absPath

	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Debug("Path does not exist, skipping watch", "path", path)
			return nil
		}
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if err := s.watcher.Add(path); err != nil {
		return fmt.Errorf("failed to add path to watcher: %w", err)
	}

	if stat.IsDir() {
		files, err := s.processor.CollectFiles([]string{path})
		if err != nil {
			return fmt.Errorf("failed to collect files: %w", err)
		}

		visited := make(map[string]bool)
		for _, file := range files {
			dir := filepath.Dir(file)
			if !visited[dir] {
				visited[dir] = true
				_ = s.watcher.Add(dir)
			}
		}
	}

	return nil
}

func (s *VectorStrategy) watchLoop(ctx context.Context, docPaths []string, chunkSize, chunkOverlap int, respectWordBoundaries bool) {
	var debounceTimer *time.Timer
	debounceDuration := 2 * time.Second
	pendingChanges := make(map[string]bool)
	var pendingMu sync.Mutex

	processChanges := func() {
		pendingMu.Lock()
		changedFiles := make([]string, 0, len(pendingChanges))
		for file := range pendingChanges {
			changedFiles = append(changedFiles, file)
		}
		pendingChanges = make(map[string]bool)
		pendingMu.Unlock()

		if len(changedFiles) == 0 {
			return
		}

		slog.Info("Processing file changes", "strategy", s.name, "count", len(changedFiles))

		filesToReindex := make([]string, 0)
		for _, file := range changedFiles {
			needsIndexing, err := s.needsIndexing(ctx, file)
			if err != nil {
				slog.Debug("File no longer exists or inaccessible", "path", file, "error", err)
				continue
			}
			if needsIndexing {
				filesToReindex = append(filesToReindex, file)
			}
		}

		if len(filesToReindex) > 0 {
			s.emitEvent(Event{
				Type:    "indexing_started",
				Message: fmt.Sprintf("Re-indexing %d changed file(s)", len(filesToReindex)),
			})

			for i, file := range filesToReindex {
				s.emitEvent(Event{
					Type:    "indexing_progress",
					Message: fmt.Sprintf("Re-indexing: %s", filepath.Base(file)),
					Progress: &Progress{
						Current: i + 1,
						Total:   len(filesToReindex),
					},
				})

				if err := s.indexFile(ctx, file, chunkSize, chunkOverlap, respectWordBoundaries); err != nil {
					slog.Error("Failed to re-index file", "path", file, "error", err)
					s.emitEvent(Event{
						Type:    "error",
						Message: fmt.Sprintf("Failed to re-index: %s", filepath.Base(file)),
						Error:   err,
					})
				}
			}

			if err := s.cleanupOrphanedDocumentsFromDisk(ctx, docPaths); err != nil {
				slog.Error("Failed to cleanup orphaned documents", "error", err)
			}

			s.emitEvent(Event{
				Type:    "indexing_completed",
				Message: fmt.Sprintf("Re-indexed %d file(s)", len(filesToReindex)),
			})
		}
	}

	for {
		select {
		case <-ctx.Done():
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			slog.Info("File watcher stopped", "strategy", s.name)
			return

		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}

			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}

			if event.Op&fsnotify.Create != 0 {
				s.watcherMu.Lock()
				if err := s.addPathToWatcher(event.Name); err != nil {
					slog.Debug("Could not watch new path", "path", event.Name, "error", err)
				}
				s.watcherMu.Unlock()
			}

			pendingMu.Lock()
			pendingChanges[event.Name] = true
			pendingMu.Unlock()

			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDuration, processChanges)

			slog.Debug("File system event detected",
				"strategy", s.name,
				"event", event.Op.String(),
				"path", event.Name)

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("File watcher error", "strategy", s.name, "error", err)
		}
	}
}

func (s *VectorStrategy) cleanupOrphanedDocumentsFromDisk(ctx context.Context, docPaths []string) error {
	files, err := s.processor.CollectFiles(docPaths)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}

	seenFiles := make(map[string]bool)
	for _, file := range files {
		seenFiles[file] = true
	}

	return s.cleanupOrphanedDocuments(ctx, seenFiles)
}

func (s *VectorStrategy) emitEvent(event Event) {
	EmitEvent(s.events, event, s.name)
}

// GetIndexingUsage returns usage statistics from indexing
func (s *VectorStrategy) GetIndexingUsage() (tokens int, cost float64) {
	return s.indexingTokens, s.indexingCost
}
