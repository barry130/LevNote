# Root acceptance — optional readonly scene refresh

2026-11-09. ACCEPTED source slice. Root read actual optional SceneRefresher,
RefreshScene implementation, all5 new test bodies and actual stdout/results
under artifacts/acceptance/fixed-engine-package-scene-refresh-20261109-124417-
attempt01. Focused package PASS1.189s, command4.0259231s/exit0, graphview/editor
vet1.5977629s/exit0 and empty streams,13 scoped inputs unchanged. Wrapper
explicitly exits LASTEXITCODE. ResultSHA256
`626AB085ED7F23F5F55DAE09C250ECA54221BECDB0594AB38E12D98EA1622835`.

Refresh revokes epoch and clears transient state before invoking captured
cancel; no writes after that external callback. Requests remain lazy and
coalesced at unchanged source. Tests cover clean/dirty document, actual drawn
cache and held-pan cancellation, old sync/late callbacks, error/stale navigation,
Release reentrancy, replacement missing selection and optional readonly scope.
No actionable source review finding. Writable views remain untouched.

App/runtime wiring is not yet implemented; .8 delivery unchanged. No new real
worker/UI/IME/resource evidence. This source acceptance activates Astra's store
slice under fixed-engine-package-store-astra-activation.md, separate from GLM
pure inventory decoder. No additional scene checks needed without new changes.
