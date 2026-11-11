# Growth audit accepted; proposed implementation needs refinement

2026-11-10. Root verified the source gap:128MiB is a stable-file-read bound,
not a live UTF8-buffer limit; ReplaceUTF16 has no growth admission.100tabs is
only a count bound. Accept this finding, not yet the audit's implementation
proposal. .27 remains frozen through delivery.

Refine before implementing:
- text cannot import document to reuse MaxEditableBytes: document already
  imports text. Keep dependency direction and put shared policy at an appropriate
  lower layer, or pass a buffer budget explicitly from the document owner.
- UTF16/GBK stable input<=128MiB can decode into more than128MiB UTF8. Consider
  existing seeds/snapshots and read-only inspector behavior; do not make them
  panic, silently truncate, or break untouched-byte preservation.
- Never break accepted Undo/Redo/CancelGroup states merely by lowering a new
  limit after edits. A fixed per-buffer policy should make their accepted
  reachable states legal, or explicitly grandfather existing oversized seed
  states. Do not change bool Undo APIs into silent refusal or clear history.
- Check prospective logical size via validated byte offsets before SliceUTF16
  materializes removed text/history. Use overflow-safe subtraction/comparison.
  A guard after a full old slice does not satisfy before-allocation wording.
- Snapshot Restore currently returns void and is used by Revert; constructors
  likewise return pointers. Choose a coherent compatible policy rather than
  asserting every seam now returns errors. No caller may silently ignore refusal.
- ReplaceAll/indent/format group rollback must retain exact bytes/selection/
  history even if a late sub-edit hits limit; CancelGroup increments revision
  today. Do not claim revision unchanged from existing rollback. Prefer
  preflight where a whole-command size is known; state exact partial-group rules.
- Buffer admission cannot undo the clipboard/regex/codec allocation that
  created the input string. State this boundary. Clipboard currently appears
  read twice when an injected Paste provider is present; assess separately,
  not as proof of an entire memory ceiling.

Next step is one concrete small design against actual APIs, with no product
execution or source changes before .27 closure. Avoid blanket checking of every
method when accepted-state invariants already prove safety. Test small explicit
limits and real rollback/IME/recovery seams; keep legal edits and fidelity.
