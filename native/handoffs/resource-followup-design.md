# Next native process-resource acceptance — bounded design

Status: planning only. This document proposes the next finite resource
checkpoint after the active Markdown links stage is accepted. No benchmark was
run, no product file was changed, and this plan does not alter current
ownership.

## Baseline and limitations

The accepted 12:56 evidence measured one blank launch and one launch with four
tiny representative documents (`source.go`, `notes.md`, `ideas.mindmap`,
`flow.drawio`). It correctly reported main-process working set, private bytes,
and CPU separately over ten one-second samples after ten seconds of settling.
It did not isolate per-format costs, active-versus-inactive tab costs, cache
release, repeated reopen costs, or long-idle behavior.

The next checkpoint should preserve the existing measurement discipline:

- Measure only the main native process.
- Keep working set and private bytes separate; they are different metrics.
- Report CPU seconds and percentages both per one core and as a share of the
  machine's logical processors, with the processor count recorded.
- Record executable size and SHA-256, fixture paths and byte sizes, launch
  order, active tab, mode, settlement duration, sample count, and graceful
  exit code.
- Keep every fixture deterministic and under the format's existing preview
  limits. This is a memory-lifecycle check, not a stress or parser-limit test.
- Do not compare with Tauri. The old stack has no recorded identical workload,
  matching process scope, or matching build configuration; an unmeasured claim
  would be misleading.

## Current lifecycle facts the design relies on

- CLI startup calls `OpenMany(os.Args[1:])`; `openRequests` preserves request
  order, reuses an already open path, and leaves the last requested document
  active.
- Tab switching and opening release the former tab's presentation view and
  outline. Closing a tab disposes that view, cancels its file watch, and removes
  it from the tab list.
- `markdownview.Release` drops parsed doc/rows/list caches, task descriptors,
  and the outline cache while preserving failed recoverable drafts.
- `graphview.Release` clears the parsed model/layout, render label cache,
  warnings, canvas handle/focus state, sync identity, and outline cache while
  preserving a recoverable draft.
- Session recovery is written transactionally through the bounded snapshot
  builder: 16 MiB maximum file size, 4 MiB per draft, aggregate draft budget,
  preflight encoded-size accounting, and the previous file retained on failure.
- Existing preview limits remain relevant context: Markdown 8 MiB, mindmap and
  draw.io each guarded by their engine defaults, and large blocks/list items use
  bounded placeholders rather than unbounded shaping.

These facts should be evidenced at process level; the design should not add
new release hooks merely for measurement.

## Proposed finite scenarios

Use one isolated app-data directory per launch, one fixed workspace directory
with immutable fixtures, and the release executable accepted at the end of the
links stage. Keep the existing window size. All scenarios close gracefully and
require exit code zero.

### 1. Blank idle baseline

Launch with no arguments and no session tabs.

Purpose: distinguish app/runtime baseline from document/view costs.

Record 10 one-second samples after 10 seconds, then a second 30-second idle
sample set (for example samples every 3 seconds). Report both. The first
establishes comparability with 12:56; the second gives a bounded view of
whether memory continues growing while the app does nothing.

### 2. One active format at a time

Run four separate launches, each with one document:

1. `plain.txt`
2. `markdown.md`
3. `ideas.mindmap`
4. `flow.drawio`

Use representative but controlled fixtures, not tiny samples:

- Plain text: about 512 KiB, mostly deterministic repeated lines plus a few
  Unicode/emoji and CRLF lines.
- Markdown: about 1 MiB with headings, paragraphs, lists/tasks, links, inline
  spans, and CRLF/Unicode content, all below the 8 MiB parser limit.
- Mindmap: a valid deterministic graph with roughly 500 visible nodes, under
  the engine's byte limit.
- Draw.io: a valid deterministic diagram with a bounded node/edge population,
  under the engine's byte limit.

Launch each in its normal adapter mode (source/preview as applicable), settle
for 10 seconds, sample for 20 seconds, and record the active tab and mode.
The goal is format-at-a-time cost, not a strict equal-byte comparison; record
actual fixture sizes and do not normalize results.

### 3. Ordered four-format batch

Launch all four fixtures in one `OpenMany` call in this order:

```text
plain.txt markdown.md ideas.mindmap flow.drawio
```

This intentionally differs from the earlier tiny-file scenario while remaining
finite. Confirm that tab order and the final active tab match the CLI order.
Measure after 10 seconds idle. This checks combined model/session/watch state
without manufacturing an unrealistic document set.

### 4. Presentation cache release on switching

Start with the four-format batch. For each active-format sequence:

1. Ensure the selected format tab is visible in its native presentation mode.
2. Wait for a bounded settle period.
3. Switch to another tab through the real tab UI.
4. Wait a short fixed interval (for example 3 seconds).
5. Sample memory after the switch.

Then switch back and confirm the format view rebuilds correctly. This should
show that inactive presentation state is released without requiring a benchmark
assertion that working set must fall: Windows may retain working set after
freeing private bytes. Report the direction and both metrics, and require only
that views rebuild and no navigation/edit state is corrupted.

### 5. Close-and-reopen cache lifecycle

Use the four-format batch and a scripted or test-only UI route only if the
product already exposes one; do not add a product hook for this design.

Close one clean tab through the real close path, wait a short settle interval,
and sample. Reopen the same file through the normal opening route, wait for the
same interval, and sample again. Repeat once for Markdown and once for graph.

The evidence should show that close invokes the existing view release path and
that reopen rebuilds without visible corruption. A private-byte reduction is
expected to be informative, but a working-set reduction is not a requirement
because the OS may keep trimmed pages resident.

### 6. Clean recovery lifecycle

Launch with one clean Markdown document, wait for automatic persistence, close
gracefully, then relaunch with the same isolated app-data directory and no CLI
arguments. Confirm the saved tab restores, and sample both launches.

This checks ordinary clean-session restore without large drafts. Do not create
an oversized recovery fixture in this process checkpoint; recovery size limits
are already covered by focused unit tests and would turn this into a different
scenario.

## Measurement mechanics

Prefer a reusable PowerShell sampling script with these constraints:

- Start from an explicit executable path and isolated `APPDATA`/`LOCALAPPDATA`.
- Do not mutate system or user profile state outside the acceptance artifact.
- Create the acceptance directory before launching.
- Wait up to 20 seconds for the exact main-window title.
- Use a fixed settle period before sampling.
- Record raw per-sample working set/private bytes, not only aggregates.
- Record raw CPU totals at the first and last sample.
- Record logical processor count once.
- Request normal main-window close; kill only if the graceful close deadline is
  exceeded, and treat that as a failed checkpoint rather than acceptable data.
- Emit a JSON or CSV artifact plus the exact command and fixture hashes.

If UI automation is needed for switching/closing, first inspect the existing
native tester or hidden startup harness. If no existing headless route can
drive real tabs, keep scenarios 1–3 and 6 and defer 4–5 rather than adding
product instrumentation.

## Acceptance questions

The finite pass should answer:

1. What is the current blank main-process baseline for this exact release?
2. What does one controlled active document cost for each native format?
3. Does ordered multi-document startup keep CLI order and activate the last tab?
4. Does switching to another tab release presentation caches without breaking
   return-to-tab behavior?
5. Does close/reopen use the existing release/rebuild path without permanent
   growth over two bounded cycles?
6. Does clean session restore return to a usable tab without unexpected idle
   CPU growth?

It should not claim a performance guarantee, peak stress coverage, OS
accessibility, IME candidate-window behavior, or superiority over Tauri.

## Suggested evidence format

For each scenario, record:

```text
scenario
executable size/sha256
arguments
active tab and mode
fixture names, byte sizes, hashes
settle seconds
sample count and interval
working set min/avg/peak/final MiB
private bytes min/avg/peak/final MiB
CPU seconds delta, one-core percent, system logical percent
logical processor count
exit code
notes and visible failures
```

Separate the conclusion into three honest sections:

1. Observed process numbers for this exact executable and workload.
2. Lifecycle behavior demonstrated through real routes.
3. What was not measured and cannot yet be claimed.
