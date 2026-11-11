# Managed runtime — root acceptance

2026-11-09. Root read actual implementation/tests, first failed run and final
followup04 raw streams/wrapper/result. Runtime source is accepted for app and
later delivery. No production runtime change occurred to fix the test oracle.

Production: NewManaged performs no filesystem verification at construction;
Export captures source/digest/version/private roots and load failure at admission.
Exact installed package is reverified with its whole held owner retained through
process and pipe cleanup. Later selection changes affect only later requests.
Source and PackageDigest are separate from engine ID/version/executable SHA.
Selected missing/corrupt bytes refuse without bundled fallback; cleanup failures
remain sticky through Close. Original bundled construction remains supported.

Actual evidence:
- runtime-20261109-135024-attempt01: seven new Managed cases plus relevant
  existing bundle/admission checks; four new cases and existing checks passed,
  three failed only because Windows closed native directory File.Stat reports
  ERROR_INVALID_HANDLE instead of wrapping os.ErrClosed. Helper PID20260 joined.
  Reparse test SKIP (privilege) and resource test default SKIP; no resource run.
- runtime-20261109-135343-followup02 and140012-followup03: compile failures from
  concurrent GLM smoke-test unused import / premature runner reference retained;
  no runtime checks executed in those invocations.
- runtime-20261109-151125-followup04: exact three affected cases PASS, package
  1.970s / command3.3083456s, scoped vet exit0/1.0304439s,28 inputs unchanged.
  Same owned File object's invalid Fd + failed Stat and postclose successful
  rename replace the incorrect error-class assertion. Helper PID18980 joined.

Thus seven distinct managed behavior cases are covered across original/final
gates; only one helper in each actual runtime test invocation (two total), not
the older production-release smoke. No package/application build, physical UI,
IME, GPU or memory/resource claim. Original failure evidence stays immutable.

Next: finite app operation/lifecycle tests+760 captures, then .9 build with a
separate actual old-release import/activate/reset smoke. Do not repeat unchanged
runtime/store/protocol matrices merely because a release is packaged.
