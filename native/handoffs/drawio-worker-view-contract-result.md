# Fourth worker view contract: result

FROZEN 2026-11-10 for root and independent review. This implements only the
neutral view-export request contract and graphview preparation/epoch layer. No
adapter advertises the fourth engine yet, and app/UI/source-mode integration
remains inactive.

## Contract

`ui/editor/export_request.go` adds value types:

- `ExportPage{Index int; ID string}`
- `ExportToken uint64`
- `ViewExportRequest{Page ExportPage; Token ExportToken}`
- optional `ExportRequestPreparer.PrepareExportRequest()`
- side-effect-free
  `ExportRequestTokenProvider.ExportRequestToken()`

`ExportEngine` gains only `RequiresView bool`, whose zero value preserves every
existing engine's behavior. `DrawioAdapter()` still has no `ExportEngine` field,
so there is no premature engine advertisement or app route.

## Graphview semantics

`PrepareExportRequest` is draw.io-only. It refuses an incompatible view, nil
buffer, open edit group, over-32-MiB source, pending graph draft, parse failure,
missing/invalid page, unreadable document, or page ID over 512 UTF-8 bytes. It
does not call `Flush`, edit/publish source, discard a draft, or fall back to
page zero.

Before optional cache synchronization it captures the host buffer, buffer
version, release epoch, selected page index/ID, and an internal page-selection
epoch. After synchronization it requires all of those identities to remain
stable. This rejects callbacks that attempt to retarget/select another page,
replace source, release the view, or open a group while the request is being
prepared. The returned `ExportPage` copies the selected page's exact `Index` and
`ID`; the token is the current export epoch.

The monotonic export epoch advances on:

- every actual parsed-cache replacement/invalidation when a parsed document
  existed;
- every actual selected-page transition through one centralized setter,
  including draft restoration and A -> B -> A;
- every `Release`, with later reuse still permitted.

Cache hits, repeated preparation, unchanged `Draw`, and unchanged `Warnings` do
not advance it. A separate internal page epoch lets preparation detect even a
reentrant A -> B -> A page change during synchronization. Release does not set a
permanent dead flag.

## Files

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/ui/editor/contract.go` | 4,107 | `4C13D2658B252E62D5A58991CF7D37174309B6B98E53FBCD3B5028211EE6DE2D` |
| `native/ui/editor/export_request.go` | 938 | `51AB9A485EDF9626A696C7BC5EE196E38E8A2B828EB7873C78664AC40FECAE41` |
| `native/modules/graphview/adapter.go` | 8,829 | `29AC4724019C7D72C2595C711BCD6B924F3DEAF25C96D5F5C81F97967847D6E8` |
| `native/modules/graphview/draft.go` | 12,984 | `B482887442D57103CB0B610D4469882CCDE32F2D665CBF96AC444154E19C1DA2` |
| `native/modules/graphview/export_request.go` | 4,159 | `910FFAB7AC8E7EFC53D056CA520FBBCD5DF0FF550233BC6B2B720B3F4582A7F2` |
| `native/modules/graphview/export_request_test.go` | 6,763 | `AA334E1E799CE303D88C6E8EF7AF54E9C9CD48EE733EB5671EAA0B5F3DA1B078` |
| `native/modules/graphview/preview_limit.go` | 1,276 | `D2AA08246538C4E92F09A729F6661A69890F971401620D11CBF0F1A96CCF65F0` |
| `native/modules/graphview/sync.go` | 5,379 | `05DC3FF0B8D759833FF294BC823DA8C10BB533A59FBAF97E254E152667E1EBBB` |

## Focused coverage

The new tests cover exact page copy/token, stable cache-hit/Draw/Warnings
tokens, A -> B -> A page invalidation, same-index source reparse, Release then
reuse, stale no-draft synchronization, all requested refusals, pending-draft
state preservation without Flush, and reentrant synchronization retargeting.

## Commands and raw evidence

Final affected test:

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./modules/graphview','./ui/editor','-count=1')
```

- stdout: `native/handoffs/drawio-worker-view-contract-evidence/final-affected-test.stdout.log`,
  96 bytes, SHA-256
  `14E027D6AF4554D9D256A0F199CD4301171D862164470609BB80B0D0049E3277`
- stderr: `native/handoffs/drawio-worker-view-contract-evidence/final-affected-test.stderr.log`,
  0 bytes, SHA-256
  `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`
- exit: `0`
- output:

```text
ok  	levnote.local/native/modules/graphview	9.718s
ok  	levnote.local/native/ui/editor	2.174s
```

Final affected vet:

```powershell
.\native\scripts\go.ps1 -GoArguments @('vet','./modules/graphview','./ui/editor')
```

- stdout: `native/handoffs/drawio-worker-view-contract-evidence/final-affected-vet.stdout.log`,
  0 bytes
- stderr: `native/handoffs/drawio-worker-view-contract-evidence/final-affected-vet.stderr.log`,
  0 bytes
- exit: `0`
- both streams are empty.

## Limits

This freezes only the prepared view contract and graphview-owned cache/page
state. It does not advertise the draw.io fourth engine, add source-mode/app
guards, alter `.18`, build a package, launch the UI, or claim physical
input/resource behavior. Those remain Stage 2 integration work after the
lower-layer and this contract are accepted.
