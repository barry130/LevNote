# .21 docs: three factual corrections only

ACTIVE 2026-11-10, existing GLM docs owner. Root read the submitted result and
actual docs. The broad update is sound, but these exact discrepancies remain.
No product/evidence/tests/builds/runs or new general link scan.

1. native/VALIDATION.md current .21 aggregate SHA is transcribed incorrectly:
   it says `...B868EEB9B96915...`. The actual immutable aggregate and root
   acceptance say
   `E34D2FE39096978E88FB5DB2B868EEB9B96315EA141D73E879B0910F191D1FB8`.
   Correct that one value from the actual file, not memory.
2. drawio-efficiency-release21-delivery.md still says the202-file root
   "includes the corrected aggregate streams". It does not. Replace only that
   sentence to say it includes preserved first-attempt evidence and final
   aggregate/validator; corrected invocation streams are in the separate4-file
   retry sibling. The same file still says "Job held1process ... after exit".
   Replace with actual Job TOTAL processes1 and explicitly no owned process
   remains after exit. These two requested repairs were claimed but not applied.
3. In VALIDATION's benchmark paragraph, preserve the observed timing direction:
   median ns/op Parse1082480->1244525,Update2291545->2478320. These are short
   samples and not a speed conclusion, but simply saying fluctuation hides the
   actual worse samples. One concise sentence suffices; no new benchmark.

Update only the identities affected in docs-result.md, add a one-paragraph
correction note, verify these literal facts against existing raw records, and
finish. No further prose polishing or scope expansion.
