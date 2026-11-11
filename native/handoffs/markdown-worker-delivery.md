# Markdown worker clean launch delivery — root accepted

Root accepted this delivery after reading actual payload hashes, notices,
18local-link evidence,7/7checks and the separate17,016,396-byte summary correction.
The directory/evidence remain immutable; portable-build-next-stage.md is now
active. Earlier delivery-step history below is preserved.

2026-11-09local. This finite step only copied accepted payload/assets and updated
documentation. No new build, test, helper launch, product/runtime edit or replay
of passed integration stages. Next portable-build stage remains queued.

User launch directory: `native/bin/levnote-native-worker-20261109.1/`.
Run `levnote-native.exe` there; keep the whole folder together. User configuration
remains `%APPDATA%\LevNoteNative`, not beside the executable.

| Accepted source → clean copy | Bytes | SHA256 |
| --- | ---: | --- |
| `levnote-native.exe` | 13,613,568 | `4B5DCECA5E5F5C3D3609336563F138247942E35E0C3BE65B382B32771A80B23C` |
| `engines/markdown-export/levnote-markdown-export.exe` | 3,352,576 | `DB8F02E6E187833FB52B4FC02B076B4AC96E357FB3E554533A698D8F050F6AD0` |
| `engines/markdown-export/manifest.json` | 337 | `64472251BD737D3BB82AFB05C35F648882D2E9D8D4A57BCFD1CDE8CF457CC453` |

Source is immutable
`artifacts/acceptance/markdown-worker-integration-20261109-013547/bundle/`.
All three current source hashes and copied destination hashes equal accepted
aggregate values. Protected old `native/bin/levnote-native.exe` remains
13,461,504bytes/SHA17E014.../UTCmtime2026-11-08T16:04:08.3209774Z.

Delivery has18files totaling17,016,396bytes: three payload files, rootLICENSE,
seven exact`native/licenses/`texts, portableREADME/notices, exact original
notices and MyGo/purego source README/patch attribution. Thirteen copied license/
attribution files match their source bytes and hashes. The portable notices
adjust relative links and explicitly mark source-only repository references;
the original notices are an exact `.txt`copy. All18local links in the portable
README/notices resolve within delivery. Upstream README links are attribution,
explicitly not a local file/dependency promise. No testexe, logs, build helpers
or scripts are included.

`artifacts/acceptance/markdown-worker-delivery-20261109-020115/` contains:

- `delivery-manifest.json`,26,398bytes,
  SHA256`74725706549A89978D68BE35EE9A5D83DB3CC0287B77E6DA195DF7011357FECB`:
  per-file source/destination fingerprints,7/7checks,18local-link checks,
  173unchanged product/test/build inputs, accepted records and13finaldoc hashes.
- `delivery-notes.json`,SHA256
  `022258E44DC4813AC5118AC618EB1B636EAE55B981736BF8089C5A21281733F9`:
  corrects only the manifest's null total-size summary. PowerShell Measure-Object
  did not sum OrderedDictionary keys; explicit summation of the already-recorded
  individual sizes gives17,016,396bytes. All original individual sizes/hashes and
  seven checks are valid; original manifest retained, no replay.

Separate root acceptance record was created at actual UTC17:55:18.9616705Z in
accepted integration attempt04. All SHA fields are derived by Get-FileHash from
actual paths; validator field is64hex and matches file (tool display may mask
it). Record SHA256`12E9AF0EF7BC92D917D2FD7D4BDA0EDB98BE5FE458F18D0B0A53B3EC3319B5AC`.
Accepted aggregateFB0468... and20/20validator remain untouched.

Final docs: rootREADME/ROADMAP/CHANGELOG; nativeREADME/VALIDATION/ARCHITECTURE,
formats/API, THIRD_PARTY_NOTICES and current integration/protocol/runtime/app
handoffs. They lead with the clean launch path, fixed worker vsJSONplugins,
APPDATA behavior, visible host-only export refusal, synchronous publication,
post-render64MiB bound and exact finite validation. No sandbox/authentication,
peak-memory, resource comparison, OSIME/GPU, signature or reproducible-release
claim. The next stage should separate a fresh user OutputDirectory from explicit
EvidenceDirectory or fresh deterministic sibling, but no such script work is
activated or implemented by this delivery.
