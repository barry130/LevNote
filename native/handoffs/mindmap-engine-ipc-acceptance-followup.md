# IPC acceptance follow-up evidence

Status: COMPLETE 2026-11-09. Product/API sources remained frozen; this pass
changed only `native/core/engineipc/protocol_test.go` to sharpen the promised
pre-payload oracles.

## Concrete oracle additions

- Foreign request coverage now covers expected Markdown/actual Mindmap,
  expected Mindmap/actual Markdown, and expected Mindmap/wrong HTML format.
  Each full frame includes a six-byte source. `ReadRequestFor` and the legacy
  Markdown-only `ReadRequest` wrapper reject at request metadata and leave all
  six source bytes unread (`reads == 2`, `remaining == 6`).
- `EncodeRequestPrefix` now pins exact acceptance at Markdown 8 MiB and
  Mindmap 16 MiB, and rejection at each cap +1, without allocating fixtures.
- The successful Mindmap SVG response is checked against an independently
  constructed literal frame, including the whole-document warning and payload.
- The response decoder uses header+metadata only while declaring six payload
  bytes. Foreign known engine, same-engine wrong format, and wrong version are
  rejected at the metadata boundary: the delivered byte count is consumed, no
  payload is retained, and the decoder failure is terminal.

No product or API source was changed for these results.

## Final retained evidence

Fresh final folder:
`native/artifacts/acceptance/mindmap-engine-ipc-20261109-063836-followup-final/`

- `test.result.json` and `test.stdout.log`: catalog PASS 0.435s, engineipc PASS
  0.572s, overall command exit 0, measured 1243 ms.
- `vet.result.json` and `vet.stdout.log`: scoped vet exit 0, measured 478 ms,
  zero-byte output log.
- `source-before.json`: fingerprints captured before the final test run.
- `source-after.json`: fingerprints captured after the final test/vet run.
- Both `.stdout.log` files capture the command's combined stdout and stderr
  streams (PowerShell `*>`), so an empty vet log means no output on either
  stream.
- `native/artifacts/acceptance/mindmap-engine-ipc-20261109-063214-followup/`
  preserves the two actual failed attempts described below; its logs were not
  overwritten.

Final owned-source fingerprints:

```text
b064e0165f5e6577b460616465be6456da2a0027ee02a378827de3cdc87d46b2 1754 native/core/enginecatalog/catalog.go
10c4497d9891348ac803264f3997ee631effc93c25e2c9731f169a98bc4621dc 1484 native/core/enginecatalog/catalog_test.go
e1dc708e0391899430feb43b4ed2755938d570f11d3b0fa713778b1dd5a2a7da 8219 native/core/engineipc/protocol.go
fe829ae271e28d6d237e6605f03663dbd740295818511ceb68d425403e5e802c 4880 native/core/engineipc/json.go
663a6b44ee185dd8b1383955db529441ba63b2f10ad66946fbb72dc04f8f6ea5 3398 native/core/engineipc/decode.go
54f849d6c71fdb453e0a2926e813cd0877d9a776a0a588943669a8ebfd1db743 20014 native/core/engineipc/protocol_test.go
```

The five product/API files are byte-identical between before and after. Only
`protocol_test.go` changed.

`protocol_test.go` was 20015 bytes with SHA256
`759da8cb872be28df8096389461aeae7e43fad398fd8fcbc39c4b77528acd372` in the
final folder's before inventory. Immediately before the final test run, `gofmt`
removed one formatting byte, producing the tested 20014-byte source with SHA256
`54f849d6c71fdb453e0a2926e813cd0877d9a776a0a588943669a8ebfd1db743`. The after
inventory and current file both match that tested hash. The file is not claimed
unchanged; the exact sequence is edit, `gofmt`, then final test and vet.

## Retained failed attempts

The first failed attempt used a full foreign request frame but asserted that
the source had been consumed; the product correctly returned after metadata
and left the six source bytes unread. The second retained failure adjusted the
oracle toward an EOF probe that is not part of the product contract. Both
attempts remain in the preserved 063214 evidence folder. The final full-frame
oracle asserts the actual contract: `reads == 2` and `remaining == 6`.

The scoped commands run for final evidence were exactly:

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./core/enginecatalog','./core/engineipc','-count=1','-v')
.\native\scripts\go.ps1 -GoArguments @('vet','./core/enginecatalog','./core/engineipc')
```

No runtime, worker, app, view, script, vendor, package, process, network, or
full-repository execution occurred.
