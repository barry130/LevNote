# Whole-word Find: independent reconciliation

2026-11-10. Reconciled both correction passes and the focused evidence against
the [prior independent review](find-whole-word-independent-review.md),
[first correction handoff](find-whole-word-review-corrections.md) and
[second correction handoff](find-whole-word-review-corrections-2.md). I ran no
test, vet, build, package, worker, launch or duplicate capture.

## Verdict

The three evidence gaps in the prior review are now resolved. Production wiring
is unchanged and remains consistent with the reviewed app-only design.
Corrected focused tests pass, and the evidence claims accurately distinguish
actual checkbox interactions from direct runtime-option setup. I recommend
bounded source acceptance of the app-only whole-word slice.

## Reconciliation

1. **Navigation cache offsets changed.** 
   `native/app/search_navigation_test.go:167-194` now uses
   `needle needless needle`. Before enabling `WholeWord`, it requires exact
   cached offsets `[0,6) [7,13) [16,22)`. After the option change it requires
   the filtered offsets `[0,6) [16,22)`, one additional cold scan, and a
   subsequent warm-cache reuse with no extra scan. The existing version,
   edit, undo, restore and new-buffer cache checks remain in the same test.

2. **Replace Current refusal is non-vacuous.** 
   `native/app/find_whole_word_test.go:107-137` now uses paired independent
   `category` fixtures. Both select UTF-16 `[0,3)` and require
   `SliceUTF16(0,3) == "cat"`. With `WholeWord=false`, Replace Current changes
   the text to `dogegory`, proving the selection is the matching occurrence.
   With `WholeWord=true`, a fresh fixture must preserve text, buffer version and
   selection exactly. The former `[1,4)`/`ate` assertion is no longer present.

3. **Evidence wording now matches the test mechanism.** 
   The correction handoff identifies the visible `全词` click and Tester
   Next/Previous actions in
   `TestFindWholeWordCheckboxNavigationAndWrap`, the actual checkbox toggle and
   layout/focus checks in `findbar_layout_test.go`, and separately states that
   the Unicode/regex/case table and workspace test assign runtime options
   directly. It also states that replacement and lifecycle tests call the app
   methods directly rather than claiming pointer/button event coverage. This
   accurately replaces the earlier combined-UI wording; the prior result
   remains preserved as history and the correction handoff supersedes that
   wording.

## Final identities

| Artifact | Bytes | SHA-256 |
|---|---:|---|
| `native/app/app.go` | 33,365 | `58B6A783968C8860BBE6E8EF87E74CE9C6BEA3560BD4C19A569FC44465557931` |
| `native/app/state.go` | 11,176 | `023C76A143CB572D0A2B45D783A20E105996FF570615357BB1C3F31032E79984` |
| `native/app/find_whole_word_test.go` | 7,548 | `BEBFEBC560AA78D8A47B1390A2CA4E29E956C8328CB4AFEE0B080BF81328E744` |
| `native/app/search_navigation_test.go` | 21,407 | `FA8EF3C1DAF50B67AA03F7ADE032F465EB0535202FAB4328EB7C28DFC4E72A40` |
| `native/app/findbar_layout_test.go` | 7,275 | `662C167BFFA6CC344EF2E2A73E3566824E03B0DA3BA3D19FA463279FFCF12180` |
| `review-corrections-2-focused.log` | 51 | `5F50F57B160F074F46D04688300C2AC68AE903758F14A1E40CC96021E0F51065` |
| light capture | 30,108 | `68D6F293F5E3F95CD9AFA4032E57916AF8C33494C3875E1FCE665D8FFECE1FAC` |
| dark capture | 30,155 | `C72BF002E237A3D8E43F843D88236EA4F634326189A25232978EF8FAB502CD3E` |

The second focused log records exactly:

```text
ok      levnote.local/native/app    1.740s
EXIT_CODE=0
```

The two production hashes and unchanged layout/capture hashes match the prior
review. The first correction log remains preserved at SHA-256
`C61637F38CB83B574D59EC7F5B6833C7EBDC9F27A91B264454F315C6416C0D77`;
the second pass changes only the replacement test. The prior full
app/core-search suite and vet predate both correction passes and are not
represented as executing these new assertions. Since no production source
changed, the bounded correction reran only the affected focused test as
requested.

## Limits

- This evidence covers the app wiring, method-level search behavior and
  synthetic MyGo Tester interactions only. It is not physical Win32
  focus/IME, accessibility, packaged-host visual, resource or release evidence.
- Whole-word behavior is the existing Unicode letter/number/mark/underscore
  boundary predicate, not Chinese segmentation or syntax-aware identifier
  matching.
- The accepted `.16` package does not contain this later whole-word slice. A
  future package delivery must produce its own build, exact-host smoke and
  immutable evidence.
- No duplicate full suite, vet, benchmark, screenshot generation or resource
  measurement was performed during reconciliation.
