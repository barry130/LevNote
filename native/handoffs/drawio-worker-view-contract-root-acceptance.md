# Prepared view contract: root acceptance for app integration

ACCEPTED by root for Stage2 integration, 2026-11-10, with frozen reentry correction
and warning followup. Independent review is finishing separately and remains a
required pre-package gate; any concrete finding will be routed before delivery.
App work is disjoint from this accepted cache/contract slice and may now proceed.

Root read actual neutral editor contract, graphview preparation/page setter,
Release/draft/synchronization paths, original tests and both correction tests.
Preparation carries exact copied page index+ID and monotonic token; source/group/
draft gates, cache hit stability, pageA->B->A, reparse and reversible Release
semantics are implemented. The token getter has no synchronization or callbacks.

Original root finding was a Report callback setting host.Buffer=nil before the
outer sync's String call, so a later Prepare check could not prevent a panic.
Final sync now checks ownership immediately after callback and before reading
or publishing stale state. Parse-error state is set before callback. Subsequent
root followup restored visible canceled-gesture warning after parse warning-list
assignment, without an unconditional deferred write to callback-owned state.

Root read actual raw final graphview test12.354s/exit0, empty stderr and vet
streams/exit0. Original graphview+editor test9.718s/2.174s and first correction
graphview10.581s are retained prior evidence; final changed graphview tested
again only due the concrete warning regression. Editor contract unchanged.
No root tests/builds/launches, no passing gates repeated just for review.

Root independently checked all6 unchanged contract files against original result
hash/length table and final2 changed source hashes. Final sync.go:
F09DB5207C96724CC2DEB936D989C733543AD538E7BB5FC0E9C6549CBE1F1A10.
Final export_request_test.go:
D992C5B856358333A3BEA6C2691564B68C601D2A37B3BAA11B91FCA60E3E95B7.
Original result remains75EB1B45B6EF59E7A7785697E7B3BA09A6FCCB2B07CAFC2E45647B39546879AF;
reentry correction remainsDD33860EDA14B265128EA51FAC8DC332099D96E2B527D2D6309D577BCD982B33;
warning followup result0014C9446EE1E7C8B61CF4914499EF7DF73B0E825723B1F2EB60F0E10D3A28DC.

No adapter advertisement/app source-mode/post-dialog policy or actual engine
execution is proven by this slice. Stage2 must implement those gates around
existing contract and preserve unchanged old3 behavior. No physical UI/IME,
whole-process memory or copyright guarantee. `.18` release remains unchanged.

## Independent gate completed

GLM final independent review now received and read by root:
`drawio-worker-view-contract-independent-review.md`, SHA256
5DB4A8E03AF7D7BCE67DFA1295D05BE97284DAC209B195F996082F3A29826C72.
Both concrete findings reconciled, all8final files matched, no remaining blocker.
No test/build/launch rerun. This closes the pending independent view gate noted
above; Stage2 app and actual package delivery gates remain.
