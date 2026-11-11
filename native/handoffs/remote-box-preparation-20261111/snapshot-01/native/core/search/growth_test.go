package search

import (
	"errors"
	"testing"

	"levnote.local/native/core/text"
)

func TestGrowthReplaceAllPreflightsWholeCommand(t *testing.T) {
	for _, tc := range []struct {
		name, source, query, replacement string
		limit                            int
		regex                            bool
		wantErr                          error
	}{
		{"literal-late-refusal", "x x", "x", "xx", 4, false, text.ErrSizeLimit},
		{"regex-late-refusal", "a1 b2", `([ab])\d`, "$1$1$1", 6, true, text.ErrSizeLimit},
		{"unicode-growth", "中 中", "中", "🙂", 8, false, text.ErrSizeLimit},
		{"zero-width", "甲\n乙", "(?m)^", ">", 8, true, text.ErrSizeLimit},
		{"regex-final-growth", "aaaa b", "(aaaa)|b", "${1}${1}", 6, true, text.ErrSizeLimit},
		{"invalid-utf8", "x x", "x", "\xff", 4, false, text.ErrUTF8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := text.NewWithLimit(tc.source, tc.limit)
			_ = b.ReplaceUTF16(0, b.LenUTF16(), "")
			if !b.Undo() {
				t.Fatal("redo fixture")
			}
			_ = b.SetSelection(b.LenUTF16(), 0)
			version, selection, history, snapshot := b.Version(), b.Selection(), b.History(), b.Snapshot()
			n, err := ReplaceAll(b, Options{Query: tc.query, Regex: tc.regex, CaseSensitive: true}, tc.replacement)
			if n != 0 || !errors.Is(err, tc.wantErr) {
				t.Fatal(n, err)
			}
			if b.Version() != version || b.Selection() != selection || b.History() != history || !b.EqualSnapshot(snapshot) || b.InGroup() {
				t.Fatal("whole-command refusal changed state before BeginGroup")
			}
		})
	}
}

func TestGrowthReplaceAllConservativeReverseBoundary(t *testing.T) {
	// Reverse order expands final b first, although replacing the preceding
	// aaaa with xx would later shrink enough. Final result "xx xx" fits6bytes.
	b := text.NewWithLimit("aaaa b", 6)
	version := b.Version()
	if n, err := ReplaceAll(b, Options{Query: "aaaa|b", Regex: true}, "xx"); n != 0 || !errors.Is(err, text.ErrSizeLimit) || b.Version() != version || b.String() != "aaaa b" {
		t.Fatal("oversized intermediate was not conservatively refused", n, err)
	}
}

func TestGrowthReplaceAllAcceptedHistoryAndForeignGroup(t *testing.T) {
	b := text.NewWithLimit("甲1 乙2", 11)
	_ = b.SetSelection(b.LenUTF16(), 0)
	selection := b.Selection()
	opts := Options{Query: `([甲乙])(\d)`, Regex: true}
	n, err := ReplaceAll(b, opts, "$2$1!")
	if err != nil || n != 2 || b.String() != "1甲! 2乙!" || b.History().UndoDepth != 1 {
		t.Fatal("exact-boundary regex", b.String(), n, err)
	}
	if !b.Undo() || b.String() != "甲1 乙2" || b.Selection() != selection || !b.Redo() || b.String() != "1甲! 2乙!" {
		t.Fatal("group undo/redo failed")
	}
	if err = b.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	version, snapshot := b.Version(), b.Snapshot()
	if _, err = ReplaceAll(b, Options{Query: "!"}, "too big"); !errors.Is(err, text.ErrGroup) || !b.InGroup() || b.Version() != version || !b.EqualSnapshot(snapshot) {
		t.Fatal("caller group ownership", err)
	}
	b.CancelGroup()
}
