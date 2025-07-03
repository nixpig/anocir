# OCI

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

