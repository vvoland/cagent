package teamloader

import (
	"fmt"
	"os"
	"path/filepath"
)

type AgentSource interface {
	Name() string
	ParentDir() string
	Read() ([]byte, error)
}

// fileSource is used to load an agent configuration from a YAML file.
type fileSource struct {
	path string
}

func NewFileSource(path string) AgentSource {
	return fileSource{
		path: path,
	}
}

func (a fileSource) Name() string {
	return filepath.Base(a.path)
}

func (a fileSource) ParentDir() string {
	return filepath.Dir(a.path)
}

func (a fileSource) Read() ([]byte, error) {
	parentDir := a.ParentDir()
	fs, err := os.OpenRoot(parentDir)
	if err != nil {
		return nil, fmt.Errorf("opening filesystem %s: %w", parentDir, err)
	}

	fileName := a.Name()
	data, err := fs.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", fileName, err)
	}

	return data, nil
}

// bytesSource is used to load an agent configuration from a []byte.
type bytesSource struct {
	data []byte
}

func NewBytesSource(data []byte) AgentSource {
	return bytesSource{
		data: data,
	}
}

func (a bytesSource) Name() string {
	return ""
}

func (a bytesSource) ParentDir() string {
	return ""
}

func (a bytesSource) Read() ([]byte, error) {
	return a.data, nil
}
