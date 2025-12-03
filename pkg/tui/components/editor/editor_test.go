package editor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileParts(t *testing.T) {
	t.Parallel()

	t.Run("appends file content as attachment", func(t *testing.T) {
		t.Parallel()

		// Create a temp file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(tmpFile, []byte("file content here"), 0o644))

		ref := "@" + tmpFile
		e := &editor{fileRefs: []string{ref}}
		content := "analyze " + ref

		result := e.fileParts(content)

		assert.Contains(t, result[ref], "file content here")
		assert.Nil(t, e.fileRefs, "fileRefs should be cleared after expansion")
	})

	t.Run("multiple file attachments", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		file1 := filepath.Join(tmpDir, "first.go")
		file2 := filepath.Join(tmpDir, "second.go")
		require.NoError(t, os.WriteFile(file1, []byte("package first"), 0o644))
		require.NoError(t, os.WriteFile(file2, []byte("package second"), 0o644))

		ref1 := "@" + file1
		ref2 := "@" + file2
		e := &editor{fileRefs: []string{ref1, ref2}}
		content := "compare " + ref1 + " with " + ref2

		result := e.fileParts(content)

		assert.Contains(t, result[ref1], "package first")
		assert.Contains(t, result[ref2], "package second")
	})

	t.Run("skips nonexistent files", func(t *testing.T) {
		t.Parallel()

		ref := "@/nonexistent/path/file.txt"
		e := &editor{fileRefs: []string{ref}}
		content := "analyze " + ref

		result := e.fileParts(content)

		assert.Empty(t, result)
		assert.Nil(t, e.fileRefs, "fileRefs should still be cleared")
	})

	t.Run("skips directories", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		ref := "@" + tmpDir
		e := &editor{fileRefs: []string{ref}}
		content := "analyze " + ref

		result := e.fileParts(content)

		assert.Empty(t, result)
		assert.Nil(t, e.fileRefs, "fileRefs should be cleared after expansion")
	})

	t.Run("mixed valid and invalid refs", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		validFile := filepath.Join(tmpDir, "valid.txt")
		require.NoError(t, os.WriteFile(validFile, []byte("valid content"), 0o644))

		validRef := "@" + validFile
		invalidRef := "@/nonexistent/file.txt"
		e := &editor{fileRefs: []string{validRef, invalidRef}}
		content := "check " + validRef + " and " + invalidRef

		result := e.fileParts(content)

		assert.Contains(t, result[validRef], "valid content")
		assert.NotContains(t, result, invalidRef)
		assert.Nil(t, e.fileRefs, "fileRefs should be cleared after expansion")
	})
}

func TestTryAddFileRef(t *testing.T) {
	t.Parallel()

	t.Run("adds valid file path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "manual.txt")
		require.NoError(t, os.WriteFile(tmpFile, []byte("content"), 0o644))

		e := &editor{fileRefs: nil}
		e.tryAddFileRef("@" + tmpFile)

		require.Len(t, e.fileRefs, 1)
		assert.Equal(t, "@"+tmpFile, e.fileRefs[0])
	})

	t.Run("ignores @mentions without path characters", func(t *testing.T) {
		t.Parallel()

		e := &editor{fileRefs: nil}
		e.tryAddFileRef("@username")

		assert.Nil(t, e.fileRefs, "@mentions without / or . should be ignored")
	})

	t.Run("ignores nonexistent files", func(t *testing.T) {
		t.Parallel()

		e := &editor{fileRefs: nil}
		e.tryAddFileRef("@/nonexistent/file.txt")

		assert.Nil(t, e.fileRefs, "nonexistent files should be ignored")
	})

	t.Run("ignores directories", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		e := &editor{fileRefs: nil}
		e.tryAddFileRef("@" + tmpDir)

		assert.Nil(t, e.fileRefs, "directories should be ignored")
	})

	t.Run("avoids duplicates", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "file.go")
		require.NoError(t, os.WriteFile(tmpFile, []byte("content"), 0o644))

		ref := "@" + tmpFile
		e := &editor{fileRefs: []string{ref}}
		e.tryAddFileRef(ref)

		assert.Len(t, e.fileRefs, 1, "should not add duplicate")
	})

	t.Run("combines with completion refs", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		completedFile := filepath.Join(tmpDir, "completed.go")
		manualFile := filepath.Join(tmpDir, "manual.go")
		require.NoError(t, os.WriteFile(completedFile, []byte("package completed"), 0o644))
		require.NoError(t, os.WriteFile(manualFile, []byte("package manual"), 0o644))

		// completedFile was selected via completion
		e := &editor{fileRefs: []string{"@" + completedFile}}
		// User typed manualFile and cursor left the word
		e.tryAddFileRef("@" + manualFile)

		require.Len(t, e.fileRefs, 2)
		assert.Contains(t, e.fileRefs, "@"+completedFile)
		assert.Contains(t, e.fileRefs, "@"+manualFile)

		// Verify both get attached
		content := "compare @" + completedFile + " with @" + manualFile
		result := e.fileParts(content)

		assert.Contains(t, result["@"+completedFile], "package completed")
		assert.Contains(t, result["@"+manualFile], "package manual")
	})
}
