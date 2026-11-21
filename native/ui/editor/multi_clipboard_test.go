package editor

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func TestMultiClipboardRevalidatesFullOwner(t *testing.T) {
	for _, action := range []string{"cut", "paste"} {
		for _, change := range []string{"secondary-only", "selection-ABA", "same-bytes-undo", "read-only-ABA", "buffer-ABA", "view-release", "new-composition"} {
			t.Run(action+"/"+change, func(t *testing.T) {
				b := text.New("abc | def")
				e := New(b)
				tester := multiTester(t, e)
				start := setSelections(t, b, []text.Range{{Anchor: 3, Head: 0}, {Anchor: 6, Head: 7}}, 0)
				secondary := text.SelectionSet{Ranges: []text.Range{start.Ranges[0], {Anchor: 9, Head: 8}}, Primary: 0}
				var expected editState
				var reported error
				calls, changes := 0, 0
				e.OnError = func(err error) { reported = err }
				e.OnChange = func() { changes++ }
				mutate := func() {
					t.Helper()
					switch change {
					case "secondary-only", "selection-ABA":
						if err := b.SetSelectionSet(secondary); err != nil {
							t.Fatal(err)
						}
						if change == "selection-ABA" {
							if err := b.SetSelectionSet(start); err != nil {
								t.Fatal(err)
							}
						}
					case "same-bytes-undo":
						if err := b.ReplaceUTF16(8, 9, "!"); err != nil || !b.Undo() || b.String() != "abc | def" {
							t.Fatal("same-bytes ABA fixture", err)
						}
						requireSelections(t, b, start)
					case "read-only-ABA":
						e.SetReadOnly(true)
						e.SetReadOnly(false)
					case "buffer-ABA":
						e.SetBuffer(text.New("abc | def"))
						e.SetBuffer(b)
					case "view-release":
						e.ReleaseView()
					case "new-composition":
						e.nativeClient.SetMarkedText(nil, "abc", ui.TextInputRange{Start: 3, End: 3})
						if !b.OwnsComposition(e.compositionToken) || !errors.Is(e.Flush(), text.ErrCompositionPending) {
							t.Fatal("new owner fixture not active")
						}
					}
					expected = currentEditState(b)
				}
				if action == "cut" {
					var done func(error)
					e.CopyThen = func(value string, complete func(error)) {
						calls++
						if value != "abc" {
							t.Fatalf("cut captured %q", value)
						}
						done = complete
					}
					tester.Command("cut")
					if done == nil || b.String() != "abc | def" {
						t.Fatal("cut did not wait for the clipboard result")
					}
					mutate()
					done(nil)
					done(nil) // a duplicate completion must stay harmless
				} else {
					// The clipboard read can reenter the UI before returning its value.
					e.Paste = func() (string, error) {
						calls++
						mutate()
						return "paste", nil
					}
					tester.Command("pasteAndMatchStyle")
				}
				requireEditState(t, b, expected)
				if b.Selection() != start.Ranges[0] && change != "new-composition" {
					t.Fatal("test fixture changed the primary selection")
				}
				expectedCalls := 0
				if change == "new-composition" {
					expectedCalls = 1
					if !errors.Is(reported, text.ErrCompositionPending) {
						t.Fatalf("clipboard failed to report the new pending owner: %v", reported)
					}
				}
				if calls != 1 || changes != expectedCalls || change != "view-release" && reported == nil {
					t.Fatalf("clipboard outcome: reads/writes %d, changes %d, reported %v", calls, changes, reported)
				}
			})
		}
	}
}

func TestMultiClipboardDeferredWriteDoesNotStartWhilePending(t *testing.T) {
	b := text.New("abc | def")
	e := New(b)
	tester := multiTester(t, e)
	setSelections(t, b, []text.Range{{Anchor: 3, Head: 0}, {Anchor: 6, Head: 7}}, 0)
	tester.SetClipboard("original clipboard")
	if !e.Command("cut") || e.pendingClipboard == nil || e.pendingClipboard.dispatched {
		t.Fatal("expected a queued native clipboard write")
	}
	e.nativeClient.SetMarkedText(nil, "abc", ui.TextInputRange{Start: 3, End: 3})
	before, token := currentEditState(b), e.compositionToken
	var reported error
	e.OnError = func(err error) { reported = err }
	tester.Frame()
	requireEditState(t, b, before)
	if tester.Clipboard() != "original clipboard" || !b.OwnsComposition(token) || !errors.Is(reported, text.ErrCompositionPending) || e.pendingClipboard != nil {
		t.Fatalf("queued write escaped pending state: clipboard %q, %v", tester.Clipboard(), reported)
	}
}

func TestNativeClipboardChecksSettledSameStateComposition(t *testing.T) {
	b := text.New("same")
	e := New(b)
	tester := multiTester(t, e)
	before := currentEditState(b)
	var reported error
	e.OnError = func(err error) { reported = err }
	e.Paste = func() (string, error) {
		e.nativeClient.SetMarkedText(nil, "", ui.TextInputRange{})
		e.nativeClient.UnmarkText()
		requireEditState(t, b, before)
		return "paste", nil
	}
	tester.Command("paste")
	requireEditState(t, b, before)
	if reported == nil {
		t.Fatal("clipboard did not detect the completed same-state composition")
	}
}
