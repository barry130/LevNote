# Managed real-release test — source readiness

2026-11-09. Root read final managed_bundle_test.go and actual default run/vet
under evidence/fixed-engine-package-smoke-default-20261109-065958. Both exit0;
test stdout ok0.599s, empty stderr, frozen before/after source SHA
47e0162df75f73e2146469aabf24bf5c3f976e99a18bfa55cb8c7a913fca0a90.

Source compilation is now ready. Without -v the stdout does not explicitly
print SKIP; source guard plus owner's cleared opt-in records establish the
intended default-only scope. No installed/bundled request has been executed by
this test, and neither its oracle nor release behavior is claimed passed.
Do not rerun this source gate to add verbosity. Later .9 delivery will use
explicit -v/count1 and the opt-in exact name, and preserve all actual outcomes.

The initial compile errors (unused import then premature runner closure) are
retained in Astra runtime evidence. Remaining corrections are source-read:
cleanup after declared Runner, separate explicit/fallback close fields,
cleanup JSON log survives Fatal, actual manifest length vs inventory entry,
named-error joined guard cleanup, current host presence and distinct versions.

Astra may now run its already scoped 3case runtime followup plus vet, excluding
the real smoke test. App implementation/test scope continues independently.
This acceptance is source readiness only, not product package acceptance.
