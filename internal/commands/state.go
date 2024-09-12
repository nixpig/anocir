package commands

import (
	"encoding/json"
	"fmt"

	"github.com/nixpig/brownie/internal"
)

func QueryState(containerID string) error {
	state, err := internal.GetState(containerID)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	s, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	fmt.Printf("%+v\n", string(s))

	return nil
}
