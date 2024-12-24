package commands

import (
	"encoding/json"
	"fmt"

	"github.com/nixpig/brownie/features"
)

func Features() (string, error) {
	f, err := json.Marshal(features.Get())
	if err != nil {
		return "", fmt.Errorf("marshal features: %w", err)
	}

	return string(f), nil
}
