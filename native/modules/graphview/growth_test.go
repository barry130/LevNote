package graphview

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func TestGrowthGraphDraftRefusalRetainsOwnerThroughRelease(t *testing.T) {
	source := MindmapAdapter().Template()
	buffer := text.NewWithLimit(source, len(source))
	var reported error
	var recovered string
	changes := 0
	v := newView(kindMindmap, editor.Host{Buffer: buffer, Path: "growth.mindmap", Report: func(err error) { reported = err }, Changed: func() { changes++ }, RecoverDraft: func(_, value string) error {
		recovered = value
		return nil
	}})
	tester := ui.NewTester(v.Draw, 960, 640)
	focusDraft(t, tester)
	value := strings.Repeat("节点🙂", 100)
	tester.Compose(value, len([]rune(value)))
	draft := v.draft
	version, selection, history, snapshot := buffer.Version(), buffer.Selection(), buffer.History(), buffer.Snapshot()
	if err := v.Flush(); !errors.Is(err, text.ErrSizeLimit) || !errors.Is(reported, text.ErrSizeLimit) {
		t.Fatal("serialized graph growth not refused", err, reported)
	}
	if v.draft != draft || !v.pendingDraft() || draft.input.Buffer.String() != value || changes != 0 ||
		buffer.Version() != version || buffer.Selection() != selection || buffer.History() != history || !buffer.EqualSnapshot(snapshot) || buffer.InGroup() {
		t.Fatal("graph refusal damaged source/group/draft")
	}
	if v.selectField(fieldNote) {
		t.Fatal("field switch bypassed failed writeback")
	}
	v.Release()
	tester.Frame()
	if v.draft != draft || draft.input.Buffer.String() != value || !v.pendingDraft() || buffer.String() != source {
		t.Fatal("graph release discarded failed growth draft")
	}
	if err := v.RecoverDraft(); err != nil || !strings.HasSuffix(recovered, value) || v.draft != nil || buffer.String() != source || buffer.Version() != version {
		t.Fatal("graph recovery lost full draft or changed source", err)
	}
}
