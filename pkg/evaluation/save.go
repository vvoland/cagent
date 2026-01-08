package evaluation

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/cagent/pkg/session"
)

func SaveRunJSON(run *EvalRun, outputDir string) (string, error) {
	return saveJSON(run, filepath.Join(outputDir, run.Name+".json"))
}

func Save(sess *session.Session, filename string) (string, error) {
	baseName := cmp.Or(filename, sess.ID)

	evalFile := filepath.Join("evals", fmt.Sprintf("%s.json", baseName))
	for number := 1; ; number++ {
		if _, err := os.Stat(evalFile); err != nil {
			break
		}

		evalFile = filepath.Join("evals", fmt.Sprintf("%s_%d.json", baseName, number))
	}

	return saveJSON(sess, evalFile)
}

func saveJSON(value any, outputPath string) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", err
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return "", err
	}

	return outputPath, nil
}
