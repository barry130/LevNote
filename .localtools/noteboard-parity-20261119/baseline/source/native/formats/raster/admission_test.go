package raster

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"testing"
)

type readAtFunc func(p []byte, off int64) (int, error)

func (f readAtFunc) ReadAt(p []byte, off int64) (int, error) {
	return f(p, off)
}

type rejectionProbe struct {
	t      *testing.T
	reason RejectionReason
}

func rejectWith(t *testing.T, reason RejectionReason) rejectionProbe {
	t.Helper()
	return rejectionProbe{t: t, reason: reason}
}

func (p rejectionProbe) Test(actual error) {
	p.t.Helper()
	if actual == nil {
		p.t.Fatalf("expected rejection %q, got success", p.reason)
	}
	var rejection *RejectionError
	if !errors.As(actual, &rejection) {
		p.t.Fatalf("expected *RejectionError, got %T: %v", actual, actual)
	}
	if rejection.Reason != p.reason {
		p.t.Fatalf("reason = %q, want %q (%v)", rejection.Reason, p.reason, actual)
	}
}

func pngFixture(t *testing.T, img image.Image) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return out.Bytes()
}

func opaqueRGBA(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(20*x + 1),
				G: uint8(30*y + 2),
				B: uint8(10*x + 20*y + 3),
				A: 255,
			})
		}
	}
	return img
}

func translucentNRGBA(width, height int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(20*x + 1),
				G: uint8(30*y + 2),
				B: 9,
				A: 128,
			})
		}
	}
	return img
}

func palettedImage(width, height, entries int) *image.Paletted {
	palette := make(color.Palette, entries)
	for i := range palette {
		palette[i] = color.RGBA{
			R: uint8(i),
			G: uint8(255 - i),
			B: uint8(i * 17),
			A: 255,
		}
	}
	return image.NewPaletted(image.Rect(0, 0, width, height), palette)
}

func pngChunk(t *testing.T, kind string, data []byte) []byte {
	t.Helper()
	if len(kind) != 4 {
		t.Fatalf("PNG chunk type %q is not four bytes", kind)
	}
	chunk := make([]byte, 12+len(data))
	binary.BigEndian.PutUint32(chunk[0:4], uint32(len(data)))
	copy(chunk[4:8], kind)
	copy(chunk[8:8+len(data)], data)
	binary.BigEndian.PutUint32(chunk[8+len(data):12+len(data)], crc32.ChecksumIEEE(chunk[4:8+len(data)]))
	return chunk
}

func findPNGChunk(t *testing.T, fixture []byte, target string) (int, int) {
	t.Helper()
	if len(fixture) < 8 || string(fixture[:8]) != string(pngSignature) {
		t.Fatalf("fixture is not PNG")
	}
	for offset := 8; offset < len(fixture); {
		if offset+12 > len(fixture) {
			t.Fatalf("malformed PNG fixture at offset %d", offset)
		}
		length := int(binary.BigEndian.Uint32(fixture[offset : offset+4]))
		kind := string(fixture[offset+4 : offset+8])
		end := offset + 12 + length
		if length < 0 || end < offset || end > len(fixture) {
			t.Fatalf("malformed PNG chunk %q at offset %d", kind, offset)
		}
		if kind == target {
			return offset, end
		}
		offset = end
	}
	t.Fatalf("PNG chunk %q not found", target)
	return 0, 0
}

func insertPNGChunk(t *testing.T, fixture []byte, kind string, data []byte, before string) []byte {
	t.Helper()
	offset, _ := findPNGChunk(t, fixture, before)
	chunk := pngChunk(t, kind, data)
	out := make([]byte, 0, len(fixture)+len(chunk))
	out = append(out, fixture[:offset]...)
	out = append(out, chunk...)
	out = append(out, fixture[offset:]...)
	return out
}

func replaceIHDR(t *testing.T, fixture []byte, edit func(ihdr []byte)) []byte {
	t.Helper()
	patched := append([]byte(nil), fixture...)
	if len(patched) < 33 || string(patched[12:16]) != "IHDR" {
		t.Fatalf("fixture first chunk is not IHDR")
	}
	if length := int(binary.BigEndian.Uint32(patched[8:12])); length != 13 {
		t.Fatalf("fixture IHDR length = %d, want 13", length)
	}
	edit(patched[16:29])
	binary.BigEndian.PutUint32(patched[29:33], crc32.ChecksumIEEE(patched[12:29]))
	return patched
}

func pngHeader(t *testing.T, fixture []byte) (width, height uint32, depth, colorType, interlace uint8) {
	t.Helper()
	if len(fixture) < 33 || string(fixture[:8]) != string(pngSignature) || string(fixture[12:16]) != "IHDR" {
		t.Fatalf("fixture does not begin with PNG signature and IHDR")
	}
	if length := binary.BigEndian.Uint32(fixture[8:12]); length != 13 {
		t.Fatalf("IHDR length = %d, want 13", length)
	}
	return binary.BigEndian.Uint32(fixture[16:20]),
		binary.BigEndian.Uint32(fixture[20:24]),
		fixture[24],
		fixture[25],
		fixture[28]
}

func pngPayload(
	size, pixels, destinationBytesPerPixel, bitsPerPixel, rowWidth, orientation int64,
) int64 {
	rows := 2 * (1 + (bitsPerPixel*rowWidth+7)/8)
	payload := size + destinationBytesPerPixel*pixels + rows + 4*pixels
	if orientation >= 2 && orientation <= 8 {
		payload += 4 * pixels
	}
	return payload
}

func jpegFixture(t *testing.T, img image.Image, opts *jpeg.Options) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, opts); err != nil {
		t.Fatalf("jpeg.Encode: %v", err)
	}
	return out.Bytes()
}

// findJPEGMarker walks complete marker segments and stops at SOS. It never
// searches entropy-coded bytes.
func findJPEGMarker(t *testing.T, fixture []byte, target byte) int {
	t.Helper()
	if len(fixture) < 4 || fixture[0] != 0xff || fixture[1] != 0xd8 {
		t.Fatalf("fixture does not begin with JPEG SOI")
	}
	for offset := 2; offset < len(fixture); {
		markerStart := offset
		if fixture[offset] != 0xff {
			t.Fatalf("JPEG non-marker byte at offset %d", offset)
		}
		for offset < len(fixture) && fixture[offset] == 0xff {
			offset++
		}
		if offset >= len(fixture) {
			t.Fatalf("JPEG marker runs to end")
		}
		marker := fixture[offset]
		offset++
		if marker == target {
			return markerStart
		}
		switch marker {
		case 0x01, 0xd0, 0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7:
			continue
		case 0xd8, 0xd9, 0x00:
			t.Fatalf("unexpected standalone JPEG marker 0x%02x before target 0x%02x", marker, target)
		}
		if offset+2 > len(fixture) {
			t.Fatalf("JPEG segment length is truncated")
		}
		length := int(binary.BigEndian.Uint16(fixture[offset : offset+2]))
		if length < 2 || offset+length > len(fixture) {
			t.Fatalf("JPEG marker 0x%02x has invalid segment length %d", marker, length)
		}
		if marker == 0xda {
			t.Fatalf("JPEG SOS reached before target marker 0x%02x", target)
		}
		offset += length
	}
	t.Fatalf("JPEG marker 0x%02x not found before SOS", target)
	return 0
}

func insertJPEGBytes(t *testing.T, fixture, inserted []byte, before byte) []byte {
	t.Helper()
	offset := findJPEGMarker(t, fixture, before)
	out := make([]byte, 0, len(fixture)+len(inserted))
	out = append(out, fixture[:offset]...)
	out = append(out, inserted...)
	out = append(out, fixture[offset:]...)
	return out
}

func insertJPEGSegment(t *testing.T, fixture []byte, marker byte, data []byte, before byte) []byte {
	t.Helper()
	return insertJPEGBytes(t, fixture, jpegSegment(t, marker, data), before)
}

func jpegSegment(t *testing.T, marker byte, data []byte) []byte {
	t.Helper()
	if len(data)+2 > math.MaxUint16 {
		t.Fatalf("JPEG segment too long")
	}
	segment := make([]byte, 4+len(data))
	segment[0], segment[1] = 0xff, marker
	binary.BigEndian.PutUint16(segment[2:4], uint16(2+len(data)))
	copy(segment[4:], data)
	return segment
}

func minimalJPEG(segments ...[]byte) []byte {
	out := []byte{0xff, 0xd8}
	for _, segment := range segments {
		out = append(out, segment...)
	}
	return out
}

func jpegSOF3(width, height int, lumaHV, chromaHV byte) []byte {
	sof := make([]byte, 15)
	sof[0] = 8
	binary.BigEndian.PutUint16(sof[1:3], uint16(height))
	binary.BigEndian.PutUint16(sof[3:5], uint16(width))
	sof[5] = 3
	sof[6], sof[7], sof[8] = 1, lumaHV, 0
	sof[9], sof[10], sof[11] = 2, chromaHV, 0
	sof[12], sof[13], sof[14] = 3, chromaHV, 0
	return sof
}

func jpegSOF1(width, height int) []byte {
	sof := make([]byte, 9)
	sof[0] = 8
	binary.BigEndian.PutUint16(sof[1:3], uint16(height))
	binary.BigEndian.PutUint16(sof[3:5], uint16(width))
	sof[5] = 1
	sof[6], sof[7], sof[8] = 1, 0x11, 0
	return sof
}

func minimalSOS(componentCount int) []byte {
	sos := make([]byte, 1+2*componentCount+3)
	sos[0] = byte(componentCount)
	for i := 0; i < componentCount; i++ {
		sos[1+2*i] = byte(i + 1)
	}
	sos[1+2*componentCount+1] = 63
	return jpegSegmentForTest(0xda, sos)
}

func jpegSegmentForTest(marker byte, data []byte) []byte {
	segment := make([]byte, 4+len(data))
	segment[0], segment[1] = 0xff, marker
	binary.BigEndian.PutUint16(segment[2:4], uint16(2+len(data)))
	copy(segment[4:], data)
	return segment
}

func encodedJPEGSOF(t *testing.T, fixture []byte) []byte {
	t.Helper()
	offset := findJPEGMarker(t, fixture, 0xc0)
	if offset+4 > len(fixture) {
		t.Fatalf("SOF marker header truncated")
	}
	length := int(binary.BigEndian.Uint16(fixture[offset+2 : offset+4]))
	if length < 2 || offset+2+length > len(fixture) {
		t.Fatalf("SOF segment malformed")
	}
	return append([]byte(nil), fixture[offset+4:offset+2+length]...)
}

func exifTIFF(t *testing.T, order binary.ByteOrder, orientations ...uint16) []byte {
	t.Helper()
	tiff := make([]byte, 8, 8+2+12*len(orientations)+4)
	if order == binary.LittleEndian {
		copy(tiff, "II")
	} else if order == binary.BigEndian {
		copy(tiff, "MM")
	} else {
		t.Fatalf("unsupported byte order")
	}
	order.PutUint16(tiff[2:4], 42)
	order.PutUint32(tiff[4:8], 8)
	tiff = binary.BigEndian.AppendUint16(tiff, 0)
	if order == binary.LittleEndian {
		tiff = tiff[:len(tiff)-2]
		tiff = append(tiff, byte(len(orientations)), 0)
	} else {
		order.PutUint16(tiff[len(tiff)-2:], uint16(len(orientations)))
	}
	for _, orientation := range orientations {
		entry := make([]byte, 12)
		order.PutUint16(entry[0:2], orientationTag)
		order.PutUint16(entry[2:4], 3)
		order.PutUint32(entry[4:8], 1)
		order.PutUint16(entry[8:10], orientation)
		tiff = append(tiff, entry...)
	}
	return append(tiff, 0, 0, 0, 0)
}

func TestAdmitValidPNGVariants(t *testing.T) {
	tests := []struct {
		name            string
		img             image.Image
		depth           uint8
		colorType       uint8
		destinationBPP  int64
		bitsPerPixel    int64
		baseRGBAFormula int64
	}{
		{"grayscale8", image.NewGray(image.Rect(0, 0, 3, 2)), 8, 0, 1, 8, 24},
		{"truecolor8", opaqueRGBA(3, 2), 8, 2, 4, 24, 24},
		{"truecoloralpha8", translucentNRGBA(3, 2), 8, 6, 4, 32, 24},
		{"paletted1", palettedImage(3, 2, 2), 1, 3, 1, 1, 24},
		{"paletted8", palettedImage(3, 2, 256), 8, 3, 1, 8, 24},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := pngFixture(t, test.img)
			width, height, depth, colorType, interlace := pngHeader(t, fixture)
			if width != 3 || height != 2 || depth != test.depth || colorType != test.colorType || interlace != 0 {
				t.Fatalf(
					"encoded IHDR = %dx%d depth=%d color=%d interlace=%d, want 3x2 depth=%d color=%d interlace=0",
					width, height, depth, colorType, interlace, test.depth, test.colorType,
				)
			}

			got, err := Admit(bytes.NewReader(fixture), int64(len(fixture)))
			if err != nil {
				t.Fatalf("Admit: %v", err)
			}
			if got.Kind != KindPNG || got.Subset != SubsetPNG8 {
				t.Fatalf("kind/subset = %v/%v", got.Kind, got.Subset)
			}
			if got.Width != 3 || got.Height != 2 || got.Pixels != 6 {
				t.Fatalf("dimensions = %dx%d, pixels = %d", got.Width, got.Height, got.Pixels)
			}
			if got.Orientation != 0 || got.BaseRGBABytes != test.baseRGBAFormula || got.MipLevels != 3 {
				t.Fatalf(
					"orientation/baseRGBABytes/mips = %d/%d/%d",
					got.Orientation, got.BaseRGBABytes, got.MipLevels,
				)
			}
			want := pngPayload(
				int64(len(fixture)),
				got.Pixels,
				test.destinationBPP,
				test.bitsPerPixel,
				int64(got.Width),
				int64(got.Orientation),
			)
			if got.DecodePayloadBytes != want {
				t.Fatalf("payload = %d, want %d", got.DecodePayloadBytes, want)
			}
		})
	}
}

func TestAdmitPNGMetadataAndTransparency(t *testing.T) {
	base := pngFixture(t, image.NewGray(image.Rect(0, 0, 3, 2)))

	orientation := insertPNGChunk(t, base, "eXIf", exifTIFF(t, binary.LittleEndian, 6), "IEND")
	got, err := Admit(bytes.NewReader(orientation), int64(len(orientation)))
	if err != nil {
		t.Fatalf("Admit PNG with eXIf after IDAT: %v", err)
	}
	if got.Orientation != 6 {
		t.Fatalf("orientation = %d, want 6", got.Orientation)
	}
	want := pngPayload(int64(len(orientation)), 6, 1, 8, 3, 6)
	if got.DecodePayloadBytes != want {
		t.Fatalf("oriented payload = %d, want %d", got.DecodePayloadBytes, want)
	}

	trns := insertPNGChunk(t, base, "tRNS", []byte{0, 255}, "IDAT")
	got, err = Admit(bytes.NewReader(trns), int64(len(trns)))
	if err != nil {
		t.Fatalf("Admit PNG with tRNS: %v", err)
	}
	want = pngPayload(int64(len(trns)), 6, 4, 8, 3, 0)
	if got.DecodePayloadBytes != want {
		t.Fatalf("tRNS payload = %d, want %d", got.DecodePayloadBytes, want)
	}

	duplicate := insertPNGChunk(t, orientation, "eXIf", exifTIFF(t, binary.BigEndian, 3), "IDAT")
	_, err = Admit(bytes.NewReader(duplicate), int64(len(duplicate)))
	rejectWith(t, RejectionMetadata).Test(err)
}

func TestAdmitPNGRejections(t *testing.T) {
	base := pngFixture(t, opaqueRGBA(3, 2))
	tooManyChunks := base
	for i := 0; i < 4097; i++ {
		tooManyChunks = insertPNGChunk(t, tooManyChunks, "ruSt", nil, "IDAT")
	}

	tests := []struct {
		name    string
		fixture []byte
		size    int64
		reason  RejectionReason
	}{
		{"sixteen_bit", replaceIHDR(t, base, func(ihdr []byte) { ihdr[8] = 16 }), int64(len(base)), RejectionUnsupported},
		{"illegal_color_depth", replaceIHDR(t, base, func(ihdr []byte) { ihdr[9] = 5; ihdr[8] = 8 }), int64(len(base)), RejectionUnsupported},
		{"interlaced", replaceIHDR(t, base, func(ihdr []byte) { ihdr[12] = 1 }), int64(len(base)), RejectionUnsupported},
		{"apng_control", insertPNGChunk(t, base, "acTL", make([]byte, 8), "IDAT"), int64(len(base) + 20), RejectionUnsupported},
		{"apng_frame_control", insertPNGChunk(t, base, "fcTL", make([]byte, 26), "IDAT"), int64(len(base) + 38), RejectionUnsupported},
		{"apng_frame_data", insertPNGChunk(t, base, "fdAT", make([]byte, 4), "IDAT"), int64(len(base) + 16), RejectionUnsupported},
		{"trns_after_idat", insertPNGChunk(t, base, "tRNS", []byte{0, 0, 0, 0, 0, 0}, "IEND"), int64(len(base) + 18), RejectionInvalid},
		{"unknown_critical", insertPNGChunk(t, base, "ABCD", nil, "IDAT"), int64(len(base) + 12), RejectionInvalid},
		{"trailing_bytes", append(append([]byte(nil), base...), 0, 0), int64(len(base) + 2), RejectionInvalid},
		{"too_many_chunks", tooManyChunks, int64(len(tooManyChunks)), RejectionInvalid},
		{"too_wide", replaceIHDR(t, base, func(ihdr []byte) {
			binary.BigEndian.PutUint32(ihdr[0:4], MaxDimension+1)
		}), int64(len(base)), RejectionDimensions},
		{"too_many_pixels", replaceIHDR(t, base, func(ihdr []byte) {
			binary.BigEndian.PutUint32(ihdr[0:4], 4096)
			binary.BigEndian.PutUint32(ihdr[4:8], 2049)
		}), int64(len(base)), RejectionDimensions},
		{"truncated_ihdr", base[:30], 30, RejectionTruncated},
		{"compressed_limit", base, MaxCompressedBytes + 1, RejectionCompressed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Admit(bytes.NewReader(test.fixture), test.size)
			rejectWith(t, test.reason).Test(err)
		})
	}
}

func TestAdmitPNGLengthAndCRCIndependentEnvelope(t *testing.T) {
	base := pngFixture(t, opaqueRGBA(3, 2))
	idatOffset, _ := findPNGChunk(t, base, "IDAT")
	chunkHeader := make([]byte, 8)
	binary.BigEndian.PutUint32(chunkHeader[0:4], 0xffffffff)
	copy(chunkHeader[4:8], "ruSt")
	fixture := make([]byte, 0, len(base)+len(chunkHeader))
	fixture = append(fixture, base[:idatOffset]...)
	fixture = append(fixture, chunkHeader...)
	fixture = append(fixture, base[idatOffset:]...)
	_, err := Admit(bytes.NewReader(fixture), int64(len(fixture)))
	rejectWith(t, RejectionInvalid).Test(err)

	badCRC := append([]byte(nil), base...)
	badCRC[29] ^= 0xff
	if _, err := Admit(bytes.NewReader(badCRC), int64(len(badCRC))); err != nil {
		t.Fatalf("header-only admission unexpectedly validated IHDR CRC: %v", err)
	}
}

func TestExifBoundedParsing(t *testing.T) {
	little := exifTIFF(t, binary.LittleEndian, 5)
	big := exifTIFF(t, binary.BigEndian, 7)
	for _, test := range []struct {
		name string
		data []byte
		want uint8
	}{
		{"little_endian", little, 5},
		{"big_endian", big, 7},
		{"absent", exifTIFF(t, binary.LittleEndian), 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := exifOrientation(test.data)
			if err != nil || got != test.want {
				t.Fatalf("orientation = %d, %v; want %d, nil", got, err, test.want)
			}
		})
	}

	for length := 0; length < len(little); length++ {
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("short TIFF length %d panicked: %v", length, recovered)
				}
			}()
			if _, err := exifOrientation(little[:length]); err == nil {
				t.Fatalf("short TIFF length %d was accepted", length)
			}
		}()
	}

	pointerAtEnd := append([]byte("II*\x00\x00\x00\x00\x00"), 0xff, 0xff, 0xff, 0xfd)
	if _, err := exifOrientation(pointerAtEnd); err == nil {
		t.Fatal("TIFF IFD pointer outside payload was accepted")
	}
	duplicate := exifTIFF(t, binary.LittleEndian, 6, 7)
	if _, err := exifOrientation(duplicate); err == nil {
		t.Fatal("duplicate orientation tag was accepted")
	}
}

func TestAdmitJPEGPublicEXIF(t *testing.T) {
	base := jpegFixture(t, opaqueRGBA(3, 2), &jpeg.Options{Quality: 90})
	sof := encodedJPEGSOF(t, base)
	if len(sof) != 15 || sof[5] != 3 || sof[7] != 0x22 {
		t.Fatalf("encoded fixture is not the expected three-component 4:2:0 JPEG: %x", sof)
	}
	payload := append([]byte("Exif\x00\x00"), exifTIFF(t, binary.LittleEndian, 8)...)
	fixture := insertJPEGSegment(t, base, 0xe1, payload, 0xda)
	got, err := Admit(bytes.NewReader(fixture), int64(len(fixture)))
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if got.Orientation != 8 {
		t.Fatalf("orientation = %d, want 8", got.Orientation)
	}
	want := int64(len(fixture)) + 384 + jpegStreamScratchBytes + 4*6*3
	if got.DecodePayloadBytes != want {
		t.Fatalf("payload = %d, want %d", got.DecodePayloadBytes, want)
	}
}

func TestAdmitEncodedJPEGVariants(t *testing.T) {
	tests := []struct {
		name        string
		img         image.Image
		sofLength   int
		components  int
		lumaHV      byte
		wantSurface int64
		wantRGBA    int64
	}{
		{"grayscale", image.NewGray(image.Rect(0, 0, 3, 2)), 9, 1, 0x11, 8 * 8, 24},
		{"three_component", opaqueRGBA(3, 2), 15, 3, 0x22, 16*16 + 2*8*8, 48},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := jpegFixture(t, test.img, &jpeg.Options{Quality: 90})
			sof := encodedJPEGSOF(t, fixture)
			if len(sof) != test.sofLength || int(sof[5]) != test.components || sof[7] != test.lumaHV {
				t.Fatalf("encoded SOF = %x, want %d components with luma 0x%02x", sof, test.components, test.lumaHV)
			}
			got, err := Admit(bytes.NewReader(fixture), int64(len(fixture)))
			if err != nil {
				t.Fatalf("Admit: %v", err)
			}
			if got.Width != 3 || got.Height != 2 || got.Pixels != 6 || got.MipLevels != 3 {
				t.Fatalf("admission dimensions/mips = %dx%d/%d/%d", got.Width, got.Height, got.Pixels, got.MipLevels)
			}
			want := int64(len(fixture)) + test.wantSurface + jpegStreamScratchBytes + test.wantRGBA
			if got.DecodePayloadBytes != want {
				t.Fatalf("payload = %d, want %d", got.DecodePayloadBytes, want)
			}
		})
	}
}

func TestJPEGDecoderSurfaceBySampling(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		height      int
		lumaHV      byte
		chromaHV    byte
		wantSurface int64
	}{
		{"4:4:4_16x16", 16, 16, 0x11, 0x11, 16 * 16 * 3},
		{"4:4:0_17x17", 17, 17, 0x12, 0x11, 24*32 + 2*24*16},
		{"4:2:2_17x17", 17, 17, 0x21, 0x11, 32*24 + 2*16*24},
		{"4:2:0_16x16", 16, 16, 0x22, 0x11, 16*16 + 2*8*8},
		{"4:2:0_17x19", 17, 19, 0x22, 0x11, 32*32 + 2*16*16},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sof := jpegSegmentForTest(0xc0, jpegSOF3(test.width, test.height, test.lumaHV, test.chromaHV))
			fixture := minimalJPEG(sof, minimalSOS(3))
			got, err := Admit(bytes.NewReader(fixture), int64(len(fixture)))
			if err != nil {
				t.Fatalf("Admit: %v", err)
			}
			rgbaBytes := int64(test.width * test.height * 4)
			want := int64(len(fixture)) + test.wantSurface + jpegStreamScratchBytes + 2*rgbaBytes
			if got.DecodePayloadBytes != want {
				t.Fatalf("payload = %d, want %d (surface=%d)", got.DecodePayloadBytes, want, test.wantSurface)
			}
		})
	}
}

func TestAdmitJPEGGrayscaleBudgetAndFillBytes(t *testing.T) {
	grayFixture := minimalJPEG(
		jpegSegmentForTest(0xc0, jpegSOF1(17, 9)),
		minimalSOS(1),
	)
	gray, err := Admit(bytes.NewReader(grayFixture), int64(len(grayFixture)))
	if err != nil {
		t.Fatalf("Admit grayscale: %v", err)
	}
	graySurface := int64(24 * 16)
	grayRGBA := int64(17 * 9 * 4)
	wantGray := int64(len(grayFixture)) + graySurface + jpegStreamScratchBytes + grayRGBA
	if gray.DecodePayloadBytes != wantGray {
		t.Fatalf("grayscale payload = %d, want %d", gray.DecodePayloadBytes, wantGray)
	}

	filled := minimalJPEG(
		jpegSegmentForTest(0xc0, jpegSOF3(3, 2, 0x11, 0x11)),
		[]byte{0xff},
		minimalSOS(3),
	)
	if _, err := Admit(bytes.NewReader(filled), int64(len(filled))); err != nil {
		t.Fatalf("Admit legal fill byte: %v", err)
	}
	tooManyFills := minimalJPEG(
		jpegSegmentForTest(0xc0, jpegSOF3(3, 2, 0x11, 0x11)),
		bytes.Repeat([]byte{0xff}, int(maxJPEGSegments)+1),
		minimalSOS(3),
	)
	_, err = Admit(bytes.NewReader(tooManyFills), int64(len(tooManyFills)))
	rejectWith(t, RejectionInvalid).Test(err)
}

func TestAdmitJPEGUnsupportedModesAndSampling(t *testing.T) {
	base := jpegFixture(t, opaqueRGBA(3, 2), nil)
	tests := []struct {
		name    string
		fixture []byte
		reason  RejectionReason
	}{
		{
			"progressive",
			insertJPEGSegment(t, base, 0xc2, make([]byte, 9), 0xda),
			RejectionUnsupported,
		},
		{
			"lossless",
			insertJPEGSegment(t, base, 0xc3, make([]byte, 9), 0xda),
			RejectionUnsupported,
		},
		{
			"four_component",
			minimalJPEG(jpegSegmentForTest(0xc0, make([]byte, 18))),
			RejectionUnsupported,
		},
		{
			"sampling_three",
			minimalJPEG(jpegSegmentForTest(0xc0, jpegSOF3(3, 2, 0x13, 0x11))),
			RejectionUnsupported,
		},
		{
			"mixed_chroma",
			minimalJPEG(jpegSegmentForTest(0xc0, []byte{
				8, 0, 2, 0, 3, 3,
				1, 0x21, 0,
				2, 0x21, 0,
				3, 0x11, 0,
			})),
			RejectionUnsupported,
		},
		{
			"nonmaximal_luma",
			minimalJPEG(jpegSegmentForTest(0xc0, []byte{
				8, 0, 2, 0, 3, 3,
				1, 0x11, 0,
				2, 0x21, 0,
				3, 0x21, 0,
			})),
			RejectionUnsupported,
		},
		{
			"total_sampling",
			minimalJPEG(jpegSegmentForTest(0xc0, jpegSOF3(3, 2, 0x22, 0x22))),
			RejectionUnsupported,
		},
		{
			"unsupported_marker",
			insertJPEGSegment(t, base, 0xcc, []byte{0, 0}, 0xda),
			RejectionUnsupported,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Admit(bytes.NewReader(test.fixture), int64(len(test.fixture)))
			rejectWith(t, test.reason).Test(err)
		})
	}
}

func TestAdmitJPEGSOSEnvelope(t *testing.T) {
	sof := jpegSegmentForTest(0xc0, jpegSOF3(3, 2, 0x11, 0x11))
	tests := []struct {
		name string
		sos  []byte
	}{
		{"unknown_selector", jpegSegmentForTest(0xda, []byte{3, 1, 0, 2, 0, 4, 0, 0, 63, 0})},
		{"bad_table_selector", jpegSegmentForTest(0xda, []byte{3, 1, 0x20, 2, 0, 3, 0, 0, 63, 0})},
		{"bad_spectral_start", jpegSegmentForTest(0xda, []byte{3, 1, 0, 2, 0, 3, 0, 1, 63, 0})},
		{"bad_spectral_end", jpegSegmentForTest(0xda, []byte{3, 1, 0, 2, 0, 3, 0, 0, 62, 0})},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := minimalJPEG(sof, test.sos)
			_, err := Admit(bytes.NewReader(fixture), int64(len(fixture)))
			rejectWith(t, RejectionInvalid).Test(err)
		})
	}
}

func TestAdmitJPEGTruncatedAndOversized(t *testing.T) {
	base := jpegFixture(t, opaqueRGBA(3, 2), nil)
	_, err := Admit(bytes.NewReader(base[:20]), 20)
	rejectWith(t, RejectionTruncated).Test(err)

	minimal := minimalJPEG(
		jpegSegmentForTest(0xc0, jpegSOF3(3, 2, 0x11, 0x11)),
		minimalSOS(3),
	)
	_, err = Admit(bytes.NewReader(minimal), int64(len(minimal)-1))
	rejectWith(t, RejectionTruncated).Test(err)

	oversizedSegment := insertJPEGBytes(t, base, []byte{0xff, 0xe1, 0xff, 0xff}, 0xda)
	_, err = Admit(bytes.NewReader(oversizedSegment), int64(len(oversizedSegment)))
	rejectWith(t, RejectionTruncated).Test(err)

	tooWide := minimalJPEG(
		jpegSegmentForTest(0xc0, jpegSOF3(MaxDimension+1, 2, 0x11, 0x11)),
		minimalSOS(3),
	)
	_, err = Admit(bytes.NewReader(tooWide), int64(len(tooWide)))
	rejectWith(t, RejectionDimensions).Test(err)

	tooManyPixels := minimalJPEG(
		jpegSegmentForTest(0xc0, jpegSOF3(4096, 2049, 0x11, 0x11)),
		minimalSOS(3),
	)
	_, err = Admit(bytes.NewReader(tooManyPixels), int64(len(tooManyPixels)))
	rejectWith(t, RejectionDimensions).Test(err)

	_, err = Admit(bytes.NewReader(base), MaxCompressedBytes+1)
	rejectWith(t, RejectionCompressed).Test(err)
}

func TestExactLimitsAndCheckedArithmetic(t *testing.T) {
	under, underLevels, err := mipBudget(8192, 1151)
	if err != nil {
		t.Fatalf("mipBudget under boundary: %v", err)
	}
	if under != 50_299_036 || under >= MaxMipChainBytes || underLevels != 14 {
		t.Fatalf("under boundary = %d bytes/%d levels, want 50299036/14", under, underLevels)
	}
	over, _, err := mipBudget(8192, 1152)
	if err == nil {
		t.Fatalf("expected over-boundary rejection, got %d", over)
	}
	rejectWith(t, RejectionMipBudget).Test(err)

	if got, err := decodeBudget(MaxDecodePayload); err != nil || got != MaxDecodePayload {
		t.Fatalf("exact decode budget = %d, %v; want %d, nil", got, err, MaxDecodePayload)
	}
	_, err = decodeBudget(MaxDecodePayload + 1)
	rejectWith(t, RejectionDecodeBudget).Test(err)

	if _, _, err := mipBudget(1, 1); err != nil {
		t.Fatalf("1x1 mip budget: %v", err)
	}
	for _, dimensions := range [][2]int{{0, 1}, {1, 0}, {-1, 1}} {
		if _, _, err := mipBudget(dimensions[0], dimensions[1]); err == nil {
			t.Fatalf("mipBudget(%d, %d) was accepted", dimensions[0], dimensions[1])
		}
	}
	if _, err := checkedMul(math.MaxInt64, 2); err == nil {
		t.Fatal("checkedMul overflow was accepted")
	}
}

func TestAdmitReaderAtFailures(t *testing.T) {
	fixture := pngFixture(t, image.NewGray(image.Rect(0, 0, 1, 1)))

	fullCountWithError := readAtFunc(func(p []byte, off int64) (int, error) {
		n := copy(p, fixture[off:])
		return n, io.ErrUnexpectedEOF
	})
	_, err := Admit(fullCountWithError, int64(len(fixture)))
	rejectWith(t, RejectionRead).Test(err)

	calls := 0
	failsAfterSignature := readAtFunc(func(p []byte, off int64) (int, error) {
		calls++
		if calls == 1 {
			return copy(p, fixture[off:]), nil
		}
		return 0, errors.New("injected read failure")
	})
	_, err = Admit(failsAfterSignature, int64(len(fixture)))
	rejectWith(t, RejectionRead).Test(err)

	shortRead := readAtFunc(func(p []byte, off int64) (int, error) {
		if len(p) == 0 {
			return 0, nil
		}
		return len(p) - 1, nil
	})
	_, err = Admit(shortRead, int64(len(fixture)))
	rejectWith(t, RejectionTruncated).Test(err)

	_, err = Admit(nil, 1)
	rejectWith(t, RejectionRead).Test(err)

	_, err = Admit(bytes.NewReader(fixture), 0)
	rejectWith(t, RejectionTruncated).Test(err)

	_, err = Admit(bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7}), 8)
	rejectWith(t, RejectionUnsupported).Test(err)
}
