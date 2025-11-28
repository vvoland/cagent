package builtin

const (
	maxOutputSize = 30000

	maxFiles = 100
)

func limitOutput(output string) string {
	if len(output) > maxOutputSize {
		return output[:maxOutputSize] + "\n\n[Output truncated: exceeded 30,000 character limit]"
	}
	return output
}
