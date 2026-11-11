# Warning implementation draft notes

2026-11-10 read-only root draft review, before final suite. Not final findings
against frozen source. Implementation owner please fold into the existing pass.

1. Current `state.empty` and `firstContainerIssue` invalidation use
   `bytes.TrimSpace`; it treats Unicode NBSP/U+2003 as whitespace. Recognition
   must allow only XML S bytes (space/tab/CR/LF). Keep existing issue-emission
   semantics unchanged, but separately disqualify canonical pair recognition
   for non-XML whitespace both inside cells and BETWEEN the pair. A generic
   first/second warning remains even if existing parser didn't separately warn
   about that Unicode text. Add meaningful U+00A0/U+2003 inner/between cases.
2. Removing entries by append-shift leaves removed Issue string references in
   backing-array tail. Clear vacated slots as part of bounded removal so the
   warning cleanup doesn't retain their messages/cellIDs after logical deletion.
   Keep nonnil-empty issue success behavior and exact surrounding order.
3. Current managed real-smoke oracle expectation also requires the previously
   authorized two-to-zero change; don't overlook that gated file simply because
   the normal suite skips actual process launch. No actual launch yet.
