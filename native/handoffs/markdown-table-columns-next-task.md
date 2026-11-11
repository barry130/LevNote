# Next incremental design: Markdown table columns

ACTIVE read-only planning2026-11-10. DeepSeek image keyboard owner may inspect
existing Markdown table format and UI code and write one concise handoff. No
product edits/checks/builds/launches while .24 is being validated and packaged.
No child delegation. Keep inherited context bounded below user200K threshold.

User goal is one simple native reader/editor, self-authored engines with local
format plugins. Current tables already have cell/whole-source editing, data-row
insert/delete and alignment. Next candidate adds insert-column-before/after and
delete-current-column as practical table-editor basics without extra framework.

Inspect existing source contracts and preservation rules. Produce a concrete
implementation plan in markdown-table-columns-next-plan.md: exact owning files,
existing reusable parse/offset/transaction functions, chosen behavior for ragged
rows and escaped pipes, minimum one column, newline/spacing/unknown-content
preservation, current-cell draft flush/conflict/IME/readonly/undo semantics,
bounded UI buttons and keyboard access within existing panel, and a small
behavioral test set. Include go/no-go risks grounded in actual code. Do not
broaden parser support, silently normalize whole documents, invent external
benchmark claims or introduce a new editor dependency. Mark unsupported syntax
explicitly and retain source editing. No web research or competitor claims are
required for this bounded implementation plan.

Report design and stop; implementation will be separately activated only after
.24 package inputs are released. One actionable plan, not another generic audit.
