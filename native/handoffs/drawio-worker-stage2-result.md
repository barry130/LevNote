# Fourth fixed worker, stage 2 result

FROZEN-READY 2026-11-10. Stage-2 app integration is source and evidence frozen
for independent review. No `.19` build, package, managed worker request or host
launch was performed. Existing `.18` package/evidence and the accepted lower
layers remain immutable.

## Implemented behavior

The draw.io adapter now advertises the fourth fixed `levnote.drawio-svg`
capability with `RequiresView=true`. The app admits a draw.io export only in an
active preview/split view with a prepared view. Source mode, a missing view,
grouped input, oversize source and failed Flush are refused before preparation
or runner invocation with visible guidance.

Export admission captures the tab, document, buffer, editor, name, adapter,
capability, mode, view and version. It Flushes, rechecks ownership and the
32 MiB source limit, calls the side-effect-free request preparer, then rechecks
the same owners plus the buffer version, group state and view token before
enqueueing. The worker receives an independently copied
`engineipc.RequestPage{Index, ID}`. Both the result recheck and post-Save-dialog
recheck compare the captured document, buffer, editor, tab name, adapter,
capability, mode, view, version and token. The latest stale-owner regression
therefore covers document/buffer/editor/name changes, A-to-B-to-A page
selection, same-index reparse and Release.

Successful draw.io publication writes the selected second page's exact SVG to
an exclusive output while preserving the source file bytes, BOM, selection,
history, dirty state, view and mode. Markdown and mindmap requests still carry
`Page=nil`; their existing behavior remains independent of a mounted view.

The source editor shows disabled `选择页面后导出`; preview/split shows
`导出页面`. The package panel exposes Markdown, mindmap preview, mindmap SVG,
and draw.io SVG choices with count-neutral reset wording.

## Final verification

The final affected suite was run once after the Editor/name ownership
correction, with capture output enabled:

```text
$env:LEVNOTE_DRAWIO_STAGE2_SCREENSHOT_DIR = <repo>/native/handoffs/drawio-worker-stage2-evidence/captures
.\native\scripts\go.ps1 -GoArguments @('test','./modules/graphview','./app','-count=1')
```

Exit `0`:

```text
ok  levnote.local/native/modules/graphview  12.003s
ok  levnote.local/native/app                17.788s
```

The one final vet command also exited `0` with no stdout or stderr:

```text
.\native\scripts\go.ps1 -GoArguments @('vet','./modules/graphview','./app')
```

Evidence:

| Artifact | Bytes | SHA-256 |
|---|---:|---|
| `drawio-worker-stage2-evidence/final-affected-suite.stdout.log` | 92 | `3692D66F320E888D3401CC43C35D5D4D16F48000BA1429A3F96303D944BDFC53` |
| `drawio-worker-stage2-evidence/final-affected-suite.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `drawio-worker-stage2-evidence/final-affected-suite.exit.txt` | 3 | `13BF7B3039C63BF5A50491FA3CFD8EB4E699D1BA1436315AEF9CBE5711530354` |
| `drawio-worker-stage2-evidence/final-vet.stdout.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `drawio-worker-stage2-evidence/final-vet.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `drawio-worker-stage2-evidence/final-vet.exit.txt` | 3 | `13BF7B3039C63BF5A50491FA3CFD8EB4E699D1BA1436315AEF9CBE5711530354` |

The affected suite includes the stage-2 Tester captures. Each image is exactly
760x480 and was inspected for readable, in-bounds controls:

| Capture | Bytes | SHA-256 |
|---|---:|---|
| `drawio-worker-stage2-evidence/captures/drawio-source-export-guide-760x480.png` | 31,870 | `68DA680A522BE6C330F3DCE4335024260B21E114920245BBE90B79C56E59E0F0` |
| `drawio-worker-stage2-evidence/captures/drawio-fourth-engine-panel-dark-760x480.png` | 42,593 | `AD452E803C614DE16C5C679C85EDEF09E57B0C32F05733F4D64F47CA0577669E` |

## Frozen source and test identities

| File | Bytes | SHA-256 |
|---|---:|---|
| `modules/graphview/adapter.go` | 8,958 | `680840AF4782E3481BFDFEED3D0B73D1F3F2E623920183AEC99BB7981378EDF9` |
| `modules/graphview/export_request_test.go` | 11,060 | `3ACB6E4BA117BCBFA52D579072D45940EEA913849B0D1FBFDE8962D409F699D7` |
| `app/engine_packages.go` | 10,214 | `9D3B3298140B34FB222CFA36561361857C3D9293DF2F3C74B1F5A50068F9B8C7` |
| `app/engine_packages_ui.go` | 6,682 | `3187AD685DCD065F9377F739BB2E2EF7EC01A6DE1B30F1FB09D582F45ADEE0D8` |
| `app/engine_packages_app_test.go` | 1,890 | `6D23D76A60DFF830EF1AF5116EA0907DEFDBA594FAFF6C67247E47798B6523BC` |
| `app/export.go` | 1,470 | `86708F1E04F7434D9AA51551ABDD951552A7AE23A2FB6DC1FBC6CAE8F799DB67` |
| `app/export_worker.go` | 15,546 | `46345F7EADA34FF2D0C808F0CAF7C1DB4B9F4C78F88DB28C97EBE04DCAD3C48E` |
| `app/view.go` | 21,546 | `DDBE9D6C57DB7279BE579538E8734EDA8A0FBEC04FB65929A14DA8D18DD11AAF` |
| `app/export_worker_test.go` | 26,560 | `1CCEA1C3611B96D646F1B445457CF135D7B144D327118C045431DE45E4CD76FC` |
| `app/export_mindmap_worker_test.go` | 15,228 | `939BAB05DB3E0DE1B31D1B2AD82D6AB3E4149B56B0E967E789680DD309793133` |
| `app/export_drawio_worker_test.go` | 10,947 | `A578652B33E4C565C4364634B5836434E98CA90E80BB95DA31EEE237F7097B0B` |
| `app/drawio_worker_stage2_capture_test.go` | 1,968 | `934D3C0EB4973CFC5668806167DD53CFCB3BFAFE07789BD3AB66D98BA73E6A55` |

## Limits and next gate

The evidence is synthetic MyGo Tester and controlled-runner coverage. It does
not establish physical Windows input/IME behavior, whole-process or peak memory,
or user-observed latency. No table or resource matrix was run.

Independent integrated source review remains the package gate. After that review
is accepted, the staged `.19` delivery must run one actual managed draw.io
second-page request with joined cleanup, one hidden owned host startup/normal
close, four fresh GUI worker builds and the established package/manifest
checks. This result does not authorize or claim that delivery run.
