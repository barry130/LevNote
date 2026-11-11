# Mindmap engine runtime/app checkpoint

Status: IMPLEMENTED AND FOCUSED GATE READY, 2026-11-09. Root activated this
accepted plan; runtime/app/descriptor implementation and owned checks are now
complete. See [actual source/evidence handoff](mindmap-engine-runtime.md).
Product is frozen for root review; delivery is still separately gated. The
following is the accepted design record, with catalog/IPC owned separately.
Use the controlling IPC plan over its older checkpoint's
`Lookup(id,format)`/extra decoder sketches.

## Stable seam and identity

- Keep `Runner.Export(ctx, engineipc.EngineID, Source) (Result,error)` and Close.
- Catalog `Lookup(id)` returns a value; `Match(id,format,goos,goarch)` admits the
  pair/platform. IPC aliases catalog IDs; existing decoder/writer support both.
- Markdown: `levnote.markdown-export`, html, `engines/markdown-export`,
  `levnote-markdown-export.exe`, source8MiB/output64MiB.
- Mindmap: `levnote.mindmap-svg`, svg, `engines/mindmap-svg`,
  `levnote-mindmap-svg.exe`, source16MiB/output64MiB. Both protocol1/windows-amd64.
- Manifest version, size and SHA stay runtime facts. No caller-defined path,
  descriptor, command, registration, discovery or Inspect endpoint is added.

## Runtime edits (first implementation checkpoint)

1. `core/engineruntime/run.go`: Lookup/Match before encoding, source writing or
   bundle launch; validate nonnegative Size against the chosen descriptor. Use
   its Format in the request prefix and existing operation-bound decoder.
   Private `verifyBundle(ctx,hostDir,id)` re-resolves the fixed descriptor.
2. `bundle.go`: derive directory/filename from the catalog and compare every
   fixed manifest field with that expected operation, not merely any known ID.
   Proposed private signature `parseManifest(data,id)`. Retain strict duplicate,
   unknown/null/key/UTF8/escape/EOF checks,16KiB manifest and64MiB executable cap.
   A Markdown manifest placed in the Mindmap directory must refuse even when
   its executable hash/size is coherent. Do not trust manifest paths.
3. Keep `bundle_windows.go`, process/job implementations and lock lifetime
   unchanged. `bundleOpenFailure` becomes the neutral existing-class message
   “未找到随应用打包的导出程序或清单”; this also covers ancestor-open failures.
   Unknown ID/platform => FailureUnavailable; operation Size =>
   FailureSourceLimit; wrong manifest => FailureIntegrity; existing wire,
   cancellation, engine and cleanup kinds remain unchanged.
4. `types.go`, `run.go`, `runner.go`: replace the sole `Result.HTML []byte` field
   with `Payload []byte`; clear it on all failure paths. No compatibility alias
   retaining a second reference. EngineInfo and warning ownership are unchanged.
5. Keep one active operation across both IDs,30s timeout,2s pipe wait,16KiB
   stderr, bounded stdout, hidden launch, exact source writer, Start/Wait and
   Job/descendant joins, cancellation cause order, cleanup poisoning and Close
   linearization. Hash+adjacent manifest is integrity consistency, not hostile
   administrator authentication or a process sandbox.

Mechanical rename consumers: `core/engineruntime/runner_test.go`,
`worker_resources_test.go`, `app/export_worker.go`, `export_worker_test.go`,
`export_worker_bundle_test.go`. Keep Markdown HTML payload expectations and
historical JSON `HTMLBytes`/`HTMLSHA256` schema; only Result field reads/literals
change. No resource request is rerun. Bundle test call sites gain expected ID.

## App edits (after runtime seam, separate source checkpoint)

- `app/export_worker.go`: resolve the advertised ID/Format through catalog;
  refuse mismatch visibly, never silently substitute the catalog's format.
  Check platform before starting. Capture tab/buffer/adapter/capability around
  Flush and verify the same current owners afterward before taking the snapshot;
  own successful field Flush may advance version, which is captured afterward.
- Preflight uses chosen source cap: retain “Markdown 源码超过 8 MiB 导出上限”;
  add “思维导图源码超过 16 MiB 导出上限”. Unknown/mismatched capability says
  “导出引擎与格式声明不匹配”; unsupported platform says the existing platform
  refusal. No arbitrary capability string is treated as a validated operation.
- Status uses validated format: “随包外部 HTML 导出” / “随包外部 SVG 导出”.
  Invalid pairs display unavailable. Unchecked stays unchecked until export;
  status remains keyed by ID with current versus historical verified identity.
- Keep captured capability equality in `validExportIdentity`, source version,
  modal-depth deferral, pre/post-SaveDialog Flush+recheck, cancel/close/recovery
  guards and exclusive publication. `Payload` is cleared on invalidation,
  failure, completion and shutdown. Publication remains synchronous and cannot
  be canceled once WriteNewFile starts; no new asynchronous filesystem layer.
- Add only `MindmapAdapter().ExportEngine` in `modules/graphview/adapter.go`.
  `ui/editor/contract.go`, concrete app dispatch and Drawio need no changes.
  Source mode must leave View nil. Advertised worker failure never falls back
  to View.Export; its existing pure implementation remains reference/test API.

## Meaningful tests and finite gates

Runtime: update `runner_test.go`'s private repository-child fixture to support
both fixed identities; preserve existing Markdown oracle bytes. New
`core/engineruntime/engines_test.go` covers Mindmap literal SVG success, exact
request identity, wrong response engine/format, correct per-engine cap and
unknown/oversize refusal without source writes/launch. Extend `bundle_test.go`
for fixed-path/manifest cross-pair rejection and a two-bundle fixture. Test
concurrent Markdown/Mindmap admission => Busy, then reuse after joined return.
Do not clone lifecycle tests per engine: existing lock, Close/cancel, timeout,
attach, malformed output, descendant and parent-death oracles remain the gate.

App: new `app/export_mindmap_worker_test.go` reuses controlledExporter/dialog
and lifecycle helpers. Registered adapter source/preview/split + readonly,
exact BOM-backed disk/source/encoding/history/baseline, immutable snapshot,
independent SVG payload publication, and exclusive refusal. Real native
field Flush reaches the worker with committed bytes; pending tag refuses
before worker/dialog and remains recoverable through real Host.RecoverDraft.
Cover stale source/adapter/capability, mismatched advertised pair, no fallback,
two-engine busy/next-job, missing/corrupt outcome and distinct cached status.
Existing shared modal/late-composition/close-cleanup/recovery tests are reused.

After activation, focused runtime regressions then complete engineruntime+vet
once (owned repository helper processes allowed); resource opt-ins stay SKIP.
Next focused app/descriptor cases then complete app and affected graphview+vet
once as assigned. Preserve streams, commands, exit/time and source fingerprints;
reuse pure/catalog/IPC acceptance when their inputs remain unchanged. No full
repository suite, screenshots, physical UI or resource repetition implied.

Later delivery-only ownership: new opt-in
`app/export_mindmap_worker_bundle_test.go`, alongside the existing Markdown
HostPath test. Independent SVG XML/literal expectations and pure parity,
source/BOM nonmutation and exclusive refusal; each acceptance invokes two real
workers, four total across both engines. The existing packaging script will
build both fixed workers via shared helpers after separate package activation.
Neither packaging changes nor real bundle execution belongs to preparation.

Worker implementation is a separately assigned root task; it must use
ReadRequestFor(MindmapSVG), shared NativeLayoutOptions/PresentationIssues and
bounded ExportSVG. Runtime/app do not duplicate parser/rendering or warning
mapping. Root acceptance of actual worker and runtime/app source/evidence is
required before a new version/bundle is selected.
