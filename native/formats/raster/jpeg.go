package raster

import (
	"encoding/binary"
	"io"
)

const (
	jpegStreamScratchBytes = int64(4096)
	maxJPEGSegments        = int64(4096)
)

func admitJPEG(r io.ReaderAt, size int64) (Admission, error) {
	offset := int64(2)
	var (
		orientation uint8
		seenEXIF    bool
		seenSOF     bool
		frame       jpegSOF
	)
	segmentCount := int64(0)

	for {
		if offset+2 > size {
			return Admission{}, truncatedReject("JPEG marker header exceeds declared size")
		}
		markerHeader, err := readBounded(r, size, offset, 2)
		if err != nil {
			return Admission{}, err
		}
		if markerHeader[0] != 0xff {
			return Admission{}, invalidReject("JPEG marker must start with 0xff")
		}
		marker := markerHeader[1]
		offset += 2
		fillCount := int64(0)
		for marker == 0xff {
			if offset >= size {
				return Admission{}, truncatedReject("JPEG fill bytes run to end of input")
			}
			next, fillErr := readBounded(r, size, offset, 1)
			if fillErr != nil {
				return Admission{}, fillErr
			}
			marker = next[0]
			offset++
			fillCount++
			if fillCount > maxJPEGSegments {
				return Admission{}, invalidReject("JPEG has too many fill bytes")
			}
		}

		if marker == 0x00 {
			return Admission{}, invalidReject("byte-stuffed data appears before JPEG SOS")
		}
		if marker >= 0xd0 && marker <= 0xd7 {
			return Admission{}, invalidReject("restart marker appears before JPEG SOS")
		}
		if marker == 0xd9 {
			return Admission{}, invalidReject("JPEG EOI appears before a frame")
		}
		if marker == 0xd8 {
			return Admission{}, invalidReject("duplicate JPEG SOI")
		}

		if offset+2 > size {
			return Admission{}, truncatedReject("JPEG segment length is truncated")
		}
		segmentCount++
		if segmentCount > maxJPEGSegments {
			return Admission{}, invalidReject("JPEG has too many header segments")
		}
		lengthBytes, err := readBounded(r, size, offset, 2)
		if err != nil {
			return Admission{}, err
		}
		segmentLength := int64(binary.BigEndian.Uint16(lengthBytes))
		if segmentLength < 2 {
			return Admission{}, invalidReject("JPEG segment length is smaller than its two length bytes")
		}
		segmentBytes := segmentLength - 2
		dataOffset := offset + 2
		segmentEnd, err := checkedAdd(dataOffset, segmentBytes)
		if err != nil {
			return Admission{}, err
		}
		if segmentEnd > size {
			return Admission{}, truncatedReject("JPEG segment exceeds declared size")
		}

		switch marker {
		case 0xc0: // SOF0, baseline sequential.
			if seenSOF {
				return Admission{}, invalidReject("JPEG has more than one SOF marker")
			}
			seenSOF = true
			sofBytes, readErr := readBounded(r, size, dataOffset, int(segmentBytes))
			if readErr != nil {
				return Admission{}, readErr
			}
			var sofErr error
			parsedSOF, sofErr := parseJPEGSOF0(sofBytes)
			if sofErr != nil {
				return Admission{}, sofErr
			}
			frame = parsedSOF

		case 0xc2:
			return Admission{}, unsupportedReject("progressive JPEG")

		case 0xc1, 0xc3, 0xc5, 0xc6, 0xc7,
			0xc8, 0xc9, 0xca, 0xcb, 0xcd, 0xce, 0xcf:
			return Admission{}, unsupportedReject("unsupported JPEG SOF mode")

		case 0xda: // SOS; entropy-coded bytes begin here.
			if !seenSOF {
				return Admission{}, invalidReject("JPEG SOS appears before SOF0")
			}
			sos, readErr := readBounded(r, size, dataOffset, int(segmentBytes))
			if readErr != nil {
				return Admission{}, readErr
			}
			if sosErr := validateJPEGSOS(sos, frame); sosErr != nil {
				return Admission{}, sosErr
			}
			return jpegAdmission(
				size,
				frame,
				orientation,
			)

		case 0xe1: // APP1, potentially EXIF.
			payload, readErr := readBounded(r, size, dataOffset, int(segmentBytes))
			if readErr != nil {
				return Admission{}, readErr
			}
			if len(payload) >= 6 && string(payload[:6]) == "Exif\x00\x00" {
				if seenEXIF {
					return Admission{}, metadataReject("JPEG has more than one EXIF APP1 segment")
				}
				if len(payload)-6 > exifMaxBytes {
					return Admission{}, metadataReject("JPEG EXIF payload exceeds 64KiB")
				}
				orientation, err = exifOrientation(payload[6:])
				if err != nil {
					return Admission{}, err
				}
				seenEXIF = true
			}

		case 0xc4, 0xdb, 0xdd, 0xfe,
			0xe0, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7,
			0xe8, 0xe9, 0xea, 0xeb, 0xec, 0xed, 0xee, 0xef:
			// DHT, DQT, DRI, comments, and opaque APP segments are skipped.
			// Their bytes remain in the source snapshot for final validation.

		default:
			if marker < 0xc0 {
				return Admission{}, invalidReject("unknown JPEG marker")
			}
			return Admission{}, unsupportedReject("unsupported JPEG marker")
		}
		offset = segmentEnd
	}
}

type jpegSOF struct {
	width        int64
	height       int64
	nComponents  int
	lumaH        int
	lumaV        int
	chromaH      int
	chromaV      int
	componentIDs [3]uint8
}

func parseJPEGSOF0(sof []byte) (jpegSOF, error) {
	var parsed jpegSOF
	switch len(sof) {
	case 9:
		parsed.nComponents = 1
	case 15:
		parsed.nComponents = 3
	case 18:
		return parsed, unsupportedReject("four-component JPEG/CMYK")
	default:
		return parsed, unsupportedReject("unsupported JPEG component count")
	}
	if sof[0] != 8 {
		return parsed, unsupportedReject("JPEG precision other than 8 bits")
	}
	parsed.height = int64(binary.BigEndian.Uint16(sof[1:3]))
	parsed.width = int64(binary.BigEndian.Uint16(sof[3:5]))
	if int(sof[5]) != parsed.nComponents {
		return parsed, invalidReject("JPEG SOF component count disagrees with length")
	}
	if parsed.width < 1 || parsed.height < 1 ||
		parsed.width > MaxDimension || parsed.height > MaxDimension {
		return parsed, &RejectionError{
			RejectionDimensions,
			"JPEG dimensions must be 1..8192",
		}
	}

	seenIDs := make(map[uint8]bool, parsed.nComponents)
	totalSampling := 0
	for i := 0; i < parsed.nComponents; i++ {
		component := sof[6+3*i : 9+3*i]
		id := component[0]
		if seenIDs[id] {
			return parsed, invalidReject("JPEG has a repeated component identifier")
		}
		seenIDs[id] = true
		parsed.componentIDs[i] = id
		h, v := int(component[1]>>4), int(component[1]&0x0f)
		if h < 1 || h > 2 || v < 1 || v > 2 {
			return parsed, unsupportedReject("JPEG sampling factors other than 1 or 2")
		}
		totalSampling += h * v
		if int(component[2]) > 3 {
			return parsed, invalidReject("JPEG quantization table selector exceeds 3")
		}
		if i == 0 {
			parsed.lumaH, parsed.lumaV = h, v
		} else {
			if i == 1 {
				parsed.chromaH, parsed.chromaV = h, v
			} else if h != parsed.chromaH || v != parsed.chromaV {
				return parsed, unsupportedReject("nonstandard JPEG chroma sampling")
			}
			if h > parsed.lumaH || v > parsed.lumaV {
				return parsed, unsupportedReject("JPEG luma sampling is not maximal")
			}
		}
	}
	if parsed.nComponents == 1 {
		parsed.lumaH, parsed.lumaV = 1, 1
	}
	if parsed.nComponents == 3 {
		if totalSampling > 10 {
			return parsed, unsupportedReject("JPEG total sampling factors exceed baseline limit")
		}
		hRatio, vRatio := parsed.lumaH/parsed.chromaH, parsed.lumaV/parsed.chromaV
		switch {
		case hRatio == 1 && vRatio == 1:
		case hRatio == 1 && vRatio == 2:
		case hRatio == 2 && vRatio == 1:
		case hRatio == 2 && vRatio == 2:
		default:
			return parsed, unsupportedReject("unsupported JPEG subsampling ratio")
		}
	}
	return parsed, nil
}

func validateJPEGSOS(sos []byte, sof jpegSOF) error {
	if len(sos) < 2 || int(sos[0]) != sof.nComponents {
		return invalidReject("JPEG SOS component count is invalid")
	}
	if len(sos) != 4+2*sof.nComponents {
		return invalidReject("JPEG SOS length is inconsistent with components")
	}
	seen := make(map[uint8]bool, sof.nComponents)
	for i := 0; i < sof.nComponents; i++ {
		selector := sos[1+2*i]
		if seen[selector] || !slicesContainsByte(sof.componentIDs[:sof.nComponents], selector) {
			return invalidReject("JPEG SOS component selector is invalid")
		}
		seen[selector] = true
		tableSelectors := sos[2+2*i]
		if tableSelectors>>4 > 1 || tableSelectors&0x0f > 1 {
			return invalidReject("JPEG baseline SOS table selector exceeds 1")
		}
	}
	tail := 1 + 2*sof.nComponents
	if sos[tail] != 0 || sos[tail+1] != 63 || sos[tail+2] != 0 {
		return invalidReject("JPEG baseline SOS spectral parameters are invalid")
	}
	return nil
}

func slicesContainsByte(values []uint8, target uint8) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func jpegAdmission(
	size int64,
	sof jpegSOF,
	orientation uint8,
) (Admission, error) {
	pixels, err := checkedMul(sof.width, sof.height)
	if err != nil {
		return Admission{}, err
	}
	if pixels > MaxPixels {
		return Admission{}, &RejectionError{
			RejectionDimensions,
			"JPEG pixel count exceeds 8,388,608",
		}
	}
	mipBytes, mipLevels, err := mipBudget(int(sof.width), int(sof.height))
	if err != nil {
		return Admission{}, err
	}

	decoderBytes, err := jpegDecoderSurfaceBytes(sof)
	if err != nil {
		return Admission{}, err
	}
	rgbaBytes, err := checkedMul(pixels, 4)
	if err != nil {
		return Admission{}, err
	}
	payload, err := sumPayload(
		size,
		decoderBytes,
		jpegStreamScratchBytes,
		rgbaBytes,
	)
	if err != nil {
		return Admission{}, err
	}
	if sof.nComponents == 3 {
		payload, err = checkedAdd(payload, rgbaBytes)
		if err != nil {
			return Admission{}, err
		}
	}
	if orientation >= 2 && orientation <= 8 {
		payload, err = checkedAdd(payload, rgbaBytes)
		if err != nil {
			return Admission{}, err
		}
	}
	payload, err = decodeBudget(payload)
	if err != nil {
		return Admission{}, err
	}

	return Admission{
		Kind:               KindJPEG,
		Subset:             SubsetJPEGBaseline,
		Width:              int(sof.width),
		Height:             int(sof.height),
		Pixels:             pixels,
		Orientation:        orientation,
		CompressedBytes:    size,
		BaseRGBABytes:      rgbaBytes,
		MipChainBytes:      mipBytes,
		MipLevels:          mipLevels,
		DecodePayloadBytes: payload,
	}, nil
}

func jpegDecoderSurfaceBytes(sof jpegSOF) (int64, error) {
	mcuColumns := ceilDiv(sof.width, int64(8*sof.lumaH))
	mcuRows := ceilDiv(sof.height, int64(8*sof.lumaV))

	lumaWidth, err := checkedMul(mcuColumns, int64(8*sof.lumaH))
	if err != nil {
		return 0, err
	}
	lumaHeight, err := checkedMul(mcuRows, int64(8*sof.lumaV))
	if err != nil {
		return 0, err
	}
	lumaPlane, err := checkedMul(lumaWidth, lumaHeight)
	if err != nil {
		return 0, err
	}
	if sof.nComponents == 1 {
		return lumaPlane, nil
	}

	chromaWidth, err := checkedMul(mcuColumns, int64(8*sof.chromaH))
	if err != nil {
		return 0, err
	}
	chromaHeight, err := checkedMul(mcuRows, int64(8*sof.chromaV))
	if err != nil {
		return 0, err
	}
	chromaPlane, err := checkedMul(chromaWidth, chromaHeight)
	if err != nil {
		return 0, err
	}
	twoChromaPlanes, err := checkedMul(chromaPlane, 2)
	if err != nil {
		return 0, err
	}
	return checkedAdd(lumaPlane, twoChromaPlanes)
}

func ceilDiv(value int64, divisor int64) int64 {
	return (value + divisor - 1) / divisor
}
