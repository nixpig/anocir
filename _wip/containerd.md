# Misc. tests

All of the `TestContainer...` tests are passing. 

The following tests are still failing/flaky and need attention.

```
--- FAIL: TestUserNamespaces (32.33s)
    --- FAIL: TestUserNamespaces/WritableRootFS (10.58s)
    --- FAIL: TestUserNamespaces/ReadonlyRootFS (4.10s)
    --- FAIL: TestUserNamespaces/CheckSetUidBit (3.98s)
--- FAIL: TestIssue9103 (0.17s)
    --- FAIL: TestIssue9103/should_be_stopped_status_if_init_has_been_killed (0.04s)
--- FAIL: TestIssue10589 (0.08s)
--- FAIL: TestRestartMonitor (10.29s)
    --- FAIL: TestRestartMonitor/Paused_Task (0.11s)
--- FAIL: TestShimSockLength (1.59s)
--- FAIL: TestRuntimeInfo (0.04s)
```
