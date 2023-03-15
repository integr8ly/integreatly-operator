## General Guidelines

### Test Users

- rhmi-developer group user

  `test-userXX` users should be automatically added to this group after logging in for the first time. Execute `oc get group rhmi-developers` as kubeadmin to verify user is in group.

- dedicated-admin group user

  `customer-adminXX` users should be part of this group. Execute `oc get group dedicated-admins` as kubeadmin to verify user is in group.

- kubeadmin user

  `kubeadmin` user credentials should be found in the test epic description. If not, ping the `integreatly-qe` channel.

### Report the Result

Resolution Legend:

- Passed => Done
- Failure => Rejected
- Blocked => Deferred
- Skipped => Won't Do

#### Passed

If the test succeeds, resolve this task with `Done`.

> Attention: Never resolve a test as passed if not all steps have passed, because otherwise, it would
> not be retested in the next round.

#### Failed

If the test fails, [report the bug](#report-a-bug), write a **comment** with the reason why it failed,
and resolve this task with `Can't Do`.

#### Blocked

All tests that can't be executed or will not be executed in this round and should be executed in the next
round. They should be marked as `Test Pending`.

Ideally, the task should be flagged as blocked before start testing, therefore the tester should never
use the `Test Pending` resolution.

#### Skipped

All tests that have Passed in the previous round and that have not been executed in this test round.
They should be marked as `Won't Do`

> Attention: Never resolve a test as skipped if the previous didn't pass, because otherwise, it would
> not be retested in the next round.

### Report a Bug

Before opening new Jira ticket for a bug you discovered, make sure that there is no existing Jira for such bug. When testing subsequent ERs/ RCs, there will be testing cases from previous testing cycles linked to the testing Jiras for current release cut. If there is no bug created for the issue you found, create one.

The bug must be created in the [INTLY](https://issues.redhat.com/projects/INTLY) project.

The `Affects Version` and `Affects Build` **must** be defined correctly otherwise
the bug will not be triaged during the next release standup.

To provide all need information you should also compile these fields:

- `Description`
- `Steps to reproduce` (Link the test cases to this section, and add any other useful information)
- `Environment` (OSD/...)
- `Component`

After reporting the bug, link the reported bug as a blocker to this task,
and resolve this task with `Can't Do`.

### Update the Test Case

If the steps of the test case are obsolete or unclear, **please** help us maintaining them
by contributing to the [integreatly-test-cases](https://github.com/integr8ly/integreatly-operator/tree/master/test-cases) repo
or by reporting an issue [here](https://github.com/integr8ly/integreatly-operator/issues).

Check that the **estimate** time is set and accurate, otherwise it should be added or updated following [this guide](../README.md#how-to-estimate-a-test-case).

The original test cases file is linked at the beginning of the description of the task.
