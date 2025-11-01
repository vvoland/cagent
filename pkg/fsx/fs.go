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

func DirectoryTree(path string, isPathAllowed func(string) error, maxDepth, currentDepth int) (*TreeNode, error) {
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

			childNode, err := DirectoryTree(childPath, isPathAllowed, maxDepth, currentDepth+1)
			if err != nil || childNode == nil {
				continue
			}
			node.Children = append(node.Children, childNode)
		}
	}

	return node, nil
}

func ListDirectory(path string, maxDept int) ([]string, error) {
	var files []string

	tree, err := DirectoryTree(path, func(string) error { return nil }, maxDept, 0)
	if err != nil {
		return nil, err
	}

	var traverse func(node *TreeNode, currentPath string)
	traverse = func(node *TreeNode, currentPath string) {
		newPath := filepath.Join(currentPath, node.Name)
		switch node.Type {
		case "file":
			files = append(files, newPath)
		case "directory":
			for _, child := range node.Children {
				traverse(child, newPath)
			}
		}
	}

	traverse(tree, "")
	return files, nil
}
