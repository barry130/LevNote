package raster

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"math"
	"reflect"
	"testing"
)

// rgba8 is one expected premultiplied pixel.
type rgba8 struct {
	r, g, b, a uint8
}

// premultiply is the documented color.NRGBA-to-premultiplied conversion:
// widen to 16 bits, multiply by alpha, divide by 255, keep the high byte.
func premultiply(value, alpha uint8) uint8 {
	wide := uint32(value)
	wide |= wide << 8
	wide *= uint32(alpha)
	wide /= 0xff
	return uint8(wide >> 8)
}

func premultipliedNRGBA(c color.NRGBA) rgba8 {
	return rgba8{
		r: premultiply(c.R, c.A),
		g: premultiply(c.G, c.A),
		b: premultiply(c.B, c.A),
		a: c.A,
	}
}

func opaquePixel(r, g, b uint8) rgba8 {
	return rgba8{r: r, g: g, b: b, a: 255}
}

func decodeFixture(t *testing.T, fixture []byte) DecodeResult {
	t.Helper()
	got, err := Decode(context.Background(), bytes.NewReader(fixture), int64(len(fixture)))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Pixels == nil {
		t.Fatal("Decode returned nil pixels without an error")
	}
	return got
}

func mustAdmit(t *testing.T, fixture []byte) Admission {
	t.Helper()
	admission, err := Admit(bytes.NewReader(fixture), int64(len(fixture)))
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	return admission
}

func checkPixel(t *testing.T, pixels *image.RGBA, x, y int, want rgba8) {
	t.Helper()
	offset := pixels.PixOffset(x, y)
	got := rgba8{
		r: pixels.Pix[offset+0],
		g: pixels.Pix[offset+1],
		b: pixels.Pix[offset+2],
		a: pixels.Pix[offset+3],
	}
	if got != want {
		t.Fatalf("pixel (%d,%d) = %#v, want %#v", x, y, got, want)
	}
}

func checkSurface(t *testing.T, got DecodeResult, kind Kind, width, height int) {
	t.Helper()
	if got.Kind != kind {
		t.Fatalf("kind = %v, want %v", got.Kind, kind)
	}
	if got.Width != width || got.Height != height {
		t.Fatalf("surface = %dx%d, want %dx%d", got.Width, got.Height, width, height)
	}
	wantRect := image.Rect(0, 0, width, height)
	if got.Pixels.Rect != wantRect {
		t.Fatalf("pixel rect = %v, want %v", got.Pixels.Rect, wantRect)
	}
	if got.Pixels.Stride != 4*width {
		t.Fatalf("pixel stride = %d, want %d", got.Pixels.Stride, 4*width)
	}
	if len(got.Pixels.Pix) != 4*width*height {
		t.Fatalf("pixel bytes = %d, want %d", len(got.Pixels.Pix), 4*width*height)
	}
	if offset := got.Pixels.PixOffset(1, 1); offset != 4*width+4 {
		t.Fatalf("PixOffset(1,1) = %d, want %d (tightly packed)", offset, 4*width+4)
	}
}

func TestDecodePNGExactPixels(t *testing.T) {
	grayscale := image.NewGray(image.Rect(0, 0, 3, 2))
	grayValues := [][]uint8{{0, 51, 102}, {153, 204, 255}}
	for y := range grayValues {
		for x, value := range grayValues[y] {
			grayscale.SetGray(x, y, color.Gray{Y: value})
		}
	}

	truecolor := image.NewRGBA(image.Rect(0, 0, 3, 2))
	trueValues := [][3]uint8{
		{10, 20, 30}, {40, 50, 60}, {70, 80, 90},
		{100, 110, 120}, {130, 140, 150}, {160, 170, 180},
	}
	for i, value := range trueValues {
		truecolor.SetRGBA(i%3, i/3, color.RGBA{
			R: value[0], G: value[1], B: value[2], A: 255,
		})
	}

	alphaValues := []color.NRGBA{
		{R: 200, G: 100, B: 50, A: 255},
		{R: 200, G: 100, B: 50, A: 128},
		{R: 200, G: 100, B: 50, A: 0},
		{R: 10, G: 250, B: 30, A: 64},
		{R: 255, G: 255, B: 255, A: 200},
		{R: 1, G: 2, B: 3, A: 1},
	}
	alpha := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for i, value := range alphaValues {
		alpha.SetNRGBA(i%3, i/3, value)
	}

	palette := color.Palette{
		color.NRGBA{R: 250, G: 10, B: 20, A: 255},
		color.NRGBA{R: 10, G: 200, B: 30, A: 128},
		color.NRGBA{R: 40, G: 50, B: 240, A: 0},
		color.NRGBA{R: 123, G: 45, B: 67, A: 77},
	}
	paletted := image.NewPaletted(image.Rect(0, 0, 3, 2), palette)
	paletteIndices := [][]uint8{{0, 1, 2}, {3, 1, 0}}
	for y := range paletteIndices {
		for x, index := range paletteIndices[y] {
			paletted.SetColorIndex(x, y, index)
		}
	}

	tests := []struct {
		name    string
		image   image.Image
		colorAt func(x, y int) rgba8
	}{
		{
			name:  "grayscale",
			image: grayscale,
			colorAt: func(x, y int) rgba8 {
				value := grayValues[y][x]
				return opaquePixel(value, value, value)
			},
		},
		{
			name:  "truecolor",
			image: truecolor,
			colorAt: func(x, y int) rgba8 {
				value := trueValues[y*3+x]
				return opaquePixel(value[0], value[1], value[2])
			},
		},
		{
			name:  "truecolor_alpha",
			image: alpha,
			colorAt: func(x, y int) rgba8 {
				return premultipliedNRGBA(alphaValues[y*3+x])
			},
		},
		{
			name:  "palette",
			image: paletted,
			colorAt: func(x, y int) rgba8 {
				return premultipliedNRGBA(palette[paletteIndices[y][x]].(color.NRGBA))
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := pngFixture(t, test.image)
			got := decodeFixture(t, fixture)
			checkSurface(t, got, KindPNG, 3, 2)
			if got.Orientation != 0 || got.SourceWidth != 3 || got.SourceHeight != 2 {
				t.Fatalf(
					"orientation/source = %d/%dx%d, want 0/3x2",
					got.Orientation, got.SourceWidth, got.SourceHeight,
				)
			}
			for y := 0; y < 2; y++ {
				for x := 0; x < 3; x++ {
					checkPixel(t, got.Pixels, x, y, test.colorAt(x, y))
				}
			}
		})
	}
}

func TestDecodePNGOrientationExact(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 3, 2))
	cellColors := []color.RGBA{
		{R: 10, G: 20, B: 30, A: 255},    // a
		{R: 40, G: 50, B: 60, A: 255},    // b
		{R: 70, G: 80, B: 90, A: 255},    // c
		{R: 100, G: 110, B: 120, A: 255}, // d
		{R: 130, G: 140, B: 150, A: 255}, // e
		{R: 160, G: 170, B: 180, A: 255}, // f
	}
	for i, value := range cellColors {
		source.SetRGBA(i%3, i/3, value)
	}
	base := pngFixture(t, source)

	// Independent hand-written destination grids. The source rows are abc/def.
	grids := map[uint8][][]int{
		1: {{0, 1, 2}, {3, 4, 5}},
		2: {{2, 1, 0}, {5, 4, 3}},
		3: {{5, 4, 3}, {2, 1, 0}},
		4: {{3, 4, 5}, {0, 1, 2}},
		5: {{0, 3}, {1, 4}, {2, 5}},
		6: {{3, 0}, {4, 1}, {5, 2}},
		7: {{5, 2}, {4, 1}, {3, 0}},
		8: {{2, 5}, {1, 4}, {0, 3}},
	}
	for orientation := uint8(1); orientation <= 8; orientation++ {
		t.Run("orientation_"+itoa(int(orientation)), func(t *testing.T) {
			fixture := insertPNGChunk(
				t,
				base,
				"eXIf",
				exifTIFF(t, binary.LittleEndian, uint16(orientation)),
				"IEND",
			)
			admission := mustAdmit(t, fixture)
			if admission.Orientation != orientation {
				t.Fatalf("admitted orientation = %d, want %d", admission.Orientation, orientation)
			}
			got := decodeFixture(t, fixture)
			width, height := 3, 2
			if orientation >= 5 {
				width, height = 2, 3
			}
			checkSurface(t, got, KindPNG, width, height)
			if got.Orientation != orientation ||
				got.SourceWidth != 3 || got.SourceHeight != 2 {
				t.Fatalf(
					"orientation/source = %d/%dx%d, want %d/3x2",
					got.Orientation, got.SourceWidth, got.SourceHeight, orientation,
				)
			}
			want := grids[orientation]
			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					value := cellColors[want[y][x]]
					checkPixel(t, got.Pixels, x, y, opaquePixel(value.R, value.G, value.B))
				}
			}
		})
	}
}

func regionMean(pixels *image.RGBA, x0, y0, size int) (red, green, blue float64) {
	var sums [3]float64
	for y := y0; y < y0+size; y++ {
		for x := x0; x < x0+size; x++ {
			offset := pixels.PixOffset(x, y)
			sums[0] += float64(pixels.Pix[offset+0])
			sums[1] += float64(pixels.Pix[offset+1])
			sums[2] += float64(pixels.Pix[offset+2])
		}
	}
	count := float64(size * size)
	return sums[0] / count, sums[1] / count, sums[2] / count
}

func checkRegion(t *testing.T, pixels *image.RGBA, x0, y0, size int, want [3]uint8, tolerance float64) {
	t.Helper()
	red, green, blue := regionMean(pixels, x0, y0, size)
	for channel, got := range []float64{red, green, blue} {
		delta := math.Abs(got - float64(want[channel]))
		if delta > tolerance {
			t.Fatalf(
				"region (%d,%d) channel %d mean = %.1f, want %d +/- %.0f",
				x0, y0, channel, got, want[channel], tolerance,
			)
		}
	}
}

func TestDecodeJPEGKnownRegions(t *testing.T) {
	t.Run("grayscale", func(t *testing.T) {
		values := [4]uint8{30, 90, 150, 220}
		img := image.NewGray(image.Rect(0, 0, 16, 16))
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				img.SetGray(x, y, color.Gray{Y: values[(y/8)*2+x/8]})
			}
		}
		fixture := jpegFixture(t, img, &jpeg.Options{Quality: 90})
		got := decodeFixture(t, fixture)
		checkSurface(t, got, KindJPEG, 16, 16)
		if got.Orientation != 0 || got.SourceWidth != 16 || got.SourceHeight != 16 {
			t.Fatalf(
				"orientation/source = %d/%dx%d",
				got.Orientation, got.SourceWidth, got.SourceHeight,
			)
		}
		for quadrantY := 0; quadrantY < 2; quadrantY++ {
			for quadrantX := 0; quadrantX < 2; quadrantX++ {
				value := values[quadrantY*2+quadrantX]
				checkRegion(
					t,
					got.Pixels,
					quadrantX*8+2,
					quadrantY*8+2,
					4,
					[3]uint8{value, value, value},
					10,
				)
			}
		}
	})

	t.Run("three_component", func(t *testing.T) {
		quadrants := [4]color.NRGBA{
			{R: 210, G: 40, B: 40, A: 255},
			{R: 40, G: 190, B: 60, A: 255},
			{R: 50, G: 60, B: 210, A: 255},
			{R: 235, G: 225, B: 215, A: 255},
		}
		img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				img.SetNRGBA(x, y, quadrants[(y/8)*2+x/8])
			}
		}
		fixture := jpegFixture(t, img, &jpeg.Options{Quality: 95})
		got := decodeFixture(t, fixture)
		checkSurface(t, got, KindJPEG, 16, 16)
		for quadrantY := 0; quadrantY < 2; quadrantY++ {
			for quadrantX := 0; quadrantX < 2; quadrantX++ {
				value := quadrants[quadrantY*2+quadrantX]
				checkRegion(
					t,
					got.Pixels,
					quadrantX*8+2,
					quadrantY*8+2,
					4,
					[3]uint8{value.R, value.G, value.B},
					20,
				)
			}
		}
	})
}

func TestDecodeJPEGEXIFOrientation(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 32; x++ {
			value := color.NRGBA{R: 220, G: 30, B: 30, A: 255}
			if x >= 16 {
				value = color.NRGBA{R: 30, G: 40, B: 220, A: 255}
			}
			img.SetNRGBA(x, y, value)
		}
	}
	base := jpegFixture(t, img, &jpeg.Options{Quality: 95})
	payload := append([]byte("Exif\x00\x00"), exifTIFF(t, binary.LittleEndian, 6)...)
	fixture := insertJPEGSegment(t, base, 0xe1, payload, 0xda)

	got := decodeFixture(t, fixture)
	checkSurface(t, got, KindJPEG, 16, 32)
	if got.Orientation != 6 || got.SourceWidth != 32 || got.SourceHeight != 16 {
		t.Fatalf(
			"orientation/source = %d/%dx%d, want 6/32x16",
			got.Orientation, got.SourceWidth, got.SourceHeight,
		)
	}
	// Rotating the left-red/right-blue source 90 degrees clockwise puts red on
	// top and blue below.
	checkRegion(t, got.Pixels, 6, 6, 4, [3]uint8{220, 30, 30}, 25)
	checkRegion(t, got.Pixels, 6, 22, 4, [3]uint8{30, 40, 220}, 25)
}

func TestDecodeRejectsCorruptAdmittedPayloads(t *testing.T) {
	t.Run("png_corrupt_idat", func(t *testing.T) {
		base := pngFixture(t, opaqueRGBA(16, 16))
		_, end := findPNGChunk(t, base, "IDAT")
		fixture := append([]byte(nil), base...)
		fixture[end-5] ^= 0xff // Last compressed byte before the chunk CRC.
		if _, err := Admit(bytes.NewReader(fixture), int64(len(fixture))); err != nil {
			t.Fatalf("corrupt IDAT fixture must still pass header admission: %v", err)
		}
		got, err := Decode(context.Background(), bytes.NewReader(fixture), int64(len(fixture)))
		assertDecodeStage(t, err, got, DecodeStageCodec)
	})

	t.Run("png_bad_idat_crc", func(t *testing.T) {
		base := pngFixture(t, opaqueRGBA(16, 16))
		_, end := findPNGChunk(t, base, "IDAT")
		fixture := append([]byte(nil), base...)
		fixture[end-1] ^= 0xff // Chunk CRC byte.
		if _, err := Admit(bytes.NewReader(fixture), int64(len(fixture))); err != nil {
			t.Fatalf("bad CRC fixture must still pass header admission: %v", err)
		}
		got, err := Decode(context.Background(), bytes.NewReader(fixture), int64(len(fixture)))
		assertDecodeStage(t, err, got, DecodeStageCodec)
	})

	t.Run("jpeg_truncated_after_sos", func(t *testing.T) {
		base := jpegFixture(t, opaqueRGBA(16, 16), &jpeg.Options{Quality: 90})
		sos := findJPEGMarker(t, base, 0xda)
		segmentLength := int(binary.BigEndian.Uint16(base[sos+2 : sos+4]))
		cut := sos + 2 + segmentLength + 1 // Header-complete, one entropy byte.
		fixture := base[:cut]
		if _, err := Admit(bytes.NewReader(fixture), int64(len(fixture))); err != nil {
			t.Fatalf("truncated JPEG must still pass header admission at SOS: %v", err)
		}
		got, err := Decode(context.Background(), bytes.NewReader(fixture), int64(len(fixture)))
		if err == nil {
			t.Fatal("truncated JPEG decoded successfully")
		}
		if got.Pixels != nil {
			t.Fatal("truncated JPEG published pixels")
		}
		var decodeErr *DecodeError
		if !errors.As(err, &decodeErr) {
			t.Fatalf("truncated JPEG error = %T: %v, want *DecodeError", err, err)
		}
		if decodeErr.Stage != DecodeStageRead && decodeErr.Stage != DecodeStageCodec {
			t.Fatalf("truncated JPEG stage = %v, want read or codec", decodeErr.Stage)
		}
	})
}

func assertDecodeStage(t *testing.T, err error, got DecodeResult, stage DecodeStage) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected decode failure at stage %v, got success", stage)
	}
	if got.Pixels != nil {
		t.Fatal("failed decode published pixels")
	}
	var decodeErr *DecodeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("error = %T: %v, want *DecodeError", err, err)
	}
	if decodeErr.Stage != stage {
		t.Fatalf("stage = %v, want %v (%v)", decodeErr.Stage, stage, err)
	}
}

func TestDecodeRunsAdmissionInternally(t *testing.T) {
	_, err := Decode(context.Background(), bytes.NewReader([]byte("definitely not an image")), 23)
	rejectWith(t, RejectionUnsupported).Test(err)

	_, err = Decode(context.Background(), nil, 10)
	rejectWith(t, RejectionRead).Test(err)

	fixture := pngFixture(t, image.NewGray(image.Rect(0, 0, 1, 1)))
	_, err = Decode(context.Background(), bytes.NewReader(fixture), MaxCompressedBytes+1)
	rejectWith(t, RejectionCompressed).Test(err)

	_, err = Decode(nil, bytes.NewReader(fixture), int64(len(fixture)))
	var decodeErr *DecodeError
	if !errors.As(err, &decodeErr) || decodeErr.Stage != DecodeStageSetup {
		t.Fatalf("nil context error = %v, want setup decode error", err)
	}
}

// countingReaderAt counts reads and can fail every read after a fixed number.
type countingReaderAt struct {
	data      []byte
	calls     int
	failAfter int // Negative disables the failure.
	err       error
}

func (r *countingReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if r.failAfter >= 0 && r.calls >= r.failAfter {
		r.calls++
		return 0, r.err
	}
	r.calls++
	return bytes.NewReader(r.data).ReadAt(p, off)
}

func TestDecodeReaderFailures(t *testing.T) {
	fixture := pngFixture(t, opaqueRGBA(8, 8))

	t.Run("before_admission", func(t *testing.T) {
		failing := readAtFunc(func(p []byte, off int64) (int, error) {
			return 0, errors.New("reader is unavailable")
		})
		_, err := Decode(context.Background(), failing, int64(len(fixture)))
		rejectWith(t, RejectionRead).Test(err)
	})

	t.Run("during_codec", func(t *testing.T) {
		counting := &countingReaderAt{data: fixture, failAfter: -1}
		if _, err := Admit(counting, int64(len(fixture))); err != nil {
			t.Fatalf("Admit: %v", err)
		}
		admissionReads := counting.calls
		if admissionReads < 2 {
			t.Fatalf("admission read count = %d, want at least 2", admissionReads)
		}
		counting.calls = 0
		counting.failAfter = admissionReads
		injected := errors.New("reader failed during codec")
		counting.err = injected

		got, err := Decode(context.Background(), counting, int64(len(fixture)))
		assertDecodeStage(t, err, got, DecodeStageRead)
		if !errors.Is(err, injected) {
			t.Fatalf("error = %v, want wrapped %v", err, injected)
		}
	})
}

// cancelAfterRead cancels the context after a fixed number of successful reads.
type cancelAfterRead struct {
	data   []byte
	cancel context.CancelFunc
	after  int
	calls  int
}

func (r *cancelAfterRead) ReadAt(p []byte, off int64) (int, error) {
	r.calls++
	n, err := bytes.NewReader(r.data).ReadAt(p, off)
	if r.calls == r.after {
		r.cancel()
	}
	return n, err
}

// scriptedImage is a codec stand-in for seam tests. At is never expected to run
// when bounded checks must reject a surface.
type scriptedImage struct {
	bounds image.Rectangle
	at     func(x, y int) color.Color
}

func (s *scriptedImage) ColorModel() color.Model {
	return color.RGBAModel
}

func (s *scriptedImage) Bounds() image.Rectangle {
	return s.bounds
}

func (s *scriptedImage) At(x, y int) color.Color {
	return s.at(x, y)
}

func assertCanceledDecode(t *testing.T, err error, got DecodeResult) {
	t.Helper()
	if err == nil {
		t.Fatal("expected cancellation, got success")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	var decodeErr *DecodeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("error = %T: %v, want *DecodeError", err, err)
	}
	if decodeErr.Stage != DecodeStageCanceled {
		t.Fatalf("stage = %v, want canceled (%v)", decodeErr.Stage, err)
	}
	if got.Pixels != nil {
		t.Fatal("canceled decode published pixels")
	}
}

func TestDecodeCancellation(t *testing.T) {
	fixture := pngFixture(t, opaqueRGBA(8, 8))

	t.Run("initially", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		counting := &countingReaderAt{data: fixture, failAfter: -1}
		got, err := Decode(ctx, counting, int64(len(fixture)))
		assertCanceledDecode(t, err, got)
		if counting.calls != 0 {
			t.Fatalf("reader was touched %d times before cancellation", counting.calls)
		}
	})

	t.Run("during_reads", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		reader := &cancelAfterRead{data: fixture, cancel: cancel, after: 1}
		got, err := Decode(ctx, reader, int64(len(fixture)))
		assertCanceledDecode(t, err, got)
		if reader.calls != 1 {
			t.Fatalf("underlying reads = %d, want 1 (cancel checked before the next read)", reader.calls)
		}
	})

	seamFixture := pngFixture(t, image.NewGray(image.Rect(0, 0, 3, 4)))
	seamAdmission := mustAdmit(t, seamFixture)

	t.Run("between_conversion_rows", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		atCalls := 0
		source := &scriptedImage{
			bounds: image.Rect(0, 0, 3, 4),
			at: func(x, y int) color.Color {
				atCalls++
				if atCalls == 1 {
					cancel()
				}
				return color.Gray{Y: 10}
			},
		}
		got, err := decodeAdmitted(
			ctx,
			seamAdmission,
			bytes.NewReader(seamFixture),
			int64(len(seamFixture)),
			func(io.Reader) (image.Image, error) { return source, nil },
		)
		assertCanceledDecode(t, err, got)
		if atCalls != 3 {
			t.Fatalf("At calls = %d, want 3 (stop at the first row boundary)", atCalls)
		}
	})

	t.Run("during_final_row", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		atCalls := 0
		source := &scriptedImage{
			bounds: image.Rect(0, 0, 3, 4),
			at: func(x, y int) color.Color {
				atCalls++
				if atCalls == 12 {
					cancel()
				}
				return color.Gray{Y: 10}
			},
		}
		got, err := decodeAdmitted(
			ctx,
			seamAdmission,
			bytes.NewReader(seamFixture),
			int64(len(seamFixture)),
			func(io.Reader) (image.Image, error) { return source, nil },
		)
		assertCanceledDecode(t, err, got)
		if atCalls != 12 {
			t.Fatalf("At calls = %d, want 12 (final publication check)", atCalls)
		}
	})

	t.Run("after_codec", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		atCalls := 0
		source := &scriptedImage{
			bounds: image.Rect(0, 0, 3, 4),
			at: func(x, y int) color.Color {
				atCalls++
				return color.Gray{Y: 10}
			},
		}
		got, err := decodeAdmitted(
			ctx,
			seamAdmission,
			bytes.NewReader(seamFixture),
			int64(len(seamFixture)),
			func(io.Reader) (image.Image, error) {
				cancel()
				return source, nil
			},
		)
		assertCanceledDecode(t, err, got)
		if atCalls != 0 {
			t.Fatalf("At calls = %d, want 0 (canceled after the codec)", atCalls)
		}
	})
}

func TestDecodeRejectsDecodedBoundsBeforePublication(t *testing.T) {
	fixture := pngFixture(t, image.NewGray(image.Rect(0, 0, 3, 2)))
	admission := mustAdmit(t, fixture)

	tests := []struct {
		name   string
		adm    Admission
		bounds image.Rectangle
	}{
		{
			name:   "dimension_mismatch",
			adm:    admission,
			bounds: image.Rect(0, 0, 2, 2),
		},
		{
			name: "oversize_side",
			adm: Admission{
				Kind: KindPNG, Width: 9000, Height: 1, Pixels: 9000, Orientation: 1,
			},
			bounds: image.Rect(0, 0, 9000, 1),
		},
		{
			name: "too_many_pixels",
			adm: Admission{
				Kind: KindPNG, Width: 8192, Height: 1200, Pixels: 8192 * 1200, Orientation: 1,
			},
			bounds: image.Rect(0, 0, 8192, 1200),
		},
		{
			name:   "empty_surface",
			adm:    Admission{Kind: KindPNG},
			bounds: image.Rect(0, 0, 0, 0),
		},
		{
			name: "pixel_count_disagrees",
			adm: Admission{
				Kind: KindPNG, Width: 3, Height: 2, Pixels: 7, Orientation: 1,
			},
			bounds: image.Rect(0, 0, 3, 2),
		},
		{
			name: "orientation_outside_range",
			adm: Admission{
				Kind: KindPNG, Width: 3, Height: 2, Pixels: 6, Orientation: 9,
			},
			bounds: image.Rect(0, 0, 3, 2),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			atCalls := 0
			source := &scriptedImage{
				bounds: test.bounds,
				at: func(x, y int) color.Color {
					atCalls++
					t.Fatalf("At(%d,%d) ran before bounds validation", x, y)
					return color.Gray{}
				},
			}
			got, err := decodeAdmitted(
				context.Background(),
				test.adm,
				bytes.NewReader(fixture),
				int64(len(fixture)),
				func(io.Reader) (image.Image, error) { return source, nil },
			)
			assertDecodeStage(t, err, got, DecodeStageBounds)
			if atCalls != 0 {
				t.Fatalf("At calls = %d, want 0", atCalls)
			}
		})
	}
}

func TestDecodeResultRetainsNoReader(t *testing.T) {
	readerAt := reflect.TypeOf((*io.ReaderAt)(nil)).Elem()
	reader := reflect.TypeOf((*io.Reader)(nil)).Elem()
	closer := reflect.TypeOf((*io.Closer)(nil)).Elem()
	resultType := reflect.TypeOf(DecodeResult{})
	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)
		for _, iface := range []reflect.Type{readerAt, reader, closer} {
			if field.Type.Implements(iface) {
				t.Fatalf("DecodeResult.%s implements %v; the result must not retain the input", field.Name, iface)
			}
		}
	}

	data := pngFixture(t, opaqueRGBA(4, 3))
	got, err := Decode(context.Background(), bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	snapshot := append([]byte(nil), got.Pixels.Pix...)
	for i := range data {
		data[i] = 0x00
	}
	if !bytes.Equal(snapshot, got.Pixels.Pix) {
		t.Fatal("decoded pixels changed when the source bytes were overwritten")
	}
}
