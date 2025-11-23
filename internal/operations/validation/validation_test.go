package validation_test

import (
	"testing"

	"github.com/nixpig/anocir/internal/operations/validation"
	"github.com/stretchr/testify/assert"
)

func TestContainerIDValidation(t *testing.T) {
	scenarios := map[string]struct {
		id    string
		valid bool
	}{
		"test alphabetic only": {
			id:    "abcXYZabcXYZ",
			valid: true,
		},
		"test numeric only": {
			id:    "1234567890",
			valid: true,
		},
		"test alphanumeric": {
			id:    "012abcxyzABCXYZ789",
			valid: true,
		},
		"test underscores": {
			id:    "test_container_id",
			valid: true,
		},
		"test hyphens": {
			id:    "test-container-id",
			valid: true,
		},
		"test mixed": {
			id:    "test_container-90",
			valid: true,
		},
		"too max length": {
			id:    "abcxyzABCXYZ7890abcxyzABCXYZ7890abcxyzABCXYZ7890abcxyzABCXYZ7890",
			valid: true,
		},
		"test empty": {
			id:    "",
			valid: false,
		},
		"test invalid specials": {
			id:    "a$b^c*",
			valid: false,
		},
		"test invalid length": {
			id:    "abcxyzABCXYZ7890abcxyzABCXYZ7890abcxyzABCXYZ7890abcxyzABCXYZ7890x",
			valid: false,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			assert.Equal(t, data.valid, validation.ContainerID(data.id) == nil)
		})
	}
}
