# .22 user guidance and current launch documentation

ACTIVE for the Astra delivery owner after tab-workflow-release22-root-acceptance.md.
No children.
Own root README.md/ROADMAP.md/CHANGELOG.md and native README.md/ARCHITECTURE.md/
VALIDATION.md only. No source/package/portable/evidence changes, launches,
tests/builds/captures or resource measurements.

Update the current launch references to accepted .22, preserve dated historical
.21 and earlier records. Explain user-visible behavior concisely: Ctrl+Tab and
Ctrl+Shift+Tab cycle in visual tab order with wrap; active tab button becomes
visible once on actual selection change, manual strip scrolling stays put.
Zero/one tabs do nothing. Source/rich pending edits use existing safe Flush;
failure retains draft/current tab. During actual preedit in visible Find or
GoToLine fields, a visible message requests completion/cancel then explicit
retry; no deferred surprise switch. GoToLine stale composition after pointer
focus leave is corrected. Avoid exposing internal widget names in usage text.

Main READMEs should lead with launch and usage, not a replicated hash table.
Place exact build/host/manifest identities and actual gate count once in
VALIDATION, linked to tab-workflow-root-acceptance and eventual release22 root
acceptance/delivery. Tab synthetic tests/capture establish those behaviors;
one hidden host check establishes packaged initialization/clean close, not
physical Windows IME behavior or lower process memory. Matrix and image reader
remain future work unless separately accepted; do not claim them completed.

One scoped link/current-reference check after final docs. Publish
tab-workflow-release22-docs-result.md with final file identities and the actual
check result, then freeze. Keep changes concise and grouped; no new delivery
framework or repeated app verification.
