# Fixed engine catalog and strict operation-bound wire

Status: ACCEPTED 2026-11-09. Root read all product/tests, actual retained final
catalog0.435s/IPC0.572s PASS and vet0 combined streams in
artifacts/acceptance/mindmap-engine-ipc-20261109-063836-followup-final.
Foreign requests leave6 declared source bytes unread; response failures occur
at metadata with nonzero declared payload; exact caps/literal frame pass.
The documented one-byte gofmt change occurred before final execution, with final
test hash54F849D6...; product source remained frozen. Earlier failed test-oracle
attempts retained. No further IPC edits/checks required absent concrete defect.
GLM's next assignment is independent read-only runtime/app/worker review.

## Owned files

New `core/enginecatalog` and existing `core/engineipc` source/tests only, plus
the protocol checkpoint/result handoff and new evidence. No runtime, worker,
app, view, format, packaging, vendor or accepted artifact changes. No launches,
worker builds, resources, network or child delegation. Root reviews only.

## Static catalog

Keep it finite and value-returning. Two fixed operations: existing
`levnote.markdown-export`/html and new `levnote.mindmap-svg`/svg. Repository
relative locations are `engines/markdown-export/levnote-markdown-export.exe`
and `engines/mindmap-svg/levnote-mindmap-svg.exe`. Platform windows/amd64,
protocol1, Markdown source8MiB, Mindmap source16MiB, payload64MiB. No manifest
Version/Size/SHA, process state, installed plugin state, public mutable map or
caller-defined executable descriptor. No registration/discovery framework.

Avoid package cycles: catalog owns the neutral EngineID type/constants, ipc can
alias them while preserving existing MarkdownHTML and API uses. Catalog returns
copies. Runtime will later look up an ID and validate fixed path/platform; no
caller-provided descriptor will bypass lookup. Separate operation lookup from
host-platform admission so pure wire tests are portable. Pair mismatches must
fail, not silently adopt a descriptor's format.

Use `Lookup(id EngineID) (Descriptor,bool)` because the existing Runner.Export
already receives only ID; Match additionally checks format and platform. The
descriptor contains ID, Format, Protocol, RelativePath, Filename, GOOS/GOARCH,
SourceMaxBytes/PayloadMaxBytes. Consumers that carry a pair must compare Format
explicitly (or use Match); a successful ID lookup never licenses a wrong pair.

## Strict compatible protocol

Preserve Markdown v1 bytes, header, strict JSON key/type/UTF8/escape/EOF/error
semantics. Keep existing `ReadRequest(reader)` explicitly Markdown-only for the
unchanged Markdown executable. Add an expected-engine request reader for the
Mindmap worker. It rejects unsupported expected IDs before reading; it accepts
only that operation's exact engine/format pair and source limit before source
allocation/read. The bounded header/metadata may be read first. Never broaden
ReadRequest so the old worker accidentally renders Mindmap requests as Markdown.

EncodeRequestPrefix supports only the two known pairs and their limits.
ResponseDecoder binds known engine, version, expected format and source length;
validate metadata identity before payload allocation. WriteResponse applies
that operation's bound; success/errors never cross identities. Maintain existing
Markdown-only limit aliases for existing consumers; expose the new descriptor
limits rather than silently redefining SourceMaxBytes to16MiB.

Keep NewResponseDecoder(engine,version,sourceBytes) and WriteResponse(w,meta,payload)
signatures and extend their known identities directly: decoder already receives
expected engine, and writer validates the fixed exact pair in meta. Do not add
NewResponseDecoderFor/WriteResponseFor duplicate APIs merely to keep old names
Markdown-only; they have no legacy implicit worker-reader risk. Existing
Markdown byte/error behavior must stay identical. Only ReadRequest remains
Markdown-only and wraps the new `ReadRequestFor(expected EngineID,r io.Reader)`.

Use existing strict warning shape. Mindmap warnings without known byte positions
will be mapped by the worker to [0,sourceBytes], documented as whole-document
scope, never as precise node locations. No fabricated ranges or new optional
keys. Retain current error code vocabulary where sufficient. No generic codec
registry, negotiation, dynamic operation metadata, paths or filesystem access.

## Evidence

First publish chosen public names/signatures and compatibility facts; proceed
with implementation after activation. Meaningful tests prove static identity,
copies, platform description and fixed paths; cross-engine requests/responses,
wrong format/version, operation-specific source bounds, strict duplicate/unknown
JSON/UTF8/EOF and fragmented writes. For foreign identity/oversized source use
a sentinel reader proving payload bytes were never requested; do not allocate
16MiB just to assert preflight. Keep independent literal Markdown v1 frame
fixtures and actual success payloads for both operations. Test nil/error result
ownership on refusal and output cap without huge allocations.

Run complete `./core/enginecatalog ./core/engineipc` and scoped vet once after
focused changes, preserving actual commands/streams/exits/times and source
fingerprints. No process matrix or whole-repository check. Worker/runtime/app
integration gates follow separately. Record `native/handoffs/mindmap-engine-ipc.md`
and stop for source/log review; do not widen this phase on your own.
