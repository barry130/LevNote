# Fixed engine catalog and strict operation-bound IPC

Status: COMPLETE, frozen for root source/log review, 2026-11-09.

## Public API and compatibility

New `core/enginecatalog` owns neutral identity:

```go
type EngineID string
const (MarkdownHTML EngineID = "levnote.markdown-export"; MindmapSVG EngineID = "levnote.mindmap-svg")
const ProtocolVersion = 1
const MarkdownSourceMaxBytes = 8<<20
const MarkdownPayloadMaxBytes = 64<<20
const MindmapSourceMaxBytes = 16<<20
const MindmapPayloadMaxBytes = 64<<20
type Descriptor struct{ Engine EngineID; Format string; Protocol int; RelativePath, Filename string; GOOS, GOARCH string; SourceMaxBytes, PayloadMaxBytes int64 }
func Lookup(id EngineID) (Descriptor, bool)
func Match(id EngineID, format, goos, goarch string) (Descriptor, bool)
```

There is no public map, registration, discovery, host helper, or caller-defined
descriptor. Values are immutable and returned by value.

`engineipc` aliases catalog identity (`type EngineID = enginecatalog.EngineID`,
`MarkdownHTML`, `MindmapSVG`) and keeps Markdown-only aliases `SourceMaxBytes`
and `HTMLMaxBytes`. `ReadRequest(reader)` remains Markdown-only and delegates to
the sole new request API:

```go
func ReadRequestFor(expected EngineID, r io.Reader) (RequestMeta, []byte, error)
```

`EncodeRequestPrefix`, `NewResponseDecoder`, and `WriteResponse` retain their
signatures and now support exactly the two catalog pairs. Request encoding uses
portable `Lookup` plus exact format; runtime owns platform admission later.

## Behavior

- Unknown expected IDs are rejected before any input read.
- Request headers and bounded metadata are read first. Exact engine/format and
  operation source bounds are validated before source allocation/read.
- Response headers and metadata are validated for expected engine, format,
  version, and operation payload/source bounds before payload allocation.
- Markdown v1 frames, key strictness, escape handling, EOF behavior, limits,
  error codes, and `HTML is not valid UTF8` diagnostic remain unchanged.
- Mindmap SVG uses the same wire shape and error vocabulary. A warning may use
  `[0, sourceBytes]` for whole-document scope. No optional keys were added.
- Cross-engine requests, cross-format responses, wrong version/identity,
  oversized payload/source, and out-of-source warnings are rejected.

## Owned files

- `native/core/enginecatalog/catalog.go`
- `native/core/enginecatalog/catalog_test.go`
- `native/core/engineipc/protocol.go`
- `native/core/engineipc/json.go`
- `native/core/engineipc/decode.go`
- `native/core/engineipc/protocol_test.go`

No runtime, worker, app, view, script, vendor, or package files were changed.

## Actual verification

Commands:

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./core/enginecatalog','./core/engineipc','-count=1','-v')
.\native\scripts\go.ps1 -GoArguments @('vet','./core/enginecatalog','./core/engineipc')
```

Both exited 0. Test stream showed catalog PASS and engineipc PASS; vet emitted
no output. Both logs captured combined stdout and stderr; an empty vet log
therefore means no output on either stream. The first run also passed after two
test-oracle corrections:

- oversized request is intentionally rejected after one header read, so the
  sentinel oracle now requires exactly one read;
- response warning source range is checked by the decoder's actual source size,
  while the writer only applies the operation source cap.

New meaningful coverage:

- exact catalog values, fixed relative paths/filenames/platforms/limits;
- partial identity and platform mismatches rejected;
- independent literal Markdown and Mindmap request frame bytes;
- Markdown reader rejects Mindmap frame and vice versa;
- unsupported expected ID performs zero reads;
- oversized Mindmap source rejects after header without payload allocation;
- fragmented/sticky response decoder behavior remains;
- both operations' identity and bounds, wrong format/version, and whole-document
  warning beyond Markdown's 8 MiB alias;
- foreign request pairs leave the declared source unread, and the response
  decoder rejects foreign identity at the metadata boundary before payload;
- exact cap and cap+1 encode behavior for both operations, and literal Mindmap
  SVG response bytes;
- strict duplicate/unknown/case/null/type/trailing/UTF8 metadata coverage
  remains from the Markdown v1 suite.

Final retained evidence is in
`native/artifacts/acceptance/mindmap-engine-ipc-20261109-063836-followup-final/`.
It contains `test.result.json`, combined `test.stdout.log`, `vet.result.json`,
zero-byte combined `vet.stdout.log`, and before/after source inventories.
The actual final test command exited 0 in 1243 ms; scoped vet exited 0 in 478
ms. The earlier failed oracle attempts are preserved in
`native/artifacts/acceptance/mindmap-engine-ipc-20261109-063214-followup/`.
Details and the exact `protocol_test.go` hash sequence are recorded in
`native/handoffs/mindmap-engine-ipc-acceptance-followup.md`.

## Source fingerprints

```text
b064e0165f5e6577b460616465be6456da2a0027ee02a378827de3cdc87d46b2 1754 native/core/enginecatalog/catalog.go
10c4497d9891348ac803264f3997ee631effc93c25e2c9731f169a98bc4621dc 1484 native/core/enginecatalog/catalog_test.go
e1dc708e0391899430feb43b4ed2755938d570f11d3b0fa713778b1dd5a2a7da 8219 native/core/engineipc/protocol.go
fe829ae271e28d6d237e6605f03663dbd740295818511ceb68d425403e5e802c 4880 native/core/engineipc/json.go
663a6b44ee185dd8b1383955db529441ba63b2f10ad66946fbb72dc04f8f6ea5 3398 native/core/engineipc/decode.go
54f849d6c71fdb453e0a2926e813cd0877d9a776a0a588943669a8ebfd1db743 20014 native/core/engineipc/protocol_test.go
```

No process, worker, UI, network, package, or full-repository execution occurred.
