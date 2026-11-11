# Active composition coalescing checkpoint

2026-11-10 Astra. Read current task, contract review and root refinements.
Implementing only core/text/buffer.go,ui/editor/editor.go and new focused tests.
.28 package/evidence remain immutable; no child/package/host work.

Chosen shape: Buffer issues an opaque nonzero-sized identity token for each
typed composition group. Token references identity only, not text/transaction;
validation checks the exact current group in its issuing buffer. Ordinary
Replace/End/Cancel reject typed groups; Restore/ResetHistory invalidate tokens.
Editor keeps its existing buffer owner plus token and validates Flush/end/cancel;
stale Escape clears local ownership after one guarded cancellation attempt.

Coalesce only the most recent accepted full UTF16 span. Keep the last change's
original/current strings; exact successors overwrite current without slicing
old text. Irregular spans append a new suffix. Net-zero removes that suffix but
retains one baseline for later update/cancel, preserving before-selection and
prior redo. Ordinary groups remain append-based. Use saturating retained-byte
accounting with overflow awareness/recount on shrink so saturation cannot hide
remaining overflow. Accepted same-text marked updates still call changed once;
no text revision is fabricated. Final ReplaceText uses the same typed path.

Next: implement, then structural hundreds-update/ownership/net-zero tests plus
real Tester callbacks/Escape and existing growth/lifecycle seams. Focused runs
only, saved streams/real exits in fresh active-composition-coalescing-evidence-20261110.

First implementation batch complete: exactly2runtime+2newtest files. Typed
Begin/Replace/End/Cancel plus OwnsComposition validates identity+issuer; token
contains only a nonzero private identity pointer. Composition suffix state stays
in Buffer, disappears on end/cancel/reset. Net-zero clears dead edit slots and
keeps one baseline. Ordinary history accumulation now saturates safely; overflow
is explicit and a shrinking suffix recounts retained edits before clearing it.
Editor final commit, Flush/unmark and Escape use tokened operations. Restore/
reset stale-token Escape clears local owner without cancelling a foreign group.
First-refusal final retry also refuses an intervening ordinary group.

Added600update structural tests and240update real Tester callbacks, initial
same-text/net-zero/redo, irregular suffixes, atomic refusal, foreign/buffer-copy/
reset ownership and tiny accounting boundary tests. No app or format edits.
Starting new focused tests plus existing text/editor composition/growth seams;
after pass, run selected existing real app/Markdown/graph lifecycle seams only.

Frozen2026-11-10:01focused Go/shell0 text0.594s/editor1.286s;02selected lifecycle
Go/shell0 Markdown1.782s/graph1.685s/app2.857s. Both stderr empty; no failures,
reruns or subsequent source changes.4/4manifest matches first-run identities.
Result active-composition-coalescing-result.md published; owner stopped all
runs/edits pending independent review and root final activation.
