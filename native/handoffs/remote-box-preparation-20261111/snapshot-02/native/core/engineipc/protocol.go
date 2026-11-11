// Package engineipc defines the bounded one-shot export wire. It does not
// import format implementations, files, processes or user interface packages.
package engineipc

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	"levnote.local/native/core/enginecatalog"
)

type EngineID = enginecatalog.EngineID

const (
	MarkdownHTML   = enginecatalog.MarkdownHTML
	MindmapSVG     = enginecatalog.MindmapSVG
	MindmapPreview = enginecatalog.MindmapPreview
	DrawioSVG      = enginecatalog.DrawioSVG
)
const (
	ProtocolVersion        uint16 = enginecatalog.ProtocolVersion
	HeaderSize                    = 24
	RequestMagic                  = "LVEXREQ\x00"
	ResponseMagic                 = "LVEXRES\x00"
	SourceMaxBytes                = enginecatalog.MarkdownSourceMaxBytes
	HTMLMaxBytes                  = enginecatalog.MarkdownPayloadMaxBytes
	RequestMetaMaxBytes           = 4096
	ResponseMetaMaxBytes          = 1 << 20
	WarningsMaxCount              = 256
	WarningCodeMaxBytes           = 64
	WarningMessageMaxBytes        = 512
	PageIndexMax                  = 249_999
	PageIDMaxBytes                = 512
	ErrorMessageMaxBytes          = 1024
	EngineVersionMaxBytes         = 64
	StatusOK                      = "ok"
	StatusError                   = "error"
)

type RequestPage struct {
	Index int    `json:"index"`
	ID    string `json:"id"`
}

type RequestMeta struct {
	Engine EngineID     `json:"engine"`
	Format string       `json:"format"`
	Page   *RequestPage `json:"page,omitempty"`
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

var ErrProtocol = errors.New("engine protocol failure")
var ErrStdoutLimit = errors.New("stdout output limit exceeded")

// Failure contains constant diagnostic text, never untrusted document bytes.
// Code is the coherent request error code used by the worker.
type Failure struct{ Kind, Code, Detail string }

func (e *Failure) Error() string { return "engine protocol: " + e.Detail }
func (e *Failure) Is(target error) bool {
	return target == ErrProtocol || (target == ErrStdoutLimit && e.Kind == "stdout-limit")
}
func invalid(detail string) *Failure {
	return &Failure{Kind: "protocol", Code: "invalid-request", Detail: detail}
}
func sourceLimit() *Failure {
	return &Failure{Kind: "protocol-limit", Code: "source-limit", Detail: "source exceeds byte limit"}
}
func stdoutLimit() *Failure {
	return &Failure{Kind: "stdout-limit", Detail: "response exceeds byte limit"}
}

func EncodeRequestPrefix(meta RequestMeta, sourceBytes uint64) ([]byte, error) {
	descriptor, ok := enginecatalog.Lookup(meta.Engine)
	if !ok || descriptor.Format != meta.Format || !validRequestPage(meta.Engine, meta.Page) {
		return nil, invalid("unsupported request identity")
	}
	if sourceBytes > uint64(descriptor.SourceMaxBytes) {
		return nil, sourceLimit()
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return nil, invalid("request metadata encoding failed")
	}
	if len(data) > RequestMetaMaxBytes {
		return nil, invalid("request metadata exceeds byte limit")
	}
	return append(makeHeader(RequestMagic, len(data), sourceBytes), data...), nil
}

func validRequestPage(engine EngineID, page *RequestPage) bool {
	if engine != DrawioSVG {
		return page == nil
	}
	return page != nil &&
		page.Index >= 0 && page.Index <= PageIndexMax &&
		utf8.ValidString(page.ID) && len(page.ID) <= PageIDMaxBytes
}

func makeHeader(magic string, metaBytes int, payloadBytes uint64) []byte {
	h := make([]byte, HeaderSize)
	copy(h, magic)
	binary.LittleEndian.PutUint16(h[8:10], ProtocolVersion)
	binary.LittleEndian.PutUint16(h[10:12], HeaderSize)
	binary.LittleEndian.PutUint32(h[12:16], uint32(metaBytes))
	binary.LittleEndian.PutUint64(h[16:24], payloadBytes)
	return h
}

func checkHeader(h []byte, response bool, descriptor enginecatalog.Descriptor) (int, int, error) {
	magic := RequestMagic
	metaMax := RequestMetaMaxBytes
	if response {
		magic = ResponseMagic
		metaMax = ResponseMetaMaxBytes
	}
	if len(h) != HeaderSize || string(h[:8]) != magic || binary.LittleEndian.Uint16(h[8:10]) != ProtocolVersion || binary.LittleEndian.Uint16(h[10:12]) != HeaderSize {
		return 0, 0, invalid("invalid frame header")
	}
	m := binary.LittleEndian.Uint32(h[12:16])
	p := binary.LittleEndian.Uint64(h[16:24])
	if m == 0 || m > uint32(metaMax) {
		if response && m > uint32(metaMax) {
			return 0, 0, stdoutLimit()
		}
		return 0, 0, invalid("metadata size is invalid")
	}
	if response {
		if p > uint64(descriptor.PayloadMaxBytes) {
			return 0, 0, stdoutLimit()
		}
	} else if p > uint64(descriptor.SourceMaxBytes) {
		return 0, 0, sourceLimit()
	}
	// Both conversions are now below fixed limits on every supported int width.
	return int(m), int(p), nil
}

func ReadRequest(r io.Reader) (RequestMeta, []byte, error) {
	return ReadRequestFor(MarkdownHTML, r)
}

func ReadRequestFor(expected EngineID, r io.Reader) (RequestMeta, []byte, error) {
	descriptor, ok := enginecatalog.Lookup(expected)
	if !ok {
		return RequestMeta{}, nil, invalid("unsupported expected engine identity")
	}
	var meta RequestMeta
	h := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, h); err != nil {
		return meta, nil, invalid("truncated request header")
	}
	m, p, err := checkHeader(h, false, descriptor)
	if err != nil {
		return meta, nil, err
	}
	data := make([]byte, m)
	if _, err := io.ReadFull(r, data); err != nil {
		return meta, nil, invalid("truncated request metadata")
	}
	meta, err = decodeRequest(data, expected, descriptor.Format)
	if err != nil {
		return meta, nil, err
	}
	payload := make([]byte, p)
	if _, err := io.ReadFull(r, payload); err != nil {
		return meta, nil, invalid("truncated source")
	}
	if !utf8.Valid(payload) {
		return meta, nil, &Failure{Kind: "protocol", Code: "invalid-utf8", Detail: "source is not valid UTF8"}
	}
	var extra [1]byte
	n, err := io.ReadFull(r, extra[:])
	if n != 0 || err != io.EOF {
		return meta, nil, invalid("request must end at EOF")
	}
	return meta, payload, nil
}

func WriteResponse(w io.Writer, meta ResponseMeta, payload []byte) error {
	descriptor, ok := enginecatalog.Lookup(meta.Engine)
	if !ok || meta.Format != descriptor.Format {
		return invalid("invalid response identity")
	}
	if int64(len(payload)) > descriptor.PayloadMaxBytes {
		return stdoutLimit()
	}
	if !utf8.Valid(payload) {
		return invalid("response is not valid UTF8")
	}
	if err := validateResponse(meta, len(payload), uint64(descriptor.SourceMaxBytes)); err != nil {
		return err
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return invalid("response metadata encoding failed")
	}
	if len(data) > ResponseMetaMaxBytes {
		return stdoutLimit()
	}
	for _, part := range [][]byte{makeHeader(ResponseMagic, len(data), uint64(len(payload))), data, payload} {
		n, err := w.Write(part)
		if err != nil {
			return err
		}
		if n != len(part) {
			return io.ErrShortWrite
		}
	}
	return nil
}

func ValidEngineVersion(v string) bool {
	if len(v) == 0 || len(v) > EngineVersionMaxBytes {
		return false
	}
	for i := range len(v) {
		if v[i] < 0x21 || v[i] > 0x7e {
			return false
		}
	}
	return true
}

func validErrorCode(code string) bool {
	switch code {
	case "invalid-request", "invalid-page", "source-limit", "invalid-utf8", "output-limit", "warning-limit", "render-failed", "partial-document":
		return true
	}
	return false
}

func validateResponse(m ResponseMeta, payloadBytes int, sourceBytes uint64) error {
	descriptor, ok := enginecatalog.Lookup(m.Engine)
	if !ok || m.Format != descriptor.Format || !ValidEngineVersion(m.EngineVersion) {
		return invalid("invalid response identity")
	}
	if m.Warnings == nil || len(m.Warnings) > WarningsMaxCount {
		return invalid("invalid warning array")
	}
	for _, warning := range m.Warnings {
		if len(warning.Code) == 0 || len(warning.Code) > WarningCodeMaxBytes || !utf8.ValidString(warning.Code) || len(warning.Message) > WarningMessageMaxBytes || !utf8.ValidString(warning.Message) || warning.Start > warning.End || uint64(warning.End) > sourceBytes {
			return invalid("invalid warning metadata")
		}
	}
	switch m.Status {
	case StatusOK:
		if m.Error != nil {
			return invalid("success contains an error")
		}
	case StatusError:
		if payloadBytes != 0 || m.Error == nil || !validErrorCode(m.Error.Code) || len(m.Error.Message) > ErrorMessageMaxBytes || !utf8.ValidString(m.Error.Message) {
			return invalid("invalid error response")
		}
	default:
		return invalid("invalid response status")
	}
	return nil
}
