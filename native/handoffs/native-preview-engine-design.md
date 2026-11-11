# Native preview engine: finite design checkpoint

Status: DESIGN ONLY, 2026-11-09. No implementation, tests, scripts, builds,
launches, product changes or child delegation occurred for this checkpoint.

## Recommendation

Use Mindmap as the first native preview operation, but limit the first path to
read-only Mindmap preview. Editable Mindmap preview, its field/tag inspectors
and draft recovery continue through the current graphview implementation and
its concrete parser. `Adapter.Outline` also still calls `mindmap.Parse` when the
outline panel is open; the pilot preserves that behavior and exposes visible
nodes by validated ID. A folded readonly target remains a visible refusal.
The read-only pilot therefore does not claim that the host parser has been
removed from Mindmap generally.

### User-visible first path

Before: a read-only Mindmap preview calls `graphview.ensureSynced`, materializes
the source as a string, parses a full `mindmap.Document`, lays out a
`LayoutResult`, and retains scene nodes that point back to `*mindmap.Node`.

After: when `Host.ReadOnly` is true and a Mindmap preview client is available,
the host sends an immutable `text.Snapshot` to a fixed preview worker and paints
a returned bounded scene. The canvas still supports selection, pan and zoom.
The parser-backed inspector is source-only in this pilot; its UI discloses that
limit and keeps a reachable full-source entry. Source mode remains unchanged.
A missing or failed preview shows a truthful refusal, never a hidden in-host
parse.

Host allocations removed on this path: the full source string, `Document` node
tree and raw/extra clones, canonical serialization/hash work, and layout/
presentation allocations. The worker still allocates bounded source and model
memory, so total process-tree peak can increase temporarily; after the one-shot
worker exits, the host retains only the bounded DTO. These are design bounds,
not measured RSS claims.

## Operation identity

Add a third fixed catalog operation:

```go
levnote.mindmap-preview / mindmap-scene-v1
engines/mindmap-preview / levnote-mindmap-preview.exe
windows/amd64, protocol 1
source 16 MiB, response 32 MiB
```

The existing Mindmap SVG executable remains operation-bound to SVG. A separate
binary is recommended because the current catalog, manifest and protocol are
simplest to preserve when every binary serves one fixed operation; this is not
a claim that adding an operation necessarily changes old response bytes. The
cost is another packaged executable. The preview worker is a thin command that
reuses the pure Mindmap parser/layout/presentation code. Existing
Markdown/Mindmap export frames, manifests and binaries remain unchanged.

This is a fixed repository operation, not a plugin ABI or generic framework.
`engineruntime` continues to own admission, one process slot, verification and
lifecycle; no duplicate runner is introduced.

## Bounded scene DTO

Create a neutral `core/enginescene` package. It contains no renderer, UI,
process, file, HTML or source parser dependency.

```go
type Scene struct {
    Schema int
    Layout string
    Bounds Rect
    Nodes  []Node
    Edges  []Edge
}
type Node struct {
    ID, Label string
    X, Y, Width, Height float64
    Shape string // rounded only in v1
    Fill, Stroke, Text string // optional literal #rrggbb/#rrggbbaa
    Root, Collapsed bool
}
type Edge struct {
    FromID, ToID string
    From, To Point
    Stroke string
}
```

Warnings use the existing bounded `engineipc.Warning` response metadata, not a
second diagnostics array. All no-location warnings continue to use
`[0,len(source)]`. The worker sends the model with its version, ID and source
identity; the app returns those together with warnings.

Strict decode before the model is retained: exact schema, duplicate/unknown/
case-altered keys, trailing bytes, nulls and invalid UTF8 are refused. Limits:
100,000 nodes, 99,999 edges, 512-byte node IDs, 515-byte labels, 256 warnings,
and 32 MiB response. `DisplayLabel` may produce a 512-byte UTF8 prefix plus
its three-byte ellipsis; the wire admits all 515 and reports the existing
truncation warning, with no second truncation. IDs over 512 are visibly
refused, never clipped.

Decode uses a bounded shallow JSON reader that enforces field names, depth and
counts before allocating or retaining any element. A 32 MiB response alone does
not authorize unmarshalling an arbitrary tree and counting it afterward. Every
numeric field must be finite and within ±1e9; dimensions must be in
`[0,1e7]`. There is exactly one root; IDs must be unique and nonempty; edge
endpoints must reference existing nodes. The encoder counts against its hard
cap, then fills, and returns no partial payload on refusal.

Paint overrides are optional and normalized by the worker to consistent literal
RGBA hex. Empty means the renderer uses its current light/dark theme role;
root and selection state remain renderer-owned. Unsupported values retain the
existing truthful fallback warnings. The pilot needs the existing image
placeholder warning for canvas parity, but the read-only scene does not decode
or load image payloads.

## Source identity, scheduling and ownership

The app owns one generic optional scene service rather than a format-named
Host field per future format:

```go
type SceneResponse struct {
    Scene enginescene.Scene
    Warnings []engineipc.Warning
    Engine engineruntime.EngineInfo
    SourceVersion uint64
    Stale bool
}
Scene(ctx context.Context, id engineipc.EngineID,
    snapshot text.Snapshot, version uint64) (SceneResponse, error)
```

The app wires this to the fixed catalog ID and the existing runtime. It decodes
the response payload into a bounded `enginescene.Scene` before returning the
response. Graphview's read-only pilot imports the editor contract and DTO, not
`engineruntime` or the concrete Mindmap parser. Exact capability exposure and
package placement are bounded source checkpoints for the implementation owner.

The preview uses the same single `Runner` as export, with export priority:
an export request may cancel an active preview and then wait for its owned
join; admission never gives previews a loop that retries immediately. If the
source version changes while a preview is in flight, its result is marked stale
and discarded. After the current operation joins and admission is available,
the view schedules one pending newest snapshot; it does not keep a queue. The
previous bounded scene may remain visible with an explicit stale indicator so
pan/zoom do not reset on every edit, but only one old scene, one job and one
pending snapshot are retained.

Selection remains local to the view. A returned model keeps the selected ID when
it still exists; otherwise selection clears without changing the document. Pan
and zoom are viewport state and are preserved across model replacement; a theme
change repaints from the same scene without a new worker request. On tab close
or `Release`, the UI only increments its release epoch, cancels the request and
drops view ownership; it never blocks the UI thread for the runner's 30-second
timeout. The app-owned job path performs the final join and cannot dispatch a
callback into a released view. Worker failure, cleanup failure, timeout and
cancellation are displayed as unavailable/stale; there is no fallback parse or
automatic retry loop. A later version or explicit mode re-entry may request
again.

## Finite stages

1. **DTO and catalog/command source.** Add `core/enginescene` strict decode and
   the fixed catalog entry; add `cmd/levnote-mindmap-preview` using
   `ReadRequestFor`, `NativeLayoutOptions`, `PresentationIssues`, and a bounded
   DTO encoder. No runtime protocol redesign is needed.
2. **Read-only graphview pilot.** Add a separate read-only Mindmap view file and
   scheduling state; leave current editable graphview files intact. The app wires
   the host client and closes it with the existing runtime lifecycle.
3. **Consumer tests.** Use an independent literal scene, malformed/oversize/
   nonfinite/duplicate-ID cases, stale-version coalescing, release/cancel, failed
   provider with no fallback, selection/pan/zoom preservation, and one actual
   bundled request through the app seam.
4. **Reuse existing gates.** Editing, field/tag, outline, export and lifecycle
   suites remain the regression base; do not clone their full matrix.

## Remaining gap

This stage advances only read-only presentation. It does not provide installed
editable engines, a stable engine ABI, Markdown/draw.io preview separation, or
arbitrary engine installation. Editing still needs a concrete parser in the
host until an editable operation contract, source-mutation semantics, undo and
draft identity are designed. The catalog also remains compiled and fixed; a
future independently installed engine still needs signed identity, versioned
contracts and migration policy. No export-only capability or fixed manifest alias
should be presented as completion of independently packaged document engines.
