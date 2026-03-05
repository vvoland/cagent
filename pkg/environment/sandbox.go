package environment

import "os"

const sandboxVMIDEnv = "SANDBOX_VM_ID"

// InSandbox reports whether cagent is running inside a Docker sandbox.
// Detection relies on the SANDBOX_VM_ID environment variable that Docker
// Desktop sets in every sandbox VM.
func InSandbox() bool {
	return os.Getenv(sandboxVMIDEnv) != ""
}
