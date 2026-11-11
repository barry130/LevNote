# Markdown export worker protocol handoff

Owner: `astra_marker_view` after root's explicit00:37local protocol takeover.
Status: concrete pure protocol and production worker source are implemented;
focused test/vet passed. The old owner has no ownership. Root subsequently
accepted the full production bundle integration; see
[full integration](markdown-worker-full-integration.md). Earlier focused
evidence remains unchanged and does not claim production process observations.

## Shared protocol seam

`native/core/engineipc` owns the one-frame wire, validation and decoding. It is
standard-library-only and does not import the format parser, runtime, app or
process facilities.

```go
type EngineID string
const MarkdownHTML EngineID = "levnote.markdown-export"

const ProtocolVersion uint16 = 1
const HeaderSize int = 24
const RequestMagic = "LVEXREQ\x00"
const ResponseMagic = "LVEXRES\x00"

const SourceMaxBytes = 8 << 20
const HTMLMaxBytes = 64 << 20
const RequestMetaMaxBytes = 4096
const ResponseMetaMaxBytes = 1 << 20
const WarningsMaxCount = 256
const WarningCodeMaxBytes = 64
const WarningMessageMaxBytes = 512
const ErrorMessageMaxBytes = 1024
const EngineVersionMaxBytes = 64

type RequestMeta struct {
    Engine EngineID `json:"engine"`
    Format string   `json:"format"`
}
type Warning struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Start   uint32 `json:"start"`
    End     uint32 `json:"end"`
}
type EngineError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}
type ResponseMeta struct {
    Engine        EngineID     `json:"engine"`
    EngineVersion string       `json:"engineVersion"`
    Format        string       `json:"format"`
    Status        string       `json:"status"`
    Warnings      []Warning    `json:"warnings"`
    Error         *EngineError `json:"error,omitempty"`
}

func EncodeRequestPrefix(meta RequestMeta, sourceBytes uint64) ([]byte, error)
func ReadRequest(r io.Reader) (RequestMeta, []byte, error)
func WriteResponse(w io.Writer, meta ResponseMeta, html []byte) error
func NewResponseDecoder(engine EngineID, engineVersion string, sourceBytes uint64) *ResponseDecoder
func (decoder *ResponseDecoder) Write(p []byte) (int, error)
func (decoder *ResponseDecoder) Finish() (ResponseMeta, []byte, error)
```

`StatusOK = "ok"` and `StatusError = "error"` are also exported constants. The
decoder is intended as the runtime's stdout writer. A non-nil `Write` error is
sticky and terminal; the runtime wrapper cancels immediately. `Finish` must be
called after stdout EOF and transfers the bounded result slice without a second
whole-response copy.

## Classifiable failures

The protocol exposes sentinel errors the runtime can classify:

```go
var ErrProtocol = errors.New("engine protocol failure")
var ErrStdoutLimit = errors.New("stdout output limit exceeded")
```

`Finish` returns a `*Failure` for malformed/truncated/extra/identity/status/
JSON/UTF8/limit problems. `Failure.Kind` values include `protocol`, `protocol-
limit`, and `stdout-limit`, with `Is()` supporting `ErrProtocol` and
`ErrStdoutLimit`. The exact complete shape is in the implementation and focused
tests.

## Worker seam

`native/cmd/levnote-markdown-export` is the one-shot executable. Its version is
supplied at build time as:

```powershell
-X main.engineVersion=<ascii-visible-version>
```

The worker reads exactly one request from inherited stdin, validates UTF8 before
`markdown.Parse`, refuses `Partial`, maps parser warnings to bounded wire
warnings, and writes exactly one response to stdout. It imports `os` only for
inherited stdio/exit behavior.

## Manifest compatibility

Astra's proposed manifest keys are accepted: `schema`, `engine`, `engineVersion`,
`protocol`, `format`, `goos`, `goarch`, `filename`, `size`, `sha256`. This
protocol package itself does not read manifests; the runtime owns those checks.

## Focused evidence

Actual test stdout in
`artifacts/acceptance/markdown-worker-owned-20261109-005510-attempt02` records
`core/engineipc` PASS0.669s and worker PASS0.944s. That combined command exited1
because of the separate runtime descendant-cleanup failure, corrected and
rechecked in runtime attempt03; do not label the combined command green.
All three owned packages' vet passed exit0, empty stdout/stderr, wall1.148s in
`artifacts/acceptance/markdown-worker-owned-20261109-010550-vet`.
Current source/evidence hashes and protected marker executable identity are in
`artifacts/acceptance/markdown-worker-focused-checkpoint-20261109-010644/checkpoint.json`.

Pure tests construct independent frame bytes and exercise one-byte fragmented
requests/responses, raw Unicode/U+FEFF, exact EOF, truncation/extra bytes, huge
declared lengths before allocation, malformed UTF8, terminal/sticky decode
failures and ownership transfer. Strict request schemas cover duplicates,
unknown/case/missing/null/wrong-type/nested values; response schemas cover nested
warning/error duplicate keys, ranges, status/error coupling, paired/unpaired
surrogates,1..64ASCIIversion and64/512/1024byte text/256warning count boundaries.

Worker tests compare HTML and mapped warnings for empty/plain/CRLF/Unicode/
tasks/tables/links/raw HTML/doc-block fixtures. Independent assertions verify
escaping, complete fence content at exactly8MiB with five-byte ampersand
expansion, partial-document refusal, invalid-request/source-limit/invalid-utf8
mapping and writer failure. Real UTF8-BOM and UTF16LE-BOM files pass through
`document.Open` in tests only; decoded snapshot excludes encoding BOM, while a
literal U+FEFF in buffer content remains. Production worker's `os` use remains
only Stdin/Stdout/Exit, with no file/path/environment/child/network facilities.

The renderer constructs full output before64MiB refusal; no incremental or peak
memory claim. Decoder owns/transfers one bounded payload. Successful worker
error writes exit0 with no HTML; host still requires exact input writing,
stdoutEOF and process exit0. No released worker binary or app export is yet
accepted by these focused package tests.
