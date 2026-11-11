# .28 portable composition guidance finalized

Finalized2026-11-10 under
`growth-release28-portable-composition-task.md`. Preparation ownership is
released.

`native/portable/README.md` now adds one concise user-facing paragraph: a
refused first or ongoing IME preedit keeps accepted partial text pending; save,
view/tab switch and close wait for a valid shorter retry or explicit editor
Escape; focus loss/navigation cannot silently settle it. The editor does not
copy or retain rejected input text, so it is not described as a recoverable
draft. An ordinary rejected paste is nonsticky. Independent rich-format drafts
that survive host refusal remain explicitly distinguished.

Final portable identity:18,725B/SHA256
`7B25317E6480956154164323D77057883DF1AB17BBB61A868F6FFA8BABDEBFCB`.
Only the added wording and this identity were checked. The prior preparation
report remains the historical pre-correction identity18,198B/SHA256
`4340A1E0D888BECEC80A765FE63EBA349B2E8A1B631813EDB47ED8152C99245B`.

No helper, provenance, static-parse or other evidence file changed. The parser
was not repeated. No88-phrase contract, link/full sweep, test/vet/build/package/
host/worker/resource/screenshot or other product execution ran. Delivery
initialization will read the current portable identity through the existing
flow; no helper or stage change is required.
