# Final actual command gate, then wait for package activation

2026-11-09. Owner astra_marker_view. Root has read the final command main_test.go
and corrected oracles. Its16558byte file is source-accepted pending actual test
evidence. Both previous command owners are interrupted; no concurrent editor.
Read mindmap-worker-tests-review.md for resolved test problems if needed.

GLM wrote preliminary result summaries in
artifacts/acceptance/mindmap-worker-command-tests-20261109. These have no retained
raw streams or before/after inventory, despite handoff saying streams were kept.
Do NOT accept their timings/pass claim as independent evidence. Preserve them;
correct the worker handoff to label them preliminary summaries when final actual
evidence is available. Do not infer a product defect from missing evidence.

Run one actual narrow final command gate now in a fresh evidence directory:

- main.go expected7207D0BBD20F218ADEE8215ADC56F4D9919C8583FE30FA2777A4FF057F3B687E
- main_test.go expectedDDC6F1012E207CDCD1F91FFC28F787B67A250D91CEF37B17457BF02F3CB45415
- pinned scripts/go.ps1 explicit GoArguments test ./cmd/levnote-mindmap-svg
  -count=1 -v, then vet ./cmd/levnote-mindmap-svg; child invocations because
  the wrapper exits its shell. Clear GOFLAGS and LEVNOTE opt-ins.
- Direct process stdout/stderr redirects, programmatic actual exit/time metadata,
  exact runners and owned/frozen dependency before/after hashes. No huge inventory
  needed here. Tests exercise serve in-process, no production worker/build.
- If test oracles still fail, preserve the failed attempt, inspect exact source
  and narrowly correct ONLY test oracle (main only for proven product bug), then
  rerun owned command tests/vet. Report concrete correction, not fabricated red.

Publish final logs/record/source fingerprint checkpoint and revise worker handoff
truthfully. Root will read actual RUN/PASS and source. Production .6 packaging is
still inactive until root then activates the already prepared delivery plan.
No edits to other product sources, no full suite or resource/UI launch.
