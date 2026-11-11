package document

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"unicode/utf16"
	"unicode/utf8"
)

func legacyEncoding(enc Encoding) encoding.Encoding {
	switch enc {
	case GBK:
		return simplifiedchinese.GBK
	case GB18030:
		return simplifiedchinese.GB18030
	}
	return nil
}

func decodeWith(data []byte, forced Encoding) (string, Encoding, bool, error) {
	if forced == "" {
		return decode(data)
	}
	if codec := legacyEncoding(forced); codec != nil {
		if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) || bytes.HasPrefix(data, []byte{0xff, 0xfe}) || bytes.HasPrefix(data, []byte{0xfe, 0xff}) {
			return "", "", false, ErrUnsupportedEncoding
		}
		value, err := codec.NewDecoder().Bytes(data)
		if err != nil {
			return "", "", false, ErrUnsupportedEncoding
		}
		// x/text decoders replace malformed sequences with U+FFFD. Exact encode
		// comparison detects that loss, including ambiguous noncanonical codes.
		roundTrip, err := codec.NewEncoder().Bytes(value)
		if err != nil || !bytes.Equal(data, roundTrip) {
			return "", "", false, ErrUnsupportedEncoding
		}
		if bytes.IndexByte(value, 0) >= 0 {
			return "", "", false, ErrBinary
		}
		return string(value), forced, false, nil
	}
	if forced != UTF8 && forced != UTF16LE && forced != UTF16BE {
		return "", "", false, ErrUnsupportedEncoding
	}
	value, enc, bom, err := decode(data)
	if err != nil {
		return "", "", false, err
	}
	if enc != forced {
		return "", "", false, ErrUnsupportedEncoding
	}
	return value, enc, bom, nil
}

func decode(data []byte) (string, Encoding, bool, error) {
	for _, magic := range [][]byte{
		[]byte("%PDF-"), []byte("PK\x03\x04"), []byte("PK\x05\x06"),
		[]byte("\x89PNG\r\n\x1a\n"), []byte("GIF87a"), []byte("GIF89a"),
		[]byte("\xff\xd8\xff"), []byte("\x7fELF"),
	} {
		if bytes.HasPrefix(data, magic) {
			return "", "", false, ErrBinary
		}
	}
	enc, bom := UTF8, false
	if bytes.HasPrefix(data, []byte{0xff, 0xfe, 0, 0}) || bytes.HasPrefix(data, []byte{0, 0, 0xfe, 0xff}) {
		return "", "", false, ErrUnsupportedEncoding
	}
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		bom = true
		data = data[3:]
	} else if bytes.HasPrefix(data, []byte{0xff, 0xfe}) {
		enc = UTF16LE
		bom = true
		data = data[2:]
	} else if bytes.HasPrefix(data, []byte{0xfe, 0xff}) {
		enc = UTF16BE
		bom = true
		data = data[2:]
	}
	var value string
	if enc == UTF8 {
		if !utf8.Valid(data) {
			return "", "", false, ErrUnsupportedEncoding
		}
		value = string(data)
	} else {
		if len(data)%2 != 0 {
			return "", "", false, ErrUnsupportedEncoding
		}
		var order binary.ByteOrder = binary.LittleEndian
		if enc == UTF16BE {
			order = binary.BigEndian
		}
		units := make([]uint16, len(data)/2)
		for i := range units {
			units[i] = order.Uint16(data[2*i:])
		}
		for i := 0; i < len(units); i++ {
			u := units[i]
			if u >= 0xd800 && u <= 0xdbff {
				if i+1 == len(units) || units[i+1] < 0xdc00 || units[i+1] > 0xdfff {
					return "", "", false, ErrUnsupportedEncoding
				}
				i++
			} else if u >= 0xdc00 && u <= 0xdfff {
				return "", "", false, ErrUnsupportedEncoding
			}
		}
		value = string(utf16.Decode(units))
	}
	// Do not reinterpret known binary signatures or embedded NUL as text.
	// Other control characters remain lossless (logs may legitimately have ESC).
	if bytes.IndexByte([]byte(value), 0) >= 0 {
		return "", "", false, ErrBinary
	}
	return value, enc, bom, nil
}

func encode(value string, enc Encoding, bom bool) ([]byte, error) {
	if !utf8.ValidString(value) {
		return nil, ErrUnsupportedEncoding
	}
	if codec := legacyEncoding(enc); codec != nil {
		if bom {
			return nil, ErrUnsupportedEncoding
		}
		result, err := codec.NewEncoder().String(value)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUnrepresentableEncoding, err)
		}
		return []byte(result), nil
	}
	if enc == UTF8 {
		if bom {
			return append([]byte{0xef, 0xbb, 0xbf}, []byte(value)...), nil
		}
		return []byte(value), nil
	}
	if enc != UTF16LE && enc != UTF16BE {
		return nil, ErrUnsupportedEncoding
	}
	// UTF-16 without a BOM cannot be reliably reopened by automatic detection.
	// Requiring it is explicit rather than silently changing encoding metadata.
	if !bom {
		return nil, ErrUnsupportedEncoding
	}
	units := utf16.Encode([]rune(value))
	data := make([]byte, len(units)*2+2)
	var order binary.ByteOrder = binary.LittleEndian
	if enc == UTF16BE {
		order = binary.BigEndian
	}
	order.PutUint16(data, 0xfeff)
	for i, u := range units {
		order.PutUint16(data[2+2*i:], u)
	}
	return data, nil
}
