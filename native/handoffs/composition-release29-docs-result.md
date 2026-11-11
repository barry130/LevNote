# `.29` product documentation complete

Completed2026-11-10 under `composition-release29-docs-task.md`. Only the six
owned product docs changed; no runtime, test, package, frozen evidence,
portable README or product execution changed. The saved check observed the
2,802B draft at SHA256
`10037F1A6DFC8CAFCE36F2FD2230DC64F6368751F943A2FF55AAD077C58F9478`;
that deliberately is not presented as the final report identity.

The current runnable is
`native/bin/levnote-native-worker-20261110.29/levnote-native.exe`, version
`native-worker.20261110.29`. The six pages state that consecutive updates of
one complete marked IME span retain original/current undo endpoints, while
cancel and one-step undo/redo remain one transaction. Returning to the original
text adds no empty undo item and keeps earlier redo. Irregular explicit ranges
still append ordinary suffix history and may retain many intermediate edits.

`.28` size refusal, valid shorter retry, `Esc` cancel and Save/select/close
barriers remain active. The docs distinguish marked-span history behavior from
physical Windows IME, second-launch recovery, worker execution, rendered PNG
comparison and whole-process memory measurement. The accepted tests include
600-update structural and240-update Tester sequences as source behavior.

The completed `.29` evidence is recorded with the actual final9package
test/vet,26-file package, one copied hidden host, aggregate attempt1 and
external closure. The host is an ordered CLI open/session publication and
normal close, not a second restore. Initialization attempt1 failed before
product execution because a README literal crossed a newline; its streams and
records remain, and the narrowly corrected attempt2 passed without repeating
product stages.

`ROADMAP.md` now labels both `本轮集成验收` and
`宿主＋四个引擎目录交付` as current `.29`. The `.27` integration facts remain
historical, and the complete `.28`/`.27` changelog and validation bodies retain
their earlier facts with only the required current-to-historical relabeling.

## Narrow docs check

The six pre-edit files are backed up under
`native/artifacts/acceptance/composition-release29-docs-20261110/input-backup/`.
Their original sizes and hashes are in
`input-identities-before.json` (1,457B / SHA256
`6488C983C94B31E7F35FB843ED5FB2B0B3A5F7B238A4A0952C2DF2A42225E80B`).

Exactly one saved check ran. PowerShell7.6.9 completed in2,217.9711ms with
exit1; `33/37` assertions passed and four failed. The original invocation,
empty stderr, stdout, exit record and structured failure remain:

| Evidence | Bytes | SHA256 |
|---|---:|---|
| `verify-invocation.json` |808|`C71473F8414A02F5D9E36F0ADD2E7520E56F67F4BA1337FD5D9ABCFEDC56AE80`|
| `verify.stdout.log` |3,723|`FF6CFE870AFF7C3EFD4CB715F19544E94D0FE93A35EA99AFA2A7255E021BDA89`|
| `verify.stderr.log` |0|`E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`|
| `verify-exit.json` |251|`8CA9520C318CFFD7617991BBD9045A0101D61EB7F62CB1B40E87990D6B35A771`|
| `docs-check-result.json` |8,024|`B59DAAAC2C7BB492C0501B425298AE4C47398079368FE830A4C22981EFF09367`|

The four failures are verifier scope defects, and the original exit1 is not
reported as a passing run:

- The validation-heading expression omitted whitespace after the closing
  `.29` backtick. The actual heading is
  ``## 输入法组合更新合并与 `.29` 当前交付：2026-11-10``.
- The ROADMAP expression expected `保存、切换/关闭`; the actual `.28`
  barrier prose is `保存/切换/关闭不会静默提交`.
- The generic link expression read inline code examples
  `![label](target)` and `![alt](src "title")` as filesystem links, producing
  nine false broken targets. Every other parsed relative target resolved.
- The `.28` validation body differs from the backup only by the required
  `当前入口` to `历史入口` entry relabel. After restoring that single label,
  the full4,150-character body is byte-for-byte identical to the backup.

The remaining current-entry, ROADMAP label, required-fact, evidence-target,
fence, historical-row and `.27` body assertions passed. A second broad check
was not run. The exact checker
`verify-docs.ps1` is18,848B / SHA256
`F962C211C4D5756B911A7C3B8458B3BEB1CD8C32BDE12E141EA6439DDD2EA5ED`;
the one-shot wrapper is`run-docs-check.ps1`,2,006B / SHA256
`F3B2448E5A1EBA71CE3CF00154C94FFC45AB1F00CF2503084E9B1E3DF1E11849`.
No raw output was reconstructed.

### Narrow correction run

The later authorized four-condition correction ran once against the then
current5,909B report, SHA256
`E1AFF857841BF50D4BD9273A2757D28C6F716445C70C90C6FAB7CDDE9B9EDBD5`.
It exited1 with `3/4` conditions passing and empty stderr:

| Evidence | Bytes | SHA256 |
|---|---:|---|
| `narrow-correction-invocation.json` |898|`9FC5FECB554E89EB917F3F1D2152897985B6B66495A92FC0F67BCB4AE8C2B2C6`|
| `narrow-correction.stdout.log` |646|`557338521506BA3C3744B33D6EE45310DE77D673F1073B83ED44887F7F9001AA`|
| `narrow-correction.stderr.log` |0|`E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`|
| `narrow-correction-exit.json` |261|`E28880E0CADEB3595E649912B0ECB6051A8094BE84B6F06C93CE00C9EAD9233B`|
| `narrow-correction-result.json` |2,702|`F4E5E44F2FF55B391E787458252654121740E8B605525DF727120A05347C856F`|

The corrected heading-space and slash-literal checks passed. The code-aware
link scan resolved584 actual relative targets after excluding the nine inline
example tokens, without whitelisting missing filesystem paths. The fourth
helper assertion failed because the `.28` section begins with a blank line
before `历史入口`; it incorrectly expected that label as the immediate first
character. A read-only comparison after the saved run shows backup and current
bodies both4,150 characters. Replacing only the first meaningful
`历史入口` line with `当前入口` makes the complete body byte-identical. No
content difference or six-doc correction was found, and no third docs check
was run.

The narrow checker is9,606B / SHA256
`8AD082C67504116FACC8F10EB39B6EE1CA021211ACBE9485F00A8A2753EE4047`;
its one-shot wrapper is2,139B / SHA256
`4561EFE423A075C83BC58CC60C86F17C3AC83BBF748DFBA248ADC3087A8089BB`.
The previously generated identity snapshot is preserved as
`final-identities-pre-narrow.json`,4,126B / SHA256
`9143A166CAC909DA6974770A5E1FD036F4E84AA1983EBF83626F1C8448CF3FBC`.

## Changed paths and final identities

| Path | Before | After |
|---|---:|---:|
| `README.md` |20,885B / `C877577DF62F36D914BF3C998CDDB7A737C12E386C08D8BBF287FFA6371A81DE`|21,722B / `CC55C274A60FF2D178387DC5939D08ADDAB0D99ED5AD1F6E8165D8AB8599427C`|
| `ROADMAP.md` |125,180B / `C0C5B894080D7078D89B5764E55EA9A0A86EF1932B703626EDA38A34712D1F01`|125,922B / `D6C3196A913D33E3B0ABC1DF0227064E7639BBF5710ED5975C8B08733398DC3B`|
| `CHANGELOG.md` |156,129B / `A40AF07961BF84D3766188927862BA77EFFEA764B62A605FBBCC856AC391D098`|157,759B / `5D15C35D511CBD1A81AA9A06D9F0E7E2CFEEA38F525AC2881CD23FABA4E683D7`|
| `native/README.md` |72,429B / `B3E26A0BC6546608231F2E32E348ADE053848F91CCFE3DB8FA44EDCCF8F01DD0`|74,225B / `D73F310BF6089D22331D930EC0FDC636924607B026BD6687F312D89A29493CCA`|
| `native/ARCHITECTURE.md` |72,086B / `569C3809F8B48C813101269E124C3E5587F9619E50CE1DF3AD6C5DA08501B578`|73,027B / `C841FEAA490AC15A236783A764D35EB552F9EFB3776344BD2F9F2978CEC04089`|
| `native/VALIDATION.md` |171,577B / `63FCF0BAC61101ACA62C14540DFE9125082EE79A217AF7F61240113C7BED4DB7`|178,532B / `3CD044E232F3E49A3F3BA15C26E190A060BE5DE852882FB52F4475522F4990CA`|

The final report identity is written after publication to
`native/artifacts/acceptance/composition-release29-docs-20261110/final-identities.json`
so this report does not self-hash or present an intermediate value as final.

No product test, vet, parser, package, host, worker, capture, benchmark or
resource measurement is run by this docs task.
