package environment

import "strings"

type RequiredEnvError struct {
	Missing []string
}

var _ error = &RequiredEnvError{}

func (e *RequiredEnvError) Error() string {
	return "missing required environment variables: " + strings.Join(e.Missing, ", ")
}
