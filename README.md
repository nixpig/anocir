[![build](https://github.com/nixpig/brownie/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/nixpig/brownie/actions/workflows/build.yml)

# 🍪 brownie

An experimental Linux container runtime, implementing the OCI Runtime Spec.

> [!NOTE]
> As of October 1st, 2024, `brownie` passes all 270 _default_ tests in the [opencontainers OCI runtime test suite](https://github.com/opencontainers/runtime-tools?tab=readme-ov-file#testing-oci-runtimes).

This is a personal project for me to explore and better understand the [OCI Runtime Spec](https://github.com/opencontainers/runtime-spec/blob/main/spec.md) to support other projects I'm working on. The state of the code is as you would expect for something experimental and exploratory, but feel free to have a look around!

## Installation

I'm developing `brownie` on the following environment. Even with the same set up, YMMV. Maybe I'll create a Vagrant box in future.

- `go version go1.23.0 linux/amd64`
- `Linux 6.10.2-arch1-1 x86_64 GNU/Linux`

> [!CAUTION]
> This is an experimental project. It requires `sudo` and will make changes to your system. Take appropriate precautions.

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

By default, the Docker daemon uses the runc container runtime. `brownie` can be used as a drop-in replacement for runc.

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

The `brownie` CLI implements the [OCI Runtime Command Line Interface](https://github.com/opencontainers/runtime-tools/blob/master/docs/command-line-interface.md) spec.

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

## To do

- [ ] Pass _all_ OCI spec tests.
- [ ] Networking and port forwarding.

## Contributing

Given this is an exploratory personal project, I'm not interested in taking code contributions. However, if you have any comments/suggestions/feedback, do feel free to leave them in [issues](https://github.com/nixpig/brownie/issues).

## Inspiration

While this project was built entirely from scratch, inspiration was taken from existing runtimes:

- [youki](https://github.com/containers/youki) (Rust)
- [pura](https://github.com/penumbra23/pura) (Rust)
- [runc](https://github.com/opencontainers/runc) (Go)
- [crun](https://github.com/containers/crun) (C)

## License

[MIT](https://github.com/nixpig/brownie?tab=MIT-1-ov-file#readme)
