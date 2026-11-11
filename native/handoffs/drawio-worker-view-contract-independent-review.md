# Prepared view export contract independent review

Status: complete, 2026-11-10. No remaining blocker found in the final prepared
view-export contract after reconciling the frozen reentry correction and the
final canceled-gesture warning follow-up.

This review covers the original view-contract result, the frozen reentry
correction, the final warning follow-up, and the actual eight source/test files.
It does not review or activate Stage 2 app integration. No product edit, test,
build, launch, or repeated passing suite was performed.

## Concrete findings and resolutions

### 1. Reentrant report callback could crash or overwrite callback-owned state

Original location: `native/modules/graphview/sync.go`, in `ensureSynced` before
the reentry correction.

Trigger: with a stale parsed cache and an active drag/connect/reparent gesture,
`ensureSynced` called `Host.Report` for the canceled gesture before it had read
`host.Buffer.String()` or published `lastSynced` and the parsed cache. A
synchronous callback could set `host.Buffer` to nil and make the following
`String()` call panic. It could also replace the buffer, release the view,
change source/version, or retarget the page, after which the outer
synchronization continued and published state owned by the callback.

Minimal repair: capture buffer identity/version, release epoch, page epoch,
page index, and synchronized buffer/version state before reporting; immediately
after the callback returns, refuse to continue if any identity changed. Apply
the same guard around the later draft-conflict report. Do not recover panics or
mark released views permanently dead.

Resolution: frozen in
[drawio-worker-view-contract-reentry-correction-result.md](drawio-worker-view-contract-reentry-correction-result.md).
The correction checks ownership immediately after the callback and before the
buffer read or cache publication. Tests cover nil/replaced buffer, release,
source change, page A→B→A, and a callback-created pending draft.

### 2. Successful reparse hid the canceled-gesture warning

Location: `native/modules/graphview/sync.go`, after the immediate ownership
guard introduced by the reentry correction.

Trigger: a normal stale-source reparse with an active gesture reported the
cancellation, then assigned `v.warnings = drawioWarnings(issues)` (or the
mindmap equivalent). `Host.Report` still received the issue, but the graph
warning panel lost the user-visible cancellation warning.

Minimal repair: retain the cancellation text in a local value only after the
post-report ownership check succeeds, then append it after the terminal warning
list is assigned. Do not restore an unconditional deferred write against a
changed owner.

Resolution: frozen in
[drawio-worker-view-warning-followup-result.md](drawio-worker-view-warning-followup-result.md).
The final code stores `gestureWarning` only after the immediate owner check and
appends it after parse, preview-limit, or parse-error warnings are established.
The ordinary stale-source active-drag regression verifies a successful request,
gesture cancellation, and preservation of the visible warning. The reentry
early-return cases remain covered.

Both findings are corrected. Neither is a remaining acceptance blocker.

## Final scoped identity

The final effective source consists of the six unchanged contract/graphview
files from the original result plus the two files superseded by the final
warning follow-up:

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/ui/editor/contract.go` | 4,107 | `4C13D2658B252E62D5A58991CF7D37174309B6B98E53FBCD3B5028211EE6DE2D` |
| `native/ui/editor/export_request.go` | 938 | `51AB9A485EDF9626A696C7BC5EE196E38E8A2B828EB7873C78664AC40FECAE41` |
| `native/modules/graphview/adapter.go` | 8,829 | `29AC4724019C7D72C2595C711BCD6B924F3DEAF25C96D5F5C81F97967847D6E8` |
| `native/modules/graphview/draft.go` | 12,984 | `B482887442D57103CB0B610D4469882CCDE32F2D665CBF96AC444154E19C1DA2` |
| `native/modules/graphview/export_request.go` | 4,159 | `910FFAB7AC8E7EFC53D056CA520FBBCD5DF0FF550233BC6B2B720B3F4582A7F2` |
| `native/modules/graphview/preview_limit.go` | 1,276 | `D2AA08246538C4E92F09A729F6661A69890F971401620D11CBF0F1A96CCF65F0` |
| `native/modules/graphview/sync.go` | 7,016 | `F09DB5207C96724CC2DEB936D989C733543AD538E7BB5FC0E9C6549CBE1F1A10` |
| `native/modules/graphview/export_request_test.go` | 10,671 | `D992C5B856358333A3BEA6C2691564B68C601D2A37B3BAA11B91FCA60E3E95B7` |

All eight entries match the files on disk. The original and reentry result
hashes and their evidence files also remain preserved as recorded. The final
warning-followup test/vet evidence matches its recorded hashes and exit 0; no
passing suite was repeated for this review.

## Reviewed contract and graphview behavior

- The editor contract remains neutral: `ExportPage`, `ExportToken`, and
  `ViewExportRequest` are value types; `PrepareExportRequest` and
  `ExportRequestToken` are optional interfaces. `ExportEngine.RequiresView`
  defaults to false, and `DrawioAdapter` still has no `ExportEngine`, so the
  fourth engine is not advertised.
- `ExportRequestToken()` only reads the current epoch. It does not synchronize,
  flush, parse, mutate source, touch the filesystem, or invoke callbacks.
- `PrepareExportRequest` is draw.io-only and refuses an incompatible kind, nil
  buffer, open edit group, source over the preview limit, pending graph draft,
  parse failure, unreadable document, invalid/missing page, or overlong page ID.
  It does not call `Flush`, edit or publish source, discard a draft, or fall back
  to page zero.
- It captures buffer identity/version, release epoch, page-selection epoch, and
  selected page identity before optional synchronization, then refuses the
  request if any identity changed. A callback cannot retarget the accepted page
  through A→B→A, replace/release the buffer, or resurrect stale cache state.
- The returned page is an exact copied current index/ID, including an empty ID
  when that is the actual page identity. Index is bounded to `0..249999` and the
  ID to 512 UTF-8 bytes without truncation.
- The export epoch advances on actual parsed-cache invalidation when a parsed
  document existed, every actual selected-page transition (including draft
  restoration and A→B→A), and every Release. Released views can be reused.
  Cache hits, repeated preparation, unchanged `Draw`, and unchanged
  `Warnings()` do not advance it.
- Pending field and tag drafts are checked without flushing. Existing draft
  recovery, conflict, history, page association, and release behavior remains
  outside the new export request contract and was not weakened by these files.

## Review conclusion

The prepared view contract and graphview synchronization layer are acceptable
in their final frozen state. The reentrant callback crash/overwrite concern and
the canceled-gesture warning regression are both corrected, and no further view
contract blocker remains.

This acceptance is scoped to the neutral contract and graphview preparation. It
does not establish fourth-engine advertisement, source-mode policy, managed
worker execution, app integration, packaged delivery, physical input behavior,
or resource bounds; those remain explicitly gated Stage 2 work.
