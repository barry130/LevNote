# Markdown worker full integration — queued acceptance plan

Status: ACCEPTED2026-11-09 01:55local. Root accepted aggregate
FB0468DADF9F526AFD7BAE72BD1E3893CAD7CEE78176870DDDAE5BB59B12A83A and
validator C32CB352659ED708B05A3A0D4E4D4DA01F33523C3D512FA53D0F3381A5040154
with20/20checks. Execute only final docs/launch asset copy now; preserve all
accepted artifacts and record acceptance separately. No more product execution.

Historical activation2026-11-09 01:32local: Root reviewed current app source and
meaningful oracles, actual focused PASS1.773s/app-wide PASS5.411s, scoped
vet1.525383s exit0, all four760x480 native images and final opt-in bundle test
source. The opt-in test changes after app-wide testing compiled in vet; actual
production execution remains part of this stage. Root now authorizes this
finite plan. Product/tests are frozen. Root does no tests,
builds, launches or implementation. Sole eventual integrator is
`/root/astra_marker_view`; no child delegation or previous-owner resumption.

The controlling contract is `markdown-worker-contract.md`. Bottom-layer
checkpoint EAA53F06003BD871D442178F7781CB5AE2425269ED8C2E273E0BF1E7BFCBEB50
is accepted with its known file-symlink SKIP. The protected previous executable
is `native/bin/levnote-native.exe`, 13,461,504 bytes,
SHA256 17E014EB6F61E671F6FCCA439719E8831B7591042A6F29EE5354960C120C0466.
Preserve it, earlier release artifacts, user configuration and user documents.

## Execution after explicit activation

1. Freeze product and tests, capture source fingerprints including app,
   protocol, runtime, worker, contract, Markdown descriptor and packaging input.
   Verify the bottom-layer checkpoint files stayed unchanged. Use one new
   timestamped acceptance directory per attempt and preserve all failed output.
2. Run the full native suite once (`test ./... -count=1 -timeout 300s`) and full
   `vet ./...` using the pinned wrapper. Record actual command text, times,
   exits and stdout/stderr. A failed assertion requires a concrete diagnosis
   and a new evidence directory, not a weaker oracle or overwritten log.
3. Execute the checked-in packaging script to a fresh `bundle` directory,
   version `native-worker.20261109.1`. Build BOTH current host and worker;
   never substitute the accepted older host. Record their size/hash/mtime,
   Windows amd64 GUI headers and generated manifest hash. Do not replace bin.
4. Compile the opt-in app test executable beside this host/engines. Run only
   the planned bundled acceptance test with real `NewBundled`, real fixed
   worker and actual registered Markdown adapter. The injected save dialog is
   a test seam, so describe this as host-path application integration, not a
   physical OS dialog or keyboard test. Capture commands, streams and result.
5. Verify complete HTML plus meaningful independent fixture expectations and
   warning identity/ranges; exact original source bytes, encoding, baseline,
   selection, history and dirty state must survive. Include source/no-view
   routing and rejection of an existing export destination. Confirm cleanup
   before publication through the established runtime return guarantee;
   report actual observed process evidence without claiming an unobserved PID.
6. Verify missing or altered bundle refusal only in a separate disposable
   evidence copy. Do not corrupt the candidate bundle or invoke arbitrary
   installed executables. No fallback and no target creation on failure.
7. Run the actual packaged host with isolated configuration and read-only
   fixtures. Launch hidden, record exact owned PID/path/start, then WM_CLOSE
   that owned window and wait for its exit. Preserve an unrelated sentinel.
   No process-name kills, no visible helper windows, no broad cleanup command.
   Hidden startup is not real GPU/OS IME/interactive-dialog acceptance.
8. Record an immutable aggregate with exact evidence fingerprints and actual
   pass/fail/skip observations. Root will read it and record acceptance
   separately; do not rewrite an accepted record to change its status.

No new resource matrix or claimed memory improvement is part of this run.
No packaging/distribution signature, publisher authentication, sandbox or
copyright immunity is established. The current HTML renderer constructs full
output before the 64 MiB refusal; final atomic publication is synchronous on
the UI thread and cannot be canceled after it starts.

## Documentation after actual integration evidence

Update native README/VALIDATION/ARCHITECTURE, format/plugin API and relevant
root README/ROADMAP/CHANGELOG. The working run command must point to the newly
accepted bundled host, or explain ordinary host-only builds visibly refuse
Markdown HTML export when adjacent worker files are absent. Keep local JSON
schema1/2 plugins distinct from this one repository-owned executable engine.
Native editing/preview stay in the host; arbitrary executable plugin installation
and rich-view IPC are not implemented. Reference actual validation and known
limits, retain historic failures and keep unmeasured Tauri comparisons pending.
