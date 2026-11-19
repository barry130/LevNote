package graphview

import (
	"levnote.local/native/core/text"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestDrawioMultiDeleteOneTransactionPreservesRemainingXMLAndUndo(t *testing.T) {
	v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
	selectMultiNodes(t, tester, v, "a", "b")
	if err := buffer.SetSelection(8, 52); err != nil {
		t.Fatal(err)
	}
	before, caret, version := buffer.String(), buffer.Selection(), buffer.Version()
	tester.Key(0, ui.KeyDelete)
	if *changed != 1 || buffer.Version() != version+1 {
		t.Fatalf("want one source transaction: changed=%d version=%d", *changed, buffer.Version())
	}
	after := buffer.String()
	if strings.Contains(after, `id="a" `) || strings.Contains(after, `id="b" source=`) || strings.Contains(after, `id="edge" edge=`) {
		t.Fatal("selected vertices or their incident edge survived")
	}
	for _, want := range []string{`id="c"`, `width="60" height="100.0"`, "<!--unknown spelling-->", `<future z='007'/>`, `<mxfile custom="keep-🙂">`} {
		if !strings.Contains(after, want) {
			t.Fatalf("remaining XML changed: %s", want)
		}
	}
	if y := parseMultiDrawio(t, buffer).Pages[0].Model.Nodes[0].Y; y != 200.75 {
		t.Fatal("remaining node position changed", y)
	}
	if v.selectedID != "" || len(v.drawSelection.ids) != 0 {
		t.Fatal("deleted selection remains visible")
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != before || buffer.Selection() != caret || *changed != 2 {
		t.Fatal("undo failed to restore exact original XML/caret")
	}
	tester.Key(ui.Cmd, ui.KeyY)
	if buffer.String() != after || buffer.Selection() != caret || *changed != 3 {
		t.Fatal("redo failed to restore exact deleted XML/caret")
	}
}

func TestDrawioMultiDeleteSecondNoSelectionKeepsSource(t *testing.T) {
	v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
	selectMultiNodes(t, tester, v, "a", "b")
	tester.Key(0, ui.KeyDelete)
	if *changed != 1 {
		t.Fatal("delete did not commit one source transaction")
	}
	before, version := buffer.String(), buffer.Version()
	tester.Key(0, ui.KeyDelete)
	if buffer.String() != before || buffer.Version() != version || *changed != 1 {
		t.Fatal("second delete without a selection changed source")
	}
}

func TestDrawioMultiDeleteReplacedBufferKeepsOriginalSource(t *testing.T) {
	v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
	selectMultiNodes(t, tester, v, "b", "c")
	before, version := buffer.String(), buffer.Version()
	v.host.Buffer = text.New(multiDrawio)
	tester.Key(0, ui.KeyBackspace)
	if buffer.String() != before || buffer.Version() != version || *changed != 0 {
		t.Fatal("delete escaped into a replaced host buffer")
	}
}


func TestDrawioMultiDeleteInvalidTargetDoesNotWrite(t *testing.T) {
	v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
	selectMultiNodes(t, tester, v, "a", "b")
	v.setDrawioSelection([]string{"a", "missing"}, "missing")
	before, version, caret := buffer.String(), buffer.Version(), buffer.Selection()
	tester.Key(0, ui.KeyBackspace)
	if buffer.String() != before || buffer.Version() != version || buffer.Selection() != caret || *changed != 0 {
		t.Fatal("invalid target caused partial deletion or caret change")
	}
	selectMultiNodes(t, tester, v, "b", "c")
	v.host.ReadOnly = true
	tester.Key(0, ui.KeyDelete)
	if buffer.String() != before || buffer.Version() != version || *changed != 0 {
		t.Fatal("readonly deletion changed source")
	}
}
