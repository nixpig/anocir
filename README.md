# 🍪 anocir

[![release](https://img.shields.io/github/v/release/nixpig/anocir)](https://github.com/nixpig/anocir/releases/latest)
[![build](https://github.com/nixpig/anocir/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/nixpig/anocir/actions/workflows/build.yml)
[![oci-integration](https://github.com/nixpig/anocir/actions/workflows/oci-integration.yml/badge.svg?branch=main)](https://github.com/nixpig/anocir/actions/workflows/oci-integration.yml)
[![containerd-integration](https://github.com/nixpig/anocir/actions/workflows/containerd-integration.yml/badge.svg?branch=main)](https://github.com/nixpig/anocir/actions/workflows/containerd-integration.yml)
[![cni-integration](https://github.com/nixpig/anocir/actions/workflows/cni-integration.yml/badge.svg?branch=main)](https://github.com/nixpig/anocir/actions/workflows/cni-integration.yml)
<!-- [![cri-integration](https://github.com/nixpig/anocir/actions/workflows/cri-integration.yml/badge.svg?branch=main)](https://github.com/nixpig/anocir/actions/workflows/cri-integration.yml) -->

[_an-oh-cheer_] ***An***other ***OCI*** ***R***untime.

**An experimental Linux container runtime, implementing the [OCI Runtime Spec](https://github.com/opencontainers/runtime-spec/blob/main/spec.md).**

> [!NOTE]
> 
> This is a personal project to explore how container runtimes work. It's not production-ready. If you're looking for a production-ready alternative to `runc`, I think [`youki`](https://github.com/containers/youki) is pretty cool.

The process of building this is being documented in a series of blog posts which you can read here: [Building a container runtime from scratch in Go](https://nixpig.dev/posts/container-runtime-introduction/).

## 🚀 Quick start

1. Download the tarball for your architecture from [Releases](https://github.com/nixpig/anocir/releases/).
1. Extract the `anocir` binary from the tarball into somewhere in `$PATH`, e.g. `~/.local/bin`.
1. View the docs by running `anocir --help` or `anocir COMMAND --help`.

## 👩‍💻 Usage

> [!CAUTION]
>
> Some features may require `sudo` and make changes to your system. Take appropriate precautions.

### Docker

By default, the Docker daemon uses the `runc` container runtime. `anocir` can be used as a drop-in replacement for `runc`.

You can find detailed instructions on how to configure alternative runtimes in the [Docker docs](https://docs.docker.com/reference/cli/dockerd/#configure-container-runtimes). If you just want to quickly experiment, the following should suffice: 

```bash
# 1. Stop any running Docker service
sudo systemctl stop docker.service

# 2. Start the Docker Daemon with added anocir runtime
sudo dockerd --add-runtime anocir=PATH_TO_ANOCIR_BINARY

# 3. Run a container using the anocir runtime
docker run -it --runtime anocir busybox sh

```

### CLI

The `anocir` CLI implements the [OCI Runtime Command Line Interface](https://github.com/opencontainers/runtime-tools/blob/master/docs/command-line-interface.md) spec.

You can view the docs by running `anocir --help` or `anocir [COMMAND] --help`.

## ⚒️ Contributing

Feel free to leave any comments/suggestions/feedback in [issues](https://github.com/nixpig/anocir/issues).

## 💡 Inspiration

While this project was built entirely from scratch, inspiration was taken from existing runtimes, in no particular order:

- [`youki`](https://github.com/containers/youki) (Rust)
- [`pura`](https://github.com/penumbra23/pura) (Rust)
- [`runc`](https://github.com/opencontainers/runc) (Go)
- [`crun`](https://github.com/containers/crun) (C)

## 📃 License

[MIT](https://github.com/nixpig/anocir?tab=MIT-1-ov-file#readme)
