# Installed fixed-engine delivery smoke — source checkpoint

2026-11-09. GLM completed the source-only task in
`core/engineruntime/managed_bundle_test.go`. No test, worker, app, build, or
process execution was performed. Existing Runner, store, runtime, app, catalog,
script, `.7`, `.8`, or package inputs were not modified.

## Opt-in and behavior

`TestManagedRealOldReleaseThenBundledResetSmoke` defaults to `SKIP` unless both
environment variables are set:

- `LEVNOTE_MANAGED_REAL_SMOKE=1`
- `LEVNOTE_MANAGED_REAL_SMOKE_RELEASE=<absolute source-release directory>`

The test takes `hostDir` from `os.Executable()`, uses a temporary `stateDir`,
and imports Markdown from the explicit source release with the real
`PackageManager`. It checks that import leaves the bundled selection, then
activates the exact receipt digest, constructs `NewManaged`, and exports one
small immutable UTF8/CRLF/Unicode Markdown source with a 40-second context.
It compares returned source, digest, version, executable SHA, exact
independently authored HTML, payload size/hash, and unchanged source bytes,
then closes the runner.

It then explicitly calls `Reset`, constructs a fresh managed runner, and makes
exactly one bundled export of the same source. It checks bundled source/no
digest, the current bundled version and executable SHA, and output parity
against the same independent HTML oracle, then closes that runner. The whole
test performs exactly two repository engine requests; it exposes no generic
executable resolver and never writes to the source release or bundle.

The test logs one structured JSON record with scope, old/new engine versions
and executable hashes, package digest, import/activation/reset selections,
source and payload sizes/hashes, both export outcomes, and joined close
results. Missing opt-in configuration fails explicitly only after opt-in; by
default the test skips before opening or launching anything.

## Stage status

This is source preparation only. Root must authorize the later default-SKIP
compile/vet gate after Astra's runtime files are frozen, and actual execution
is reserved for the later accepted candidate. This checkpoint does not claim
delivery, integration, or execution success.
