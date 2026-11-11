# Next finite improvement: document search navigation

2026-11-09. GLM read_native_handoffs_release10_docs_task_md owns READ-ONLY
planning now. Navigation feature is frozen under independent review; .12
delivery may run next. Do not edit product source/tests/docs or run commands
other than reading/searching. No children, compact before200K. Root coordinates.

Read actual native/app/state.go find/replace/findbar and core/search/find.go,
existing tests, Buffer versions and tab/reload/configure lifecycle. Publish
handoffs/search-navigation-next-plan.md, then stop pending activation.

Concrete observations to assess, not assumptions to rubber-stamp:
- Every previous/next action materializes the whole Snapshot.String, runs regex,
  hashes source and builds Match values, even on unchanged source/query.
- Previous uses match.End < selection.Start. With adjacent matches (for example
  aa in aaaa), this can skip the immediately previous match whose End==Start.
- Zero-length regex, a lone match, direction reversal, wrapping, reversed
  selections, manual caret movement and post-Flush source change need explicit
  semantics. Existing findOptions currently exposes case/regex only.

Propose one small staged improvement compatible with mature editor previous/
next behavior. Prefer one active-document result cache of copied UTF16 start/
end positions if useful; never retain Match.Value, submatch arrays, a full source
string or snapshot for navigation. Key by exact Buffer pointer/version/options;
no stale coordinates after edit/undo/restore/reload/close/rematch. Explain
invalidation, release and fixed bounds, including ErrTooManyMatches and errors.
Do not add a cache if it complicates safety more than it helps; document tradeoff.
Replace/ReplaceAll must still perform their actual snapshot/hash checks; do not
reuse navigation-only positions as edit authority or bypass rich-view Flush.

Identify exact smallest files, behavior tests and one bounded allocation check
needed to establish avoided repeated work (not process RSS/CPU claims). No new
framework/dependencies, no streaming search rewrite, no arbitrary large resource
matrix or web code copying. Keep .12 runnable release independent of next work.
