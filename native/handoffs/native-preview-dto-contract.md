# Native preview DTO contract

Status: DESIGN ONLY, 2026-11-09. This is the corrected authored contract for the
accepted read-only Mindmap preview pilot. No implementation, tests, scripts,
product edits, execution, or child delegation occurred.

## Public neutral DTO

New package `core/enginescene` owns a pure, format-neutral scene contract. It
must not import UI, process, file, HTML/WebView, renderer, or concrete source
parser packages.

```go
const SchemaVersion = 1
const MaxPayloadBytes = 32 << 20
const MaxNodes = 100_000
const MaxEdges = 99_999
const MaxIDBytes = 512
const MaxLabelBytes = 515
const MaxLayoutBytes = 64
const MaxCoordinateAbs = 1_000_000_000
const MaxDimension = 10_000_000

type Point struct{ X, Y float64 }
type Rect struct{ X, Y, Width, Height float64 }
type Node struct {
    ID, Label string
    X, Y, Width, Height float64
    Shape string
    Fill, Stroke, Text string
    Root, Collapsed bool
}
type Edge struct {
    FromID, ToID string
    From, To Point
    Stroke string
}
type Scene struct {
    Schema int
    Layout string
    Bounds Rect
    Nodes []Node
    Edges []Edge
}
func Encode(s Scene) ([]byte, error)
func Decode(data []byte) (Scene, error)
```

`enginecatalog` adds one fixed operation:

```go
levnote.mindmap-preview / mindmap-scene-v1
engines/mindmap-preview / levnote-mindmap-preview.exe
windows/amd64, protocol 1, source 16 MiB, payload 32 MiB
```

The existing Markdown and Mindmap SVG operations remain unchanged.

## Wire schema

The `engineipc` response payload is one JSON object. `engineipc` remains
schema-agnostic: it validates engine identity, version, format, status,
warnings, and payload size, then hands the bounded payload bytes to
`enginescene.Decode`.

The object contains exactly these keys:

- `schema`
- `layout`
- `nodeCount`
- `edgeCount`
- `bounds`
- `nodes`
- `edges`

Each node object contains exactly:

- `id`
- `label`
- `x`
- `y`
- `width`
- `height`
- `shape`
- `fill`
- `stroke`
- `text`
- `root`
- `collapsed`

Each edge object contains exactly:

- `from`
- `to`
- `fromPoint`
- `toPoint`
- `stroke`

`bounds`, `fromPoint`, and `toPoint` contain exactly `x`, `y`, `width`,
`height` or `x`, `y`, as applicable. No extra keys, duplicate keys,
case-altered keys, nulls, or trailing bytes are accepted. Decoder key order is
free; encoder key order is deterministic. All numeric values use standard
JSON finite spellings.

## Invariants

- `Schema == 1`.
- `layout` is one of the known Mindmap layout names and is at most 64 bytes.
- `nodeCount == len(Nodes)`.
- `edgeCount == len(Edges)`.
- Exactly one node has `Root == true`.
- IDs are unique, nonempty, at most 512 bytes, and opaque to the DTO.
- Labels are at most 515 bytes; the existing 512-byte prefix plus three-byte
  ellipsis is valid and is not truncated a second time.
- `shape` is exactly `"rounded"` in v1.
- All numbers are finite; coordinates have absolute value at most 1e9, and
  widths/heights are in `(0,1e7]`.
- The graph is a rooted tree:
  - `len(Edges) == len(Nodes)-1`.
  - The root has no inbound edge.
  - Every nonroot node has exactly one inbound edge.
  - The graph is connected and has no cycle.
- Edge endpoints reference existing node IDs.
- Paint fields are required on every node and edge. Each value is either empty
  or a single normalized `#rrggbbaa` literal. Empty means the renderer uses its
  current light/dark theme role; root and selection state stay renderer-owned.
- Unsupported source colors keep the existing truthful fallback warnings and
  do not become CSS expressions, gradients, variables, or theme names.
- The scene retains no `*mindmap.Node`, source bytes, raw JSON, or image data.

## Encoder and decoder boundaries

`Encode` validates the full scene, counts against the hard payload cap, then
fills one exact-size buffer. On any refusal it returns no partial payload.

`Decode` uses a bounded shallow token reader, not `json.Unmarshal` into an
arbitrary tree. It enforces allowed keys, counts, depth, and all invariants
before retaining elements. Count fields may appear after their arrays, so the
decoder maintains the hard element caps incrementally and refuses the first
element beyond a cap. `MaxPayloadBytes` is the single DTO truth;
`enginecatalog` references it for the operation descriptor rather than
duplicating the number.

The worker command `cmd/levnote-mindmap-preview` uses the pure Mindmap parser,
`NativeLayoutOptions`, `PresentationIssues`, and `enginescene.Encode`. It never
imports UI or runtime. The app owns process admission and decodes the response
into `enginescene.Scene`; graphview imports the DTO, not the worker, runtime,
catalog, or concrete parser.

## Scheduling contract

Preview shares the single existing `Runner` with export, with export priority.
An export may cancel an active preview and wait for its owned join; preview
admission never becomes a busy retry loop. Only one pending newest snapshot is
retained. `Release` increments the view epoch, cancels, and drops ownership
without blocking the UI thread; final joins and callback suppression belong to
the app-owned job path.

## Meaningful future checks

Independent literal scene bytes; malformed/oversize/nonfinite/duplicate-ID/
bad-endpoint/bad-tree cases; count-before-allocation refusal; export-priority
and single-pending-snapshot behavior; nonblocking Release; and one real
bundled request through the app seam. Existing editing, field/tag, outline,
export, and lifecycle suites remain the regression base.
