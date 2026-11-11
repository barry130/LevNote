# .19 delivery harness preparation result

Status: **PREPARED AND FROZEN; DELIVERY NOT AUTHORIZED OR EXECUTED**

Date: 2026-11-10

The fresh `.19` delivery harness is in
`native/handoffs/drawio-worker-release19-preparation/`. It adapts the accepted
`.18` flow without editing or executing any accepted package or evidence tree.
No `.19` candidate, build, test executable, test, worker, host, capture, or
aggregate delivery evidence was created or run during preparation.

`native/bin/levnote-native-worker-20261110.19` is absent. The only preparation
execution was the permitted static PowerShell parser check described below.

## Frozen harness

The directory has 15 files: ten PowerShell files, `README.md`, three
`evidence/static-parse.*` files, and `harness-manifest.sha256`. The manifest
covers the other 14 files. Its parsed entries match the current bytes:

- `harness-manifest.sha256`: 1,203 bytes, SHA256
  `6606541EF2165A7F97C9C4B400E2D5F579D23018609EB8D7047A9E9089D51CA3`
- Manifest entries: 14
- Hash mismatches: 0

| Prepared file | Bytes | SHA256 |
|---|---:|---|
| `aggregate.ps1` | 21,076 | `690E11BE57446BDDD9B4B0F6DD7C293F22C0A43B7D2D0D4C9230782993F69420` |
| `baseline.ps1` | 6,395 | `7338B49097E0E192A6D483C12EFEE3BAAE81177D74F3FAFFDDFB48701EC13E7F` |
| `delivery-lib.ps1` | 10,790 | `096EE7C24DC87D1A0C9190B7D5C5D12CF97AEC1A36A0399AC77AA8B40949F44C` |
| `initialize-context.ps1` | 10,242 | `38DB3BE12516BA10AD42A29C80EBA5FF18B55D940516766AE729FF8239F63AFF` |
| `invoke-stage.ps1` | 1,700 | `B791685AC92FE8225F1780705E9B15492601186CD833C4A85A6252D7C98914FB` |
| `run-host-smoke.ps1` | 12,763 | `FF6E3178371327D2A5567DDC06C0CF67F1C4AA36F1E569AE4121068F0C18A214` |
| `run-managed-smoke.ps1` | 12,341 | `6FA6671DADC8BB1941DB433EE17A20CB94FEA1F93EC7CE1E1B73D7106A00B614` |
| `run-package.ps1` | 10,924 | `088B4F03F509A51E07BCD774424E558A46B5215F01E3015A2FB27CAC3C6E3DFB` |
| `run-stage.ps1` | 2,062 | `C619C043ED4612572BCB4AE87165CC321188C856C1EFC599C70442B3881BBDAD` |
| `source-scope.ps1` | 1,599 | `95E9EE57648CDC88815856FD77C5C9D17F5526036FF0A270063DD2009C5B1AC7` |

`README.md` and the static evidence hashes are also pinned by the harness
manifest.

## Static parse evidence

The final permitted syntax-only run parsed all ten PowerShell files and reported
zero parse errors for each. No script entry point, package/build command, Go
command, test, worker, host, or capture was invoked.

- `evidence/static-parse.stdout.log`: 946 bytes SHA256
  `1612BAC1062FC824791B9C3E7F4D5C387CFB70796CA9889198497697FDD87CE4`
- `evidence/static-parse.stderr.log`: 0 bytes SHA256
  `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`
- `evidence/static-parse.exit.txt`: `0` SHA256
  `5FECEB66FFC86F38D952786C6D696C79C2DBC239DD4E91B46729D73A27FB57E9`

One earlier shell setup attempt stopped before parsing because it attempted to
create the already-existing local `evidence` directory. It performed no product
or delivery execution. The final run above is the actual syntax evidence.

## Activation contract

Root activation must create the separate task
`drawio-worker-release19-delivery-task.md` and freeze the final Stage 2 source,
file manifest, evidence directory, and at least one Stage 2 gate. The exact
activation shape is:

1. Run the prepared `initialize-context.ps1` with
   `-CandidateDirectory F:\AIproject\LevNote\native\bin\levnote-native-worker-20261110.19`,
   a fresh absolute `-EvidenceDirectory`,
   `-FinalStage2FilesManifest`, `-FinalStage2EvidenceDirectory`,
   `-FinalStage2GatePaths`, `-Version native-worker.20261110.19`, and
   `-ExecuteDelivery`.
2. From the evidence-local copied harness, run
   `run-stage.ps1 -Name baseline`.
3. Run `run-stage.ps1 -Name package`.
4. Run `run-stage.ps1 -Name managed-smoke`.
5. Run `run-stage.ps1 -Name host-smoke`.
6. Run `run-stage.ps1 -Name aggregate`.

Initialization rejects an existing candidate or evidence root, a wrong version
or candidate leaf, an incomplete Stage 2 context, and any Stage 2 manifest path
outside the scoped inputs or differing from the current file. It copies the
frozen harness into the fresh evidence root and writes `context.json` and
`commands.json`; later stages execute from that copy. Each stage runs in a
hidden PowerShell process and preserves raw stdout, stderr, exit, timing, and
invocation metadata.

## Package and managed-worker gates

The package stage invokes
`native/scripts/package-markdown-worker.ps1` exactly once. It requires five
successful GUI builds with empty streams: host, Markdown export, mindmap SVG,
mindmap preview, and draw.io SVG. It validates five Windows amd64 GUI PE
executables and the actual candidate against the expected 25-file artifact set:
the host, release manifest, four fixed engine executables plus four engine
manifests, portable assets, and portable notices.

The release manifest must be schema `1`, version
`native-worker.20261110.19`, `windows/amd64/v1`, contain 24 payload entries with
matching sizes and SHA256 values, exclude only itself, and retain local authored
links and byte-identical portable source copies. Every fixed engine manifest
must match its worker identity. Schema `2` remains confined to package-selection
state and is not substituted for this release-manifest check. Actual counts and
hashes are recorded at delivery; 25 files is a fail-closed assertion awaiting
inspection, not an inherited claim.

The managed stage copies the complete candidate to a separate run root and
compiles `./core/engineruntime` once through `native/scripts/go.ps1`. It runs
exactly `^TestDrawioManagedRealBundleSmoke$` once with the candidate, evidence,
and isolated config roots supplied through the four
`LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE*` variables, restoring the prior process
environment afterward.

The reviewed fixture is frozen at 21,150 bytes and SHA256
`074C4894D1D05F0FEBE7B49F61BF958980BDDF6DFEF9151179A18C05A2FF4548`.
The stage requires the structured and logged records to agree: one
`NewManaged`, one export, admission/verify/start/wait counts of one, page index
`1` with page ID `second`, the exact two ordered warning messages and source
spans, unchanged source and candidate identities, exact process/image/creation
facts, one joined lifecycle, a normal worker exit, and no fallback close. The
bundle copy may add only `engineruntime.test.exe`; the candidate and payload
remain exact.

## Separate hidden-host gate

The host stage copies the complete candidate and uses a fresh isolated
`APPDATA`/`LOCALAPPDATA` tree on the normal C: temp volume. It launches the
copied `levnote-native.exe` once with `Start-Process -WindowStyle Hidden`,
records one exact PID/path/start identity, finds the exact owned top-level
window, posts exactly one `WM_CLOSE`, and requires normal exit `0`.

After close it requires one clean session record for the unchanged Markdown
source, exactly one tab, no draft, and no terminal LF or CR. This is a lifecycle
check separate from the fixed managed draw.io worker request. It does not claim
physical UI, IME, dirty Save, interactive SVG, process memory, or CPU coverage.

## Protection and aggregation

The stage creates scoped before/after inventories and hashes, then fails closed
unless all of the following remain exact:

- `.18` package: 23 files / 23,699,526 bytes; inventory identity
  `4A4F9BD0B6DF2DDCD1E597F026AAC985A9D50711CDD85F3E9310F7DC5B0B2991`;
  host SHA256
  `BFA6F460F49087390F114CEC2CF2ADC034307B07769D8B64E3E70C78BBDCB44A`;
  release-manifest SHA256
  `8CA2016C0064638BE965FDFE74468F8004A37143BE568A189FD7FEDAB2D12FDD`.
- `.18` direct evidence: 117 files / 47,816,959 bytes; inventory identity
  `8D73702A59E34583FB77E567AF7167A16A06698F751A16D623BDD469EE66C948`;
  `aggregate.json` SHA256
  `D1066FA721005A7F4DC97184BC8E1E802B8FE86013609B5A2F64975D81E14FF0`.
- Scoped current source and package inputs under `native/app`, `native/core`,
  `native/formats`, `native/modules`, `native/cmd`, `native/ui`, portable assets,
  Go module files, portable docs/notice preamble, and the three package/Go
  scripts.
- The seven accepted Stage 1/correction/view/smoke/portable evidence
  directories, final Stage 2 evidence, the fifteen fixed accepted gate files,
  and the activation-supplied Stage 2 gate file(s).
- The final activation-time Stage 2 file manifest and every later rehash.
- The candidate, its complete package copy, managed bundle copy plus tester,
  raw stage records, and hidden-host source/session evidence.

The aggregate writes `validator.json`, `aggregate.json`, and its final evidence
members to the fresh evidence root only after the package, managed, host,
protected-input, and Stage 2 checks all pass.

## Deferred gates and assumptions

Delivery remains deferred until root supplies the accepted final Stage 2 file
manifest, evidence directory, and gate file(s), and separately activates
`drawio-worker-release19-delivery-task.md`. No half-implemented Stage 2 file or
hash is recorded here.

At activation, the Stage 2 files must still match the supplied manifest; the
reviewed smoke fixture must retain the pinned identity; the accepted `.18`
package/evidence and accepted Stage 1/portable gates must remain unchanged; and
the `.19` candidate/evidence roots must still be absent. Actual candidate counts,
manifest identities, build results, managed-worker result, host lifecycle, and
aggregate outcome are deferred and may not be inferred from this preparation.

The prepared harness is frozen after publication. This result does not
authorize any package, build, Go compile, test, worker, host, capture, resource
measurement, or `.19` output execution.
