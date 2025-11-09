package platform_test

import (
	"testing"

	"github.com/nixpig/anocir/internal/platform"
	"github.com/stretchr/testify/assert"
)

func TestNamespaceMappings(t *testing.T) {
	assert.Len(t, platform.NamespaceEnvs, 8)
	assert.Len(t, platform.NamespaceFlags, 8)

	for key := range platform.NamespaceFlags {
		_, ok := platform.NamespaceEnvs[key]

		assert.True(t, ok, "missing namespace env for '%s'", key)
	}
}
