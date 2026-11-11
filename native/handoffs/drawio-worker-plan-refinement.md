# Fourth draw.io worker plan refinement

## Refined contract

### Page request

Use a required Drawio page object with both identity fields:

```go
type RequestPage struct {
    Index int    `json:"index"`
    ID     string `json:"id"`
}
```

Rules:

- Drawio requests require exactly `engine`, `format`, and `page`.
- `page` requires exactly `index` and `id`.
- `index` is a JSON integer in `0..249999`.
- `id` is a required UTF-8 string of at most 512 bytes. An empty string is
  valid only when the selected parsed page has no ID. Overlong IDs are rejected
  explicitly; they are never truncated.
- The worker resolves only `Pages[index]` and compares that page's exact ID.
  It never searches by ID and never falls back to another page.
- A valid-but-absent index or ID mismatch returns `invalid-page`.
- Null/absent fields, duplicates, unknown keys, and wrong types are invalid.
- The old three engines reject any `page` key before reading source.

Index plus the immutable snapshot already prevents source reordering. The ID is
an additional identity assertion and supports the planned invalid-ID refusal.

### Runtime ownership

Do not introduce `ExportRequest` or change the many existing runner/fake calls.
Keep the signature:

```go
Export(context.Context, engineipc.EngineID, engineruntime.Source) (Result, error)
```

Add only owned metadata to `Source`:

```go
type Source struct {
    Size int64
    Data io.WriterTo
    Page *engineipc.RequestPage
}
```

Ownership is explicit:

- On the UI thread, the app copies the selected page value into an owned
  `RequestPage` before constructing `Source`; no field of a live view crosses
  the asynchronous boundary.
- `app/preview_worker.go` owns lane construction and the lane execution
  callsite. Its stored `engineTask.source` owns the copied page value and the
  immutable buffer snapshot. `engine_lane.go` remains the internal helper.
- `engineruntime.Runner.Export` passes `src.Page` into request metadata. Old
  and preview calls set `Page:nil`.
- Lane completion clears the entire `Source`, including page ownership, as it
  already clears the snapshot today.

### Source mode and stale cache

Source mode must not be inferred from `t.View == nil`: `releasePresentation`
calls `View.Release` but can retain the pointer. Therefore:

- Drawio's `ExportEngine` advertises `RequiresView:true`.
- The app refuses the export when `t.Mode == "source"` for that capability,
  even if a retained view pointer exists and even if it still reports page 0.
- The visible button is disabled in source mode with a select-page instruction
  and tooltip. It makes no worker call and never silently exports page 0.

The original pure `CurrentExportRequest` contract is insufficient because a
successful `Flush` with no draft can leave graph cache stale. Split preparation
from validation:

```go
type ViewExportRequest struct {
    Page  ExportPage
    Token ExportToken
}
type ExportPage struct {
    Index int
    ID    string
}
type ViewExportPreparer interface {
    PrepareExportRequest() (ViewExportRequest, error)
}
type ViewExportTokenProvider interface {
    ExportRequestToken() ExportToken
}
```

Capture order in `app/export_worker.go`:

1. Capture normal tab/buffer/adapter/capability identities.
2. `flushTab`.
3. Revalidate those identities, reject `buffer.InGroup()`, and enforce the
   source-size limit.
4. Call `PrepareExportRequest()`. This may run graphview's cache
   synchronization after the size check and returns a copied page value plus
   the current token.
5. Because synchronization may report or reenter, revalidate tab, buffer,
   version, view pointer, adapter, capability, mode, and group again.
6. Capture the final buffer version and only then take `buffer.Snapshot()`.
7. Copy the page into owned runtime storage and enqueue.

`PrepareExportRequest` may synchronize; it does not publish, flush a second
time, or discard drafts. `ExportRequestToken` and final job revalidation are
side-effect-free.

### Epoch and final validation

Graphview owns one monotonic export epoch:

- Increment it on every real cache invalidation/reparse, including `Release`,
  even when the page index remains the same.
- Increment it on every actual page-index change, including A→B→A.
- Do not increment it on a cache hit or unchanged `Draw`/`Warnings`.
- A failed `Flush` must leave drafts and their recovery state intact. If a
  failed flush still invalidates cache state, it increments the epoch; it never
  uses invalidation as an excuse to discard a draft.

The export job stores the view pointer, captured token, copied page
`{index,id}`, buffer pointer, and final version. It checks that set after
initial capture, after the worker result, after final `Flush`, and after the
native save dialog. Pure token checks plus standard tab/buffer/version identity
are sufficient at those later points; they do not resynchronize or inspect
graphview internals. A stale A→B→A, released view, reparse, source transition,
or post-dialog edit refuses publication.

### Warnings and process wording

The worker source performs no filesystem or network I/O, but the OS process is
not sandboxed. State that precisely; do not claim a sandbox or memory-peak
guarantee.

Warning spans `0..len(source)` are coarse whole-source ranges, never exact
positions. Parse warnings must retain truthful scope:

- Page-specific parse issues are labeled with the actual page (and cell ID when
  available) and are never relabeled as the selected page.
- Document-wide parser issues require an explicit document-wide classification,
  not an inferred page 0. Implementation should add the smallest parser metadata
  needed to distinguish this scope without changing emitted SVG.
- Emitter warnings are selected-page warnings because export is page-scoped.
- More than 256 relevant warnings refuses with `warning-limit`; no warning is
  silently dropped.

Worker-only tests prove exact page bytes, page/ID refusal, unchanged worker
input, and warning conversion. They cannot prove host undo, dirty, BOM, or
publication state. Assign those assertions to Stage 2 app integration tests.

## Corrected ownership

- `core/enginecatalog/catalog.go`: fourth descriptor and limits.
- `core/engineipc/protocol.go` and `core/engineipc/json.go`: strict page object,
  old-engine rejection, request encoding, `invalid-page`.
- `core/engineruntime/types.go`, `run.go`, `runner.go`: additive `Source.Page`
  and unchanged `Export` signature.
- `core/engineruntime/package_types.go`, `package_state.go`,
  `package_store.go`: four statuses, schema 1→2 migration, retention 12.
- `formats/drawio/types.go` and, if needed, the smallest parser metadata for
  truthful document-wide issue scope. Do not edit `svg.go` or its budget tests.
- `cmd/levnote-drawio-svg/main.go`: exact-index/exact-ID worker.
- `scripts/package-markdown-worker.ps1`: fourth fixed build definition.
- Stage 2 only: `ui/editor/export_request.go`, graphview epoch/preparation,
  `app/preview_worker.go`/`app/export_worker.go` integration, app source-mode
  UI, and four-engine package UI.

Inventory every hardcoded three-count array, test, and fixture, but change only
behavior required by the four-engine contract. Preserve corrupt-state blocking,
cleanup poison, explicit imports, queue/cancel/close behavior, and retention.

## Refined stages

### Stage 1 — wire, worker, manager

Implement catalog, strict page wire, `Source.Page`, Drawio worker, schema-2
manager migration, and package script definition. Keep the Drawio adapter
host-side. Make only additive app changes needed for compilation; no UI or
export-flow behavior changes.

Focused tests:

- Old three request JSON bytes unchanged; any page rejected.
- Drawio required page, ID bounds/types, duplicate/unknown fields, index range,
  exact second page, absent index, mismatched ID, compressed/read-only page.
- Worker input remains untouched; worker-only scope does not claim app undo or
  dirty state.
- Schema 1 loads with old selections and bundled Drawio; schema 2 round-trips;
  corrupt state remains blocked; retention remains 12.

### Stage 2 — host integration and real run

Advertise Drawio export, add the shared neutral contract in
`ui/editor/export_request.go`, implement graphview prepare/epoch semantics,
integrate app capture/revalidation/source-mode UI, and expose four engine
packages.

Focused tests:

- App captures exact flushed source and owned page after all limits.
- A→B→A, `Release`, reparse, source-mode refusal, and post-dialog change refuse
  publication.
- Source mode is visibly disabled and makes no worker call.
- App integration proves unchanged source, BOM, history, dirty, undo, and exact
  publication state.
- Old Markdown/Mindmap requests remain unchanged.
- One real fourth-worker run and `Runner.Close` prove owned process cleanup;
  no physical-input or resource-peak claim.

Run the affected focused packages and one final affected full suite. Do not add
a redundant giant matrix.

## Remaining root decision

Stage 1 still must wait for explicit root activation after `.18` binary
acceptance. The current 30-second runner timeout remains unchanged; if a real
32 MiB run exceeds it, request a separate root decision rather than adding a
per-engine override.
