[![build](https://github.com/nixpig/brownie/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/nixpig/brownie/actions/workflows/build.yml)

# 🍪 brownie

An experimental Linux container runtime.

> [!NOTE]
> As of October 1st, 2024, `brownie` passes all 270 _default_ tests in the [opencontainers OCI runtime test suite](https://github.com/opencontainers/runtime-tools?tab=readme-ov-file#testing-oci-runtimes).

This is currently a personal project. As such, I'm not interested in taking code contributions. However, if you have any comments/suggestions/feedback, do feel free to leave them in [issues](https://github.com/nixpig/brownie/issues).

## Installation

`brownie` has been tested on:

- `go version go1.23.0 linux/amd64`
- `Linux 6.10.2-arch1-1 x86_64 GNU/Linux`

> [!CAUTION]
> This is an experimental project. It will make changes to your system. Take appropriate precautions.

### Build from source

**Prerequisite:** Compiler for Go installed ([instructions](https://go.dev/doc/install)).

1. `git clone git@github.com:nixpig/brownie.git`
1. `cd brownie`
1. `make build`
1. `mv tmp/bin/brownie ~/.local/bin`

## Usage

The `brownie` CLI implements the [OCI Runtime Command Line Interface](https://github.com/opencontainers/runtime-tools/blob/master/docs/command-line-interface.md) spec.

## License

[MIT](https://github.com/nixpig/brownie?tab=MIT-1-ov-file#readme)
