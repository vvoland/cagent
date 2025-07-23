package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/docker/cagent/pkg/tools"
)

type FilesystemTool struct {
	allowedDirectories []string
}

func NewFilesystemTool(allowedDirectories []string) *FilesystemTool {
	return &FilesystemTool{
		allowedDirectories: allowedDirectories,
	}
}

func (t *FilesystemTool) Instructions() string {
	return `## Filesystem Tool Instructions

This toolset provides comprehensive filesystem operations with built-in security restrictions.

### Security Model
- All operations are restricted to allowed directories only
- Use list_allowed_directories to see available paths
- Subdirectories within allowed directories are accessible

### Common Patterns
- Always check if directories exist before creating files
- Use directory_tree for exploring unfamiliar directory structures
- Prefer read_multiple_files for batch operations
- Use search_files_content for finding specific code or text

### Performance Tips
- Use read_multiple_files instead of multiple read_file calls
- Use directory_tree with max_depth to limit large traversals
- Use appropriate exclude patterns in search operations`
}

func (t *FilesystemTool) Tools(ctx context.Context) ([]tools.Tool, error) {
	return []tools.Tool{
		{
			Function: &tools.FunctionDefinition{
				Name:        "create_directory",
				Description: "Create a new directory or ensure a directory exists. Can create multiple nested directories in one operation.",
				Parameters: tools.FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "The directory path to create",
						},
					},
					Required: []string{"path"},
				},
			},
			Handler: t.handleCreateDirectory,
		},
		{
			Function: &tools.FunctionDefinition{
				Name:        "directory_tree",
				Description: "Get a recursive tree view of files and directories as a JSON structure.",
				Parameters: tools.FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "The directory path to traverse",
						},
						"max_depth": map[string]any{
							"type":        "number",
							"description": "Maximum depth to traverse (optional)",
						},
					},
					Required: []string{"path"},
				},
			},
			Handler: t.handleDirectoryTree,
		},
		{
			Function: &tools.FunctionDefinition{
				Name:        "edit_file",
				Description: "Make line-based edits to a text file. Each edit replaces exact line sequences with new content.",
				Parameters: tools.FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "The file path to edit",
						},
						"edits": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"oldText": map[string]any{
										"type":        "string",
										"description": "The exact text to replace",
									},
									"newText": map[string]any{
										"type":        "string",
										"description": "The replacement text",
									},
								},
								"required": []string{"oldText", "newText"},
							},
							"description": "Array of edit operations",
						},
						"dryRun": map[string]any{
							"type":        "boolean",
							"description": "If true, preview changes without applying them",
						},
					},
					Required: []string{"path", "edits"},
				},
			},
			Handler: t.handleEditFile,
		},
		{
			Function: &tools.FunctionDefinition{
				Name:        "get_file_info",
				Description: "Retrieve detailed metadata about a file or directory.",
				Parameters: tools.FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "The file or directory path to inspect",
						},
					},
					Required: []string{"path"},
				},
			},
			Handler: t.handleGetFileInfo,
		},
		{
			Function: &tools.FunctionDefinition{
				Name:        "list_allowed_directories",
				Description: "Returns a list of directories that the server has permission to access.",
				Parameters: tools.FunctionParamaters{
					Type:       "object",
					Properties: map[string]any{},
				},
			},
			Handler: t.handleListAllowedDirectories,
		},
		{
			Function: &tools.FunctionDefinition{
				Name:        "list_directory",
				Description: "Get a detailed listing of all files and directories in a specified path.",
				Parameters: tools.FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "The directory path to list",
						},
					},
					Required: []string{"path"},
				},
			},
			Handler: t.handleListDirectory,
		},
		{
			Function: &tools.FunctionDefinition{
				Name:        "list_directory_with_sizes",
				Description: "Get a detailed listing of all files and directories in a specified path, including sizes.",
				Parameters: tools.FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "The directory path to list",
						},
					},
					Required: []string{"path"},
				},
			},
			Handler: t.handleListDirectoryWithSizes,
		},
		{
			Function: &tools.FunctionDefinition{
				Name:        "move_file",
				Description: "Move or rename files and directories.",
				Parameters: tools.FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"source": map[string]any{
							"type":        "string",
							"description": "The source path",
						},
						"destination": map[string]any{
							"type":        "string",
							"description": "The destination path",
						},
					},
					Required: []string{"source", "destination"},
				},
			},
			Handler: t.handleMoveFile,
		},
		{
			Function: &tools.FunctionDefinition{
				Name:        "read_file",
				Description: "Read the complete contents of a file from the file system.",
				Parameters: tools.FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "The file path to read",
						},
					},
					Required: []string{"path"},
				},
			},
			Handler: t.handleReadFile,
		},
		{
			Function: &tools.FunctionDefinition{
				Name:        "read_multiple_files",
				Description: "Read the contents of multiple files simultaneously.",
				Parameters: tools.FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"paths": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "string",
							},
							"description": "Array of file paths to read",
						},
					},
					Required: []string{"paths"},
				},
			},
			Handler: t.handleReadMultipleFiles,
		},
		{
			Function: &tools.FunctionDefinition{
				Name:        "search_files",
				Description: "Recursively search for files and directories matching a pattern.",
				Parameters: tools.FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "The starting directory path",
						},
						"pattern": map[string]any{
							"type":        "string",
							"description": "The search pattern (case-insensitive)",
						},
						"excludePatterns": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "string",
							},
							"description": "Patterns to exclude from search",
						},
					},
					Required: []string{"path", "pattern"},
				},
			},
			Handler: t.handleSearchFiles,
		},
		{
			Function: &tools.FunctionDefinition{
				Name:        "search_files_content",
				Description: "Searches for text or regex patterns in the content of files matching a GLOB pattern.",
				Parameters: tools.FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "The starting directory path",
						},
						"pattern": map[string]any{
							"type":        "string",
							"description": "GLOB pattern for files to search in",
						},
						"query": map[string]any{
							"type":        "string",
							"description": "The text or regex pattern to search for",
						},
						"is_regex": map[string]any{
							"type":        "boolean",
							"description": "If true, treat query as regex; otherwise literal text",
						},
						"excludePatterns": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "string",
							},
							"description": "Patterns to exclude from search",
						},
					},
					Required: []string{"path", "pattern", "query"},
				},
			},
			Handler: t.handleSearchFilesContent,
		},
		{
			Function: &tools.FunctionDefinition{
				Name:        "write_file",
				Description: "Create a new file or completely overwrite an existing file with new content.",
				Parameters: tools.FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "The file path to write",
						},
						"content": map[string]any{
							"type":        "string",
							"description": "The content to write to the file",
						},
					},
					Required: []string{"path", "content"},
				},
			},
			Handler: t.handleWriteFile,
		},
	}, nil
}

// Security helper to check if path is allowed
func (t *FilesystemTool) isPathAllowed(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("unable to resolve absolute path: %w", err)
	}

	for _, allowedDir := range t.allowedDirectories {
		allowedAbs, err := filepath.Abs(allowedDir)
		if err != nil {
			continue
		}

		if strings.HasPrefix(absPath, allowedAbs) {
			return nil
		}
	}

	return fmt.Errorf("path %s is not within allowed directories", path)
}

// Handler implementations

func (t *FilesystemTool) handleCreateDirectory(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if err := t.isPathAllowed(args.Path); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error: %s", err)}, nil
	}

	if err := os.MkdirAll(args.Path, 0o755); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error creating directory: %s", err)}, nil
	}

	return &tools.ToolCallResult{Output: fmt.Sprintf("Directory created successfully: %s", args.Path)}, nil
}

func (t *FilesystemTool) handleDirectoryTree(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var args struct {
		Path     string `json:"path"`
		MaxDepth *int   `json:"max_depth"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if err := t.isPathAllowed(args.Path); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error: %s", err)}, nil
	}

	tree, err := t.buildDirectoryTree(args.Path, args.MaxDepth, 0)
	if err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error building directory tree: %s", err)}, nil
	}

	result, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error formatting tree: %s", err)}, nil
	}

	return &tools.ToolCallResult{Output: string(result)}, nil
}

type TreeNode struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Children []*TreeNode `json:"children,omitempty"`
}

func (t *FilesystemTool) buildDirectoryTree(path string, maxDepth *int, currentDepth int) (*TreeNode, error) {
	if maxDepth != nil && currentDepth >= *maxDepth {
		return nil, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	node := &TreeNode{
		Name: filepath.Base(path),
		Type: "file",
	}

	if info.IsDir() {
		node.Type = "directory"
		node.Children = []*TreeNode{}

		entries, err := os.ReadDir(path)
		if err != nil {
			return node, nil // Return partial result on error
		}

		for _, entry := range entries {
			childPath := filepath.Join(path, entry.Name())
			if err := t.isPathAllowed(childPath); err != nil {
				continue // Skip disallowed paths
			}

			childNode, err := t.buildDirectoryTree(childPath, maxDepth, currentDepth+1)
			if err != nil || childNode == nil {
				continue
			}
			node.Children = append(node.Children, childNode)
		}
	}

	return node, nil
}

func (t *FilesystemTool) handleEditFile(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var args struct {
		Path  string `json:"path"`
		Edits []struct {
			OldText string `json:"oldText"`
			NewText string `json:"newText"`
		} `json:"edits"`
		DryRun bool `json:"dryRun"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if err := t.isPathAllowed(args.Path); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error: %s", err)}, nil
	}

	content, err := os.ReadFile(args.Path)
	if err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error reading file: %s", err)}, nil
	}

	originalContent := string(content)
	modifiedContent := originalContent

	var changes []string
	for i, edit := range args.Edits {
		if !strings.Contains(modifiedContent, edit.OldText) {
			return &tools.ToolCallResult{Output: fmt.Sprintf("Edit %d failed: old text not found", i+1)}, nil
		}
		modifiedContent = strings.Replace(modifiedContent, edit.OldText, edit.NewText, 1)
		changes = append(changes, fmt.Sprintf("Edit %d: Replaced %d characters", i+1, len(edit.OldText)))
	}

	if args.DryRun {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Dry run completed. Changes:\n%s", strings.Join(changes, "\n"))}, nil
	}

	if err := os.WriteFile(args.Path, []byte(modifiedContent), 0o644); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error writing file: %s", err)}, nil
	}

	return &tools.ToolCallResult{Output: fmt.Sprintf("File edited successfully. Changes:\n%s", strings.Join(changes, "\n"))}, nil
}

func (t *FilesystemTool) handleGetFileInfo(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if err := t.isPathAllowed(args.Path); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error: %s", err)}, nil
	}

	info, err := os.Stat(args.Path)
	if err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error getting file info: %s", err)}, nil
	}

	fileInfo := map[string]any{
		"name":    info.Name(),
		"size":    info.Size(),
		"mode":    info.Mode().String(),
		"modTime": info.ModTime().Format(time.RFC3339),
		"isDir":   info.IsDir(),
	}

	result, err := json.MarshalIndent(fileInfo, "", "  ")
	if err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error formatting file info: %s", err)}, nil
	}

	return &tools.ToolCallResult{Output: string(result)}, nil
}

func (t *FilesystemTool) handleListAllowedDirectories(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	result, err := json.MarshalIndent(t.allowedDirectories, "", "  ")
	if err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error formatting directories: %s", err)}, nil
	}

	return &tools.ToolCallResult{Output: string(result)}, nil
}

func (t *FilesystemTool) handleListDirectory(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if err := t.isPathAllowed(args.Path); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error: %s", err)}, nil
	}

	entries, err := os.ReadDir(args.Path)
	if err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error reading directory: %s", err)}, nil
	}

	var result strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("DIR  %s\n", entry.Name()))
		} else {
			result.WriteString(fmt.Sprintf("FILE %s\n", entry.Name()))
		}
	}

	return &tools.ToolCallResult{Output: result.String()}, nil
}

func (t *FilesystemTool) handleListDirectoryWithSizes(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if err := t.isPathAllowed(args.Path); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error: %s", err)}, nil
	}

	entries, err := os.ReadDir(args.Path)
	if err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error reading directory: %s", err)}, nil
	}

	var result strings.Builder
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("DIR  %s\n", entry.Name()))
		} else {
			result.WriteString(fmt.Sprintf("FILE %s (%d bytes)\n", entry.Name(), info.Size()))
		}
	}

	return &tools.ToolCallResult{Output: result.String()}, nil
}

func (t *FilesystemTool) handleMoveFile(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var args struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if err := t.isPathAllowed(args.Source); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error (source): %s", err)}, nil
	}
	if err := t.isPathAllowed(args.Destination); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error (destination): %s", err)}, nil
	}

	if _, err := os.Stat(args.Destination); err == nil {
		return &tools.ToolCallResult{Output: "Error: destination already exists"}, nil
	}

	if err := os.Rename(args.Source, args.Destination); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error moving file: %s", err)}, nil
	}

	return &tools.ToolCallResult{Output: fmt.Sprintf("Successfully moved %s to %s", args.Source, args.Destination)}, nil
}

func (t *FilesystemTool) handleReadFile(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if err := t.isPathAllowed(args.Path); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error: %s", err)}, nil
	}

	content, err := os.ReadFile(args.Path)
	if err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error reading file: %s", err)}, nil
	}

	return &tools.ToolCallResult{Output: string(content)}, nil
}

func (t *FilesystemTool) handleReadMultipleFiles(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var args struct {
		Paths []string `json:"paths"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	var result strings.Builder
	for _, path := range args.Paths {
		if err := t.isPathAllowed(path); err != nil {
			result.WriteString(fmt.Sprintf("=== %s ===\nError: %s\n\n", path, err))
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			result.WriteString(fmt.Sprintf("=== %s ===\nError reading file: %s\n\n", path, err))
			continue
		}

		result.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", path, string(content)))
	}

	return &tools.ToolCallResult{Output: result.String()}, nil
}

func (t *FilesystemTool) handleSearchFiles(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var args struct {
		Path            string   `json:"path"`
		Pattern         string   `json:"pattern"`
		ExcludePatterns []string `json:"excludePatterns"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if err := t.isPathAllowed(args.Path); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error: %s", err)}, nil
	}

	var matches []string
	pattern := strings.ToLower(args.Pattern)

	err := filepath.WalkDir(args.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors and continue
		}

		if err := t.isPathAllowed(path); err != nil {
			return nil // Skip disallowed paths
		}

		// Check exclude patterns
		for _, exclude := range args.ExcludePatterns {
			if matched, _ := filepath.Match(exclude, filepath.Base(path)); matched {
				return nil
			}
		}

		// Case-insensitive match
		if strings.Contains(strings.ToLower(filepath.Base(path)), pattern) {
			matches = append(matches, path)
		}

		return nil
	})
	if err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error searching files: %s", err)}, nil
	}

	return &tools.ToolCallResult{Output: strings.Join(matches, "\n")}, nil
}

func (t *FilesystemTool) handleSearchFilesContent(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var args struct {
		Path            string   `json:"path"`
		Pattern         string   `json:"pattern"`
		Query           string   `json:"query"`
		IsRegex         bool     `json:"is_regex"`
		ExcludePatterns []string `json:"excludePatterns"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if err := t.isPathAllowed(args.Path); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error: %s", err)}, nil
	}

	var regex *regexp.Regexp
	if args.IsRegex {
		var err error
		regex, err = regexp.Compile(args.Query)
		if err != nil {
			return &tools.ToolCallResult{Output: fmt.Sprintf("Invalid regex pattern: %s", err)}, nil
		}
	}

	var results []string

	err := filepath.WalkDir(args.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if err := t.isPathAllowed(path); err != nil {
			return nil
		}

		// Check if file matches the pattern
		if matched, _ := filepath.Match(args.Pattern, filepath.Base(path)); !matched {
			return nil
		}

		// Check exclude patterns
		for _, exclude := range args.ExcludePatterns {
			if matched, _ := filepath.Match(exclude, filepath.Base(path)); matched {
				return nil
			}
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			var matched bool
			var matchStart, matchEnd int

			if args.IsRegex {
				if loc := regex.FindStringIndex(line); loc != nil {
					matched = true
					matchStart, matchEnd = loc[0], loc[1]
				}
			} else {
				if idx := strings.Index(line, args.Query); idx != -1 {
					matched = true
					matchStart, matchEnd = idx, idx+len(args.Query)
				}
			}

			if matched {
				preview := line
				if len(preview) > 100 {
					start := max(matchStart-20, 0)
					end := matchEnd + 20
					end = min(end, len(preview))
					preview = preview[start:end]
				}

				result := fmt.Sprintf("%s:%d:%d: %s", path, lineNum+1, matchStart+1, preview)
				results = append(results, result)
			}
		}

		return nil
	})
	if err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error searching file contents: %s", err)}, nil
	}

	return &tools.ToolCallResult{Output: strings.Join(results, "\n")}, nil
}

func (t *FilesystemTool) handleWriteFile(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if err := t.isPathAllowed(args.Path); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error: %s", err)}, nil
	}

	if err := os.WriteFile(args.Path, []byte(args.Content), 0o644); err != nil {
		return &tools.ToolCallResult{Output: fmt.Sprintf("Error writing file: %s", err)}, nil
	}

	return &tools.ToolCallResult{Output: fmt.Sprintf("File written successfully: %s (%d bytes)", args.Path, len(args.Content))}, nil
}

func (t *FilesystemTool) Start(ctx context.Context) error {
	return nil
}

func (t *FilesystemTool) Stop() error {
	return nil
}
