#!/bin/bash

if [[ -z ${RUNTIME} ]]; then
  echo "'RUNTIME' not set."
  exit 1
fi

if [[ -z ./runtimetest ]]; then
  echo "'runtimetest' not available."
  echo "Try running in 'runtime-tools' directory."
  exit 1
fi

logdir=/var/log/runtime-tools
mkdir -p $logdir

tests=(
    # "misc_props" # ❗️ (flaky due to test suite trying to delete container before process has exiting and status updated to stopped)

    # "linux_uid_mappings" # ❌ should be fixable

    # ✅ passing!
    "default"
    "create"
    "delete"
    "hooks"
    "hooks_stdin"
    "hostname"
    "kill"
    "kill_no_effect"
    "killsig"
    "linux_devices"
    "linux_masked_paths"
    "linux_mount_label"
    "linux_ns_itype"
    "linux_ns_nopath"
    "linux_ns_path"
    "linux_ns_path_type"
    "linux_readonly_paths"
    "linux_rootfs_propagation"
    "linux_sysctl"
    "mounts"
    "poststart"
    "poststop"
    "prestart"
    "prestart_fail"
    "process"
    "process_capabilities"
    "process_capabilities_fail"
    "process_oom_score_adj"
    "process_rlimits_fail"
    "process_user"
    "root_readonly_true"
    "start"
    "state"

    # ---

    # ❗ ️cgroups tests
    # "linux_cgroups_blkio"
    # "linux_cgroups_cpus"
    # "linux_cgroups_devices"
    # "linux_cgroups_hugetlb"
    # "linux_cgroups_memory"
    # "linux_cgroups_network"
    # "linux_cgroups_pids"
    # "linux_cgroups_relative_blkio"
    # "linux_cgroups_relative_cpus"
    # "linux_cgroups_relative_devices"
    # "linux_cgroups_relative_hugetlb"
    # "linux_cgroups_relative_memory"
    # "linux_cgroups_relative_network"
    # "linux_cgroups_relative_pids"
    # "delete_resources"
    # "delete_only_create_resources"
    
    # ---

    # ❗️ tests that also fail in runc and other runtimes
    # "pidfile"
    # "poststart_fail"
    # "poststop_fail"
    # "process_rlimits" # ❌ also fails in brownie

    # ---

    # ❗️ tests that 'pass' (seemingly) regardless of whether the feature has been implemented
    # "linux_process_apparmor_profile"
    # "linux_seccomp"
)

# run tests
for test in "${tests[@]}"; do
  ./validation/${test}/${test}.t 2>&1 | tee ${logdir}/${test}.log
done

# check for failures
total_failures=0
for test in "${tests[@]}"; do
  failures=$(grep -F "not ok" ${logdir}/${test}.log | wc -l)

  if [ 0 -ne $failures ]; then 
    total_failures=$(($total_failures + $failures))
    echo "${test} - $failures"
  fi
done

if [ 0 -ne $total_failures ]; then
  echo "Total failures: $total_failures"
  exit 1
fi
