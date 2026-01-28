package root

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldMonitorStdin(t *testing.T) {
	t.Parallel()

	t.Run("returns false when ppid is 0", func(t *testing.T) {
		t.Parallel()

		// Create a pipe to simulate stdin from parent
		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer r.Close()
		defer w.Close()

		// ppid=0 means we're init or something weird - should not monitor
		result := shouldMonitorStdin(0, r)
		assert.False(t, result, "should not monitor stdin when ppid is 0")
	})

	t.Run("returns false when ppid is 1", func(t *testing.T) {
		t.Parallel()

		// Create a pipe to simulate stdin from parent
		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer r.Close()
		defer w.Close()

		// ppid=1 means parent is init (common in containers) - should not monitor
		result := shouldMonitorStdin(1, r)
		assert.False(t, result, "should not monitor stdin when ppid is 1 (init)")
	})

	t.Run("returns false when stdin is nil", func(t *testing.T) {
		t.Parallel()

		result := shouldMonitorStdin(123, nil)
		assert.False(t, result, "should not monitor stdin when stdin is nil")
	})

	t.Run("returns false when stdin is /dev/null", func(t *testing.T) {
		t.Parallel()

		devNull, err := os.Open(os.DevNull)
		require.NoError(t, err)
		defer devNull.Close()

		// /dev/null is not a pipe, so should not monitor
		result := shouldMonitorStdin(123, devNull)
		assert.False(t, result, "should not monitor stdin when stdin is /dev/null")
	})

	t.Run("returns false when stdin is a regular file", func(t *testing.T) {
		t.Parallel()

		// Create a temp file
		f, err := os.CreateTemp(t.TempDir(), "test-stdin-*")
		require.NoError(t, err)
		defer f.Close()

		// Regular file is not a pipe, so should not monitor
		result := shouldMonitorStdin(123, f)
		assert.False(t, result, "should not monitor stdin when stdin is a regular file")
	})

	t.Run("returns true when stdin is a pipe and ppid > 1", func(t *testing.T) {
		t.Parallel()

		// Create a pipe to simulate stdin from parent
		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer r.Close()
		defer w.Close()

		// ppid > 1 and stdin is a pipe - should monitor
		result := shouldMonitorStdin(123, r)
		assert.True(t, result, "should monitor stdin when stdin is a pipe and ppid > 1")
	})

	t.Run("returns true with various valid ppids", func(t *testing.T) {
		t.Parallel()

		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer r.Close()
		defer w.Close()

		// Test various ppid values > 1
		for _, ppid := range []int{2, 100, 1000, 65535} {
			result := shouldMonitorStdin(ppid, r)
			assert.True(t, result, "should monitor stdin when ppid is %d", ppid)
		}
	})
}
