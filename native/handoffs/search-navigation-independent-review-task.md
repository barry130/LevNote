# Bounded search navigation independent review

2026-11-09. Prepared for DeepSeek read_native_handoffs_resource_workload_next_task_md
after .12 docs are accepted. Review only once implementer reports frozen-ready;
do not continually read changing files. Read search-navigation-implementation-task.md
and actual app source/tests/result. No product/test/build/doc/evidence edits or
execution, no child agents. Publish search-navigation-independent-review.md.

Scope material defects only:
- Adjacency, exact forward/reversed match selection, initial caret, direction
  reversal/wrap and zero-width repeated-result progress reflect stated semantics.
- Warm cache cannot authorize replacement or bypass rich/source Flush. Identity
  and final version checked after Flush; newly changed targets use new results.
- Cache stores copied offsets only; no retained Match/Value/source/submatch/
  snapshot and no per-tab caches. Cold-search allocations remain acknowledged.
- Query/options/version/identity failures and too-many/no results clear old
  positions; no silent partial search or repeated zero-length stuck cursor.
- Successful tab/close/reload/rematch and Find close release refs, failed tab
  close/switch does not drop user drafts; don't clear cache on every reveal.
- Real tests demonstrate search/replace freshness and lifecycle, bounded warm/
  cold evidence isolates search cache (not UI rendering or process RSS claims).

Use actual trigger/outcome/file:line, not style nits/framework requests. Route
material findings directly to implementer/root. Owner executes tests; reviewer
does not duplicate them. Report truthfully if no material issue remains.
