# Next small UX slice: expose existing whole-word search

Root planning2026-11-10. READ-ONLY REVIEW ACTIVE for DeepSeek
read_native_handoffs_release10_ui_feasibility_task_md while .16 packages.
Implementation INACTIVE until session-output .16 accepted. This is
later work, not included in .15/.16. Root remains coordinator; agent owns all
implementation/tests/captures. No children, no unrelated refactor.

During current source freeze read actual code/tests and confirm or correct this
scope in `find-whole-word-plan-review.md`. No source/test/build/capture/doc edits
or execution. Check core word boundary semantics, minimum-window layout test
and workspace options wiring. Record just concrete gaps and smallest plan; no
external browsing required to adopt this conventional search feature.

Confirmed gap from source: core/search.Options already has WholeWord with
Unicode tests, but app.findOptions passes only Query/Regex/CaseSensitive and
the native Find bar has no whole-word checkbox. Legacy product had this option.
Expose one lightweight native checkbox using the existing shared semantics.
No dependency, new parser, search algorithm, extra per-tab cache or animation.

Proposed narrow changes: app boolean + shared findOptions wiring + checkbox
alongside case/regex. Both document navigation/replacement and workspace
search consume the same options. Current cache key already stores full Options,
so verify actual invalidation on toggle rather than add a second invalidation
scheme. Keep options in current runtime state like case/regex; no new persistent
preference/schema. Same KeepFocus/keyboard/Escape/composition behavior.

Semantic boundary is existing core word(): Unicode letter/number/mark or '_'.
This is boundary matching, not linguistic Chinese segmentation or syntax-aware
identifiers. Query/regex remains unchanged. Root should ensure user-facing
description doesn't promise NLP word detection.

Verification plan: actual App/Tester toggle; cat vs scatter/cat_ and combining
mark/CJK boundary; regexp+case interaction; next/previous and wrap; warm cache
then toggle forcing fresh offsets; replace-current/all through fresh shared
options and grouped undo; readonly and failed Flush preserve source; one local
workspace fixture uses same option. Reuse prior focus tests, do not retest every
prior keyboard case redundantly. At760x480 both themes all controls visible,
Tab order includes checkbox, wrap and editor/status geometry remain usable;
actual two screenshots if markup changes, root visually reviews. Final app+
core/search suite/vet after corrections; no build until source acceptance.
