# Live-tab admission independent final review

Final review2026-11-10. GLM performed the exact read-only review of DeepSeek's
frozen result and nine-file source manifest. No source, test, product-doc,
package, or evidence changes, tests, builds, launches, resource measurements,
or child delegation were performed.

## Result

No outstanding source-grounded defect was found in the frozen live-tab
admission slice. The cap is enforced through the shared `admitNewTab` rule on
the reachable production routes, with same-path duplicate handling before
rejection and no eviction, reset, hidden truncation, or receipt consumption on
capacity refusal.

## Exact frozen identities reviewed

All nine `source-frozen.json` entries matched by length and SHA256 at review
time:

- `app/app.go`,38935B,
  SHA256 `0AC7FB907C83DC3C2B7642B197441492274CBB1CBC75E6688264D577D995ADB4`
- `app/reopen_closed.go`,7895B,
  SHA256 `BFCA75BA0E2A878483133ECE43530742C424623CBE6B8809F1285C9E04C62E92`
- `app/binary.go`,10170B,
  SHA256 `AE2F059F0E91C812A82FFE3F0317EAFE7E2F0B0CE9AEB3AA5056F713C555226D`
- `app/binary_actions.go`,10173B,
  SHA256 `D9DB7FAA2CF674ED3428F67479D6377B4901ED7C7A14C404F0536F325A9609BB`
- `app/view.go`,23002B,
  SHA256 `195AFA3D82E7612A0CDBEC6A9D0F0432777F492CAB4D94596B0C6F2329C83456`
- `app/workspace.go`,6715B,
  SHA256 `B1492575D6C64673BF02E633C0D7049F47BF151391B4CA58F58F3779599AF259`
- `app/state.go`,13647B,
  SHA256 `7F3F8E338E5619241FA3D0ACC72BA791778F1CE2A328272CBAED12E40B9400FD`
- `app/reopen_closed_test.go`,28867B,
  SHA256 `3D160E6CCA9602F285D1F0A19A1DC450EE8109F5E3B81DEDDC33C6DF037B2C0B`
- `app/live_tab_admission_test.go`,20902B,
  SHA256 `285244AD5A24FFB927F57CDA28A8AD2F5CDB7F09FE7A52960EDDBD74BAA8766D`

The frozen manifest identity itself is
`940C698E0B0195258649E6A0747A8F1EBAF1F754BAE05E6539B790B023098DAF`, as
reported by the implementation result.

## Production semantics reviewed

- `app.go:448-452` defines the single `session.MaxTabs` admission rule. It
  rejects only a genuinely new tab and names the close-one-then-retry action.
- `app.go:461-498` rejects before `Flush`, then performs a final recheck after
  Flush and any `beforeAdd` callback, but before export invalidation,
  presentation release, or `appendTab`. This keeps the original tab alive if a
  synchronous callback fills the cap.
- Template and plugin creation reject before template allocation
  (`app.go:525-552`), and recovery rejects before draft-size validation or any
  presentation release (`app.go:619-646`). A recovery refusal returns the
  admission error to its caller and leaves the owner's source/history,
  selection, view, and disk/session bytes untouched.
- Ordinary and batch opens run duplicate selection before admission
  (`app.go:723-750`). A new-path capacity refusal records the path-specific
  error and continues the ordered batch, so later already-open paths remain
  selectable and later new paths get explicit refusals.
- The second text-adoption UI phase retries the duplicate path before
  `addDocumentChecked` (`app.go:758-775`). Therefore a path opened while IO was
  running is selected instead of rejected.
- Binary opening follows the same order: receipt validation and duplicate
  selection before an early admission check and IO
  (`binary_actions.go:39-55`). Completion retries duplicate paths, checks
  capacity before Flush, and performs a post-Flush/post-callback final check
  before releasing the old presentation or appending
  (`binary_actions.go:217-263`). Receipt acknowledgement occurs only after
  real tab adoption (`binary_actions.go:263-271`).
- Restore checks each record independently and can restore exactly100 mixed
  text/binary records with the active selection mapped to one restored tab and
  no welcome slot (`state.go:45-119`). Root adjudicated that the absence of a
  second Restore-local check after binary IO is not a demonstrated reachable
  defect: Restore is synchronous on the UI owner, waits without pumping UI
  callbacks, and the binary worker has no App authority. The existing early
  record check is therefore sufficient for the reachable current model.
- Direct production append callers converge on the checked paths above; the
  remaining direct `appendTab`/`appendBinary` references found were tests
  creating fixtures, not production creation routes.

## Test-quality review

The focused admission group is substantive rather than test-shaped. It covers:

- 99→100 success and100→101 refusal for template, plugin, and recovery origins,
  including absence of template/next-ID/Flush/release/source/history/selection
  and prior-session-byte side effects.
- Mixed capped `OpenMany`, explicit continuation after refusal, and later
  already-open-path selection.
- Asynchronous text capacity change and late duplicate adoption through the
  actual UI dispatch/update-wait seams.
- Synchronous recovery creation during text and binary `Flush`, proving that a
  full-capacity race does not release the original presentation.
- A real Markdown conflict view: its public `RecoverDraft` callback is refused
  twice at100 while source/history/selection/view and the disk file remain
  unchanged, then the same retained draft is recovered after close-one.
- 99→100 byte fallback, explicit second-path refusal, exact closed-file entry
  retention, close-one and retry.
- Reuse of an already-open path for a closed-file receipt at100.
- Async binary capacity change, late duplicate, and post-Flush refusal, with
  the task slot, lane reservation, and binary snapshot cleared.
- A mixed100-record restore with exact count/order, active selection, text and
  binary authorities, and no published binary snapshot.

These tests use the actual existing seams (`openRequests`, `updateWait`,
`uiDispatch`, `flushTab`, `openBinaryRequest`, `reopenClosedFile`, real Markdown
composition/recovery) rather than bypassing the admission logic. They are
focused and bounded and do not make performance claims.

## Evidence limitation

Retained evidence entries004-007 are real failed test-construction attempts and
are correctly preserved. The README also describes an inherited, unretained
report of an earlier release-order defect. That claim should remain an explicit
evidence limitation or be clarified by its originating owner; root observed that
the prior owner ended empty with untouched source, so this review does not
treat the attribution as established history. The corrected release ordering is
nevertheless directly verified by the retained008/010 post-Flush tests and the
frozen source.

Final acceptance still awaits root's one combined full affected app test/vet,
as stated by the frozen result. This slice itself has no build, launch,
screenshot, physical-input, or resource evidence.
