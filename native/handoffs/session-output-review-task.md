# Session JSON output independent review

ACTIVE2026-11-10 after accepted .15 docs and frozen implementation. Owner DeepSeek
read_native_handoffs_resource_workload_next_task_md. No children. Read-only source
and evidence review; do not implement, test, benchmark, build, launch or edit
product docs/evidence. Write only session-output-independent-review.md (interim
if needed) and session-output-independent-reconciliation.md at final freeze.

Read session-output-plan.md, implementation-task.md and actual document state,
session output helper/Save and tests. Review only material correctness and
scope; no generic filesystem redesign or new framework. Confirm existing
[]byte private-state callers keep atomic lifecycle/permissions/errors, nil
callback rejects without side effects, and session full preflight precedes
filesystem effects. Output must equal json.Marshal exactly with only Encoder's
mandatory final LF removed. Multi-write/empty/short writes, sticky errors,
wrong lengths, extra/missing terminal byte, and sink failure must reject before
publication, removing actual partial temp and preserving previous recovery.

Check standard library Encoder still uses a full internal buffer, no misleading
token-streaming/fixed memory claim. The change removes Marshal's returned copy,
not existing draft strings, metadata/snapshot descriptors or encoder's pool.
Check meaningful byte oracles across valid float32/Unicode/control/defaults,
omitempty/null/field order, bounds and Save/Load, failures before write and exact
atomic old-file preservation. Scope must leave app retry/discard logic and
recovery format unchanged.

Review benchmark equivalent work and fixed b.N/10x: identical validated
snapshots, old Marshal plus same sink write, output oracle/fixture hashes outside
timing, no fake loop/forced-GC performance comparison. Pooled encodeState may
affect B/op samples, so report measured output boundary with raw values rather
than process/RSS or guarantees. Final affected document/session/app suite+vet
must be after corrections, with frozen actual source/test/raw evidence hashes.
No test rerun by reviewer. Notify root of concrete defects early. Final report
must distinguish actual reconciled evidence from interim observations.

Root read current freeze and noted review targets: output adapter does not
remember sink/length errors (Finish can forget an earlier failed extra Write),
although the current Encoder caller immediately returns the Write error.
Determine actual production impact versus defensive helper contract. The
atomic short/sentinel tests currently manually write then return an error;
they do not route an injected failing destination through writeSnapshotJSON
to the real temp file. The new benchmark fixtures use old Marshal oracle but
do not independently compare actual new bytes outside timing. Original task
requires those integration/oracle cases. Report bounded corrections without
generic redesign. Result currently session-output-implementation-result.md
SHA2566416BB6874F25801EE4E703AB0A28D67BF4E81C3B8D3048F59A8033784AD615E.
