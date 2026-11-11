# Image-reference `.26` delivery

Final metadata status: **PASS and frozen for root review** (2026-11-10). The
original package invocation remains a truthful exit1 attempt; the accepted
candidate was adopted through a one-file metadata repair, followed by the
authorized single hidden host, one metadata-only aggregate retry and external
closure. No build, package script, test or vet was repeated.

## Candidate and source identities

| Object | Actual identity |
|---|---|
|Candidate|26 files,27,281,492 bytes, inventory identity `7DB6E2660206EABC4D689894A0B1D50400C6DFA8AAB71CBD3689C6055EE1FB64`|
|Host|14,455,808 bytes, SHA-256 `B219024B7C5B444DA0307579C1500821A6B24FDCDD9B4D47E0FC14E863A51B67`|
|Release manifest|4,674 bytes, 25 payload entries, SHA-256 `EFF8DD2FAF05EC620AB5976C2A72E647F12570876B15B13606062B7DC9C4E20E`|
|Source `portable/README.md`|16,680 bytes, SHA-256 `5BFE74AAD557CC796A8A9C11954E730A3B5A0FC81884F997CC11F84D6D01DD91`|
|Candidate `README.md`|16,694 bytes, SHA-256 `DCA36FED6CB7CBEB97D9A6C7C879C6886563703E4BB65BD5293D9C99AA692138`|
|Scoped source inputs|389 files,4,225,295 bytes, identity `9A1D5C2E8FDA9922CFAC7F47ADF2D0309997CE7990B325FED7EE1076CAC34293`|

The failed pre-repair candidate was26 files/27,281,485 bytes with inventory
identity `3D850B1327916119ED051AEFFC93C10F6475DB8CDC0432225A8BC5B011B55F8D`.
The repair comparison proves exactly two candidate content changes:
`README.md` and `release-manifest.json`. Every executable, engine manifest,
license, notice and attribution asset remained byte-identical.

The scoped build-source comparison proves exactly one changed file:
`native/portable/README.md`. The source wording changed from the backtick
example containing adjacent `](` to:

```text
Markdown 中的单行引用由 `![标签]` 后紧跟 `(目标)` 构成
```

This preserves the explanation while avoiding the portable link scanner's
literal substitution. No runtime source changed.

## Original package failure

The sole package invocation remains `InvocationCount=1`, exit1. Stage and
outer package invocations also remain exit1. All five GUI build records remain
exit0 with empty stdout and stderr. The package stopped after the build phase at:

```text
Portable authored local link does not resolve inside output: README.md -> 目标
```

The failed candidate README and manifest remain archived with their original
bytes. The original package-result files remain absent; no successful package
result was backfilled. The five builds and original stream records are pinned
in the preserved failure inventory.

## Repair and parse history

The first adoption attempt was metadata-only and exited before writing a
repair result. It had written the approved candidate README and removed the
archived failed manifest, then the existing link scanner ran before manifest
regeneration and stopped at:

```text
Portable authored local link does not resolve inside output: README.md -> release-manifest.json
```

That partial state, exact exception and hashes are preserved in
[markdown-image-reference-release26-repair-attempt1.md](markdown-image-reference-release26-repair-attempt1.md).
The corrected resumable adoption regenerated the manifest before scanning,
then passed all source/candidate delta, local-link, engine-manifest, PE,
asset-copy and candidate-copy checks.

Parse evidence is preserved without replacing earlier attempts:

| Attempt | Result | Recorded facts |
|---|---|---|
|Initial repair parse|exit1, `Pass=false`|two per-file errors in the initial `repair-adoption.ps1`; the old aggregate summary incorrectly displayed0, and that failure file was not overwritten|
|First correction parse|exit0, `Pass=true`|syntax correction applied; retained because its parser aggregate still exposed the count bug|
|Final pre-adoption parse|exit0, `Pass=true`|24 scripts, numeric `ParseErrorCount=0`, `BadScriptCount=0`|
|Post-adoption parse|exit0, `Pass=true`|26 scripts, numeric `ParseErrorCount=0`, `BadScriptCount=0`|

The post-adoption parse covers the narrow aggregate compatibility correction
that accepts the repair record's `Host.SHA256`. Its provenance is separate from
the frozen repair result in
[repair-helper-provenance-addendum.json](../artifacts/acceptance/markdown-image-reference-release26-20261110/repair-helper-provenance-addendum.json).

Repair adoption result:

`native/artifacts/acceptance/markdown-image-reference-release26-20261110/portable-link-repair-result.json`

39,262 bytes, SHA-256
`7BE8218A41ACDD2AB4C9C59F889E785D9DE8EC19C097D4EF683D9B23C5280014`.

## Hidden host

The one authorized hidden exact-copy host passed:

- one app start and one run, `run-1-mixed-cli-open`;
- hidden window found, with no forced repaint or input;
- 5,000 ms requested alive settling, 5,461.8127 ms actual;
- fresh strict ordered two-tab text+PNG session before close;
- exactly one owned `WM_CLOSE`;
- fresh normal-close publication with the same 679-byte session,
  SHA-256 `F1E067392BD60354E1992273BD95CEA8FEE6E9A34DC1E29519EB0EA4B8B923D1`;
- app exit0, empty app streams, Job total1/active0, disposed process and Job;
- zero worker operations; candidate, copied bundle, fixtures and installed
  plugin remained preserved.

This run does not claim image rendering, physical input/IME, dirty-draft
recovery, resource use or a second restore.

## Aggregate and closure

Aggregate attempt1 exited1 and is preserved under its numbered capture path.
It failed before writing attempt1 `aggregate.json` because the repair record
exposed `Host.SHA256` while the aggregate still required the legacy scalar
`HostSHA256`. Attempt2 is the sole metadata-only retry and passed **13/13**
with 175 evidence records.

`aggregate.json` is59,808 bytes, SHA-256
`CA89586C05515313CEB75AA17744FA9A3EA04C11AA247B686AC5F7C37C9ACC8E`.

External closure passed after aggregate. It verified aggregate evidence and
protected releases were unchanged, then inventoried:

| Closed root | Files | Bytes | Inventory identity |
|---|---:|---:|---|
|Direct evidence|177|55,729,781|`ED5BE339D567A45C5F954AC42DAB3B5F78254BDBB818C6BE9949FB7FCBCF1296`|
|Aggregate capture attempt1|6|2,716|`0EF46BFEE7C164FD711C8BC7401EE93EEF35758887532A50C57613FEDC7AB8D5`|
|Aggregate capture attempt2|6|3,375|`2BD33DA06E8EC8E6C30C0C40106A300F985420A88DD39B00149F464A61B4B2F3`|
|Total|189|55,735,872|`DAFE2B31DF426BAC111AB2179A031234EA90BD28E912E624E055661C6928B27D`|

The final inventory identity is the SHA-256 of the84,468-byte closed-inventory
JSON outside the scanned evidence root.

## Frozen scope and limits

This delivery supports the exact copied `.26` host and isolated hidden
mixed-session evidence above. It does not add:

- another package, build, test or vet run;
- a worker request or worker lifecycle claim;
- physical GUI/IME input, forced painting, image rendering or a second restore;
- dirty-draft recovery, whole-process memory or performance measurement;
- authority for a `.27` stage or any runtime/source change.

Candidate, repair result, aggregate, capture siblings and external closure are
frozen for root review.
