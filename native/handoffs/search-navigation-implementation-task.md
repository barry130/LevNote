# Activate bounded document-search navigation correction

2026-11-09. Sole implementer DeepSeek
read_native_handoffs_release10_ui_feasibility_task_md. .12 runnable package is
accepted and immutable. GLM is read-only planner; other DeepSeek .12 docs only.
No child agents. Root plans/reviews, you implement and execute. This task is
active; do not wait for GLM plan. Read it if it arrives, but root constraints win.

Scope: app/state.go find, App small state/lifecycle hooks in app.go/plugins.go/
view.go/workspace.go Shutdown as necessary, new app/search_navigation.go and
meaningful tests. Do not
change search matching/replacement engine, text/editor APIs, plugin/worker,
dependencies, .12 source-navigation behavior or existing immutable evidence.
Use native/scripts/go.ps1 -GoArguments only. No build or docs until acceptance.

Concrete problems root observed: previous uses Match.End < selection.Start,
skipping adjacent matches; repeated previous/next materializes Snapshot.String,
regex matches/hash/value/submatch arrays every time even with unchanged source.

Implement small app-owned cache for only ONE active document. Key exact
Tab/Buffer identity, monotonically changing Buffer.Version, and search.Options.
Cache only a new slice of {Start,End int} values (max search.MaxMatches=200000),
small status/error if needed, and last-returned cursor. Do not retain search.Match,
source strings/substrings, text.Snapshot, submatch arrays or per-tab caches.
Cold path may use existing search.Find, convert offsets, then let temporary
source/matches die. This avoids repeated work; it does not lower cold-search
peak or prove RSS/CPU improvement. Prefer no extra query hash/framework.

All queries still flush current source/rich draft first, then validate tab/doc/
editor/buffer identity and use final buffer version. Cache hit must not bypass
Flush; failed Flush keeps view/selection/history and no navigation. Preserve
existing source reveal/mode transition behavior. Do not expand revealSource
into a new abstraction. Check final identity before reveal; abandoned results
must not act on another tab. Readonly/byte-source navigation still works.

Define/test nearest-match semantics explicitly:
- Current nonempty selection exactly equals a match (either direction): next/
  previous advances one match with wrap, so adjacency is not skipped.
- Otherwise next chooses first Start >= selection.End; previous chooses last
  End <= selection.Start. Use ordered bounds, not Head for reversed selections.
  A caret at a nonempty match start/end can include that nearest match.
- An initial zero-width match at the caret can be selected. Repeating navigation
  with the same last-returned zero-width result advances/wraps; both directions
  must make progress whenever more than one result exists. One match may wrap
  to itself. Use recorded selection/result for this, no core selection-revision
  framework; changing caret to a different position re-anchors search. Don't
  claim detection of manual move-away-and-back to identical coordinates that
  the API cannot observe.
- Query/options/version/new buffer clears result cursor. No-results/errors
  clear old positions; ErrTooManyMatches remains a visible refusal, do not
  silently navigate its partial result. Empty query clears cached results.

Drop all cache references on committed current-tab switch/add/remove, reload/
encoding/rematch and shutdown, and when Find closes. Changes to query/options/
current Buffer.Version should drop stale data cheaply on next UI/frame/action;
no background worker/timer and no per-frame whole-source conversion. Failed
close/switch preserves active document; stale version must still never match.
Do not put a cache-clear into releasePresentation used by every successful find
or the warm path will never hit. One centralized invalidation helper, no stale
closed buffer ownership.

Replace/ReplaceAll stay on existing fresh matching/hash authority; navigation
offsets may never authorize edits. Existing Replace-current followed by find
must invalidate/recompute after actual edit. Do not add new shortcuts/UI or
rebuild capture images for unchanged controls.

Meaningful tests: actual App find with adjacent aaaa/aa, direction reversal,
normal/reversed selection, wrap, one/no result, zero-length ^/$/Unicode boundary
where supported, UTF16 emoji, query/case/regex changes, edit/undo/restore/new
buffer, failed and source-changing rich Flush, close/switch/cancel cache lifetime,
readonly/source reveal, replacement freshness. Count cold scans through a tiny
test hook or scoped dependency (avoid permanent metrics). One bounded warm/cold
allocation benchmark/check isolates navigation cache; assert warm path avoids
whole-file scan and source retention by data shape/lifecycle, not flaky RSS.

Focused tests first, then one final affected full app + existing core/search
suite and scoped vet. Preserve actual commands/stdout/exits and real failures
in handoffs/search-navigation-result.md with evidence paths; show before/after
behavior, final changes, limits and specific allocation evidence. Freeze for
root/independent review. No unrelated full tests, build, resource matrix, helper
retry or product docs.
