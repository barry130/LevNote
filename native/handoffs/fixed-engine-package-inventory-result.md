# Release inventory decoder result

2026-11-09. GLM completed the narrowed pure decoder scope in
`core/engineruntime/package_inventory.go` and
`core/engineruntime/package_inventory_test.go`. No filesystem, process, worker,
build, Runner, app, catalog, script, or package input changes are included.

## Actual API

```go
const (
	releaseInventoryLimit      = 2 << 20
	releaseInventoryEntryLimit = 256
)

type releaseFile struct {
	Path   string
	Size   int64
	SHA256 string
}

type releaseInventory struct {
	Version string
	Files   map[string]releaseFile
}

func parseReleaseInventory(data []byte) (releaseInventory, error)
```

The decoder is pure and opens no files. It accepts only the exact `.8` root
and file object keys, exact schema1/windows/amd64/v1 fields, the sole
`release-manifest.json` exclusion, bounded nonempty scope/configuration and
version, 1–256 file entries, integer nonnegative sizes, and 64-character hex
hashes normalized to lowercase. It validates every listed path, including
unrelated entries, before the store selects any file.

Review corrections are covered: Unicode `EqualFold` duplicate detection with a
literal `K.txt` / `K.txt` pair, Windows-invalid characters, control/NUL paths,
reserved devices including `CONIN$`/`CONOUT$`, `CON .txt`, COM/LPT 1–9 and
superscript aliases, ordinary positive `COM3D`/`COM10`/`LPT10` controls, lone
surrogate rejection, and a valid surrogate-pair control. Inventory paths are
JSON-encoded from decoded strings in the tests rather than relying on escaped
literal backslashes.

## Verification

Fresh evidence:
`handoffs/evidence/fixed-engine-package-inventory-20261109-051510/`.

The original `commands.txt` test line records the first, incorrectly quoted
`^TestPackageInventory$` invocation that returned `[no tests to run]`. It was
not overwritten. The actual successful command and its timestamp are recorded
in `commands-correction.md` and `commands-correction.json`; no command was
rerun to create that correction. The existing `scripts/go.ps1` wrapper preserved
the process exit code.

- `scripts/go.ps1 test -run ^TestPackageInventory ./core/engineruntime` — exit 0,
  `ok levnote.local/native/core/engineruntime 0.725s`.
- `scripts/go.ps1 vet ./core/engineruntime` — exit 0.

The first test invocation was incorrectly quoted as
`-run '^TestPackageInventory$'` and returned exit 0 with
`[no tests to run]`; it is preserved under `failed-attempts/` and is not
claimed as a passing gate. The corrected run above is the accepted result.

## Source hashes at the passing gate

```text
core/engineruntime/package_inventory.go
  45b148c4dd16af512df2ef7ff96d2d62237583bb142dbbbb0569fa937ae40f32
core/engineruntime/package_inventory_test.go
  d1ac9a896a5ef253cd7f392c3e4034d40d0b94c9fd83423c237c26f6bd38a81c
```

Current adjacent store production files were included in the baseline and are
listed in `before-hashes.json`/`after-hashes.json`; Astra owns those files and
may add store tests. This result does not claim the store or runtime stage is
accepted or integrated.
