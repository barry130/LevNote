# Search navigation/cache: root acceptance

2026-11-09. Root accepts this bounded app-only feature. Current runnable package
remains .12 until .13 delivery, which will also include a separately validated
minimal Find-bar reflow if needed. Search core/editor/format semantics unchanged.

Root read actual new code, lifecycle hooks, regression source, raw initial red,
assertion-correction failures, final focused/suite/vet/benchmark logs, owner
result and independent final review. Reviewer found no material defect and
did not execute tests or builds.

Accepted behavior: adjacent matches no longer skip previous; exact forward/
reverse selections advance with wrap; otherwise nearest results use ordered
bounds. Initial zero-width at caret allowed; repeated last zero-width result
progresses/wraps, single result can wrap to itself. Manual move-away-and-back
to identical coordinates isn't observable; no selection-event claim.

Only active-document copied UTF16 offset pairs retained; exact identity/version/
options key; no Match.Value/source/Snapshot/submatch/per-tab cache. Every find
still Flushes, validates final identities, and doesn't bypass replacement hash
authority. Cache cleared on committed transitions/Find close/changed source or
query; failed actions preserve live drafts. Over-limit refusal holds no offsets.

Final focused2.035s; app16.286s/search0.716s exit0; vet exit0. Initial adjacency
product red reproduced; later two wrong setup/expectation assertions corrected
and retained separately. Seven final source/test hashes in result were reconciled
by independent reviewer. search_navigation.go:
8F77367B137135618CEB65BE0513ED5670033E2500A867C87DD8185CB5854E6D;
test: CEF83EA00713A9896C6FCD117BB7A710BFD184FB788235119121F9C99CF2F79D.

Bounded20x benchmark57,344-byte/4096-hit fixture: cached lookup0B/op,0allocs;
cold path1,063,316B/op,4137allocs. This measures only this function; cold search
still scans whole text; cache maximum200000 offset pairs~3.2MB on amd64.
No process/physical UI/concurrency/speedup percentage conclusion. During-scan
tab test uses synchronous injection, not general concurrent access validation.

References: search-navigation-result.md,
search-navigation-independent-review.md and search-navigation-evidence/.
Next findbar-layout-task.md activated; no change to accepted search semantics.
