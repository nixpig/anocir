package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/nixpig/brownie/pkg"
)

func State(containerID string) error {
	state, err := pkg.GetState(containerID)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	s, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("unmarshal state: %w", err)
	}

	fmt.Printf("%+v\n", string(s))

	return nil
}
