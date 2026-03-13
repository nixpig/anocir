package oci

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatProcessOutput(t *testing.T) {
	t.Parallel()

	scenarios := map[string]struct {
		processes []int
		format    string
		output    string
		assertErr assert.ErrorAssertionFunc
	}{
		"format processes as table": {
			processes: []int{23, 69, 12345},
			format:    "table",
			output:    "23\n69\n12345\n",
			assertErr: assert.NoError,
		},
		"format processes as json": {
			processes: []int{23, 69, 12345},
			format:    "json",
			output:    "[23,69,12345]",
			assertErr: assert.NoError,
		},
		"invalid format type": {
			processes: []int{23, 69, 12345},
			format:    "invalid",
			output:    "",
			assertErr: assert.Error,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()

			output, err := formatProcessesOutput(data.format, data.processes)
			data.assertErr(t, err)
			assert.Equal(t, data.output, output)
		})
	}
}
