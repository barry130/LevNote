# Independent final growth source review

STATIC REVIEW COMPLETE, 2026-11-10. This review read the current four runtime
files and seven growth tests only. The reviewer did not run tests, builds,
products, benchmarks, or mutate source/tests/product docs. Test outcomes below
are the implementation owner's recorded evidence, not a new execution claim.

## Result

**No remaining source-grounded blocker.**

The final editor identity is
`09BEA640C8F1953E615B43088A7F58FA25AA8342B59694CC4023CBD36A885EBD`,
including the focused-element local Escape registration. All eleven reviewed
files match
`document-growth-budget-evidence-20261110/source-frozen.json` at the final
static check; no post-review implementation difference was observed.

## Source findings

1. `core/text/buffer.go` keeps a positive immutable base limit. Admission starts
   from `max(baseLimit, seed bytes)`, ordinary shrinks never lower the seed
   ceiling, and `Restore` reseeds the ceiling from the destination's own base and
   the restored snapshot while preserving buffer identity and clearing history.
   Oversized decoded seeds remain intact. Validation order is invalid UTF-8,
   range/surrogate rejection, then subtraction-based size admission.
2. `core/document/document.go` keeps the 128 MiB value as a disk-read bound while
   allowing decoded UTF-8 seeds to exceed it. `Revert` opens/codecs the file and
   restores its snapshot into the existing destination buffer rather than
   replacing that buffer identity.
3. `core/search/find.go` plans actual reverse-order replacements before opening
   a group. Every intermediate size is checked, literal replacements reuse one
   string, regex expansions are retained only after admission, and a failure
   before `BeginGroup` leaves source, selection, history, and group state intact.
   The later replay also cancels its own group if an unexpected edit fails.
4. `ui/editor/editor.go` retains only the wrapped size error and owning buffer
   for a refused composition. First and ongoing preedit refusals make `Flush`
   fail without unmarking or committing accepted preedit. Incidental
   focus/release unmark is ignored while refused; a valid accepted update or
   final replacement clears the refusal. Focused-element and raw-input Escape
   both call cancellation, with the focused element shortcut taking precedence
   over the enclosing App/global Escape action. Cancellation restores the
   original owner group without closing a foreign group or touching a replaced
   buffer. Navigation, pointer movement, commands, delayed cut, and indent
   opening are blocked or reconciled while composition is unresolved.
5. Indent admission compares the zero-based selected-line span against the
   remaining capacity divided by four, so the arithmetic cannot overflow.
   CRLF bytes stay in the buffer, reversed selection is remapped, and an active
   composition is not committed when preflight refuses.

## Test coverage

- `core/text/growth_test.go` covers oversized seed ceilings, shrink/regrow,
  exact-boundary and no-op edits, validation precedence, Restore policy,
  identity/history clearing, and open-group rollback.
- `core/document/growth_test.go` covers UTF-8, UTF-16LE/BE, and GBK decode with
  BOM preservation; untouched byte-for-byte save; edit undo/redo; Revert's
  buffer identity and destination ceiling; and the distinct raw disk-read bound.
- `core/search/growth_test.go` covers literal, regex, Unicode, zero-width, and
  late-growth reverse intermediates, conservative refusal, accepted undo/redo,
  and foreign-group ownership.
- `ui/editor/growth_test.go` covers ordinary paste refusal/retry, first and
  ongoing composition refusal, valid retry, explicit local Escape, focus and
  release, navigation/commands, replaced-buffer ownership, foreign groups, and
  CRLF/reversed-selection/overflow indent behavior.
- `modules/markdownview/growth_test.go` exercises the real rich editor draft;
  refusal retains the exact independent draft through release, failed recovery,
  and successful full-text recovery or shortened retry.
- `modules/graphview/growth_test.go` exercises the graph field draft and
  serialized host writeback: refusal leaves source/version/selection/history/
  group and draft unchanged through release, then recovers the full text.
- `app/document_growth_test.go` uses the real Markdown adapter and App recovery
  callback. Save, source/presentation transition, tab selection, and close stay
  blocked on refused source/rich input; source buffer and disk bytes remain
  unchanged; the full rich draft opens as a separate unsaved document while the
  original remains intact; retry and local Escape unblock lifecycle operations.

The recorded focused evidence is consistent with this source read: the initial
seven-package run exited 1 for the two App Escape cases, the corrected editor and
App rerun exited 0, and the five-package existing-seam rerun exited 0. Those logs
were inspected as evidence only; this handoff makes no new test-pass claim.

## Reviewed identities

| File | SHA-256 |
|---|---|
| `core/text/buffer.go` | `0AF9281A6051B8715854665043CBCC3EA9DFBB89BA5D028BA719B312B244FB21` |
| `core/document/document.go` | `6859017748C8884DD9CC84B185A405C0A78089AA0D84F191024489CB24C11219` |
| `core/search/find.go` | `D3D7EDD2D2A2CD47C1D362A225387566D3F8709A24D92A479CCD57207645A4F5` |
| `ui/editor/editor.go` | `09BEA640C8F1953E615B43088A7F58FA25AA8342B59694CC4023CBD36A885EBD` |
| `core/text/growth_test.go` | `D7284C8285BD48061BB31224C6DF1596C1580F413164192F80883D2A3A518F38` |
| `core/document/growth_test.go` | `0E8E8803FA22EEE748897F59FCEE148F16E4D920E98991A5CA0640BCDE847E45` |
| `core/search/growth_test.go` | `6C0DEBA2F982F0A4A10D5F7288C13F9BE4030CAEF6BDA44EDAD0DF9AA5D64D3D` |
| `ui/editor/growth_test.go` | `21FB5BCC2464AFEA0D5E5FC76236B5FF4CB4AF110370E0F55E756CCD00F775A8` |
| `modules/markdownview/growth_test.go` | `1F45BCE17706E8CEE44C26727C920B2AE08F68A1605B627954B80A436F9B0990` |
| `modules/graphview/growth_test.go` | `0F14DE124EB3108FC07F5913A29A9512EAE892DD9DDC7BC6FBC53BD4AB594119` |
| `app/document_growth_test.go` | `613446886000151828C5DB505A24CD9B3CD9A2617B080E67B82D55C705F7B47D` |

## Scope exclusions

Unbounded retention of active IME group intermediate strings remains the
documented future retention optimization. Revision of existing table structural
rollback behavior remains a separate scope. Neither is counted as a defect in
this growth increment.
