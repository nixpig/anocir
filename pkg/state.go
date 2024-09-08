package pkg

type status string

const (
	Creating status = "creating"
	Created  status = "created"
	Running  status = "running"
	Stopped  status = "stopped"
)

type State struct {
	OCIVersion  string            `json:"ociVersion"`
	ID          string            `json:"id"`
	Status      status            `json:"status"`
	PID         *int              `json:"pid,omitempty"`
	Bundle      string            `json:"bundle"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
