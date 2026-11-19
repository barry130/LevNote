package markdown

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func applyAlignmentPatch(source string, patch TableAlignmentPatch) string {
	if patch.Noop {
		return source
	}
	return source[:patch.Range.Start] + patch.Text + source[patch.Range.End:]
}

func TestTableAlignmentPlanEveryTransitionAndNoop(t *testing.T) {
	markers := []string{"-----", ":-----", ":-----:", "-----:"}
	for from, original := range markers {
		for to, wantMarker := range markers {
			t.Run(fmt.Sprintf("%d-to-%d", from, to), func(t *testing.T) {
				const prefix = "\ufeff前🙂\r\n\r\n  H🙂 | B\r\n  \t"
				const suffix = " \t| ---:\r\n  甲\\|乙 | **原文**\r\n\r\n尾\n"
				source := prefix + original + suffix
				table := rowEditTable(t, source, 0)
				patch, err := PlanTableAlignmentEdit(source, table.Range, 0, TableAlignment(to))
				if err != nil {
					t.Fatal(err)
				}
				if from == to {
					if patch != (TableAlignmentPatch{Noop: true}) {
						t.Fatalf("semantic noop normalized marker: %+v", patch)
					}
				} else if patch.Noop || patch.Range != (SourceRange{len(prefix), len(prefix) + len(original)}) || patch.Text != wantMarker {
					t.Fatalf("patch=%+v want marker%q", patch, wantMarker)
				}
				if got := applyAlignmentPatch(source, patch); got != prefix+wantMarker+suffix {
					t.Fatalf("exact source mismatch: %q", got)
				}
			})
		}
	}
}

func TestTableAlignmentPlanExactSyntaxAndContext(t *testing.T) {
	for _, test := range []struct {
		name, source, old, marker, want string
		column                          int
		alignment                       TableAlignment
	}{
		{"header-only-no-ending", "| H |\r\n|:-----:|", ":-----:", "-----:", "| H |\r\n|-----:|", 0, TableAlignRight},
		{"one-empty-column", "|\r|---|\r|", "---", ":---", "|\r|:---|\r|", 0, TableAlignLeft},
		{"mixed-ending-padding", "\t| H | B |\n | --- |\t:----:\t|\r\n| 甲 | 乙 |\r", ":----:", "----", "\t| H | B |\n | --- |\t----\t|\r\n| 甲 | 乙 |\r", 1, TableAlignDefault},
		{"eight-columns", "a|b|c|d|e|f|g|h\n---|---|---|---|---|---|---|---:\n1|2|3|4|5|6|7|8\n", "---:", ":---:", "a|b|c|d|e|f|g|h\n---|---|---|---|---|---|---|:---:\n1|2|3|4|5|6|7|8\n", 7, TableAlignCenter},
	} {
		t.Run(test.name, func(t *testing.T) {
			table := rowEditTable(t, test.source, 0)
			patch, err := PlanTableAlignmentEdit(test.source, table.Range, test.column, test.alignment)
			if err != nil || patch.Noop || patch.Text != test.marker || test.source[patch.Range.Start:patch.Range.End] != test.old ||
				applyAlignmentPatch(test.source, patch) != test.want {
				t.Fatalf("patch=%+v err=%v", patch, err)
			}
		})
	}
	const table = "| H | B |\n|---|---|\n| x | y |\n"
	const context = "\n# Heading *em*\n\n- [x] task [link](local.md)\n\n> quote **b**\n\n---\n\n```go\nx\n```\n\n```doc-block\nid: card\nkind: drawio\nmode: reference\nsrc: local.drawio\ncustom: value\n---\nraw\n```\n\n| K |\n|--|\n| unsupported |\n"
	source := "前\n\n" + table + context + "\n| other |\n|---|\n"
	patch, err := PlanTableAlignmentEdit(source, rowEditTable(t, source, 0).Range, 1, TableAlignCenter)
	want := "前\n\n| H | B |\n|---|:---:|\n| x | y |\n" + context + "\n| other |\n|---|\n"
	if err != nil || applyAlignmentPatch(source, patch) != want {
		t.Fatalf("unrelated structures/warning offsets changed: %+v %v", patch, err)
	}
}

func TestTableAlignmentPlanRefusesInvalidIdentity(t *testing.T) {
	const source = "前🙂\n\n| H |\n|---|\n| a |\n"
	table := rowEditTable(t, source, 0)
	for _, test := range []struct {
		name, source string
		rng          SourceRange
		column       int
		alignment    TableAlignment
		want         error
	}{
		{"enum", source, table.Range, 0, TableAlignment(255), ErrTableAlignmentValue},
		{"negative-column", source, table.Range, -1, TableAlignLeft, ErrTableAlignmentTarget},
		{"missing-column", source, table.Range, 1, TableAlignLeft, ErrTableAlignmentTarget},
		{"nine-column", source, table.Range, 8, TableAlignLeft, ErrTableAlignmentTarget},
		{"negative-range", source, rangeAt(-1, table.Range.End), 0, TableAlignLeft, ErrTableAlignmentTarget},
		{"range-end", source, rangeAt(table.Range.Start, len(source)+1), 0, TableAlignLeft, ErrTableAlignmentTarget},
		{"truncated", source, rangeAt(table.Range.Start, table.Range.End-1), 0, TableAlignLeft, ErrTableAlignmentTarget},
		{"utf8", "\xff" + source[1:], table.Range, 0, TableAlignLeft, ErrTableAlignmentTarget},
		{"fallback", strings.Replace(source, "|---|", "|-- |", 1), table.Range, 0, TableAlignLeft, ErrTableAlignmentTarget},
		{"fence", "```\n" + source + "```\n", shiftTableRange(table.Range, 4), 0, TableAlignLeft, ErrTableAlignmentTarget},
		{"partial", source + strings.Repeat("# h\n", maxBlocks), table.Range, 0, TableAlignLeft, ErrTableAlignmentLimit},
		{"over-document", source + strings.Repeat("x", MaxParseBytes), table.Range, 0, TableAlignLeft, ErrTableAlignmentLimit},
	} {
		t.Run(test.name, func(t *testing.T) {
			patch, err := PlanTableAlignmentEdit(test.source, test.rng, test.column, test.alignment)
			if patch != (TableAlignmentPatch{}) || !errors.Is(err, test.want) {
				t.Fatalf("patch=%+v err=%v want=%v", patch, err, test.want)
			}
		})
	}
}

func TestTableAlignmentPlanExactLimits(t *testing.T) {
	for _, kind := range []string{"delimiter-cell", "table-bytes", "document-bytes"} {
		t.Run(kind, func(t *testing.T) {
			for _, shrinking := range []bool{false, true} {
				marker, requested := "---", TableAlignCenter
				if shrinking {
					marker, requested = ":---:", TableAlignDefault
				}
				var source string
				switch kind {
				case "delimiter-cell":
					marker = strings.Repeat("-", maxTableCellTextBytes)
					if shrinking {
						marker = ":" + strings.Repeat("-", maxTableCellTextBytes-2) + ":"
					}
					source = "| H |\n|" + marker + "|\n"
				case "table-bytes":
					prefix := "| H |\n|" + marker + "|\n|"
					source = prefix + strings.Repeat(" ", maxTableRawBytes-len(prefix)-2) + "|\n"
				case "document-bytes":
					prefix := "| H |\n|" + marker + "|\n\n```\n"
					source = prefix + strings.Repeat("x", MaxParseBytes-len(prefix)-5) + "\n```\n"
				}
				table := rowEditTable(t, source, 0)
				patch, err := PlanTableAlignmentEdit(source, table.Range, 0, requested)
				if !shrinking {
					if !errors.Is(err, ErrTableAlignmentLimit) || patch != (TableAlignmentPatch{}) {
						t.Fatalf("growth at exact cap: %+v %v", patch, err)
					}
					noop, noopErr := PlanTableAlignmentEdit(source, table.Range, 0, TableAlignDefault)
					if noopErr != nil || noop != (TableAlignmentPatch{Noop: true}) {
						t.Fatalf("exact cap noop refused: %+v %v", noop, noopErr)
					}
				} else if err != nil || patch.Noop || patch.Text != strings.Trim(marker, ":") ||
					applyAlignmentPatch(source, patch) != strings.Replace(source, "|"+marker+"|", "|"+strings.Trim(marker, ":")+"|", 1) {
					t.Fatalf("shrinking at cap: %+v %v", patch, err)
				}
			}
		})
	}
	const header = "|a|b|c|d|e|f|g|h|\n|---|---|---|---|---|---|---|---|\n"
	const body = "|1|2|3|4|5|6|7|8|\n"
	source := header + strings.Repeat(body, maxTableBodyRows)
	patch, err := PlanTableAlignmentEdit(source, rowEditTable(t, source, 0).Range, 7, TableAlignRight)
	if err != nil || patch.Text != "---:" || len(Parse(applyAlignmentPatch(source, patch)).Blocks[0].Table.Rows) != maxTableBodyRows+1 {
		t.Fatalf("full row count should allow alignment: %+v %v", patch, err)
	}
}

func TestTableAlignmentPlanPreservesGlobalDescriptorBoundary(t *testing.T) {
	const header = "|a|b|c|d|e|f|g|h|\n|---|---|---|---|---|---|---|---|\n"
	const body = "|1|2|3|4|5|6|7|8|\n"
	one := header + strings.Repeat(body, 255)
	source := strings.Repeat(one+"\n", 8) + "| fallback |\n|---|\n"
	doc := Parse(source)
	if doc.Partial || len(doc.Blocks) != 9 || doc.Blocks[7].Table.Problem != "" || doc.Blocks[8].Table.Problem == "" {
		t.Fatal("fixture must exhaust descriptors without Partial")
	}
	patch, err := PlanTableAlignmentEdit(source, doc.Blocks[0].Range, 0, TableAlignLeft)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(source, "|---|---|", "|:---|---|", 1)
	if applyAlignmentPatch(source, patch) != want {
		t.Fatal("global limit action changed unrelated bytes")
	}
	after := Parse(want)
	if after.Partial || len(after.Blocks) != 9 || after.Blocks[7].Table.Problem != "" ||
		after.Blocks[8].Table.Problem != doc.Blocks[8].Table.Problem || len(after.Warnings) != len(doc.Warnings) ||
		after.Warnings[0].Range != shiftTableRange(doc.Warnings[0].Range, 1) {
		t.Fatal("fallback/warning/global descriptor boundary changed")
	}
}
