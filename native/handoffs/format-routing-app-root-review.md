# App routing source review — concrete lifecycle checks

2026-11-09. Root read actual app/app.go, plugins.go, view.go, registry.go,
modules/modules.go and the in-progress format_routing_test.go.

The installed manifest identity is separate from the native adapter identity;
the schema3 semantic outline and descriptors remain owned by the shipped
adapter. `pluginDocumentTemplate` already checks fresh registry Match, so
calling it again after Flush DOES recheck first-suffix shadowing. No duplicate
matching helper is needed. Post-save reload sentinel handling is appropriately
narrow; the original settings-persistence refusal remains separate.

Two concrete points belong in the active app slice:

1. `NewFromPlugin`'s receipt checks tab/document/buffer/generation but currently
   not the originating `View`. Its new replacement-during-Flush test should
   expose this: a successful callback replacing the presentation would let
   addDocumentChecked release the replacement and append. Capture and compare
   the original presentation before Release, without rejecting ordinary
   successful edits made by its own Flush. Verify existing replacement/recovery
   tests meaningfully retain the new owner/draft on refusal.
2. `Save` currently flushes only before chooseSavePath. It writes and then
   disposePresentation on a new path without flushing after the modal returns.
   This is the same nested-input risk already fixed for installPlugin. Add a
   narrow post-dialog Flush barrier before SaveAs, plus ownership checks against
   a replaced/closed/reconfigured tab/document/view as applicable. On failure,
   do not create the selected file, change route/path or release the pending
   draft. A successful new input committed during that barrier must reach disk
   and stay undoable. Use focused alias SaveAs regressions through the existing
   saveDialog seam; include one actual format draft if existing helpers allow.
   No new generalized transaction layer or unrelated engine refactor needed.

This is source review, not a test result. Astra continues implementation and
scoped checks autonomously and preserves any failed attempt honestly. Accepted
.7 bundle/evidence stay unchanged. Root performs no product or test edits.
