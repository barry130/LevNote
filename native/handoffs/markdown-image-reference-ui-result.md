# Markdown image-reference UI: focused freeze

2026-11-10. Implementation and focused corrections complete; six owned files
frozen for root review. Format owner is independently correcting rejected
multiline envelopes. Final combined format/view/app test+vet is not yet run.
.25 package, evidence and product docs were not modified.

## Behavior and ownership

`InlineImage="image"` uses existing exact Range/raw Text/raw Href. Images and
ordinary links share16native controls/64inline spans per row and the existing
preview budgets. The native image control shows complete original Markdown,
uses a distinct `图片引用：` accessible name (including empty alt), and a bounded
tooltip explaining reference-only display. Dense fallback also preserves the
complete image syntax. Oversized rejected targets remain warned literal text;
no partial or disabled image descriptor is invented. Missing host is visibly
disabled. Code/bold/italic/table inner text remains literal; an existing outer
ordinary link retains its own destination.

Activation still goes through Host.OpenReference after Flush and exact mapped
range/reparse validation. Captured identity now includes inline kind and view
release generation, plus existing buffer/version/row/enclosing/syntax identity
and a bounded cloned target. It does not retain parsed source through a gesture.
The pressed immutable row snapshot lasts through release, then settles. Cancel,
scroll, owner change, release, stale source or image-to-link mutation never open
replacement content. Image auto-repeat/rapid pointer continuation is consumed;
fresh Ctrl/Cmd+click or Tab+Enter opens once. Plain drag/copy, doubleclick editing,
readonly navigation, draft composition Flush/conflict/recovery and ordinary-link
behavior remain covered. Synchronous host release swaps list ownership safely.

Runtime edits: modules/markdownview/links.go, blocks.go, view.go. No rows.go edit
needed: existing capability detection calls the shared counter. links_test.go
only adds new inline-kind/release fields to its existing independent capture
helper. New focused tests: modules/markdownview/image_reference_test.go and
app/markdown_image_reference_app_test.go. No format, app runtime, core, imageview,
decoder, plugin, session, vendor or route changes; no I/O while drawing.

## Evidence and correction history

Evidence directory: [markdown-image-reference-ui-evidence-20261110](markdown-image-reference-ui-evidence-20261110).

- `00-fmt-mixed-*`: initial named-file fmt rejected files from two directories,
  exit1 and no edit. Subsequent fmt was scoped separately by directory.
- `01-focused-*`: first view/new-image plus affected ordinary-link tests;
  markdownview PASS2.173s, app FAIL2.616s at PNG existing-image selection.
  **Outer result exit0 is erroneous as a Go outcome:** the command omitted
  `exit $LASTEXITCODE`. Original files are preserved. Exact numeric Go exit
  was not captured; actual stdout proves FAIL. See01-focused-interpretation.md.
- `02-png-diagnostic-*`: exact one failing PNG subtest, exit1, app1.718s.
  Actual state was tabs2/current==source, not a duplicate. Programmatic Select
  followed immediately by the same pointer position was a doubleclick
  continuation, correctly suppressed. Diagnostic output preserves the pointers,
  active index/path and message before any fixture timing change.
- Test correction: advance the synthetic clock1s before the explicitly fresh
  existing-tab activation. Rapid-repeat suppression stays separately asserted.
  A narrow deferred-invalidation condition is limited to an actual held inline
  gesture, avoiding frame scheduling outside that gesture. Added a substantive
  code/bold/italic/table/outer-link ownership test.
- `03-corrected-focused-*`: `test ./modules/markdownview ./app -run
  '^(TestImageReference|TestMarkdownImageReference)' -count=1 -v`, through
  native/scripts/go.ps1; explicit forwarded exit0, empty stderr.
  **markdownview PASS1.445s; app PASS2.151s.** Wall6.393s, timestamps and full
  command/streams in result. No passed ordinary-link tests repeated here.

Actual app action opens saved BOM/CRLF Markdown through the registered native
view, opens PNG by pointer/JPEG by Tab+Enter, and reaches the real40x24image
reader. A fresh second action selects the existing tab. It checks original
Markdown buffer/version/selection/undo/dirty state, exact BOM/CRLF disk bytes
after Save/reopen and original image bytes. Missing, unsaved-relative, remote,
UNC and unsafe-scheme actions report refusal; disabled image plugin preserves
readonly binary authority and byte inspection. No new resolver was introduced.

One265x480Tester image was captured in01 and viewed by owner/root; original
image syntax wraps within the pane, empty-alt and list/task controls remain
visible. [Capture](markdown-image-reference-ui-evidence-20261110/markdown-image-reference-265x480.png)
SHA2568323E8A7883D89217B1DBC57C7A4C5B98CC9A96ED916997152EBF13956C61412.
No later capture was produced.

## Frozen identities and limits

[04-source-freeze.json](markdown-image-reference-ui-evidence-20261110/04-source-freeze.json):
6files/82,822bytes, inventory SHA256
97093DD64DF3AD2FA6401A0FED1531864E6E06513D7ED4AD3E65D98F067A735F.

| File | Bytes | SHA256 |
|---|---:|---|
| modules/markdownview/links.go |11353|6F3283C9B0C36861A327713996FC10561EF2F0DF37292AE0C903E8480CC6D293|
| modules/markdownview/blocks.go |3247|1C1BBCD0282C97E5608E51D2380C50ED791BF27F6D727AB46127183E9B3E3821|
| modules/markdownview/view.go |19104|A720F9D028755DD3EBDEFC16B212C9A19BFA4283B69C238C38816CADDE9268DE|
| modules/markdownview/links_test.go |24226|6264D1490C4E225DCD61329D3FC0B910843F1621857B0AD6A4E54C8A6566A54F|
| modules/markdownview/image_reference_test.go |16963|9EF851FA5A414E0F0F26512822CC4E63AD62F128C9CC9E271A1C6C44E57000BC|
| app/markdown_image_reference_app_test.go |7929|A5CBA2E870AEBBDFD3CD4F9F0C41E1D60962278BCC542DE8C6973C944ABBB0AC|

Synthetic Tester/model assertions do not establish physical OS input/IME,
accessibility screen-reader behavior, resource usage, rendering drivers or
hidden host readiness. No tests/vet beyond the stated focused runs, package,
host, worker, resource run or capture repeat. Root coordinates the single final
affected formats/markdown+modules/markdownview+app test/vet after both freezes.
