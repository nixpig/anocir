package commands

import (
	"encoding/json"
	"fmt"

	"github.com/nixpig/brownie/internal/container"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type StateOpts struct {
	ID string
}

type StateCLI struct {
	Version string `json:"ociVersion"`
	// ID is the container ID
	ID string `json:"id"`
	// Status is the runtime status of the container.
	Status specs.ContainerState `json:"status"`
	// Pid is the process ID for the container process.
	Pid int `json:"pid,omitempty"`
	// Bundle is the path to the container's bundle directory.
	Bundle string `json:"bundle"`

	Rootfs string `json:"rootfs"`

	// Annotations are key values associated with the container.
	Annotations map[string]string `json:"annotations,omitempty"`
}

func stateToCliState(state *container.ContainerState) StateCLI {
	return StateCLI{
		Version:     state.Version,
		ID:          state.ID,
		Status:      state.Status,
		Pid:         state.Pid,
		Bundle:      state.Bundle,
		Annotations: state.Annotations,
	}
}

func State(opts *StateOpts, log *zerolog.Logger) (string, error) {
	container, err := container.LoadContainer(opts.ID)
	if err != nil {
		return "", fmt.Errorf("load container: %w", err)
	}

	state := container.State

	s, err := json.Marshal(stateToCliState(state))
	if err != nil {
		log.Error().Err(err).Msg("marshal state")
		return "", fmt.Errorf("marshal state: %w", err)
	}

	// if _, err := os.Stdout.Write(s); err != nil {
	// 	log.Error().Err(err).Msg("write state to stdout")
	// }

	return string(s), nil
}
