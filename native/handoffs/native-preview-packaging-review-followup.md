# Packaging source review corrections

2026-11-09. Root read the three saved files and source checkpoint. GLM
read_native_preview_ipc_task retains the exact three-file source ownership.
No build, package, worker or product launch is active yet.

Correct the following and publish a concise updated checkpoint before execution:

1. The executable size guard in package-markdown-worker.ps1 must remain the
   runtime's fixed 64 MiB limit. PayloadMaxBytes is an IPC result budget, not an
   executable budget. Retain the per-engine output metadata (preview 32 MiB,
   exports 64 MiB), but restore the separate executable cap and correct the
   checkpoint's current claim that using PayloadMaxBytes for both is intended.
2. Portable README must distinguish the SVG export's image "图" placeholder
   from the readonly scene preview, which loads/displays neither images nor that
   badge; references remain in source. Neither reads image files/network.
3. State the actual boundary clearly: native UI renders in the host; editable
   Mindmap parsing/layout and source-outline parsing remain there; the readonly
   Mindmap scene parser/layout runs in its fixed worker. Missing preview must
   not silently fall back to the host parser/layout. Do not say all previews
   are in host, or that the native drawing is outside the host.
4. Add concise readonly scene inspector/source-only behavior and the writable
   Save As transition back to the editable inspector. The tag-editing section
   applies to editable Mindmap, not the new readonly scene.
5. Restore independent coverage of required attribution/license files in the
   asset fixture. The new copies.Count > 0 oracle can pass after a required
   license has been silently removed. Use an independent required relative-path
   list for all 13 current copies (or retain exact 13 plus path coverage), then
   actual byte/digest comparison. Total manifest counts may be derived from
   actual inventory; do not weaken license coverage for that purpose.

After these source corrections root will read them and activate only the
existing asset/path fixture. That run must retain its real commands, streams,
result and before/after owned input hashes in a fresh evidence directory. It
must not invoke Go, build executables, launch workers or alter old bundles.
