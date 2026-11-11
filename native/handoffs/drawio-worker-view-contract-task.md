# Fourth worker view contract: parallel isolated implementation

ACTIVE 2026-11-10 for DeepSeek
`/root/read_native_handoffs_resource_workload_next_task_md`.
Root delegates all implementation/tests. No children; compact before200K.
All Go uses native/scripts/go.ps1. .18 binary/docs accepted and immutable.

Root allows this disjoint part of future stage2 to proceed before stage1
acceptance. Stage1 owner edits core/runtime/IPC/parser/new worker/script. You own
ONLY ui/editor/export_request.go, tiny contract.go ExportEngine flag addition,
modules/graphview cache/page/epoch/prepare methods and narrow tests. DO NOT set
DrawioAdapter.ExportEngine yet; app behavior/toolbar/package UI remains as .18.
No app/core/formats/worker/script/product docs, build/launch/capture changes.

Read stable graphview lifecycle plus drawio-worker-stage2-task.md and preflight.
Implement neutral value types ExportPage{Index int; ID string}, ExportToken uint64,
ViewExportRequest{Page ExportPage; Token ExportToken}; optional interfaces with
PrepareExportRequest() (ViewExportRequest,error) and side-effect-free
ExportRequestToken() ExportToken. ExportEngine gains RequiresView bool default
false. No runtime/IPC imports in editor; no new global routing framework.

Graphview requests are only for kindDrawio. Prepare synchronizes stale cache
after checking buffer/group/32MiB size, returns exact copied current index+ID
and current epoch; index0..249999 and ID<=512UTF8bytes, no truncation/fallback.
Explicit error for nil buffer/parse failure/missing page/incompatible kind.
Do not Flush a second time or publish/edit source; host controls Flush. Pending
draft must not be silently accepted/discarded. Sync can report/reenter: check
buffer/version/release epoch and selected-page identity around it so callbacks
cannot silently retarget the request. Snapshot belongs to later app stage.

Monotonic export epoch invalidated by actual parsed-cache invalidation/reparse,
Release and actual page changes including draft restoration. PageA->B->A
invalidates even same source version. Cache hit/unchanged Draw/Warnings doesn't
bump. Inventory pageIndex assignments across sync/draft/navigation; centralize
as appropriate, preserve existing behavior. Released view may later be reused:
no permanent dead flag. Token getter does not synchronize, flush, mutate, touch
filesystem or emit callbacks. Source-mode eligibility is later app responsibility.

Tests should prove second page exact copied identity/token; A->B->A; Release
then reuse with new token; same-index source reparse; cache hit stable token;
stale no-draft cache synchronized; nil/parse/size/group/overlong ID refusal;
failed pending draft retains source/history/draft state; reentrant sync report
doesn't retarget accepted identity. No busywork test for trivial type aliases.
Preserve existing graphview behavior/caller tests. One final graphview+editor
affected test/vet after corrections; no unrelated suite or screenshots since
no visible behavior is changed. Coordinate if stage1 parser test failure is
transient; don't edit its files or declare it a defect while half-implemented.

Publish drawio-worker-view-contract-result.md with exact changed hashes,
commands/raw logs/exits and neutral contract actually chosen. Freeze for root
and independent review. Stage2 app integration stays inactive until accepted
lower layers and this contract; no automatic engine advertisement.
