package history

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/natefinch/atomic"
)

type History struct {
	Messages []string `json:"messages"`

	path    string
	current int
}

type options struct {
	homeDir string
}

type Opt func(*options)

func WithBaseDir(dir string) Opt {
	return func(o *options) {
		o.homeDir = dir
	}
}

func New(opts ...Opt) (*History, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	homeDir := o.homeDir
	if homeDir == "" {
		var err error
		if homeDir, err = os.UserHomeDir(); err != nil {
			return nil, err
		}
	}

	h := &History{
		path:    filepath.Join(homeDir, ".cagent", "history.json"),
		current: -1,
	}

	if err := h.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return h, nil
}

func (h *History) Add(message string) error {
	// Add the message last but avoid duplicate messages
	h.Messages = slices.DeleteFunc(h.Messages, func(m string) bool {
		return m == message
	})
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

// LatestMatch returns the most recent history entry that extends the provided
// prefix, or the latest message when no prefix is supplied.
func (h *History) LatestMatch(prefix string) string {
	if prefix == "" {
		if len(h.Messages) == 0 {
			return ""
		}
		return h.Messages[len(h.Messages)-1]
	}

	for i := len(h.Messages) - 1; i >= 0; i-- {
		if strings.HasPrefix(h.Messages[i], prefix) && len(h.Messages[i]) > len(prefix) {
			return h.Messages[i]
		}
	}

	return ""
}

func (h *History) save() error {
	data, err := json.Marshal(h)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(h.path), 0o755); err != nil {
		return err
	}

	return atomic.WriteFile(h.path, bytes.NewReader(data))
}

func (h *History) load() error {
	data, err := os.ReadFile(h.path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, h)
}
