# Reopen closed saved files: scope decision

2026-11-10. Root accepts the source-grounded direction in
[Astra's plan](reopen-closed-file-next-plan.md). Implementation starts only after
.26 package/evidence closure and root acceptance; no source mutation during its
build/lifecycle verification. This is a daily file-workflow increment, retaining
at most16 path/ID entries in memory instead of live closed documents.

First slice is deliberately path-only. Reopening reads current disk bytes using
the current normal route, plugin settings, permissions and encoding detection.
An unknown encoding retains explicit byte-inspection/reopen-with-encoding UI;
the previous document's code page is not silently applied to replaced bytes.
Do not claim tab/draft/session restoration. A saved discard can reopen only its
disk path; unsaved/recovered drafts without a saved path are excluded. Normal
unsaved close needs no new warning. Only an unusual overlimit/nonlocal saved
path omission needs a truthful notice when relevant.

Add the smallest private completion receipt to actual successful UI selection/
adoption points. Keep Open/OpenMany/OpenAt signatures, queue ordering, normal
diagnostics and binary lane behavior. A queue-finished bool is not a success
receipt. Reopen reservation/terminal outcome and exact entry ID prevent stale
completion from consuming a later close. Do not introduce a generic job or
command framework. Failure retains that entry; a compact native menu offers
other entries without silently dropping missing files.

Native menu and Ctrl+Shift+T share a guarded command. No action for empty/busy,
modal/restoring/shutdown, and no utility preedit loss. Check the request again at
UI adoption. A stale menu item ID must fail, never select the current top.
Labels/tooltips are UTF8-bounded without changing the stored full path. No
per-frame filesystem reads, retained document/undo/bitmap or persistent state.

The initial receipt and feature should be reviewed as one small coherent change;
tests must establish actual adoption, failures, async identities and unchanged
source bytes rather than only exercising a new boolean. Current app/decoder/
session tests are inherited except the affected app suite after freeze.

Independent GLM review is complete in reopen-closed-file-plan-review.md. Root
read that result and the actual CloseTab/openRequests/completeBinaryExplicit
seams. No additional prerequisite is needed. Revalidate the captured close
target and slot after dialog reentry; do not record a close merely because a
void removal callback returned. Use terminal receipts at real successful
selection/adoption, including the binary duplicate route. Menu entry identity
is a stable key, and each reclose gets a new numeric ID. The implementation task
already contains these requirements and remains inactive until .26 closure.
