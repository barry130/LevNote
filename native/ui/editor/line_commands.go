package editor

import "levnote.local/native/core/text"

// editLines shares admission, composition ownership and history semantics for
// keyboard and command callers. The planner never changes the live buffer.
func (e *Editor) editLines(action text.LineEditAction) {
	if e.ReadOnly {
		return
	}
	if err := e.compositionIssue(); err != nil {
		e.fail(err)
		return
	}
	if e.Buffer.InGroup() && !e.composing {
		e.fail(text.ErrGroup)
		return
	}
	plan, err := e.Buffer.PlanLineEdit(action)
	if err != nil {
		e.fail(err)
		return
	}
	if !plan.Changed && plan.Selection == e.Buffer.Selection() {
		return
	}
	// Admission must precede settlement: a rejected copy leaves the marked
	// text, its owner, the selection and both history stacks exactly intact.
	e.UnmarkText()
	if !plan.Changed {
		if err := e.Buffer.SetSelection(plan.Selection.Anchor, plan.Selection.Head); err != nil {
			e.fail(err)
			return
		}
		e.hasDesiredX = false
		e.ensureCaret()
		e.services.Invalidate()
		return
	}
	if err := e.Buffer.BeginGroup(); err != nil {
		e.fail(err)
		return
	}
	if err := e.Buffer.ReplaceUTF16(plan.Start, plan.End, plan.Text); err != nil {
		e.Buffer.CancelGroup()
		e.fail(err)
		return
	}
	if err := e.Buffer.SetSelection(plan.Selection.Anchor, plan.Selection.Head); err != nil {
		e.Buffer.CancelGroup()
		e.fail(err)
		return
	}
	if e.Buffer.EndGroup() {
		e.changed()
	}
}
