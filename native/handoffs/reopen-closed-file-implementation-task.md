# Reopen closed saved files implementation

ACTIVE2026-11-10 after markdown-image-reference-release26-root-acceptance.md.
Owner Astra /root/read_native_handoffs_tab_workflow_resume_task_md.
No child delegation. Root decision and next plan in sibling handoffs define
the behavior; root-decision overrides any conflicting plan detail.

Own native/app app.go/open orchestration, binary open adoption receipt in
binary.go/binary_actions.go as necessary, view.go command/menu wiring, new
reopen_closed.go, workspace.go shutdown cleanup and focused app tests. No core/session schema, format, decoder,
vendor, package or product-doc edits. Keep source changes minimal; the shared
private open receipt is only for the reopened-file path, preserving existing
normal callers/ordering. Explicitly recheck target identity after a CloseTab
SaveAs/dialog before capturing/removing. Whole-window shutdown never records.

Implement16 lightweight cloned path+ID entries, newest first with same identity
dedup; clear removed backing slots and clear on shutdown. In-flight state holds
small IDs/outcomes, no closed Tab/Buffer/View/source/bitmap/callback retaining
those resources. Consume only exact history ID on actual successful Select or
new tab adoption; stale/timeout/refused/canceled work retains history and cannot
later open or consume a replacement. One in-flight reopen, no hidden retry.
Preserve byte fallback as a valid opened result and preserve current errors.

Use compact public native menu by Open and Ctrl+Shift+T, bounded readable entry
labels/tooltips, clear empty/busy/result wording and older-item selection. Avoid
extra permanent panel. Guard utility IME before focus transfer and at adoption;
source/rich composition follows Flush with conflict recovery. Preserve ordinary
open, last-tab welcome, tab reveal and narrow760x480 layout.

Run focused meaningful regressions on receipt/close/history/input and actual
app open paths first. Cover text/PNG/bytes/current replacement/unknown encoding,
SaveAs/discard/no path, canceled or failed close/persist, existing-tab Flush
refusal, pending async + newer close/reclose, modal/preedit, canceled/timeout/
shutdown/late delivery,16-entry bound/dedup and no backing retention. Preserve
all initial failures and explicit forwarded numeric exits. One optional
760x480Tester menu capture is sufficient if visual geometry needs confirmation;
no physical input/resource/package claims. After source freeze report exact
identities/results; root will activate one full app test/vet, no unrelated suite.

Write a short implementation checkpoint and final
native/handoffs/reopen-closed-file-implementation-result.md. Keep actual command
outputs/exits in a fresh handoffs evidence directory. No .27 build/host is
implicitly authorized. .26 package/evidence stay immutable; workspace runtime
source is now released for this assigned implementation only.
