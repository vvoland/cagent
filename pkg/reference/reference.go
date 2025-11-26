package reference

import "strings"

// OciRefToFilename converts an OCI reference to a safe, consistent filename
// Examples:
//   - "docker.io/myorg/agent:v1" -> "docker.io_myorg_agent_v1.yaml"
//   - "localhost:5000/test" -> "localhost_5000_test.yaml"
func OciRefToFilename(ociRef string) string {
	// Replace characters that are invalid in filenames with underscores
	// Keep the structure recognizable but filesystem-safe
	safe := strings.NewReplacer(
		"/", "_",
		":", "_",
		"@", "_",
		"\\", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	).Replace(ociRef)

	// Ensure it has .yaml extension
	if !strings.HasSuffix(safe, ".yaml") {
		safe += ".yaml"
	}

	return safe
}
