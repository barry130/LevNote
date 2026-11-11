// Package raster provides bounded, local PNG/JPEG admission checks.
//
// Admission is a header preflight. It does not replace codec validation and it
// does not guarantee physical memory use.
package raster

import "io"

// Kind identifies an admitted raster container.
type Kind uint8

const (
	KindUnknown Kind = iota
	KindPNG
	KindJPEG
)

// String returns a stable, user-readable container name.
func (k Kind) String() string {
	switch k {
	case KindPNG:
		return "PNG"
	case KindJPEG:
		return "JPEG"
	default:
		return "unknown"
	}
}

// Subset identifies the admitted variant of a container.
type Subset uint8

const (
	// SubsetPNG8 is a noninterlaced PNG using one of the legal at-most-8-bit
	// color/depth pairs.
	SubsetPNG8 Subset = iota + 1
	// SubsetJPEGBaseline is 8-bit baseline SOF0 JPEG with one grayscale or
	// standard three-component scan.
	SubsetJPEGBaseline
)

// String returns a stable, user-readable subset name.
func (s Subset) String() string {
	switch s {
	case SubsetPNG8:
		return "PNG <=8-bit noninterlaced"
	case SubsetJPEGBaseline:
		return "JPEG 8-bit baseline SOF0"
	default:
		return "unknown"
	}
}

// RejectionReason classifies why admission failed.
type RejectionReason uint8

const (
	// RejectionRead is a failure or error return from ReaderAt.
	RejectionRead RejectionReason = iota + 1
	// RejectionTruncated means the declared size or a segment ends early.
	RejectionTruncated
	// RejectionInvalid means a header or marker violates its format.
	RejectionInvalid
	// RejectionUnsupported means the format feature is outside this subset.
	RejectionUnsupported
	// RejectionMetadata means bounded EXIF metadata is malformed or ambiguous.
	RejectionMetadata
	// RejectionDimensions means a dimension or pixel-count limit failed.
	RejectionDimensions
	// RejectionCompressed means the compressed input limit failed.
	RejectionCompressed
	// RejectionMipBudget means the base+MIP RGBA byte limit failed.
	RejectionMipBudget
	// RejectionDecodeBudget means the modeled decode payload limit failed.
	RejectionDecodeBudget
)

// String returns a stable, user-readable rejection name.
func (r RejectionReason) String() string {
	switch r {
	case RejectionRead:
		return "read failed"
	case RejectionTruncated:
		return "input truncated"
	case RejectionInvalid:
		return "invalid header"
	case RejectionUnsupported:
		return "unsupported raster subset"
	case RejectionMetadata:
		return "malformed or ambiguous metadata"
	case RejectionDimensions:
		return "dimensions exceed limits"
	case RejectionCompressed:
		return "compressed input exceeds limit"
	case RejectionMipBudget:
		return "RGBA mip chain exceeds limit"
	case RejectionDecodeBudget:
		return "modeled decode payload exceeds limit"
	default:
		return "unknown rejection"
	}
}

// RejectionError is an explicit admission failure.
type RejectionError struct {
	Reason RejectionReason
	Detail string
}

// Error implements error with the reason and concrete detail.
func (e *RejectionError) Error() string {
	return "raster admission: " + e.Reason.String() + ": " + e.Detail
}

// Admission contains small header metadata and payload budgets.
type Admission struct {
	Kind               Kind
	Subset             Subset
	Width              int
	Height             int
	Pixels             int64
	Orientation        uint8
	CompressedBytes    int64
	BaseRGBABytes      int64
	MipChainBytes      int64
	MipLevels          int
	DecodePayloadBytes int64
}

// Exact admission limits.
const (
	MaxCompressedBytes = int64(16 * 1024 * 1024)
	MaxDimension       = 8192
	MaxPixels          = int64(8_388_608)
	MaxMipChainBytes   = int64(48 * 1024 * 1024)
	MaxDecodePayload   = int64(128 * 1024 * 1024)
)

// Admit performs a bounded, pure PNG/JPEG header admission check.
func Admit(r io.ReaderAt, size int64) (Admission, error) {
	if r == nil {
		return Admission{}, &RejectionError{RejectionRead, "ReaderAt is nil"}
	}
	if size <= 0 {
		return Admission{}, &RejectionError{RejectionTruncated, "declared size must be positive"}
	}
	if size > MaxCompressedBytes {
		return Admission{}, &RejectionError{
			RejectionCompressed,
			"declared size exceeds 16MiB",
		}
	}

	if size < int64(len(pngSignature)) {
		return Admission{}, truncatedReject("input is shorter than a PNG/JPEG signature")
	}

	var signature [8]byte
	n, err := r.ReadAt(signature[:], 0)
	if err != nil {
		return Admission{}, readReject("signature", err)
	}
	if n != len(signature) {
		return Admission{}, &RejectionError{RejectionTruncated, "short signature"}
	}
	switch {
	case string(signature[:]) == "\x89PNG\r\n\x1a\n":
		return admitPNG(r, size)
	case signature[0] == 0xff && signature[1] == 0xd8:
		return admitJPEG(r, size)
	default:
		return Admission{}, &RejectionError{RejectionUnsupported, "expected PNG or JPEG signature"}
	}
}

func readReject(where string, err error) error {
	return &RejectionError{RejectionRead, where + ": " + err.Error()}
}

func truncatedReject(detail string) error {
	return &RejectionError{RejectionTruncated, detail}
}

func invalidReject(detail string) error {
	return &RejectionError{RejectionInvalid, detail}
}

func unsupportedReject(detail string) error {
	return &RejectionError{RejectionUnsupported, detail}
}

func metadataReject(detail string) error {
	return &RejectionError{RejectionMetadata, detail}
}
