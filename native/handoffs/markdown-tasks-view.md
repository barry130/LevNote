# Markdown native task checkbox — focused freeze

Implementation and owned checks are complete; product files are frozen for
root/independent review and DeepSeek's app integration. No full suite or release
build was run by Astra. New files: `modules/markdownview/tasks.go`,
`tasks_test.go`; modified: `view.go`, `rows.go`, `blocks.go`, `preview_limit.go`.

Final focused commands:

```powershell
$env:LEVNOTE_MARKDOWN_SCREENSHOT_DIR = 'F:\AIproject\LevNote\native\artifacts\acceptance\markdown-tasks-view'
native/scripts/go.ps1 -GoArguments @('test','./modules/markdownview','-count=1','-timeout=120s','-v')
# PASS, 1.846s; includes existing draft/outline/oversize/resource regressions
native/scripts/go.ps1 -GoArguments @('vet','./modules/markdownview')
# exit 0, no diagnostics
```

Native input evidence: actual pointer, six consecutive Space activations and
six rapid pointer activations preserve focus, exact bytes and reversed source
selection. One toggle is one undo action; uppercase X returns on undo. Empty,
ordered, formatted and clipped labels; configured/live readonly; disabled
state after refused foreign-group edit; separate selectable label; actual
Compose in another block then toggle, same-list Compose refusal, conflicting
Flush recovery; native duplicate far-row held click across prefix insertion;
1000-row far scroll and in-build reentrant Warnings; narrow light/dark readonly
geometry all pass. Command safety additionally covers target label/marker edits,
deletion, Restore, expired journal, replacement Buffer, partial/oversize and
closed/unclosed fence context. No label/index lookup is used to resolve a task.

The app-discovered stale Undo state is fixed: a clean independent draft follows
mapped external source edits and invalidates the parser snapshot; its field and
range remain live but old block index is detached. The actual Compose/toggle
test now checks native checked state through Undo and Redo while retaining the
same clean draft. Dirty/conflicting drafts keep existing recovery behavior.

The active List build captures immutable doc/rows/buffer/version. A toggle
validates against current bounded parse, mutates one marker character inside a
new selection-preserving group, then invalidates only the next preview. Numeric
identity for the exact own-toggle row survives reparse, keeping Space focus.
Partial documents expose source guidance and no active task controls. Own
task descriptors are cleared by Release/oversize cache dropping. The stale
message says `未切换任务状态`, since a preceding successful Flush can itself
have changed the source.

Single headless sample, fixture/first parse outside measured frames: 1000 tasks,
five unchanged native frames **1.0012ms / 74,008B / 972 allocations**; 17 rows
built, `FramePending=false`. This is tester work allocation, not OS process
memory or a repeated process benchmark.

Four images are in `artifacts/acceptance/markdown-tasks-view/`:
`markdown-tasks-narrow-dark-{false,true}-readonly-{false,true}.png`.
265x360 frame with330px bounded content and30px status region; every checkbox
fits content. Light editable and dark readonly images inspected. Root accepted
geometry/contrast, then requested the now-applied truthful `只读预览` hint;
the final images above were regenerated after that change.

Residual scope: existing simple list grammar only, no nested/continuation list
expansion; names cap raw-label metadata at128 UTF8 bytes+rune-safe ellipsis;
large body placeholder still points to full source. Native tester Compose is
not an actual OS IME candidate-window test. App save/BOM/source/split/reopen
and final release evidence are owned by DeepSeek.

---

Initial acknowledgement and agreed UI contract:

Root accepted15:12 oversize integration and activated markdown_tasks_stage_plan.
Astra has read AGENTS and markdown-tasks-contract. Oversize docs are complete
with independently verified artifact hash/size/time and163 checked local links.

Exclusive current scope: modules/markdownview/** controls, source-safe toggle,
focused actual native input/lifecycle tests, then capability/API docs. GLM owns
TaskMarker/parser/export; no graph/core/editor/plugin edits or child agents.

Design: capture buffer/version plus whole item and three-byte marker ranges;
Flush, revalidate, map both ranges, and require current bounded parser exact
task recognition before one-character grouped mutation. Preserve reversed
source selection inside the same transaction. Native checkbox binds a local
source-derived bool; pending drafts stay authoritative and failure recoverable.
Keep checkbox rapid/double pointer events from beginning the parent whole-list
draft, cover Space and readonly, and preserve existing row virtualization.

Implementation is active; GLM's additive structs are present and its focused
format checkpoint has passed. There is no metadata dependency blocker.

Concrete UI contract (sent to DeepSeek): styled `ui.Checkbox` with empty
visible label, a separate selectable formatted RichText label, and native
RoleCheckBox/Checked/Disabled semantics. Access/tooltip names are
`未完成任务：<label>` / `已完成任务：<label>`; empty label is `（空任务）`.
The raw Markdown label is bounded to128 UTF-8 bytes at a rune boundary plus
`…`; tooltip uses the same bounded name. Only the checkbox box activates;
the separate text remains selectable/double-click editable. The checkbox
keeps native focus and Space behavior while shielding the parent double-click
handler. Normal command refusal reports through Host.Report and keeps source
and draft. Exact own-toggle row identity survives refresh, without any fallback
by duplicate label or list index. Focused completion evidence is above.
