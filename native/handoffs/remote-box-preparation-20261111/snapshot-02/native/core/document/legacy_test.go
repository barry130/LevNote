package document

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
)

func TestLegacyEncodingRequiresExplicitChoiceAndPreservesBytes(t *testing.T) {
	for _, enc := range []Encoding{GBK, GB18030} {
		t.Run(string(enc), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "中文.txt")
			value := "中文\r\n原文"
			if enc == GB18030 {
				value += "🙂"
			}
			original, err := encode(value, enc, false)
			if err != nil {
				t.Fatal(err)
			}
			put(t, path, original)
			if _, err = Open(path); !errors.Is(err, ErrUnsupportedEncoding) {
				t.Fatal("auto-guessed legacy encoding", err)
			}
			doc, err := OpenWithEncoding(path, enc)
			if err != nil || doc.Buffer.String() != value {
				t.Fatal(err)
			}
			if err = doc.Save(); err != nil || !bytes.Equal(read(t, path), original) {
				t.Fatal("unchanged roundtrip", err)
			}
			_ = doc.Buffer.ReplaceUTF16(0, 1, "新")
			if err = doc.Save(); err != nil {
				t.Fatal(err)
			}
			if err = doc.Revert(); err != nil || doc.Buffer.String() != "新"+string([]rune(value)[1:]) {
				t.Fatal("explicit encoding lost on reload", err)
			}
		})
	}
}
func TestUnrepresentableGBKAndMalformedInputNeverRewrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gbk.txt")
	original, _ := encode("中文", GBK, false)
	put(t, path, original)
	doc, err := OpenWithEncoding(path, GBK)
	if err != nil {
		t.Fatal(err)
	}
	_ = doc.Buffer.ReplaceUTF16(0, 0, "🙂")
	if err = doc.Save(); !errors.Is(err, ErrUnrepresentableEncoding) {
		t.Fatal(err)
	}
	if !bytes.Equal(read(t, path), original) {
		t.Fatal("unrepresentable text changed disk")
	}
	for _, raw := range [][]byte{{0x81}, {0xfe, 0xfe}, {0x81, 0x30}} {
		put(t, path, raw)
		if _, err = OpenWithEncoding(path, GBK); !errors.Is(err, ErrUnsupportedEncoding) {
			t.Fatal(raw, err)
		}
	}
}
