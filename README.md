[![build](https://github.com/nixpig/anocir/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/nixpig/anocir/actions/workflows/build.yml)
[![Coverage Status](https://coveralls.io/repos/github/nixpig/anocir/badge.svg?branch=main)](https://coveralls.io/github/nixpig/anocir?branch=main)
[![release](https://img.shields.io/github/v/release/nixpig/anocir)](https://github.com/nixpig/anocir/releases/latest)
[![license](https://img.shields.io/github/license/nixpig/anocir)](LICENSE)

# 🍪 anocir

[_an-oh-cheer_] ***An***other ***OCI*** ***R***untime.

**An experimental Linux container runtime, implementing the [OCI Runtime Spec](https://github.com/opencontainers/runtime-spec/blob/main/spec.md).**

![docker hello-world with anocir runtime](demo.gif)

The process of building this is being documented in a series of blog posts which you can read here: [Building a container runtime from scratch in Go](https://nixpig.dev/posts/container-runtime-introduction/).


### 📈 Project Status

> [!NOTE]
> 
> This is a personal project to explore how container runtimes work. It's not production-ready. If you're looking for a production-ready alternative to `runc`, I think [`youki`](https://github.com/containers/youki) is pretty cool.

#### Roadmap

- [x] Pass all tests in the [opencontainers OCI Runtime Spec test suite](https://github.com/opencontainers/runtime-tools?tab=readme-ov-file#testing-oci-runtimes).
- [ ] Pass all tests in the [Kubernetes CRI validation test suite](https://github.com/kubernetes-sigs/cri-tools/blob/master/docs/validation.md).
- [ ] Implement optional [Seccomp](https://github.com/opencontainers/runtime-spec/blob/main/config-linux.md#seccomp).
- [ ] Implement optional [AppArmor](https://github.com/opencontainers/runtime-spec/blob/main/config.md#linux-process).

## 🚀 Quick start

1. Download the tarball for your architecture from [Releases](https://github.com/nixpig/anocir/releases/).
1. Extract the `anocir` binary from the tarball into somewhere in `$PATH`, e.g. `~/.local/bin`.
1. View docs by running `anocir --help` or `anocir COMMAND --help`.

## 👩‍💻 Usage

> [!CAUTION]
>
> Some features may require `sudo` and make changes to your system. Take appropriate precautions.

### 🐋 Docker

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

### 💻️ CLI

The `anocir` CLI implements the [OCI Runtime Command Line Interface](https://github.com/opencontainers/runtime-tools/blob/master/docs/command-line-interface.md) spec. You can view the docs by running `anocir --help` or `anocir [COMMAND] --help`.

## 🤝 Contributing

**Feel free to leave any comments/suggestions/feedback in [issues](https://github.com/nixpig/anocir/issues).**

### ⚒️ Build from source

**Prerequisite:** Compiler for Go installed ([instructions](https://go.dev/doc/install)).

1. `git clone git@github.com:nixpig/anocir.git`
1. `cd anocir`
1. `make build`
1. `mv tmp/bin/anocir ~/.local/bin`

I'm developing `anocir` on the following environment. Even with the same set up, YMMV. 

- `Linux vagrant 6.8.0-31-generic #31-Ubuntu SMP PREEMPT_DYNAMIC Sat Apr 20 00:40:06 UTC 2024 x86_64 x86_64 x86_64 GNU/Linux`
- `go version go1.25.5 linux/amd64`
- `Docker version 27.3.1, build ce12230`

You can spin up this VM from the included `Vagrantfile`, just run `vagrant up`.

### 🎯 Run the OCI test suite

See [OCI.md](OCI.md) for details of tests.

1. `vagrant up --provision`
1. `vagrant ssh`
1. `cd /anocir`
1. `make build`
1. `cd /anocir/test/runtime-tools`
1. `make runtimetest validation-executables`
1. `sudo RUNTIME=../../tmp/bin/anocir ../scripts/oci-integration.sh`

## 💡 Inspiration

While this project was built entirely from scratch, inspiration was taken from existing runtimes, in no particular order:

- [`youki`](https://github.com/containers/youki) (Rust)
- [`pura`](https://github.com/penumbra23/pura) (Rust)
- [`runc`](https://github.com/opencontainers/runc) (Go)
- [`crun`](https://github.com/containers/crun) (C)

## 📃 License

[MIT](https://github.com/nixpig/anocir?tab=MIT-1-ov-file#readme)
