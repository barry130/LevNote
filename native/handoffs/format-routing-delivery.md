# Schema3 .8 delivery — accepted

2026-11-09. Root accepted the exact package, single hidden startup and preserved
metadata corrections in [root acceptance](format-routing-delivery-root-acceptance.md).
Core/app/source/capture gates were accepted before this finite delivery.
Product, tests, package script and accepted evidence stay frozen. Source launch
documentation is updated under the separate docs-only activation; no further
product execution is needed.

Launch: `native/bin/levnote-native-worker-20261109.8/levnote-native.exe`.
Version `native-worker.20261109.8`; complete23-file folder is **23,457,235 bytes**.
The folder includes the host, three fixed engines/manifests, inline schema3
examples, licenses and attribution. APPDATA configuration remains separate.

| Payload | Bytes | SHA256 |
|---|---:|---|
| Host | 13,836,288 | `60D5C11A8EECF7626DD692098B92DD0CF4810E7F8F289515849982AE23954FBA` |
| Markdown HTML | 3,300,352 | `7487710E2882099F522C4F8A14E61C03A3F1086FA5C70E5A073DAA83C1D85F4C` |
| Mindmap SVG | 3,122,688 | `8E62E8B4EF23A725FFB2BCA3E29286FC7B703C965FFECB8F44C8DAF34A758C47` |
| Readonly scene | 3,136,512 | `02F9DA3367C45B774ECB504D8B573E1093305A6274C15107A12D5E748196803C` |

Evidence root:
[`format-routing-delivery-20261109-114942`](../artifacts/acceptance/format-routing-delivery-20261109-114942/).

- [aggregate02.json](../artifacts/acceptance/format-routing-delivery-20261109-114942/aggregate02.json):
  **12/12 PASS**, SHA256
  `3869CF65CD66CAF6107BF42865CA56EBFCA76E63DB741DCC0CDFB89904F450C9`.
- [validator02.json](../artifacts/acceptance/format-routing-delivery-20261109-114942/validator02.json):
  `7FC86AC0AD5EEDFB3D20291D5DF6EF550ED845F752C60EF1543B022F969D2947`.
- [aggregate-attempt02.ps1](../artifacts/acceptance/format-routing-delivery-20261109-114942/aggregate-attempt02.ps1):
  `1C3FFCBD986E456EC254A6B46356584FED488CDFC6ED38825618BB8678149A8A`.
- [total-byte-correction.json](../artifacts/acceptance/format-routing-delivery-20261109-114942/total-byte-correction.json):
  `7731F97A690B943455C3BE67C92E0F65479DE26054912881824A398B4E6503FD`.
  The aggregate's `TotalBytes:null` is a PowerShell OrderedDictionary
  `Measure-Object` metadata defect; this separate actual-length sum supplies the
  total above without changing the aggregate or repeating any product action.
- Shipped release manifest SHA256:
  `9957E49CA27A42F88182AF14758C47C8D09B6654F9F57C8F68B2288A2D45DC41`.

Execution and assertions

1. `run-package.ps1` invoked the unchanged
   `scripts/package-markdown-worker.ps1` once with fresh output/evidence and
   version `.8`. Actual command exit0, wall6.015914s. All four underlying pinned
   Go builds exit0 with empty captured stdout/stderr. See `package-command.json`,
   `package.stdout.log`, `package.stderr.log` and `package/*-build.json` with
   their exact generated `*-build.ps1` argument arrays and raw streams.
2. Validator reads four actual AMD64/PE32+/GUI headers, all three engine
   identities/version/size/digests/caps, the complete23-file set and22-entry
   release manifest. It checks13 exact asset copies, substituted README/notices,
   local authored links and both inline JSON examples against the real examples.
3. `smoke/run-smoke.ps1` reuses the accepted corrected held-handle helper in a
   fresh byte-identical complete bundle copy. One hidden host PID26236 was
   identified through `QueryFullProcessImageName` on its held handle; the owned
   window stayed hidden. WM_CLOSE exited0, no timeout/forced cleanup/live owned
   process. Sentinel PID9712 survived host close and exited through its own
   event. Smoke wall4.5523857s; wrapper wall5.1336847s/exit0. See raw
   `smoke.stdout.log`, empty `smoke.stderr.log`, `smoke/result.json` and
   `smoke/session-after-close.json`.
4. Isolated APPDATA contained the two unchanged example configs. Post-close
   session contains exactly writable `任务笔记.taskmd`→`task-notes` and
   `想法导图.thoughts`→`thoughts`, both preview, UTF-8, respectively BOMtrue/false,
   no draft. All23 candidate/copy file hashes and four fixture/config hashes
   match. This establishes startup/session/source preservation, not visible
   rendering. `logicalStructure` is the known `LayoutBoth` constant.
5. All44 scoped inputs and all23 protected `.7` files remain byte/time unchanged.
   Three accepted core/app result fingerprints are linked, not rerun. No worker
   operation was requested; no full-suite, resource, physical UI/OSIME/GPU or
   independent process-tree worker count is claimed.

Failure history is preserved. First `aggregate.ps1` wrote12/12 validator then
exited1 because four evidence paths were concatenated into one. Attempt02 fixes
only those expressions and writes separate outputs; no build/start/test repeat.
Its final stream hashes are in `aggregate02-command.json` after the stream
handles closed. [evidence-notes.md](../artifacts/acceptance/format-routing-delivery-20261109-114942/evidence-notes.md)
also records helper provenance and the corrected coordination-only layout
statement. The later null-total correction above is separate and immutable.
