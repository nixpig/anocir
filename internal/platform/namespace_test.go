package platform_test

import (
	"testing"

	"github.com/nixpig/anocir/internal/anosys"
	"github.com/stretchr/testify/assert"
)

func TestNamespaceMappings(t *testing.T) {
	assert.Len(t, anosys.NamespaceEnvs, 8)
	assert.Len(t, anosys.NamespaceFlags, 8)

	for key := range anosys.NamespaceFlags {
		_, ok := anosys.NamespaceEnvs[key]

		assert.True(t, ok, "missing namespace env for '%s'", key)
	}
}
