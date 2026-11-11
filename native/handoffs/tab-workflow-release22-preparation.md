# `.22` tab-workflow host delivery preparation

Status: **PREPARED ONLY; EXECUTION INACTIVE**

Prepared 2026-11-10 in
[`native/handoffs/tab-workflow-release22-preparation-20261110/`](native/handoffs/tab-workflow-release22-preparation-20261110/).
This is a small adapted copy of the accepted package/host delivery harness. The
only execution performed was static PowerShell parsing. The first parse found
one syntax error; after repair, the second parse passed. Root then found three
runtime-schema reference defects that parsing cannot catch. Those references
were corrected and the third parse passed. Root then found four additional
helper defects: the installed-plugin comparison required an identical path, the
PE helper was absent, aggregate capture used an unsafe inline `if`, and scoped
source fingerprints lacked a path association in inventory identity. Those were
corrected and the final parse passed. No product/package/source freeze, package
build, host launch, test, tester, benchmark, capture, or worker request was run.

The complete helper set uses the normal `ExecutionAuthorized` context guard:

1. `initialize-context.ps1` — records `.22` paths, root-supplied accepted tab
   gate files, the actual scoped source identity, the current portable README
   fingerprint, reviewed `diagram-notes.json`, and the immutable `.21` package
   plus its separate direct/retry inventories.
2. `baseline.ps1` — freezes scoped inputs and gates and confirms the absent
   candidate.
3. `run-package.ps1` — performs one package invocation and five GUI builds,
   then validates the actual 25-file set, release manifest, four engine
   manifests, five Windows amd64 GUI PE payloads, assets/notices/local links,
   and the complete package copy.
4. `run-host-smoke.ps1` — performs one hidden, byte-identical copied host
   startup from a fresh isolated `C:` config, with the reviewed plugin JSON and
   UTF-8/BOM/CRLF `.diagramnote` fixture, exact held process identity, one
   owned `WM_CLOSE`, normal exit 0, clean one-tab session, no worker operation,
   and unchanged fixture/plugin/candidate/copy.
5. `aggregate.ps1` — validates the gates and hashes all closed files in the
   evidence root.

Invocation is not active. After root accepts the final tab source/tests and
activates `tab-workflow-release22-delivery-task.md`, the owner initializes with
the exact harness directory, absent candidate
`F:\AIproject\LevNote\native\bin\levnote-native-worker-20261110.22`, accepted tab
gate paths, and `-ExecuteDelivery`, then runs only
`run-stage.ps1 -Name baseline|package|host-smoke|aggregate`. Candidate version
is fixed to `native-worker.20261110.22`.

The static parse record is
[`static-parse.json`](native/handoffs/tab-workflow-release22-preparation-20261110/static-parse.json):
10 scripts, 0 parse errors, pass, PowerShell 7.6.9.

`static-parse-first-attempt.*` preserves the first parse record,
`static-parse-schema-fix-attempt.*` preserves the parse after the context-schema
repair, and the final parse and streams are
`static-parse.json`, `static-parse-final.stdout.log`,
`static-parse-final.stderr.log`, and `static-parse-final.exit.txt`
(the zero byte for ASCII `0`). Aggregate execution capture, when later
authorized, is redirected to the separate sibling directory
`tab-workflow-release22-preparation-20261110-aggregate-capture`; it is not
inside the evidence root being hashed.

The source identity is defined concretely: the complete inventory identity of
the actual scoped product/build inputs returned by `source-scope.ps1`, captured
by `initialize-context.ps1` and compared through baseline and aggregate. It is
not a filesystem path. No speculative app hash or acceptance result is embedded.

Protected scope:

- `.21` package: 25 files / 26,859,809 bytes / inventory identity
  `3126E7898073CEBBC02EAB907EEF541C8A3D5538F0B77A45D58B6CE9DBE72ED7`.
- `.21` direct evidence: 202 files / 91,722,328 bytes / inventory identity
  `55B150C3A423BC3D213E7CF7282F1D87F9247AD35768236144D9217E7F4CE79B`.
- Separate `.21` retry sibling: 4 files / 1,635 bytes / inventory identity
  `FA8CDD03BA1AC32257A97403BE861DC82E9EE3EF2443866D7860B6BC1EF6AB16`.

Later host evidence proves packaged initialization and clean lifecycle only.
It does not establish tab behavior, physical UI/IME, interactive export, dirty
recovery, process memory/CPU, installer behavior, or cross-platform operation.
Tab shortcut/reveal behavior remains with accepted app tests and any optional
Tester geometry capture; worker behavior is inherited from accepted `.21`
evidence. Frozen for root review.

## Final helper identities

| File | Bytes | SHA-256 |
|---|---:|---|
| `aggregate.ps1` | 19,460 | `1E62861A7EC3C505FB1D33202DB549FD9FA60D57040475233B1134F4C60008F1` |
| `baseline.ps1` | 8,084 | `E2D48E046B9A394DE1BFF7647E3D31BBA2796D15C34A9F7EC048D21484D19AB3` |
| `delivery-lib.ps1` | 8,635 | `5FEABF25DCD8D2FB07E9EB05A63AED905646B9F0B8CA0A1BA5D5C65DBAABBD3E` |
| `initialize-context.ps1` | 8,661 | `1688D2DAFD6B5092228F15ABE22F57164F74764FB655332EF803897385DFC022` |
| `invoke-stage.ps1` | 2,018 | `7F9B08082BAEBD3B5F736A928448C50576C19FA8D2B3763AA06C75A9EC27DB90` |
| `run-host-smoke.ps1` | 19,856 | `5F4A3401A65616F817A47D3566FCB4AC657849CF83ABE2013584FE59EF824A79` |
| `run-package.ps1` | 12,880 | `56BC3031BBACE1D9F98C47873B8E08782F277D54D28EAC31AA4CB6E340D414F4` |
| `run-stage.ps1` | 2,317 | `FACB9D74C34894CC970E74B28301AC6D7AB3721DF5188B2930676D7345FA3F22` |
| `source-scope.ps1` | 1,599 | `95E9EE57648CDC88815856FD77C5C9D17F5526036FF0A270063DD2009C5B1AC7` |
| `static-parse.ps1` | 1,568 | `1687DE5352F6CEC39B9C0B12BDB1E028C28A723079995A886279039A48C21F31` |

Parse records:

| File | Bytes | SHA-256 |
|---|---:|---|
| `static-parse-first-attempt.json` | 3,162 | `A346419E4CD0739962C53839D0C75A208444B5CBEF6B9778B666D7096545ED93` |
| `static-parse-first-attempt.stdout.log` | 3,163 | `C59038A6243ED5BF9A3D74C042D5A92DEDE5F12C669D1BA6F249F9CE987EB8D8` |
| `static-parse-first-attempt.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `static-parse-schema-fix-attempt.json` | 3,162 | `8CEFFD50E040FE92D8C75B43E40C4940494287CD018D36FEB8092AB21F9EB3BB` |
| `static-parse-schema-fix-attempt.stdout.log` | 3,163 | `2B0A807F8D603AF1CDCB3C09A5C610F16D1D3DD7CCB6782D83EDF0980E5314B8` |
| `static-parse-schema-fix-attempt.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `static-parse.json` | 3,162 | `8CEFFD50E040FE92D8C75B43E40C4940494287CD018D36FEB8092AB21F9EB3BB` |
| `static-parse.json` (final) | 3,162 | `A9D9BBE0BB421C4CFF2CFECFBB1BE9A5809C6E1A6EB04B2EE4320811DE417EB4` |
| `static-parse-final.stdout.log` | 3,163 | `5E2FB4103385DDF730E8099FD47E17E56C5BE7762D279C65A8B8B3B1866A5A3F` |
| `static-parse-final.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `static-parse-final.exit.txt` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
