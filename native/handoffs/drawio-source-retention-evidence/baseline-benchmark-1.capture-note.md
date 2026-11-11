# Baseline hash capture label correction

The `baseline-benchmark-1.invocation.json` field named
`sourceSha256Before` was written by the first evidence-harness version after
the measured Go process exited. It did not wrap the invocation at the claimed
time. The values were captured after the baseline process and before product
edits.

This is supported by both the retained command/output and the accepted known
pre-change parse.go and edit.go identities. The baseline itself was valid and
will not be repeated. Later invocations record distinct
`sourceSha256Before` and `sourceSha256After` snapshots before process start and
after process exit.
