package engineipc

import (
	"unicode/utf8"

	"levnote.local/native/core/enginecatalog"
)

// ResponseDecoder accepts arbitrarily fragmented writes from os/exec. Write is
// nonblocking. Its first failure is terminal and releases any payload buffer.
type ResponseDecoder struct {
	descriptor                     enginecatalog.Descriptor
	version                        string
	sourceBytes                    uint64
	header                         [HeaderSize]byte
	headerN                        int
	metadata                       []byte
	metaN                          int
	meta                           ResponseMeta
	payload                        []byte
	payloadBytes, payloadN         int
	metaParsed, complete, finished bool
	err                            error
}

func NewResponseDecoder(engine EngineID, version string, sourceBytes uint64) *ResponseDecoder {
	descriptor, ok := enginecatalog.Lookup(engine)
	d := &ResponseDecoder{descriptor: descriptor, version: version, sourceBytes: sourceBytes}
	if !ok || !ValidEngineVersion(version) || sourceBytes > uint64(descriptor.SourceMaxBytes) {
		d.fail(invalid("invalid expected response identity"))
	}
	return d
}
func (d *ResponseDecoder) fail(err error) error {
	if d.err == nil {
		d.err = err
	}
	d.payload = nil
	d.metadata = nil
	d.meta = ResponseMeta{}
	return d.err
}
func (d *ResponseDecoder) Write(p []byte) (int, error) {
	if d.err != nil {
		return 0, d.err
	}
	total := 0
	for len(p) > 0 {
		if d.complete || d.finished {
			return total, d.fail(invalid("extra response bytes"))
		}
		if d.headerN < HeaderSize {
			n := copy(d.header[d.headerN:], p)
			d.headerN += n
			p = p[n:]
			total += n
			if d.headerN < HeaderSize {
				continue
			}
			metaBytes, payloadBytes, err := checkHeader(d.header[:], true, d.descriptor)
			if err != nil {
				return total, d.fail(err)
			}
			d.metadata = make([]byte, metaBytes)
			d.payloadBytes = payloadBytes
		}
		if !d.metaParsed {
			n := copy(d.metadata[d.metaN:], p)
			d.metaN += n
			p = p[n:]
			total += n
			if d.metaN < len(d.metadata) {
				continue
			}
			meta, err := decodeResponse(d.metadata, d.payloadBytes, d.sourceBytes)
			if err != nil {
				return total, d.fail(err)
			}
			if meta.Engine != d.descriptor.Engine || meta.EngineVersion != d.version {
				return total, d.fail(invalid("response identity does not match bundle"))
			}
			d.meta = meta
			d.metadata = nil
			d.metaParsed = true
			if d.payloadBytes > 0 {
				d.payload = make([]byte, d.payloadBytes)
			}
		}
		n := copy(d.payload[d.payloadN:], p)
		d.payloadN += n
		p = p[n:]
		total += n
		if d.payloadN == d.payloadBytes {
			if !utf8.Valid(d.payload) {
				detail := "response payload is not valid UTF8"
				if d.descriptor.Engine == MarkdownHTML {
					detail = "HTML is not valid UTF8"
				}
				return total, d.fail(invalid(detail))
			}
			d.complete = true
		}
	}
	return total, nil
}

// Finish must only be called after stdout EOF and successful process/copy-loop
// completion. It transfers the sole result slice; repeated Finish is invalid.
func (d *ResponseDecoder) Finish() (ResponseMeta, []byte, error) {
	if d.err != nil {
		return ResponseMeta{}, nil, d.err
	}
	if !d.complete || d.finished {
		return ResponseMeta{}, nil, d.fail(invalid("incomplete or already consumed response"))
	}
	d.finished = true
	meta, payload := d.meta, d.payload
	d.meta = ResponseMeta{}
	d.payload = nil
	return meta, payload, nil
}
