package history

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type History struct {
	Messages []string `json:"messages"`
	path     string
	current  int
}

func New() (*History, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	historyDir := filepath.Join(homeDir, ".cagent")
	if err := os.MkdirAll(historyDir, 0o755); err != nil {
		return nil, err
	}

	historyFile := filepath.Join(historyDir, "history.json")

	h := &History{
		path:    historyFile,
		current: -1,
	}

	if err := h.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return h, nil
}

func (h *History) Add(message string) error {
	h.Messages = append(h.Messages, message)
	h.current = len(h.Messages)
	return h.save()
}

func (h *History) Previous() string {
	if len(h.Messages) == 0 {
		return ""
	}

	// If we're at -1 (initial state), start from the end
	if h.current == -1 {
		h.current = len(h.Messages) - 1
		return h.Messages[h.current]
	}

	// If we're at the beginning, stay there
	if h.current <= 0 {
		return h.Messages[0]
	}

	h.current--
	return h.Messages[h.current]
}

func (h *History) Next() string {
	if len(h.Messages) == 0 {
		return ""
	}

	if h.current >= len(h.Messages)-1 {
		h.current = len(h.Messages)
		return ""
	}

	h.current++
	return h.Messages[h.current]
}

func (h *History) save() error {
	data, err := json.Marshal(h)
	if err != nil {
		return err
	}
	return os.WriteFile(h.path, data, 0o644)
}

func (h *History) load() error {
	data, err := os.ReadFile(h.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, h)
}
