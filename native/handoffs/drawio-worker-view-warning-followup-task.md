# View synchronization: preserve canceled-gesture warning

ACTIVE 2026-11-10 for same DeepSeek view owner. Root read completed reentry
correction and confirms immediate owner guard. Preserve its frozen result/logs.

One concrete regression remains in sync.go: old code deferred
addWarning(canceledGestureError). Correction adds it immediately before
`v.warnings = drawioWarnings(issues)` or `mindmapWarnings(issues)`, so a normal
successful reparse overwrites the visible canceled-gesture warning. Report still
records an issue, but graph warning panel loses the prior supported warning.

Keep the immediate ownership guard; ensure the cancellation warning is appended
to the completed same-owner warning list only when doing so cannot overwrite or
mutate callback-owned released/retargeted state. Don't restore an unconditional
defer writing to a changed owner. Can retain one local warning and publish at a
safe same-owner point. Cover a normal stale-source active-gesture reparse keeps
warning; existing reentry cases prove early returns don't resurrect stale state.
Tiny sync.go/test repair only. No new contract/app/advertisement. One final
graphview test/vet, new evidence/output
`drawio-worker-view-warning-followup-result.md`, original artifacts immutable.
