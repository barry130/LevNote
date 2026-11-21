package editor

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

const maxLanguageSourceBytes = 2 << 20

// SourceAction is a host-owned language intent from the actual source input.
// Other project Editors remain unbound unless their host explicitly opts in.
type SourceAction uint8

const (
	SourceActionComplete SourceAction = iota + 1
	SourceActionReferences
	SourceActionRename
)

// SourceActionBinding is an opaque registration and attachment lease. It does
// not capture source text, selections, input readiness or host authorization.
type SourceActionBinding struct {
	editor       *Editor
	owner        inputOwner
	registration *sourceActionRegistration
}

type sourceActionRegistration struct{ marker byte }

// SetSourceActionHandler replaces the UI-thread registration without invoking
// any callback. nil revokes it. Hosts retain the returned lease in their closure
// and rebind only on actual lifecycle transitions. The lease recorded by View
// for registered shortcuts stays tied to the element build that registered them.
func (e *Editor) SetSourceActionHandler(handler func(SourceAction)) SourceActionBinding {
	if e == nil {
		return SourceActionBinding{}
	}
	owner := e.observeInputOwner()
	e.sourceActionBinding = SourceActionBinding{editor: e, owner: owner, registration: &sourceActionRegistration{marker: 1}}
	e.sourceActionHandler = handler
	return e.sourceActionBinding
}

// SourceActionBindingCurrent is a pure identity check. Setter-observed buffer
// and readonly ABA and ReleaseView invalidate the lease; ordinary frames, text
// edits, selection movement and retained input do not. It never observes owners,
// probes input, invokes callbacks or materializes a source snapshot.
func (e *Editor) SourceActionBindingCurrent(binding SourceActionBinding) bool {
	return e != nil && binding.editor == e && binding.registration != nil && binding.owner.buffer != nil &&
		e.sourceActionHandler != nil && binding == e.sourceActionBinding && binding.owner == e.inputOwner &&
		e.Buffer == binding.owner.buffer && e.ReadOnly == binding.owner.readOnly && e.viewGeneration == binding.owner.viewEpoch
}

func sourceActionKey(ev ui.InputEvent) (SourceAction, bool) {
	if ev.Kind != ui.InputKeyDown {
		return 0, false
	}
	switch {
	case ev.Mods == ui.Ctrl && ev.Key == ui.KeySpace:
		return SourceActionComplete, true
	case ev.Mods == ui.Shift && ev.Key == ui.KeyF12:
		return SourceActionReferences, true
	case ev.Mods == 0 && ev.Key == ui.KeyF2:
		return SourceActionRename, true
	}
	return 0, false
}

// dispatchSourceAction owns a recognized bound key even when stale, repeated
// or refused. Both callbacks are final: no old-action writes follow reentry.
func (e *Editor) dispatchSourceAction(binding SourceActionBinding, action SourceAction, repeated bool) {
	if repeated || !e.SourceActionBindingCurrent(binding) {
		return
	}
	if err := e.InputActionError(); err != nil {
		if report := e.OnError; report != nil {
			report(err)
		}
		return
	}
	handler := e.sourceActionHandler
	handler(action)
}

var (
	errSourceSnapshotBudget = errors.New("source snapshot exceeds its positive byte budget or the 2 MiB hard limit")
	errSourceTargetStale    = errors.New("source target is stale or belongs to another editor")
)

// SourceTarget is an opaque identity issued by one Editor. It binds an
// attachment, revision, composition state and, when requested, the full directed
// selection set. Callers must use SetBuffer and SetReadOnly for observed ABA.
type SourceTarget struct {
	issuer     *sourceTargetIssuer
	editor     *Editor
	stamp      inputStamp
	selections text.SelectionSet
}

type sourceTargetIssuer struct {
	editor *Editor
	marker byte
}

// SourceSnapshot owns one bounded materialization of the original Buffer text.
type SourceSnapshot struct {
	Source     string
	Version    uint64
	Selections text.SelectionSet
	Target     SourceTarget
}

// CaptureSourceSnapshot is UI-thread-only. It captures logical UTF-8 bytes, not
// disk encoding metadata, and never adds a BOM or normalizes line endings.
func (e *Editor) CaptureSourceSnapshot(maxBytes int) (SourceSnapshot, error) {
	if e == nil || e.Buffer == nil {
		return SourceSnapshot{}, errSourceTargetStale
	}
	if maxBytes <= 0 {
		return SourceSnapshot{}, errSourceSnapshotBudget
	}
	if err := e.sourceReadAdmission(); err != nil {
		return SourceSnapshot{}, err
	}

	buffer := e.Buffer
	limit := min(maxBytes, maxLanguageSourceBytes)
	if buffer.LenBytes() > limit {
		return SourceSnapshot{}, fmt.Errorf("%w: source is %d bytes, limit is %d", errSourceSnapshotBudget, buffer.LenBytes(), limit)
	}

	stamp := e.inputStamp()
	snapshot := buffer.Snapshot()
	source := snapshot.String()
	selections := snapshot.SelectionSet()
	targetSelections := cloneSourceSelectionSet(selections)
	target := SourceTarget{
		issuer:     &sourceTargetIssuer{editor: e, marker: 1},
		editor:     e,
		stamp:      stamp,
		selections: targetSelections,
	}
	return SourceSnapshot{
		Source:     source,
		Version:    stamp.version,
		Selections: cloneSourceSelectionSet(selections),
		Target:     target,
	}, nil
}

// CheckSourceTarget is UI-thread-only. Ordinary selection changes are ignored
// unless requireSelection is true; text, attachment, readonly, view and
// composition identities are always checked.
func (e *Editor) CheckSourceTarget(t SourceTarget, requireSelection bool) error {
	if err := e.sourceReadAdmission(); err != nil {
		return err
	}
	if t.issuer == nil || t.issuer.marker == 0 || t.issuer.editor != e || t.editor != e || t.stamp.owner.buffer == nil {
		return errSourceTargetStale
	}

	// Capture may observe legacy assignments, but validation must leave even
	// stale owner metadata untouched. Compare both the recorded attachment
	// epochs (including setter ABA) and the current compatibility fields.
	owner := t.stamp.owner
	if e.inputOwner != owner || e.Buffer != owner.buffer || e.ReadOnly != owner.readOnly || e.viewGeneration != owner.viewEpoch ||
		e.Buffer.Version() != t.stamp.version || e.compositionGeneration != t.stamp.compositionGen ||
		e.compositionToken != t.stamp.token || e.Buffer.InGroup() != t.stamp.group {
		return errSourceTargetStale
	}
	if requireSelection {
		if e.Buffer.SelectionGeneration() != t.stamp.selectionGen || !equalSourceSelectionSet(e.Buffer.SelectionSet(), t.selections) {
			return errSourceTargetStale
		}
	}
	return nil
}

// RevealSourceTarget prepares navigation without publishing provisional state.
// Highlight and long-line OnError run against the real entry Editor. Each host
// callback must return to the same UI-thread boundary before preparation can
// continue; a stale call discards only its private plan, never callback work.
// Ordinary selection movement since token capture is allowed, but the complete
// directed selection and generation at THIS call's entry are protected.
//
// As with SourceTarget, unobserved public-field/function-closure ABA is outside
// this contract. An empty Buffer BeginGroup/EndGroup has no exposed epoch either.
// No concurrent goroutine mutation is supported by this UI-thread API.
func (e *Editor) RevealSourceTarget(t SourceTarget, start, end int) error {
	return e.RevealSourceTargetChecked(t, start, end, nil)
}

// RevealSourceTargetChecked adds an optional request check to RevealSourceTarget.
// stillCurrent must be a pure predicate of existing UI-owned state. It must not
// flush input, mutate state, capture source snapshots or inspect background-owned
// state. It runs on the UI thread at entry, after each synchronous navigation
// callback and immediately before commit, and is never retained. A false result
// returns the stale-target error and preserves any work done by callbacks.
func (e *Editor) RevealSourceTargetChecked(t SourceTarget, start, end int, stillCurrent func() bool) error {
	if stillCurrent != nil && !stillCurrent() {
		return errSourceTargetStale
	}
	if err := e.CheckSourceTarget(t, false); err != nil {
		return err
	}
	buffer := e.Buffer
	if _, err := buffer.SliceUTF16(start, start); err != nil {
		return err
	}
	if _, err := buffer.SliceUTF16(end, end); err != nil {
		return err
	}
	guard := e.sourceRevealGuard(stillCurrent)
	plan := newSourceRevealLayout(e)
	if err := plan.prepare(e, guard, start, end); err != nil {
		return err
	}
	if !guard.matches(e) {
		return errSourceTargetStale
	}

	// All fallible work and user callbacks have finished. SetSelection validates
	// before writing and invokes no callbacks. The fixed Buffer and both UTF-16
	// endpoints still match the admitted entry, so publication cannot reenter.
	if err := buffer.SetSelection(start, end); err != nil {
		return err
	}
	plan.commit(e)
	e.hasDesiredX = false
	e.geometryGeneration++
	e.navigationGeneration++
	// Handle.Focus only schedules toolkit focus; it does not invoke host code.
	e.handle.Focus()
	e.services.Invalidate()
	return nil
}

func (e *Editor) sourceReadAdmission() error {
	if e == nil || e.Buffer == nil {
		return errSourceTargetStale
	}
	if e.MultiCompositionPending() {
		return text.ErrCompositionPending
	}
	if e.compositionError != nil {
		return e.compositionError
	}
	// These retained owner fields also cover first-refused preedit, where there
	// is no MarkedRange. Do not Flush, Unmark or inspect toolkit mark geometry.
	if e.composing || e.compositionMulti || e.compositionMarked || e.compositionBuffer != nil || e.compositionToken != (text.CompositionToken{}) {
		return text.ErrComposition
	}
	if e.Buffer.InGroup() {
		return text.ErrGroup
	}
	return nil
}

func cloneSourceSelectionSet(set text.SelectionSet) text.SelectionSet {
	return text.SelectionSet{Ranges: append([]text.Range(nil), set.Ranges...), Primary: set.Primary}
}

func equalSourceSelectionSet(a, b text.SelectionSet) bool {
	return a.Primary == b.Primary && slices.Equal(a.Ranges, b.Ranges)
}

// This is an explicit boundary record, not a copied Editor. Getters below are
// pure: in particular this must never call inputStamp/observeInputOwner, a fold
// synchronizer, a geometry query or a focus API.
type sourceRevealBoundary struct {
	stamp                                 inputStamp
	buffer, cachedBuffer                  *text.Buffer
	readOnly, softWrap, lineNumbers       bool
	fontSize, width, height, cachedGutter float32
	shapeLimit, cachedShapeLimit, indexed int
	version, view, geometry, navigation   uint64
	composing, multi, marked, admitted    bool
	compositionBuffer                     *text.Buffer
	compositionStamp                      inputStamp
	compositionPrimary                    text.Range
	mark                                  ui.TextInputRange
	focused, dragging, reveal, longWarned bool
	scrollX, scrollY, desiredX            float32
	hasDesiredX, caretBlink               bool
	affinity                              ui.TextAffinity
	frameNow, blinkStart                  time.Time
	native                                *nativeInputClient
	clipboard                             *clipboardRequest
	foldControl                           foldControlPress
	handle                                ui.Handle
	services                              ui.Services
	highlightCode, errorCode              uintptr
	history                               text.HistoryStatus
	canUndo, canRedo                      bool
	label                                 string
	minHeight                             float32
	theme                                 *ui.Theme
}

// Go exposes a function's code address, not an epoch for assigning closures.
// This rejects observable code changes but makes no same-code-closure/ABA claim.
func sourceCallbackCode(callback any) uintptr {
	value := reflect.ValueOf(callback)
	if value.IsNil() {
		return 0
	}
	return value.Pointer()
}

func (e *Editor) sourceRevealBoundary() sourceRevealBoundary {
	b := e.Buffer
	return sourceRevealBoundary{
		stamp:  inputStamp{owner: e.inputOwner, version: b.Version(), selectionGen: b.SelectionGeneration(), compositionGen: e.compositionGeneration, primary: b.Selection(), token: e.compositionToken, group: b.InGroup()},
		buffer: b, cachedBuffer: e.cachedBuffer, readOnly: e.ReadOnly, softWrap: e.SoftWrap, lineNumbers: e.LineNumbers,
		fontSize: e.FontSize, width: e.width, height: e.height, cachedGutter: e.cachedGutter,
		shapeLimit: e.ShapeLimit, cachedShapeLimit: e.cachedShapeLimit, indexed: e.indexedLines,
		version: e.version, view: e.viewGeneration, geometry: e.geometryGeneration, navigation: e.navigationGeneration,
		composing: e.composing, multi: e.compositionMulti, marked: e.compositionMarked, admitted: e.compositionTargetAdmitted,
		compositionBuffer: e.compositionBuffer, compositionStamp: e.compositionStamp, compositionPrimary: e.compositionPrimary, mark: e.marked,
		focused: e.focused, dragging: e.dragging, reveal: e.revealCaret, longWarned: e.longLineWarned,
		scrollX: e.ScrollX, scrollY: e.ScrollY, desiredX: e.desiredX, hasDesiredX: e.hasDesiredX, caretBlink: e.CaretBlink,
		affinity: e.affinity, frameNow: e.frameNow, blinkStart: e.blinkStart, native: e.nativeClient,
		clipboard: e.pendingClipboard, foldControl: e.foldControl, handle: e.handle, services: e.services,
		highlightCode: sourceCallbackCode(e.Highlight), errorCode: sourceCallbackCode(e.OnError),
		history: b.History(), canUndo: b.CanUndo(), canRedo: b.CanRedo(),
		label: e.Label, minHeight: e.MinHeight, theme: e.theme,
	}
}

type sourceNavigationGuard struct {
	boundary     sourceRevealBoundary
	selections   text.SelectionSet
	cache        map[int]lineLayout
	fold         *foldState
	foldSnapshot *foldState
	stillCurrent func() bool
}

func (e *Editor) sourceRevealGuard(stillCurrent func() bool) sourceNavigationGuard {
	return sourceNavigationGuard{
		boundary: e.sourceRevealBoundary(), selections: e.Buffer.SelectionSet(),
		stillCurrent: stillCurrent,
		// Row maps can retain arbitrarily many previously visited lines. Their
		// writes are covered by geometryGeneration, never full copies/scans.
		// Keep the bounded shape/fold snapshots: folding may evict a hidden
		// shape while advancing only its own projection generation.
		cache: cloneSourceLineCache(e.cache),
		fold:  e.folds, foldSnapshot: cloneSourceFolds(e.folds),
	}
}

func (guard sourceNavigationGuard) matches(e *Editor) bool {
	if guard.stillCurrent != nil && !guard.stillCurrent() {
		return false
	}
	if e == nil || e.Buffer == nil || e.sourceReadAdmission() != nil || guard.boundary != e.sourceRevealBoundary() {
		return false
	}
	return equalSourceSelectionSet(guard.selections, e.Buffer.SelectionSet()) &&
		maps.EqualFunc(guard.cache, e.cache, equalSourceLineLayout) &&
		guard.fold == e.folds && equalSourceFolds(guard.foldSnapshot, e.folds)
}

func cloneSourceLineCache(cache map[int]lineLayout) map[int]lineLayout {
	if cache == nil {
		return nil
	}
	result := make(map[int]lineLayout, len(cache))
	for index, line := range cache {
		line.rowStart = slices.Clone(line.rowStart)
		line.lines = slices.Clone(line.lines)
		// TextLayout geometry is semantically immutable; query methods may
		// populate synchronized lazy caches. Row slices are independently owned.
		result[index] = line
	}
	return result
}

func equalSourceLineLayout(a, b lineLayout) bool {
	return a.value == b.value && a.start == b.start && a.windowStart == b.windowStart &&
		a.totalUnits == b.totalUnits && a.rows == b.rows && a.layout == b.layout &&
		slices.Equal(a.rowStart, b.rowStart) && slices.Equal(a.lines, b.lines)
}

func cloneSourceFolds(f *foldState) *foldState {
	if f == nil {
		return nil
	}
	copy := *f
	copy.ranges = slices.Clone(f.ranges)
	copy.intervals = slices.Clone(f.intervals)
	copy.reported = maps.Clone(f.reported)
	copy.collapsed = maps.Clone(f.collapsed)
	if f.pointer != nil {
		target := *f.pointer
		copy.pointer = &target
	}
	return &copy
}

func equalSourceFolds(a, b *foldState) bool {
	if a == nil || b == nil {
		return a == b
	}
	pointerEqual := a.pointer == nil && b.pointer == nil || a.pointer != nil && b.pointer != nil && *a.pointer == *b.pointer
	return a.identity == b.identity && a.generation == b.generation && a.candidates == b.candidates && a.projection == b.projection &&
		a.maxBytes == b.maxBytes && a.buffer == b.buffer && a.version == b.version && a.ready == b.ready &&
		a.viewGeneration == b.viewGeneration && a.status == b.status && a.painted == b.painted && pointerEqual &&
		sourceCallbackCode(a.provider) == sourceCallbackCode(b.provider) &&
		slices.Equal(a.ranges, b.ranges) && slices.Equal(a.intervals, b.intervals) && maps.Equal(a.collapsed, b.collapsed) && maps.Equal(a.reported, b.reported)
}
