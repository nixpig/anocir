[![build](https://github.com/nixpig/anocir/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/nixpig/anocir/actions/workflows/build.yml)

# 🍪 anocir

[_an-oh-cheer_] ***An***other ***OCI*** ***R***untime.

**An experimental Linux container runtime, implementing the [OCI Runtime Spec](https://github.com/opencontainers/runtime-spec/blob/main/spec.md).**

The process of building this is being documented in a series of blog posts which you can read here: [Building a container runtime from scratch in Go](https://nixpig.dev/posts/container-runtime-introduction/).

![docker hello-world with anocir runtime](demo.gif)

This is a personal project for me to explore and better understand the OCI Runtime Spec. It's not production-ready, and it probably never will be, but feel free to look around! If you're looking for a production-ready alternative to `runc`, take a look at [`youki`](https://github.com/containers/youki), which I think is pretty cool.

`anocir` [passes all _passable_ tests](#progress) in the opencontainers OCI runtime test suite. That doesn't mean that `anocir` is feature-complete...yet. See below for outstanding items.

**🗒️ To do** (items remaining for _me_ to consider this 'complete')

- [ ] ~Unit tests~ Integration tests seem to be sufficing
- [ ] Implement optional [Seccomp](https://github.com/opencontainers/runtime-spec/blob/main/config-linux.md#seccomp)
- [ ] Implement optional [AppArmor](https://github.com/opencontainers/runtime-spec/blob/main/config.md#linux-process)

## Installation

> [!CAUTION]
>
> Some features may require `sudo` and make changes to your system.
>
> Given this is an experimental project, take appropriate precautions.

### Download pre-built binary

1. Go to [Releases](https://github.com/nixpig/anocir/releases/) and download the tarball for your architecture, e.g. `anocir_0.0.1_linux_amd64.tar.gz`.
1. Extract the `anocir` binary from the tarball and put somewhere in `$PATH`, e.g. `~/.local/bin`.


### Build from source

**Prerequisite:** Compiler for Go installed ([instructions](https://go.dev/doc/install)).

```
git clone git@github.com:nixpig/anocir.git
cd anocir
make build
mv tmp/bin/anocir ~/.local/bin
```

---

I'm developing `anocir` on the following environment. Even with the same set up, YMMV. 

- `Linux vagrant 6.8.0-31-generic #31-Ubuntu SMP PREEMPT_DYNAMIC Sat Apr 20 00:40:06 UTC 2024 x86_64 x86_64 x86_64 GNU/Linux`
- `go version go1.23.4 linux/amd64`
- `Docker version 27.3.1, build ce12230`

You can spin up this VM from the included `Vagrantfile`, just run `vagrant up`.

## Usage

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

View full docs by running `anocir --help` or `anocir COMMAND --help`.

## Progress

My goal is for `anocir` to (eventually) pass all tests in the [opencontainers OCI Runtime Spec tests](https://github.com/opencontainers/runtime-tools?tab=readme-ov-file#testing-oci-runtimes). Below is progress against that goal.

### ✅ Passing

Tests are run on every build in [this Github Action](https://github.com/nixpig/anocir/actions/workflows/build.yml).

- [x] default
- [x] \_\_\_
- [x] config_updates_without_affect
- [x] create
- [x] delete
- [x] hooks
- [x] hooks_stdin
- [x] hostname
- [x] kill
- [x] killsig
- [x] kill_no_effect
- [x] linux_devices
- [x] linux_masked_paths
- [x] linux_mount_label
- [x] linux_ns_itype
- [x] linux_ns_nopath
- [x] linux_ns_path
- [x] linux_ns_path_type
- [x] linux_readonly_paths
- [x] linux_rootfs_propagation
- [x] linux_sysctl
- [x] misc_props (flaky due to test suite trying to delete container before process has exiting and status updated to stopped)
- [x] mounts
- [x] poststart
- [x] poststop
- [x] prestart
- [x] prestart_fail
- [x] process
- [x] process_capabilities
- [x] process_capabilities_fail
- [x] process_oom_score_adj
- [ ] ❌ process_rlimits
- [x] process_rlimits_fail
- [x] process_user
- [x] root_readonly_true
- [x] start
- [x] state
- [x] linux_uid_mappings

### ⚠️ Unsupported tests

#### cgroups v1 & v2 support

The OCI Runtime Spec test suite provided by opencontainers [_does not_ support cgroup v2](https://github.com/opencontainers/runtime-tools/blob/6c9570a1678f3bc7eb6ef1caa9099920b7f17383/cgroups/cgroups.go#L73).

The OCI Runtime Spec test suite provided by opencontainers _does_ support cgroup v1.

`anocir` currently implements both cgroup v1 and v2. However, like `runc` and other container runtimes, the `find x cgroup` tests pass and the `get x cgroup data` tests fail.

<details>
  <summary>Full list of cgroups tests</summary>

- [ ] ~~linux_cgroups_blkio~~
- [ ] ~~linux_cgroups_cpus~~
- [ ] ~~linux_cgroups_devices~~
- [ ] ~~linux_cgroups_hugetlb~~
- [ ] ~~linux_cgroups_memory~~
- [ ] ~~linux_cgroups_network~~
- [ ] ~~linux_cgroups_pids~~
- [ ] ~~linux_cgroups_relative_blkio~~
- [ ] ~~linux_cgroups_relative_cpus~~
- [ ] ~~linux_cgroups_relative_devices~~
- [ ] ~~linux_cgroups_relative_hugetlb~~
- [ ] ~~linux_cgroups_relative_memory~~
- [ ] ~~linux_cgroups_relative_network~~
- [ ] ~~linux_cgroups_relative_pids~~
- [ ] ~~delete_resources~~
- [ ] ~~delete_only_create_resources~~

</details>

#### Broken tests

Tests failed by `runc` and other container runtimes. In some cases the tests may be broken; in others, who knows. Either way, for my purposes, parity with other runtimes is more important than passing the tests.

- [ ] ~~pidfile~~
- [ ] ~~poststart_fail~~
- [ ] ~~poststop_fail~~

Tests that 'pass' (seemingly) regardless of whether the feature has been implemented. May indicate a bad test.

- [ ] ~~linux_process_apparmor_profile~~
- [ ] ~~linux_seccomp~~

## Contributing

Feel free to leave any comments/suggestions/feedback in [issues](https://github.com/nixpig/anocir/issues).

## Inspiration

While this project was built entirely from scratch, inspiration was taken from existing runtimes, in no particular order:

- [`youki`](https://github.com/containers/youki) (Rust)
- [`pura`](https://github.com/penumbra23/pura) (Rust)
- [`runc`](https://github.com/opencontainers/runc) (Go)
- [`crun`](https://github.com/containers/crun) (C)

## License

[MIT](https://github.com/nixpig/anocir?tab=MIT-1-ov-file#readme)
