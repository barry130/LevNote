# Draw.io fourth engine: bounded worker plan

## Decision

Add one fixed export engine, `levnote.drawio-svg`, for Draw.io SVG. Keep frame
protocol version 1 and manifest schema 1. Add a required, Draw.io-only request
page object; leave every old engine's request bytes exactly as they are today.
The host still captures the source snapshot and owns process lifetime, dialog,
and publication. The worker is one-shot, local, and has no filesystem or
network access.

The active bounded SVG emitter remains owned by its current task. This plan
does not modify `formats/drawio/svg.go`; the worker only calls its accepted
public API.

## Wire and runtime contracts

In `core/engineipc`, add:

```go
type RequestPage struct {
    Index int `json:"index"`
}

type RequestMeta struct {
    Engine EngineID      `json:"engine"`
    Format string        `json:"format"`
    Page   *RequestPage  `json:"page,omitempty"`
}
```

Strict decoding stays per-engine:

- Markdown, Mindmap SVG, and Mindmap Preview accept only `engine` and `format`.
  A `page` key is unknown metadata and is rejected before source is read.
- Drawio SVG requires exactly `engine`, `format`, and `page`.
- `page` must be an object containing exactly one key, `index`.
- `index` must be a nonnegative JSON integer. `page: {"index":0}` is valid and
  distinct from an absent `page`.
- No page ID, name, path, executable, or arbitrary option is accepted. The
  current view owns an index; the buffer version guards source reordering.
- `EncodeRequestPrefix` enforces the same rules, so malformed options cannot
  reach a worker.

Keep `ProtocolVersion = 1`. The frame header and old request JSON are unchanged;
the optional field is admitted only for the new engine. Add response error code
`invalid-page` for a well-formed but nonexistent page index.

In `core/engineruntime`, add a typed request carrier without changing `Source`:

```go
type ExportRequest struct {
    Source Source
    Page   *engineipc.RequestPage
}
```

`Runner.Export` accepts this request. Preview and old exports pass `Page:nil`.
The engine lane stores the whole immutable request, not a live view. The worker
receives the exact untouched source snapshot after host-side flush, identity,
group, and byte-limit checks.

## Catalog, worker, and packages

In `core/enginecatalog`, add:

- `DrawioSVG EngineID = "levnote.drawio-svg"`
- `DrawioSourceMaxBytes = 32 << 20`
- `DrawioPayloadMaxBytes = 64 << 20`
- Descriptor: format `svg`, relative path `engines/drawio-svg`, filename
  `levnote-drawio-svg.exe`, Windows amd64, protocol 1.

Add `cmd/levnote-drawio-svg`. It:

1. Reads one frame with `ReadRequestFor(DrawioSVG, ...)`.
2. Parses the complete source with the project Draw.io parser and its existing
   bounded options; it never rewrites or serializes the source.
3. Exports exactly `meta.Page.Index` with the bounded SVG emitter.
4. Maps `drawio.ErrPageNotFound` to `invalid-page`, parser limits to
   `source-limit`, SVG output overflow to `output-limit`, and invalid XML to
   `render-failed`.
5. Returns only warnings whose `Page` equals the requested page, plus emitter
   warnings. Each bounded message retains actual page and cell meaning, for
   example `page 1 cell <id>: ...`; it does not invent precise source offsets.
   `Start` and `End` span the whole source, as the Mindmap worker does.
6. Refuses with `warning-limit` if more than 256 relevant warnings would be
   returned, rather than silently dropping them.

Compressed and read-only pages export through the existing format behavior; no
edit permission is required or requested.

Package selection state needs a real migration. Current schema 1 stores exactly
three selections in fixed arrays. Move to schema 2 with four entries:

- Reading schema 1 requires exactly the three old known engines, rejects
  duplicates and unknown IDs as today, and initializes Drawio SVG as bundled.
- Reading schema 2 requires exactly the four catalog engines.
- Writing always emits schema 2 and four entries.
- Existing installed selections for the old three engines remain active.
- Corrupt state remains blocked; explicit repair resets all four to bundled.
- Per-engine retention remains 12 versions; import, activate, list, queue,
  cancel, and close semantics are unchanged.

Update `packageEngines`, status arrays, parser, persistence, package UI labels,
and the package script's fixed engine list. Replace literal “three engine”
copy with “four”. The script adds the Drawio build definition and updates its
scope text; release inventory and package verification otherwise remain
engine-independent.

## View identity and app integration

Add a neutral editor contract in `ui/editor`:

```go
type ExportPage struct{ Index int }
type ExportToken uint64
type ViewExportRequest struct {
    Page  *ExportPage
    Token ExportToken
}
type ViewExportRequester interface {
    CurrentExportRequest() (ViewExportRequest, error)
    ExportRequestToken() ExportToken
}
```

`ExportEngine` gains `RequiresView bool`. Only the Drawio adapter sets it true.
The shell never reads `pageIndex`, `diagram`, page IDs, or graphview internals.

In `modules/graphview`:

- Add one monotonic view-owned export epoch.
- Centralize page-index changes in a helper that increments the epoch only when
  the index changes. Cover page navigation, draft restoration, reparse
  selection, and `Release`.
- `CurrentExportRequest()` validates that a page is currently readable and
  returns a copied page index plus the token. It does not flush or mutate.
- `ExportRequestToken()` returns the token for cheap revalidation.
- Page A→B→A produces a different token, so a stale invocation is refused even
  though the source version never changed.

In `app/export_worker.go`:

- Capture view pointer, page descriptor, and token only after `flushTab`, all
  existing identity checks, `buffer.InGroup()`, and the source-size check.
- Then take `buffer.Snapshot()` and enqueue the immutable runtime request.
- Extend `validExportIdentity` to require the same view and token.
- `recheckExport` revalidates before result publication and again after the
  native save dialog; nested modal delivery continues through the existing
  deferral.
- A page switch, view release, reparse, source-mode transition, or new edit
  refuses publication. It need not add a new global cancellation route.

In source mode, Drawio export is visibly disabled because there is no selected
page. The button shows a select-page instruction and a tooltip; it never silently
exports page 0 and never starts a worker.

## Ownership

- `core/enginecatalog/catalog.go` and focused test: fourth descriptor and limits.
- `core/engineipc/{protocol,decode}.go` and tests: typed page, strict decode,
  old-engine no-change, `invalid-page`.
- `core/engineruntime/{types,run,runner,engine_lane-related callers}.go`: typed
  export request and unchanged one-shot cleanup.
- `core/engineruntime/package_{types,state,store}.go` and tests: schema-2
  migration and four-engine manager.
- `cmd/levnote-drawio-svg/main.go` and test: exact-page worker.
- `ui/editor/{contract}.go`: neutral view export request/token.
- `modules/graphview/{adapter,draft,sync}.go`: epoch, selector, required-view
  capability.
- `app/{view,export_worker,engine_packages*}.go`: button policy, job identity,
  four-engine UI.
- `scripts/package-markdown-worker.ps1`: fourth fixed build definition and scope.

Do not edit `formats/drawio/svg.go` or its budget tests in this task.

## Stages

### Stage 1 — wire, worker, and manager

Implement catalog, strict page wire, runtime request carrier, Drawio worker,
schema-2 package migration, and package script definition. Do not advertise the
engine from the Drawio adapter yet, so the app keeps its current synchronous
view export while the lower layers are introduced.

Decisive tests:

- Old engine request JSON is byte-for-byte unchanged; page is rejected.
- Drawio page 0, absent page, duplicate/unknown keys, negative index, floating
  index, and a page ID are distinguished correctly.
- Worker exports exact second-page bytes and page-specific warnings; invalid
  page returns `invalid-page`; compressed and read-only pages export.
- Source bytes, BOM, history, and dirty state remain unchanged.
- Schema 1 loads into four statuses, old selections remain installed, schema 2
  round-trips, corrupt state stays blocked, and retention remains 12.

### Stage 2 — host integration and real run

Advertise Drawio's export engine, add the neutral view token, integrate app job
validation and source-mode UI, then run the actual fourth worker through the
existing managed runner and package path.

Decisive tests:

- App captures the exact flushed source and requested page after all limits.
- Page A→B→A, `Release`, source-mode transition, and post-dialog changes refuse
  publication.
- Source mode shows the disabled select-page instruction and makes no worker
  call.
- Old Markdown and Mindmap requests remain unchanged.
- Manager exposes and selects four engines while old selections survive.
- One real Drawio worker run publishes exact SVG and `Runner.Close` proves
  owned process cleanup; no memory-peak or physical-input claim is made.

Reuse the existing focused suites, bundle/runner patterns, packaging script,
and one final affected full suite. Do not add a redundant matrix.

## Honest open facts

- The bounded SVG emitter is still actively owned elsewhere. Stage 1 must wait
  for its accepted handoff or consume only its already accepted API; it must not
  patch that file concurrently.
- All direct `pageIndex` assignments must be inventoried during implementation
  and routed through the epoch helper. The plan identifies the known page
  navigation, draft restoration, sync, and release paths, but implementation
  must verify no hidden assignment remains.
- The existing 30-second runner timeout stays unchanged. A real 32 MiB run may
  reveal it as too tight; that would require a separate root decision rather
  than a silent per-engine override.
