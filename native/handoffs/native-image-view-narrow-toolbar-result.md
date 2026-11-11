# Narrow image toolbar result

FROZEN 2026-11-10 for root review and Astra's integrated app verification.

## Correction

The image toolbar is now a compact two-row column:

- `image-zoom-controls` contains fit, 100%, decrease, current percentage, and
  increase. The row wraps when needed.
- `image-details` contains the metadata and the `查看字节` action. Metadata
  grows within the remaining width, has `MinWidth(0)`, and keeps one-line
  ellipsis. The byte-view button retains its natural width and remains
  right-aligned inside the row.

The zoom percentage and metadata have stable accessible labels
`当前缩放比例` and `图片元数据` for bounds testing. Public MyGo layout and Tester
APIs are used; no app, editor, core, `modules.go`, or vendor source changed.

## Source identities

Changed and frozen:

```text
2B168A93E3D3B0140D3114CD282113B02CE4CA1944820A7FA37F9F853071FBAC  modules/imageview/ui.go
E4FE9CA693ABFDCC0535E629A1845965DA64CEECB7F251FEE7088F8854A36592  modules/imageview/imageview_test.go
```

Unchanged under this correction:

```text
E24EE00325B412DE2D67571808E103C7C96B0E35ACFE8C5E6C28C90D1AC0DFBA  modules/imageview/adapter.go
23B1DD60ACB7C6C76E823428DC2201931A868D4C37C5092CC7B6A8D1E5F52856  modules/imageview/geometry.go
AA1EB6A586C38579EFF6F65EF9276B9E4FE35C75FD87BFC0F89821E06943B346  modules/imageview/view.go
EAE500A2ACDD90D65EFAE2EF92117EB92E62353E4A048CD56B98A649783CBC0A  modules/imageview/input.go
```

## Focused verification

`TestToolbarControlsRemainReachableAtNarrowWidths` completed a synthetic image
result with deliberately long format metadata at 532x320 and 300x320. At each
width, public `Tester.Find` bounds for `适合窗口`, `按百分之百显示`,
`减小图片显示`, `当前缩放比例`, `增大图片显示`, `图片元数据`, and
`查看只读字节` were positive and fully inside the content pane. The test then
clicked the `查看只读字节` input target; hit testing succeeded and exactly one
source callback was observed.

Command:

```text
native/scripts/go.ps1 -GoArguments @('test','./modules/imageview','-run','TestToolbarControlsRemainReachableAtNarrowWidths','-count=1')
```

Stdout:

```text
ok  	levnote.local/native/modules/imageview	1.150s
```

Stderr: empty. Exit: `0`.

Evidence is in
`native/handoffs/native-image-view-narrow-toolbar-evidence-20261110/`. The
`02-*` compile-failure record and `01-*` PowerShell-host failure record are
preserved alongside the successful `03-*` run. No screenshot was generated;
the accepted imageview captures and prior result remain untouched.

The imageview source is frozen. Astra may now make one corrected integrated
image capture and run the final affected modules/app suite and vet against
these identities. Root's acceptance should precede resuming `.23` preparation;
product build, host, worker, and resource execution remain inactive.
