# Doc-block reference — app acceptance

## Current integration status — ACCEPTED

Root authorized the final integration after accepting the focused gate and
GLM's no-findings review. One full test/vet/build and one isolated hidden
startup/close sequence completed successfully. Root then accepted the exact
13,298,176-byte executable after reviewing the saved command, package,
executable, validation and smoke evidence. Exact accepted metadata,
commands, logs, package results, screenshots and smoke evidence are in
`handoffs/docblock-reference-integration.md`.

The accepted executable is now this doc-block reference release. The preserved
18:17 mindmap-fields binary is the previous accepted artifact.

## Ownership update — 2026-11-08

The stopped `astra_editor_finish` references below are historical. Root assigned
the native view checkpoint to `deepseek_reference_view`; DeepSeek keeps sole app
implementation/test ownership. That replacement checkpoint has now passed the
real app focused gate recorded below.

## Historical focused checkpoint — accepted, superseded by final integration

GLM's parser seam is frozen. The real-adapter app cases now cover pointer plus
focused Enter/Space, canceled press, pending Compose in an unrelated heading,
literal local target, same-document anchor, local `file:` URL, readonly no-op,
remote refusal, failed Flush draft retention, synchronous open/release, a
same-ID/title stale gesture, an unrelated insertion ahead of the held target,
and the exact 4097-byte disabled boundary.

The replacement view fixed stale-gesture reporting by retaining the old list
snapshot until pointer-up validation. The app's held-target rewrite now opens
nothing and visibly reports a `引用` safety refusal; the unrelated insertion
ahead rebases through the buffer edit journal and opens target A. The disabled
4097 fixture uses a raw-coordinate click at the disabled control's center and
freezes source bytes, selection, version and history; this avoids treating
`Tester.Click` lookup success as activation.

Complete focused command:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestDocBlockReference(RealAdapter.*|Over4096TargetDisabled|NarrowScreenshot)$','-count=1','-timeout','120s','-v')
```

PASS `levnote.local/native/app 1.858s` at
`2026-11-08T11:06:23.3766038Z`; wrapper exit `0`, wall time `4794ms`.

Artifact directory:

`native/artifacts/acceptance/docblock-reference-20261108-190618`

- `focused.log`, 2726 bytes, SHA-256
  `2C0F2B903AB4BE61DA1AA45BC19F384F237AFD10D882CD2B5C337F8A74E63365`
- `focused-result.json`, 405 bytes, SHA-256
  `4BFF8D2E63B94A8A195493D5DC73F029E1B459FA1FAE6ED66D2CEDBB4874A1EF`
- `docblock-reference-normal-760x480-light.png`, 35867 bytes, `760x480`,
  SHA-256
  `781716AB440E12EC40C459B8833B13E3DA01B2A4DC4B96487C93E854F5D8560D`
- `docblock-reference-normal-760x480-dark.png`, 36232 bytes, `760x480`,
  SHA-256
  `837B82236E60301DE04D1EEED61998B353E6C1094D687F17D0AE7B2DBAC0FCD9`
- `docblock-reference-refused-760x480-light.png`, 38519 bytes, `760x480`,
  SHA-256
  `F6B2A8DF4157C4288D0DFBCEF64007CCB200265183DED13478987938673C4E3E`
- `docblock-reference-refused-760x480-dark.png`, 38927 bytes, SHA-256
  `2108E252847D38B9112B53544F08F8EA9EBBB46E557A1C2FAD1001941E7C2E8B`

All four 760x480 images were visually inspected. The normal local target is
enabled in light and dark themes; both refused images show
`本地版不会打开网络地址、外部 URL 或系统外链接。` in the status bar. The
reference control stays above the status boundary in every case, and no chrome
or content clipping was found.

Current source fingerprints used by the focused run:

- `native/app/docblock_reference_app_test.go`, 19101 bytes, SHA-256
  `03B7DD00255FF87285E919D238098E26D8677159A889E46CDFAB175B1D99F31E`
- `native/modules/markdownview/reference.go`, 9681 bytes, SHA-256
  `DCF84B5A3A644994C8F7FDE50F3A04DF4859BE4A56CE6822541F8F05E0F0FEE1`
- `native/modules/markdownview/view.go`, 10639 bytes, SHA-256
  `D3C78E1DD190D7480AF14B80215581751072640A30966B3ED86AA2FED668538A`
- `native/modules/markdownview/blocks.go`, 3056 bytes, SHA-256
  `18D04F8E00F08CBFC4D027AAA6A651AEC4EC2084973434BB724081E79B160ABB`

Tooltip checking accepts an intact value up to 512 bytes or a rune-valid
512-byte prefix plus the UTF-8 ellipsis, for a final bound of 515 bytes. The
successful pending-Compose assertion additionally requires exactly one new
undo entry and no redo entry, so a visible source substring alone cannot claim
that the host Flush occurred.

Successful focused runs against unchanged app/view behavior:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestDocBlockReferenceRealAdapterAnchorNavigation$','-count=1','-timeout','120s','-v')
```

PASS `levnote.local/native/app 1.478s`; the anchor remains in the current tab,
reports `已定位：目标`, and preserves source/version/selection/history/dirty
state.

```powershell
native\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestDocBlockReferenceRealAdapterLocalGesturesFlushAndRelease$','-count=1','-timeout','120s','-v')
```

PASS `levnote.local/native/app 1.462s`: pointer, Enter and Space each flush the
pending unrelated composition and open the local target once; their canceled
focus press opens nothing.

Earlier focused runs remain valid for the unchanged cases: local/readonly/
file-URL/refusal PASS `1.555s`, failed-Flush retention PASS `1.541s`, and
compile-only PASS `1.314s`.

Focused gate: stop for root review. No full suite, vet, build, smoke or resource
sample has run in this stage.

## Acknowledgement — 2026-11-08

DeepSeek acknowledges the active `docblock_reference_stage_plan.txt` and final
`handoffs/docblock-reference-contract.md`. App ownership is limited to real
registered-adapter acceptance under `app/**`; shell policy changes are allowed
only for a concrete relevant defect. `deepseek_reference_view` owns
`modules/markdownview/**` native capture/gesture/lifecycle behavior and view
images. GLM owns the small
`formats/markdown/{types,docblock}.go` metadata seam and independent read-only
review.

The focused app plan will exercise the final registered Markdown adapter through
real navigation gestures rather than duplicate Astra's component matrix:

1. local target navigation after an unrelated source insertion, with pending
   Compose flushed exactly once and no unintended source/history mutation;
2. target-block edit, same-title/same-ID sibling distinction, Buffer
   replacement/Restore, failed or conflicting pending draft, and stale callback
   refusal without opening the wrong target;
3. literal local filename, local `file:` target, unsupported/remote/empty
   target, readonly source bytes/history, over-4096-byte target disabled state,
   and synchronous `Host.OpenReference`/tab `Release` behavior;
4. one representative narrow normal/refused app image after the native control
   checkpoint, using bounded labels/tooltips and complete source as the
   authority.

No product test, full suite, vet, build or resource sample is running before
the parser API and native compiling checkpoint are available. If the current
shell cannot distinguish an exact native gesture result or maps a stale
reference to a new target, I will record the reproduction before making the
smallest app-side correction. No child delegation, commits or pushes.
