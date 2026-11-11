# Whole-word Find: root source acceptance

2026-11-10. Accepted the app-only WholeWord runtime option, single findOptions
wiring and native checkbox. Core search semantics, session schema, replacement
algorithm and cache design remain unchanged. Whole-word uses existing Unicode
letter/number/mark/underscore boundaries, not Chinese segmentation.

Root read actual production/test source, implementation and both correction
handoffs, raw final/focused logs and independent reconciliation, and hashed the
five final files. Root viewed BOTH actual fresh760x480 light/dark PNGs: controls
and status fit without overlap and editor remains usable. Test geometry also
covers maximum editor font. These are MyGo Tester images, not physical OS input.

Evidence demonstrates actual checkbox navigation/wrap and Tab/Escape controls;
method-level Unicode/regex/case and workspace options; filtered warm-cache
invalidation/reuse; Replace Current allowed/refused pair, Replace All grouped
undo, readonly and failed-Flush preservation. The first refusal correction
selected `ate` instead of `cat`; root caught this vacuous proof. Second correction
uses exact[0,3), verifies the selected text and proves paired WholeWord=false
replacement versus true refusal. Earlier attempts are retained.

Initial final suite app15.298s/core-search0.742s passed; vet exit0. After only two
test corrections, scoped tests passed1.581s and final Replace Current1.740s.
No production/UI changed during corrections; broad green gates weren't repeated.

Final identities:

| File | SHA256 |
|---|---|
| app/app.go | 58B6A783968C8860BBE6E8EF87E74CE9C6BEA3560BD4C19A569FC44465557931 |
| app/state.go | 023C76A143CB572D0A2B45D783A20E105996FF570615357BB1C3F31032E79984 |
| app/find_whole_word_test.go | BEBFEBC560AA78D8A47B1390A2CA4E29E956C8328CB4AFEE0B080BF81328E744 |
| app/search_navigation_test.go | FA8EF3C1DAF50B67AA03F7ADE032F465EB0535202FAB4328EB7C28DFC4E72A40 |
| app/findbar_layout_test.go | 662C167BFFA6CC344EF2E2A73E3566824E03B0DA3BA3D19FA463279FFCF12180 |

Light PNG SHA25668D6F293F5E3F95CD9AFA4032E57916AF8C33494C3875E1FCE665D8FFECE1FAC;
dark PNG SHA256C72BF002E237A3D8E43F843D88236EA4F634326189A25232978EF8FAB502CD3E.
Independent reconciliation SHA256
16C1A31CF96E958C221F9995E794C2CC5348DF41BEE2F3F977B604BCCE7D2E5E.

Read [implementation](find-whole-word-implementation-result.md),
[first correction](find-whole-word-review-corrections.md),
[second correction](find-whole-word-review-corrections-2.md),
[reconciliation](find-whole-word-independent-reconciliation.md).
Source frozen for .17 package delivery. .16 stays current until actual new
binary acceptance; no process-resource/physical IME/platform claim is made.
