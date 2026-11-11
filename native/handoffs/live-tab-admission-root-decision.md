# Live-tab admission next-stage decision

PLANNED2026-11-10, not active. Root accepts the concrete source finding in
native-memory-next-increment-review.md: session.MaxTabs100 and unconstrained
live creation can disagree, leaving additional live tabs outside recoverable
capacity. Root read actual appendTab/appendBinary/NewDocument/openRequests and
session builder seams. Treat this as a recovery-consistency and bounded growth
increment, not a measured process-memory optimization.

Implement after Astra freezes/releases the closed-file workflow source. Do not
mix it into that feature or change current .26 package. Keep the existing100
ceiling, no eviction, no history/content reset, no schema change. An already
open file remains selectable at the cap. Refuse a genuinely new tab before
current-draft Flush, presentation release or IO where the route allows, and
recheck before asynchronous adoption. One shared app admission helper should
use session.MaxTabs. All public creation routes must converge on it, including
plugins, recovered drafts, batch text/byte fallback and binary completion.

Specific refinements to the proposal:
- NewDocument/template/recoverDraft must check before allocating a new large
  buffer, not only inside addDocumentChecked after a document was constructed.
- Restoring the accepted maximum must still work; restore must not create a
  hidden welcome tab that consumes a slot. Inspect actual recovery orchestration.
- A recovered local draft refused at capacity must remain in its owner view;
  propagate the callback error and never release it to satisfy admission.
- An asynchronous image/text refusal at adoption must complete the original
  request as failure, retain closed-file history if applicable, and dispose its
  unadopted payload through existing ownership. Do not return a success receipt.
- Preserve prior recovery file bytes and visible active document/history on
  each refusal. Opening an existing path may perform the normal guarded switch.

Use a focused boundary group covering99/100/101 and actual creation origins,
async capacity change, duplicate select, close-one/retry, and refusal without
Flush/source/history/session side effects. No performance claims from these
tests. Affected final app test/vet should run once after final source freeze,
not repeatedly while unrelated owners write. Task ownership/activation follows
later; this planning file authorizes no source or product execution.
