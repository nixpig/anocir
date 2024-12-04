#!/bin/sh

RUNTIME=${RUNTIME:-./brownie}

logdir=./logs

tests=(
  "default"
  "config_updates_without_affect"
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
  "linux_process_apparmor_profile" # test passes even though feature hasn't been implemented
  "linux_readonly_paths"
  "linux_rootfs_propagation"
  "linux_seccomp" # test passes even though feature isn't implemented
  "linux_sysctl"
  "linux_uid_mappings"
  "misc_props" # flaky due to test suite trying to delete container before process has exited and status updated to stopped
  "mounts"
# "pidfile" # runc also hangs on this
  "poststart"
# "poststart_fail" # runc also fails this
  "poststop"
# "poststop_fail" # runc fails this
  "prestart"
  "prestart_fail"
  "process"
  "process_capabilities"
  "process_capabilities_fail"
  "process_oom_score_adj"
  "process_rlimits"
  "process_rlimits_fail"
  "process_user"
  "root_readonly_true"
  "start"
  "state"

  # Unsupported; see note in readme.
  # ---------------------------
  # "delete_resources"
  # "delete_only_create_resources"
  # "linux_cgroups_blkio" # use of features deprecated in Linux kernel 5.0
  # "linux_cgroups_cpus"
  # "linux_cgroups_devices"
  # "linux_cgroups_hugetlb"
  # "linux_cgroups_memory"
  # "linux_cgroups_network"
  # "linux_cgroups_pids"
  # "linux_cgroups_relative_blkio" # use of features deprecated in Linux kernel 5.0
  # "linux_cgroups_relative_cpus"
  # "linux_cgroups_relative_devices"
  # "linux_cgroups_relative_hugetlb"
  # "linux_cgroups_relative_memory"
  # "linux_cgroups_relative_network"
  # "linux_cgroups_relative_pids"
)

mkdir -p $logdir

mkdir -p /sys/fs/cgroup/systemd
mount -t cgroup -o none,name=systemd cgroup /sys/fs/cgroup/systemd

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

