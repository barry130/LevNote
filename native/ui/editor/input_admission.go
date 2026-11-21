package editor

// InputActionError reports whether retained or active native input currently
// prevents a host action that may move focus. It is a neutral readiness check,
// not a source snapshot, target identity, route check, or permission to write.
// Callers must separately validate attachment, routing, readonly policy for
// writes, and operation ownership at the effect boundary.
//
// This method is UI-thread-only and observational: repeated calls do not
// observe or update input-owner epochs, materialize source text, invoke
// callbacks, change focus, or settle input groups. A clean idle readonly Editor
// is admissible. Nil Editor or Buffer returns the existing stale-target error.
//
// Actual retained input errors and foreign group/composition errors are
// returned unchanged. This includes first-refused input without a marked range
// or live Buffer group; the owner retains its own final-text/Escape route.
func (e *Editor) InputActionError() error {
	return e.sourceReadAdmission()
}
