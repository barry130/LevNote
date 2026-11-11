# Live-tab admission implementation — frozen

2026-11-10. Assigned implementation and focused validation are complete. The
native app source is frozen for read-only root and independent review.

## Behavior

- `App.admitNewTab()` is the shared admission rule and reads
  `session.MaxTabs`; it refuses only a genuinely new tab with an actionable
  close-one-then-retry message.
- New document, plugin template, recovered draft, ordinary/batch text open,
  byte fallback, asynchronous binary open and session restore converge on that
  rule. Early checks precede template allocation, document/source IO, payload
  Flush and presentation release where each route permits.
- Same-path reuse runs before cap rejection and remains usable at100. A refused
  new path in `OpenMany` no longer aborts the ordered remainder, so later
  already-open paths are selected and later new paths receive individual
  refusals.
- Normal asynchronous text adoption rechecks the duplicate path as well as
  capacity. Binary completion performs the same duplicate, admission,
  receipt-validity and post-Flush checks before releasing the old presentation.
- `addDocumentChecked` retains the origin through `Flush` and any synchronous
  callback, then performs final admission before export invalidation,
  presentation release or append. A cap refusal leaves the active document,
  source, history, selection and previous recovery bytes untouched.
- A real rich-view `RecoverDraft` callback returns the capacity error to its
  owner and leaves the draft available for another attempt. Closing one tab and
  retrying adopts the retained draft.
- Refused asynchronous binary results follow existing task/lane cleanup; they
  publish no tab and consume no closed-file receipt. Closed-file receipts are
  acknowledged only after successful adoption.
- Restore checks each record independently, so exactly100 accepted records can
  restore without a hidden welcome slot. A mixed100-record fixture verifies the
  maximum with text and binary authorities preserved in order.

The Restore loop keeps its existing synchronous owner semantics. Per the route
review decision, no worker-to-App test hook or manufactured cross-goroutine race
was added; the real asynchronous open and synchronous Flush paths carry the
post-IO/post-Flush coverage.

## Files changed by this increment

| File | Bytes | SHA256 |
|---|---:|---|
| app/app.go |38935| `0AC7FB907C83DC3C2B7642B197441492274CBB1CBC75E6688264D577D995ADB4` |
| app/binary_actions.go |10173| `D9DB7FAA2CF674ED3428F67479D6377B4901ED7C7A14C404F0536F325A9609BB` |
| app/state.go |13647| `7F3F8E338E5619241FA3D0ACC72BA791778F1CE2A328272CBAED12E40B9400FD` |
| app/live_tab_admission_test.go |20902| `285244AD5A24FFB927F57CDA28A8AD2F5CDB7F09FE7A52960EDDBD74BAA8766D` |

## Focused coverage

The final admission group contains nine top-level tests covering:

- 99→100 successful creation and100→101 refusal for template, plugin and
  recovery origins, without template/next-ID/Flush/release or session-byte
  side effects;
- mixed capped `OpenMany`, including explicit refusal continuation and later
  duplicate selection;
- asynchronous text capacity change and a late normal-request duplicate;
- synchronous tab creation during text and binary `Flush`;
- a real Markdown conflict view whose retained draft is refused twice at100,
  then recovered successfully after close-one;
- 99→100 byte fallback, exact closed-file receipt retention and close-one/retry;
- closed receipt reuse of an already-open path at100;
- asynchronous binary capacity change, late duplicate and post-Flush refusal,
  including task-slot and lane cleanup;
- successful restore of a mixed100-record session with exact count, order,
  active selection and text/binary authority.

Preserved execution:

- Admission group: exit0,5.540s wall, app2.420s.
- Existing affected open-many/reopen/native-image set: exit0,4.406s wall,
  app3.267s.
- Final formatting checks: exit0.

An inherited working-session report described a pre-evidence release-order
defect: `releasePresentation(old)` ran before the final capacity recheck after
`Flush`. This agent did not personally run or observe that unretained check, and
no original stdout, stderr, exit code or timing exists for it; it is reported
history, not preserved evidence. The source was corrected and the retained text
and binary post-Flush tests verify the final ordering. Later evidence 004-007 is
preserved test-construction failure history, not a remaining product failure.
Every retained command and numeric exit is indexed in
[the evidence README](live-tab-admission-evidence-20261110/README.md).

## Source freeze and limitations

[source-frozen.json](live-tab-admission-evidence-20261110/source-frozen.json) is
the nine-file union requested by root: the seven accepted reopen files,
`app/state.go`, and the new admission test. It totals160,306 bytes; the manifest
is1,348 bytes, SHA256
`940C698E0B0195258649E6A0747A8F1EBAF1F754BAE05E6539B790B023098DAF`.

The seven reopen source/test identities other than the two files changed here
match the accepted reopen manifest. No core/session/schema, decoder, vendor,
plugin, format, package or product-document source changed.

This slice has no full app-suite/vet run, build, process launch, screenshot,
physical UI/IME verification or resource measurement. Those checks remain for
the one combined final validation after frozen-source review.
