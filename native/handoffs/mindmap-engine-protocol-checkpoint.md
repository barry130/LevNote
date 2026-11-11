# Mindmap engine IPC protocol checkpoint

Status: COMPLETE DESIGN REVIEW, 2026-11-09. The queued plan is implementable as
written. No product source was changed and no implementation/test/build was run.

## Exact minimal public API

New `core/enginecatalog` owns neutral engine identity. It must not import
`engineipc`, runtime, UI, files, or process packages.

```go
package enginecatalog

type EngineID string

const (
    MarkdownHTML EngineID = "levnote.markdown-export"
    MindmapSVG   EngineID = "levnote.mindmap-svg"
)

type Descriptor struct {
    Engine          EngineID
    Format          string
    Protocol        int
    RelativePath    string
    Filename        string
    GOOS, GOARCH   string
    SourceMaxBytes  int64
    PayloadMaxBytes int64
}

func Lookup(engine EngineID, format string) (Descriptor, bool)
func Match(engine EngineID, format, goos, goarch string) (Descriptor, bool)
```

`Lookup` is the portable operation identity lookup. `Match` additionally admits
the host platform. Both return copies by value; there is no public map, mutable
catalog, registration, discovery, or caller-defined descriptor. Unknown or
partial pairs fail rather than adopting another descriptor's format.

The two fixed values are:

- Markdown HTML: path `engines/markdown-export`, filename
  `levnote-markdown-export.exe`, format `html`, protocol 1, windows/amd64,
  source 8 MiB, payload 64 MiB.
- Mindmap SVG: path `engines/mindmap-svg`, filename
  `levnote-mindmap-svg.exe`, format `svg`, protocol 1, windows/amd64,
  source 16 MiB, payload 64 MiB.

No compiled version, size, or SHA256 belongs here.

## engineipc aliases and compatibility

`engineipc` aliases catalog identity so existing imports continue to compile:

```go
type EngineID = enginecatalog.EngineID
const MarkdownHTML = enginecatalog.MarkdownHTML
const MindmapSVG = enginecatalog.MindmapSVG
```

Keep these existing Markdown-only names and semantics:

- `SourceMaxBytes` remains the Markdown 8 MiB alias.
- `HTMLMaxBytes` remains the Markdown 64 MiB payload alias.
- `ReadRequest(io.Reader)` remains Markdown-only.
- `NewResponseDecoder(EngineID, string, uint64)` remains Markdown-only.
- `WriteResponse(io.Writer, ResponseMeta, []byte)` remains Markdown-only.

Add these operation-bound entry points:

```go
func ReadRequestFor(expected enginecatalog.EngineID, r io.Reader) (RequestMeta, []byte, error)
func NewResponseDecoderFor(expected enginecatalog.EngineID, version string, sourceBytes uint64) *ResponseDecoder
func WriteResponseFor(expected enginecatalog.EngineID, w io.Writer, meta ResponseMeta, payload []byte) error
```

`expected` is an `EngineID`, not a caller-supplied descriptor. Reject an
unsupported expected ID before reading any input. Resolve that ID to its one
fixed catalog descriptor internally, then require the request's exact
engine/format pair and source limit.

`EncodeRequestPrefix(RequestMeta, uint64)` keeps its current signature but
supports exactly the two catalog pairs and each operation's source limit. It
must not silently use the Markdown limit for Mindmap.

## Allocation and decode order

For `ReadRequestFor`:

1. reject unsupported expected ID before any read;
2. read the fixed 24-byte header;
3. validate magic, protocol, header size, metadata bound, and this descriptor's
   source bound before source allocation;
4. read only bounded request metadata;
5. strict-decode metadata and require the exact expected engine/format pair;
6. only then allocate and read source bytes;
7. validate UTF-8 and EOF.

The old `ReadRequest` can delegate to `ReadRequestFor(MarkdownHTML, r)` only if
its literal Markdown v1 frame behavior and errors remain unchanged.

For response decoding, bind expected ID, manifest version, format, source size,
and payload bound before payload allocation. Read header and bounded metadata,
strict-decode and verify identity/version, then allocate payload. Existing
fragmented-write, first-failure-terminal, ownership-transfer, EOF, and repeated
Finish semantics remain.

Preserve the current Markdown invalid-UTF8 diagnostic for the Markdown decoder;
Mindmap may use an SVG-specific diagnostic. Do not change Markdown v1 fixtures
merely to make messages generic.

## Shared shape, no generic framework

Use the same fixed request/response metadata keys, warning object, error object,
strict JSON/UTF8/escape/duplicate/EOF rules, and current error-code vocabulary.
Mindmap warnings without a real source position use `[0, sourceBytes]`, meaning
whole-document scope; do not invent node offsets. Add no optional keys,
negotiation, codec registry, dynamic paths, or filesystem access.

## Acceptance focus

Run only `./core/enginecatalog` and `./core/engineipc` plus scoped vet after
implementation. Meaningful checks:

- catalog identity/platform/path/limit positives and cross-pair negatives;
- copies and no exported mutable catalog state;
- Markdown v1 literal frame fixtures unchanged;
- both operations' request/response identity and bounds;
- unsupported expected ID never reads; foreign identity or oversized source
  never requests payload bytes from a sentinel reader;
- strict duplicate/unknown key, malformed UTF8 escape, truncated/extra bytes,
  fragmented response writes;
- nil/error result ownership on refusal and output cap without huge allocations.

No worker, runtime, app, process matrix, package, or full-repository gate belongs
in this phase.
