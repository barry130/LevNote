# Fourth worker .19 delivery

ACTIVE 2026-11-10 after root accepted stage1+view contract+stage2 integration
and portable docs. Owner assigned in native/AGENTS.md. Root accepts actual
binary. Existing packages/evidence immutable. All product source/tests/scripts/
assets frozen for this run; only fresh evidence/harness/handoff edits permitted.

Activation inputs (root reviewed): use the corrected preparation directory
`handoffs/drawio-worker-release19-preparation-corrected/`, manifest SHA256
`3F50584200F28D62B9EBC6CFD5589309C83D2EF575321DDF433F80E08611F8CB`.
The automatic-variable collisions are resolved; both Stage2 evidence directories
are protected. FinalStage2FilesManifest is the absolute path to
`handoffs/drawio-worker-stage2-files.json`; FinalStage2EvidenceDirectory is
`handoffs/drawio-worker-stage2-test-followup-evidence`. FinalStage2GatePaths must
include the absolute paths to `drawio-worker-stage2-root-acceptance.md`,
`drawio-worker-stage2-result.md`, and the harness correction result in handoffs.
Use a fresh timestamped directory below native/artifacts/acceptance. Initialize
with ExecuteDelivery, then run baseline, package, managed-smoke, host-smoke and
aggregate sequentially from the copied harness. Publish progress at meaningful
stage changes, final delivery handoff, then freeze.

No more preparation gate is needed. If only evidence/harness metadata fails,
preserve the failure and correct the evidence-local copy narrowly without
repeating passed builds, managed worker or host runs. If a product operation or
source check fails, stop that delivery and report the concrete issue to root.

One existing package-markdown-worker.ps1 invocation, now FIVE Windows amd64 GUI
builds (host+4), output new absent native/bin/levnote-native-worker-20261110.19
at version native-worker.20261110.19. Fresh timestamped acceptance directory.
Use actual file totals/identities (expect25files but verify), manifest entries,
four engine manifests and notices/assets/local links; no copied-assumed counts.
Protect complete .18 package and direct delivery evidence, accepted new source/
test/capture/review gates and scoped source/build inputs. No toolchain or entire
history inventories. No repeat source suites/vet/benchmarks/captures.

Unlike .18, .19 requires actual fourth-engine execution. Prepare one small
integration tester in an evidence-owned complete bundle COPY, never modify
candidate. Use existing runtime package test/harness patterns. Existing private
observational hooks (afterStart/afterWait) may record exact handles and join;
never override executable, wire response, arguments or lifecycle. Create isolated
run-owned config/store, load a valid legacy schema1
state with old3 selections (fixtures may be bundled; installed migration has
unit coverage), confirm fourth bundled without implicit persistence. Import
fourth engine from complete accepted-candidate release, explicitly activate and
make exactly ONE actual drawio request through NewManaged/Runner.Export. Other
engines must not run; selection of old3 remains intact.

Prepared fixture is `core/engineruntime/drawio_managed_bundle_test.go`, reviewed
through `drawio-worker-real-smoke-preparation-result.md` and later compile result.
Use exactly TestDrawioManagedRealBundleSmoke and its explicit DRAWIO env variables
for one enabled execution; do not write a second fixture. Its no-op/default-Skip
compile may already have passed but test -c is needed to make copied tester.
Record actual final test hash. Ordinary old smoke env remains disabled. This
enabled execution occurs during candidate delivery BEFORE root binary acceptance,
after accepted source/portable stages; the prepared result's 'after .19 acceptance'
wording is superseded by this schedule. No run until root explicitly activates
this delivery task in native/AGENTS.md.

Fixture: tiny full original XML with two distinct page IDs/labels, request page1
zero-based plus exact ID. Compare final payload against an independent frozen
literal SVG/page oracle from accepted source tests, not calling same exporter to
generate expected output during smoke. Validate warning metadata/order according
to fixture. Input bytes/hash unchanged; verified EngineInfo installed/digest/
version/hash matches imported candidate. Retain actual source/response hashes,
owned process handle/path/creation identity and one Start/Wait/Close proof.
Use existing safe managed lifetime/Job/cleanup path. No process-name kills.
No claim this bypasses OS sandbox or constitutes UI export behavior.

One exact copied hidden host startup/owned WM_CLOSE with run-owned C: config
and unchanged generated Markdown source, as .18. This is separate from managed
worker test and establishes only startup/clean close. Start-Process Hidden.
No physical OS helper retry, invented UI automation, memory/CPU claim. Resource
measurements require a separately defined workload, not this smoke.

Every execution/stream/exit preserved. If test/build fails, preserve attempt and
stop delivery to route actual source defect; don't patch candidate or silently
repeat all gates. Metadata-only fixes may rerun only failing aggregation step.

Publish drawio-worker-release19-delivery.md candidate-ready with actual counts,
hashes, commands, validator/aggregate, scope of new real worker evidence and
inherited old3 evidence. Root binary acceptance before six-doc .19 update.
