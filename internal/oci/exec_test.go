package oci

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/nixpig/anocir/internal/container"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUser(t *testing.T) {
	scenarios := map[string]struct {
		user  string
		uid   int
		gid   int
		valid bool
	}{
		"empty user": {
			user:  "",
			uid:   0,
			gid:   0,
			valid: true,
		},
		"uid only": {
			user:  "1000",
			uid:   1000,
			gid:   0,
			valid: true,
		},
		"uid and gid": {
			user:  "1000:1001",
			uid:   1000,
			gid:   1001,
			valid: true,
		},
		"missing uid": {
			user:  ":1001",
			uid:   0,
			gid:   0,
			valid: false,
		},
		"missing gid": {
			user:  "1000:",
			uid:   0,
			gid:   0,
			valid: false,
		},
		"invalid uid only": {
			user:  "invalid",
			uid:   0,
			gid:   0,
			valid: false,
		},
		"invalid uid, valid gid": {
			user:  "invalid:1001",
			uid:   0,
			gid:   0,
			valid: false,
		},
		"valid uid, invalid gid": {
			user:  "1000:invalid",
			uid:   0,
			gid:   0,
			valid: false,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			uid, gid, err := parseUser(data.user)

			assert.Equal(t, data.uid, uid)
			assert.Equal(t, data.gid, gid)
			assert.Equal(t, data.valid, err == nil)
		})
	}
}

func TestParseProcessFile(t *testing.T) {
	scenarios := map[string]struct {
		path     string
		testData map[string]any
		wantErr  bool
		wantOpts *container.ExecOpts
	}{
		"invalid file path": {
			path:     "",
			testData: map[string]any{},
			wantErr:  true,
			wantOpts: &container.ExecOpts{},
		},
		"invalid process data": {
			path: "process.json",
			testData: map[string]any{
				"invalid": "data",
				"user":    []string{"bork"},
			},
			wantErr:  true,
			wantOpts: &container.ExecOpts{},
		},
		"valid process data": {
			path: "process.json",
			testData: map[string]any{
				"terminal": true,
				"user": map[string]any{
					"uid":            1000,
					"gid":            1000,
					"additionalGids": []int{100, 200},
				},
				"args": []string{"/bin/sh", "-c", "echo hello"},
				"env":  []string{"PATH=/usr/bin", "TERM=xterm"},
				"cwd":  "/home/user",
				"capabilities": map[string]any{
					"bounding": []string{"CAP_NET_BIND_SERVICE", "CAP_KILL"},
				},
				"noNewPrivileges": true,
				"apparmorProfile": "default",
				"selinuxLabel":    "system_u:system_r:container_t:s0",
			},
			wantErr: false,
			wantOpts: &container.ExecOpts{
				Cwd:            "/home/user",
				Env:            []string{"PATH=/usr/bin", "TERM=xterm"},
				Args:           []string{"/bin/sh", "-c", "echo hello"},
				UID:            1000,
				GID:            1000,
				NoNewPrivs:     true,
				AppArmor:       "default",
				TTY:            true,
				ProcessLabel:   "system_u:system_r:container_t:s0",
				Capabilities:   []string{"CAP_NET_BIND_SERVICE", "CAP_KILL"},
				AdditionalGIDs: []int{100, 200},
			},
		},
	}

	for scenario, data := range scenarios {
		tempDir := t.TempDir()
		if data.path != "" {
			f, err := os.Create(filepath.Join(tempDir, data.path))
			require.NoError(t, err)

			err = json.NewEncoder(f).Encode(data.testData)
			require.NoError(t, err)
		}

		t.Run(scenario, func(t *testing.T) {
			gotOpts := &container.ExecOpts{}
			err := parseProcessFile(gotOpts, filepath.Join(tempDir, data.path))
			if data.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, data.wantOpts, gotOpts)
		})
	}
}

func TestParseProcessFlags(t *testing.T) {
	scenarios := map[string]struct {
		flags    func() *pflag.FlagSet
		args     []string
		wantErr  bool
		wantOpts *container.ExecOpts
	}{
		"empty flags and empty args": {
			flags: func() *pflag.FlagSet {
				fs := execCmd().Flags()
				fs.Parse([]string{})
				return fs
			},
			args:     []string{},
			wantErr:  false,
			wantOpts: &container.ExecOpts{},
		},
		"args with empty flags": {
			flags: func() *pflag.FlagSet {
				fs := execCmd().Flags()
				fs.Parse([]string{})
				return fs
			},
			args:    []string{"container_id", "/bin/sh", "-c", "echo hello"},
			wantErr: false,
			wantOpts: &container.ExecOpts{
				Args: []string{"/bin/sh", "-c", "echo hello"},
			},
		},

		"flags with empty args": {
			flags: func() *pflag.FlagSet {
				fs := execCmd().Flags()
				fs.Parse([]string{
					"--cwd", "/home/user",
					"--no-new-privs",
					"--apparmor", "default",
					"--tty",
					"--process-label", "system_u:system_r:container_t:s0",
					"--user", "1000:1000",
					"--additional-gids", "100,200",
					"-e", "PATH=/usr/bin",
					"-e", "TERM=xterm",
					"--cap", "CAP_NET_BIND_SERVICE",
					"--cap", "CAP_KILL",
				})
				return fs
			},
			args:    []string{},
			wantErr: false,
			wantOpts: &container.ExecOpts{
				Cwd:            "/home/user",
				UID:            1000,
				GID:            1000,
				NoNewPrivs:     true,
				AppArmor:       "default",
				TTY:            true,
				ProcessLabel:   "system_u:system_r:container_t:s0",
				AdditionalGIDs: []int{100, 200},
				Env:            []string{"PATH=/usr/bin", "TERM=xterm"},
				Capabilities:   []string{"CAP_NET_BIND_SERVICE", "CAP_KILL"},
			},
		},
		"flags and args": {
			flags: func() *pflag.FlagSet {
				fs := execCmd().Flags()
				fs.Parse([]string{
					"--cwd", "/home/user",
					"--no-new-privs",
					"--apparmor", "default",
					"--tty",
					"--process-label", "system_u:system_r:container_t:s0",
					"--user", "1000:1000",
					"--additional-gids", "100,200",
					"-e", "PATH=/usr/bin",
					"-e", "TERM=xterm",
					"--cap", "CAP_NET_BIND_SERVICE",
					"--cap", "CAP_KILL",
				})
				return fs
			},
			args:    []string{"container_id", "/bin/sh", "-c", "echo hello"},
			wantErr: false,
			wantOpts: &container.ExecOpts{
				Cwd:            "/home/user",
				Args:           []string{"/bin/sh", "-c", "echo hello"},
				UID:            1000,
				GID:            1000,
				NoNewPrivs:     true,
				AppArmor:       "default",
				TTY:            true,
				ProcessLabel:   "system_u:system_r:container_t:s0",
				AdditionalGIDs: []int{100, 200},
				Env:            []string{"PATH=/usr/bin", "TERM=xterm"},
				Capabilities:   []string{"CAP_NET_BIND_SERVICE", "CAP_KILL"},
			},
		},
		"invalid user flag": {
			flags: func() *pflag.FlagSet {
				fs := execCmd().Flags()
				fs.Parse([]string{"--user", "invalid:user"})
				return fs
			},
			args:     []string{},
			wantErr:  true,
			wantOpts: &container.ExecOpts{},
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			gotOpts := &container.ExecOpts{}
			err := parseProcessFlags(gotOpts, data.flags(), data.args)
			if data.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.EqualValues(t, data.wantOpts, gotOpts)
		})
	}
}
