# `.25` current documentation result

Completed and frozen 2026-11-10 after root accepted the `.25` package 13/13 and
reviewed the documentation corrections. This task changed only the six owned
product documents and this result handoff.

## Result

- Root `README.md` names `.25` as the accepted runnable delivery, links its
  acceptance/delivery, and documents actual `前插列` / `后插列` / `删此列`
  actions, 1–8 column limits, raw-byte preservation, separate undo, keyboard
  flow and visible refusal/fallback. The duplicate `.24` introductory sentence
  was removed; `.24` image/recovery limits remain.
- `ROADMAP.md` marks `.25` as the current integration/delivery, adds the
  Markdown table-column row, updates packaging to current `.25`, and removes
  obsolete no-column-operation claims while preserving missing sorting/merge/
  formula capabilities.
- `CHANGELOG.md` adds the `.25` user-facing entry with the actual visible
  `前插列` / `后插列` / `删此列` names. It states that F6 enters the first row
  action and Tab/Shift+Tab continues to column/alignment actions, rather than
  claiming F6 directly opens a column action.
- `native/README.md` updates launch/package references to `.25`, uses the same
  actual control names and accurate F6→first row action flow, changes the
  current version paragraph to `.25`, and preserves `.24` and earlier as
  historical.
- `native/ARCHITECTURE.md` updates the current-delivery paragraph to `.25`,
  summarizes precise source planning, raw-cell preservation, structural undo,
  keyboard safeguards and whole-table fallback, and removes the stale claim
  that column insert/delete tools do not exist.
- `native/VALIDATION.md` inserts the sole `.25` identity/evidence section,
  uses the release manifest's actual 64-hex SHA-256 fingerprint, and preserves
  the complete accepted `.24` section as historical evidence with its table,
  hidden host, recovery, regression and metadata-history paragraphs intact.

The separately activated image-reference source work is not part of `.25`.

## Root review corrections applied

1. Removed the repeated `.24` introductory sentence in root `README.md`.
2. Replaced the mistyped `.25` release-manifest digest with the actual fingerprint
   read from `native/bin/levnote-native-worker-20261110.25/release-manifest.json`.
3. Restored the full accepted `.24` validation evidence and relabeled it as
   historical; no historical evidence paragraph was deleted.
4. Removed the stale architecture claim that column insert/delete tools are absent
   while preserving sorting/merge/formula/full-GFM limitations.
5. Corrected `CHANGELOG.md` to the actual `前插列` / `后插列` / `删此列` labels
   and accurate F6→Tab/Shift+Tab action flow.
6. Regenerated this result and all six identities after the final files were
   frozen; the preliminary result is replaced by this corrected result.
7. Corrected the remaining `native/README.md` usage sentence so F6 enters the
  first row action and Tab/Shift+Tab continues to column/alignment actions;
  refreshed only this affected native README identity and this result.

## Scoped check

One focused wording/link/hash check was run after corrections:

```powershell
$files=@('README.md','ROADMAP.md','CHANGELOG.md','native/README.md','native/ARCHITECTURE.md','native/VALIDATION.md')
# duplicate consecutive .24 image sentence
# actual release-manifest hash exists in VALIDATION and is 64 hex characters
# stale current-.24/current-no-column/F6-direct-column/old-control-label wording
# .24 acceptance/delivery/source links exist
# current .25 references
# final size/SHA256 identity for all six files
```

Actual results:

- duplicate `.24` intro: no duplicates.
- manifest fingerprint: `OK CA064C3E7824EDCB179465CD8E60D982D057D2EA1D45A0E8407BD44508CF9BC2`, exact 64-hex match with
  the accepted manifest.
- stale current/capability/action wording: no matches.
- restored `.24` links: all `OK`; no `MISSING`.
- current `.25` references listed root/native launch paths, the current version
  paragraph, the changelog heading and the validation section.

## Edited file identities

Actual sizes and SHA-256 values generated from the final edited files:

| File | Bytes | SHA-256 |
|---|---:|---|
| `README.md` |16761|3592886039C4FB2384D9A744D6107A368D6BBD1AAB82D246F935F4D49AF3A4A2|
| `ROADMAP.md` |122258|37F274C79343ADF4DC340A098B16FA9D00ABEB9EACFE8BA3B469A7A7207BD852|
| `CHANGELOG.md` |150738|1BB77ED2DA621BF29B7A77C1CDAC6BDBD021260398B081A51078B5E1D597702A|
| `native/README.md` |65047|BDDB007AE63D17CB2371BC3F64D15FBDA56ACBE82DAF809A230EA52F8234AC86|
| `native/ARCHITECTURE.md` |66917|A2F29F580D586C6363493890A173514536A26D4358DE65B26239C5D4EF7524C6|
| `native/VALIDATION.md` |156787|53654C03EA77AC1D40EB91566B325BD85F68513536E8FECA37CB6682F5A34F10|

No portable source/package/evidence, runtime source, tests, builds, launches,
workers, captures, benchmarks or resource operations were modified or executed.

