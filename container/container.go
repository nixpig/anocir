package container

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nixpig/brownie/container/lifecycle"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

const initSockFilename = "init.sock"
const containerSockFilename = "container.sock"

type Container struct {
	State *State
	Spec  *specs.Spec

	forkCmd *exec.Cmd
	initIPC ipcCtrl
	db      *sql.DB
}

type State struct {
	Version     string
	ID          string
	Bundle      string
	Annotations map[string]string
	Status      specs.ContainerState
	PID         int
}

type ipcCtrl struct {
	ch     chan []byte
	closer func() error
}

func New(
	id string,
	bundle string,
	status specs.ContainerState,
	log *zerolog.Logger,
	db *sql.DB,
) (*Container, error) {
	_, err := db.Query(`select id_ from containers_ where id_ = $id`, sql.Named("id", id))
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf(
			"container already exists (%s): %w",
			id, err,
		)
	}

	b, err := os.ReadFile(filepath.Join(bundle, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("read container config: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(b, &spec); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal spec")
		return nil, fmt.Errorf("parse container config: %w", err)
	}

	if spec.Linux == nil {
		return nil, errors.New("only linux containers are supported")
	}

	absBundlePath, err := filepath.Abs(bundle)
	if err != nil {
		return nil, fmt.Errorf("construct absolute bundle path: %w", err)
	}

	state := &State{
		Version:     pkg.OCIVersion,
		ID:          id,
		Bundle:      absBundlePath,
		Annotations: map[string]string{},
		Status:      status,
	}

	// TODO: save to database
	query := `insert into containers_ (
		id_, version_, bundle_, pid_, status_, config_
	) values (
		$id, $version, $bundle, $pid, $status, $config
	)`

	if _, err := db.Exec(
		query,
		sql.Named("id", id),
		sql.Named("version", state.Version),
		sql.Named("bundle", state.Bundle),
		sql.Named("pid", state.PID),
		sql.Named("status", state.Status),
		sql.Named("config", string(b)),
	); err != nil {
		return nil, fmt.Errorf("insert into db: %w", err)
	}

	cntr := Container{
		State: state,
		Spec:  &spec,
		db:    db,
	}

	if err := cntr.Save(); err != nil {
		return nil, fmt.Errorf("save newly created container: %w", err)
	}

	return &cntr, nil
}

func Load(id string, log *zerolog.Logger, db *sql.DB) (*Container, error) {
	state := State{}
	var c string

	row := db.QueryRow(`select id_, version_, bundle_, pid_, status_, config_ from containers_ where id_ = $id`, sql.Named("id", id))

	if err := row.Scan(
		&state.ID,
		&state.Version,
		&state.Bundle,
		&state.PID,
		&state.Status,
		&c,
	); err != nil {
		return nil, fmt.Errorf("scan container to struct: %w", err)
	}

	conf := specs.Spec{}
	if err := json.Unmarshal([]byte(c), &conf); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal state in loader")
		return nil, fmt.Errorf("unmarshall state to struct: %w", err)
	}

	cntr := &Container{
		State: &state,
		Spec:  &conf,
		db:    db,
	}

	if err := cntr.RefreshState(); err != nil {
		log.Error().Err(err).Msg("failed to refresh state")
		return nil, fmt.Errorf("refresh state: %w", err)
	}

	return cntr, nil
}

func (c *Container) RefreshState() error {
	b, err := os.ReadFile(filepath.Join(c.State.Bundle, "state.json"))
	if err != nil {
		fmt.Println("WARNING: unable to refresh from state file")
		return nil
	}

	if err := json.Unmarshal(b, c.State); err != nil {
		return fmt.Errorf("unmarshall refreshed state: %w", err)
	}

	return nil
}

func (c *Container) SaveState() error {
	b, err := json.Marshal(c.State)
	if err != nil {
		return err
	}
	if err := os.WriteFile("/state.json", b, 0644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}

	return nil
}

func (c *Container) Save() error {
	_, err := c.db.Exec(
		`update containers_ set 
		status_ = $status,
		pid_ = $pid,
		bundle_ = $bundle,
		version_ = $version
		where id_ = $id`,
		sql.Named("status", c.State.Status),
		sql.Named("id", c.State.ID),
		sql.Named("pid", c.State.PID),
		sql.Named("bundle", c.State.Bundle),
		sql.Named("version", c.State.Version),
	)
	if err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	b, err := json.Marshal(c.State)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(c.State.Bundle, "state.json"), b, 0644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}

	return nil
}

func (c *Container) Clean() error {
	return os.RemoveAll(c.State.Bundle)
}

func (c *Container) ExecHooks(lifecycleHook string) error {
	if c.Spec.Hooks == nil {
		return nil
	}

	var specHooks []specs.Hook
	switch lifecycleHook {
	case "prestart":
		//lint:ignore SA1019 marked as deprecated, but still required by OCI Runtime integration tests
		specHooks = c.Spec.Hooks.Prestart
	case "createRuntime":
		specHooks = c.Spec.Hooks.CreateRuntime
	case "createContainer":
		specHooks = c.Spec.Hooks.CreateContainer
	case "startContainer":
		specHooks = c.Spec.Hooks.StartContainer
	case "poststart":
		specHooks = c.Spec.Hooks.Poststart
	case "poststop":
		specHooks = c.Spec.Hooks.Poststop
	}

	return lifecycle.ExecHooks(specHooks)
}

func (c *Container) CanBeStarted() bool {
	return c.State.Status == specs.StateCreated
}

func (c *Container) CanBeKilled() bool {
	return c.State.Status == specs.StateRunning ||
		c.State.Status == specs.StateCreated
}

func (c *Container) CanBeDeleted() bool {
	return c.State.Status == specs.StateStopped
}
