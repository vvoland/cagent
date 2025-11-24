package fsx

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// VCSMatcher handles VCS ignore pattern matching for a git repository
type VCSMatcher struct {
	repoRoot string
	matcher  gitignore.Matcher
}

// NewVCSMatcher creates a new VCS matcher for the given path.
// It searches for a git repository and loads .gitignore patterns.
// Returns (nil, nil) if no git repository is found - this is not an error.
func NewVCSMatcher(basePath string) (*VCSMatcher, error) {
	// PlainOpen automatically searches up the directory tree for .git
	repo, err := git.PlainOpen(basePath)
	if err != nil {
		slog.Debug("No git repository found", "directory", basePath)
		return nil, nil
	}

	// Get the worktree
	worktree, err := repo.Worktree()
	if err != nil {
		slog.Warn("Failed to get worktree", "path", basePath, "error", err)
		return nil, err
	}

	// Get the repository root path
	repoRoot := worktree.Filesystem.Root()

	// Read gitignore patterns from the repository
	patterns, err := gitignore.ReadPatterns(worktree.Filesystem, nil)
	if err != nil {
		slog.Warn("Failed to read gitignore patterns", "path", repoRoot, "error", err)
		return nil, err
	}

	// Create matcher from patterns
	matcher := gitignore.NewMatcher(patterns)

	slog.Debug("Loaded gitignore patterns", "repository", repoRoot)

	return &VCSMatcher{
		repoRoot: repoRoot,
		matcher:  matcher,
	}, nil
}

// RepoRoot returns the repository root path for this matcher
func (m *VCSMatcher) RepoRoot() string {
	if m == nil {
		return ""
	}
	return m.repoRoot
}

// ShouldIgnore checks if a path should be ignored based on VCS rules.
// It checks both .git directories and .gitignore patterns.
func (m *VCSMatcher) ShouldIgnore(path string) bool {
	if m == nil {
		return false
	}

	// Always ignore .git directories and their contents
	// Check both the original path and normalized path
	base := filepath.Base(path)
	if base == ".git" {
		return true
	}
	normalizedPath := filepath.ToSlash(path)
	if strings.Contains(normalizedPath, "/.git/") || strings.HasSuffix(normalizedPath, "/.git") || strings.HasPrefix(normalizedPath, ".git/") {
		return true
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Check if path is within this repository
	if !strings.HasPrefix(absPath, m.repoRoot) {
		return false
	}

	// Create a relative path from the repository root for matching
	relPath, err := filepath.Rel(m.repoRoot, absPath)
	if err != nil {
		return false
	}

	// Check if the path is a directory
	info, err := os.Stat(path)
	isDir := err == nil && info.IsDir()

	normalizedRelPath := filepath.ToSlash(relPath)
	pathComponents := strings.Split(normalizedRelPath, "/")
	matched := m.matcher.Match(pathComponents, isDir)

	return matched
}
