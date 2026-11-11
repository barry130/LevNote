# `.21` drawio efficiency delivery harness

Status: **PREPARED ONLY; EXECUTION INACTIVE**

This directory is the immutable preparation source for
`native-worker.20261110.21`. Preparation is limited to static parsing and one
tiny no-product JSON-shape sanity check. The `.21` candidate is absent, no
product freeze inventory was taken, and no build, tester compile, worker
request, host launch, suite, capture, or benchmark was run.

The execution harness combines the accepted `.20` package and custom-suffix
host route gates with the accepted `.19` managed installed-drawio lifecycle
runner. At activation it copies these scripts into a fresh evidence directory.
It runs, in order:

1. `baseline.ps1`
2. `run-package.ps1`
3. `run-managed-smoke.ps1`
4. `run-host-smoke.ps1`
5. `aggregate.ps1`

Activation must provide:

- the absent `.21` candidate directory;
- a fresh evidence directory;
- the frozen warning and retention result/review/acceptance gate files;
- the accepted source-retention benchmark fixture files;
- `-ExecuteDelivery`.

The initializer records all supplied gate/fixture identities, protects the
accepted `.20` package and complete 139-file delivery evidence, and captures
the accepted managed-smoke test source identity after source freeze. The
managed gate uses a complete byte-identical candidate copy, one tester compile,
one actual installed-worker request, exact second page/516-byte source/
598-byte SVG, zero warnings, and joined no-fallback process lifecycle. The
separate hidden host gate uses the `.20` custom-suffix fixture and configuration
setup with no worker request.

The raw managed record's uppercase `Warnings` member must be present and either
JSON `null` or an array; scalar, object, lowercase `warnings`, and absent forms
are rejected. This follows the actual Go struct with no JSON field tag and the
immutable `.19` managed record. For the zero-warning oracle it normalizes to a
retained empty array. The normalized managed result writes that array under
uppercase `Warnings`, plus `WarningPropertyKind`, `WarningCount`, `WarningsJson`,
and `WarningsSHA256`. The aggregate reads uppercase `Warnings` from both exact
JSON records, requires the normalized member to be an empty array, validates the
companion fields, and hashes the exact `[]` JSON bytes. Neither path hashes an
empty pipeline or confuses JSON `null` with an empty array.

The delivery excludes source suites, vet, captures, benchmarks, installer or
cross-platform builds, physical UI/input/IME checks, interactive SVG export,
dirty recovery, and process resource claims. Source microbenchmark evidence is
inherited only from the accepted warning/retention source slices.
