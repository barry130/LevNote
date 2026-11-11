# Whole-word Find: narrow review corrections

SECOND CORRECTION REQUIRED: root read the actual new refusal test. `category`
has cat at UTF16[0,3), but the current fixture selects[1,4) (`ate`). That test
passes even with WholeWord disabled and doesn't prove the requested behavior.
Use[0,3), first assert selected text is exactly query, and prove the paired
WholeWord=false state does replace that same candidate. Re-enable true in an
independent fresh fixture for no-mutation assertion. One focused test rerun only;
preserve earlier correction/log, publish a second correction handoff.

ACTIVE2026-11-10. Owner implementation DeepSeek
`/root/read_native_handoffs_release10_ui_feasibility_task_md`. Production wiring
and screenshots reviewed without material defect. Read root-review-notes. No
production/UI changes expected; no children/build/launch/capture/docs changes.

1. Existing cold-scan test's fixture only has whole-word hits. Include an
   embedded hit and assert exact cached offsets before/after WholeWord toggle,
   plus one new cold scan and reuse of the new warm cache. Keep existing checks
   meaningful, don't duplicate full cache matrix.
2. Extend Replace Current test with an in-word selected occurrence; WholeWord
   must refuse it, preserving text, version and selection. Keep eligible
   replacement and grouped Replace All undo proof.
3. Record precise evidence: Unicode/regex/case table and workspace fixture set
   runtime options directly; only navigation/layout use actual checkbox. No need
   to inflate UI tests just to match earlier overwording. Preserve prior result,
   publish find-whole-word-review-corrections.md with accurate final claims/hashes.

Run just affected focused tests after edits; since only tests/claims change, no
repeat full suite/vet or screenshot needed unless production changes become
necessary. Preserve raw log separately and report for independent reconciliation.
