# Hidden host session-readiness feasibility

ACTIVE read-only 2026-11-10 for Astra. No source edits or execution; .23 inputs
remain frozen. Give a short concrete verdict in
native-image-hidden-readiness-review.md, not another broad review.

Root found a potential first-host readiness issue while reviewing actual MyGo:
internal/windows/surface.go RequestFrame at275 only posts drawHidden before
`s.drawn`; comments say hidden windows do not get WM_PAINT. ui/engine.changed
requests a frame rather than rebuilding synchronously. App View owns
persistWhenIdle; new binary open complete marks persistPending but does not call
Persist. Our .23 hidden host helper polls for mixed session before its one close.
CLI open does updateWait(deliverBinary), so UI adoption itself is independent of
draw, but automatic session may not update after the first hidden frame.

Trace these actual methods and accepted .22 host evidence. Determine whether
the planned hidden poll can progress without depending on a frame that Windows
will never request. No speculative product patch or homemade paint/input
automation. If blocked, recommend the smallest honest verification adjustment
using accepted hidden-host lifecycle tools (bounded delay/one close/fresh
post-close publication is allowed if no reliable pre-close session exists),
or identify a real product persistence defect for root decision. Distinguish
first run's empty config from run2's existing session. Preserve truthful scope:
hidden startup/close checks, not decode/pixels or physical input.
