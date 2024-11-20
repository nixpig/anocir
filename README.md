[![build](https://github.com/nixpig/brownie/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/nixpig/brownie/actions/workflows/build.yml)

# 🍪 brownie

An experimental Linux container runtime, attempting to implement the [OCI Runtime Spec](https://github.com/opencontainers/runtime-spec/blob/main/spec.md). See the [Progress](#progress) section below for progress against that goal.

> [!NOTE]
>
> 📅 **October 1st, 2024**
>
> ✅ `brownie` passes all 270 _default_ tests in the [opencontainers OCI runtime test suite](https://github.com/opencontainers/runtime-tools?tab=readme-ov-file#testing-oci-runtimes).

This is a personal project for me to explore and better understand the OCI Runtime Spec to support other projects I'm working on. It's not production-ready, and it probably never will be, but feel free to look around! If you're looking for a production-ready alternative to `runc`, take a look at [`youki`](https://github.com/containers/youki), which I think is pretty cool.

## Installation

> [!CAUTION]
>
> This is an experimental project. It requires `sudo` and will make changes to your system. Take appropriate precautions.

I'm developing `brownie` on the following environment. Even with the same set up, YMMV. Maybe I'll create a Vagrant box in future.

- `go version go1.23.0 linux/amd64`
- `Linux 6.10.2-arch1-1 x86_64 GNU/Linux`

### Build from source

**Prerequisite:** Compiler for Go installed ([instructions](https://go.dev/doc/install)).

```
git clone git@github.com:nixpig/brownie.git
cd brownie
make build
mv tmp/bin/brownie ~/.local/bin
```

## Usage

### Docker

> [!IMPORTANT]
>
> Using `brownie` as a drop in replacement for `runc` in Docker is currently broken and I don't know why. FWIW, other runtimes seem to also be broken, so maybe it's just 'on my machine'.
>
> Until I get around to investigating and fixing this, you may get issues, should you choose to try it.

By default, the Docker daemon uses the `runc` container runtime. `brownie` can be used as a drop-in replacement for `runc`.

You can find detailed instructions on how to configure alternative runtimes in the [Docker docs](https://docs.docker.com/reference/cli/dockerd/#configure-container-runtimes). If you just want to quickly experiment, the following should suffice:

```
# 1. Stop any running Docker service
sudo systemctl stop docker.service

# 2. Start the Docker Daemon with added brownie runtime
sudo dockerd --add-runtime brownie=PATH_TO_BROWNIE_BINARY

# 3. Run a container using the brownie runtime
docker run -it --runtime brownie busybox sh

```

### CLI

The `brownie` CLI is a wrapper around the [brownie core library](#library) and implements the [OCI Runtime Command Line Interface](https://github.com/opencontainers/runtime-tools/blob/master/docs/command-line-interface.md) spec.

#### `brownie create`

Create a new container.

```
Usage:
  brownie create [flags] CONTAINER_ID

Examples:
  brownie create busybox

Flags:
  -b, --bundle string           Path to bundle directory
  -s, --console-socket string   Console socket
  -h, --help                    help for create
  -p, --pid-file string         File to write container PID to
```

#### `brownie start`

Start an existing container.

```
Usage:
  brownie start [flags] CONTAINER_ID

Examples:
  brownie start busybox

Flags:
  -h, --help   help for start
```

#### `brownie kill`

Send a signal to a running container.

```
Usage:
  brownie kill [flags] CONTAINER_ID SIGNAL

Examples:
  brownie kill busybox 9

Flags:
  -h, --help   help for kill
```

#### `brownie delete`

Delete a container.

```
Usage:
  brownie delete [flags] CONTAINER_ID

Examples:
  brownie delete busybox

Flags:
  -f, --force   force delete
  -h, --help    help for delete
```

#### `brownie state`

Get the state of a container.

```
Usage:
  brownie state [flags] CONTAINER_ID

Examples:
  brownie state busybox

Flags:
  -h, --help   help for state
```

### Library

The `container` package of `brownie` can be used directly as a library (in the same way that the CLI does).

The consumer will be responsible for all of the 'bookkeeping', e.g. tracking the location of container bundles, reexec process.

#### Example

```shell
go get github.com/nixpig/brownie/container
```

```go
package main

import "github.com/nixpig/brownie/container"

func main() {
  // TODO: example usage
}
```

## Progress

My goal is for `brownie` to (eventually) pass all tests in the [opencontainers OCI Runtime Spec tests](https://github.com/opencontainers/runtime-tools?tab=readme-ov-file#testing-oci-runtimes). Below is progress against that goal.

### ✅ Passing

Tests are run on every build in [this Github Action](https://github.com/nixpig/brownie/actions/workflows/build.yml).

- [x] default
- [x] \_\_\_
- [x] config_updates_without_affect
- [x] create
- [x] delete
- [x] hooks_stdin
- [x] hostname
- [x] kill
- [x] killsig
- [x] kill_no_effect
- [x] linux_devices
- [x] linux_masked_paths
- [x] linux_mount_label
- [x] linux_ns_itype
- [x] linux_readonly_paths
- [x] linux_rootfs_propagation
- [x] linux_sysctl
- [x] mounts
- [x] poststart
- [x] poststop
- [x] prestart
- [x] prestart_fail
- [x] process
- [x] process_capabilities
- [x] process_capabilities_fail
- [x] process_oom_score_adj
- [x] process_user
- [x] root_readonly_true
- [x] start
- [x] state

### ⚠️ To do

- [ ] delete_only_create_resources
- [ ] hooks
- [ ] linux_ns_nopath
- [ ] linux_ns_path
- [ ] linux_ns_path_type
- [ ] linux_process_apparmor_profile
- [ ] linux_seccomp
- [ ] linux_uid_mappings
- [ ] pidfile
- [ ] poststart_fail
- [ ] poststop_fail
- [ ] process_rlimits
- [ ] process_rlimits_fail

### Unsupported tests

#### linux_cgroups\_\*

The OCI Runtime Spec test suite provided by opencontainers [appears not to support cgroupv2](https://github.com/opencontainers/runtime-tools/blob/6c9570a1678f3bc7eb6ef1caa9099920b7f17383/cgroups/cgroups.go#L73), thus all of the `linux_cgroups_*` tests and a few others are unable to run. For example:

```
TAP version 13
not ok 1 - find devices
# cgroupv2 is not supported yet
Clean: Delete:  exit status 1
1..1
```

<details>
  <summary>Full list of cgroups tests</summary>

- linux_cgroups_blkio
- linux_cgroups_cpus
- linux_cgroups_devices
- linux_cgroups_hugetlb
- linux_cgroups_memory
- linux_cgroups_network
- linux_cgroups_pids
- linux_cgroups_relative_blkio
- linux_cgroups_relative_cpus
- linux_cgroups_relative_devices
- linux_cgroups_relative_hugetlb
- linux_cgroups_relative_memory
- linux_cgroups_relative_network
- linux_cgroups_relative_pids

</details>

#### Other unsupported tests

- misc_props (flaky due to test suite trying to delete container before process has exiting and status updated to stopped)
- delete_resources (depends on cgroupv2, same as linux_cgroups\_\* tests)

## Contributing

Given this is an exploratory personal project, I'm not interested in taking code contributions. However, if you have any comments/suggestions/feedback, do feel free to leave them in [issues](https://github.com/nixpig/brownie/issues).

## Inspiration

While this project was built entirely from scratch, inspiration was taken from existing runtimes, in no particular order:

- [`youki`](https://github.com/containers/youki) (Rust)
- [`pura`](https://github.com/penumbra23/pura) (Rust)
- [`runc`](https://github.com/opencontainers/runc) (Go)
- [`crun`](https://github.com/containers/crun) (C)

## License

[MIT](https://github.com/nixpig/brownie?tab=MIT-1-ov-file#readme)
