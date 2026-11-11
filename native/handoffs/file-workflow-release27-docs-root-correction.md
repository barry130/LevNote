# Narrow .27 docs correction before acceptance

2026-11-10 root review. DeepSeek remains docs-only owner for this correction;
no product execution/source/package/evidence changes.

1. native/VALIDATION.md current .27 section says41 alive checks. The actual
   frozen host-smoke/run-1-mixed-cli-open/result.json has
   SessionSettlingAliveChecks=46 and5005.7892ms. Correct41 to46.
2. The current evidence-limit prose says it does not prove "binary delivery"
   / "二进制交付". .27 did deliver and validate a binary package. If intended
   to distinguish new behavior in the actual GUI, state "physical interaction
   of the packaged binary" / "交付程序中的物理交互行为". Keep lack of physical
   IME/keyboard/rendering/resource/dirty-recovery proof explicit. Remove this
   contradictory blanket delivery denial from affected current prose/result.
3. Complete historical sections were required, not only headings. Use the
   available pre-edit backup or accepted prior docs to confirm .26/.25/.24
   section bodies were retained, allowing only current-to-historical labels.
   Do not rerun the already passed559-link/fence sweep for two wording edits.
   Check affected wording/identities and write updated result. No extra large
   diagnostics document needed.

Root read current docs and accepted .27 actual host facts; this is a factual
docs correction, not reason to rerun any product check.
