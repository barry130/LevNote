package raster

import (
	"encoding/binary"
	"io"
)

const (
	exifMaxBytes   = 64 * 1024
	orientationTag = 0x0112
)

// exifOrientation parses one bounded IFD0 orientation SHORT. It returns zero
// when there is no orientation tag. Malformed or ambiguous metadata is an
// explicit rejection.
func exifOrientation(tiff []byte) (uint8, error) {
	if len(tiff) < 8 {
		return 0, truncatedReject("EXIF shorter than TIFF header")
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0, metadataReject("EXIF TIFF byte order is invalid")
	}
	if order.Uint16(tiff[2:4]) != 42 {
		return 0, metadataReject("EXIF TIFF magic is invalid")
	}
	ifdOffset := int(order.Uint32(tiff[4:8]))
	if ifdOffset < 8 || ifdOffset > len(tiff)-2 || ifdOffset > exifMaxBytes-2 {
		return 0, metadataReject("EXIF IFD0 offset is invalid")
	}
	entryCount := int(order.Uint16(tiff[ifdOffset : ifdOffset+2]))
	if entryCount > (exifMaxBytes-2)/12 {
		return 0, metadataReject("EXIF IFD0 entry count exceeds bounds")
	}
	end := ifdOffset + 2 + entryCount*12
	if end < ifdOffset || end > len(tiff)-4 {
		return 0, metadataReject("EXIF IFD0 entries exceed payload")
	}

	var orientation uint8
	found := false
	for i := 0; i < entryCount; i++ {
		entry := ifdOffset + 2 + i*12
		if order.Uint16(tiff[entry:entry+2]) != orientationTag {
			continue
		}
		if found {
			return 0, metadataReject("EXIF orientation tag appears more than once")
		}
		if order.Uint16(tiff[entry+2:entry+4]) != 3 || order.Uint32(tiff[entry+4:entry+8]) != 1 {
			return 0, metadataReject("EXIF orientation is not one SHORT value")
		}
		value := order.Uint16(tiff[entry+8 : entry+10])
		if value < 1 || value > 8 {
			return 0, metadataReject("EXIF orientation is outside 1..8")
		}
		orientation, found = uint8(value), true
	}
	return orientation, nil
}

// readBounded returns data read through ReaderAt without copying the input.
func readBounded(r io.ReaderAt, size int64, offset int64, n int) ([]byte, error) {
	if offset < 0 {
		return nil, invalidReject("negative metadata offset")
	}
	if n < 0 {
		return nil, invalidReject("negative metadata length")
	}
	length := int64(n)
	if offset > size || length > size-offset {
		return nil, truncatedReject("metadata segment exceeds declared size")
	}
	buf := make([]byte, n)
	read, err := r.ReadAt(buf, offset)
	if err != nil {
		return nil, readReject("metadata segment", err)
	}
	if read != n {
		return nil, truncatedReject("metadata segment ended early")
	}
	return buf, nil
}
