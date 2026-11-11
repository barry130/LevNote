# .15 UTF-8 streaming-save documentation

ACTIVE after utf8-save-release15-root-acceptance.md. Owner DeepSeek
read_native_handoffs_resource_workload_next_task_md. No children. Only root
README/ROADMAP/CHANGELOG and native README/ARCHITECTURE/VALIDATION plus
utf8-save-release15-docs-result.md. No code/tests/builds/launches/evidence edits.

Update current runnable entry to accepted .15 actual path/facts. Lead user docs
with the practical result: UTF-8 saves avoid assembling another whole text plus
encoded-output copy; optional BOM and exact original bytes/line endings remain,
safe save/conflict behavior unchanged. Preserve .14 keyboard and earlier feature
history. Keep technical save/verification contracts in ARCHITECTURE and actual
test/benchmark/package evidence in VALIDATION. Avoid accumulating another full
audit table in concise root README; links suffice for detailed evidence.

Report allocation comparison honestly: output-only fixed10x1/8MiB benchmark,
both preset snapshots to io.Discard with SHA256. 2,105,398 to33,131B/op and
16,785,478 to33,139B/op; allocation count actually 4 to8 per operation. Don't
say fewer allocations without specifying bytes. No whole-Save latency, peak
process memory, working-set/RSS, UI/CPU or physical input proof. Snapshot
capture/baseline retention are excluded; UTF16/GBK/GB18030 keep old path.

Use actual accepted .15 package/manifest totals, source/independent acceptance,
final affected document/session/app suite and vet, current delivery validator
and one hidden startup. It did not exercise dirty Save or workers. Preserve
.14 and .13 records as historical; inherited .11 protocol execution remains
historical. Preserve no-op saves, write errors, exact digest gating and existing
post-publication SaveAs metadata timing, with no rollback guarantee.

Check local Markdown target paths and actual current executable+manifest once,
publish result and freeze. No test rerun for docs-only changes. Small wording
fixes may be made without rerunning unchanged checks.
