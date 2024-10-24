#!/bin/sh
echo "" > results.tap
RUNTIME=${RUNTIME:./brownie}
./validation/default/default.t 2>&1 | tee -a results.tap
./validation/config_updates_without_affect/config_updates_without_affect.t 2>&1 | tee -a results.tap
./validation/create/create.t 2>&1 | tee -a results.tap
./validation/delete/delete.t 2>&1 | tee -a results.tap
./validation/hostname/hostname.t 2>&1 | tee -a results.tap
./validation/kill/kill.t 2>&1 | tee -a results.tap
./validation/kill_no_effect/kill_no_effect.t 2>&1 | tee -a results.tap
./validation/linux_mount_label/linux_mount_label.t 2>&1 | tee -a results.tap
./validation/linux_sysctl/linux_sysctl.t 2>&1 | tee -a results.tap
# ./validation/pidfile/pidfile.t 2>&1 | tee -a results.tap
./validation/process/process.t 2>&1 | tee -a results.tap
./validation/process_capabilities/process_capabilities.t 2>&1 | tee -a results.tap
./validation/start/start.t 2>&1 | tee -a results.tap
./validation/state/state.t 2>&1 | tee -a results.tap
(! grep -F "not ok" results.tap)
