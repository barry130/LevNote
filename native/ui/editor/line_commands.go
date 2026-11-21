package editor

import "levnote.local/native/core/text"

// editLines shares admission, composition ownership and history semantics for
// keyboard and command callers. The planner never changes the live buffer.
func (e *Editor) editLines(action text.LineEditAction) {
	a := e.beginInputAction()
	e.editLinesInputAction(a, action)
	a.finish()
}
func (e *Editor) editLinesInputAction(a *inputAction, action text.LineEditAction) {
	if err := e.compositionIssue(); err != nil {
		a.fail(err)
		return
	}
	if e.ReadOnly {
		return
	}
	if e.Buffer.InGroup() && !e.composing {
		a.fail(text.ErrGroup)
		return
	}
	plan, err := e.Buffer.PlanLineEdit(action)
	if err != nil {
		a.fail(err)
		return
	}
	if !plan.Changed && plan.Selection == e.Buffer.Selection() {
		return
	}
	if plan.Changed {
		if err := e.Buffer.CheckReplaceUTF16(plan.Start, plan.End, plan.Text); err != nil {
			a.fail(err)
			return
		}
	}
	e.revealFolds(plan.Start, plan.End, true)
	if err := e.settleSingleInput(a); err != nil {
		a.fail(err)
		return
	}
	if !plan.Changed {
		if err := e.Buffer.SetSelection(plan.Selection.Anchor, plan.Selection.Head); err != nil {
			a.fail(err)
			return
		}
		e.hasDesiredX = false
		e.prepareInputSelection()
		return
	}
	if err := e.Buffer.BeginGroup(); err != nil {
		a.fail(err)
		return
	}
	if err := e.Buffer.ReplaceUTF16(plan.Start, plan.End, plan.Text); err != nil {
		e.Buffer.CancelGroup()
		a.fail(err)
		return
	}
	if err := e.Buffer.SetSelection(plan.Selection.Anchor, plan.Selection.Head); err != nil {
		e.Buffer.CancelGroup()
		a.fail(err)
		return
	}
	if e.Buffer.EndGroup() {
		a.preparedChange()
	}
}
