# Root acceptance — schema3 core and composition

2026-11-09. Root read actual product source/API, new strict/lifecycle/composition
tests, original syntax-failure excerpt, final raw test output and result record.
The scoped core gate is accepted. App continuation is already authorized in
`format-routing-app-activation.md`; no additional idle gate is required.

Actual final commands: `test ./core/plugins ./modules -count=1 -v` and
`vet ./core/plugins ./modules`. Plugins PASS2.463s, modules PASS0.905s; both
commands exit0, empty stderr,14 recorded inputs unchanged. Evidence directory:
`artifacts/acceptance/format-routing-core-20261109-093553/`.
Result SHA256 `554790CEE2B01AB32CDFEC033170BC2E791D398A5609B2BB21F3B03CCBCDE0B3`.

The original missing test-loop brace and preliminary tool output are disclosed
as an excerpt rather than raw redirected evidence. Final tests include old
schema1/2 behavior and exercise actual install/persistence/rollback plus fresh
native adapter capability ownership, not just DTO construction. The earlier
source review remains the API basis. No arbitrary engine installation, app
workflow or shipped schema3 binary is claimed by this core acceptance.
