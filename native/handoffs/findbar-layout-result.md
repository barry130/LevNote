# Find-bar minimum-window layout result

2026-11-09. Implementation and scoped verification complete. Source and tests
are frozen pending root acceptance. No build, package, product docs, helper,
launch, or physical OS/IME/resource claim was made.

## Geometry and product change

The native minimum window is 760x480 (`native/cmd/levnote/main.go`). The fixed
sidebar leaves the main pane at x=228, width 532, and the fixed status row starts
at y=451.

The original geometry test produced a real overflow before any production
change:

```
Find-bar control "正则" is outside the main pane above status at 760x480:
ui.Rect{X:759, Y:101, W:50, H:16.510254}
```

That control ended at x=809. The minimal fix is exactly two `.Wrap()` calls on
the two existing Find-bar rows in `native/app/state.go`. Widths, labels,
actions, focus behavior, and search semantics/cache are unchanged.

After the fix, light/default, dark/default, and the largest existing editor-font
setting (32) have identical Find-bar geometry:

- controls start at x=240; the farthest right edge is the close action at
  x=748;
- the last row ends at y=238.041 below the status top y=451;
- controls do not overlap;
- the useful text-editor area is
  `ui.Rect{X:228, Y:246.04102, W:532, H:203.95898}`.

## Interaction coverage

`native/app/findbar_layout_test.go` uses MyGo Tester controls for:

- query and replacement typing;
- query composition active before an actual 1180x780 to 760x480 resize, with
  focus and value retained across reflow, then composition committed after
  reflow (`needle猫`);
- previous/next navigation, including a wrap between the two matches;
- case-sensitive and regex option actions;
- replace current and close;
- Tab traversal from query through every listed Find-bar control.

This is rendered-toolkit evidence, not a physical Windows IME or keyboard-flow
test. The separate read-only Ctrl+F/Enter/Escape review is outside this change
and was not implemented.

## Fixture corrections kept separate from the product red

The original overflow above remains the only product failure. Two later harness
issues were not layout defects:

1. An expanded fixture created a rich Markdown tab but queried the accessibility
   role/name of the plain source editor. Its zero rectangle was corrected by
   exercising the intended `text` source tab; the real editor geometry above
   then became available.
2. The screenshot test resolves a relative output path from its package working
   directory. The first successful run therefore placed the exact captures
   under `native/app/native/handoffs/findbar-layout-evidence/`. They were moved
   to the task evidence directory without rerunning the capture test.

The earlier failed expanded run is preserved in
`findbar-layout-evidence/focused-layout-actions.log`; it is not presented as
another product red.

## Captures

Both captures are 760x480 PNGs and were visually inspected:

| File | Bytes | SHA256 |
|---|---:|---|
| `findbar-layout-evidence/findbar-light-760x480.png` | 29,487 | `1860EB9E8E96F8F1EA51052CC907AA1D3D188B73122C565969D9045C8EA599CB` |
| `findbar-layout-evidence/findbar-dark-760x480.png` | 29,464 | `BF139E982E55183C515FA0965033643B748C611598ACDF5B28A89F3CA6DAA2F3` |

All fields, actions, and the status row are visible; the four-row Find-bar
layout is legible in both themes and leaves a useful editor area.

## Actual checks

- Focused layout plus accepted Find/search-navigation tests:
  `PASS`, exit 0 (`focused-layout-search-pass.log`).
- `TestFindbarLayoutScreenshots`: `PASS`, exit 0; package time 2.027s
  (`screenshot-run.log`).
- Final affected suite:
  `ok levnote.local/native/app 16.235s`, exit 0 (`final-app-suite.log`).
- Final `vet ./app`: exit 0 with no output (`final-app-vet.log`).
- Format: exit 0 with no output (`final-fmt.log`).

No additional suite or build was run.

## Frozen files

| File | Bytes | SHA256 |
|---|---:|---|
| `native/app/state.go` | 7,475 | `59E1DF87541AC786D2CE5603AD10CBCE14CF4B1D9775358B1241E66FAB461D7E` |
| `native/app/findbar_layout_test.go` | 6,509 | `F20B0A9F777D8DE80632092DB3A1BE5FAEEF5E32AF392FF9DF03896C82193D1C` |

Evidence hashes:

| Evidence | SHA256 |
|---|---|
| `red-minimum-geometry.log` | `87F3351CF09B535155452831DDEBE8CBCFC59D2F09D57A78F5E1E93D8D4A8297` |
| `wrap-minimum-pass.log` | `C70C638ABB8809D72A57E26E9695ED0B1F73CA7D9ECA4E523C739140B623AA31` |
| `focused-layout-actions.log` | `0062719917E4652B3D843AADBCA6929728BA2F77837B6CBEF4D7432E63064F73` |
| `focused-layout-search-pass.log` | `4D040367F3154BC764F975152A15D6524CC1370984BD684F6C9D5FCBBBDD6ACE` |
| `screenshot-run.log` | `317A2DB6A307AC9F060C4657F7C0765FEA72E0DC0BEFB12CA91A101EA6334D40` |
| `final-app-suite.log` | `C0F79694E57FA65C034D639E7D36D99FFBE346DA27DA43E569A4CADBBD4093DD` |
| `final-app-vet.log` | `B92E5654B20650D1EA2F8A48C67962FBC1228103B1174E5B3093A7822BF56B29` |

Freeze rule: no further product source, test, evidence, or capture edits before
root acceptance.
