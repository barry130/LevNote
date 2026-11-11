# Narrow image toolbar checkpoint

2026-11-10, before focused verification.

The imageview toolbar is now a compact two-row layout. The first row contains
the fit, 100%, decrease, current zoom, and increase controls with wrapping
enabled. The second row contains metadata and the byte-view action. Metadata
grows only within the remaining row width, can shrink to zero, and keeps its
ellipsis; the byte-view button retains its natural width.

Changed files and pre-verification SHA-256:

```text
2B168A93E3D3B0140D3114CD282113B02CE4CA1944820A7FA37F9F853071FBAC  modules/imageview/ui.go
9AAC63FAE26DE151EE86AF646B16CEF53B9DF4F0DC1234FA6974815983B3F337  modules/imageview/imageview_test.go
```

The new `TestToolbarControlsRemainReachableAtNarrowWidths` covers 532x320 and
300x320 with deliberately long format metadata. It checks every visible image
control and metadata label with public `Tester.Find` bounds, then clicks the
`查看只读字节` accessible target and verifies the source callback.

No screenshot was regenerated. The accepted imageview evidence directory and
all prior captures remain untouched.

The first harness attempt (`01-*`) stopped in Windows PowerShell before Go
because that host lacked `Get-FileHash`. The second attempt (`02-*`) entered Go
and exposed a missing `fmt` import in the new regression; no Tester assertion
ran. Both failure records are preserved. The import is corrected, and the
refrozen test identity is:

```text
E4FE9CA693ABFDCC0535E629A1845965DA64CEECB7F251FEE7088F8854A36592  modules/imageview/imageview_test.go
```

The corrected focused run has not yet started.
