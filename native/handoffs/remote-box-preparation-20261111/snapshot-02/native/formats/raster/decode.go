package raster

import (
	"context"
	"image"
	"image/jpeg"
	"image/png"
	"io"
)

// DecodeResult is one decoded raster surface plus the small metadata needed to
// present it. Pixels is owned by the caller, is always origin zero and tightly
// packed (four premultiplied RGBA bytes per pixel), and never aliases the
// input reader or any intermediate codec surface.
type DecodeResult struct {
	Kind         Kind
	Width        int // final width after EXIF orientation
	Height       int // final height after EXIF orientation
	SourceWidth  int // codec width before orientation
	SourceHeight int // codec height before orientation
	Orientation  uint8
	Pixels       *image.RGBA
}

// DecodeStage identifies the phase of a decode failure after admission.
type DecodeStage uint8

const (
	// DecodeStageSetup is an invalid call, such as a nil context.
	DecodeStageSetup DecodeStage = iota + 1
	// DecodeStageRead is a failure or error return from the ReaderAt while the
	// codec was reading.
	DecodeStageRead
	// DecodeStageCodec is a codec rejection of admitted but corrupt or
	// truncated bytes.
	DecodeStageCodec
	// DecodeStageBounds is a decoded surface that disagrees with admission or
	// exceeds the independent hard limits.
	DecodeStageBounds
	// DecodeStageCanceled is an honored context cancellation.
	DecodeStageCanceled
)

// String returns a stable, user-readable stage name.
func (s DecodeStage) String() string {
	switch s {
	case DecodeStageSetup:
		return "setup"
	case DecodeStageRead:
		return "read"
	case DecodeStageCodec:
		return "codec"
	case DecodeStageBounds:
		return "decoded bounds"
	case DecodeStageCanceled:
		return "canceled"
	default:
		return "unknown stage"
	}
}

// DecodeError is an explicit failure after admission succeeded. Admission
// rejections are returned unchanged as *RejectionError.
type DecodeError struct {
	Stage  DecodeStage
	Detail string
	Err    error
}

// Error implements error with the stage, concrete detail, and cause.
func (e *DecodeError) Error() string {
	text := "raster decode: " + e.Stage.String()
	if e.Detail != "" {
		text += ": " + e.Detail
	}
	if e.Err != nil {
		text += ": " + e.Err.Error()
	}
	return text
}

// Unwrap exposes the underlying reader, codec, or context error.
func (e *DecodeError) Unwrap() error {
	return e.Err
}

// Decode admits and then decodes one PNG or JPEG from a caller-owned immutable
// reader. Admission always runs here; a caller cannot supply or forge an
// Admission estimate. The same ReaderAt is read through a context-aware bounded
// section reader, with no full-input clone, no filesystem access, no background
// goroutine, and no retained reader in the result.
//
// The caller must keep r immutable for the duration of the call; the host
// supplies a BinarySnapshot. On success the result holds exactly one owned
// tightly packed origin-zero premultiplied *image.RGBA, suitable for
// ui.NewBitmap without another full-size conversion. On failure or
// cancellation it holds nil pixels.
//
// Cancellation is checked before every read, before and after codec execution,
// between conversion rows, and once more before the result is published. Codec
// CPU work between reads cannot be preempted, so cancellation can lag behind a
// slow codec call; the returned error is still an explicit cancellation and no
// pixels are published.
func Decode(ctx context.Context, r io.ReaderAt, declaredSize int64) (DecodeResult, error) {
	if ctx == nil {
		return DecodeResult{}, &DecodeError{
			Stage:  DecodeStageSetup,
			Detail: "nil context",
		}
	}
	if r == nil {
		return DecodeResult{}, &RejectionError{Reason: RejectionRead, Detail: "ReaderAt is nil"}
	}
	if err := ctx.Err(); err != nil {
		return DecodeResult{}, canceledDecode("before admission", err)
	}

	guarded := &contextReaderAt{ctx: ctx, reader: r}
	admission, err := Admit(guarded, declaredSize)
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return DecodeResult{}, canceledDecode("during admission", contextErr)
		}
		return DecodeResult{}, err
	}

	var codec decodeCodec
	switch admission.Kind {
	case KindPNG:
		codec = decodePNG
	case KindJPEG:
		codec = decodeJPEG
	default:
		return DecodeResult{}, &DecodeError{
			Stage:  DecodeStageSetup,
			Detail: "admission returned an unknown kind",
		}
	}
	return decodeAdmitted(ctx, admission, guarded, declaredSize, codec)
}

// decodeCodec is one explicit codec entry point. Generic image.Decode and any
// pluggable codec registry stay out of this package.
type decodeCodec func(r io.Reader) (image.Image, error)

func decodePNG(r io.Reader) (image.Image, error) {
	return png.Decode(r)
}

func decodeJPEG(r io.Reader) (image.Image, error) {
	return jpeg.Decode(r)
}

// contextReaderAt checks ctx before every actual read and records the first
// reader error so a codec failure can be attributed to the input instead of the
// codec. It is used by one decode call on one goroutine.
type contextReaderAt struct {
	ctx     context.Context
	reader  io.ReaderAt
	lastErr error
}

// ReadAt implements io.ReaderAt with a cancellation check before each read.
func (r *contextReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if err := r.ctx.Err(); err != nil {
		r.lastErr = err
		return 0, err
	}
	n, err := r.reader.ReadAt(p, off)
	if err != nil {
		r.lastErr = err
	}
	return n, err
}

// decodeAdmitted runs one explicit codec over the admitted byte range and
// converts its result directly into the final oriented surface. The codec and
// reader are parameters so tests can exercise the real bounds and conversion
// logic with controlled codec output.
func decodeAdmitted(
	ctx context.Context,
	admission Admission,
	reader io.ReaderAt,
	size int64,
	codec decodeCodec,
) (DecodeResult, error) {
	if err := ctx.Err(); err != nil {
		return DecodeResult{}, canceledDecode("before codec", err)
	}
	source, err := codec(io.NewSectionReader(reader, 0, size))
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return DecodeResult{}, canceledDecode("during codec", contextErr)
		}
		if readErr, ok := contextReadFailure(reader); ok {
			return DecodeResult{}, &DecodeError{
				Stage:  DecodeStageRead,
				Detail: "codec read",
				Err:    readErr,
			}
		}
		return DecodeResult{}, &DecodeError{
			Stage:  DecodeStageCodec,
			Detail: "admitted bytes failed codec validation",
			Err:    err,
		}
	}
	if err := ctx.Err(); err != nil {
		return DecodeResult{}, canceledDecode("after codec", err)
	}
	if source == nil {
		return DecodeResult{}, &DecodeError{
			Stage:  DecodeStageBounds,
			Detail: "codec returned no image",
		}
	}
	if err := validateDecodedBounds(admission, source.Bounds()); err != nil {
		return DecodeResult{}, err
	}

	width, height := admission.Width, admission.Height
	if admission.Orientation >= 5 {
		width, height = height, width
	}
	pixels := image.NewRGBA(image.Rect(0, 0, width, height))
	if err := convertOriented(ctx, source, admission.Orientation, pixels); err != nil {
		return DecodeResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return DecodeResult{}, canceledDecode("before publication", err)
	}

	return DecodeResult{
		Kind:         admission.Kind,
		Width:        width,
		Height:       height,
		SourceWidth:  admission.Width,
		SourceHeight: admission.Height,
		Orientation:  admission.Orientation,
		Pixels:       pixels,
	}, nil
}

// validateDecodedBounds independently rechecks codec geometry before any final
// surface is allocated. Admission already bounded the header, but the decoder
// must not trust a codec or a changed header to agree with it.
func validateDecodedBounds(admission Admission, bounds image.Rectangle) error {
	width, height := bounds.Dx(), bounds.Dy()
	if width < 1 || height < 1 {
		return &DecodeError{
			Stage:  DecodeStageBounds,
			Detail: "decoded image is empty",
		}
	}
	if width != admission.Width || height != admission.Height {
		return &DecodeError{
			Stage:  DecodeStageBounds,
			Detail: "decoded dimensions " + itoa(width) + "x" + itoa(height) + " disagree with admitted " + itoa(admission.Width) + "x" + itoa(admission.Height),
		}
	}
	if width > MaxDimension || height > MaxDimension {
		return &DecodeError{
			Stage:  DecodeStageBounds,
			Detail: "decoded dimensions exceed 8192",
		}
	}
	if admission.Orientation > 8 {
		return &DecodeError{
			Stage:  DecodeStageBounds,
			Detail: "admission orientation is outside 0..8; refusing a silent identity fallback",
		}
	}
	pixels := int64(width) * int64(height)
	if pixels > MaxPixels || pixels != admission.Pixels {
		return &DecodeError{
			Stage:  DecodeStageBounds,
			Detail: "decoded pixel count exceeds or disagrees with admission",
		}
	}
	return nil
}

// convertOriented writes source pixels straight into the final surface with the
// requested EXIF orientation. No intermediate full-size image is allocated.
// Forward source-to-destination mappings, where the source is w x h:
//
//	1 (x, y)             5 (y, x)              output h x w
//	2 (w-1-x, y)         6 (h-1-y, x)          output h x w
//	3 (w-1-x, h-1-y)     7 (h-1-y, w-1-x)      output h x w
//	4 (x, h-1-y)         8 (y, w-1-x)          output h x w
//
// color.Color.RGBA already returns premultiplied components, so truncating each
// 16-bit value to its high byte yields the premultiplied destination bytes that
// image.RGBA requires.
func convertOriented(ctx context.Context, source image.Image, orientation uint8, pixels *image.RGBA) error {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	for y := 0; y < height; y++ {
		if err := ctx.Err(); err != nil {
			return canceledDecode("between conversion rows", err)
		}
		for x := 0; x < width; x++ {
			red, green, blue, alpha := source.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			destinationX, destinationY := orientedDestination(orientation, x, y, width, height)
			offset := pixels.PixOffset(destinationX, destinationY)
			pixels.Pix[offset+0] = uint8(red >> 8)
			pixels.Pix[offset+1] = uint8(green >> 8)
			pixels.Pix[offset+2] = uint8(blue >> 8)
			pixels.Pix[offset+3] = uint8(alpha >> 8)
		}
	}
	return nil
}

// orientedDestination maps one source coordinate to its final position. The
// values outside 1..8 and an absent tag (zero) are the identity.
func orientedDestination(orientation uint8, x, y, width, height int) (int, int) {
	switch orientation {
	case 2:
		return width - 1 - x, y
	case 3:
		return width - 1 - x, height - 1 - y
	case 4:
		return x, height - 1 - y
	case 5:
		return y, x
	case 6:
		return height - 1 - y, x
	case 7:
		return height - 1 - y, width - 1 - x
	case 8:
		return y, width - 1 - x
	default:
		return x, y
	}
}

// contextReadFailure reports the first reader error seen by the guarded reader.
func contextReadFailure(reader io.ReaderAt) (error, bool) {
	guarded, ok := reader.(*contextReaderAt)
	if !ok || guarded.lastErr == nil {
		return nil, false
	}
	return guarded.lastErr, true
}

func canceledDecode(where string, err error) error {
	return &DecodeError{
		Stage:  DecodeStageCanceled,
		Detail: "canceled " + where,
		Err:    err,
	}
}
