package editor

import "image"

// ImageRequest asks the host to decode the binary document owned by the
// active view. The view supplies no path, bytes, or reader: the host captures
// that authority together with the active token, owner, and generation when it
// accepts the request. Epoch is the view's generation at that moment.
type ImageRequest struct {
	Epoch uint64
}

// ImageResult is one decoded raster plus the small metadata needed to present
// it. Pixels, when successful, is one origin-zero, tightly packed,
// premultiplied RGBA surface owned by the callback receiver. Format is a
// stable display label such as "PNG" or "JPEG". Width and Height are the final
// dimensions after EXIF orientation; SourceWidth and SourceHeight are the
// codec dimensions before orientation. Orientation is the EXIF orientation
// (zero when absent), and CompressedBytes is the admitted document size.
type ImageResult struct {
	Format          string
	Width, Height   int
	SourceWidth     int
	SourceHeight    int
	Orientation     uint8
	CompressedBytes int64
	Pixels          *image.RGBA
}
