package evaluation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cagent/pkg/session"
)

func TestSaveWithCustomFilename(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir := t.TempDir()

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Create a test session
	sess := session.New()
	sess.ID = "test-session-id"

	// Test 1: Save with custom filename
	evalFile, err := Save(sess, "my-custom-eval")
	if err != nil {
		t.Fatalf("Failed to save eval: %v", err)
	}

	expectedPath := filepath.Join("evals", "my-custom-eval.json")
	if evalFile != expectedPath {
		t.Errorf("Expected file path %q, got %q", expectedPath, evalFile)
	}

	if _, err := os.Stat(evalFile); os.IsNotExist(err) {
		t.Errorf("Expected file %q to exist", evalFile)
	}

	// Test 2: Save without filename (should use session ID)
	evalFile2, err := Save(sess, "")
	if err != nil {
		t.Fatalf("Failed to save eval: %v", err)
	}

	expectedPath2 := filepath.Join("evals", sess.ID+".json")
	if evalFile2 != expectedPath2 {
		t.Errorf("Expected file path %q, got %q", expectedPath2, evalFile2)
	}

	if _, err := os.Stat(evalFile2); os.IsNotExist(err) {
		t.Errorf("Expected file %q to exist", evalFile2)
	}

	// Test 3: Save with same filename (should add _1 suffix)
	evalFile3, err := Save(sess, "my-custom-eval")
	if err != nil {
		t.Fatalf("Failed to save eval: %v", err)
	}

	expectedPath3 := filepath.Join("evals", "my-custom-eval_1.json")
	if evalFile3 != expectedPath3 {
		t.Errorf("Expected file path %q, got %q", expectedPath3, evalFile3)
	}

	if _, err := os.Stat(evalFile3); os.IsNotExist(err) {
		t.Errorf("Expected file %q to exist", evalFile3)
	}
}
