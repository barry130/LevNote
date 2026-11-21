package ui

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A1.1 tests exercise the production controller with direct clients and with
// the actual adapter installed by HandleTextInput. They do not emulate Win32
// message provenance or the later A1.2 public no-frame fixture.
type cancelText struct {
	text                                   string
	mark                                   TextInputRange
	marked                                 bool
	rejectPreedit, rejectFinal, settleEnd  bool
	trace                                  []string
	lastReplacement                        *TextInputRange
	owner                                  *cancelOptional
	onPreedit, onReplace, onUnmark, onMark func()
}

func (c *cancelText) TextForRange(r TextInputRange) (string, TextInputRange) {
	return c.text, TextInputRange{End: UTF16Len(c.text)}
}
func (c *cancelText) Selection() TextInputSelection {
	return TextInputSelection{Range: TextInputRange{End: UTF16Len(c.text)}}
}
func (c *cancelText) MarkedRange() (TextInputRange, bool) {
	mark, marked := c.mark, c.marked
	if c.onMark != nil {
		c.onMark()
	}
	return mark, marked
}
func (c *cancelText) BoundsForRange(r TextInputRange) (Rect, TextInputRange, bool) {
	return Rect{W: 1, H: 16}, r, true
}
func (c *cancelText) IndexForPoint(Point) (int, bool) { return 0, true }
func (c *cancelText) SetMarkedText(r *TextInputRange, value string, selected TextInputRange) {
	c.trace = append(c.trace, "preedit")
	if c.owner != nil && !c.owner.stamp.HasTarget {
		c.owner.stamp.Generation++
		c.owner.stamp.HasTarget = true
	}
	if !c.rejectPreedit {
		c.text = value
		c.mark = TextInputRange{Start: 3, End: 3 + UTF16Len(value)}
		c.marked = true
	}
	if c.onPreedit != nil {
		c.onPreedit()
	}
}
func (c *cancelText) ReplaceText(r *TextInputRange, value string) {
	c.trace = append(c.trace, "replace")
	if r != nil {
		copyRange := *r
		c.lastReplacement = &copyRange
	} else {
		c.lastReplacement = nil
	}
	if !c.rejectFinal {
		c.text, c.marked = value, false
		if c.owner != nil && c.owner.stamp.HasTarget {
			c.owner.stamp.Generation++
			c.owner.stamp.HasTarget = false
		}
	}
	if c.onReplace != nil {
		c.onReplace()
	}
}
func (c *cancelText) UnmarkText() {
	c.trace = append(c.trace, "unmark")
	if c.owner == nil || c.settleEnd {
		c.marked = false
		if c.owner != nil && c.owner.stamp.HasTarget {
			c.owner.stamp.Generation++
			c.owner.stamp.HasTarget = false
		}
	}
	if c.onUnmark != nil {
		c.onUnmark()
	}
}

type cancelOptional struct {
	*cancelText
	stamp                     CompositionCancelStamp
	unavailable, refuseCancel bool
	cancels                   int
	onCancel, onQuery         func()
}

func (c *cancelOptional) CompositionCancelState() (CompositionCancelStamp, CompositionCancelAvailability) {
	if c.onQuery != nil {
		c.onQuery()
	}
	if c.unavailable {
		return c.stamp, CancelUnavailable
	}
	return c.stamp, CancelReady
}
func (c *cancelOptional) CancelComposition(expected CompositionCancelStamp) (CompositionCancelStamp, bool) {
	if c.refuseCancel || c.unavailable || c.stamp != expected {
		return c.stamp, false
	}
	c.cancels++
	c.trace = append(c.trace, "cancel")
	c.text, c.marked = "original", false
	if c.stamp.HasTarget {
		c.stamp.Generation++
	}
	c.stamp.HasTarget = false
	after := c.stamp
	if c.onCancel != nil {
		c.onCancel()
	}
	return after, true // no callback-exit write
}

// Rect/point geometry is the only difference between direct platform and UI clients.
type directCancelText struct{ *cancelText }

func (c *directCancelText) BoundsForRange(r platform.TextRange) (platform.RectF, platform.TextRange, bool) {
	return platform.RectF{W: 1, H: 16}, r, true
}
func (c *directCancelText) IndexForPoint(_, _ float64) (int, bool) { return 0, true }

type directCancelOptional struct{ *cancelOptional }

func (c *directCancelOptional) BoundsForRange(r platform.TextRange) (platform.RectF, platform.TextRange, bool) {
	return platform.RectF{W: 1, H: 16}, r, true
}
func (c *directCancelOptional) IndexForPoint(_, _ float64) (int, bool) { return 0, true }

type cancelEndpoint struct {
	base     *cancelText
	optional *cancelOptional
	client   platform.TextInputClient
	tester   *Tester
}

func makeCancelEndpoint(t *testing.T, adapted, optional bool) cancelEndpoint {
	t.Helper()
	base := &cancelText{text: "original"}
	out := cancelEndpoint{base: base}
	var uiClient TextInputClient = base
	if optional {
		out.optional = &cancelOptional{cancelText: base}
		base.owner = out.optional
		uiClient = out.optional
	}
	if adapted {
		out.tester = NewTester(func(c *Context) {
			Box(c).MinWidth(160).MinHeight(60).Label("cancel target").Focusable().HandleTextInput(uiClient)
		}, 260, 160)
		if err := out.tester.Click("cancel target"); err != nil {
			t.Fatal(err)
		}
		adapter, ok := out.tester.h.ime.Client.(*textInputAdapter)
		if !ok || adapter == nil {
			t.Fatal("actual installed native adapter missing")
		}
		actual, _ := adapter.client()
		if actual != uiClient {
			t.Fatal("adapter is not bound to the initiating client")
		}
		out.client = adapter
	} else if optional {
		out.client = &directCancelOptional{out.optional}
	} else {
		out.client = &directCancelText{base}
	}
	return out
}

func startPreedit(t *testing.T, controller *platform.InputComposition, endpoint cancelEndpoint, value string) platform.CompositionSession {
	t.Helper()
	controller.SetClient(endpoint.client)
	controller.Start(endpoint.client)
	if !controller.Preedit(endpoint.client, nil, value, TextInputRange{}) {
		t.Fatal("preedit was not dispatched/bound")
	}
	return controller.Capture()
}

func TestLevNoteNativeCancelTransport(t *testing.T) {
	t.Run("legal-initiating-client-preedit-acquisition", cancelCorrectedAcquisitionFixture)
	t.Run("observed-owner-negative-control", cancelCorrectedObservedOwnerFixture)
	for _, adapted := range []bool{false, true} {
		mode := "direct"
		if adapted {
			mode = "adapted"
		}
		t.Run(mode, func(t *testing.T) {
			t.Run("cancel-soft-reset-late-before-and-after-End", func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				var c platform.InputComposition
				startPreedit(t, &c, e, "preedit")
				if !c.Cancel(e.client) || e.optional.cancels != 1 || e.base.marked {
					t.Fatal("optional cancellation not completed")
				}
				c.Reset()
				if c.Replace(e.client, nil, "late") || c.End(e.client) || c.Replace(e.client, nil, "later") {
					t.Fatal("cancelled session delivered a late mutation")
				}
				if c.Cancel(e.client) || e.optional.cancels != 1 || e.base.text != "original" {
					t.Fatal("duplicate Cancel or old tail changed text")
				}
			})
			t.Run("new-START-result-only", func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				var c platform.InputComposition
				old := startPreedit(t, &c, e, "preedit")
				if !c.Cancel(e.client) {
					t.Fatal("cancel")
				}
				c.Start(e.client)
				if c.CancelFor(old, e.client) || c.ReplaceFor(old, e.client, nil, "old") {
					t.Fatal("old session regained authority")
				}
				if !c.Replace(e.client, nil, "new") || e.base.text != "new" {
					t.Fatal("new START did not admit normal result-only")
				}
			})
			for _, value := range []string{"new preedit", ""} {
				name := "nonempty"
				if value == "" {
					name = "empty"
				}
				t.Run("new-flagged-preedit-without-START-"+name, func(t *testing.T) {
					e := makeCancelEndpoint(t, adapted, true)
					var c platform.InputComposition
					startPreedit(t, &c, e, "old")
					if !c.Cancel(e.client) {
						t.Fatal("cancel")
					}
					if !c.Preedit(e.client, nil, value, TextInputRange{}) || !c.Replace(e.client, nil, "final") || e.base.text != "final" {
						t.Fatal("flagged preedit did not restart")
					}
				})
			}
			t.Run("dual-flags-result-first-then-new-COMP", func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				var c platform.InputComposition
				startPreedit(t, &c, e, "old")
				if !c.Cancel(e.client) {
					t.Fatal("cancel")
				}
				before := len(e.base.trace)
				if c.Replace(e.client, nil, "old result") || !c.SuppressesResult(e.client) {
					t.Fatal("old RESULT was not quarantined")
				}
				if len(e.base.trace) != before {
					t.Fatal("suppressed RESULT reached the client")
				}
				if !c.Preedit(e.client, nil, "", TextInputRange{}) || !c.Replace(e.client, nil, "new final") {
					t.Fatal("new flagged COMP/final")
				}
			})
			t.Run("normal-End-before-final-and-live-START", func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				e.base.settleEnd = true
				var c platform.InputComposition
				h := startPreedit(t, &c, e, "🙂")
				mark := e.base.mark
				c.Start(e.client)
				if !c.Current(h, e.client) {
					t.Fatal("START retired a live owner")
				}
				if !c.End(e.client) {
					t.Fatal("normal End")
				}
				c.Start(e.client)
				if !c.Current(h, e.client) {
					t.Fatal("START discarded End->final span")
				}
				if !c.Replace(e.client, nil, "accepted") || e.base.lastReplacement == nil || *e.base.lastReplacement != mark {
					t.Fatal("End->final range lost")
				}
			})
			t.Run("multi-End-keeps-target-and-short-final-retry", func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				var c platform.InputComposition
				startPreedit(t, &c, e, "x")
				if !c.End(e.client) || !e.optional.stamp.HasTarget {
					t.Fatal("multi End settled owner")
				}
				e.base.rejectFinal = true
				if !c.Replace(e.client, nil, "refused") || !e.optional.stamp.HasTarget {
					t.Fatal("refusal lost retry")
				}
				e.base.rejectFinal = false
				if !c.Replace(e.client, nil, "short") || e.base.text != "short" {
					t.Fatal("retry failed")
				}
			})
			t.Run("first-refusal-no-mark-still-target", func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				e.base.rejectPreedit = true
				var c platform.InputComposition
				startPreedit(t, &c, e, "too large")
				if e.base.marked || !e.optional.stamp.HasTarget || e.base.text != "original" {
					t.Fatal("first-refusal target fixture")
				}
				if !c.Cancel(e.client) || e.optional.stamp.HasTarget || e.optional.cancels != 1 {
					t.Fatal("markless target not cancelled")
				}
			})
			t.Run("idle-START-versus-no-session", func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				var c platform.InputComposition
				c.SetClient(e.client)
				if c.Cancel(e.client) {
					t.Fatal("no-session idle cancel adopted a target")
				}
				c.Start(e.client)
				if !c.Cancel(e.client) || e.base.text != "original" {
					t.Fatal("verified idle START cancel failed")
				}
			})
			t.Run("unavailable-and-handled-false-no-fallback", func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				var c platform.InputComposition
				startPreedit(t, &c, e, "keep")
				before := append([]string(nil), e.base.trace...)
				e.optional.refuseCancel = true
				if c.Cancel(e.client) || !reflect.DeepEqual(before, e.base.trace) || e.base.text != "keep" {
					t.Fatal("handled=false fell back or edited")
				}
				e.optional.refuseCancel = false
				e.optional.unavailable = true
				if c.Cancel(e.client) || !reflect.DeepEqual(before, e.base.trace) {
					t.Fatal("unavailable fell back")
				}
				e.optional.unavailable = false
				if !c.Replace(e.client, nil, "own final") {
					t.Fatal("refused cancellation suppressed valid final")
				}
			})
			t.Run("failed-Cancel-keeps-End-pending-span", func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				var c platform.InputComposition
				startPreedit(t, &c, e, "kept")
				mark := e.base.mark
				if !c.End(e.client) {
					t.Fatal("End fixture")
				}
				e.optional.refuseCancel = true
				if c.Cancel(e.client) {
					t.Fatal("refused Cancel handled")
				}
				e.optional.refuseCancel = false
				if !c.Replace(e.client, nil, "own final") || e.base.lastReplacement == nil || *e.base.lastReplacement != mark {
					t.Fatal("failed Cancel consumed pending span")
				}
			})
			t.Run("End-callback-successor-keeps-own-span", func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				e.base.settleEnd = true
				var c platform.InputComposition
				startPreedit(t, &c, e, "old")
				e.base.onUnmark = func() {
					e.base.onUnmark = nil
					c.Start(e.client)
					if !c.Preedit(e.client, nil, "new", TextInputRange{}) {
						t.Fatal("new preedit")
					}
				}
				if c.End(e.client) {
					t.Fatal("old End adopted a successor")
				}
				if !c.Replace(e.client, nil, "new final") || e.base.lastReplacement != nil {
					t.Fatal("old End span overwrote successor")
				}
			})
			t.Run("normal-result-only-without-START", func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				var c platform.InputComposition
				c.SetClient(e.client)
				if !c.Replace(e.client, nil, "result") || e.base.text != "result" {
					t.Fatal("normal result-only regressed")
				}
			})
			t.Run("callback-created-successor-survives-old-Cancel", func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				var c platform.InputComposition
				startPreedit(t, &c, e, "old")
				e.optional.onCancel = func() {
					e.optional.onCancel = nil
					c.Start(e.client)
					if !c.Preedit(e.client, nil, "successor", TextInputRange{}) {
						t.Fatal("successor preedit")
					}
				}
				if c.Cancel(e.client) {
					t.Fatal("old callback installed suppression over successor")
				}
				if !e.optional.stamp.HasTarget || e.base.text != "successor" || !c.Replace(e.client, nil, "successor final") {
					t.Fatal("old tail destroyed successor")
				}
			})
			t.Run("preedit-callback-target-ABA-not-acquired", func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				var c platform.InputComposition
				c.SetClient(e.client)
				c.Start(e.client)
				e.base.onPreedit = func() { e.optional.stamp.Generation += 2; e.base.text = "replacement" }
				if c.Preedit(e.client, nil, "old", TextInputRange{}) || c.Cancel(e.client) {
					t.Fatal("callback replacement owner was acquired")
				}
				if e.base.text != "replacement" || e.optional.cancels != 0 {
					t.Fatal("old caller changed replacement")
				}
			})
			t.Run("client-attachment-ABA", func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				var c platform.InputComposition
				old := startPreedit(t, &c, e, "old")
				c.SetClient(nil)
				c.SetClient(e.client)
				if c.CancelFor(old, e.client) || c.EndFor(old, e.client) || c.ReplaceFor(old, e.client, nil, "old") {
					t.Fatal("attachment ABA accepted old handle")
				}
			})
			t.Run("getter-callback-replaces-attachment", func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				other := makeCancelEndpoint(t, false, true)
				var c platform.InputComposition
				startPreedit(t, &c, e, "old")
				e.optional.onQuery = func() { e.optional.onQuery = nil; c.SetClient(other.client); c.Start(other.client) }
				if c.Cancel(e.client) {
					t.Fatal("state getter replacement was ignored")
				}
				if !c.Preedit(other.client, nil, "new", TextInputRange{}) || !c.Replace(other.client, nil, "new final") {
					t.Fatal("getter successor was overwritten")
				}
			})
		})
		for _, marked := range []bool{false, true} {
			for _, order := range []string{"replace-end", "end-replace"} {
				name := mode + "-legacy-" + order
				if marked {
					name += "-marked"
				} else {
					name += "-markless"
				}
				t.Run(name, func(t *testing.T) {
					e := makeCancelEndpoint(t, adapted, false)
					e.base.marked, e.base.mark = marked, TextInputRange{Start: 2, End: 4}
					var c platform.InputComposition
					c.SetClient(e.client)
					c.Start(e.client) // legacy START must not change dispatch/pending
					e.base.trace = nil
					c.Cancel(e.client)
					want := []string{}
					if marked {
						want = []string{"preedit", "unmark"}
					}
					if order == "replace-end" {
						c.Replace(e.client, nil, "ordinary")
						c.End(e.client)
						want = append(want, "replace", "unmark")
					} else {
						c.End(e.client)
						c.Replace(e.client, nil, "ordinary")
						want = append(want, "unmark", "replace")
					}
					if !reflect.DeepEqual(e.base.trace, want) || e.base.text != "ordinary" {
						t.Fatalf("legacy trace=%v want=%v text=%q", e.base.trace, want, e.base.text)
					}
				})
			}
		}
	}
	t.Run("adapted-focus-ABA-and-original-adapter", func(t *testing.T) {
		e := makeCancelEndpoint(t, true, true)
		var c platform.InputComposition
		startPreedit(t, &c, e, "old")
		e.optional.onCancel = func() { e.optional.onCancel = nil; e.tester.SetFocused(false); e.tester.SetFocused(true) }
		if c.Cancel(e.client) {
			t.Fatal("adapter focus ABA passed post-call lease")
		}
		if c.SuppressesResult(e.client) {
			t.Fatal("failed adapter cancellation installed suppression")
		}
	})
	t.Run("unmigrated-backend-End-Replace-remain-compatible", func(t *testing.T) {
		e := makeCancelEndpoint(t, false, true)
		e.base.settleEnd = true
		e.base.SetMarkedText(nil, "old", TextInputRange{})
		mark := e.base.mark
		var c platform.InputComposition // Linux still uses no SetClient/Start
		c.End(e.client)
		c.Replace(e.client, nil, "normal final")
		if e.base.text != "normal final" || e.base.lastReplacement == nil || *e.base.lastReplacement != mark {
			t.Fatal("ordinary backend End/Replace was gated by new opt-in sessions")
		}
	})
	t.Run("new-ready-stamp-retires-only-old-isolation", func(t *testing.T) {
		e := makeCancelEndpoint(t, false, true)
		var c platform.InputComposition
		old := startPreedit(t, &c, e, "old")
		if !c.Cancel(e.client) {
			t.Fatal("cancel")
		}
		e.optional.stamp.Generation += 2
		if c.CanReplace(e.client) || c.SuppressesResult(e.client) || c.CancelFor(old, e.client) {
			t.Fatal("old isolation or handle survived a new ready stamp")
		}
		if !c.Replace(e.client, nil, "new result") {
			t.Fatal("new idle result route stayed suppressed")
		}
	})
	t.Run("direct-legacy-pending-explicit-priority-and-Reset", func(t *testing.T) {
		e := makeCancelEndpoint(t, false, false)
		var c platform.InputComposition
		e.base.marked, e.base.mark = true, TextInputRange{Start: 2, End: 5}
		c.End(e.client)
		explicit := TextInputRange{Start: 7, End: 8}
		c.Start(e.client)
		c.Replace(e.client, &explicit, "x")
		if e.base.lastReplacement == nil || *e.base.lastReplacement != explicit {
			t.Fatal("explicit range lost priority")
		}
		e.base.marked = true
		c.End(e.client)
		c.Reset()
		c.Replace(e.client, nil, "y")
		if e.base.lastReplacement != nil {
			t.Fatal("Reset retained pending span")
		}
	})
}

// Test-only observation of the controller's pending record. Behavioral range
// delivery is checked separately; this also proves a refused action did not
// clear or replace a successor's private pending state.
func cancelPendingState(c *platform.InputComposition) (bool, TextInputRange) {
	field := reflect.ValueOf(c).Elem().FieldByName("pending")
	if field.IsNil() {
		return false, TextInputRange{}
	}
	r := field.Elem()
	return true, TextInputRange{Start: int(r.FieldByName("Start").Int()), End: int(r.FieldByName("End").Int())}
}

func TestLevNoteNativeCancelForeignTargetEndNeverAcquires(t *testing.T) {
	for _, adapted := range []bool{false, true} {
		mode := "direct"
		if adapted {
			mode = "installed-adapter"
		}
		for _, markless := range []bool{false, true} {
			for _, settleEnd := range []bool{false, true} {
				name := mode + "/marked"
				if markless {
					name = mode + "/markless-first-refusal"
				}
				if settleEnd {
					name += "/settling-Unmark"
				} else {
					name += "/retained-Unmark"
				}
				t.Run(name, func(t *testing.T) {
					e := makeCancelEndpoint(t, adapted, true)
					e.base.rejectPreedit, e.base.settleEnd = markless, settleEnd
					// This owner was created outside this controller. Merely seeing
					// Gen1 at START must never turn it into transport authority.
					e.client.SetMarkedText(nil, "foreign", TextInputRange{})
					var c platform.InputComposition
					c.SetClient(e.client)
					h := c.Start(e.client)
					stamp, text, marked, mark := e.optional.stamp, e.base.text, e.base.marked, e.base.mark
					trace := append([]string(nil), e.base.trace...)
					if c.Cancel(e.client) || c.End(e.client) || c.Cancel(e.client) || c.Preedit(e.client, nil, "retry", TextInputRange{}) || c.Replace(e.client, nil, "late") || c.EndFor(h, e.client) {
						t.Fatal("observed target was acquired by a refused End/retry")
					}
					if e.optional.stamp != stamp || e.base.text != text || e.base.marked != marked || e.base.mark != mark || e.optional.cancels != 0 || !reflect.DeepEqual(e.base.trace, trace) || !c.Current(h, e.client) {
						t.Fatal("foreign owner or transport session changed on rejection")
					}
					if pending, _ := cancelPendingState(&c); pending || c.SuppressesResult(e.client) {
						t.Fatal("foreign End created pending/suppression")
					}
					// Its original application caller remains usable; the rejecting
					// controller neither cancelled it nor stole its final route.
					e.client.ReplaceText(nil, "owner final")
					if e.base.text != "owner final" || e.optional.stamp.HasTarget {
						t.Fatal("original caller could not finish its owner")
					}
				})
			}
		}
		for _, markless := range []bool{false, true} {
			for _, endFirst := range []bool{false, true} {
				name := mode + "/owned-marked"
				if markless {
					name = mode + "/owned-markless"
				}
				if endFirst {
					name += "/End-Cancel"
				} else {
					name += "/Cancel-End"
				}
				t.Run(name, func(t *testing.T) {
					e := makeCancelEndpoint(t, adapted, true)
					e.base.rejectPreedit = markless
					var c platform.InputComposition
					startPreedit(t, &c, e, "owned")
					if endFirst && !c.End(e.client) {
						t.Fatal("owned retained End was refused")
					}
					if !c.Cancel(e.client) || e.optional.cancels != 1 {
						t.Fatal("owned marked/markless cancellation was lost")
					}
					if c.End(e.client) || c.Replace(e.client, nil, "late") || e.base.text != "original" {
						t.Fatal("old End/result escaped owned cancellation")
					}
				})
			}
		}
	}
}

func TestLevNoteNativeCancelEndGetterOwnerABA(t *testing.T) {
	for _, adapted := range []bool{false, true} {
		mode := "direct"
		if adapted {
			mode = "installed-adapter"
		}
		t.Run(mode, func(t *testing.T) {
			e := makeCancelEndpoint(t, adapted, true)
			var c platform.InputComposition
			startPreedit(t, &c, e, "old")
			if !c.End(e.client) {
				t.Fatal("owned pending fixture")
			}
			pendingBefore, rangeBefore := cancelPendingState(&c)
			if !pendingBefore {
				t.Fatal("fixture has no retained End span")
			}
			e.base.settleEnd = true
			e.base.onMark = func() {
				e.base.onMark = nil
				// Gen1 is retired and Gen3 is a different application owner.
				// No controller/attachment call occurs in this getter callback.
				e.optional.stamp.Generation += 2
				e.optional.stamp.HasTarget = true
				e.base.text, e.base.marked, e.base.mark = "successor text", true, TextInputRange{Start: 9, End: 11}
			}
			trace := append([]string(nil), e.base.trace...)
			if c.End(e.client) || c.Cancel(e.client) || c.Replace(e.client, nil, "old final") {
				t.Fatal("getter-created successor was acted on")
			}
			pendingAfter, rangeAfter := cancelPendingState(&c)
			if pendingAfter != pendingBefore || rangeAfter != rangeBefore || e.optional.stamp.Generation != 3 || !e.optional.stamp.HasTarget || e.base.text != "successor text" || !e.base.marked || e.optional.cancels != 0 || !reflect.DeepEqual(e.base.trace, trace) {
				t.Fatal("old End tail unmarked/cancelled/replaced successor or changed pending")
			}
			e.client.ReplaceText(nil, "successor final")
			if e.base.text != "successor final" {
				t.Fatal("successor's own caller was not usable")
			}
		})
	}
}

func TestLevNoteNativeCancelScopedLegacyReentry(t *testing.T) {
	for _, adapted := range []bool{false, true} {
		mode := "direct"
		if adapted {
			mode = "installed-adapter"
		}
		for _, aba := range []bool{false, true} {
			for _, initiallyScoped := range []bool{false, true} {
				for _, phase := range []string{"End-getter", "Cancel-getter", "Cancel-preedit", "End-unmark", "Replace"} {
					name := mode + "/" + phase
					if aba {
						name += "/A-B-A"
					} else {
						name += "/A-B"
					}
					if initiallyScoped {
						name += "/scoped"
					} else {
						name += "/initially-unconfigured"
					}
					t.Run(name, func(t *testing.T) {
						a, b := makeCancelEndpoint(t, adapted, false), makeCancelEndpoint(t, adapted, false)
						a.base.marked, a.base.mark = true, TextInputRange{Start: 0, End: 1}
						b.base.marked, b.base.mark = true, TextInputRange{Start: 4, End: 6}
						var c platform.InputComposition
						if initiallyScoped {
							c.SetClient(a.client)
						}
						if phase == "Replace" {
							c.End(a.client)
						}
						target, expected := b, b.base.mark
						var exitA, exitB []string
						transitioned := false
						transition := func() {
							a.base.onMark, a.base.onPreedit, a.base.onUnmark, a.base.onReplace = nil, nil, nil, nil
							c.SetClient(b.client)
							if !c.End(b.client) {
								t.Fatal("nested B End")
							}
							if aba {
								c.SetClient(a.client)
								a.base.text = "successor text"
								a.base.marked, a.base.mark = true, TextInputRange{Start: 8, End: 10}
								target, expected = a, a.base.mark
								if !c.End(a.client) {
									t.Fatal("nested successor A End")
								}
							}
							transitioned = true
							exitA, exitB = append([]string(nil), a.base.trace...), append([]string(nil), b.base.trace...)
						}
						switch phase {
						case "End-getter", "Cancel-getter":
							a.base.onMark = transition
						case "Cancel-preedit":
							a.base.onPreedit = transition
						case "End-unmark":
							a.base.onUnmark = transition
						case "Replace":
							a.base.onReplace = transition
						}
						var delivered bool
						switch phase {
						case "End-getter", "End-unmark":
							delivered = c.End(a.client)
						case "Cancel-getter", "Cancel-preedit":
							delivered = c.Cancel(a.client)
						case "Replace":
							delivered = c.Replace(a.client, nil, "old action")
						}
						if !transitioned || delivered || !reflect.DeepEqual(a.base.trace, exitA) || !reflect.DeepEqual(b.base.trace, exitB) {
							t.Fatal("stale legacy tail continued after scope transfer")
						}
						if pending, r := cancelPendingState(&c); !pending || r != expected {
							t.Fatal("old scope overwrote successor pending", pending, r, expected)
						}
						if !c.Replace(target.client, nil, "target final") || target.base.lastReplacement == nil || *target.base.lastReplacement != expected {
							t.Fatal("successor did not receive its own pending range")
						}
					})
				}
			}
		}
	}
}

// BEGIN TASK92 SHARED LEGAL ACQUISITION FIXTURES
// These fixtures use only API signatures supplied by the independent typed-API
// snapshot. Ownership starts idle and is acquired through its actual Preedit
// call; a pre-existing target is a separate negative control, never a positive.
type cancelCorrectedClient struct {
	text                                     string
	generation                               uint64
	hasTarget, marked, refusePreedit         bool
	preedits, unmarks, replacements, cancels int
}

func (c *cancelCorrectedClient) TextForRange(r platform.TextRange) (string, platform.TextRange) {
	return c.text, platform.TextRange{End: platform.UTF16Len(c.text)}
}
func (c *cancelCorrectedClient) Selection() platform.TextSelection { return platform.TextSelection{} }
func (c *cancelCorrectedClient) MarkedRange() (platform.TextRange, bool) {
	return platform.TextRange{End: platform.UTF16Len(c.text)}, c.marked
}
func (c *cancelCorrectedClient) BoundsForRange(r platform.TextRange) (platform.RectF, platform.TextRange, bool) {
	return platform.RectF{W: 1, H: 16}, r, true
}
func (c *cancelCorrectedClient) IndexForPoint(_, _ float64) (int, bool) { return 0, true }
func (c *cancelCorrectedClient) SetMarkedText(_ *platform.TextRange, value string, _ platform.TextRange) {
	c.preedits++
	if !c.hasTarget {
		c.generation++
		c.hasTarget = true
	}
	if !c.refusePreedit {
		c.text, c.marked = value, true
	}
}
func (c *cancelCorrectedClient) ReplaceText(_ *platform.TextRange, value string) {
	c.replacements++
	c.text = value
}
func (c *cancelCorrectedClient) UnmarkText() { c.unmarks++; c.marked = false }
func (c *cancelCorrectedClient) CompositionCancelState() (platform.CompositionCancelStamp, platform.CompositionCancelAvailability) {
	return platform.CompositionCancelStamp{Generation: c.generation, HasTarget: c.hasTarget}, platform.CancelReady
}
func (c *cancelCorrectedClient) CancelComposition(expected platform.CompositionCancelStamp) (platform.CompositionCancelStamp, bool) {
	current, _ := c.CompositionCancelState()
	if !c.hasTarget || current != expected {
		return current, false
	}
	c.cancels++
	c.generation++
	c.hasTarget, c.marked, c.text = false, false, "original"
	return platform.CompositionCancelStamp{Generation: c.generation}, true
}

// The application is the same in both snapshots; only the transport changes.
type cancelCorrectedUIClient struct{ *cancelCorrectedClient }

func (c *cancelCorrectedUIClient) BoundsForRange(r TextInputRange) (Rect, TextInputRange, bool) {
	return Rect{W: 1, H: 16}, r, true
}
func (c *cancelCorrectedUIClient) IndexForPoint(Point) (int, bool) { return 0, true }
func cancelCorrectedEndpoint(t *testing.T, adapted, markless bool) (*cancelCorrectedClient, platform.TextInputClient) {
	t.Helper()
	client := &cancelCorrectedClient{text: "original", refusePreedit: markless}
	if !adapted {
		return client, client
	}
	uiClient := &cancelCorrectedUIClient{client}
	tester := NewTester(func(c *Context) {
		Box(c).MinWidth(160).MinHeight(60).Label("legal cancel target").Focusable().HandleTextInput(uiClient)
	}, 260, 160)
	if err := tester.Click("legal cancel target"); err != nil {
		t.Fatal(err)
	}
	adapter, ok := tester.h.ime.Client.(*textInputAdapter)
	if !ok || adapter == nil {
		t.Fatal("real installed adapter missing")
	}
	actual, _ := adapter.client()
	if actual != uiClient {
		t.Fatal("adapter bound to another initiating client")
	}
	return client, adapter
}

func cancelCorrectedAcquisitionFixture(t *testing.T) {
	for _, adapted := range []bool{false, true} {
		mode := "direct"
		if adapted {
			mode = "installed-adapter"
		}
		for _, markless := range []bool{false, true} {
			name := mode + "/accepted-mark"
			if markless {
				name = mode + "/retained-first-refusal-no-mark"
			}
			t.Run(name, func(t *testing.T) {
				client, endpoint := cancelCorrectedEndpoint(t, adapted, markless)
				var controller platform.InputComposition
				controller.SetClient(endpoint)
				controller.Start(endpoint)
				if !controller.Preedit(endpoint, nil, "preedit", platform.TextRange{}) {
					t.Fatal("legal initiating preedit was not delivered")
				}
				before, _ := client.CompositionCancelState()
				if !before.HasTarget || before.Generation != 1 || client.preedits != 1 || client.marked == markless {
					t.Fatal("fixture did not establish the intended application owner")
				}
				controller.Cancel(endpoint)
				if client.cancels != 1 || client.hasTarget || client.marked || client.generation != 2 || client.text != "original" || client.preedits != 1 || client.unmarks != 0 || client.replacements != 0 {
					t.Fatal("legal explicit cancel bypassed optional dispatch or used synthetic legacy mutations")
				}
			})
		}
	}
}

func cancelCorrectedObservedOwnerFixture(t *testing.T) {
	for _, adapted := range []bool{false, true} {
		mode := "direct"
		if adapted {
			mode = "installed-adapter"
		}
		for _, markless := range []bool{false, true} {
			name := mode + "/marked"
			if markless {
				name = mode + "/markless"
			}
			t.Run(name, func(t *testing.T) {
				client, endpoint := cancelCorrectedEndpoint(t, adapted, markless)
				endpoint.SetMarkedText(nil, "foreign", platform.TextRange{})
				before, _ := client.CompositionCancelState()
				text, marked, preedits := client.text, client.marked, client.preedits
				var controller platform.InputComposition
				controller.SetClient(endpoint)
				controller.Start(endpoint)
				controller.Cancel(endpoint)
				controller.End(endpoint)
				controller.Replace(endpoint, nil, "unauthorized late result")
				after, _ := client.CompositionCancelState()
				if after != before || client.text != text || client.marked != marked || client.preedits != preedits || client.unmarks != 0 || client.replacements != 0 || client.cancels != 0 {
					t.Fatal("an observed foreign owner was acquired by End/Cancel/late result")
				}
			})
		}
	}
}

// END TASK92 SHARED LEGAL ACQUISITION FIXTURES

func TestLevNoteNativeCancelLegacySameScopeResetKeepsSequence(t *testing.T) {
	cases := []struct {
		name, phase string
		marked      bool
		want        []string
		wantPending bool
	}{
		{"End-getter-marked", "End-getter", true, []string{"reset-getter", "unmark"}, true},
		{"End-getter-markless", "End-getter", false, []string{"reset-getter", "unmark"}, false},
		{"Cancel-getter-marked", "Cancel-getter", true, []string{"reset-getter", "preedit", "unmark"}, false},
		{"Cancel-getter-markless", "Cancel-getter", false, []string{"reset-getter"}, false},
		{"Cancel-preedit", "Cancel-preedit", true, []string{"preedit", "reset-preedit", "unmark"}, false},
		{"Cancel-unmark", "Cancel-unmark", true, []string{"preedit", "unmark", "reset-unmark"}, false},
		{"End-unmark", "End-unmark", true, []string{"unmark", "reset-unmark"}, false},
		{"Preedit-completion", "Preedit", true, []string{"preedit", "reset-preedit"}, false},
		{"Replace-completion", "Replace", true, []string{"replace", "reset-replace"}, false},
	}
	for _, adapted := range []bool{false, true} {
		mode := "direct"
		if adapted {
			mode = "installed-adapter"
		}
		for _, configured := range []bool{false, true} {
			scope := "initially-unconfigured"
			if configured {
				scope = "configured"
			}
			for _, tt := range cases {
				t.Run(mode+"/"+scope+"/"+tt.name, func(t *testing.T) {
					e := makeCancelEndpoint(t, adapted, false)
					var c platform.InputComposition
					if configured {
						c.SetClient(e.client)
					}
					// A distinct old pending span exposes both Cancel's early clear
					// timing and End's publication of its returned current mark.
					seed := TextInputRange{Start: 0, End: 1}
					e.base.mark, e.base.marked = seed, true
					if !c.End(e.client) {
						t.Fatal("seed End")
					}
					current := TextInputRange{Start: 2, End: 5}
					e.base.text, e.base.mark, e.base.marked = "original", current, tt.marked
					e.base.trace = nil
					handle := c.Capture()
					resets := 0
					reset := func(event string, expectPending bool, expectedRange TextInputRange) {
						t.Helper()
						pending, value := cancelPendingState(&c)
						if pending != expectPending || pending && value != expectedRange {
							t.Fatal("pre-Reset timing/range changed", pending, value, expectedRange)
						}
						e.base.trace = append(e.base.trace, event)
						c.Reset()
						resets++
						if c.Capture() != handle {
							t.Fatal("pending-only Reset changed transport scope")
						}
						if pending, _ := cancelPendingState(&c); pending {
							t.Fatal("Reset did not clear only the pending span")
						}
					}
					switch tt.phase {
					case "End-getter":
						e.base.onMark = func() { e.base.onMark = nil; reset("reset-getter", true, seed) }
					case "Cancel-getter":
						e.base.onMark = func() { e.base.onMark = nil; reset("reset-getter", false, TextInputRange{}) }
					case "Cancel-preedit", "Preedit":
						e.base.onPreedit = func() { e.base.onPreedit = nil; reset("reset-preedit", false, TextInputRange{}) }
					case "Cancel-unmark":
						e.base.onUnmark = func() { e.base.onUnmark = nil; reset("reset-unmark", false, TextInputRange{}) }
					case "End-unmark":
						e.base.onUnmark = func() { e.base.onUnmark = nil; reset("reset-unmark", true, current) }
					case "Replace":
						e.base.onReplace = func() {
							e.base.onReplace = nil
							if e.base.lastReplacement == nil || *e.base.lastReplacement != seed {
								t.Fatal("Replace lost its pre-callback pending range")
							}
							reset("reset-replace", false, TextInputRange{})
						}
					}
					var delivered bool
					switch tt.phase {
					case "End-getter", "End-unmark":
						delivered = c.End(e.client)
					case "Cancel-getter", "Cancel-preedit", "Cancel-unmark":
						delivered = c.Cancel(e.client)
					case "Preedit":
						delivered = c.Preedit(e.client, nil, "new", TextInputRange{})
					case "Replace":
						delivered = c.Replace(e.client, nil, "result")
					}
					if !delivered || resets != 1 || !reflect.DeepEqual(e.base.trace, tt.want) || c.Capture() != handle {
						t.Fatal("same-scope Reset stopped or reordered the original legacy action", delivered, resets, e.base.trace, tt.want)
					}
					pending, value := cancelPendingState(&c)
					if pending != tt.wantPending || pending && value != current {
						t.Fatal("legacy pending outcome changed", pending, value)
					}
					if !c.Replace(e.client, nil, "after Reset") {
						t.Fatal("following legacy Replace was refused")
					}
					if tt.wantPending {
						if e.base.lastReplacement == nil || *e.base.lastReplacement != current {
							t.Fatal("End did not keep its returned marked range for the next result")
						}
					} else if e.base.lastReplacement != nil {
						t.Fatal("Reset-cleared range leaked to the next result")
					}
				})
			}
		}
	}
}

func TestLevNoteNativeCancelOptionalResetRetainsStrongTurnProof(t *testing.T) {
	for _, adapted := range []bool{false, true} {
		mode := "direct"
		if adapted {
			mode = "installed-adapter"
		}
		for _, phase := range []string{"End-state", "Replace-state", "Cancel-state", "Preedit-state", "End-marked-getter"} {
			t.Run(mode+"/"+phase, func(t *testing.T) {
				e := makeCancelEndpoint(t, adapted, true)
				var c platform.InputComposition
				startPreedit(t, &c, e, "owned")
				if !c.End(e.client) {
					t.Fatal("owned pending fixture")
				}
				before, text, mark := e.optional.stamp, e.base.text, e.base.mark
				trace := append([]string(nil), e.base.trace...)
				resets := 0
				reset := func() { c.Reset(); resets++ }
				if phase == "End-marked-getter" {
					e.base.onMark = func() { e.base.onMark = nil; reset() }
				} else {
					queries := 0
					resetAt := 1
					if phase == "Replace-state" {
						resetAt = 2
					} // after the separate CanReplace query
					e.optional.onQuery = func() {
						queries++
						if queries == resetAt {
							e.optional.onQuery = nil
							reset()
						}
					}
				}
				var delivered bool
				switch phase {
				case "End-state", "End-marked-getter":
					delivered = c.End(e.client)
				case "Replace-state":
					delivered = c.Replace(e.client, nil, "old result")
				case "Cancel-state":
					delivered = c.Cancel(e.client)
				case "Preedit-state":
					delivered = c.Preedit(e.client, nil, "old preedit", TextInputRange{})
				}
				if delivered || resets != 1 || e.optional.stamp != before || e.base.text != text || e.base.mark != mark || e.optional.cancels != 0 || !reflect.DeepEqual(e.base.trace, trace) {
					t.Fatal("optional mutation crossed a stale revision proof", delivered, resets)
				}
				if pending, _ := cancelPendingState(&c); pending {
					t.Fatal("Reset itself did not clear pending")
				}
				// Reset is not a cancellation: a later fresh optional action still
				// owns this target and can perform the real explicit cancellation.
				if !c.Cancel(e.client) || e.optional.cancels != 1 {
					t.Fatal("fresh cancellation authority was lost")
				}
			})
		}
	}
}
