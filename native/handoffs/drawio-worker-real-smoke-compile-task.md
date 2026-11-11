# Compile new real-worker fixture with execution disabled

ACTIVE 2026-11-10 for same DeepSeek smoke owner. Root read prepared fixture and
accepted its design for later .19 one-request delivery. Before freezing the
entire candidate, compile the new test with its enable variable absent/disabled;
this is new fixture verification, not a repeat of the passing runtime suite.

Own only new drawio_managed_bundle_test.go formatting or concrete compile repair,
new compile evidence/result. Preserve prepared result/evidence/hash as original.
Use required wrapper, run only `test ./core/engineruntime -run
^TestDrawioManagedRealBundleSmoke$ -count=1 -v`, explicitly absent enable flag.
Expected one Skip before filesystem/process operations. Record command/env
handling/raw streams/exit and restore prior environment. No enabled smoke,
worker/host launch, packaging, full runtime tests, app edits or unrelated vet.
gofmt via configured Go toolchain permitted; no raw/system Go. If compile fails,
retain attempt, repair only new fixture, recheck. Avoid expansion into new tests.

Publish `drawio-worker-real-smoke-compile-result.md` with final hash and log. Note
actual enabled run occurs DURING .19 candidate delivery after accepted stage2/
portable source and root fixture review, BEFORE root binary acceptance. Prepared
result's phrase 'After .19 acceptance' is imprecise; don't rewrite old result.
Final no-op execution must demonstrate default Skip and no worker invocation.
