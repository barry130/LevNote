# Image keyboard command manifest

Working directory: `F:\AIproject\LevNote`

Logs without a separate `.exit.txt` contain the recorded exit in their final
`EXIT=` line. Final runs have both a `.log` and `.exit.txt`.

| Id | Exact command | Exit | Evidence |
| --- | --- | ---: | --- |
| imageview-attempt1 | `.\native\scripts\go.ps1 -GoArguments @('test','./modules/imageview','-run','TestImageKeyboard','-count=1')` | 1 | `focused-imageview-attempt1.log` |
| imageview-attempt2 | same command as imageview-attempt1 | 0 | `focused-imageview-attempt2.log` |
| imageview-attempt3 | `.\native\scripts\go.ps1 -GoArguments @('test','./modules/imageview','-run','^TestImageKeyboardFocusAndInputContainment$','-count=1')` | 0 | `focused-imageview-attempt3.log` |
| app-attempt1 | `.\native\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestNativeImageKeyboardByteAndTabBoundaries$','-count=1')` | 1 | `focused-app-attempt1.log` |
| app-attempt2 | same command as app-attempt1 | 1 | `focused-app-attempt2.log` |
| app-attempt3 | same command as app-attempt1 | 1 | `focused-app-attempt3.log` |
| app-attempt4 | same command as app-attempt1 | 1 | `focused-app-attempt4.log` |
| app-attempt5 | same command as app-attempt1 | 1 | `focused-app-attempt5.log` |
| format-corrections | `.\native\scripts\go.ps1 -GoArguments @('fmt','./modules/imageview','./app')` | 0 | `format-corrections.log`, `format-corrections.exit.txt` |
| imageview-final-attempt1 | `.\native\scripts\go.ps1 -GoArguments @('test','./modules/imageview','-run','TestImageKeyboard','-count=1')` | 0 | `focused-imageview-final-attempt1.log`, `focused-imageview-final-attempt1.exit.txt` |
| app-final-attempt1 | `.\native\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestNativeImageKeyboardByteAndTabBoundaries$','-count=1')` | 0 | `focused-app-final-attempt1.log`, `focused-app-final-attempt1.exit.txt` |
| full-suite-correction-attempt1 | `.\native\scripts\go.ps1 -GoArguments @('test','./modules/imageview','-run','^(TestExplicit100PercentFitAtOneRemainsManualAcrossResize|TestOneRequestAndPublicToolbarInput)$','-count=1')` | 0 | `focused-full-suite-correction-attempt1.log`, `focused-full-suite-correction-attempt1.exit.txt` |
| full-suite-correction-attempt2 | same command as full-suite-correction-attempt1 | 0 | `focused-full-suite-correction-attempt2.log`, `focused-full-suite-correction-attempt2.exit.txt` |

Failed attempts are preserved in full:

- `imageview-attempt1`: the first plus from a sub-floor fit did not enter the
  manual zoom floor.
- `app-attempt1`: build failure because the test called the nonexistent
  `document.BinarySnapshot.SameBaseline`; the test now compares tokens.
- `app-attempt2` through `app-attempt4`: each retained the same wrong
  `HasText("125%")` observation. Tester text exposes the `当前缩放比例` label,
  so the assertion could not observe displayed text.
- `app-attempt5`: source compiled and reached the byte editor, but the test
  incorrectly required `TextInputActive` from the intentionally readonly,
  navigation-only editor. It now verifies the public `只读原始字节` focus target.
- `full-suite-correction-attempt1` passed, but root review found its `20x10`
  resize remained fit-equivalent at 100% in either mode. The final
  `full-suite-correction-attempt2` uses admitted `400x200`, resizes to
  `300x240`, verifies cropped/clamped manual 100% for both activation paths,
  then verifies public `0` restores fit below 100%.

The combined app/imageview suite and vet were not run; those remain held for the
coordinated post-autosave freeze stated in the active task.
