# Finalize portable composition guidance only

ACTIVE2026-11-10. Owner
/root/read_native_handoffs_growth_release28_preparation_task_md. Prepared
helpers structurally accepted in growth-release28-preparation-root-review.md.
Do not rerun parser or alter helper/evidence files. No product execution.

Astra implemented the root correction in ui/editor/editor.go; details are in
document-growth-budget-progress.md and document-growth-composition-root-correction.md.
Read actual current code and add a short paragraph only to portable/README.md:
after refused IME preedit/final composition input, accepted partial text stays
pending; save/switch/close wait until a valid shorter retry or explicit Escape
in the editor. Focus loss/navigation must not silently settle it. First refused
preedit also needs retry/Escape. Rejected incoming text is not copied/retained
by the editor; do not tell users it is safely stored as a recoverable draft.
Ordinary rejected paste does not permanently block saving. Distinguish this
from independent rich-format drafts, which are retained on host refusal.

Keep it concise and user-readable; no implementation class names or exhaustive
architecture. Check only added wording and identity; no repeat88phrase parse/
link/full sweep. Publish growth-release28-portable-final-result.md with actual
portable bytes/SHA, leave prior preparation report as historical pre-correction
identity. At delivery context initialization current portable identity is read
normally; no helper change or new stage framework needed. Then release ownership.
