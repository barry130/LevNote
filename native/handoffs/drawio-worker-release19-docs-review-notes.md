# Root review notes for active .19 docs owner

2026-11-10. Complete these within the active docs task before final freeze.
No source/build/test/launch change. Root read the current draft and found:

1. ARCHITECTURE and VALIDATION incorrectly say PrepareExportRequest synchronizes
   pending fields / creates the undo transaction and captures Buffer/Editor/mode
   owners. Actual app startEngineExport captures owners and performs Flush FIRST;
   Prepare may synchronize parsed cache and reports only page+epoch. Prepare
   does NOT Flush, publish or discard pending draft. Pure token getter is for
   later identity checks. Fix this distinction in every newly touched summary.
2. Root README says intentional draft Flush occurs while dirty state/selection/
   history remain unchanged. State exact scope: original disk bytes/encoding/BOM
   are unchanged by export; a pending draft's Flush intentionally changes buffer,
   dirty state/history in one undo transaction. Clean no-draft export preserves
   buffer/selection/history/dirty state. Avoid unconditional conflicting claims.
3. ARCHITECTURE NewBundled fixed-directory list still contains only three paths;
   add engines/drawio-svg/. Protocol has required page object for drawio only,
   empty ID allowed only when the selected source page really has no ID; don't
   imply mandatory nonempty ID or old engines require page metadata.
4. The existing statement UI never flattens source should be scoped to snapshot
   transfer. Drawio Prepare can refresh parsed view cache and stringify source;
   no whole-app zero-copy or no-parsing claim for this route.

Keep the user-facing opening concise; exact caller/callee distinctions belong
in architecture/API/validation, not a long top README walkthrough. Preserve
actual .19 bytes/hashes and separate forthcoming schema3 drawio route. Final
result should mention these corrections and the single scoped link-check.
