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
	Version     string               `json:"ociVersion"`
	ID          string               `json:"id"`
	Status      specs.ContainerState `json:"status"`
	Pid         int                  `json:"pid,omitempty"`
	Bundle      string               `json:"bundle"`
	Rootfs      string               `json:"rootfs"`
	Annotations map[string]string    `json:"annotations,omitempty"`
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
	log.Info().Any("opts", opts).Msg("run state command")
	log.Info().Str("id", opts.ID).Msg("load container")
	cntr, err := container.LoadContainer(opts.ID)
	if err != nil {
		log.Error().Err(err).Str("id", opts.ID).Msg("failed to load container")
		return "", fmt.Errorf("load container: %w", err)
	}

	state := cntr.State

	log.Info().Any("state", state).Msg("parse container state")
	s, err := json.Marshal(stateToCliState(state))
	if err != nil {
		log.Error().Err(err).Msg("failed to parse container state")
		return "", fmt.Errorf("marshal state: %w", err)
	}

	log.Info().Any("state", s).Msg("returning container state")
	return string(s), nil
}
