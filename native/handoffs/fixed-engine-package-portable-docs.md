# Portable engine-package docs — source-only result

2026-11-09. Updated only [../portable/README.md](../portable/README.md)
for the current fixed-engine package UI and runtime behavior.

Changed content:

- Replaced the stale single “插件 → 安装” instruction with the current
  “格式配置” and “引擎包” pages, while keeping format configurations
  described as non-executing schema1/2/3 JSON data.
- Documented the three supported fixed engine IDs, complete local release
  directory import, no-activation-on-import, explicit digest-based
  activation, explicit bundled reset, corruption handling, cancellation,
  bounded startup metadata, and explicit installed-package scanning.
- Documented retention plus the 12-package-per-engine/36-total limits, the
  private `%APPDATA%\LevNoteNative\engine-packages` store, and its
  separation from a moved program directory without overpromising fully
  asynchronous I/O.
- Clarified that checksums/digests prove local consistency only, not
  publisher authentication or executable sandboxing, and that activation
  means the app will run the chosen local executable.
- Kept the host-native scope statement for editable models/layout, outline,
  draw.io, and the separate readonly mindmap scene worker, including
  draft/undo preservation and readonly scene invalidation on switch.

`{{VERSION}}` and the existing release-manifest/LICENSE/notices/licenses/
attribution links are preserved. This is a source-only Markdown change;
tests/builds/processes/Go or script checks were not run because none were
needed for this scope.

## Review corrections

2026-11-09, after `fixed-engine-package-portable-docs-review.md`:

- Removed the incorrect `可写` qualifier from `levnote.mindmap-svg`; it also
  serves readonly documents.
- Replaced the manual readonly scene refresh wording with automatic refresh on
  the next Draw/Warnings pass, retaining old content and showing failure when
  refresh fails.
- Clarified that cancel is cleanup, not rollback after an atomic import or
  selection commit; committed state may remain and the panel status is
  authoritative. Import itself never activates.
- Clarified that a failed cleanup blocks engine transitions for the current
  run, including all-three reset, while import/list checks remain independent.
- Corrected the asynchronous-I/O boundary: startup metadata is bounded and
  synchronous, and final export file writing is synchronous and uncancellable
  after it starts. No disk-latency bound is claimed.

The portable README is now frozen for source-reviewed `.9` packaging. No
tests, builds, processes, Go files, or scripts were run or changed.
