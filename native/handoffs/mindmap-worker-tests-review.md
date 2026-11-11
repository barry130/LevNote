# Root review of command test checkpoint

2026-11-09, first main_test.go read (14156 bytes, 06:56 checkpoint).
Main production source has no new finding. Test owner should correct these
test/oracle gaps before its final owned command gate:

- SVG multiline labels are one text element with two tspan children. Decode
  nested tspans instead of expecting two text/chardata elements. Preserve the
  independently computed geometry and escaping checks.
- Deep fixture currently lacks each node's closing brace and repeats ID n.
  Construct valid JSON with unique IDs and assert json.Valid before expecting
  parser depth source-limit. Malformed JSON must not masquerade as depth proof.
- Hidden metadata fixture currently has no collapsed ancestor. Make a true
  hidden descendant own image/color/long-label metadata, check disclosure and
  verify its label is absent from SVG. A separate short fixture should check
  exact icon/topic/tag label; the current long topic truncates tags.
- Exercise warning-limit through actual serve wire, not only wireWarnings:
  small valid tree producing257warnings, plus overlong unknown layout/ID
  warning; both must produce coherent error metadata with no partial payload.
  Retain exact256 helper boundary and invalid lengths/severity coverage.
- Remove errors.Is(io.ErrShortWrite,io.ErrShortWrite) tautology and unused import.

These are test corrections; do not weaken implementation to match wrong
expectations. Preserve all attempts actually executed, then record real command
tests/vet with before/after owned and frozen dependency fingerprints. Root only
reads/reviews, implementation and execution remain delegated.

07:01 follow-up of revised16497byte test: excessive-warning fixture root is
missing required topic; add topic:"root" and require ONLY warning-limit, remove
the render-failed alternative. Otherwise the oracle merely proves invalid
schema rejection. Assert json.Valid. The single short tag label must be
"⭐ custom<& root [甲<&]"; DisplayLabel has no leading separator on first tag.
Use the exact escaped equivalent. Deep unique-ID/valid-JSON and hidden collapsed
descendant changes look correct. Finish finite owned tests/vet after correction.

07:03 corrected16558byte source is read; no further source finding. Current
evidence JSON embeds only a summarized ok line for a -v command, and has no
retained actual streams/before-after pair. Preserve it as preliminary summary.
Run one final narrow actual test/vet in a fresh subdirectory with explicit
GoArguments arrays and separate child invocations (go.ps1 exits its shell).
Redirect stdout/stderr directly, programmatically capture code/time, and record
owned/dependency hashes before/after. Actual RUN/PASS log, not manually authored
measurement or synthetic lines, is required for package acceptance. No broad
suite, production executable or real worker launch is authorized at this step.
