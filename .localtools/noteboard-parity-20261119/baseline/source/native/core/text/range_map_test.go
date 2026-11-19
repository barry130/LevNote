package text

import "testing"

func TestRangeMapPreservesUnicodeFieldThroughOutsideEditsAndUndo(t *testing.T) {
	b := New("前🙂 target 尾")
	version := b.Version()
	span := Range{11, 4} // reversed "target "
	_ = b.ReplaceUTF16(0, 1, "前文")
	_ = b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "后")
	mapped, ok := b.MapRangeSince(version, span)
	value, _ := b.SliceUTF16(mapped.Start(), mapped.End())
	if !ok || mapped.Anchor <= mapped.Head || value != "target " {
		t.Fatalf("mapped=%+v ok=%v value=%q", mapped, ok, value)
	}
	b.Undo()
	b.Undo()
	if mapped, ok = b.MapRangeSince(version, span); !ok || mapped != span {
		t.Fatalf("outside undo mapping: %+v %v", mapped, ok)
	}
	b.Redo()
	if mapped, ok = b.MapRangeSince(version, span); !ok || mapped != (Range{12, 5}) {
		t.Fatalf("outside redo mapping: %+v %v", mapped, ok)
	}
}

func TestRangeMapBoundaryInsertionsAndOverlap(t *testing.T) {
	for _, tc := range []struct {
		name       string
		start, end int
		span       Range
		want       Range
		ok         bool
	}{
		{"insert-start-before", 2, 2, Range{2, 4}, Range{3, 5}, true},
		{"insert-end-after", 4, 4, Range{2, 4}, Range{2, 4}, true},
		{"inside", 3, 3, Range{2, 4}, Range{}, false},
		{"empty", 2, 2, Range{2, 2}, Range{}, false},
		{"overlap", 1, 3, Range{2, 4}, Range{}, false},
		{"replace-entire", 2, 4, Range{2, 4}, Range{}, false},
		{"replace-before", 0, 2, Range{2, 4}, Range{1, 3}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := New("abcdef")
			v := b.Version()
			_ = b.ReplaceUTF16(tc.start, tc.end, "x")
			got, ok := b.MapRangeSince(v, tc.span)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("%+v %v want %+v %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestRangeMapCancelRestoreAndExpiryRejectAmbiguousTargets(t *testing.T) {
	b := New("abcdef")
	v := b.Version()
	_ = b.BeginGroup()
	_ = b.ReplaceUTF16(0, 1, "outside")
	b.CancelGroup()
	if span, ok := b.MapRangeSince(v, Range{2, 4}); !ok || span != (Range{2, 4}) {
		t.Fatal("canceled outside group did not map", span, ok)
	}
	_ = b.BeginGroup()
	_ = b.ReplaceUTF16(2, 3, "inside")
	b.CancelGroup()
	if _, ok := b.MapRangeSince(v, Range{2, 4}); ok {
		t.Fatal("target edited then canceled must be invalidated conservatively")
	}
	v = b.Version()
	b.Restore(b.Snapshot())
	if _, ok := b.MapRangeSince(v, Range{2, 4}); ok {
		t.Fatal("restore did not invalidate target")
	}
	v = b.Version()
	for i := 0; i < 65; i++ {
		_ = b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "x")
	}
	if _, ok := b.MapRangeSince(v, Range{2, 4}); ok {
		t.Fatal("expired journal accepted")
	}
	if _, ok := b.MapRangeSince(b.Version()+1, Range{2, 4}); ok {
		t.Fatal("future version accepted")
	}
}
