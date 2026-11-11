# Deliver native readonly scene preview after consumer acceptance

2026-11-09. QUEUED planning only. No build/package/real-worker execution is yet
authorized by this stage. Current accepted `.6` remains the runnable version.
Root must first read consumer source/tests/vet and packaging source/asset gate.
Astra owns app tests and later delivery; GLM read_native_preview_ipc_task owns
packaging source prep only (transferred from interrupted DeepSeek).

## One actual registered readonly consumer

After the consumer gate, add a single opt-in app preview bundle test beside the
existing Markdown/Mindmap export bundle tests. Default ordinary tests skip it.
Use the registered Mindmap adapter, actual `document.Open` of a bounded fixture
made readonly by os.Chmod(0400), and restore mode in cleanup. Verify Doc.ReadOnly
was detected; do not merely set the flag or skip failed detection on Windows.
The test runs in a disposable complete copy of the new bundle and uses
engineruntime.NewBundled. No fake scene/provider or parser fallback.

Use a small BOM/CRLF/Unicode source with visible nodes and a folded descendant,
literal color, an image reference and preserved unknown metadata. Assert an
independent literal scene/geometry/paints/labels and exact whole-source warnings,
including the truthful no-image warning, from the actual job slot before UI
delivery. Do not use only production Encode/Decode/Parse as its expected oracle.
Then deliver through actual App.View, navigate a visible ID and refuse the known
hidden target, switch theme without requesting another worker, use source action,
and check the document remains unchanged. Check disk bytes, Buffer version and
history/selection, encoding/BOM, readonly/dirty/baseline and dropped job result.
Runtime Close must join without draining UI callbacks; capture actual engine
version/hash from the verified result and compare with its manifest.

Capture a small focused pair of760x480 light/dark native tester images of this
real decoded scene plus one source view if useful. Keep the canvas usable with
warnings/outline visible and the source entry reachable. These are headless
native tester images, not physical window/IME or OS accessibility proof.
One preview worker request is sufficient here. No resource process suite.

## Fresh complete bundle

After root reads the opt-in test source and packaging sources, choose the next
absent candidate `native-worker.20261109.7` / matching native/bin directory.
Never overwrite old bundle/evidence; use a later suffix if occupied. Build host
and three fixed workers once through the shared package script; retain actual
commands, streams, durations and exit codes. Asset/checksum checking and the
unchanged attribution source copies stay part of the package.

Compile one app tester into a disposable complete copy. Run the three explicit
opt-in entries once: existing Markdown HTML export and Mindmap SVG export (two
requests each: success+exclusive refusal), plus the new readonly preview (one
request). Clear unrelated inherited LevNote test switches. Retain explicit
RUN/PASS and result records for all three, expected five real worker requests.
No new generic test runner or executable registry. Keep tests/helpers/logs
outside the user bundle.

Independently validate four Windows amd64 GUI PEs, three engine manifests with
exact catalog identity/path/platform/caps, package inventory and local links.
The ordinary complete asset tree now expects23files (previous21 plus preview
exe+manifest), but derive/verify actual contents rather than accepting count alone.
Compare scoped source/toolchain/protected prior and fresh bundle/copy inventories
before/after actual execution. Preserve original failed attempts; a correction
reruns only affected checks. Do not overwrite original evidence with a summary.

## Documentation after root accepts actual package

Update source README/ROADMAP/CHANGELOG and native README/VALIDATION/ARCHITECTURE/
formats/API with current launch and precise scope. Update readonly inspector
description, which previously promised note/link fields in readonly Mindmap.
Writable copy via Save As restores the editable native inspector. Describe
one shared execution lane, preview coalescing/export priority, missing-worker
refusal/source access, metadata/wire limits and exact-source safeguards.
Keep host outline parsing/editable Mindmap/drawio and current arbitrary-plugin
limitations explicit. Write concise delivery handoff and separate root acceptance
record. Do not rewrite shipped or accepted package evidence. No new resource,
copyright immunity, full migration or physical IME claims.
