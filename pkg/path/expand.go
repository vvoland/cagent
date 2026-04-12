package path

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath expands shell-like patterns in a file path:
//   - ~ or ~/ at the start is replaced with the user's home directory
//   - Environment variables like ${HOME} or $HOME are expanded
func ExpandPath(p string) string {
	if p == "" {
		return p
	}

	// Expand environment variables
	p = os.ExpandEnv(p)

	// Expand tilde to home directory
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~"+string(filepath.Separator)) {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}

	return p
}
