package text

import (
	"sort"
	"unicode/utf8"

	"github.com/go-text/typesetting/segmenter"
)

// GraphemeBoundary returns the adjacent logical grapheme boundary. Unicode
// segmentation, unlike font shaping, is part of editing the text itself. Small
// context windows keep ordinary movement/deletion independent of line length;
// an unusually long combining sequence grows the window until its full
// boundary is known. CRLF is one grapheme and surrogate pairs are never split.
func (b *Buffer) GraphemeBoundary(offset, direction int) (int, error) {
	if _, err := b.byteOffset(offset); err != nil {
		return offset, err
	}
	if direction < 0 && offset == 0 || direction >= 0 && offset == b.units {
		return offset, nil
	}
	context := 128
	for {
		start, end := max(0, offset-context), min(b.units, offset+context)
		if _, err := b.byteOffset(start); err == ErrSurrogate {
			start--
		}
		if _, err := b.byteOffset(end); err == ErrSurrogate {
			end++
		}
		value, err := b.SliceUTF16(start, end)
		if err != nil {
			return offset, err
		}
		var seg segmenter.Segmenter
		if err := seg.InitWithString(value); err != nil {
			return offset, err
		}
		boundaries := []int{start}
		iter := seg.GraphemeIterator()
		at := start
		clusterStart := 0
		for iter.Next() {
			cluster := iter.Grapheme()
			// go-text v0.3.5 loses GB11's pictograph state when one emoji
			// immediately follows another (🙂👨‍👩). Re-evaluate only a
			// candidate ZWJ boundary from the preceding accepted cluster's
			// true start. This preserves UAX behavior without copying or
			// modifying the toolkit or treating every ZWJ as an emoji join.
			if cluster.OffsetInBytes > 0 && len(boundaries) > 1 {
				previous, _ := utf8.DecodeLastRuneInString(value[:cluster.OffsetInBytes])
				if previous == '\u200d' {
					var check segmenter.Segmenter
					_ = check.InitWithString(value[clusterStart : cluster.OffsetInBytes+cluster.LengthInBytes])
					joined := check.GraphemeIterator()
					if joined.Next() && joined.Grapheme().LengthInBytes > cluster.OffsetInBytes-clusterStart {
						boundaries = boundaries[:len(boundaries)-1]
					} else {
						clusterStart = cluster.OffsetInBytes
					}
				} else {
					clusterStart = cluster.OffsetInBytes
				}
			}
			for _, r := range cluster.Text {
				at++
				if r > 0xffff {
					at++
				}
			}
			boundaries = append(boundaries, at)
		}
		i := sort.SearchInts(boundaries, offset)
		if direction < 0 {
			i--
		} else if i < len(boundaries) && boundaries[i] == offset {
			i++
		}
		i = max(0, min(i, len(boundaries)-1))
		// The artificial start/end can split a cluster. Require a complete
		// cluster of leading context and a following real boundary. Regional
		// indicator pairing carries parity through a whole run, so extend to
		// its true beginning instead of accepting a window's guessed parity.
		first, _ := utf8.DecodeRuneInString(value)
		regional := first >= 0x1f1e6 && first <= 0x1f1ff
		leftKnown := start == 0 || i >= 2 && !regional
		rightKnown := end == b.units || i < len(boundaries)-1
		if leftKnown && rightKnown {
			return boundaries[i], nil
		}
		context *= 2
	}
}
