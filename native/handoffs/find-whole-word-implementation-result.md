# Whole-word Find implementation result

2026-11-10. Bounded implementation is complete and frozen for independent
review. The accepted `.16` package and historical evidence were not changed.
No build, host launch, physical UI run, resource claim or product-doc edit was
made.

## Changed source and tests

Production changes:

- `native/app/app.go`: added runtime `WholeWord bool`; it remains global
  runtime Find state like `CaseSensitive` and `Regex`.
- `native/app/state.go`: `findOptions()` now carries `WholeWord`, and the first
  Find row has a `全词` checkbox using the existing `KeepFocus` and Escape
  lifecycle.

Test/evidence changes:

- `native/app/find_whole_word_test.go`: actual `ui.Tester` checkbox click,
  next/previous/wrap, identifier/CJK/combining-mark boundaries, regex + case,
  Replace Current/All, grouped undo, readonly/failed-Flush preservation and the
  real app workspace-search entry point.
- `native/app/search_navigation_test.go`: the existing cold-cache test now
  proves toggling `WholeWord` causes one additional cold scan.
- `native/app/findbar_layout_test.go`: includes `全词` in bounds/overlap/tab
  coverage and pointer action coverage, proves Tab focus plus Escape closes the
  panel, and marks the checkbox active in both fresh captures.

No `core/search`, workspace traversal, replacement algorithm, cache structure,
session schema or dependency was changed.

## Actual commands and results

All Go commands used the required wrapper:

```powershell
& .\native\scripts\go.ps1 -GoArguments @('test','-count=1','-run','TestFindWholeWordCheckboxNavigationAndWrap|TestFindbarWholeWordEscapeAtMinimumWindow','./app')
```

Initial red was the expected missing wiring:

```text
app\find_whole_word_test.go:24:8: a.WholeWord undefined (type *App has no field or method WholeWord)
FAIL    levnote.local/native/app [build failed]
EXIT_CODE=1
```

After the minimal production wiring, the first focused run exposed two test
issues rather than product defects:

```text
workspace whole-word results = one standalone cat, want two
whole-word checkbox did not retain focus after pointer click
FAIL    levnote.local/native/app    1.989s
EXIT_CODE=1
```

The workspace fixture had only one eligible match (`cat_` and `_cat` are
correctly excluded). Checkbox `KeepFocus` intentionally leaves pointer focus on
the prior field, so the Escape test was corrected to reach the checkbox through
Tab.

Corrected focused run:

```powershell
& .\native\scripts\go.ps1 -GoArguments @('test','-count=1','-run','TestFindWholeWord|TestFindCacheReusesColdOffsetsAndTracksSourceVersion|TestFindbarControlsFitMainPaneAtMinimumWindow|TestFindbarActionsAndFocusSurviveMinimumReflow|TestFindbarTabFocusAtMinimumWindow|TestFindbarWholeWordEscapeAtMinimumWindow','./app')
```

```text
ok      levnote.local/native/app    2.153s
EXIT_CODE=0
```

Fresh capture run:

```powershell
$env:LEVNOTE_FINDBAR_LAYOUT_SCREENSHOT_DIR = (Resolve-Path 'native\handoffs\find-whole-word-evidence\screenshots').Path
& .\native\scripts\go.ps1 -GoArguments @('test','-count=1','-run','^TestFindbarLayoutScreenshots$','./app')
```

```text
ok      levnote.local/native/app    1.561s
EXIT_CODE=0
```

One final affected suite:

```powershell
& .\native\scripts\go.ps1 -GoArguments @('test','-count=1','./app','./core/search')
```

```text
ok      levnote.local/native/app          15.298s
ok      levnote.local/native/core/search   0.742s
EXIT_CODE=0
```

One final vet:

```powershell
& .\native\scripts\go.ps1 -GoArguments @('vet','./app','./core/search')
```

```text
EXIT_CODE=0
```

Raw logs are retained in `native/handoffs/find-whole-word-evidence/`:
`initial-unwired-focused.log`, `focused-1.log`, `focused-2.log`,
`screenshots.log`, `final-affected-suite.log` and `final-vet.log`.

## Assertions exercised

- The visible `全词` checkbox updates runtime state and actual Next/Previous
  actions select standalone `cat`, skip `category`, `_cat` and `cat_`, and wrap.
- Literal and regex searches retain the existing Unicode boundary rule:
  letter/number/mark/underscore are word characters. Standalone `猫` is accepted,
  `猫咪` is not split, and `e` before a combining mark is rejected.
- Regex plus case-insensitive and case-sensitive interactions use the same
  whole-word checkbox without changing query/replacement text.
- A warm navigation cache performs one new cold scan and stores the filtered
  offsets after `WholeWord` changes.
- Replace Current and Replace All use the shared option. Replace All is one
  undo group.
- Readonly and failed rich Flush leave text, buffer version and selection
  unchanged.
- The local workspace-search fixture returns only the standalone match, proving
  app workspace search receives the shared option.
- At `760x480`, light/dark/default font and maximum editor font retain control
  bounds, no overlap, usable editor/status geometry and correct tab traversal.
  Tabbing to `全词` and pressing Escape closes Find.

## Fresh captures

These are tester-rendered images, not physical OS input, IME or resource evidence:

- `native/handoffs/find-whole-word-evidence/screenshots/findbar-light-760x480.png`

  SHA-256 `68D6F293F5E3F95CD9AFA4032E57916AF8C33494C3875E1FCE665D8FFECE1FAC`

- `native/handoffs/find-whole-word-evidence/screenshots/findbar-dark-760x480.png`

  SHA-256 `C72BF002E237A3D8E43F843D88236EA4F634326189A25232978EF8FAB502CD3E`

Both show the checked `全词` control, all Find actions and status, a usable
editor, and remain within the `760x480` window.

## Frozen file hashes

| File | SHA-256 |
|---|---|
| `native/app/app.go` | `58B6A783968C8860BBE6E8EF87E74CE9C6BEA3560BD4C19A569FC44465557931` |
| `native/app/state.go` | `023C76A143CB572D0A2B45D783A20E105996FF570615357BB1C3F31032E79984` |
| `native/app/find_whole_word_test.go` | `1AA1E1047DE5EE917D34097FC044784D992E4F6D79EC8C0FBC7F918BA20E6AF9` |
| `native/app/findbar_layout_test.go` | `662C167BFFA6CC344EF2E2A73E3566824E03B0DA3BA3D19FA463279FFCF12180` |
| `native/app/search_navigation_test.go` | `7773D1C1DB801CD640BB09D329806C040E905B87AD8E9712E61FD509399B30BD` |

## Limits

- `ui.Tester` establishes rendered-layout and synthetic input behavior only; it
  does not establish physical Win32 focus, OS IME behavior, accessibility or
  visual behavior of the packaged host.
- Whole-word means the current Unicode boundary predicate. It is not Chinese
  word segmentation, syntax-aware identifier matching or parser behavior.
- No build, package, launch, process memory, CPU or release-readiness claim is
  made.

Source, tests and captures are frozen and ready for independent review. This
result does not self-accept the feature.
