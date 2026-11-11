# Fixed first-party engine packages — next design task

2026-11-09. GLM read_native_handoffs_editable_engine_next_design_task_md owns
only native/handoffs/fixed-engine-package-next-design.md for this next task.
Read this task once; work from actual files; no code/scripts/tests/execution,
no child agents. Astra continues .8 packaging; do not touch its inputs.

Root read the corrected rename proposal. The optional worker Apply gesture
alongside synchronous Enter would duplicate user behavior and host work, so
that pilot is NOT activated. Design the first useful separately managed package
boundary for the three existing repository-authored engines instead. Keep this
short (about150lines), decide one path and concrete API rather than an essay.

Inspect core/engineruntime/{runner,bundle,bundle_windows,process_windows}.go,
enginecatalog/catalog.go, app/export_worker.go, preview_worker.go and
engine_lane.go, current plugin install/state code and packaging manifests.

Answer these implementation questions concretely:
- What minimal package format and destination layout lets an existing fixed
  engine be installed/switched/restored without rebuilding or replacing the
  host? Do not accept scripts/commands/unknown capabilities or new engine IDs.
- Which identity facts are trusted, and why? Current sha+manifest only detects
  consistency; an arbitrary file named first-party is not provenance. For the
  first increment prefer importing a verified existing shipped engine from a
  user-selected complete release directory, with an explicit distinction
  between local-file trust and cryptographically verified publisher. Do not
  invent a publisher signing system or imply the source can prove provenance.
  Product install text should let the user make the real code-execution choice;
  this is not a new root/user permission request for our own authorized build.
- What bounded admission/atomic state publication reuses current file locking
  and private-state code? Reject links/reparse, escapes, collisions, platform/
  protocol mismatch and oversized files. Keep32KiB streaming hash/copy; do not
  load entire executable. Exactly which limits and APIs need to be shared?
- How does runner resolve installed selection vs bundled fallback? On selected
  missing/corrupt package refuse visibly; do not silently fall back to a different
  version. Explicit reset-to-bundled is a user action. Manifest version isn't
  assumed SemVer. Keep engine identity/version/digest distinct from schema3
  document suffix configuration.
- Switching while work runs must drain/cancel-and-join the existing single lane
  and invalidate owner generations before activation; persistence failure must
  keep old live selection/drafts. Needn't change plain document/plugin routing.
  Where does the lifecycle contract live without blocking UI on a worker?
- Include one basic UI workflow (fixed engine list, installed/bundled selection,
  local import, reset; no marketplace/network) and meaningful bounded tests plus
  one exact real package smoke. Packages affect exports/readonly-scene only;
  editable parsing/outline still in host and resource savings unproven.

Report a small staged file/owner plan and concrete risks grounded in source.
No implementation until root explicitly activates it after .8 acceptance.
