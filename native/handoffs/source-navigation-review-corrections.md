# Initial implementation review corrections

2026-11-09. This is a planning addendum for the implementation owner, not extra
scope. Root/reviewer saw these in the first draft; fix/test as part of the
already active finite feature. No further permission is needed.

- Help `空列表示第1列` contradicts rejecting `1:`. Say omitting `:列` defaults
  to1; identify1-based coordinates, UTF-16 and short query limit. Leading zeros
  currently rejected; decide/document that intentionally or allow positive
  decimal values with zeros. Keep grammar unambiguous.
- `goToLineTargetCurrent` also needs nonnil Editor and Editor.Buffer matching
  the captured buffer. Persist selection after source/split navigation too.
- Numeric field's Escape needs composition-safe handling, and Escape while
  a panel button is focused needs a scoped route. Suppressing global Escape
  alone leaves button Escape with no handler. Test focused-button Escape
  directly; opening Find first can move focus and hide the defect.
- Pointer submit while numeric IME preedit is active may read the old bound
  query; block/defer until composition confirmed. Add actual Compose+pointer
  and Compose+Escape tests. Do not steal/commit a document's preedit on open.
- Captured callbacks must be tied to this panel instance/target. Closing and
  reopening (same or different tab) must not let an old key/click callback use
  the new global target. One small generation guard suffices if needed.
- Initial test `if !openGoToLine() || !submitGoToLine("bad")` inverted expected
  rejection; correct the assertion before treating its failure as product red.
- Actual SetPosition uses UTF-16/surrogate normalization, not full grapheme
  normalization. Correct the older plan's claim and Focus-every-frame wording.
- Layout at760px has ~532px main area; fixed210px input plus long action labels
  may overflow. Use compact faces with full accessibility labels/wrapping and
  assert all four horizontal edges as well as vertical bounds in captures.

Publish actual initial focused test result/checkpoint, then one corrected
focused run and final affected app/modules/editor suite/vet as necessary.
Reviewer will make one final pass after ready; no duplicate runtime testing.
