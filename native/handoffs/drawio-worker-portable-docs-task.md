# Fourth-worker portable assets before delivery

ACTIVE 2026-11-10 for DeepSeek
`/root/read_native_handoffs_release10_ui_feasibility_task_md`. Root allows this
disjoint doc-only preparation in parallel with Stage2 implementation. Base facts
on accepted protocol/view contract and specified Stage2 behavior, mark handoff as
pending final app confirmation before packaging. Do not review changing app as
final. Root will reconcile exact UI wording with frozen Stage2 result. .18 package
never rewritten; no .19 packaged/accepted claim yet.

Owner will update native/portable/README.md, native/portable/NOTICE-PREAMBLE.md
only if needed, and native/THIRD_PARTY_NOTICES.md only for true provenance scope
changes; no new dependency is planned. Runtime copies actual source notices into
packages, so update accurate module paths/first-party engine scope if required.
No tests/builds/launches. Keep generated .18 asset files immutable.

Actual current portable README hardcodes three workers, no drawio worker,
drawio all host-side, and total36 retained engine packages. After stage2 accepted,
document four fixed engines, drawio SVG current-page independent worker, readonly
mindmap scene vs host native rendering/editing, source-mode drawio select-page
instruction, worker32MiB input/64MiB SVG output/30s timeout and issue limits as
implemented. Source text is local immutable snapshot, original file untouched;
no first-page fallback, no arbitrary executable/plugin sandbox claims.

Update explicit-import/activate/reset copy (all4), old3 schema1 selections load
preserved and new drawio bundled, four-engine schema2 persistence, per-engine
retention12 (total48) without adding deletion UI. Preserve cleanup poison,
whole-directory distribution and actual limitations. Version template remains
{{VERSION}}; no hardcoded .19 claim before binary accepted. Whole-app memory not
measured. Keep ordinary user guide concise and move implementation internals to
source docs where available.

Read final accepted implementations before stating behavior; no speculative
completion. One scoped link check, final changed hashes/result in
drawio-worker-portable-docs-result.md. Freeze assets for .19 build, then full
six release docs can follow actual binary acceptance separately.
