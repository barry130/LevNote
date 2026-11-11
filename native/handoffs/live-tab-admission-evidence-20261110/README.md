# Live-tab admission focused evidence

2026-11-10. Commands were run from the repository root through
`native/scripts/go.ps1`; each numbered `.command.txt` records the exact command
and numeric exit code. Stdout and stderr are stored separately.

| Evidence | Result |
|---|---|
| 001-format | exit0 after the initial source/test correction |
| 002-admission-focused | exit0; admission group passed, app3.397s |
| 003-format-after-real-recover | exit0 |
| 004-admission-with-real-recover | exit1; preserved failed first real-view test attempt |
| 005-real-recover-focused | exit1; preserved failed direct-call attempt |
| 006-real-recover-button-check | exit1; preserved stale-surface assertion attempt |
| 007-real-recover-direct | exit1; preserved overly specific warning assertion attempt |
| 008-real-recover-retry | exit0; real Markdown view retained its draft across two cap refusals and recovered after close-one |
| 009-format-data-safety | exit0 |
| 010-admission-data-safety | exit0, 5.540s wall; app2.420s |
| 011-existing-affected | exit0, 4.406s wall; app3.267s |

An inherited working-session report described an earlier focused admission run
that found fixture errors and a release-order defect:
`addDocumentChecked` released the origin presentation after `Flush` but before
the final admission recheck. This agent did not personally run or observe that
unretained check, and no original stdout, stderr, exit code or timing is
available. It is reported history, not preserved evidence. The corrected source
is retained and verified by the later post-Flush cases in008 and010.

004-007 are retained test-construction failures, not product regressions. The
first attempted UI click did not dispatch the real recovery callback; after the
test called the real Markdown view's public `RecoverDraft` directly, it then
asserted a capacity sentence that the view's redraw intentionally replaced with
its conflict surface. The final test retries the callback at capacity and then
after closing one tab, proving retention without relying on transient text.

The final admission group covers nine top-level tests and their focused
subcases. The existing affected run covers open-many, every `TestReopenClosed`
group and the selected native-image open/session/legacy-route tests.

No full app suite, vet, build, app launch, screenshot, physical-input test,
resource measurement, package change or product-document change was performed.
