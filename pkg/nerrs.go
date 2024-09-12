package pkg

import "errors"

var (
	ErrContainerExists = errors.New("container with specified ID already exists")
)
