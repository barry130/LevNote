# Fourth draw.io worker: resolve plan seams before implementation

ACTIVE READ-ONLY 2026-11-10 for GLM plan owner. .18 delivery freezes all source
and tests. Write only `drawio-worker-plan-refinement.md`; original plan stays
immutable. No tests/builds/docs/launches or children. Keep refinement concise.

Root read the plan and actual files. Accept the two stages and fixed engine,
v1 frame/manifest compatibility, schema1->2 selection migration, and shared
runner/lane. Resolve these concrete issues for a ready implementation contract:

1. Use a required Drawio page object `{index,id}`. `id` is a required string,
   empty only when selected parsed page has no ID; compare exact ID at that
   index, never search/fallback to another page. Max ID512 UTF8 bytes; reject
   overly long selection explicitly, never truncate. Index0..249999 (parser
   element ceiling); valid-but-absent index or mismatched ID => `invalid-page`.
   Null/absent/duplicate/unknown fields/types invalid. Old3 reject any page.
   Index+snapshot already prevents reorder; explicit ID adds a useful identity
   assertion and matches our planned invalid-ID check without source rewriting.
2. Current `Source` has Size/Data, and many runner tests and app fake runners
   use `Export(ctx,id,Source)`. Prefer minimal additive `Source.Page` metadata
   and preserve this signature, rather than mechanically changing all callers
   for ExportRequest. If using pointer, copy the page value into owned request
   storage at admission/before asynchronous enqueue. Read actual lane/run
   callsites and describe ownership exactly; no live view crosses threads.
3. Source-mode does NOT imply `t.View==nil`: releasePresentation calls Release
   but retains the pointer! Disable/refuse by mode/capability and live state,
   with a visible instruction. Neither retained view nor valid page0 may bypass.
4. `Flush` with no draft can leave graph cache stale (ensureSynced is not
   guaranteed). Pure CurrentExportRequest must not return page from old cache.
   Choose explicit prepare/capture that may synchronize cache after size check,
   then revalidate all tab/buffer/version/view/capability identities in case
   sync reports/reenters; pure token validation remains side-effect-free.
5. Bump export epoch on real cache invalidation/reparse and Release even if
   page index remains identical; page-index-change-only helper cannot do this.
   Don't bump on a cache hit/unchanged Draw/Warnings. Failed Flush never discards
   drafts. Require pointer+epoch+version checks after initial capture and after
   final Flush/dialog. A->B->A must refuse stale export.
6. Plan claimed worker has no filesystem/network access: code may do no such
   I/O, but OS process is NOT sandboxed. Correct wording. Whole-source warning
   spans0..len are coarse ranges, never claimed exact positions. Decide selected
   vs document-wide parse issues without falsely assigning another page.
7. Correct file references: engineipc/json.go (not decode.go); lane is app
   preview_worker.go. Worker-only tests cannot prove host undo/dirty state;
   assign those to app integration. Check all hardcoded3 arrays/tests/fixtures
   but only change relevant behavior, preserve retention12 and cleanup poison.

Stage1 should minimize app changes (only additive compile compatibility if
needed); Drawio adapter still host-side until Stage2. Document concrete file
ownership and focused final packages. Stage2 shared contract can be separate
file ui/editor/export_request.go, reducing conflict. Publish refinement only;
do not implement until root explicitly activates stage1 after .18 acceptance.
