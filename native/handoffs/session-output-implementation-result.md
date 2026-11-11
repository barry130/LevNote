# Session JSON output implementation result

Status: review corrections implemented and scoped verification complete.
Production source and tests are frozen pending independent reconciliation. No
build, launch, package change, UI/resource test, or product docs were performed.
The accepted `.15` package and evidence were not modified.

## Implemented boundary

- `core/document/state.go` adds `WritePrivateStateTo(path, write)` as a thin
  callback seam over the existing atomic lifecycle. A nil callback is rejected
  before directory or temp-file creation. `WritePrivateState(path, data)`
  delegates to the seam and retains its short-write check.
- `core/session/session.go` sets `Version`, completes `preflightSnapshot`, and
  retains the exact expected output length before any output or filesystem work.
  `Save` passes that validated snapshot and length to the callback writer.
- `core/session/output.go` uses the standard `json.Encoder` through a
  one-byte-pending adapter. It supports arbitrary chunks and empty writes,
  removes only Encoder's mandatory final LF, detects short writes explicitly,
  and prevents excess, truncated, wrong-length, or missing-LF output from being
  published.
- The adapter now retains the first write/finalization error. Every later
  `Write` and `Finish` returns that same error, and a sink failure does not
  cause another pending-byte emission. This is bounded helper-contract
  hardening. The bundled Encoder already returns its first write error, so no
  current production data-loss defect is claimed or implied.

Private permissions (`0600`), parent mode (`0700`), create/replace selection,
temp sync/close, cleanup, publication commit point, and post-publication
directory-sync behavior remain unchanged. Validation, snapshot fields, JSON
tags, field order, escaping, `omitempty`/nil behavior, draft limits, recovery
format, draft ownership, and document-save behavior remain unchanged.

This is not token streaming or fixed-memory JSON encoding. Go 1.27
`json.Encoder.Encode` still builds the complete JSON value in its internal
encoder buffer. This slice removes `json.Marshal`'s additional full returned
copy. Dirty `Buffer.String()` capture remains outside this slice.

## Review corrections

- Real atomic-pipeline failures now call:
  `writeSnapshotJSON -> prefixFailureWriter -> actual temp io.Writer supplied by
  WritePrivateStateTo`. The wrapper forwards a nonzero seven-byte prefix into
  the real temp file, then returns either a short count with no error or an
  injected sentinel. Tests require exact `io.ErrShortWrite`/sentinel
  propagation, preservation of the old recovery bytes and timestamp, and no
  leaked temp file. The earlier callback-only cleanup tests remain honest but
  are no longer used as pipeline evidence.
- The benchmark now encodes each actual fixture through the new helper into a
  buffer before timing and requires byte equality with the Marshal oracle,
  exact length, and no terminal LF. Setup and oracle checks remain outside the
  fixed `-benchtime=10x` loops.
- The Marshal byte oracle now has a real saved-only tab (`Draft == nil` with a
  valid absolute path) and an explicit true `LineNumbers` case. Existing false,
  float32, Unicode, control, and bound coverage remains.
- Sticky behavior is covered for excess output, sentinel destination failure,
  and short destination failure. The tests require the original error from both
  later `Write` and `Finish`, and require no additional sink call after failure.

## Failure and focused evidence

The original failure-first compile evidence remains:

```text
native/handoffs/session-output-evidence/focused-red-1.log
```

It records the expected missing seam/helper API at exit 1 before the original
implementation.

The first review-correction red is also preserved:

```text
native/handoffs/session-output-evidence/review-focused-red-1.log
```

It reproduces the real non-sticky excess defect: `Finish after excess = <nil>`.
Its two sink subcases hit the excess-length guard before reaching the intended
second sink call; that is a fixture mistake, not a product finding.

The corrected split-write red is:

```text
native/handoffs/session-output-evidence/review-focused-red-2.log
```

It preserves the excess failure and reproduces the original sentinel and
short-write errors being replaced by the finalization mismatch before sticky
state was implemented.

The focused post-fix command:

```powershell
.\native\scripts\go.ps1 -GoArguments @("test","./core/session","-run","^(TestSessionOutputMatchesJSONMarshal|TestSessionOutputAdapter|TestSessionOutputRepresentativeBounds|TestSaveRejectsInvalidAndOverLimit|TestSaveOutputFailures|TestSessionOutputSaveLoadRoundTrip)", "-count=1")
```

Result: exit 0; raw output is in `review-focused-green-1.log`.

The focused set covers byte-for-byte equality with `json.Marshal` for nil versus
empty tabs, omitted/present fields, nil/false/true line numbers, a saved-only
tab, controls/quotes/backslashes/slash/HTML escapes, CRLF/CJK/emoji/
U+2028/U+2029, a dirty tab, selections, BOM/encodings, float32 Scroll extremes
and `-0` omission; multiple and empty adapter writes; sticky excess/sentinel/
short-write behavior; wrong lengths, excess, truncation, missing/wrong final
bytes; focused 4 MiB draft, 100-tab, and near-16 MiB valid outputs; invalid and
just-over rejection before directory creation; real-pipeline failures through
the atomic temp writer; old recovery bytes and timestamp preservation; temp
removal; and Save/Load.

## Corrected benchmark

The benchmark was rerun once because its setup changed:

```powershell
.\native\scripts\go.ps1 -GoArguments @("test","./core/session","-run","^$","-bench","^BenchmarkSessionOutput$","-benchmem","-benchtime=10x","-count=1","-v")
```

Exit 0. Fixtures, preflight, Marshal oracle, new helper output, byte comparison,
and hashes were established outside timing. Both timed paths used actual
`io.Discard.Write`; no forced GC or fake fixed-count loop was used.

| Fixture | Encoded bytes | Marshal + discard | Encoder/adapter + discard |
|---|---:|---:|---:|
| one 1 MiB dirty draft, SHA256 `8c8daeb37bf681a3bdb852a99e41c988e839d60d96f774c5e8db5ff48a883a88` | 1,048,701 | 1,164,120 B/op, 5 allocs/op | 106,213 B/op, 4 allocs/op |
| eight tabs totaling about 1 MiB, SHA256 `fe7ac1833c52576d91a2a516c4b024b544a0e5ecff26fe65ef55f2fa14dbb7a7` | 1,049,114 | 1,469,992 B/op, 5 allocs/op | 412,825 B/op, 4 allocs/op |

The fixture hash is the SHA-256 of the deterministic `json.Marshal` oracle. The
new helper bytes were compared with that oracle before the timer. The exact
path's B/op varied with standard-library encoder pool state, so the observations
are allocation samples, not guarantees or application-level claims. Raw ns/op
values are retained in evidence but support no application speed claim. No
whole-Save, disk, CPU, RSS, working-set, or process-memory conclusion follows.

The prior `session-output-benchmark-1.log` remains historical. It lacked the
new-output byte oracle and is not used for the corrected source.

## Final affected checks

The single final command, output, and exit are recorded in
`review-final-affected-suite-1.log`:

```powershell
.\native\scripts\go.ps1 -GoArguments @("test","./core/document","./core/session","./app","-count=1")
```

Exit 0:

```text
ok  	levnote.local/native/core/document	0.894s
ok  	levnote.local/native/core/session	1.646s
ok  	levnote.local/native/app	17.571s
```

The single final vet command, empty output, and exit are recorded in
`review-final-affected-vet-1.log`:

```powershell
.\native\scripts\go.ps1 -GoArguments @("vet","./core/document","./core/session","./app")
```

Exit 0 with no stdout/stderr.

## Frozen files

Production:

```text
979DFD1E0945EBC57341712C2CE9CF8D5A55B279447382ACCD1F03748C9FB368  native/core/document/state.go
BA3580DBDA6B92984EA9E1844B211FC5FF7ED690CF0E19D30F799620E5402132  native/core/session/session.go
AA06E8E3C5C85BAE8940FABA92673582DF41B8E9F7FBD64AAD1B47DED45407EA  native/core/session/output.go
```

Tests and benchmark:

```text
940C51F8ABD1CF731C196276C6A77B4F5EBADE570DE44DD42A56E25AAAB35E24  native/core/document/state_test.go
43739380C6C5E9208FE18D94A9194DA2D812465D678702A2371A0EA33D7DDFA7  native/core/session/output_test.go
D3C6DD70CFF82D20F129BC3A9BF273DEA5E111F6CEF1EAF5125B27AD7850B7B0  native/core/session/output_benchmark_test.go
```

New review-correction evidence:

```text
F970B73C3D09D505A5692515E43F00652E8C8AAB4F52823B19EF503EE7AE5E23  native/handoffs/session-output-evidence/review-focused-red-1.log
964CAE1AAC6B7AFC07181CB8C0337938742EBD2E1EA5F9987FBE3918F34C4BFF  native/handoffs/session-output-evidence/review-focused-red-2.log
A35C416D36A39D6C5A32CFB8CC7F9955A5DEC2B1F93C061AD1E2EE61187D79AD  native/handoffs/session-output-evidence/review-focused-green-1.log
4457C3757892F8BCD9B3AB5B8122B028B49008519DE1296E2C1ECC72B49FE8D3  native/handoffs/session-output-evidence/review-session-output-benchmark-1.log
B2268E1D62703561D65604CD2F519F035AD660C944A3F291E42321893269F524  native/handoffs/session-output-evidence/review-final-affected-suite-1.log
EAFAF8D4E06C5830A9BD826C69B9E66867C2C8A4AE7D257361E9738061B88CBC  native/handoffs/session-output-evidence/review-final-affected-vet-1.log
```

Earlier evidence remains immutable:

```text
E1A991905553A6234ABA5EEA28D5AAEA87686414F1EE2486509E032AE2E2359D  native/handoffs/session-output-evidence/focused-compile-1.log
A8C084A18B371202378168E557D87215AEEB75D0A3F497F1BE32859661BC938C  native/handoffs/session-output-evidence/focused-red-1.log
746B9FDDCC882E8E0B187696FF34B00C144F6613EFA1737D153D5F3441BC64CB  native/handoffs/session-output-evidence/focused-green-1.log
061AECE1A94B4C53F25364829AC5E1A598D64F8FDDC01288F6291EEEEEC41F8F  native/handoffs/session-output-evidence/focused-suite-1.log
44E5611FC90E4D1A452438652085F62AABB197D271C55F5AB0A62D708F3E7A10  native/handoffs/session-output-evidence/session-output-benchmark-1.log
14E07D84F9743289CDB5362D9615604B3CB1D2EA50A29E1214D9ECDDFA0F60B6  native/handoffs/session-output-evidence/final-affected-suite-1.log
B0FBBE633BC213628C408A5C28E66C9AD4B4BB9AAB16A409965B5FCEFB2E1CB4  native/handoffs/session-output-evidence/final-affected-vet-1.log
```

## Remaining limits

- `json.Encoder` still owns the complete encoded JSON buffer up to the 16 MiB
  session cap; the extra `json.Marshal` returned copy is the only removed full
  output allocation.
- The benchmark measures output encoding to `io.Discard`, not temp-file writes,
  sync, replacement, or whole application Save.
- The dirty draft string retained by the public snapshot model and its capture
  path remain unchanged.
- Independent reconciliation and root acceptance are still required before a
  future package may include this slice.
