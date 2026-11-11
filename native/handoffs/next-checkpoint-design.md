# Next finite checkpoint — external format-engine seam

Status: design-only recommendation, written while the mindmap marker view/app
owners finish their current stage. No product source, test, build, benchmark, or
network operation was performed for this review. This handoff does not review the
pending marker implementation.

## Current position

The native app already has a truthful, bounded data-plugin slice, but it is not
yet “supported formats as plugins” in the architecture sense.

- `core/plugins` orders and enables manifests, and installed JSON schema1/2
  plugins can claim extensions, provide single-line highlight metadata, and
  provide literal source-outline rules.
- `modules.Builtins()` is still the hardcoded composition root for the rich
  adapters: Markdown view and the two graph adapters.
- `app.configureTab` attaches a rich adapter only when the matched manifest is
  a builtin. An installed schema2 plugin can receive an outline factory, but it
  cannot provide a rich view, export, image loader, or lifecycle adapter.
- Go's `plugin` package is not a Windows-compatible route. A process boundary is
  therefore the practical option for independently distributed native engines.

The current resource evidence is also narrower than the original goal:

- The accepted matrix measured hidden, no-input main-process startup and one
  restore case. It does not measure visible rendering or interactive editing.
- There is still no matched Tauri/native workload, so the project must not claim
  that migration reduced memory or CPU.
- MyGo tester Compose tests prove the app's preedit path across Save/mode/tab
  barriers, but not real Windows IME candidate-window placement, physical
  language switching, or full accessibility.

Mindmap functionality remains intentionally partial: topic/note/link editing is
implemented, icons are the active marker slice, tags are displayed but not
editable, node images are parsed/preserved but only shown as a bounded “图”
placeholder, and relationships/summarize are not complete. These are real
product gaps, but another metadata editor would not address the larger plugin
architecture gap.

Evidence inspected: `native/core/plugins/registry.go`,
`native/app/plugins.go`, `native/modules/modules.go`, `native/README.md`,
`native/formats/API.md`, `native/VALIDATION.md`, root `ROADMAP.md`,
`native/handoffs/resource-followup-artifact-review.md`,
`native/handoffs/integration.md`, and the current mindmap
parser/serializer/graphview sources. No pending marker implementation file was
reviewed as part of this planning task.

## Candidate comparison

Two realistic worker candidates were considered.

1. **Installed schema2 source-outline worker.**
   - It is easy to bound: the current scanner has explicit byte, line, item, and
     label limits, and its output is already a small bounded outline.
   - But the in-process scanner is already cheap and self-contained. Moving it
     to a second process would add idle or scan-time memory for a feature that
     is already small, and would demonstrate avoidable duplicate-process
     overhead rather than meaningful extensibility.
   - It also does not move the architecture closer to the original goal because
     the installed plugin remains declarative data only; the worker would be an
     internal implementation detail, not a real format-engine seam.

2. **Repository-owned Markdown export worker.**
   - It uses a real supported rich format, not a tiny literal scanner.
   - The host keeps the existing native Markdown view for editing and preview,
     so the worker does not have to replace or reimplement the view.
   - Export is a user-visible, already supported operation, and the engine can
     be launched one-shot for a single export request rather than kept resident
     for interactive work.
   - It gives the first honest answer to the actual architecture question: how
     much bounded-IPC and process overhead a real native format engine costs
     under the low-resource goal.

The Markdown export worker is the better next checkpoint. It has more real
product value, a natural on-demand lifecycle, and a bounded data contract that is
still small enough to finish in one slice.

## Recommended next checkpoint

**Build one repository-owned, one-shot Markdown export worker and prove the
host/engine protocol, provenance, and lifecycle.** This is deliberately not a
third-party plugin SDK and not a rich-view migration. It is the smallest real
process seam that can later carry richer format engines.

The first engine should wrap the existing bounded `formats/markdown` parser and
`ExportHTMLChecked` path. The app continues to own matching, files, UI, settings,
and publishing. The sidecar owns only pure data interpretation and export
rendering.

### Protocol and trust boundary

Use a local child process with stdin/stdout framing and captured/bounded stderr;
do not open a network socket. Suggested v0 messages:

1. `Initialize`: protocol version, engine ID/version/capabilities/limits.
2. `ExportBegin`: source size, parse/export limits, and format options.
3. One or more bounded source chunks. The host sends bytes only; it never sends
   a file path or asks the engine to read the workspace.
4. `ExportEnd` and a bounded result: HTML, warnings, truncation flags, and errors.

Every message has an explicit maximum size. Cancellation kills the child through
the host context and is surfaced as an export error rather than as empty output.

For v0 provenance, ship a build-generated engine manifest beside the engine
executable containing the exact engine ID, protocol version, capabilities, and
SHA-256. The host refuses to launch on hash or protocol mismatch and shows the
reason in the plugin panel. This proves integrity and explicit provenance, not
third-party trust; code signing, an update channel, and an arbitrary user-code
installation model remain out of scope.

A process boundary does **not** remove the engine's operating-system privileges.
Even though v0 never sends a path, a compromised worker still runs with the
user's normal file, process, and network privileges. This design therefore makes
no sandboxing promise. It only narrows the protocol surface and keeps file
ownership in the host. A later security slice would have to add explicit OS-level
restrictions; that is out of scope here.

### Lifecycle

- Start one child per export request for v0; do not keep an idle parser process
  resident while the user is only editing or previewing.
- Kill or wait for the child on export completion, cancellation, failure,
  timeout, and app shutdown; no orphan process may survive app close.
- If startup cost becomes measurable, a later slice may add a bounded reuse
  window, but that is not part of v0.
- Record launch, reuse, timeout, abnormal exit, and cleanup; expose failures
  distinctly rather than falling back silently to the in-process parser.
- Keep the host as the only file reader and writer. The worker never receives a
  path and never writes the export destination.

### Focused acceptance matrix

1. **Parity:** the worker returns the same HTML and warnings as the current
   in-process export path for BOM, CRLF, Unicode, tables, tasks, links,
   doc-blocks, `Partial`, and oversize rejection.
2. **Bounded IPC:** source chunks and result messages stay under explicit
   limits; no whole-source allocation is required beyond the already accepted
   8 MiB parse bound, and a failed export is visible.
3. **Provenance:** valid launch, wrong hash refusal, wrong protocol refusal,
   and missing manifest are each distinct and visible; no in-process fallback
   is silently presented as the external engine.
4. **Lifecycle:** each export completes and kills the child; cancellation,
   failure, timeout, and app close leave no owned process. The existing native
   Markdown view remains the editing and preview host.
5. **Resource evidence for the worker candidate:** build the exact host+worker
   candidate under test and measure the app and worker separately, with
   one-shot launch, parse/export, and shutdown. Use the already accepted
   sampling discipline (settle, ten observations, same PID checks) on that
   candidate rather than reusing the 16:45 values or the marker-release binary.
   Report overhead honestly; do not claim a Tauri comparison.
6. **UI truth:** the plugin panel distinguishes declarative data plugins from
   an external engine, and shows engine version, hash prefix, status, and
   errors. It does not claim rich view capability or sandboxing.

## Why this is the one next milestone

This checkpoint directly addresses the real architecture gap without inventing
an unbounded plugin framework. It proves four properties needed by every future
rich engine: a Windows-compatible process boundary, host-owned file access, a
pure data protocol, and an on-demand resource lifecycle. If the worker's
launch/export overhead is unacceptable, that must be learned before moving
Markdown or graph parsing across the boundary.

It also keeps later work honest: real OS IME/keyboard acceptance, tags editing,
image loading, and rich view/export engines remain separate finite stages. None
should be bundled into this protocol pilot.

## Explicit non-goals

- No arbitrary user-provided executable installation in this slice.
- No rich Markdown/mindmap/drawio adapter externalization yet.
- No network, update service, script runtime, dynamic library loader, or shell
   hook.
- No claim that process isolation alone makes untrusted code safe; v0 is a
  repository-owned engine with integrity metadata, not a sandbox.
- No claim that the worker has reduced filesystem, process, or network
  privileges; it runs with the user's normal OS privileges.
- No reuse of old hidden-startup resource numbers as evidence for the marker
  release, the current build, or this worker candidate.

