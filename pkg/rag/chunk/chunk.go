package chunk

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Chunk represents a piece of text from a document
type Chunk struct {
	Index    int
	Content  string
	Metadata map[string]string
}

// Processor handles document processing
type Processor struct{}

// New creates a new document processor
func New() *Processor {
	return &Processor{}
}

// ProcessFile reads a file and splits it into chunks
func (p *Processor) ProcessFile(path string, chunkSize, overlap int, respectWordBoundaries bool) ([]Chunk, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.ChunkText(string(content), chunkSize, overlap, respectWordBoundaries), nil
}

// ChunkText splits text into overlapping chunks
func (p *Processor) ChunkText(text string, size, overlap int, respectWordBoundaries bool) []Chunk {
	if size <= 0 {
		size = 1000
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size / 2
	}

	var chunks []Chunk
	runes := []rune(text)
	totalLen := len(runes)

	if totalLen == 0 {
		return chunks
	}

	index := 0
	start := 0

	for start < totalLen {
		// Calculate end position (start + size, but not beyond document end)
		end := start + size
		if end > totalLen {
			end = totalLen
		}

		// If respecting word boundaries and we're NOT on the final chunk,
		// try to adjust the end so that we don't cut in the middle of a word.
		// For the final chunk (end == totalLen) we always take the remainder
		// of the document as-is to avoid generating progressively smaller
		// tail chunks.
		if respectWordBoundaries && end > start && end < totalLen {
			// Limit search to the current chunk window.
			target := end

			// Backtrack from target to find whitespace; if none is found
			// in a reasonable range, keep the original end so that we
			// still make progress even for very long "words".
			searchEnd := p.findNearestWhitespace(runes[start:target+1], target-start) + start
			if searchEnd > start && searchEnd < end {
				end = searchEnd
			}
		}

		// Create chunk
		chunk := string(runes[start:end])
		chunks = append(chunks, Chunk{
			Index:   index,
			Content: strings.TrimSpace(chunk),
		})

		index++

		// If we've reached the end of the document, we're done
		if end >= totalLen {
			break
		}

		// Next chunk starts at the end of the previous chunk minus overlap
		nextStart := end - overlap

		// CRITICAL: Ensure we always make forward progress
		// If nextStart would move us backward or keep us in place, advance by at least 1
		if nextStart <= start {
			nextStart = start + 1
		}

		// When respecting word boundaries, make sure the next chunk
		// does not start in the middle of a word. Move the start
		// forward to the next whitespace, then to the next non-whitespace.
		if respectWordBoundaries {
			// Move forward until we hit whitespace or end-of-text
			for nextStart < totalLen && !p.isWhitespace(runes[nextStart]) {
				nextStart++
			}
			// Skip the whitespace itself so we start at the first character
			// of the next word (if any).
			for nextStart < totalLen && p.isWhitespace(runes[nextStart]) {
				nextStart++
			}
		}

		start = nextStart
	}

	return chunks
}

// findNearestWhitespace finds the nearest whitespace boundary to the target position
// It searches backward first (within a reasonable distance), then forward if needed
func (p *Processor) findNearestWhitespace(runes []rune, target int) int {
	// Don't search beyond 20% of the total length in either direction
	maxSearchDistance := len(runes) / 5
	if maxSearchDistance < 50 {
		maxSearchDistance = 50
	}
	if maxSearchDistance > 500 {
		maxSearchDistance = 500
	}

	// Search backward first (prefer to keep chunks slightly smaller)
	for i := 0; i < maxSearchDistance && target-i > 0; i++ {
		pos := target - i
		if p.isWhitespace(runes[pos]) {
			// Skip consecutive whitespace
			for pos > 0 && p.isWhitespace(runes[pos-1]) {
				pos--
			}
			return pos
		}
	}

	// Search forward if no whitespace found backward
	for i := 1; i < maxSearchDistance && target+i < len(runes); i++ {
		pos := target + i
		if p.isWhitespace(runes[pos]) {
			return pos
		}
	}

	// If no whitespace found in search range, return original target
	return target
}

// isWhitespace checks if a rune is whitespace
func (p *Processor) isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// FileHash calculates SHA256 hash of a file
func (p *Processor) FileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// CollectFiles recursively collects all files from given paths
// Skips paths that don't exist instead of returning an error
func (p *Processor) CollectFiles(paths []string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	for _, pattern := range paths {
		expanded, err := p.expandPattern(pattern)
		if err != nil {
			return nil, err
		}
		if len(expanded) == 0 {
			expanded = []string{pattern}
		}

		for _, entry := range expanded {
			normalized := normalizePath(entry)

			info, err := os.Stat(normalized)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, fmt.Errorf("failed to stat %s: %w", entry, err)
			}

			if info.IsDir() {
				err := filepath.Walk(normalized, func(p string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if info.IsDir() {
						return nil
					}
					filePath := normalizePath(p)
					if !seen[filePath] {
						files = append(files, filePath)
						seen[filePath] = true
					}
					return nil
				})
				if err != nil {
					return nil, fmt.Errorf("failed to walk directory %s: %w", normalized, err)
				}
				continue
			}

			if !seen[normalized] {
				files = append(files, normalized)
				seen[normalized] = true
			}
		}
	}

	return files, nil
}

// Matches reports whether the given path matches any configured document path or glob pattern.
// To be used in file watchers to determine if a new/changed file matches the glob patterns or not.
func (p *Processor) Matches(path string, patterns []string) (bool, error) {
	if len(patterns) == 0 {
		return false, nil
	}

	cleanPath := normalizePath(path)

	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}

		normalizedPattern := normalizePath(pattern)

		if hasGlob(pattern) {
			match, err := doublestar.PathMatch(normalizedPattern, cleanPath)
			if err != nil {
				return false, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
			}
			if match {
				return true, nil
			}
			continue
		}

		info, err := os.Stat(normalizedPattern)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return false, fmt.Errorf("failed to stat %s: %w", normalizedPattern, err)
		}

		if info.IsDir() {
			if cleanPath == normalizedPattern || strings.HasPrefix(cleanPath, normalizedPattern+string(os.PathSeparator)) {
				return true, nil
			}
			continue
		}

		if cleanPath == normalizedPattern {
			return true, nil
		}
	}

	return false, nil
}

func (p *Processor) expandPattern(pattern string) ([]string, error) {
	if !hasGlob(pattern) {
		return []string{normalizePath(pattern)}, nil
	}

	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}

	results := make([]string, 0, len(matches))
	for _, match := range matches {
		results = append(results, normalizePath(match))
	}

	return results, nil
}

func hasGlob(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

func normalizePath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(p)
}
