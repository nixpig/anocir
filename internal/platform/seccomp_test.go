package platform

import (
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	libseccomp "github.com/seccomp/libseccomp-golang"
	"github.com/stretchr/testify/assert"
)

func TestMapSeccompAction(t *testing.T) {
	t.Run("valid seccomp action", func(t *testing.T) {
		assert.Equal(t, libseccomp.ActAllow, mapSeccompAction(specs.ActAllow))
	})

	t.Run("invalid seccomp action", func(t *testing.T) {
		assert.Equal(t, libseccomp.ActInvalid, mapSeccompAction(specs.LinuxSeccompAction("SOMETHING INVALID")))
	})
}

func TestMapSeccompOperator(t *testing.T) {
	t.Run("valid seccomp operator", func(t *testing.T) {
		assert.Equal(t, libseccomp.CompareMaskedEqual, mapSeccompOperator(specs.OpMaskedEqual))
	})

	t.Run("invalid seccomp operator", func(t *testing.T) {
		assert.Equal(t, libseccomp.CompareInvalid, mapSeccompOperator(specs.LinuxSeccompOperator("SOMETHING INVALID")))
	})
}

func TestMapSeccompArch(t *testing.T) {
	t.Run("valid seccomp architecture", func(t *testing.T) {
		assert.Equal(t, libseccomp.ArchARM, mapSeccompArch(specs.ArchARM))
	})

	t.Run("invalid seccomp architecture", func(t *testing.T) {
		assert.Equal(t, libseccomp.ArchInvalid, mapSeccompArch(specs.Arch("SOMETHING INVALID")))
	})
}
