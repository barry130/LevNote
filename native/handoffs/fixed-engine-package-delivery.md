# Engine package delivery — accepted `.10`

2026-11-09. Root accepted this delivery as the current runnable native package
in [fixed-engine-package-delivery-root-acceptance.md](fixed-engine-package-delivery-root-acceptance.md).
`.8` remains a historical accepted release and `.9` remains failed history.

## Accepted current identity

- Accepted current directory:
  `native/bin/levnote-native-worker-20261109.10`
- Version: `native-worker.20261109.10`
- Package inventory: 23 files, 23,622,214 bytes
- Host: 13,998,592 bytes
- Host SHA256:
  `4EB096FAEA7CA79E71F0C9FF30E659478C0E2AA135803DDBF7C0AEE47A0AEF6C`
- Release manifest: 4,137 bytes
- Release manifest SHA256:
  `38D79797ABA4EDFB0DD12E3ED78476D8CC956D1A3069921E447690E842A0670D`

The package contains exactly four Windows amd64 GUI PE executables and 13
portable source assets. The release manifest has 22 entries plus its documented
self-exclusion. Its version, sizes and hashes match the accepted inventory.
The complete evidence copy is byte-identical to the accepted `.10` directory.

## Delivery sequence

One package wrapper invocation built exactly four GUI binaries:

| Build | Exit | Wall seconds | Stdout/stderr bytes |
|---|---:|---:|---:|
| Markdown export worker | 0 | 1.51 | 0 / 0 |
| Mindmap SVG worker | 0 | 1.45 | 0 / 0 |
| Mindmap readonly preview worker | 0 | 1.58 | 0 / 0 |
| Native host | 0 | 4.32 | 0 / 0 |

The runtime tester was compiled once beside an exact copied package and run
once under the normal default `TEMP` on `C:`. Runtime arguments were exactly:

```text
-test.run ^TestManagedRealOldReleaseThenBundledResetSmoke$
-test.v
-test.count=1
```

The test emitted `PASS`, exited 0, and made exactly two production worker
requests:

1. import the immutable `.8` Markdown engine, activate and export;
2. explicit Reset to bundled `.10`, then export through the selected bundled
   engine.

The imported `.8` Markdown package digest is
`fb291276e904a570219cf8d8ce28882b44811f4f4af89bf1cc423cabf929f155`.
It identifies that imported package, not the entire `.10` release.

Both exports produced 104-byte HTML with SHA256
`681487B9A59F3D162BC719ED242B2242AB4A5B16CCD61C5A945D15EEAEDC50A0`.
Both engine closes were called without fallback close or close errors.

One hidden start used the exact copied `.10` bundle. The executable and bundle
were on `F:`; isolated run-owned `APPDATA` and `LOCALAPPDATA` were on the normal
`C:` temp volume. Its exact owned windows became invisible, received
`WM_CLOSE`, and the process exited normally with code 0. No forced cleanup or
process-name scan/kill occurred. The sole opened Markdown source remained byte
and timestamp identical and closed without a draft. The retained session is:

- `C:\Users\canace\AppData\Local\Temp\levnote-native-host-smoke-9244f2d0a2364e25bc667eccd315ada2\appdata\Roaming\LevNoteNative\session.json`
- 427 bytes
- SHA256 `6DF53A3DCB5EC4C6C78AC4C8414656F8502C105900795EF97C835840EFADBDFF`

No user config was read or rewritten. The run-owned config directory was
preserved and no recursive cleanup was performed.

## Accepted source delta and preservation

The only accepted runtime source delta from the 117-file accepted UI capture
inventory is:

- `native/core/engineruntime/package_files.go`
  - prior SHA256:
    `BCA39FF8FF78DAD77E5AEC06137B25B18F1FAEE1544D0B8D332CE08BD25A4CC4`
  - fixed SHA256:
    `BC7A4254A62F4A2E21610EC35EA64CB73951FCB6367EE12A468DA24A3447CA05`
  - 6,632 bytes
- `native/core/engineruntime/package_guard_windows_test.go` (new)
  - SHA256:
    `3B7A12F49B2B06BBA2FC9A2F39D1645FB95BF2DF0E1B372DFCA668D9BDD9EBF7`
  - 3,709 bytes

The correction evidence is
`native/artifacts/acceptance/engine-package-volume-root-fix-20261109-190635`;
the reviewed summary is
`native/handoffs/engine-package-volume-root-fix-result.md`, and root acceptance
is recorded in
`native/handoffs/engine-package-volume-root-fix-root-acceptance.md`.

Focused fix tests passed, the affected `TestPackage*`/`TestManaged*` set passed
in 6.787s, and scoped `vet ./core/engineruntime` exited 0 with the wrapper's
recorded wall duration of 0.591s. Hardlink protection passed. The symlink-file
and symlink-dir fixtures were skipped because this Windows account lacks symlink
privilege; those two protections are not claimed as exercised.

The delivery baseline covered 284 current build/source inputs totaling 2,833,081
bytes. Their file hashes, sizes and timestamps were unchanged from baseline
through aggregate. The accepted app result and four existing 760x480 captures
were retained unchanged; the captures were reused rather than regenerated.

Both protected releases stayed exactly unchanged:

- accepted `.8`: 23 files, 23,457,235 bytes; host SHA256
  `60D5C11A8EECF7626DD692098B92DD0CF4810E7F8F289515849982AE23954FBA`
- failed `.9`: 23 files, 23,621,697 bytes; host SHA256
  `5357708C405DAE09B57D8C859B9F343BDE875E802694D3F539D6BC18F0488956`;
  release manifest SHA256
  `94868A3AF53960D493A27F0EEBD10F0FD0BB4FF1D626D9A067105C9239947E2E`

## Evidence and harness notes

The complete evidence root is
`native/artifacts/acceptance/fixed-engine-package-delivery-20261109-20261109-191019`.
The compact aggregate passed 14/14 checks; see `validator.json`,
`aggregate.json`, `stage-aggregate-invocation.json` and the package, managed and
host records under that root.

Two metadata-only harness corrections are preserved rather than hidden:

- The first managed wrapper status treated JSON `Failures: null` as one element
  through `@($null).Count`. The actual runtime test and two worker requests had
  already passed. The false status and raw streams are retained in
  `managed-real-attempt01-null-failures/`; the existing record was reprocessed
  without rerunning any worker, tester or build.
- The first aggregate attempt collided with PowerShell's read-only `$Host`
  variable. Its invocation and error are retained under
  `aggregate-attempt01-host-variable/`; the corrected aggregate only reread
  existing evidence.

The requested `C:\Program Files\PowerShell\7\pwsh.exe` is absent on this host.
Captured stages used the installed absolute Windows PowerShell 7.6.9 runtime at
`C:\Users\canace\.cache\codex-runtimes\codex-primary-runtime\dependencies\native\powershell\pwsh.exe`
with `-NoProfile`; no WSL shell was used.

Scope is the exact local package and recorded processes above. The reused
MyGo headless tester captures do not prove physical OS UI behavior. No physical
UI automation, IME, GPU, memory, CPU or resource-efficiency claim is made.
