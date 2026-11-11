# .26 package stopped by the portable README link scanner

2026-11-10. The sole authorized `.26` package invocation exited1. All five GUI
build records immediately before the failing check exited0 with empty stdout and
stderr. No host, restore, worker request, retest, capture or second product run
has occurred.

The package script reached
`portable-package-lib.ps1:87` and failed:

`Portable authored local link does not resolve inside output: README.md -> 目标`

The scanner at `portable-package-lib.ps1:82` treats every literal `](` substring
as a local Markdown link. The documentation example at
`native/portable/README.md:71` therefore looks like a link to `目标`, even though
it is inside backticks and describes an image.

The exact failure is preserved in
[markdown-image-reference-release26-portable-link-failure-inventory.json](markdown-image-reference-release26-portable-link-failure-inventory.json),
52,414 bytes, SHA-256
`B99D274C6A4CC94AF4FBAD039EAFB410E4527A5FD1CF65548A3C2C9318E8A337`.
It records all72 direct evidence files and the five build records. The original
failed candidate README and manifest bytes are retained under
`markdown-image-reference-release26-portable-link-failure-assets-20261110/`.

| Failed object | Bytes | SHA-256 |
|---|---:|---|
|Candidate `README.md`|16,687|`BEB184ED5E52B9B2E309B1D6F046379DC378B7760578F124F53A050ED70B1FF9`|
|Candidate `release-manifest.json`|4,674|`6BF61E468A87169987976F15D696C75B38C7C39BE2A115A0F084CC087D177290`|
|Source `README.md` before repair|16,673|`66066F3E032BC7CB13B132C1E540A88437CE463AA116C01B40AAD444680FCE7A`|
|Stage package stderr|382|`95000DE89448CF46ED6419F9CE71E775B3015CE4A7DF211B7E93C55356E0CE9B`|
|Inner package stderr|337|`214319C4C7C7DF3290C4A8CD66F0C15F6E3ABE7FF28A754929D6243D19BCD367`|

## Proposed metadata-only repair

No build is needed. The clear source wording change is:

```text
before: Markdown 中的单行 `![标签](目标)` 是显式图片引用
after:  Markdown 中的单行引用由 `![标签]` 后紧跟 `(目标)` 构成
```

This keeps the exact user explanation while removing the adjacent `](` that
the scanner mistakes for a link. It is deterministic and avoids relying on an
entity rendering rule.

The corresponding copied helper contract at
`delivery-lib.ps1:293` must replace `'单行', '![标签](目标)'` with:

```powershell
'单行引用由 `![标签]` 后紧跟 `(目标)` 构成'
```

The predicted identities for that exact repair are:

| Metadata / output | Current | Predicted after repair |
|---|---:|---:|
|Source README|16,673 B<br>`66066F3E…FCE7A`|16,680 B<br>`5BFE74AAD557CC796A8A9C11954E730A3B5A0FC81884F997CC11F84D6D01DD91`|
|Candidate README after version render|16,687 B<br>`BEB184ED…1FF9`|16,694 B<br>`DCA36FED6CB7CBEB97D9A6C7C879C6886563703E4BB65BD5293D9C99AA692138`|
|Copied `delivery-lib.ps1`|29,289 B<br>`CF4CF4F4…419E`|29,317 B<br>`5F02C2B02703E4DB9C6EAEEF4DF2B2CEAD282A8C57AD07D5DCD4041B3B22A118`|
|Scoped source input|389 files<br>4,225,288 B<br>`1683D74C…8BC3`|389 files<br>4,225,295 B<br>`7434DD07FE16FF3DEB961650B91904829C7337A25D8A3D9BF023F9BCBFA43035`|
|Candidate payload manifest|4,674 B<br>`6BF61E46…7290`|4,674 B<br>`EFF8DD2FAF05EC620AB5976C2A72E647F12570876B15B13606062B7DC9C4E20E`|
|Complete candidate|26 files<br>27,281,485 B<br>`3D850B13…F8D`|26 files<br>27,281,492 B<br>`7A19787C4949F72281193900B03117FFC9E5E5B1BB589F67CAE0AEB25A6C72CE`|

The candidate manifest calculation excludes the old manifest before invoking
`Write-PortableReleaseManifest`, preserving its established25 payload entries.
The old manifest and README are already copied outside the candidate.

## Required metadata-only sequence after activation

1. Apply the one-line source README change and the copied helper contract
   change. This is one build-source file delta; all runtime and package
   executables remain untouched.
2. Leave the initialized `context.json` unchanged. Its original source identity
   and portable README fingerprint remain the pre-repair baseline.
3. Leave the top-level `inputs-before.json` unchanged. Its current identity is
   94,726 bytes, SHA-256
   `16BB863AB128EA78CC8DABF4F038D6BD407B8388CBB99F9F67615B2BD1101CF1`.
4. Update `copied-helper-provenance.json` portable README identity and the
   `delivery-lib.ps1` target bytes/hash; preserve every other source/target
   provenance field.
5. Render the repaired source README with the same
   `.Replace('{{VERSION}}', $ctx.version)` operation used inside
   `Write-PortableAssets`, and write only candidate `README.md`. Do not call
   `Write-PortableAssets`: it would rewrite every portable asset.
6. Verify the current candidate manifest matches its already archived failure
   copy, resolve both the candidate root and manifest to absolute paths, require
   the manifest path to remain inside the candidate root, and remove only
   `release-manifest.json`.
7. Run the existing `Assert-PortableLocalLinks`; then call only the existing
   `Write-PortableReleaseManifest` function. The repaired manifest must contain
   exactly25 payload entries.
8. Compare the complete before/after candidate inventories. Exactly two files
   may differ: `README.md` and the regenerated `release-manifest.json`. All
   executables, four engine manifests, licenses, attribution and notices must
   remain byte-identical.
9. Write a separate truthful `portable-link-repair-result.json`. It must retain
   the original package invocation count1 and exit1, all five successful build
   records, original failed README/manifest fingerprints and archive paths,
   exact source/helper/candidate deltas, local-link PASS, unchanged-file proof,
   and `ProductRerun=false`. Do not create a fake successful
   `package-result.json` or `package/package-result.json`.
10. Update the preparation/result and delivery handoffs with the failed package
    attempt, repair adoption and remaining one-host budget.

The hidden-host launcher currently requires `package-result.json`, and aggregate
currently requires a successful package command plus internal package result.
Those copied helpers need narrow branches that load and verify this repair
record instead: original package stage/outer exit1 must remain visible, the
five original builds must remain the only builds, the repaired candidate and
manifest identities must match, and no product rerun is allowed. The external
closure helper likewise needs an explicit branch accepting exactly this
preserved package failure and adopting the repair result. Those branches must
not rewrite original exit records.

Because the copied helper changes, a final scoped parse is needed before the
repair result is frozen. Aggregate must validate the repaired source inventory
against both the preserved pre-repair baseline and the exact adoption record;
it must not overwrite `context.json` or accept an arbitrary new source identity.

This repair has not been applied. Root activation is required before editing the
source README, copied helper, candidate README/manifest, provenance, repair
result, narrow helper branches or delivery handoffs.
