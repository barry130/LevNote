# Doc-block reference — native view

## Current status — post-review correction PASS, frozen for GLM (2026-11-08)

`deepseek_reference_view` replaced the stopped Astra checkpoint. Ownership is
limited to `native/modules/markdownview/**`; app/parser/host policy remain
unchanged. GLM's final source identity finding is applied: `Clicked` requires
the captured and rendered gesture to touch the same Buffer and row ID, while
mapped source ranges remain the sole authority for insertion-ahead rebasing.
No commit, push, full suite, build, or resource matrix ran. Product source is
frozen for GLM's final read-only review unless a concrete defect is reported.

## Implemented behavior

- One ordinary native `打开引用` button is shown for a structurally valid
  `mode: reference` doc-block with non-empty, at most 4096-byte `src`.
- Pointer-down and focused Enter/Space capture the numeric Buffer/version,
  preview row ID, enclosing/header UTF-16 ranges and complete bounded `src`.
  No Block/Document closure, raw header copy, or oversize destination is
  retained.
- The pressed preview snapshot remains stable through physical release. On
  release, the captured ranges alone are checked with `MapRangeSince`; no
  comparison against newly rendered absolute ranges rejects an unrelated
  insertion ahead.
- Activation runs `Flush`, requires the same Buffer and a complete bounded
  parse, maps the captured enclosing/header ranges, then verifies the exact
  doc-block kind/range, empty `ReferenceProblem`, `mode: reference`, and full
  `src`. It uses the current cached complete parse when Buffer/version match
  and does not parse a second time merely to validate an unchanged snapshot.
- A complete source and its exact header are scanned in one UTF-16 cursor
  pass. `Host.OpenReference` is called exactly once and is the only target
  classifier/opener. No view state is read after that call, so synchronous
  tab switch/`Release` is safe.
- Stale target/construct edits, Restore, expired journal, replaced/nil Buffer,
  foreign groups, partial/oversize source, and losing the host mid-gesture
  refuse visibly. Press cancellation, ordinary release outside, and scrolling
  over the pressed button clear the gesture without a false refusal.
- Pending release cleanup is scoped to the captured Buffer/row ID, so an
  earlier visible reference cannot consume a later same-ID/title reference's
  click. Deferred Draw cleanup handles a release whose old widget disappeared
  and remembers no obsolete capture. The actual `Clicked` branch independently
  requires `captured.buffer == target.buffer` and
  `captured.rowID == target.rowID`; it does not compare absolute
  enclosing/header ranges, so a mapped insertion ahead still opens its target.
- A target over 4096 UTF-8 bytes is checked before cloning and remains
  disabled, non-focusable, visibly labelled `引用不可用`; the complete source
  is unchanged. 4096 bytes remains enabled.
- Title/kind/mode/tooltip copies are rune-safe and bounded. Inline with `src`,
  malformed headers, and other non-reference cases render bounded diagnostics
  and preserve raw/unknown source and payload.
- Readonly navigation remains allowed because opening adds no history. Pending
  unrelated Compose Flush produces the normal one undo entry and no redo;
  failed Flush preserves the same View/draft/source/selection/history and its
  `完成编辑` and `打开未同步副本` recovery controls.

## Post-review focused verification

All commands used the bundled wrapper and Go 1.27.3.

```powershell
.\scripts\go.ps1 -GoArguments @('test','./modules/markdownview','-run','^(TestReferenceSecondVisibleTargetKeepsItsGesture|TestReferenceNativeStalePressSourceIdentity|TestReferenceUnrelatedInsertionRebasesAndSameBlockEditRefuses)$','-count=1','-timeout','120s','-v')
```

PASS `levnote.local/native/modules/markdownview 1.120s`. The second-reference
case performs a real pointer press at the second rendered button of two
visible same-ID/title blocks, verifies that the captured row is the second row,
releases, and opens only `targets/b.txt`; its Enter path remains covered.
Stale release and unrelated insertion with and without an intervening frame
also pass.

```powershell
.\scripts\go.ps1 -GoArguments @('vet','./modules/markdownview')
```

PASS, wrapper exit `0`.

```powershell
.\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestDocBlockReferenceRealAdapter.*$','-count=1','-timeout','120s','-v')
```

PASS `levnote.local/native/app 1.889s`, wrapper exit `0`. This includes the
real registered adapter's stale press/mutate/release visible refusal, unrelated
held-target rebase, pointer/Enter/Space Flush, anchor, readonly/local/file URL,
remote refusal, synchronous Release, and failed-Flush draft retention.

```powershell
.\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestDocBlockReferenceOver4096TargetDisabled$','-count=1','-timeout','120s','-v')
```

PASS `levnote.local/native/app 1.677s`, wrapper exit `0`. The button is
disabled/non-focusable with visible `引用不可用`; raw-coordinate input changes
no tab, source, version, selection, or history.

The app owner separately captured and root reviewed four 760x480 real-app
images in
`native/artifacts/acceptance/docblock-reference-20261108-190618`:

- `docblock-reference-normal-760x480-light.png`, 35867 bytes, SHA-256
  `781716AB440E12EC40C459B8833B13E3DA01B2A4DC4B96487C93E854F5D8560D`
- `docblock-reference-normal-760x480-dark.png`, 36232 bytes, SHA-256
  `837B82236E60301DE04D1EEED61998B353E6C1094D687F17D0AE7B2DBAC0FCD9`
- `docblock-reference-refused-760x480-light.png`, 38519 bytes, SHA-256
  `F6B2A8DF4157C4288D0DFBCEF64007CCB200265183DED13478987938673C4E3E`
- `docblock-reference-refused-760x480-dark.png`, 38927 bytes, SHA-256
  `2108E252847D38B9112B53544F08F8EA9EBBB46E557A1C2FAD1001941E7C2E8B`

Normal shows an enabled local reference; refused shows the visible local-only
remote-target status. Controls stay above the status bar in every image.

The pre-correction checkpoint also passed the complete owned package in
`1.907s` and the full `^TestReference` suite in `1.219s`. Per root instruction,
the post-review correction reran only the affected component tests and vet; no
duplicate app run was made.

## Source fingerprint

- `modules/markdownview/reference.go`, 10091 bytes, SHA-256
  `319126A54EB1ED2E692EA11F5264EE28648F47040ACCEB76DDD996C05715D68D`
- `modules/markdownview/blocks.go`, 3056 bytes, SHA-256
  `18D04F8E00F08CBFC4D027AAA6A651AEC4EC2084973434BB724081E79B160ABB`
- `modules/markdownview/view.go`, 11255 bytes, SHA-256
  `F1E21EE19CC3A44E7A7A534CCA9C7C35BF27B5E7039B4993A33D494E95CA0AD7`
- `modules/markdownview/preview_limit.go`, 1657 bytes, SHA-256
  `A81DF2A503178A7283F98FFEEFA0C5288DCC78F6A06D96B52B73E7497E59B511`
- `modules/markdownview/reference_test.go`, 22374 bytes, SHA-256
  `26C702931FA06061D4384F32FA101B540030178D26AB61A3499C78B04EC9EA2D`

## Remaining gate

The post-review native correction has no known blocker. GLM owns the final
read-only source review; root owns the integration gate. Full
`./...` test/vet, build, smoke/resource matrix, and any parser/core/app changes
remain outside this checkpoint.
