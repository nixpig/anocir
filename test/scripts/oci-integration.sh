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
    "create"
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
