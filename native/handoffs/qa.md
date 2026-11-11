# Independent cross-package QA — Astra, 2026-11-08 11:44 +08:00

Read/test review only for GLM/DeepSeek implementation. No owner files edited. Reproduction source is `native/scripts/qa_probe.go` (build-tagged ignore; runs only explicitly). Measurements are one local Windows/amd64 run, cumulative allocated bytes, not peak RSS. Inputs are deliberately bounded; no 8 MiB heading/newline stress was attempted. Root should route fixes to owners. No concurrent full suite was run.

## Confirmed: parser allocations grow far beyond input cap (P1, GLM)

`formats/markdown/parser.go:20` preallocates a `[]line` for every LF plus every CR. Each line is 32 bytes on amd64; CRLF is counted twice. Blocks are then appended without a count/rendering budget.

Reproduction (`native/scripts/go.ps1 run ./scripts/qa_probe.go`):

| Input | Parsed result | Elapsed | Allocated |
| --- | --- | ---: | ---: |
| 64 KiB LF only | 0 blocks, 0 warnings | 0.589 ms | 2,110,360 B |
| 256 KiB LF only | 0 blocks, 0 warnings | 2.780 ms | 8,401,880 B |
| 64 KiB `# a\n` | 16,384 blocks, 0 warnings | 13.522 ms | 29,973,400 B |
| 256 KiB `# a\n` | 65,536 blocks, 0 warnings | 48.203 ms | 126,395,256 B |

Every case preserves source bytes, but the 8 MiB input limit does not bound parse allocations reasonably. At that limit the line preallocation alone would be about 256 MiB for pure LF; repeated tiny blocks would be substantially worse (not run). The Markdown view's existing per-block placeholder only bounds shaping a huge single block and cannot protect against tens of thousands of tiny blocks. Add bounded line/block processing with explicit source guidance; avoid allocate-all-then-truncate.

## Confirmed: exhausted inline budget still scans/allocates marker tails (P1, GLM)

`formats/markdown/inline.go:27` loop never exits on `p.exhausted`. The wrapper `suffix` stops repeated suffix searches, but the backtick run scan and `strings.Repeat("`", marker)` occur before consulting it. Minimal public-API input:

```go
strings.Repeat("[", 32) + strings.Repeat("x", 65536) + strings.Repeat("`", 1<<20)
```

32 unmatched brackets exhaust the 1 MiB search budget near the beginning. A later `unmatched-code` warning after `scan-budget` proves the tail branch still runs. Probe results:

- Without backtick tail: 2.000 ms, 293,296 B, 33 warnings.
- With 1 MiB tail: 4.208 ms, 5,538,488 B, 34 warnings; `continued-code-scan=true`.

Current `TestRepeatedInlineScanExhaustionIsBoundedAndDisclosed` passes because it checks only `scanSpent` and warning count; the uncharged work is invisible to that counter. Terminate inline interpretation when exhausted while keeping source visible, and bound marker-run accounting before allocation.

## Confirmed: recovery budget preflight allocates/can be poisoned (P2, GLM + app integration)

`core/session/session.go:188/228/234`:

- `draftEncodedBytes` copies the entire source string into `[]byte` before counting.
- `draftEncodedLength` uses `Buffer.WriteTo`; Snapshot.WriteTo converts chunk strings into byte slices for this writer, and `jsonStringCounter.Write` converts them again for `utf8.ValidString`. It rejects before `CaptureBounded.String`, but preflight still allocates roughly twice the draft size.
- `estimateSnapshotSize` omits `Preferences` and relies on a 1 MiB reserve. Oversized metadata is not constrained earlier.

Actual results:

| Case | Outcome | Allocated |
| --- | --- | ---: |
| `Save` one 3 MiB NUL draft | rejected **before** JSON marshal; previous recovery unchanged | 3,155,528 B |
| Builder add same NUL document | encoded limit rejects before storing tab | 6,341,320 B |
| `Preferences.Theme = strings.Repeat("x", MaxBytes+1)` | estimate passes, JSON marshal occurs, then size rejection | 33,583,696 B |

The final case reports only `session exceeds recovery storage limit`, not the pre-marshal message. It is an exported API issue; the current app normally supplies a fixed theme token, so the large theme is not claimed as an ordinary UI route.

`SnapshotBuilder.AddDocument` at line 81 also updates `encodedDraftBytes` and `metadataBytes` before `CaptureBounded` validates fields. Repro: add a 1 MiB `document.New` with `Encoding="invalid"` 14 times (all rejected), correct encoding to UTF8, then add once. Reused builder now rejects on aggregate budget; a fresh builder accepts the identical document. Failed captures must not consume budget. Passing nil document currently panics in `CaptureBounded`; return an error instead.

App source read at 11:44 still uses raw-size `preflightRecoveryDrafts` and per-tab `CaptureBounded` in `app/state.go:71/110/135`; it does not use `SnapshotBuilder` yet. Thus escaped aggregate budgeting only runs at final `Save` after draft strings have already been captured. DeepSeek's resource stage is expected to replace this path; this is a timestamped integration finding, not a request to duplicate its work.

## Verified: ordinary vendor resolution + real refresh reaches patch and test

- `native/scripts/go.ps1 -GoArguments @('list','-f','{{.Dir}}','github.com/egoist/mygo','github.com/egoist/mygo/ui')` resolves both under `native/vendor/...` without `-mod=vendor`. The pinned go.mod + vendor/modules.txt trigger ordinary vendor mode.
- Current `vendor/.../mygo/app.go` contains the LevNote marker and no startup `go cleanUpdateLeftovers()`; repository search finds only the unused helper definitions/comments.
- `native/scripts/go.ps1 -GoArguments @('test','./dependency-patches','-count=1')` passes (0.454s).
- Tested exact copied `dependency-patches/refresh.ps1` **with exact copied `scripts/go.ps1`** in `native/build-cache/qa-vendor-refresh-real/native`, an isolated Git repo with a tiny MyGo-importing main, copied module files and junctions to existing Go/module/build caches. Actual `mod vendor` ran, then output showed both `git apply --check` and `git apply`, then `ok levnote.local/native/dependency-patches 0.509s`. The child wrapper's `exit` returns from the invoked script in this setup; it does **not** prematurely terminate refresh. No production vendor or module-cache source mutation was made by QA.
- An earlier isolated fake-wrapper exercise at `build-cache/qa-vendor-refresh` separately logged `mod vendor` and `test ./dependency-patches -count=1`. The real-wrapper exercise above supersedes it as evidence.

No new isolated runtime launch was necessary here: DeepSeek owns and reports the actual startup regression. This QA independently confirms source selection and refresh execution, not a second end-user launch result.

## Focused commands

```powershell
native/scripts/go.ps1 run ./scripts/qa_probe.go
native/scripts/go.ps1 -GoArguments @('test','./core/session','./formats/markdown','-run','SnapshotBuilder|SaveRejectsJSON|RepeatedInlineScan','-count=1')
native/scripts/go.ps1 -GoArguments @('test','./dependency-patches','-count=1')
native/scripts/go.ps1 -GoArguments @('list','-f','{{.Dir}}','github.com/egoist/mygo','github.com/egoist/mygo/ui')
```

Existing focused parser/session tests pass (session 0.732s, Markdown parser 0.606s), demonstrating why the measured failures need stronger assertions. Full suite/build remains DeepSeek's responsibility.
