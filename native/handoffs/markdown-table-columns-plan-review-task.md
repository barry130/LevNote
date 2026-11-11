# Independent table-column plan review

ACTIVE read-only2026-11-10. GLM docs owner, no child delegation. .23 docs are
accepted. While .24 package runs, review markdown-table-columns-next-plan.md
against actual formats/markdown/table.go, table_rows.go/table_alignment.go and
modules/markdownview/table_actions.go. No implementation/check/build/launch.

Write one concise markdown-table-columns-plan-review.md with concrete fixes
only, no generic audit. Focus on column count vs zero-based selected index,
optional leading/trailing pipes and empty inserted edge columns, escaping when
an unchanged cell ends in backslashes, deletion2->1 (mandatory leading pipe),
per-row offsets/selection mapping and grouped undo. Ground the advice in the
current parser, not wider CommonMark expectations. Do not demand parser changes.
Clarify any class of valid current source whose edit must be explicitly refused
to preserve content. Full candidate parse/warning checks must not hide silently
lost outer markers or changed untouched cell bytes.

The intended implementation may use multiple original-source patches in one
undo group. Prefer minimal extension of existing row/alignment contracts rather
than redesign. Detect actual plan problems; do not implement or test them.
Report and stop; product docs .24 will be assigned after binary acceptance.
