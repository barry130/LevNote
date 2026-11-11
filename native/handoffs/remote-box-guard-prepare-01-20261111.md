# Remote guard prepare 01 receipt and root review handoff (2026-11-11)

Status: executed, not a product run. Wrapper phase `prepare` ran once on
the authorized main box and returned GUARD_READY_NOT_PRODUCT_ACCEPTANCE
with wrapper exit 0 (09:02:23Z to 09:02:47Z). No Go command was executed
and ProductExecuted is false. App test/vet and .30 remain held for root
review of the real remote result.

## Delivery route and byte-identity repair

Source reached the box exclusively through git: local commit, CNB push,
box clone. A fresh inbox /home/user/levnote-inbox-20261111-01 received
the three pinned preparation-02 files from the clone plus the official
go1.27.3.linux-amd64.tar.gz; all four were SHA256-verified against the
pinned identities before prepare:

- levnote-source-inputs-20261111-02.zip, 5,227,577 B, 457503deecbcf0a534cc3240027048454bdf758076e0ee8d3a0a64f7773db7f9
- inputs-02.json, 244,715 B, a51734c0e9b40051ad1ba06ad4b5e0acc4d92ef1a3de5261259fa435b5df2e51
- linux-guarded-validation-02.py, 33,772 B, 7d53c6b7374a52c9d3e6749e423449d073778ac1c33c49888a121b97d43e17d7
- go1.27.3.linux-amd64.tar.gz, 70,588,678 B, 65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40

One delivery defect was found and fixed before the run. The repository
rule `* text=auto eol=lf` had normalized the pinned inputs-02.json
(244,715 bytes, mixed line endings) into a 238,674-byte LF blob, so the
first box checkout failed the pinned identity check. Fix commit
12b420b1e953f48b4d8cf2bc96eb59f5df0d7a8f added `-text` exemptions for
native/handoffs/** and native/formats/mindmap/testdata/** and
renormalized 814 files to their working-tree bytes. After the fix the
pinned file checked out byte-identically and the guard accepted it. The
same commit is pushed to both remotes (CNB and GitHub).

## Caller capture (external wrapper streams)

- UTC before: 2026-11-11T09:02:23Z; UTC after: 2026-11-11T09:02:47Z.
- Argv: python3 -I -B /home/user/levnote-inbox-20261111-01/linux-guarded-validation-02.py prepare --box-id <authorized main box id, recorded in evidence/prepare-01/invocation.json> --inbox /home/user/levnote-inbox-20261111-01 --run-root /home/user/levnote-remote-20261111-01
- Wrapper exit 0. stdout: one JSON line (Phase prepare, Status
  GUARD_READY_NOT_PRODUCT_ACCEPTANCE, Evidence evidence/prepare-01/result.json,
  ExitCode 0). stderr: empty.
- Raw files: /home/user/levnote-remote-caller-evidence-20261111-01/
  (prepare-utc-before.txt, prepare-stdout.txt, prepare-stderr.txt,
  prepare-exit.txt, prepare-utc-after.txt).

## Guard receipt and identities

guard-ready.json at the run root /home/user/levnote-remote-20261111-01,
SHA256 da70b0421c48165f9c3db880c75cac31c59f7901f9f55de583f05b0a135e0853:
Status GUARD_READY_NOT_PRODUCT_ACCEPTANCE, ProductExecuted false,
ManifestSHA256 a51734c0... (pinned match), SourceArchiveSHA256
457503de... (pinned match), ToolchainArchiveSHA256 65efbc80... (pinned
match), Wrapper 33,772 B 7d53c6b7... (pinned match), InputIdentity
96e0a103b8538f9810333082ed05217e47b1360e963a7dbc8dbef0585654e12d.

evidence/prepare-01/result.json: InputsBefore equals InputsAfter (1,208
files, 17,045,848 bytes, 5 picker files, 26 patch entries); ToolchainBefore
equals ToolchainAfter (15,657 files, 244,139,206 bytes, go binary
17,164,222 bytes SHA256 5f5eb6d0..., VERSION go1.27.3, Executed false);
PrerequisitesBefore passed (ldconfig lookup, resolved gobject/pango
library identities under /usr/lib/x86_64-linux-gnu/, fc-match font
probes, Pango font map and context creation through ctypes);
PostGuardErrors empty; NoPhysicalUIIMEOrWindowsDeliveryClaim true.

## Evidence locations (box)

- Run root: /home/user/levnote-remote-20261111-01 with siblings source,
  toolchain, transport, state-prepare, evidence, guard-ready.json.
- evidence/prepare-01/: result.json, invocation.json, environment.json,
  inputs-before.json, inputs-after.json, toolchain-before.json,
  toolchain-after.json, prerequisites-before/ (ldconfig, font-1..font-3,
  library-identities-before-load.json, prerequisites.json).

## Holds and the next step for root

Preparation success means neither app tests nor Windows delivery have
passed. Per remote-runbook-02, the single future `test` invocation
requires the root-reviewed guard-ready.json SHA256
(da70b0421c48165f9c3db880c75cac31c59f7901f9f55de583f05b0a135e0853) and
--activate-product-stage picker-final-app-linux-20261111, using the same
run root and no inbox; `vet` stays held behind a passed test. Linux
results do not prove Windows physical IME or Windows delivery; .29 and
all closed evidence remain unchanged. Root must inspect the real remote
result above and complete backend/platform concerns before authorizing
the product stage.
