# Whole-word Find: independent review

2026-11-10. Reviewed the frozen app-only implementation, tests, raw logs and
both captures against the [implementation result](find-whole-word-implementation-result.md),
whose SHA-256 is
`8B92E45A89AE7285F4AB9EB76044A0F2CC5FBEFBB1DB7183AEF18341C6366EA2`.
I changed no product source, test, documentation or evidence, and ran no
duplicate test, vet, build, package or launch.

## Verdict

The production wiring appears correct and I found no product defect in the
changed path. The frozen result and current tests do contain three concrete
test/evidence gaps, so final acceptance should wait for an implementer-owned
test correction and evidence reconciliation. A corrected test could still
reveal a product defect; the present evidence does not establish one.

## Source and behavior audit

- `native/app/app.go:124` adds one runtime `WholeWord bool` beside the existing
  Find options. It has no session-capture, schema or dependency reference.
- `native/app/state.go:184-185` propagates that bool through the single
  `findOptions()` result. Navigation (`state.go:202`), Replace/Replace All and
  workspace search therefore consume the same option.
- `native/app/state.go:315-320` renders `全词` with the same `KeepFocus()` and
  Escape lifecycle as the case and regex checkboxes. The checkbox is inside the
  existing wrapped first row and introduces no new focus or composition path.
- `native/app/workspace.go:97-99` copies `findOptions()` before starting the
  workspace worker and passes it through `search.WorkspaceOptions.Options`.
- `native/app/search_navigation.go:26-33` already stores the complete
  `search.Options` in the navigation cache. The equality checks at lines 49-57
  and 249-252 make a `WholeWord` change discard warm offsets and perform one
  fresh scan; no parallel invalidation scheme was added.
- Existing core behavior is unchanged. `core/search/find.go:20-23` owns
  `WholeWord`, and `:76-84` applies the existing Unicode
  letter/number/mark/underscore boundary predicate. This remains boundary
  matching, not Chinese segmentation or syntax-aware identifier matching.

The stale-callback, buffer identity, failed-Flush and cache code surrounding
the changed path is unchanged. Persisting a tab or the workspace session does
not capture `WholeWord`, matching the existing runtime-only case/regex options.

## Test audit

The focused tests use the actual app paths and contain meaningful exact
assertions:

- `find_whole_word_test.go:13-43` clicks the visible checkbox, then checks the
  selected UTF-16 ranges for standalone `cat`, identifier rejection and both
  wrap directions.
- `find_whole_word_test.go:47-93` checks literal, CJK, combining-mark and
  regex/case-insensitive/case-sensitive offsets, but the table assigns
  `Regex`, `CaseSensitive` and `WholeWord` directly at `:78`; it does not click
  the controls. The expected arrays include both standalone matches and reject
  `category`, `_cat`, `cat_` and `猫咪`.
- `find_whole_word_test.go:96-121` checks an eligible standalone Replace
  Current and proves Replace All is exactly one undo group. It does not select
  an in-identifier match and prove Replace Current refuses it while preserving
  the buffer, version and selection.
- `find_whole_word_test.go:124-154` checks readonly and failed rich-Flush paths
  preserve source, buffer version and selection.
- `find_whole_word_test.go:157-190` reaches the real `searchWorkspace()` entry
  and requires exactly the standalone fixture match, although it assigns
  `WholeWord` directly at `:169`.
- `search_navigation_test.go:166-184` proves changing `CaseSensitive`, `Regex`
  and then `WholeWord` increments the cold-scan count once each. Its
  `needle filler needle` fixture has no non-whole-word `needle` occurrence, so
  it does not prove the cached offsets themselves changed after the toggle.
- `findbar_layout_test.go:12-88` retains bounds, overlap, editor and status
  geometry checks for light, dark and maximum editor font at `760x480`.
  `:184-225` proves Tab reaches `全词` and Escape closes Find.

The expected ranges and replacement strings match the implementation and core
predicate. The earlier workspace and focus failures were fixture/assumption
errors as documented, not product reds hidden as passes.

## Required test/evidence correction

1. In `native/app/search_navigation_test.go:167`, change the existing cache
   fixture to include a `needle` embedded in a larger word. Retain the
   scan-count assertions and additionally assert the filtered offsets before
   and after toggling `WholeWord`, so the test proves the cache discarded the
   old offsets and stored a different set. A cold-count change alone is weaker
   than the result's “filtered offsets” claim.
2. In `native/app/find_whole_word_test.go:96-104`, choose a selected
   in-identifier occurrence for the Replace Current refusal case. Assert that
   text, buffer version and selection remain unchanged; keep the existing
   eligible replacement and grouped-Replace-All cases.
3. In `native/app/find_whole_word_test.go:47-79`, the regex/case/whole-word
   table directly assigns runtime booleans. The workspace test at `:157-169`
   also directly assigns `WholeWord`. Either describe these accurately as
   method-level shared-option tests, or add one narrow actual checkbox
   interaction for the combined behavior. The current result should not
   describe these direct assignments as evidence of checkbox interaction.

## Evidence identities

| Artifact | SHA-256 |
|---|---|
| `native/app/app.go` | `58B6A783968C8860BBE6E8EF87E74CE9C6BEA3560BD4C19A569FC44465557931` |
| `native/app/state.go` | `023C76A143CB572D0A2B45D783A20E105996FF570615357BB1C3F31032E79984` |
| `native/app/find_whole_word_test.go` | `1AA1E1047DE5EE917D34097FC044784D992E4F6D79EC8C0FBC7F918BA20E6AF9` |
| `native/app/findbar_layout_test.go` | `662C167BFFA6CC344EF2E2A73E3566824E03B0DA3BA3D19FA463279FFCF12180` |
| `native/app/search_navigation_test.go` | `7773D1C1DB801CD640BB09D329806C040E905B87AD8E9712E61FD509399B30BD` |
| light capture | `68D6F293F5E3F95CD9AFA4032E57916AF8C33494C3875E1FCE665D8FFECE1FAC` |
| dark capture | `C72BF002E237A3D8E43F843D88236EA4F634326189A25232978EF8FAB502CD3E` |
| focused green log | `843B880EEDB949C3FFF18CC648490E1257B8917395F01EFB6C02ED84793F394C` |
| final affected suite | `1C2D7B6ACA88709D2EDB82FCED6301C20B8A3CDEEF6A11C8CB8E384C9BB4253D` |
| final vet | `C1E97067C5F479A44A6F57297A0A8F87A59D181C3C529910F8BF059094BC3ABB` |

Both current PNGs were inspected and decode as exactly `760x480`; light and
dark variants show all Find actions, checked `全词`, a usable editor and the
status row without visible clipping. This supplements, rather than replaces,
the geometry/event assertions. Testers establish synthetic rendering and input
only and are not physical OS focus/IME, packaged-host visual or resource
evidence.

## Bounded recommendation

Treat the app-only wiring and two captures as technically sound, but hold final
acceptance until the implementation owner corrects the three concrete tests or
evidence claims above and publishes replacement identities/results. No product
source change is indicated by the current review. A future ordinary package
delivery still needs its own build, exact-host smoke and recorded identities.
The unchanged number-boundary rule needs no new general suite here, and the
control's pointer behavior remains covered by the existing `KeepFocus()`
contract plus the direct keyboard focus/Escape assertions.
