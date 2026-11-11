# Next everyday workflow: find an already open document

ACTIVE read-only planning2026-11-10 for Astra, while composition source stays
frozen for independent review/final gate. No source/tests/docs changes, no
execution or children. Current accepted runnable .28; .29 preparation proceeds
elsewhere. This task does not activate a new feature before .29 closure.

The next product need is navigating many open files without widening the tab
strip or keeping more editor views alive. Use the familiar searchable open-file
list pattern of mature editors, adapted to this product's existing native UI and
maximum100live tabs. Do not claim a fresh external software audit; this is a
design reference. Scope only already open documents: no workspace index,
background scanner, recent-file DB, network or new parser/dependency.

Inspect actual app state, shortcut/focus/utility input composition patterns,
Tab identity/Select/Flush and public MyGo list/input APIs. Save one concise
open-document-picker-next-plan.md with concrete source locations and smallest
design: native entry + keyboard accelerator if available, type-to-filter by
title and path, dirty/readonly/binary badges only where useful, visually distinct
same filenames, keyboard up/down/Enter/Escape and click. Retain bounded stable
IDs, not buffer/view pointers or stale index ownership. Selecting must go through
existing Select so failed Flush preserves the original draft and focus.

Define exact composition handling for source/rich/utilities when opening,
query preedit and committing/cancelling picker; ensure Esc doesn't discard
unaccepted source input or cause focus to move to a wrong tab. Filtering must
never enumerate disk or modify source, tab order, undo, or tab count. Match
Unicode reasonably with bounded strings; if not fuzzy, describe as substring.
Use one simple panel that fits760x480 and supports up to100entries without
rendering every editor or unbounded row detail. No animations.

Give a small meaningful test plan and ownership slice; highlight only genuine
unresolved public toolkit constraints. No code/pseudo-framework or preparatory
test execution. Keep artifact compact, then stop and return to final gate only
when root activates it. Next implementation awaits .29 acceptance.
