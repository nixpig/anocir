#!/bin/sh

RUNTIME=${RUNTIME:-./brownie}

logdir=./logs

tests=(
  "default"
  "config_updates_without_affect"
  "create"
  "delete"
  "hostname"
  "kill"
  "kill_no_effect"
  "linux_devices"
  "linux_mount_label"
  "linux_rootfs_propagation"
  "linux_sysctl"
  "mounts"
  "prestart"
  "prestart_fail"
  "process"
  "process_capabilities"
  "process_oom_score_adj"
  "start"
  "state"
)

mkdir -p $logdir

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

