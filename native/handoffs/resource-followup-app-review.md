# Resource follow-up app plan — read-only review

Reviewer: GLM formats agent. This is a planning review of the current
`handoffs/resource-followup-app.md` against `resource_followup_stage_plan.txt`.
No benchmark, process launch, or product change was performed.

## Verified corrections

The current plan incorporates the root's corrections:

- Extended blank measurement stays on the same PID, with an immediate extension
  baseline and ten observations after three-second intervals. It is correctly
  not described as a restarted long-idle baseline.
- Memory observations and CPU windows are separated. CPU deltas use actual
  baseline/end timestamps and cumulative process CPU, not assumed sample
  cadence.
- Markdown fixture validation goes through `core/document.Open`, records raw
  bytes and BOM status separately, and validates decoded parser text. This is
  the application loader path rather than feeding BOM bytes directly to the
  parser.
- The plan explicitly uses ten finite measurement intervals per scenario; the
  earlier proposal's longer per-format sampling window is no longer a conflict.
- Pre-existing `levnote-native` processes are enumerated before launch, the
  checkpoint aborts if any exist, and unrelated processes are not stopped.
- Hidden-window scope, isolated app data, fixed executable hash, parser-backed
  fixture validation, raw main-process metrics, no Tauri comparison, no forced
  GC/trimming, and no product hooks all remain correctly bounded.

## Final plan verification

All previously identified gaps are now incorporated:

1. **Bounded waits and failed-checkpoint states are defined.** Window
   discovery, session-file validation, graceful-close, and each scenario should
   emit a recorded failed checkpoint on timeout rather than wait indefinitely
   or publish a partial sample set. The plan now defines 20 seconds for
   window discovery and session write/validation, 15 seconds for `WM_CLOSE`,
   and 5 seconds for sampler-owned cleanup, with explicit status values.
   A partial set remains diagnostic raw data and cannot become a completed
   result.

2. **Session validation uses the product loader.** `session.json` is validated
   through current `core/session.Load`, with expected/actual tab order, active
   index, mode, validator path, and load-error status recorded. A replacement
   schema-specific loader is not used.

3. **Fixture integrity is enforced without changing the workload.** Fixtures
   remain normally writable; no read-only attributes or ACL denials are
   applied because those would alter the application's readonly-workload
   conditions. Instead, the sampler hashes every fixture immediately before the
   matrix and again after every process exits; any change or missing fixture
   is an explicit `failed-fixture-immutability` checkpoint even if all
   process samples were collected. The earlier recommendation to make the
   fixture directory read-only is superseded.

No additional scenario, instrumentation, benchmark run, or product hook is
requested. The resource stage can proceed to generated fixtures, validator,
sampler, and final evidence review when those files exist.
