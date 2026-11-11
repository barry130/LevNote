package modules

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"testing"

	"levnote.local/native/formats/raster"
)

func TestImageBuiltinAdapterIsNeutralAndReadOnly(t *testing.T) {
	adapters := Builtins()
	adapter := adapters[len(adapters)-1]
	if adapter.ID != "image" || adapter.Create == nil || adapter.Matches == nil {
		t.Fatalf("image adapter = %#v", adapter)
	}
	for _, path := range []string{"photo.png", "photo.PNG", "photo.jpg", "photo.JPG", "photo.jpeg"} {
		if !adapter.Matches(path) {
			t.Fatalf("adapter rejected %q", path)
		}
	}
	for _, path := range []string{"photo.gif", "photo.svg", "note.md", "main.go"} {
		if adapter.Matches(path) {
			t.Fatalf("adapter accepted %q", path)
		}
	}
	if adapter.Template != nil || adapter.Outline != nil || adapter.ExportEngine != nil ||
		adapter.SceneEngine != nil || len(adapter.Exports) != 0 {
		t.Fatalf("image adapter invented template/export/outline/scene capabilities: %#v", adapter)
	}
}

func TestDecodeImageComposesPNGAndJPEGWithoutPixelCopies(t *testing.T) {
	pngBytes := encodePNG(t, image.NewRGBA(image.Rect(0, 0, 2, 3)))
	jpegBytes := encodeJPEG(t, image.NewGray(image.Rect(0, 0, 3, 2)))
	for _, fixture := range []struct {
		name string
		data []byte
		want string
	}{
		{"png", pngBytes, "PNG"},
		{"jpeg", jpegBytes, "JPEG"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			reader := bytes.NewReader(fixture.data)
			result, err := DecodeImage(context.Background(), reader, int64(len(fixture.data)))
			if err != nil {
				t.Fatal(err)
			}
			if result.Format != fixture.want {
				t.Fatalf("format = %q; want %q", result.Format, fixture.want)
			}
			if result.CompressedBytes != int64(len(fixture.data)) {
				t.Fatalf("compressed size = %d; want %d", result.CompressedBytes, len(fixture.data))
			}
			if result.Pixels == nil || len(result.Pixels.Pix) == 0 {
				t.Fatal("successful result has no pixels")
			}
			if result.Width != result.Pixels.Bounds().Dx() || result.Height != result.Pixels.Bounds().Dy() {
				t.Fatalf("final dimensions disagree: result=%d:%d pixels=%s", result.Width, result.Height, result.Pixels.Bounds())
			}
			before := &result.Pixels.Pix[0]
			again, err := DecodeImage(context.Background(), reader, int64(len(fixture.data)))
			if err != nil || &again.Pixels.Pix[0] == before {
				t.Fatalf("decode was cached or failed: %#v %v", again, err)
			}
		})
	}
}

func TestDecodeImageTransfersRejectionAndCancellation(t *testing.T) {
	if _, err := DecodeImage(context.Background(), nil, 1); err == nil {
		t.Fatal("nil ReaderAt accepted")
	}
	if _, err := DecodeImage(context.Background(), bytes.NewReader([]byte("not image")), 9); err == nil {
		t.Fatal("invalid signature accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := DecodeImage(ctx, bytes.NewReader(pngBytesFixture(t)), 64)
	var decodeErr *raster.DecodeError
	if !errors.As(err, &decodeErr) || decodeErr.Stage != raster.DecodeStageCanceled {
		t.Fatalf("cancellation error = %#v", err)
	}
}

func TestDecodeImageUsesDeclaredSizeWithoutReadingFile(t *testing.T) {
	fixture := pngBytesFixture(t)
	reader := &oneReadReaderAt{data: fixture, reads: 0}
	result, err := DecodeImage(context.Background(), reader, int64(len(fixture)))
	if err != nil {
		t.Fatal(err)
	}
	if result.CompressedBytes != int64(len(fixture)) || result.Format != "PNG" {
		t.Fatalf("result = %#v", result)
	}
	if reader.reads == 0 {
		t.Fatal("composition bypassed the real raster admission/decode")
	}
}

func pngBytesFixture(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	img.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	return encodePNG(t, img)
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, nil); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

type oneReadReaderAt struct {
	data  []byte
	reads int
}

func (r *oneReadReaderAt) ReadAt(p []byte, off int64) (int, error) {
	r.reads++
	if off < 0 || off > int64(len(r.data)) {
		return 0, io.EOF
	}
	n := copy(p, r.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}
