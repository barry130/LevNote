# Enforce the existing100-tab recovery ceiling in the live workspace

ACTIVE2026-11-10. Sole implementation/test owner fresh DeepSeek
/root/read_native_handoffs_live_tab_admission_implementation_task_md. Prior
DeepSeek follow-up ended with empty output and no saved source/result changes;
it has no ownership. No child delegation;
compact before200K. Reopen source is accepted and Astra has released its app
files. Read live-tab-admission-root-decision.md, reopen-closed-file-root-acceptance.md
and reopen-and-tab-admission-stage-plan.md. Current .26 package/evidence immutable.

Implement the bounded increment, not another plan. Own necessary native/app
runtime files plus new live_tab_admission_test.go; preserve existing reopen
tests except a justified regression addition. No core/session/schema, decoder,
vendor, plugin, format, package or product-document changes.

Use one app helper based on session.MaxTabs(100). Refuse genuinely new tabs with
an actionable message, without eviction/reset/hidden truncation. Inspect all
production creation routes: NewDocument, NewFromPlugin, recoverDraft, ordinary/
batch text or byte fallback, asynchronous binary adoption, session restore.
Check before new buffers/templates/IO/Flush/release where possible; retain a
final adoption recheck because UI state can change during IO or synchronous
Flush callbacks. Same-path reuse must precede cap rejection and remain usable
at100. No reservation-based complexity is needed if existing serialization
plus final checks guarantees the invariant; keep actual lifecycle semantics.

Refused recovery callback returns error and leaves the owning rich-view draft
intact. Refused async results complete as failures and release unadopted payload
through existing cleanup. A capacity refusal must not consume a closed-file
receipt/history entry. Preserve existing active document/selection/history and
previous recovery bytes; don't persist or flush merely to report refusal.
Successful restore of100 records must work without an extra welcome slot.

Focused tests must cover99/100/101 and distinct creation origins, early refusal
without Flush/allocation side effects where observable, existing-path reuse,
async capacity changes, post-Flush synchronous tab creation, binary payload
cleanup and exact reopen entry retention, close-one/retry, recovery callback
draft ownership and restore100. Use tiny fixtures, deterministic public seams
and actual existing test helpers; avoid100large buffers or timing guesses.

Run only affected focused tests while implementing, through
native/scripts/go.ps1 -GoArguments @(...). Preserve real failures and explicit
exit codes/streams in a fresh handoffs/live-tab-admission-evidence-20261110.
When focused work passes, freeze source and publish
live-tab-admission-implementation-result.md with changed files, meaningful
coverage and limitations. Do NOT run the full app suite/vet yet: root will
request one combined final run after independent review. No builds/processes/
screenshots/resource measurement now. Send a concise checkpoint after code is
ready for read-only review; no separate approval needed to implement.

Root route-review refinement: a rejected new path in OpenMany must not abort
the remaining ordered list merely because it hits capacity. Later already-open
paths should still be selected/revealed, and later new paths get explicit
refusal. Keep actual failure semantics distinct from dispatch/shutdown abort.
Also recheck same-path reuse at async text adoption for normal requests as well
as reopen receipts: another UI operation can have opened/saved that path while
loading. Add focused mixed-batch and late-duplicate boundary regressions where
the existing dispatch seams allow deterministic simulation.
