package anosys

import (
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

func TestSetRlimits(t *testing.T) {
	rlimits := []specs.POSIXRlimit{
		{
			Type: "RLIMIT_NOFILE",
			Hard: 1024,
			Soft: 1024,
		},
	}

	err := SetRlimits(rlimits)
	assert.NoError(t, err)
}
