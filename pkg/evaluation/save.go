package evaluation

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/cagent/pkg/session"
)

func Save(sess *session.Session, filename string) (string, error) {
	if err := os.MkdirAll("evals", 0o755); err != nil {
		return "", err
	}

	// Use provided filename if given, otherwise default to session ID
	baseName := cmp.Or(filename, sess.ID)

	evalFile := filepath.Join("evals", fmt.Sprintf("%s.json", baseName))
	for number := 1; ; number++ {
		if _, err := os.Stat(evalFile); err != nil {
			break
		}

		evalFile = filepath.Join("evals", fmt.Sprintf("%s_%d.json", baseName, number))
	}

	file, err := os.Create(evalFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return evalFile, encoder.Encode(sess)
}
