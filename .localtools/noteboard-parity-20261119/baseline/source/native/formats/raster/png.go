package raster

import (
	"encoding/binary"
	"io"
)

const maxPNGChunkLength = int64(0x7fffffff)

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

func admitPNG(r io.ReaderAt, size int64) (Admission, error) {
	offset := int64(len(pngSignature))
	chunkLength, chunkType, err := pngChunkHeader(r, size, offset)
	if err != nil {
		return Admission{}, err
	}
	if chunkType != "IHDR" || chunkLength != 13 {
		return Admission{}, invalidReject("PNG first chunk must be a 13-byte IHDR")
	}
	if _, err := pngChunkEnd(offset, chunkLength, size); err != nil {
		return Admission{}, err
	}

	ihdr, err := readBounded(r, size, offset+8, int(chunkLength))
	if err != nil {
		return Admission{}, err
	}
	width := int64(binary.BigEndian.Uint32(ihdr[0:4]))
	height := int64(binary.BigEndian.Uint32(ihdr[4:8]))
	depth := ihdr[8]
	colorType := ihdr[9]
	compression := ihdr[10]
	filter := ihdr[11]
	interlace := ihdr[12]
	if compression != 0 {
		return Admission{}, unsupportedReject("PNG compression method")
	}
	if filter != 0 {
		return Admission{}, unsupportedReject("PNG filter method")
	}
	bitsPerPixel, ok := pngBitsPerPixel(depth, colorType)
	if !ok {
		return Admission{}, unsupportedReject(pngColorDepthDetail(depth, colorType))
	}
	if interlace != 0 {
		if interlace == 1 {
			return Admission{}, unsupportedReject("interlaced PNG")
		}
		return Admission{}, invalidReject("PNG interlace method")
	}
	if width < 1 || height < 1 || width > MaxDimension || height > MaxDimension {
		return Admission{}, &RejectionError{
			RejectionDimensions,
			"PNG dimensions must be 1..8192",
		}
	}
	pixels, err := checkedMul(width, height)
	if err != nil {
		return Admission{}, err
	}
	if pixels > MaxPixels {
		return Admission{}, &RejectionError{
			RejectionDimensions,
			"PNG pixel count exceeds 8,388,608",
		}
	}
	mipBytes, mipLevels, err := mipBudget(int(width), int(height))
	if err != nil {
		return Admission{}, err
	}

	offset += 8 + chunkLength + 4
	destinationBytesPerPixel := pngDestinationBytesPerPixel(depth, colorType)
	seenIDAT := false
	hasIEND := false
	var orientation uint8
	seenEXIF := false
	seenPLTE := false
	seenTRNS := false
	var paletteEntries int64
	idatEnded := false
	chunkCount := int64(1) // The IHDR was already inspected above.
	const maxChunkCount = int64(4096)
	for {
		chunkCount++
		if chunkCount > maxChunkCount {
			return Admission{}, invalidReject("PNG contains too many chunks")
		}
		chunkLength, chunkType, err = pngChunkHeader(r, size, offset)
		if err != nil {
			return Admission{}, err
		}
		if chunkType == "acTL" {
			return Admission{}, unsupportedReject("APNG animation control chunk")
		}
		if chunkType == "fcTL" || chunkType == "fdAT" {
			return Admission{}, unsupportedReject("APNG frame chunk")
		}
		dataOffset := offset + 8
		chunkEnd, err := pngChunkEnd(offset, chunkLength, size)
		if err != nil {
			return Admission{}, err
		}
		nextOffset, err := checkedAdd(chunkEnd, 4)
		if err != nil {
			return Admission{}, err
		}
		if seenIDAT && chunkType != "IDAT" {
			idatEnded = true
		}
		switch chunkType {
		case "IHDR":
			return Admission{}, invalidReject("PNG has more than one IHDR chunk")
		case "PLTE":
			if seenPLTE || seenIDAT {
				return Admission{}, invalidReject("PNG PLTE chunk is duplicated or after IDAT")
			}
			if colorType == 0 || colorType == 4 {
				return Admission{}, invalidReject("PNG PLTE chunk is invalid for this color type")
			}
			if chunkLength < 3 || chunkLength > 768 || chunkLength%3 != 0 {
				return Admission{}, invalidReject("PNG PLTE length is invalid")
			}
			seenPLTE = true
			paletteEntries = chunkLength / 3
		case "eXIf":
			if seenEXIF {
				return Admission{}, metadataReject("PNG has more than one eXIf chunk")
			}
			if chunkLength > exifMaxBytes {
				return Admission{}, metadataReject("PNG eXIf chunk exceeds 64KiB")
			}
			payload, readErr := readBounded(r, size, dataOffset, int(chunkLength))
			if readErr != nil {
				return Admission{}, readErr
			}
			orientation, err = exifOrientation(payload)
			if err != nil {
				return Admission{}, err
			}
			seenEXIF = true
		case "tRNS":
			if seenIDAT || seenTRNS {
				return Admission{}, invalidReject("PNG tRNS chunk is duplicated or after IDAT")
			}
			switch colorType {
			case 0:
				if chunkLength != 2 {
					return Admission{}, invalidReject("PNG grayscale tRNS length is not 2")
				}
			case 2:
				if chunkLength != 6 {
					return Admission{}, invalidReject("PNG truecolor tRNS length is not 6")
				}
			case 3:
				if !seenPLTE || chunkLength > paletteEntries {
					return Admission{}, invalidReject("PNG indexed tRNS length or order is invalid")
				}
			default:
				return Admission{}, invalidReject("PNG tRNS is invalid for this color type")
			}
			if colorType == 0 || colorType == 2 {
				destinationBytesPerPixel = 4
			}
			seenTRNS = true
		case "IDAT":
			if idatEnded {
				return Admission{}, invalidReject("PNG IDAT chunks are not consecutive")
			}
			if colorType == 3 && !seenPLTE {
				return Admission{}, invalidReject("indexed PNG has no PLTE before IDAT")
			}
			seenIDAT = true
		case "IEND":
			if chunkLength != 0 {
				return Admission{}, invalidReject("PNG IEND chunk must be empty")
			}
			if nextOffset != size {
				return Admission{}, invalidReject("PNG has bytes after IEND")
			}
			hasIEND = true
		default:
			if !pngChunkTypeValid(chunkType) {
				return Admission{}, invalidReject("PNG chunk type is invalid")
			}
			if chunkType[0]&0x20 == 0 {
				return Admission{}, invalidReject("PNG has an unknown critical chunk")
			}
		}
		if hasIEND {
			break
		}
		offset = nextOffset
	}
	if !seenIDAT || !hasIEND {
		return Admission{}, invalidReject("PNG requires IDAT and IEND before end of input")
	}

	rowBytes := 1 + (bitsPerPixel*width+7)/8
	rowBytes, err = checkedMul(rowBytes, 2)
	if err != nil {
		return Admission{}, err
	}
	decoderBytes, err := checkedMul(pixels, destinationBytesPerPixel)
	if err != nil {
		return Admission{}, err
	}
	rgbaBytes, err := checkedMul(pixels, 4)
	if err != nil {
		return Admission{}, err
	}
	payload, err := sumPayload(size, decoderBytes, rowBytes, rgbaBytes)
	if err != nil {
		return Admission{}, err
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
		Kind:               KindPNG,
		Subset:             SubsetPNG8,
		Width:              int(width),
		Height:             int(height),
		Pixels:             pixels,
		Orientation:        orientation,
		CompressedBytes:    size,
		BaseRGBABytes:      pixels * 4,
		MipChainBytes:      mipBytes,
		MipLevels:          mipLevels,
		DecodePayloadBytes: payload,
	}, nil
}

func pngChunkTypeValid(chunkType string) bool {
	if len(chunkType) != 4 {
		return false
	}
	for i := 0; i < 4; i++ {
		c := chunkType[i]
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
			return false
		}
	}
	return chunkType[2]&0x20 == 0
}

func pngChunkHeader(r io.ReaderAt, size int64, offset int64) (int64, string, error) {
	if offset > size || size-offset < 8 {
		return 0, "", truncatedReject("PNG chunk header exceeds declared size")
	}
	header, err := readBounded(r, size, offset, 8)
	if err != nil {
		return 0, "", err
	}
	length := int64(binary.BigEndian.Uint32(header[0:4]))
	if length > maxPNGChunkLength {
		return 0, "", invalidReject("PNG chunk length exceeds 31-bit limit")
	}
	return length, string(header[4:8]), nil
}

func pngChunkEnd(offset int64, length int64, size int64) (int64, error) {
	dataEnd, err := checkedAdd(offset+8, length)
	if err != nil {
		return 0, err
	}
	total, err := checkedAdd(dataEnd, 4)
	if err != nil {
		return 0, err
	}
	if total > size {
		return 0, truncatedReject("PNG chunk exceeds declared size")
	}
	return dataEnd, nil
}

func pngBitsPerPixel(depth uint8, colorType uint8) (int64, bool) {
	switch colorType {
	case 0:
		if depth == 1 || depth == 2 || depth == 4 || depth == 8 {
			return int64(depth), true
		}
	case 2:
		if depth == 8 {
			return 24, true
		}
	case 3:
		if depth == 1 || depth == 2 || depth == 4 || depth == 8 {
			return int64(depth), true
		}
	case 4:
		if depth == 8 {
			return 16, true
		}
	case 6:
		if depth == 8 {
			return 32, true
		}
	}
	return 0, false
}

func pngDestinationBytesPerPixel(depth uint8, colorType uint8) int64 {
	switch colorType {
	case 0, 3:
		return 1
	case 2, 4, 6:
		return 4
	default:
		return 1
	}
}

func pngColorDepthDetail(depth uint8, colorType uint8) string {
	return "PNG bit depth/color type " +
		itoa(int(depth)) + "/" + itoa(int(colorType))
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	negative := value < 0
	if negative {
		value = -value
	}
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		i--
		digits[i] = '-'
	}
	return string(digits[i:])
}

func sumPayload(size int64, parts ...int64) (int64, error) {
	var total int64
	for _, part := range parts {
		var err error
		total, err = checkedAdd(total, part)
		if err != nil {
			return 0, err
		}
	}
	return checkedAdd(total, size)
}
