# Active stage — repeatable native portable packages

Status: COMPLETE; FINAL DOCUMENTATION ACCEPTED. Root read final docs and separate
actual-time acceptance record046AFCC5864EC871D18A812DD0429764A5D6553ECE7A16E736D236F1833D57EA.
No further execution is needed. Product's next stage is controlled separately.
Root read actual package
evidence, source/oracles and same-version byte-identical payload.19files,
17,020,368B,13copies,19links,18manifest entries; manifestA1E511BA...; preserved
11/12validator09B2A7AA... and4/4correctionE8C093B9... complete12conditions.
Only final docs and concise acceptance record remain, no tests/builds/launches.

Earlier activation: ACTIVE after root accepted the Markdown-worker integration and clean
delivery. Sole owner `/root/astra_marker_view` implements/tests; root plans and
reviews. Ownership: scripts/**, tracked portable asset templates (choose one
small directory), dependency-patches guard regressions/docs, related product
docs. Product/runtime/parser/UI/vendor and previous releases/evidence are frozen.
No child delegation. Do not resume interrupted owners.

Focused gate ACCEPTED: checkpoint8C9F5A7A40A3158511161465C30DC617EEEB6C743ADF9CBA93A5687AC70D0787,
32distinct guard cases and24asset/path checks. Root read actual source, failures,
corrected JSON oracle and streams; original failed31-case run stays unchanged.
One same-version production package is now active. Compare three payload hashes
to accepted original; new mismatch requires root review, no automatic launch.

Two concrete shortcomings were observed during the actual worker integration:
the pinned PowerShell Go guard rejects valid lowercase `go test -c` because its
`-C` match is case-insensitive, and the new package command builds the host and
worker but does not itself copy required third-party notices/licenses. Those
are reproducibility problems to fix before expanding executable-engine scope.

## Proposed finite scope

- Make the argument check distinguish legitimate lowercase test `-c` from
  uppercase Go `-C`. Preserve rejection of actual alternate directory/module
  inputs, `GOFLAGS` module bypass, absent vendor and absent startup patch. Use
  isolated fixtures; never mutate production vendor for a test.
- Add meaningful positive `test -c`/normal-build and negative guard regressions
  using the same pinned wrapper. Diagnose command-line parsing precisely; do
  not widen an opt-out or hide failures behind an acceptance-only workaround.
- Extend the explicit package script to assemble a fresh self-contained local
  launch folder: current host, fixed worker/manifest, exact license texts,
  readable portable instructions and source/patch attribution. License links
  shipped to users must resolve within the package, or be explicitly labeled
  as repository source references rather than nonexistent local files.
- Preserve no-overwrite output directory semantics and `native/bin` historical
  release. No downloader, installer, registry mutation, signature, auto-update,
  arbitrary executable plugin loader or new dependency.
- A release manifest should fingerprint the executable/engine/manifest and
  shipped notices. Build logs and test helper binaries stay in evidence, not
  the minimal user launch folder. The fixed worker-relative path must remain
  unchanged. Describe configuration location truthfully; a movable program
  folder does not imply that user configuration is stored beside the exe.

## Review and validation

This is a build/distribution follow-up, not a new editor feature. Test the
changed guard/package behavior and exact assets with isolated destinations,
including paths containing spaces. Keep original failure evidence untouched.
After focused checks, root decides the smallest integration needed for new
packaging output; do not repeat unchanged full app/runtime tests automatically.
Verify byte-identical copies when no rebuild is necessary and disclose when
new binary hashes require their own acceptance. No resource comparison claim.

## Activated acceptance details

- Send an incremental source/contract checkpoint before lengthy commands. Use
  actual literal argument arrays in child PowerShell, including paths with spaces.
- Keep user output and evidence directories distinct and non-nested. Both are
  fresh/no-overwrite; logs/build runners/package-result live only in evidence.
  Validate inputs before creating output where practical, and preserve failed
  attempts rather than removing directories. Do not follow new output through
  existing reparse points silently. Never delete or rewrite historical output.
- Explicitly test valid lowercase `test -c` emits a tester, ordinary build emits
  an executable, and uppercase `-C`/`-C=`, `-modfile`, explicit/environment
  alternate module modes, missing vendor/patch and unsafe startup still reject.
  Refresh's permitted module-operation path must remain usable. Use tiny isolated
  fixtures with the verified toolchain, never edit production vendor.
- Portable templates must avoid hardcoded claims that arbitrary future output
  is already accepted. Include current version, limits, APPDATA location, actual
  notices and exact license/source-patch copies. Fingerprint all shipped files
  except the release manifest itself; say so in its schema. No machine-specific
  source paths in the user's manifest. Copied upstream READMEs may carry source
  links if clearly labeled attribution; authored local links must resolve.
- A focused successful production package with the SAME version
  native-worker.20261109.1 is authorized after script/guard review. Compare all
  three payload hashes with accepted4B5DCECA.../DB8F02E6.../644722... to determine
  whether it is a byte-identical packaging change or a new product candidate.
  No need to rebuild/run full product tests merely for documentation/assets.
- Capture source fingerprints, actual command streams/exits and checks in a
  compact handoff native/handoffs/portable-build.md. Avoid creating redundant
  evidence layers or historical replays; retain any genuine failed attempt.
