package state

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nixpig/brownie/internal/database"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

const stateFilename = "state.json"

type State struct {
	Version     string
	ID          string
	Bundle      string
	Annotations map[string]string
	Status      specs.ContainerState
	PID         int
}

func New(
	id string,
	bundle string,
	status specs.ContainerState,
) *State {
	return &State{
		Version:     pkg.OCIVersion,
		ID:          id,
		Bundle:      bundle,
		Annotations: map[string]string{},
		Status:      status,
	}
}

func Load(root string, log *zerolog.Logger) (*State, error) {
	b, err := os.ReadFile(
		filepath.Join(root, stateFilename),
	)
	if err != nil {
		return nil, fmt.Errorf("read container state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(b, &state); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal state in state loader")
		return nil, fmt.Errorf("parse state: %w", err)
	}

	return &state, nil
}

func (s *State) Save() error {
	db := database.New()
	if err := db.Connect(); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	_, err := db.Conn.Exec(
		`update containers_ set 
		status_ = $status,
		pid_ = $pid,
		bundle_ = $bundle,
		version_ = $version
		where id_ = $id`,
		sql.Named("status", s.Status),
		sql.Named("id", s.ID),
		sql.Named("pid", s.PID),
		sql.Named("bundle", s.Bundle),
		sql.Named("version", s.Version),
	)
	if err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	return nil
}
