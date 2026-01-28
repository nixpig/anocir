package validation

import (
	"errors"
	"fmt"
)

// maxLength is the maximum length of the container ID. It's somewhat
// arbitrary, but 64 should be plenty long enough.
const maxLength = 64

// ContainerID validates that the provided ID is not empty, does not exceed the
// maxLength, and only contains alphanumeric, '_' and '-' characters.
func ContainerID(id string) error {
	if id == "" {
		return errors.New("empty container ID")
	}

	if len(id) > maxLength {
		return fmt.Errorf("max length is %d chars", maxLength)
	}

	for _, c := range id {
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '-' ||
			c == '_') {
			return errors.New(
				"may only contain alphanumeric, '-' and '_' chars",
			)
		}
	}

	return nil
}
