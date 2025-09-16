package oci

type Servers struct {
	MCPServers map[string]Server `json:"mcpServers"`
}

type Server struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Env     []string `json:"env"`
}
