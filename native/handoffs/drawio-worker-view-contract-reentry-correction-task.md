# Prepared view export: stop after reentrant owner changes

ACTIVE 2026-11-10 for DeepSeek
`/root/read_native_handoffs_resource_workload_next_task_md`.
Preserve original frozen view-contract result/evidence. No children, app
integration/advertisement/build/product docs yet. Own graphview sync/prepare
and narrow tests only. Coordinate with independent GLM read-only review.

Root concrete source finding: Prepare's post-ensureSynced identity check is too
late to guard every claimed callback change. In ensureSynced, a stale cache plus
active drag/connect/reparent gesture calls v.report BEFORE setting lastSynced
and reading v.host.Buffer.String(). A Host.Report callback replacing Buffer with
nil therefore panics in String before Prepare can return ErrViewExportChanged.
Replacement/release also lets outer sync overwrite callback-owned cache state
before the later refusal. Current reentrant test only changes page index.

Make synchronization stop consuming or publishing stale owner state after a
report callback changes the relevant buffer/version/release/page identity.
Capture stable ownership before callback; check immediately after callback,
before any host.Buffer read or parsed cache write. Inspect other report exits
in this same synchronization path so they don't overwrite newly released or
reparsed state after returning from callback. Don't suppress meaningful errors,
recover a panic, or permanently mark released views dead. Ordinary no-callback
sync, graph editing, draft conflicts and reuse must stay intact. Prepare still
returns a copied current page only after unchanged-owner guards; no second Flush.

Add narrow tests with stale source plus canceled gesture Report that nils/
replaces buffer, calls Release, changes source, and pageA->B->A. Check no panic,
request refused, no stale resurrection/publication, and pending draft remains
when callback creates one. Prefer targeted meaningful subcases, not broad matrix.
Existing ordinary tests remain. One final graphview test/vet via wrapper (editor
contract unchanged, no need repeat editor suite). Preserve actual commands/logs/
exits/failures. Publish
`drawio-worker-view-contract-reentry-correction-result.md` with final hashes and
scope, freeze for root/independent reconciliation. No Stage2 until activated.
