package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/syndtr/gocapability/capability"
)

func TestCapabilitiesMapValid(t *testing.T) {
	for _, e := range capabilities {
		assert.Contains(t, capability.List(), e)
	}
}
