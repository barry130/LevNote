# Root review of proposed next editable boundary

2026-11-09. Design is read, not accepted for implementation. Correct the same
design handoff only; no product/test/script changes or execution. Astra still
owns current schema3 delivery. Keep this finite and source-grounded.

The proposal has several contradictions that must be resolved before code:

- Two 16KiB topic strings plus a512B ID cannot fit a4KiB metadata frame,
  particularly after JSON escaping. Do not silently lower topic limits. Prefer
  an operation-specific bounded payload envelope with lengths for source and
  command, or source-token/old-value digest plus a bounded new-text payload;
  explicitly state byte accounting, reject-before-allocation and compatibility
  with the unchanged three operations. Pick one concrete design.
- Returning a full serialized16MiB document and reparsing/computing the same
  expected semantic edit in the host duplicates the supposed offload. Inspect
  formats/mindmap parsing/JSON helpers and assess one exact topic-string token
  replacement with preimage verification. Can the worker resolve the actual
  node's topic token and return a bounded byte-range patch, preserving unrelated
  source bytes? Cite actual constraints; do not invent token offsets if absent.
  If extraction requires real work, say so and stage it explicitly. The host
  should enforce source identity/preimage/valid UTF8/range/size and own undo,
  not independently redo the parser mutation merely to validate the worker.
- Current Flush auto-commits topic changes synchronously, so making Enter fire
  a worker and every Save/navigation refuse is a UX change with cost. Recommend
  an explicit Apply gesture for this first optional capability, with visible
  pending/cancel state, and state what remains in-process. Cancel keeps the
  draft and cannot promise navigation proceeds while an uncommitted draft
  remains. No implicit discard. Do not claim this improves memory or latency.
- Pending job state may retain bounded identity references while active; source
  and worker callbacks must not read mutable UI state off-thread. On cancellation
  clear owner/result references and join via app-owned lane. Your 'no owner
  references while running' vs 'same owner on completion' currently conflicts.
- A publisher signature, or an OS sandbox, is a possible policy for third-party
  code and cannot be represented as a requirement imposed by the user. Separate
  trusted first-party separately installable packages from arbitrary third-party
  executable support. Pure local installation can be useful before a generalized
  third-party trust ecosystem exists. Describe actual risk and chosen boundary.
- Design currently ends mid-sentence at 'app host wiring belongs ... in'. Finish
  it, stay concise, and include real filenames, not only package names.

After correcting the feasibility, include a short recommendation comparing
the explicit rename operation with first-party fixed engine package installation
as the very next slice. User priorities are modular formats and lower application
resource use; a fourth executable that duplicates host work is not automatically
the best next product step. Root will choose after .8 is accepted.
