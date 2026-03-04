package toolinstall

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// templateData holds the variables available in aqua asset name templates.
type templateData struct {
	Version string
	OS      string
	Arch    string
	Format  string
}

var templateFuncs = template.FuncMap{
	"trimV": func(s string) string { return strings.TrimPrefix(s, "v") },
}

// renderTemplate renders a Go template string with the given data.
func renderTemplate(tmplStr string, data templateData) (string, error) {
	tmpl, err := template.New("asset").Funcs(templateFuncs).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing template %q: %w", tmplStr, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template %q: %w", tmplStr, err)
	}

	return buf.String(), nil
}

// extractRelease extracts files from a release asset stream based on format.
// For tar.gz, the response body is streamed directly through gzip → tar.
// For zip, the body is spooled to a temporary file (zip requires random access).
// Raw/single-binary formats are handled by the caller before reaching this function.
func extractRelease(body io.ReadCloser, destDir, format string, files []PackageFile, tmplData templateData) error {
	switch format {
	case "tar.gz", "tgz":
		return extractTarGz(body, destDir, files, tmplData)
	case "zip":
		return extractZipFromStream(body, destDir, files, tmplData)
	default:
		return fmt.Errorf("unsupported archive format: %s", format)
	}
}

// writeRawBinary writes a raw (non-archived) binary stream directly to destPath
// with executable permissions.
func writeRawBinary(r io.Reader, destPath string) error {
	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("creating raw binary %s: %w", destPath, err)
	}

	_, copyErr := io.Copy(f, r) //nolint:gosec // binary size bounded by GitHub release asset limits
	closeErr := f.Close()

	if copyErr != nil {
		return fmt.Errorf("writing raw binary %s: %w", destPath, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("closing raw binary %s: %w", destPath, closeErr)
	}

	return nil
}

// extractTarGz extracts files from a tar.gz archive.
// It reads from the provided reader in a streaming fashion (gzip → tar)
// without buffering the entire archive in memory.
func extractTarGz(r io.Reader, destDir string, files []PackageFile, tmplData templateData) error {
	gzReader, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("extracting tar.gz: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	fileMap, err := buildFileMap(files, tmplData)
	if err != nil {
		return err
	}

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("extracting tar.gz: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		destName, ok := matchFile(header.Name, fileMap)
		if !ok {
			continue
		}

		destPath, err := safePath(destDir, destName)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}

		_, copyErr := io.Copy(f, tarReader) //nolint:gosec // archive size bounded by GitHub release asset limits
		f.Close()
		if copyErr != nil {
			return copyErr
		}
	}

	return nil
}

// extractZip extracts files from a zip archive.
// It requires random access via io.ReaderAt; callers should provide either
// an *os.File (spooled to a temp file) or a *bytes.Reader.
func extractZip(ra io.ReaderAt, size int64, destDir string, files []PackageFile, tmplData templateData) error {
	reader, err := zip.NewReader(ra, size)
	if err != nil {
		return fmt.Errorf("extracting zip: %w", err)
	}

	fileMap, err := buildFileMap(files, tmplData)
	if err != nil {
		return err
	}

	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}

		destName, ok := matchFile(f.Name, fileMap)
		if !ok {
			continue
		}

		destPath, err := safePath(destDir, destName)
		if err != nil {
			return err
		}

		if err := extractZipFile(f, destPath); err != nil {
			return err
		}
	}

	return nil
}

func extractZipFile(f *zip.File, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc) //nolint:gosec // archive size bounded by GitHub release asset limits
	return err
}

// extractZipFromStream spools an io.Reader to a temporary file and then
// extracts the zip archive. This avoids holding the entire archive in memory
// while satisfying zip's requirement for random access (io.ReaderAt).
func extractZipFromStream(r io.Reader, destDir string, files []PackageFile, tmplData templateData) error {
	tmpFile, err := os.CreateTemp("", "cagent-zip-*.zip")
	if err != nil {
		return fmt.Errorf("creating temp file for zip: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	size, err := io.Copy(tmpFile, r) //nolint:gosec // archive size bounded by GitHub release asset limits
	if err != nil {
		return fmt.Errorf("spooling zip to temp file: %w", err)
	}

	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seeking temp file: %w", err)
	}

	return extractZip(tmpFile, size, destDir, files, tmplData)
}

// buildFileMap builds a map from rendered src paths to destination binary names.
func buildFileMap(files []PackageFile, data templateData) (map[string]string, error) {
	m := make(map[string]string, len(files))
	for _, f := range files {
		src := f.Src
		if src != "" {
			rendered, err := renderTemplate(src, data)
			if err != nil {
				return nil, fmt.Errorf("rendering file src template: %w", err)
			}
			src = rendered
		}
		name := f.Name
		if name == "" {
			name = filepath.Base(src)
		}
		m[src] = name
	}
	return m, nil
}

// matchFile checks if an archive entry matches any expected file.
// An empty fileMap means extract everything.
func matchFile(entryName string, fileMap map[string]string) (string, bool) {
	if len(fileMap) == 0 {
		return filepath.Base(entryName), true
	}

	for src, dest := range fileMap {
		if entryName == src || filepath.Base(entryName) == filepath.Base(src) {
			return dest, true
		}
	}

	return "", false
}

// errPathTraversal is returned when an archive entry attempts to write
// outside the destination directory (Zip Slip / Tar Slip attack).
var errPathTraversal = errors.New("archive entry attempts path traversal")

// safePath validates that joining destDir with name stays within destDir.
// Returns the cleaned absolute path or an error on path traversal.
func safePath(destDir, name string) (string, error) {
	destPath := filepath.Join(destDir, name)
	cleanDest := filepath.Clean(destPath)
	cleanDir := filepath.Clean(destDir) + string(os.PathSeparator)

	if !strings.HasPrefix(cleanDest, cleanDir) {
		return "", fmt.Errorf("%w: %q resolves to %q (outside %q)", errPathTraversal, name, cleanDest, destDir)
	}

	return cleanDest, nil
}
