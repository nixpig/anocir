package pkg

type status string

const (
	creating status = "creating"
	created  status = "created"
	running  status = "running"
	stopped  status = "stopped"
)

type State struct {
	OCIVersion  string            `json:"ociVersion"`
	ID          string            `json:"id"`
	Status      status            `json:"status"`
	PID         int               `json:"pid"`
	Bundle      string            `json:"bundle"`
	Annotations map[string]string `json:"annotations"`
}

func (s *State) Save() error {
	// TODO: save to file, e.g. state.json??
	return nil
}
