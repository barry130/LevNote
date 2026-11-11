# Fixed engine catalog and Mindmap SVG export engine

Status: DESIGN ONLY, 2026-11-09. This is the corrected refinement; no source,
tests, build, launch, network, package, or child-agent action is authorized.

## Recommendation

Build a repository-owned **fixed engine catalog** plus one second packaged
capability: **Mindmap source → SVG**. Prefer this over Markdown read-only
parse/outline in this slice. It removes the last one-engine special cases from
IPC/runtime and app dispatch while reusing a format the product already owns.
It is still export-only; it does not make preview or editing modular.

User-visible result: a `.mindmap` document can export SVG from **source mode
without mounting graphview**, using a separately packaged, manifest-verified
engine. The SVG retains current graph semantics: the exact layout values,
512-byte bounded presentation labels, image placeholders, concrete literal
colors, XML escaping, default dimensions, arrow marker, and the current
whole-view warning disclosure. Missing, unsupported, corrupt, failing,
canceled, timed-out, or cleanup-failed engines are visible; there is no silent
fallback to the compiled view.

## Catalog and manifests

`core/enginecatalog` should contain only immutable build-time admission rules:

```go
type Descriptor struct {
    ID, Format, Path, Filename string
    Protocol int
    GOOS, GOARCH string
    SourceMaxBytes int64
    PayloadMaxBytes int64
}
```

No `Version`, `Size`, or `SHA256` belongs in compiled code. Those are runtime
facts from the installed manifest and are verified against the actual local
executable as the current runtime already does. This avoids rebuilding the app
for a worker byte change and keeps executable identity separate from format
capability identity.

`Catalog.Find(id, format, goos, goarch)` rejects unknown pairs even when one
field happens to match. Format capability remains
`editor.ExportEngine{ID, Format}`; adapter ID (`mindmap`) is not confused with
engine ID. The plugin registry continues to own adapter activation and cannot
declare executables.

## Extracting the pure Mindmap presentation

The current exporter is private in `modules/graphview/mindmap_svg.go`, and it
depends on graphview's private label/color/object-string helpers. The first
implementation sub-gate is therefore not the worker itself: move one
presentation model to a package without MyGo/UI imports.

Preferred owner: `formats/mindmap` (for example `presentation.go`), because it
already owns the document and layout models and can expose a pure data result.
Do not create a parallel semantic model. The minimal API:

```go
type PresentationOptions struct { LabelMaxBytes int }
type PresentationIssue struct { Code, Message string; NodeID string; Scope string }
type PresentationResult struct { SVG []byte; Issues []PresentationIssue }
func Present(document *Document, layout LayoutResult, options PresentationOptions) (PresentationResult, error)
```

`graphview` calls the same pure function for canvas metadata and export. It may
retain UI-only layout caching; it must not retain a second label/color truth.
Move the shared pieces with it:

- the bounded Mindmap label assembly in `render_cache.go`, including topic,
  icons, tags, separators, UTF-8-safe suffix, and the 512-byte maximum;
- concrete color admission/normalization in `color.go` (hex/named literal
  colors, `none`/`transparent`, the existing 64-byte bound), plus a tiny
  object-string reader;
- the exact SVG writer from `mindmap_svg.go`.

Preserve graphview's explicit layout options exactly:
`NodeHeight:40, HorizontalGap:72, VerticalGap:24, Padding:28, MinWidth:92,
MaxWidth:380, CharWidth:8, LineHeight:20`. Define these once as the shared
presentation layout constant rather than leaving one copy in `sync.go` and
another in the worker.

Warnings must remain truthful and non-inventive:

- parse/layout `Issue` has no source range, so map only existing `Severity`,
  `Code`, `Message`, and `NodeID`; set warning range to the whole document
  `[0, sourceBytes]` rather than fabricating offsets.
- Count existing unsupported colors and image placeholders, and retain the
  current long-label disclosure. Use one shared warning code for each family.
- Unknown or unsupported values stay in the source; presentation only refuses
  paint servers/variables/expressions and falls back to current defaults.

## Bounded SVG writer

Do not adapt the current unbounded `bytes.Buffer` growth. Add a pre-growth
writer with `PayloadMaxBytes` (64 MiB is consistent, not a resource claim):

1. validate UTF-8 and source size before allocating the source string;
2. parse and layout with existing limits;
3. serialize once with a two-pass counter or a writer that refuses before the
   fixed cap; never emit partial SVG on refusal;
4. return exact complete bytes and only fully constructed warnings.

The output cap does not cap worker RSS, source/model allocations, or process
lifetime; those remain bounded by existing parser/layout limits, the one-shot
request, and runtime timeout.

## Protocol, runtime, and app ownership

Keep Markdown request/response v1 bytes and behavior unchanged. Add
**operation-specific codecs**, not a generic schema. A codec owns identity,
metadata fields, response status/error codes, source/payload limits, and
warning shape.

Before allocating source bytes, reject:

- unknown engine/format pair;
- unsupported platform;
- source over the operation limit (Mindmap: existing 16 MiB default);
- request metadata over its fixed bound;
- any protocol mismatch.

For Mindmap SVG, response metadata carries engine identity/version, format,
status, warnings, and optional error. Warnings are bounded count/length and use
whole-document ranges where the format has no exact source range. Response
payload is SVG bytes.

Runtime changes should be narrow:

- catalog descriptor replaces hardcoded directory/filename/platform identity;
- manifest still owns version/size/SHA256 and existing locked-file verification;
- retain one Start/Wait per request, hidden process, Job attachment, cancellation,
  timeout, stderr limit, EOF/exit checks, cleanup failure poisoning, one-job
  serialization, Close linearization, and no publication by runtime;
- rename `Result.HTML` to operation-neutral `Result.Payload`, updating owned
  call sites in the same slice rather than leaving parallel APIs.

The app continues to flush, capture immutable snapshot/version/adapter, run one
job, reject stale results, defer modal publication, publish only after save
dialog and version checks, and refuse overwrite. Existing Markdown tests should
require only mechanical result-field updates.

## Focused meaningful gates

Run focused packages in staged gates, not a full repository/process matrix:

- catalog: positive/negative identity, platform, path derivation, limits; no
  runtime process.
- graphview pure extraction: exact SVG fixture equivalence before/after moving
  presentation; long labels/images/unsupported literal colors; existing 512-byte
  and exact-source invariants.
- Mindmap presentation: output-limit refusal without partial payload; whole
  document warning range; no fabricated offsets.
- engineipc codecs: strict keys/types/limits/EOF for both operations; Markdown
  v1 fixture unchanged.
- engineruntime: both manifests and executable identities; missing/integrity/
  platform/cancel/busy/timeout behavior with existing fake-worker tests.
- Mindmap worker: one real child process gate with exact SVG, parse warning,
  layout issue, oversized source, and output cap.
- app export job: source-mode Mindmap export using the existing controlled
  exporter and mocked save-dialog seam; one real bundled-worker integration
  invocation only after the focused source gate.

## What remains native

After this slice, native code still owns `.mindmap` canvas editing/preview,
all format file access, undo, dirty state, and save. The second engine is a
one-shot source-to-SVG program. The next missing modular boundary is a bounded
read-only parsed-model DTO and stale-identity contract for native view
consumption; that contract should be designed before claiming preview parsing is
independent.
