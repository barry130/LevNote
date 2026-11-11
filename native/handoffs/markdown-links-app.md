# Native Markdown links — app acceptance

## Current focused checkpoint — 2026-11-08 16:44 +08:00

The real registered Markdown adapter now exercises the published RoleLink
contract. Focused app tests added in `app/markdown_links_app_test.go` cover:

- saved-directory local opens by Ctrl/Cmd-click and Tab+Enter, including the
  Space-is-not-Enter distinction, one-open/no-source-mutation, Save and reopen;
- source mode exposing no authored preview RoleLink and preview restoring it;
- readonly preview/split Unicode duplicate-anchor navigation through the
  shared outline resolver, including the UTF-16 split caret;
- unsupported external-link refusal with unchanged tabs, bytes, version,
  caret/history and no source-editor focus;
- successful and failed pending-Compose Flush barriers;
- one focused 760x480 dark wrapped-link image with a neighboring task checkbox.

Targeted app acceptance command (a `-run` regex, not a full `./app`
package run):

```powershell
native\scripts\go.ps1 -GoArguments @('test','./app','-run','TestMarkdownLinkRealAdapterLocalPointerKeyboardSaveReopen|TestMarkdownLinkRealAdapterReadonlyAnchorsAndRejection|TestMarkdownLinkRealAdapterFlushBarriers|TestEditorHostStaleLocalReferenceIsRefused','-count=1','-timeout','120s','-v')
```

PASS selected app tests in `1.818s`. All pointer, keyboard, readonly-anchor,
rejection, successful/failed Flush, and closed/inactive stale-callback subtests
passed. The failed-Flush subtest retains source/version/selection/history, view
and draft recovery controls and does not open a target. This checkpoint does
not claim that the full `./app` package passed.

```powershell
native\scripts\go.ps1 -GoArguments @('vet','./app')
```

PASS with no output.

Narrow visual evidence:

`artifacts/acceptance/markdown-links-focused-20261108-1644/markdown-links-wrapped-760x480-dark.png`

The image is `44,500` bytes, SHA-256
`EED9A8B3434F72F73691940EEBC960A34ADA3D644D627BF307DF8E9BFCB582A9`.
It was regenerated with the existing `TestMarkdownLinksNarrowScreenshot` and
`LEVNOTE_MARKDOWN_LINKS_SCREENSHOT_DIR`, without rerunning the focused suite.
PASS selected app test in `1.300s`. The image is byte-identical to the
root-reviewed 16:28 capture (same size/hash), confirming the final dense-row
fallback and palette changes do not alter this normal one-link fixture. The
test checked the full authored label
wrapped over multiple lines, the bounded accessible name, the neighboring task
checkbox, the host refusal in the status row, and all relevant bounds above the
status row. Direct image inspection confirms the updated dark link foreground
has clear contrast while preserving the accepted geometry.

Astra corrected the previously isolated synchronous-release crash by retaining
the active List state for the toolkit frame while swapping the released view to
a fresh state. The real Ctrl/Cmd-click path, Tab+Enter path and successful
pending-Compose local-open path now pass without an app workaround.

The earlier stale source callback finding and membership/current-tab guard
below remain accepted and green.

Astra bounded the isolated dense-row allocation by falling back the whole row
to readable formatted text without interactive RoleLinks when it has more than
16 links or more than 64 parsed inline spans. Her owned package checkpoint is
PASS `2.050s` with vet clean. That fallback does not affect this normal
one-link fixture.

Root then froze product source and authorized one final integration. Full test
PASS exit0/8.998s (`app`5.454s), full vet PASS exit0/2.452s with zero
diagnostics, and the stripped Windows GUI release build PASS exit0/3.066s.
Current executable:
`artifacts/acceptance/markdown-links-20261108-164403/levnote-native.exe`,
`13,252,608` bytes, SHA-256
`5BFB5A445CC619FEA9380F0363D3DF26EC4F3FF944C8549B1992F5B462EF3F10`.
The hidden PID-scoped startup/`WM_CLOSE` smoke passed exit0 while the window
remained hidden; source/copy hashes matched and updater-shaped sentinel
files/directories survived.

Exact aggregate evidence:

- `artifacts/acceptance/markdown-links-20261108-164403/result.json`
- `artifacts/acceptance/markdown-links-20261108-164403/OUTPUT-PROVENANCE.txt`
- `artifacts/acceptance/startup-smoke-markdown-links-20261108-164536/result.json`
- `artifacts/acceptance/startup-smoke-markdown-links-20261108-164536/smoke-meta.json`

The smoke directory preserves both harness-only failures and their encoding /
missing-`Get-FileHash` corrections; neither was a product failure. No resource
process sampling ran. Root artifact acceptance remains pending.

## ACTIVE acknowledgement — 2026-11-08

I acknowledge the active `markdown_links_stage_plan.txt` and the final contract
in `handoffs/markdown-links-contract.md`. Ownership remains limited to
`native/app/**` tests, app evidence and a minimal shell liveness correction
only if an actual stale source callback demonstrates the need. Astra owns the
native modifier-sensitive input probe and `modules/markdownview/**`; GLM owns
independent read-only review. No product change is being made from the
historical candidate design.

Astra's probe must first establish the exact public primitive behavior:

- ordinary plain pointer selection and double-click editing inside a link;
- Ctrl/Cmd primary-click activation;
- Tab then Enter keyboard activation and Space non-activation;
- wrapped-fragment hit testing and modifier changes, press/cancel behavior;
- exact RoleLink accessible name, tooltip and unavailable metadata budgets;
- source-identity stale/context refusal semantics.

## Compact app acceptance plan

Once Astra publishes those semantics with a focused checkpoint, app tests will
use one real registered Markdown adapter rather than a test-only click
shortcut:

1. Open a saved fixture containing a relative local-file link, same-document
   Unicode/duplicate anchor, unsupported URL and a wrapped link label. Assert
   native RoleLink tree names/bounds and that preview/plain selection remains
   usable.
2. Activate the local link via the agreed pointer gesture and via Tab/Enter.
   Verify ordered `Open`/OpenMany behavior selects or opens the target exactly
   once, resolves relative paths from the saved source directory, and uses no
   browser or OS shell route.
3. Activate Unicode/duplicate anchors in preview and split. Verify the shared
   outline resolver is used, preview does not focus hidden source, split follows
   the accepted source-plus-preview policy, and readonly navigation remains
   allowed.
4. Start native Compose in another block before activation. A successful Flush
   precedes navigation; a failed Flush retains source/draft/caret/view and
   visibly refuses before classification or opening.
5. Refuse `https`, protocol-relative, mail, data/javascript, UNC and remote
   `file:` targets without tabs, source bytes, dirty state, history or focus
   changing. Unsaved relative references refuse explicitly.
6. Capture the source tab's `OpenReference` closure. Close or replace that tab,
   then invoke the stale callback. It must not open a local target or resolve an
   anchor against another tab. Implement only a membership/current-tab guard
   after this test demonstrates the existing gap.
7. Open/export/navigation without edits must leave exact BOM/CRLF bytes, undo
   history and dirty state unchanged; save/reopen preserves authored source.
   A narrow `760x480` wrapped-link screenshot must keep tabs, modes, link bounds
   and status separated. Task checkbox controls remain present in at least one
   composition fixture.

No parser expansion, relative-path percent decoding, file+anchor splitting,
network support or vendor patch is included. Tester input is not claimed as an
OS IME candidate-window or full accessibility acceptance.

## Initial app-boundary finding to demonstrate

Current `App.openReference(t, reference)` starts with a nil check, then resolves
a local target and calls the ordered `Open` route. It does not verify that `t`
is still a member/current tab. An anchor eventually enters
`activateOutlineTarget`, which does check `t == a.Current()`, but that check is
after local classification and is not shared by local-file navigation. A test
will capture a real Host callback, remove its source tab, invoke the callback,
and determine whether a local target is incorrectly opened. Any fix will be a
small shared guard before classification.

### Reproduced and corrected

`TestEditorHostStaleLocalReferenceIsRefused` captures the actual
`editorHost(source).OpenReference` closure in two reachable cases:

- remove the source tab, then invoke its closure with `target.txt`;
- switch to another tab and invoke the still-live but inactive source's closure.

Before the guard, both cases opened a new target tab. The initial output showed
the closed-source case growing from 0 to 1 tab and the inactive-source case from
2 to 3 tabs. The narrow correction validates the concrete tab pointer's
membership and current status before classification, then reports either
`引用所属的文档已关闭，未执行打开。` or
`引用所属的标签已不再活动，未执行打开。`

Focused commands and results:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./app','-run','TestEditorHostStaleLocalReferenceIsRefused$','-count=1','-timeout','60s','-v')
native\scripts\go.ps1 -GoArguments @('test','./app','-run','TestResolveLocalReferenceClassification|TestEditorHostOpenReference|TestEditorHostAnchorUsesOutlineSourceResolverAfterFlush|TestEditorHostStaleLocalReferenceIsRefused','-count=1','-timeout','90s','-v')
```

PASS, app `1.516s`; then PASS, app `1.631s`. This permits a stale view callback
to fail visibly without weakening normal local, anchor or ordered-open routes.
The only app product change is the shared pre-classification membership/current
tab guard in `app/reference.go`. Real modifier/pointer/keyboard link activation
and the narrow screenshot remain blocked on Astra's probe/checkpoint.

Root accepted this demonstrated minimal guard. For refused or unsupported
destinations, app tests will not require native focus to match the pre-gesture
focus: moving focus to the activated link is legitimate. The no-navigation
invariants instead require unchanged source caret/history/draft, no tab/open
request, no hidden source-editor focus, and no bytes changed beyond an actual
prior successful Flush. `vet ./app` passes after the guard.
