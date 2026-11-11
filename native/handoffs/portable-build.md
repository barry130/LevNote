# Portable build — root accepted

Root explicitly accepted the exact package and separate E8C093B9… correction
after reading sources/oracles/actual streams and hashes. Final documentation
was then updated; accepted package files and earlier evidence remain unchanged.

2026-11-09 local. Only scripts, portable templates, guard fixtures and related
docs changed. Product/app/runtime/parser/UI/vendor remain frozen. Root accepted
the focused gate before authorizing one same-version production package.

Generated launch folder:
`native/bin/levnote native worker 20261109.1 packaged/`.
Its host, worker and engine manifest are byte-identical to the accepted worker
payload. The older marker exe, accepted manual delivery and original candidate
folder remain unchanged. No product test suite or app launch was repeated.

## Implemented behavior

- `scripts/go.ps1` treats Go flags case-sensitively: lowercase `test -c` emits
  a tester; `-C`, `-modfile`, `-overlay` and non-vendor `-mod` in explicit args
  or quoted GOFLAGS refuse. Quoting follows the pinned Go `quoted.Split` rules.
  Invocation-local `GOWORK=off` and `GOENV=off` isolate external workspace and
  persisted settings, restoring both afterward. Non-guarded `mod` remains
  usable for dependency refresh. This is build-context isolation, not a
  hostile-code/OS security boundary.
- `scripts/package-markdown-worker.ps1` requires PowerShell7 and fresh,
  distinct, non-nested OutputDirectory/EvidenceDirectory with existing ordinary
  local parents. Relative paths use the caller's PowerShell filesystem location.
  Existing ancestor reparse points refuse; this is not a concurrent hostile
  rename protection claim. Builds force windows/amd64/v1 and verify GUI PE32+.
- Output contains only the fixed payload, complete notices/licenses/attribution,
  versioned README and relative `release-manifest.json`. Build helpers, streams
  and metadata stay in evidence. The manifest lists every other shipped file
  with size/SHA256 and explicitly excludes itself. It is not a signature or an
  acceptance claim. Templates say configuration remains in APPDATA.
- Shared small helpers are in `scripts/portable-package-lib.ps1`; reusable
  authored templates are in `portable/`. Original source notices and thirteen
  license/attribution files are copied exactly; authored local links resolve.

## Actual focused evidence

`artifacts/acceptance/portable-build-20261109-021532-focused/` preserves the
original run: 26 negative guard cases and four actual compilation cases passed.
The latter emitted a lowercase `test -c` tester, normal vendor executable,
quoted useful GOFLAGS executable and executable despite invalid external
GOWORK/persisted GOENV inputs. Guard exit1/wall30.6577072s was a test-oracle
failure: it expected Go to report GOENV as `off`; the toolchain reports the
empty effective config path. No guard implementation failure occurred.

The corrected named JSON check and previously skipped unpatched `mod edit`
refresh check passed2/2, exit0/wall1.9602773s in
`portable-build-20261109-022055-guard-correction/`. The wrapper hash is identical
across both attempts. These records cover all32 distinct guard cases. The
unchanged package/path/assets fixture passed24/24, exit0/wall1.0431872s, including
changed PowerShell cwd, refusal before side effects, spaces/reparse ancestors,
all exact copies and independent manifest coverage/hash checks.

Root accepted `focused-checkpoint.json`,5,833B,
SHA256`8C9F5A7A40A3158511161465C30DC617EEEB6C743ADF9CBA93A5687AC70D0787`.
It records actual source/evidence hashes and171 unchanged frozen product files.
The only post-initial-capture edits were the named JSON fixture/optional focused
case selection and the corrected package error wording `EvidenceDirectory`.

## One production package

Actual invocation from the repository root, with literal child argument array:

```powershell
.\native\scripts\package-markdown-worker.ps1 `
  -OutputDirectory '.\native\bin\levnote native worker 20261109.1 packaged' `
  -EvidenceDirectory '.\native\artifacts\acceptance\portable build 20261109-022315 production' `
  -Version native-worker.20261109.1
```

Actual UTC18:23:15.4037914–18:23:18.5775332, exit0/wall3.1721468s.
Worker build0.9537688s and host build1.3266586s both exited0 with empty stdout/
stderr. The outer command's75-byte stdout is the output path; stderr is empty.
Captured streams and generated build helpers are retained only in evidence.

| File | Bytes | SHA256, equal to original accepted payload |
| --- | ---: | --- |
| `levnote-native.exe` | 13,613,568 | `4B5DCECA5E5F5C3D3609336563F138247942E35E0C3BE65B382B32771A80B23C` |
| `engines/markdown-export/levnote-markdown-export.exe` | 3,352,576 | `DB8F02E6E187833FB52B4FC02B076B4AC96E357FB3E554533A698D8F050F6AD0` |
| `engines/markdown-export/manifest.json` | 337 | `64472251BD737D3BB82AFB05C35F648882D2E9D8D4A57BCFD1CDE8CF457CC453` |

Output has19 files totaling17,020,368B. All13 exact source copies and19 authored
local links were independently checked. The3,411B release manifest lists18
files and excludes itself; SHA256
`A1E511BAC26E57F49103F947D4D197541480D9BD4918ECD4FA9007AA7D6976FB`.
All181 pre-build source fingerprints (171 frozen product +10 reviewed packaging/
docs inputs) match. One local repeat build does not establish cross-machine
reproducibility, resource performance or OSIME/GPU behavior.

## Verification correction and immutable records

The first `package-verification.json` is retained at32,556B,
SHA256`09B2A7AA56AD0FB9DD38A79467399A535EBA915F2A7971BA8628D85AECE7A4D9`.
It has11/12 passing checks. Its only failure hardcoded22 protected files while
the baseline intentionally captured32: marker1 + manual delivery18 + original
candidate13. The extra10 are that historical candidate's build/test evidence.
Every actual protected size/hash/UTCmtime comparison passed; no bytes changed.

Fresh metadata-only `portable-build-20261109-022549-protected-correction/`
contains explicit membership for all22 required prior payload/assets and all10
additional evidence files, rereads32 unchanged fingerprints, and confirms the
first result stayed immutable. `protected-correction.json` passes4/4,17,646B,
SHA256`E8C093B9A687FAD0EB380608FF946B6C26F115097513B7AE5909422722B1C8F8`.
Together with the first record's11 valid checks this completes12/12 package
conditions. No package/product/asset test was rerun for the correction.

Other production evidence fingerprints:

- `package-result.json`,5,090B:
  `8412E096CB4E78D0EE5B9D4F07648687B4BEE7760EC508FED63143B47F2178D7`.
- `source-before.json`,57,117B:
  `1EEF2DB179B0027FD807BADAABA34D051F2C357613133BE64CD5E3A26BB6BA2B`.
- `payload-comparison.json`,897B:
  `3EC1CE796754706F31C8A0A819B0B3791B56536F0DA82AF070623446691AADFD`.

Final related documentation: root README/ROADMAP/CHANGELOG and native README/
VALIDATION now describe automated complete packaging, the same accepted payload,
32guard/24asset checks and both preserved oracle corrections. A separate
actual-time `root-acceptance.json` in production evidence records acceptance
against hashes computed from actual files. Revised docs were not recopied into
either accepted running folder. The Markdown allocation next stage remains
queued until separate root activation; no implementation from it is included.
