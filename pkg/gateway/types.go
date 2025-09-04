package gateway

type topLevel struct {
	Catalog Catalog `json:"registry" yaml:"registry"`
}

type Catalog map[string]Server

type Server struct {
	Secrets []Secret `json:"secrets,omitempty" yaml:"secrets,omitempty"`
}

type Secret struct {
	Name    string `json:"name" yaml:"name"`
	Env     string `json:"env" yaml:"env"`
	Example string `json:"example" yaml:"example"`
}
