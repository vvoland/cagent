package strategy

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/fsx"
	"github.com/docker/cagent/pkg/rag/chunk"
	"github.com/docker/cagent/pkg/rag/database"
	"github.com/docker/cagent/pkg/rag/treesitter"
	"github.com/docker/cagent/pkg/rag/types"
)

// NewBM25FromConfig creates a BM25 strategy from configuration
func NewBM25FromConfig(_ context.Context, cfg latest.RAGStrategyConfig, buildCtx BuildContext, events chan<- types.Event) (*Config, error) {
	// Get optional parameters with defaults
	k1 := GetParam(cfg.Params, "k1", 1.5)
	bParam := GetParam(cfg.Params, "b", 0.75)
	threshold := GetParam(cfg.Params, "threshold", 0.0)

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
		fmt.Sprintf("rag_%s_bm25.db", buildCtx.RAGName))
	if err != nil {
		return nil, fmt.Errorf("invalid database config: %w", err)
	}

	// Create BM25-specific database (no vectors needed).
	// Pass strategy type as table prefix so multiple strategies can share the same DB file.
	db, err := newBM25DB(dbPath, cfg.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Set default limit if not provided
	limit := cmp.Or(cfg.Limit, 5)

	// Parse chunking configuration
	chunkingCfg := ParseChunkingConfig(cfg)

	// Create strategy
	strategy := newBM25Strategy(
		"bm25",
		db,
		events,
		k1,
		bParam,
		chunkingCfg,
		BuildShouldIgnore(buildCtx, cfg.Params),
	)

	return &Config{
		Name:      "bm25",
		Strategy:  strategy,
		Docs:      docs,
		Limit:     limit,
		Threshold: thresholdVal,
		Chunking:  chunkingCfg,
	}, nil
}

// BM25Strategy implements BM25 keyword-based retrieval
// BM25 is a ranking function that uses term frequency and inverse document frequency
type BM25Strategy struct {
	name         string
	db           *bm25DB
	docProcessor chunk.DocumentProcessor
	fileHashes   map[string]string
	watcher      *fsnotify.Watcher
	watcherMu    sync.Mutex
	events       chan<- types.Event
	shouldIgnore func(path string) bool // Optional filter for gitignore support

	// BM25 parameters
	k1           float64 // term frequency saturation parameter (typically 1.2 to 2.0)
	b            float64 // length normalization parameter (typically 0.75)
	avgDocLength float64 // average document length
	docCount     int     // total number of documents
}

// newBM25Strategy creates a new BM25-based retrieval strategy
func newBM25Strategy(name string, db *bm25DB, events chan<- types.Event, k1, b float64, chunking ChunkingConfig, shouldIgnore func(string) bool) *BM25Strategy {
	// Create the appropriate document processor based on config
	var dp chunk.DocumentProcessor
	if chunking.CodeAware {
		dp = treesitter.NewDocumentProcessor(chunking.Size, chunking.Overlap, chunking.RespectWordBoundaries)
	} else {
		dp = chunk.NewTextDocumentProcessor(chunking.Size, chunking.Overlap, chunking.RespectWordBoundaries)
	}

	return &BM25Strategy{
		name:         name,
		db:           db,
		docProcessor: dp,
		fileHashes:   make(map[string]string),
		events:       events,
		shouldIgnore: shouldIgnore,
		k1:           k1,
		b:            b,
	}
}

// Initialize indexes all documents for BM25 retrieval
func (s *BM25Strategy) Initialize(ctx context.Context, docPaths []string, chunking ChunkingConfig) error {
	slog.Info("Starting BM25 strategy initialization",
		"name", s.name,
		"doc_paths", docPaths,
		"chunk_size", chunking.Size,
		"chunk_overlap", chunking.Overlap,
		"respect_word_boundaries", chunking.RespectWordBoundaries)

	// Load existing file hashes
	slog.Debug("Loading existing file hashes", "strategy", s.name)
	if err := s.loadExistingHashes(ctx); err != nil {
		slog.Warn("Failed to load existing file hashes", "strategy", s.name, "error", err)
	}

	// Collect all files
	slog.Debug("Collecting files", "strategy", s.name, "paths", docPaths)
	files, err := fsx.CollectFiles(docPaths, s.shouldIgnore)
	if err != nil {
		s.emitEvent(types.Event{Type: types.EventTypeError, Error: err})
		return fmt.Errorf("failed to collect files: %w", err)
	}

	seenFilesForCleanup := make(map[string]bool)
	for _, f := range files {
		seenFilesForCleanup[f] = true
	}

	if err := s.cleanupOrphanedDocuments(ctx, seenFilesForCleanup); err != nil {
		slog.Error("Failed to cleanup orphaned documents", "error", err)
	}

	if len(files) == 0 {
		slog.Warn("No files found for BM25 strategy", "name", s.name)
		return nil
	}

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
			slog.Error("Failed to check if file needs indexing", "path", filePath, "error", err)
			fileStatuses = append(fileStatuses, fileStatus{path: filePath, needsIndexing: false})
			continue
		}

		fileStatuses = append(fileStatuses, fileStatus{path: filePath, needsIndexing: needsIndexing})
		if needsIndexing {
			filesToIndex++
		}
	}

	if filesToIndex == 0 {
		slog.Info("All files up to date, no indexing needed", "name", s.name)
		return nil
	}

	s.emitEvent(types.Event{Type: types.EventTypeIndexingStarted})

	indexed := 0
	for _, status := range fileStatuses {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !status.needsIndexing {
			continue
		}

		indexed++

		s.emitEvent(types.Event{
			Type: types.EventTypeIndexingProgress,
			Progress: &types.Progress{
				Current: indexed,
				Total:   filesToIndex,
			},
		})

		if err := s.indexFile(ctx, status.path); err != nil {
			slog.Error("Failed to index file", "path", status.path, "error", err)
			continue
		}
	}

	if err := s.cleanupOrphanedDocuments(ctx, seenFiles); err != nil {
		slog.Error("Failed to cleanup orphaned documents", "error", err)
	}

	// Calculate average document length for BM25
	if err := s.calculateAvgDocLength(ctx); err != nil {
		slog.Error("Failed to calculate average document length", "error", err)
	}

	s.emitEvent(types.Event{Type: types.EventTypeIndexingComplete})

	slog.Info("BM25 strategy initialization completed",
		"name", s.name,
		"indexed", indexed)

	return nil
}

// Query searches for relevant documents using BM25 scoring
func (s *BM25Strategy) Query(ctx context.Context, query string, numResults int, threshold float64) ([]database.SearchResult, error) {
	// Tokenize query
	queryTerms := s.tokenize(query)
	if len(queryTerms) == 0 {
		return nil, fmt.Errorf("query contains no valid terms")
	}

	// For BM25, we need to retrieve all documents and score them
	// In a production system, you'd use an inverted index for efficiency
	// For now, this is a simplified implementation

	// Get all documents (in production, use inverted index to get only relevant docs)
	allDocs, err := s.getAllDocuments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}

	if len(allDocs) == 0 {
		return []database.SearchResult{}, nil
	}

	// Score each document using BM25
	scores := make([]database.SearchResult, 0, len(allDocs))
	for _, doc := range allDocs {
		score := s.calculateBM25Score(queryTerms, doc, allDocs)
		if score >= threshold {
			scores = append(scores, database.SearchResult{
				Document:   doc,
				Similarity: score,
			})
		}
	}

	// Sort by score descending
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].Similarity > scores[i].Similarity {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Return top N results
	if len(scores) > numResults {
		scores = scores[:numResults]
	}

	return scores, nil
}

// CheckAndReindexChangedFiles checks for file changes and re-indexes if needed
func (s *BM25Strategy) CheckAndReindexChangedFiles(ctx context.Context, docPaths []string, chunking ChunkingConfig) error {
	files, err := fsx.CollectFiles(docPaths, s.shouldIgnore)
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
			if err := s.indexFile(ctx, filePath); err != nil {
				slog.Error("Failed to re-index file", "path", filePath, "error", err)
			}
		}
	}

	if err := s.cleanupOrphanedDocuments(ctx, seenFiles); err != nil {
		slog.Error("Failed to cleanup orphaned documents", "error", err)
	}

	// Recalculate average document length
	if err := s.calculateAvgDocLength(ctx); err != nil {
		slog.Error("Failed to recalculate average document length", "error", err)
	}

	return nil
}

// StartFileWatcher starts monitoring files for changes
func (s *BM25Strategy) StartFileWatcher(ctx context.Context, docPaths []string, chunking ChunkingConfig) error {
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
	}

	go s.watchLoop(ctx, docPaths)

	slog.Info("File watcher started", "strategy", s.name)
	return nil
}

// Close releases resources
func (s *BM25Strategy) Close() error {
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

func (s *BM25Strategy) tokenize(text string) []string {
	// Simple tokenization: lowercase and split on whitespace/punctuation
	text = strings.ToLower(text)
	// Replace common punctuation with spaces
	replacer := strings.NewReplacer(
		".", " ", ",", " ", "!", " ", "?", " ",
		";", " ", ":", " ", "(", " ", ")", " ",
		"[", " ", "]", " ", "{", " ", "}", " ",
		"\"", " ", "'", " ", "\n", " ", "\t", " ",
	)
	text = replacer.Replace(text)

	tokens := strings.Fields(text)

	// Remove stopwords (simplified list)
	stopwords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "as": true, "by": true, "is": true,
		"was": true, "are": true, "were": true, "be": true, "been": true,
	}

	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if len(token) > 2 && !stopwords[token] {
			filtered = append(filtered, token)
		}
	}

	return filtered
}

func (s *BM25Strategy) calculateBM25Score(queryTerms []string, doc database.Document, allDocs []database.Document) float64 {
	docLength := float64(len(s.tokenize(doc.Content)))
	score := 0.0

	docTerms := s.tokenize(doc.Content)
	docTermFreq := make(map[string]int)
	for _, term := range docTerms {
		docTermFreq[term]++
	}

	for _, queryTerm := range queryTerms {
		// Term frequency in document
		tf := float64(docTermFreq[queryTerm])
		if tf == 0 {
			continue
		}

		// Document frequency (number of documents containing the term)
		df := 0
		for _, d := range allDocs {
			if strings.Contains(strings.ToLower(d.Content), queryTerm) {
				df++
			}
		}

		if df == 0 {
			continue
		}

		// IDF calculation
		idf := math.Log((float64(s.docCount)-float64(df)+0.5)/(float64(df)+0.5) + 1.0)

		// BM25 formula
		numerator := tf * (s.k1 + 1.0)
		denominator := tf + s.k1*(1.0-s.b+s.b*(docLength/s.avgDocLength))
		score += idf * (numerator / denominator)
	}

	// Normalize score to 0-1 range for consistency with vector similarity
	// This is a simple normalization; in production, you might use a different approach
	return math.Min(score/float64(len(queryTerms)), 1.0)
}

func (s *BM25Strategy) getAllDocuments(ctx context.Context) ([]database.Document, error) {
	docs, err := s.db.GetAllDocuments(ctx)
	if err != nil {
		return nil, err
	}
	s.docCount = len(docs)
	return docs, nil
}

func (s *BM25Strategy) calculateAvgDocLength(ctx context.Context) error {
	docs, err := s.getAllDocuments(ctx)
	if err != nil {
		return err
	}

	if len(docs) == 0 {
		s.avgDocLength = 0
		return nil
	}

	totalLength := 0
	for _, doc := range docs {
		totalLength += len(s.tokenize(doc.Content))
	}

	s.avgDocLength = float64(totalLength) / float64(len(docs))
	slog.Debug("Calculated average document length",
		"strategy", s.name,
		"avgLength", s.avgDocLength,
		"docCount", len(docs))

	return nil
}

func (s *BM25Strategy) loadExistingHashes(ctx context.Context) error {
	metadata, err := s.db.GetAllFileMetadata(ctx)
	if err != nil {
		return fmt.Errorf("failed to get file metadata: %w", err)
	}

	for _, meta := range metadata {
		s.fileHashes[meta.SourcePath] = meta.FileHash
	}

	return nil
}

func (s *BM25Strategy) needsIndexing(_ context.Context, filePath string) (bool, error) {
	currentHash, err := chunk.FileHash(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to hash file: %w", err)
	}

	storedHash, exists := s.fileHashes[filePath]
	if !exists {
		return true, nil
	}

	return storedHash != currentHash, nil
}

func (s *BM25Strategy) indexFile(ctx context.Context, filePath string) error {
	fileHash, err := chunk.FileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to hash file: %w", err)
	}

	if err := s.db.DeleteDocumentsByPath(ctx, filePath); err != nil {
		return fmt.Errorf("failed to delete old documents: %w", err)
	}

	chunks, err := chunk.ProcessFile(s.docProcessor, filePath)
	if err != nil {
		return fmt.Errorf("failed to process file: %w", err)
	}

	storedChunks := 0
	for _, chunk := range chunks {
		if chunk.Content == "" {
			continue
		}

		doc := database.Document{
			ID:         fmt.Sprintf("%s_%d_%d", filePath, chunk.Index, time.Now().UnixNano()),
			SourcePath: filePath,
			ChunkIndex: chunk.Index,
			Content:    chunk.Content,
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
	slog.Debug("Indexed file with BM25", "path", filePath, "chunks", storedChunks)
	return nil
}

func (s *BM25Strategy) cleanupOrphanedDocuments(ctx context.Context, seenFiles map[string]bool) error {
	metadata, err := s.db.GetAllFileMetadata(ctx)
	if err != nil {
		return fmt.Errorf("failed to get file metadata: %w", err)
	}

	for _, meta := range metadata {
		if !seenFiles[meta.SourcePath] {
			if err := s.db.DeleteDocumentsByPath(ctx, meta.SourcePath); err != nil {
				slog.Error("Failed to delete orphaned documents", "path", meta.SourcePath, "error", err)
				continue
			}

			if err := s.db.DeleteFileMetadata(ctx, meta.SourcePath); err != nil {
				slog.Error("Failed to delete orphaned metadata", "path", meta.SourcePath, "error", err)
				continue
			}

			delete(s.fileHashes, meta.SourcePath)
		}
	}

	return nil
}

func (s *BM25Strategy) addPathToWatcher(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	stat, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if err := s.watcher.Add(absPath); err != nil {
		return fmt.Errorf("failed to add path to watcher: %w", err)
	}

	if stat.IsDir() {
		files, err := fsx.CollectFiles([]string{absPath}, s.shouldIgnore)
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

func (s *BM25Strategy) watchLoop(ctx context.Context, docPaths []string) {
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

		for _, file := range changedFiles {
			// Check if the file matches any of the configured document paths/patterns
			matches, matchErr := fsx.Matches(file, docPaths)
			if matchErr != nil {
				slog.Error("Failed to match path", "file", file, "error", matchErr)
				continue
			}
			if !matches {
				continue
			}
			// Check if the file should be ignored (e.g., gitignore)
			if s.shouldIgnore != nil && s.shouldIgnore(file) {
				slog.Debug("File changed but is ignored by filter, skipping", "path", file)
				continue
			}

			needsIndexing, err := s.needsIndexing(ctx, file)
			if err != nil || !needsIndexing {
				continue
			}

			slog.Debug("Indexing file", "path", file, "strategy", s.name)
			if err := s.indexFile(ctx, file); err != nil {
				slog.Error("Failed to re-index file", "path", file, "error", err)
			}
		}

		_ = s.calculateAvgDocLength(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
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
				_ = s.addPathToWatcher(event.Name)
				s.watcherMu.Unlock()
			}

			// Early filter: only track changes for files that match configured doc patterns
			matches, err := fsx.Matches(event.Name, docPaths)
			if err != nil || !matches {
				continue
			}
			// Skip files that should be ignored (e.g., gitignore)
			if s.shouldIgnore != nil && s.shouldIgnore(event.Name) {
				continue
			}

			pendingMu.Lock()
			pendingChanges[event.Name] = true
			pendingMu.Unlock()

			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDuration, processChanges)

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("File watcher error", "strategy", s.name, "error", err)
		}
	}
}

func (s *BM25Strategy) emitEvent(event types.Event) {
	EmitEvent(s.events, event, s.name)
}
