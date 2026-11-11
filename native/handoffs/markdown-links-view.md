# Markdown native link view — active checkpoint

## Owner checkpoint ready for freeze — 2026-11-08 16:42 +08:00

Product implementation is complete within modules/markdownview only. Files:
links.go (render/control/identity/palette/budgets), rows.go/blocks.go (immutable
snapshot and native inline composition), view.go/preview_limit.go (single
captured gesture, list-state ownership swap). New links_test.go and
links_probe_test.go exercise the public input route. No parser/vendor edits.

```powershell
$env:LEVNOTE_MARKDOWN_SCREENSHOT_DIR='F:\AIproject\LevNote\native\artifacts\acceptance\markdown-links-view'
native/scripts/go.ps1 -GoArguments @('test','./modules/markdownview','-count=1','-timeout','120s','-v')
native/scripts/go.ps1 -GoArguments @('vet','./modules/markdownview')
```

Complete owned package PASS2.050s; vet exit0/no diagnostics. Includes prior
task/outline/draft/oversize regressions and all new link cases: pointer and
keyboard, wrapped fragments and union gap, plain drag+copy/double-click,
readonly, cancellation/modifier-loss, actual keyboard capture then duplicate
source insertion, exact enclosing+syntax/full-href journal checks, same-list
versus other-block Compose, failed Flush retention, foreign group, Restore,
expiry, replaced Buffer, edited/deleted target, open/closed fence, Partial and
oversize. Native synchronous Host.OpenReference->Release is covered: Release
swaps the ListState pointer, leaving the current List build's state valid.
No activation falls back from missing captured identity to a newly drawn item.

## Final rendering bounds and resource evidence

Root approved per-row eligibility: at most16 parsed links AND at most64 parsed
InlineSpan values for a linked row. Exceeding either makes the WHOLE row a
readable formatted Span preview, with theme-aware link color but no underline,
RoleLink/focus/target copies. No first-N interaction and no shortened target.
The original32KiB row shaping guard remains; no new label-length cap was added.
One nearby notice is drawn ABOVE the row, including readonly/list-item cases:

- >16 links: `此段链接较多，已显示完整文字；请切到源码查看目标`
- >64 spans: `此段行内格式较多，已显示完整文字；请切到源码查看链接目标`

Plain selection/double-click still works and draft retains exact original
targets/trailing spaces. Boundary16/17 and64/65 regressions pass; independent
task checkbox state remains available. Empty disabled controls and over4096
href use the earlier specific unavailable reason.

Initial700-link single row five unchanged frames:1.0331694s,
2,150,546,456B/70,028alloc. A focused allocation profile attributes99.49% to
MyGo internal/text.DecodeSpans called from underline Decorate per glyph run;
this is the reason merely switching overflow to underlined Span would not fix
it. Profile commands: focused test -memprofile
artifacts/acceptance/markdown-links-view/dense-before.pprof and
go tool pprof -top -alloc_space against the saved dense-probe.test.exe.
Those artifacts describe the PRE-FIX implementation, not the final binary.

Pre-bound five-frame curve:8links0.5275ms/171,560B;16links0.5395ms/663,360B;
32links1.5938ms/2,632,832B;64links5.2525ms/10,504,656B.

Final sample (fixture/parser construction outside unchanged-frame measurement):

| Fixture | Five unchanged frames | Cumulative allocated bytes / allocations | Built rows / pending frame |
|---|---:|---:|---|
|700 links in one row, visible fallback|4.6967ms|3,935,736 /106|1 /false|
|1000 task+link rows, at row900|1.0583ms|98,256 /1640|16 /false|
|One30KiB ASCII link label,265px|2.119ms|184,208 /228|1 /false|
|16 links among512bold spans,265px, visible fallback|1.2539ms|3,935,648 /102|1 /false|

The30KiB label cold view was19.2355ms/7,415,864B; styled cold view8.6624ms/
3,433,120B. Before style fallback the latter unchanged sample was8.4838ms/
6,537,488B. These are headless cumulative allocation/time samples, NOT process
RSS or runtime idle CPU. No Tauri comparison or new process measurement.

Theme contrast: former shell accent/background dark3.331:1. Final light
#24675d is6.618:1 on background/6.021 on Surface; dark#7dd3c4 is10.113/8.502.
All exceed4.5:1 for16px normal text; readonly navigation has the same colors.
Owned light/dark × writable/readonly265px screenshots are in
artifacts/acceptance/markdown-links-view. Inspected light/dark plus
markdown-links-dense-fallback-265px.png: notice readable, controls/source
guidance retained, no content over the bounded status area in normal fixture.

Remaining acceptance belongs to root/DeepSeek: one fresh normal narrow app
image, independent review and final integration/build/smoke. No full suite or
release build run by this owner. Native tester Compose/modifier events are not
an OS candidate-window/physical keyboard or screen-reader acceptance claim.

## Initial probe and UI contract (superseded only by bounds above)

Native feasibility passed: public row ObserveInput records current event
modifiers before child HitTest; using Context.Modifiers from the event callback
alone is a deliberate failing negative control. No vendor/internal API needed.
Focused command: go.ps1 -GoArguments @('test','./modules/markdownview','-run',
'TestNativeInline','-count=1','-v') passed1.138s. Actual native tester covers
plain drag+copy starting in link, ordinary double-click reaching row edit,
Ctrl/Cmd click, Tab+Enter, Space not activating, first/last wrapped fragments,
non-link space inside accessibility union bounds, canceled press and modifier
loss before release. The added plain-down -> HoldModifiers(Cmd) -> actual
Release case uses the public tester's Mods=0 release; it does not claim an OS
release retaining Cmd. Product implementation is now in progress, not blocked.

## Exact UI contract for application acceptance

- Native RichText inline text child, never ui.Link. RoleLink accessible name
  `链接：<label>`. Label is parsed visible text, capped128 UTF-8 bytes at a rune
  boundary plus `…`; empty label displays/names `（空链接）`.
- Normal tooltip: `Ctrl/Cmd+点击或 Tab 后 Enter 打开：<href>`; href detail capped
 512 UTF-8 bytes plus rune-safe `…`. The destination passed to Host is the full
  unmodified href, copied only when at most4096 bytes.
- Unsupported local/external syntax is not classified in the view: same normal
  control/name; Host.OpenReference returns visible host-policy refusal. No OS
  browser route. Readonly controls remain enabled for navigation.
- Href over4096 bytes: name unchanged, RoleLink Disabled=true; original visible
  label remains with visible `〔链接不可用〕` suffix. Tooltip/reason:
  `链接目标超过4096字节；请在源码中查看，未打开链接`.
  With no host callback the equivalent reason is `当前宿主未提供链接打开能力`.
  Disabled links do not take plain pointer selection or activate by keyboard.
- Plain click/drag selects paragraph; plain double-click opens the existing
  independent draft. Only Ctrl/Cmd+primary click activates. Tab+Enter opens once;
  Space keeps native scrolling. Pointer hit areas are actual wrapped glyph
  fragments; access bounds are their union. Canceled gesture never navigates.
- Activation captures numeric buffer/version/enclosing and inline ranges plus
  bounded full href before redraw. Flush and current parser revalidation precede
  Host.OpenReference. Failed/stale activation goes through Host.Report with no
  view switch; a preceding successful Flush may still have changed source.
- App needs no test seam: AccessByRole(RoleLink,name) for metadata, Find(name)
  for first visible fragment coordinates, ClickAtWith(ui.Cmd,x,y), Tab/Enter.
  Tester.ClickWith prechecks modifier-free hit and is unsuitable here.

Implementation/focused command safety and final resource evidence are recorded
above. Existing task-checkbox release remains the accepted executable until
root accepts the final link-stage integration.
