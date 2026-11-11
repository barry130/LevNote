# Markdown tables — native view focused checkpoint

Status: FOCUSED PASS; product source is ready for root/source review and app
integration. Scope stayed inside `native/modules/markdownview/**` plus this
handoff. No parser, app, core, vendor, build, full-suite or resource-matrix
change was made.

## Final owned commands

Focused table suite:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./modules/markdownview','-run','Table','-count=1','-timeout','120s','-v')
```

PASS. Wrapper exit `0`, 2062 ms; Go package result
`ok levnote.local/native/modules/markdownview 1.078s`. The screenshot test was
the expected skip in this run because no image destination was set.

Complete owned package:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./modules/markdownview','-count=1','-timeout','120s','-v')
```

PASS. Wrapper exit `0`, 2726 ms; package result
`ok levnote.local/native/modules/markdownview 1.809s`. Accepted link, task and
doc-block-reference suites also passed.

Owned vet:

```powershell
native\scripts\go.ps1 -GoArguments @('vet','./modules/markdownview')
```

PASS. Wrapper exit `0`, 832 ms.

The screenshot gate was run separately with
`LEVNOTE_TABLE_SCREENSHOT_DIR` set to the artifact directory below. PASS,
package result 1.057 s, six 760x480/265x480 PNGs, all visually inspected.
`gofmt -d` on the changed owned Go files is empty.

## Native product behavior

- Valid header and body rows flatten into the existing `ui.List`; the delimiter
  is not a display row. Stable identity includes source span, block kind and
  physical table-row position.
- Responsive layout uses the committed List rectangle, 36 DIP List horizontal
  padding and 16 DIP row horizontal padding. Unknown first and resize frames
  choose the labelled stacked record. Grid starts only when actual usable row
  width is at least `columns * 112` DIP. One measured change requests one settle
  frame; idle frames do not keep invalidating. No document-wide `RowWidth`
  panning exists.
- Grid cells are equal width. Stacked physical rows show a distinct header or
  numbered data-row label and bounded `column + header` labels. Values remain
  literal, monospace 13 DIP, at most 3 lines with visible ellipsis. Empty and
  invalid ranges have a muted placeholder and a rune-safe tooltip.
- Cell and column tooltips are complete valid UTF-8 strings at most 512 bytes,
  including any final ellipsis. The visible hint states that ellipsis affects
  preview only and complete source remains available.
- Writable headers expose one ordinary `RoleButton` named exactly
  `编辑表格源码`. Readonly headers instead show `只读表格` and complete-source
  guidance, with no editor action.

## Action, Flush and source identity

- Pointer down and focused Enter/Space keydown capture numeric Buffer identity,
  version, row ID, complete table UTF-16 span, byte range and enclosing ranges
  from the row actually receiving native input. The complete source string is
  cloned only at that gesture start, after checking the 32 KiB bound.
- Pointer release/click is consumed only for the captured Buffer and row ID.
  Pointer cancel, release outside, scroll and source-equivalent virtualized-out
  rows open nothing. They also do not falsely report a stale-target refusal.
- Pending unrelated Compose is retained under the same draft object and host
  bytes when the action press is canceled. Actual pointer, focused Enter and
  focused Space activation each Flush exactly once, then open the complete
  `BlockTable.Range`. The real regression checks host text, one Changed call,
  undo depth 1 and redo depth 0.
- An unrelated insertion before a held table maps and rebases the target, with
  or without an intervening frame. A change inside the captured table,
  replacement/expiry/group/partial/oversize source or changed row identity is
  cleared and visibly refused with text containing `表格`.
- Activation Flushes before validating the mapped target, rechecks that the
  current parse still has the exact `BlockTable.Range`, and verifies the current
  complete source bytes. An unchanged buffer/version reuses the cached parse;
  it does not parse twice.
- The second-visible-table regression selects the expected source with
  `strings.LastIndex`, maps its exact UTF-16 start to the numeric header row,
  waits for responsive layout to settle, presses and releases the real native
  pointer at the second table, and asserts the second complete span opens.

## Draft and navigation semantics

- Whole-table editing reuses the existing independent source editor with
  `ShapeLimit=0`, complete table field/range and a 240 DIP panel. It does not
  impose future cell-shape restrictions.
- A failed Flush keeps the same View, draft, Buffer version, selection, history,
  source and recovery controls. The required `完成编辑` and
  `打开未同步副本` controls remain reachable at 760x480.
- Readonly is checked after Flush, during source opening and while a held action
  disappears. A readonly transition preserves both a prior pending draft and
  the host bytes, reports a visible readonly refusal, and creates no editor.
- Double-click on any valid table row or bounded recognized raw-fallback table
  first creates the same immutable Buffer/version/complete-source snapshot.
  Rename, delete, overlap, stale journal and insertion-before are handled by the
  existing exact draft lifecycle.
- A table over the 32 KiB preview bound has no bounded whole-table button. Its
  explicit large-block placeholder is exercised by a real double-click and
  opens the complete source in the existing editor; source bytes and history
  remain unchanged on open.
- `Release` and preview-cache drops clear table rows, gesture state, measured
  widths, List state and table flags while preserving accepted draft behavior.

## Focused coverage

`tables_test.go` exercises native pointer/Enter/Space and cancel; pending Compose
Flush; insertion-before rebase with and without a frame; same-table stale
refusal; two settled same-shape tables with a real second pointer; valid and
fallback immutable double-click; over-32-KiB source fallback; responsive
breakpoint convergence and stable row identity; 8-column literal/Unicode/empty
tooltip bounds; readonly and readonly-after-down; failed-Flush recovery; the
240-DIP conflict geometry; far-row virtualization; idle-frame stability; cache
Release; and light/dark responsive captures.

The temporary range-only diagnostic test was removed. The current suite has no
success-only table diagnostic test.

## App integration checkpoint

The registered app adapter can rely on:

- action label `编辑表格源码`;
- readonly label `只读表格`;
- pointer down / focused Enter / focused Space capture semantics described
  above;
- pointer activation Flushes pending unrelated Compose once before opening the
  complete table source;
- failed Flush and readonly retain the same draft/recovery controls;
- valid and bounded fallback double-click use immutable source identity;
- narrow widths show stacked labelled records, wide enough widths show a grid;
- over-32-KiB tables show source guidance and the existing double-click large
  block editor instead of the bounded table button;
- no cell editing, link activation or parser behavior is introduced by this
  view.

## Evidence

Artifact:
`native/artifacts/acceptance/markdown-tables-view-20261108-200101/`

| File | Size/Dimensions | SHA-256 |
|---|---:|---|
| `table-conflict-dark-760x480.png` | 19,656 B, 760x480 | `359C069500E6B88A42BB6828F0F970EFC526E5579718D27AD78FB1420AE52F54` |
| `table-conflict-light-760x480.png` | 19,605 B, 760x480 | `32B12ED39CF2D609AD79D838D589010294154359401F0BECBCB208C527BC9E30` |
| `table-grid-dark-760x480.png` | 19,854 B, 760x480 | `B9D729961043E47305C7E71CB8AD750B3711D80E018218146457021706DA8B32` |
| `table-grid-light-760x480.png` | 19,756 B, 760x480 | `2D5FA6672D879CBF4AC0E61835047EA16FC06336C24C39AEDF576CB4F6B6A86F` |
| `table-stacked-dark-265x480.png` | 19,506 B, 265x480 | `1D3BB15DEE3846A65D49CF79B1B82416AC3F1154CA409ECFD0C12A57D0CFD3A1` |
| `table-stacked-light-265x480.png` | 19,152 B, 265x480 | `2CF72F4FF6682D2FBE18C03EE79947580464FDA53AB9E487764925F9D2C7E90E` |

The 265 DIP files are asserted and captured as stacked, not renamed 760 DIP
images. The 760 DIP files are asserted as grids. The conflict files show the
fixed 240 DIP editor plus visible `完成编辑`, `打开未同步副本`, and
`放弃草稿` controls. The obsolete blank
`markdown-tables-view-debug-20261108-1955` directory was removed.

Changed owned source identities:

| File | SHA-256 |
|---|---|
| `blocks.go` | `92DAE58D7D24710809986480C687FCD6A56EE963213E5215267893360CBBC48F` |
| `draft.go` | `CE3FE734804EB3077FF5E1FCBF739D64FB57F2BCE61460C6E5874AA45A606C21` |
| `preview_limit.go` | `2952CAC523C322F27C7BB1842795A61AC471F98BED3D6F00EFC42CA9B391D65C` |
| `rows.go` | `875C547E66747A581115B97120019FDE4D73FF17C063A10C0E8ADAB8DD6F72F0` |
| `tables.go` | `EC7850D76942B1AB3F3F28BDB49C11538F65499DA8C54624756E1806A769C3ED` |
| `tables_test.go` | `EDE9DA865B2F84F4D4A02512A84E0DFEC193657928EA77B2DAC9B96D0E1B9A0A` |
| `view.go` | `7A097B0EC6FDD2DCAF7E55352831F68CAFD432D96BDF9880238410B15EEA1AF5` |

Root or GLM source review remains the gate. No commit, push, app-owned test,
full native suite, release build or resource sampling was performed by this
owner.
