package markdownview

import (
	"errors"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

// Actual Tester input only; callback successor construction reuses the pinned
// Task140 source/Editor seam and never calls Tester, Advance or Frame.
func TestMarkdownUIReportingReferenceForeignGroup(t *testing.T) {
	f := newMarkdownActionFixture(t, false)
	focusReference(t, f.tester, referenceAccess(t, f.tester, true))
	sameMain := markdownActionBufferReceipt(f.buffer)
	version, history, selection := f.buffer.Version(), f.buffer.History(), f.buffer.Selection()
	row := captureMarkdownAction(t, f.view, "reference").reference.rowID
	if err := f.buffer.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	assertGrouped := func() {
		t.Helper()
		if f.buffer.String() != markdownActionSource || f.buffer.Version() != version || f.buffer.History() != history ||
			f.buffer.Selection() != selection || !f.buffer.InGroup() || f.changes != 0 || len(f.opened) != 0 ||
			captureMarkdownAction(t, f.view, "reference").reference.rowID != row {
			t.Fatal("reference refusal changed source/row/selection/history or stole the foreign group")
		}
	}
	for attempt := 0; attempt < 2; attempt++ {
		f.tester.Advance(time.Second)
		info := referenceAccess(t, f.tester, true)
		x, y := info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2
		f.tester.Press(x, y)
		if f.view.referenceActivation.buffer != f.buffer || f.view.referenceActivation.rowID != row {
			t.Fatal("actual pointer-down did not capture the unchanged enabled reference")
		}
		f.tester.Release(x, y)
		if len(f.reports) != attempt+1 || f.reports[attempt] != errReferenceStale {
			t.Fatalf("one release must report once, independently: attempt=%d reports=%v", attempt, f.reports)
		}
		assertGrouped()
		f.tester.Frame()
		if len(f.reports) != attempt+1 || f.view.referenceActivation.buffer != nil || f.view.referencePointerUpBuffer != nil {
			t.Fatal("a queued click/frame repeated the refusal or retained the gesture")
		}
	}
	// Keyboard activation is a new invocation and keeps its action-level cause.
	if !f.tester.Focused("打开引用") {
		t.Fatal("real pointer did not leave the reference button focused")
	}
	f.tester.Key(0, ui.KeyEnter)
	if len(f.reports) != 3 || !errors.Is(f.reports[2], text.ErrGroup) {
		t.Fatalf("the earlier pointer receipt suppressed/changed a new Enter refusal: %v", f.reports)
	}
	assertGrouped()
	if f.buffer.EndGroup() {
		t.Fatal("refused reference gesture wrote into the external group")
	}
	sameMain(t)
	f.tester.Advance(time.Second)
	clickReference(t, f.tester, referenceAccess(t, f.tester, true))
	if len(f.reports) != 3 || len(f.opened) != 1 || f.opened[0] != "target.txt" || f.changes != 0 {
		t.Fatal("expired refusal receipt hid a later successful pointer action")
	}
	sameMain(t)
}

func TestMarkdownUIReportingReferenceGestureControls(t *testing.T) {
	for _, control := range []string{"canceled", "outside", "group-canceled", "group-outside", "no-capture", "row-rebuild"} {
		t.Run(control, func(t *testing.T) {
			f := newMarkdownActionFixture(t, false)
			sameMain := markdownActionBufferReceipt(f.buffer)
			info := referenceAccess(t, f.tester, true)
			x, y := info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2
			grouped := control == "group-canceled" || control == "group-outside"
			if grouped {
				if err := f.buffer.BeginGroup(); err != nil {
					t.Fatal(err)
				}
			}
			if control == "no-capture" {
				// Shift never captures a reference, but native Button still clicks.
				// A blanket missing-capture suppression would hide this refusal.
				f.tester.ClickAtWith(ui.Shift, x, y)
			} else {
				f.tester.Press(x, y)
				if control == "row-rebuild" {
					replaceReferenceBufferText(t, f.buffer, "src: target.txt", "src: other.txt")
					f.tester.Frame()
				}
				switch control {
				case "canceled", "group-canceled":
					f.tester.CancelPress()
				case "outside", "group-outside":
					f.tester.Release(info.Bounds.X+info.Bounds.W*3/2, y)
				default:
					f.tester.Release(x, y)
				}
			}
			f.tester.Frame()
			wantReports := 0
			if grouped || control == "no-capture" || control == "row-rebuild" {
				wantReports = 1
			}
			if len(f.reports) != wantReports || len(f.opened) != 0 || f.changes != 0 ||
				f.view.referenceActivation.buffer != nil || f.view.referencePointerUpBuffer != nil {
				t.Fatalf("control changed cancellation/stale behavior: reports=%v opened=%v", f.reports, f.opened)
			}
			if wantReports == 1 && f.reports[0] != errReferenceStale {
				t.Fatalf("control lost the original stale object: %v", f.reports)
			}
			if grouped {
				if !f.buffer.InGroup() || f.buffer.EndGroup() {
					t.Fatal("canceled/outside refusal stole or wrote into the foreign group")
				}
			}
			if control != "row-rebuild" {
				sameMain(t)
			}
			f.tester.Advance(time.Second)
			clickReference(t, f.tester, referenceAccess(t, f.tester, true))
			destination := "target.txt"
			if control == "row-rebuild" {
				destination = "other.txt"
			}
			if len(f.reports) != wantReports || len(f.opened) != 1 || f.opened[0] != destination || f.changes != 0 {
				t.Fatalf("old gesture consumed a fresh successful action: reports=%v opened=%v", f.reports, f.opened)
			}
			if control != "row-rebuild" {
				sameMain(t)
			}
		})
	}
}

func TestMarkdownUIReportingReferenceReportReentry(t *testing.T) {
	for _, replacement := range []string{"same-view-successor", "new-host-successor"} {
		t.Run(replacement, func(t *testing.T) {
			f := newMarkdownActionFixture(t, false)
			oldBuffer := f.buffer
			version, history, selection := oldBuffer.Version(), oldBuffer.History(), oldBuffer.Selection()
			info := referenceAccess(t, f.tester, true)
			if err := oldBuffer.BeginGroup(); err != nil {
				t.Fatal(err)
			}
			var successor *editor.Editor
			var sameMain, sameOwner, sameSuccessor func(*testing.T)
			f.onReport = func(err error) {
				if err != errReferenceStale || f.view.referenceActivation.buffer != nil || f.view.referencePointerUpBuffer != nil {
					t.Fatal("pointer Report arrived before old gesture bookkeeping was cleared")
				}
				if replacement == "new-host-successor" {
					// Real Release invalidates the old gesture before mounting another
					// host. The original foreign group remains owned by its caller.
					f.view.Release()
					f.buffer = text.New(markdownActionSource)
					f.view.host.Buffer = f.buffer
				} else if !oldBuffer.InGroup() || oldBuffer.EndGroup() {
					// Explicit callback-owned retirement permits this same host's
					// new cell to open; the old input handler must not do it itself.
					t.Fatal("callback did not retire an unchanged foreign group")
				}
				sameMain = markdownActionBufferReceipt(f.buffer)
				successor = f.installSuccessor(t)
				sameOwner = f.ownerReceipt(t, successor)
				sameSuccessor = markdownActionBufferReceipt(successor.Buffer)
			}
			x, y := info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2
			f.tester.Press(x, y)
			f.tester.Release(x, y)
			if successor == nil || sameMain == nil || sameOwner == nil || sameSuccessor == nil ||
				len(f.reports) != 1 || f.reports[0] != errReferenceStale || len(f.opened) != 0 || f.changes != 0 {
				t.Fatalf("terminal pointer Report was repeated or did not reenter: reports=%v opened=%v", f.reports, f.opened)
			}
			sameMain(t)
			sameOwner(t)
			sameSuccessor(t)
			f.focusSuccessor(t)
			sameMain(t)
			sameOwner(t)
			sameSuccessor(t)
			if oldBuffer.String() != markdownActionSource || oldBuffer.Version() != version || oldBuffer.History() != history ||
				oldBuffer.Selection() != selection || oldBuffer.InGroup() != (replacement == "new-host-successor") {
				t.Fatal("old pointer tail changed original source/history/selection/group ownership")
			}
			f.independentSuccessorHistory(t, successor)
			if len(f.reports) != 1 || len(f.opened) != 0 || f.changes != 0 {
				t.Fatal("old reference action continued after replacement")
			}
		})
	}
}
