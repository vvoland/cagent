package fsx

import (
	"os"
	"path/filepath"
)

type TreeNode struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Children []*TreeNode `json:"children,omitempty"`
}

func DirectoryTree(path string, isPathAllowed func(string) error, shouldIgnore func(string) bool, maxDepth int) (*TreeNode, error) {
	return directoryTree(path, isPathAllowed, shouldIgnore, maxDepth, 0)
}

func directoryTree(path string, isPathAllowed func(string) error, shouldIgnore func(string) bool, maxDepth, currentDepth int) (*TreeNode, error) {
	if maxDepth > 0 && currentDepth >= maxDepth {
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
			if err := isPathAllowed(childPath); err != nil {
				continue // Skip disallowed paths
			}

			// Skip if should be ignored (VCS, gitignore, etc.)
			if shouldIgnore != nil && shouldIgnore(childPath) {
				continue
			}

			childNode, err := directoryTree(childPath, isPathAllowed, shouldIgnore, maxDepth, currentDepth+1)
			if err != nil || childNode == nil {
				continue
			}
			node.Children = append(node.Children, childNode)
		}
	}

	return node, nil
}

func ListDirectory(path string, maxDepth int, shouldIgnore func(string) bool) ([]string, error) {
	tree, err := DirectoryTree(path, func(string) error { return nil }, shouldIgnore, maxDepth)
	if err != nil {
		return nil, err
	}

	var files []string
	CollectFilesFromTree(tree, "", &files)
	return files, nil
}

// CollectFilesFromTree recursively collects file paths from a DirectoryTree.
// Pass basePath="" for relative paths, or a parent directory for absolute paths.
func CollectFilesFromTree(node *TreeNode, basePath string, files *[]string) {
	if node == nil {
		return
	}
	fullPath := filepath.Join(basePath, node.Name)
	switch node.Type {
	case "file":
		*files = append(*files, fullPath)
	case "directory":
		for _, child := range node.Children {
			CollectFilesFromTree(child, fullPath, files)
		}
	}
}
