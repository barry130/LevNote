# Release inventory decoder — narrowed GLM implementation

2026-11-09. ACTIVE. Root has paused the broader store assignment before any Go
files landed. GLM read_native_handoffs_editable_engine_next_design_task_md now
owns ONLY `core/engineruntime/package_inventory.go` and
`core/engineruntime/package_inventory_test.go`. Implement this small pure decoder
first and write source early. Do not resume the large store/windows/state task.
Astra will own those separate files. No child delegation.

Use exactly these private shared names so store can consume them:

```go
const releaseInventoryLimit = 2 << 20
const releaseInventoryEntryLimit = 256
type releaseFile struct { Path string; Size int64; SHA256 string }
type releaseInventory struct { Version string; Files map[string]releaseFile }
func parseReleaseInventory(data []byte) (releaseInventory, error)
```

Read the accepted .8 release-manifest.json for actual required fields. Source
keys exactly: schema, version, goos, goarch, goamd64, excludes, scope,
configuration, files. Require schema1/windows/amd64/v1, bounded validVersion,
excludes exactly ["release-manifest.json"], nonempty scope/configuration strings
bounded1024UTF8bytes each (descriptive metadata only, no filesystem use).
File object exact path,size,sha256 fields; all required, no null/wrongcase/unknown
or duplicate decoded keys at any depth. Before decode reject >2MiB, invalid UTF8
and invalid escapes via existing validJSONEscapes. Max256files, nonempty list.
Size integer>=0, int64 without float precision/overflow. SHA25664hex (normalize
to lower if uppercase accepted). Inventory cannot list itself even case-alias.

Validate EVERY path even for an unrelated engine: canonical slash-relative,
nonempty, <=1024UTF8bytes, no absolute/drive/backslash/colon/ADS/dot/dotdot/empty
segments/control/NUL/trailing dot/space or Windows reserved device components;
no duplicate including case alias. Include COM/LPT device-number forms. Do not
clean an invalid path into acceptance. Use private helpers prefixed inventory
to avoid collision with pending store. No opens/hashing/execution in decoder.
No generic third-party parser or extra dependency. Reuse existing error type.

Tests with independent literal inventory fixtures, not self-generated expected
parser output: valid .8 shape with arbitrary harmless unrelated entry, missing/
wrongcase/duplicate/null/unknown root and file keys, noninteger/overflow/negative
size, UTF8/surrogate, caps, count, path traversal/aliases/ADS/device forms/self.
Can add capped ignored trailing whitespace to test byte cap without huge files.

Run only TestPackageInventory* and scoped core/engineruntime vet after source
lands; no workers, process matrices, build or real app. Preserve actual
stdout/stderr/exit and wrapper exit LASTEXITCODE in a fresh small evidence dir.
Write fixed-engine-package-inventory-result.md with actual API/files/results.
Root will review; avoid a large generic validator or waiting for another gate.
