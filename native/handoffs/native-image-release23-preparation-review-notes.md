# .23 preparation review notes

2026-11-10 root WIP read-only notes for the preparation owner. Do not execute
product work. Finish helpers and then one static parse; preserve copied .22.

1. Include binary-copy-receipt-root-acceptance.md among the source gates. The
   host copy path depends on it in addition to the binary core; the current
   six-gate list omits that accepted extension. Composition acceptance exists;
   app acceptance remains pending. Avoid stale comments claiming both absent.

2. First mixed CLI start must finish explicit opening before WM_CLOSE. App now
   refuses/cancels close while explicit binary work is in flight so its outcome
   can be shown, which is intentional. Replace fixed 1800ms readiness assumption
   with a bounded poll for the actual expected mixed session shape/active record
   (using existing strict facts helper and verifying no process exit). Automatic
   persistence is available after opening. Keep one owned WM_CLOSE after this
   prerequisite, not a repeated close that conceals a lifecycle failure. Polling
   session confirms open/persistence readiness, never pixels/decoder/driver.

3. View bounds still concern a supported PNG/JPEG subset; use actual current
   helper/codec result wording in portable text. Normal open/reload can hold the
   old active image/source while a single replacement read is pending adoption;
   app source/independent review is settling the exact bound. Leave that prose
   explicit and avoid a guarantee of exactly one full source at all moments.

4. Runtime source scope must include the already accepted patch source entries
   by their actual native-relative paths. Existing source-scope code does this;
   no full toolchain/vendor inventory is needed. Derive payload set/count from
   real asset copies and fixed binaries/manifests (the added bitmap patch changes
   the previous release's count). All aggregate fields must match the actual new
   per-run host result schema; static parsing alone cannot confirm those names.

5. Portable README needs plain Chinese, without `path-only`, `single-lane`, or
   `单车道行为` in user instructions. Say recovery reads the saved path again,
   and image reading/decoding/copying run one at a time. Remove the helper's
   literal assertions for those English implementation labels. State that
   baseline 8-bit JPEG supports grayscale/standard three components, not CMYK
   or progressive JPEG. Clarify >16MiB candidates keep bounded byte inspection
   and cannot use this bounded image copy operation. The accepted snapshot may
   also be retained during a native Save As dialog; avoid wording that claims
   explicit open/reload is the only possible transient source reference.

6. The imageview toolbar source is narrowly reopened. Before delivery include
   native-image-view-narrow-toolbar-root-acceptance.md as the new layout gate in
   addition to its original source acceptance. Final app acceptance must bind
   the corrected integrated capture and combined suite. Preparation may finish
   without these future gates; execution still awaits root activation.

7. Independent helper review is saved in native-image-release23-helper-review.md.
   Root agrees its four bounded findings need correction. Pre-run session
   baseline plus fresh post-close publication after this process start is needed
   for run2; seeing run1 JSON is only pre-close shape observation. Record actual
   sequential creation/completion times, not unequal reusable PID/handle values.
   Both toolbar and final app acceptance now exist; include them at initialization.

8. Keep aggregate metadata repair small: preserve each attempted validator,
   aggregate and capture output under a distinct attempt name/directory, allow
   at most the explicitly chosen metadata-only retry after failure, and never
   rerun baseline/package/hosts. The current aggregate writes aggregate.json
   even when Pass=false, so merely numbering outer logs while retaining an
   unconditional aggregate.json-exists guard is insufficient. A successful
   aggregate must still prohibit rerun. Don't alter original attempted files.
   No general retry framework or extra product execution is requested.

9. Root's 13:05 WIP read found `if (Test-Path ... -PathType Container -and ...)`
   in aggregate.ps1. Parenthesize the Test-Path command before `-and`; otherwise
   PowerShell treats `-and` as a cmdlet parameter despite static syntax passing.
   Attempt2 must also handle a preserved attempt1 exception before aggregate.json
   was written: existing attempt directory + recorded failed invocation and no
   successful result is enough; do not force a nonexistent prior JSON. Preserve
   failed captures. Session publication booleans must be enforced before a host
   run reports PASS, as well as in aggregate. Run1 requires absent pre-run state,
   run2 requires its captured baseline and exact preserved copy.

10. Hidden-readiness feasibility is being traced read-only by Astra in
    native-image-hidden-readiness-review-task.md. Root found that MyGo's hidden
    Surface draws only its first frame, while App automatic persistence runs
    from View. Therefore item2's assumption that the mixed session must publish
    before close is not yet established. Keep preparation only; wait for that
    narrow verdict before freezing the readiness condition. Do not patch
    product/vendor or add paint/input automation to satisfy the smoke helper.
    Preserve any static parse already run; a subsequent actual helper correction
    justifies one scoped parse of the corrected files.

11. The checkpoint's example commands currently initialize and execute in the
    preparation directory. Final delivery must copy the reviewed helpers to the
    fresh artifacts/acceptance/native-image-release23-20261110 directory, then
    initialize and execute there, as specified in the delivery task. Correct
    the final result's example commands; preparation evidence remains separate.

12. The new run-host-smoke pre-run copy code converts LastWriteUtcTicks with
    DateTime.FromFileTimeUtc. Get-FileFingerprint stores DateTime.Ticks (epoch
    year0001), not Windows FILETIME (epoch1601). That conversion sets a wrong
    timestamp and makes run2's own exact-baseline check fail. Preserve the copy's
    actual LastWriteTimeUtc directly or construct the UTC DateTime using the
    matching DateTime tick epoch. Do not change frozen .22 evidence timestamps.

13. Astra confirms the hidden pre-close session poll is unreliable. Apply the
    binding native-image-release23-hidden-readiness-decision.md:5000ms bounded
    settling/alive checks, one owned close, strict fresh post-close mixed session.
    Pre-close state is optional/descriptive. Update host and aggregate schemas
    together, then finalize preparation. This overrides item2's mandatory poll.

14. In the corrected aggregate schema, SessionBeforeCloseFingerprint is still
    asserted unconditionally despite its observation being optional and allowed
    to fail. Validate its fields only when non-null; keep post-close fingerprint
    mandatory. Dispose each held Process after final fields/result are captured,
    including the failure branch, so the two-run helper releases owned handles
    explicitly. Then freeze/parse; no further broad review is requested.
