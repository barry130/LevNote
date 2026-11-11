# Fourth worker stage1: bound warning intermediates

ACTIVE 2026-11-10 for DeepSeek
`/root/read_native_handoffs_release10_ui_feasibility_task_md`.
Root reviewed frozen stage1. GLM independent review remains against that frozen
record; preserve original result/logs/hash manifest. No child delegation.

Concrete finding: `cmd/levnote-drawio-svg/main.go` allocates `relevant` with
capacity len(parsedIssues) before filtering or checking the 256 warning bound.
`wireWarnings` also allocates len(issues) capacity before rejecting an excessive
count. A source with many unselected-page warnings or selected/document warnings
therefore causes additional proportional allocations in the new IPC conversion
layer even though it can accept at most256 entries. Existing parser and emitter
issue allocation is out of this task's scope; don't claim a process memory cap.

Own only new worker main.go/main_test.go and new correction evidence/result.
Keep extra filtering/conversion storage bounded by the wire warning count,
reject excessive relevant warning count before a large intermediate allocation,
and stop converting after refusal. Avoid collecting all unselected warnings.
Preserve ordered document+selected parse issues followed by emitter issues,
exact selected SVG bytes, strict raw component checks, nil payload/empty warning
array on failures. No changes to accepted SVG emitter, protocol, app, runtime,
formats, scripts or docs. Check count before allocating in the standalone helper.

Use substantive narrow tests covering a large unselected warning population,
257 relevant issues, combined parse+emitter count, exact256 boundary and unchanged
warning scope/order. Reuse existing fixtures where possible; don't assert heap
metrics as whole-process memory. One final worker test/vet through wrapper, retain
all actual failures. No repeat full8package suite since other packages unchanged.
Publish `drawio-worker-stage1-warning-correction-result.md` with changed hashes,
actual commands/raw logs/exits and scope, then freeze. If GLM reports other
findings, root will assign them; do not edit other owners' files proactively.
