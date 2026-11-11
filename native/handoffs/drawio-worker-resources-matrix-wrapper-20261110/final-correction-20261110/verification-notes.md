# Fixture verification notes

PowerShell7.6.9, fixtures-01 exit0, fixture-results.json records128/128 PASS.
All Start-Process calls were intercepted by a fixture function; no real tester,
worker, host, Go test/build or preflight was executed. Accepted tester/preflight
and package files were only read. Synthetic records live beneath fixtures-01,
with FIXTURE-ONLY.txt, mock stdout/stderr, synthetic measurement FixtureOnly,
and run/summary FixtureOnly labels. Original run JSON before labeling is kept.

Expected error messages in stderr exercise malformed measurement JSON, existing
output, wrong cwd and protected-package output. They are asserted failure cases,
not failed verification. Start/wait/timeout/cleanup failures remain in each
synthetic run record with nonzero wrapper exit and package-after inventory.

The fixture runner's final Select-Object projection on an OrderedDictionary
printed four null fields in stdout, although the saved result JSON was correct.
After inspecting that JSON, only this display line was corrected to cast to
PSCustomObject. No product/wrapper behavior changed; no verification rerun was
needed. All original stdout/stderr/exit records remain untouched.

Prior GLM scripts/preparation are preserved as *.before.* here; all earlier
parse and fixture attempts remain in their original locations. Final wrapper
source identity is recorded separately by final-parse.json.
