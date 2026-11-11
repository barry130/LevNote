# Native preview worker checkpoint

Status: DESIGN ONLY, 2026-11-09. This is the bounded follow-on plan after
acceptance of the `core/enginescene` codec. No product source, tests, scripts,
builds, launches, network, runtime/UI activation, or child delegation occurred.

## Decision

Implement one fixed operation, not a plugin framework:

```go
MindmapPreview enginecatalog.EngineID = "levnote.mindmap-preview"

Descriptor{
    Engine: MindmapPreview,
    Format: "mindmap-scene-v1",
    Protocol: enginecatalog.ProtocolVersion,
    RelativePath: "engines/mindmap-preview",
    Filename: "levnote-mindmap-preview.exe",
    GOOS: "windows", GOARCH: "amd64",
    SourceMaxBytes: enginecatalog.MindmapSourceMaxBytes, // 16 MiB
    PayloadMaxBytes: enginescene.MaxPayloadBytes,        // 32 MiB
}
```

`enginecatalog` adds only the ID, descriptor, and `enginescene` constant
reference. It adds no descriptor mutation, host matcher, generic plugin API, or
second protocol truth. Existing Markdown and Mindmap SVG descriptors, their
bytes, and manifest comparison semantics stay unchanged.

`engineipc` adds only:

```go
MindmapPreview = enginecatalog.MindmapPreview
```

`ReadRequest` remains the Markdown-only compatibility wrapper. The preview
command uses `ReadRequestFor(MindmapPreview, input)`. Existing request/response
frame encoding, decoding, validation, warnings, and exit conventions are reused.
Runtime manifest admission continues to compare every fixed descriptor field;
no manifest fields or interpretation change.

## Producer mapping

The command-local adapter converts pure `formats/mindmap` results to
`enginescene.Scene`. It does not retain `SceneNode.Node`, source bytes, raw
JSON, metadata, images, tabs, buffers, or view identity.

- `Scene.Schema = enginescene.SchemaVersion`.
- `Scene.Layout = layout.Kind`, the effective layout. The four accepted values
  are `logicalStructureLeft`, `logicalStructureRight`, `logicalStructure`, and
  `orgChart`. An unknown source layout keeps its existing pure behavior:
  `UNKNOWN_LAYOUT` warning plus effective `logicalStructure`.
- `Scene.Bounds = layout.Bounds`.
- A node maps `ID`, geometry, `Shape` (`rounded`), `Collapsed`, and
  `Root = (node.ID == document.RootID())`.
- `Label = mindmap.DisplayLabel(sceneNode.Node, sceneNode.Text)`. This is the
  existing shared 512-byte-prefix/ellipsis truth; it is reused, not changed or
  truncated again.
- Edge endpoints and points map directly from `SceneEdge`. The wire edge is
  oriented parent-to-child; the DTO does not carry `Parent`, `Depth`, `Side`,
  `ChildCount`, or `VisibleChildren`.
- Hidden descendants under a collapsed parent are not emitted, exactly as the
  pure layout already does. The visible collapsed node keeps `Collapsed=true`.
- `IMAGE_PLACEHOLDER` remains a warning from `PresentationIssues`; no image
  object, URL, bytes, or metadata enters the scene.

Native paint semantics come from the actual graph code, not a new format
interpretation:

- Host defaults (`Surface`, `Border`, `Text`), root accent, selection accent,
  and the collapsed `+` badge remain renderer-owned. Therefore the producer
  emits empty literal paint for absent overrides.
- Mindmap node style keys are exactly `background-color` (fill), `border`
  (stroke), and `color` (text). A valid literal style value overrides the theme
  role, including on the root; unsupported or non-string values produce empty
  paint and keep the existing truthful `UNSUPPORTED_COLOR` warning.
- Edge stroke comes from the child's inherited `BranchColor`. An absent value
  is empty (host theme border); an invalid value is empty and keeps the same
  warning.

Keep `mindmap.LiteralColor` and `mindmap.DisplayLabel` as the shared admission
truth. Add a private command-local helper that converts an admitted literal to
the DTO's only normalized spelling:

- `#rgb` and `#rgba` expand each component digit.
- Six-digit results (including named colors and `rebeccapurple`) append `ff`.
- `#rrggbbaa` is unchanged.
- Failure returns the empty paint, which the host resolves as its theme role.

The helper belongs in the new command package. Do not edit accepted
`formats/mindmap` presentation helpers, put CSS/UI color knowledge into
`enginescene`, or grow the neutral DTO with native-only style keys.

## One-shot command

Add `native/cmd/levnote-mindmap-preview/main.go` and a private command-local
adapter file. The command inherits only stdin, stdout, and the process exit
path. It must not import file, network, process-launch, UI, runtime, catalog
matching, or application packages.

Follow the existing Mindmap SVG worker shape:

1. Refuse an invalid packaged `engineVersion` with exit 2.
2. `ReadRequestFor(MindmapPreview, input)` before payload work. Map request
   failures to `invalid-request`, except `source-limit` and `invalid-utf8`.
3. Parse with `mindmap.Parse(..., ParseOptions{MaxBytes: 16 MiB})`.
4. Run `mindmap.Layout(document, mindmap.NativeLayoutOptions())`.
5. Run `mindmap.PresentationIssues(document)`.
6. Convert parser, layout, and presentation warnings in that order using the
   existing bounded wire-warning conversion. Every warning uses the whole
   source range `[0,len(source)]`; no offset is invented. Warning count, code,
   message, and UTF-8 budget violations become `warning-limit` before payload
   publication.
7. Build the neutral scene and call `enginescene.Encode`.
8. On any refusal, return a status-error response with zero payload and no
   partial scene. Map `enginescene.ErrLimitExceeded` to `output-limit`; other
   encode errors to `render-failed`.

Parser and layout failures map like the existing command: `ErrLimitExceeded`
and other valid-but-too-deep parser refusals are `source-limit`; invalid
JSON/schema/root/tree data is `render-failed`. Error messages are bounded
constants and do not echo source or app identity. A failing response writer is
exit 1, matching the existing worker contract.

Keep small private seams for meaningful fixtures (for example a payload limit
and adapter normalization helper); production still enters through the fixed
public limit. Do not add a generic executable installer or runtime activation.

## Exact future test gates

No stage is activated by this design. After codec acceptance, implementation
may add only:

- `native/core/enginecatalog/catalog_test.go`: the exact preview descriptor,
  partial-identity and platform rejection, and unchanged Markdown/SVG literals.
- `native/core/engineipc/protocol_test.go`: the preview pair uses the fixed
  16 MiB source and 32 MiB payload; wrong-operation requests are refused from
  metadata before source allocation; old `ReadRequest` remains Markdown-only.
- `native/cmd/levnote-mindmap-preview/main_test.go`: independent literal scene
  values for all four layouts, including labels/metadata, admitted paints,
  theme-empty fallbacks, inherited edge color, a collapsed child, and
  `IMAGE_PLACEHOLDER`/`LABEL_TRUNCATED`/`UNSUPPORTED_COLOR` warnings; invalid
  parser/schema, output cap, valid deep-source refusal, warning excess, no
  partial payload, and response-writer failure.

Use literal expected DTO values rather than an encode/decode round trip alone.
Exercise the private seams with readers/writers that prove how many bytes were
consumed and whether allocation happened. Future evidence must retain actual
command arguments, combined stdout/stderr, exits, timings, and focused vet
streams; this design itself is not a passing test.
