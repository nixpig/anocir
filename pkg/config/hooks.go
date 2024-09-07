package config

type Hooks struct {
	Prestart        Hook `json:"prestart,omitempty"`
	CreateRuntime   Hook `json:"createRuntime,omitempty"`
	CreateContainer Hook `json:"createContainer,omitempty"`
	StartContainer  Hook `json:"startContainer,omitempty"`
	PostStart       Hook `json:"postStart,omitempty"`
	PostStop        Hook `json:"postStop,omitempty"`
}

type Hook struct {
	Path    string   `json:"path"`
	Args    []string `json:"args,omitempty"`
	Env     []string `json:"env,omitempty"`
	Timeout *int     `json:"timeout,omitempty"`
}
