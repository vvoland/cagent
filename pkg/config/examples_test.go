package config

import (
	"testing"

	"io/fs"
	"path/filepath"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseExamples(t *testing.T) {
	var files []string
	err := filepath.WalkDir("../../examples", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".yaml" {
			files = append(files, path)
		}
		return nil
	})
	require.NoError(t, err)
	assert.NotEmpty(t, files)

	for _, file := range files {
		cfg, err := loadConfig(file)

		require.NoError(t, err)
		require.NotEmpty(t, cfg.Agents["root"].Instruction, "Instruction should not be empty in %s", file)
	}
}
