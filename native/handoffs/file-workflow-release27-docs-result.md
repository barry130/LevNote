# .27 product documentation complete

Completed2026-11-10 under file-workflow-release27-docs-task.md. Only the six
owned product docs changed, and this result handoff was added. No runtime/test
source, package, frozen evidence, portable README, build, host, worker, test,
vet, capture, resource measurement or product execution was changed or
repeated.

Changed docs:

- `README.md`
- `ROADMAP.md`
- `CHANGELOG.md`
- `native/README.md`
- `native/ARCHITECTURE.md`
- `native/VALIDATION.md`

The docs now use
`native/bin/levnote-native-worker-20261110.27/levnote-native.exe` and
`native-worker.20261110.27` as the current accepted entry. They describe
recent16 transient saved-local-file reopen, current-disk reads, explicit
retry, no discarded-edit/draft restoration, the native menu and Ctrl+Shift+T,
duplicate reuse, and shared session.MaxTabs=100 admission for every assigned
new-tab route. `.26`, `.25` and `.24` validation records remain complete
historical sections.

Validation records the actual combined app test/vet commands and exits,
UTC intervals and streams, the unchanged nine-file/160,306-byte source union,
one package invocation/five GUI builds, first-aggregate13/13 acceptance, and
the one copied hidden-host lifecycle. The docs keep these scopes separate:
the .27 binary package itself was delivered and verified, while source tests,
package verification and bounded startup/session/close evidence do not prove
physical interaction of the packaged binary, physical keyboard/IME, 100-tab
RSS, image rendering, dirty recovery, second restore or cross-machine behavior.

Root correction review found the host evidence has 46 settling alive checks,
not 41; `native/VALIDATION.md` now records the frozen
`SessionSettlingAliveChecks=46` value. The prior blanket "binary delivery"
denial was removed because .27 did deliver and validate a binary package; the
remaining limit is specifically physical interaction inside the packaged
binary. Comparison with the pre-edit patch context and the accepted prior
validation document confirms that the complete .26, .25 and .24 section bodies
were retained. Only the .26 current-to-historical heading and entry labels
changed; those bodies were not reduced to headings. The focused check found
52, 46 and 41 body lines respectively, including the .26 679-byte/189-file,
.25 675-byte/152-file and .24 659-byte/attempt1 evidence.

The 128 MiB boundary is described only as a text source-file read limit.
Live-buffer/content growth admission is explicitly uncompleted in .27. No
label-count/source-limit product, content or memory cap is claimed. The
existing 4 MiB per-draft, 16 MiB encoded-session, 32 MiB history/graphics and
parser/worker limits remain separate.

## Narrow consistency check

Read-only PowerShell check result: PASS.

- all6 Markdown files had balanced code fences
  (`README.md`10, `ROADMAP.md`8, `CHANGELOG.md`0,
  `native/README.md`10, `native/ARCHITECTURE.md`6,
  `native/VALIDATION.md`36);
- all559 relative Markdown link targets resolved;
- all9 required .27 executable/package/acceptance/delivery/evidence paths
  existed;
- current-entry assertions selected .27 and no stale .27-current wording
  pointed at .26;
- `.26`, `.25` and `.24` historical validation headings remained;
- the prohibited 100-times-128 product-claim pattern was absent.

This correction did not rerun the already passed559-link/fence sweep. A
focused check covered the corrected46 value, removed delivery-denial wording,
retained historical-body markers, and updated document identities.

## Final product-doc identities

| File | Bytes | SHA256 |
|---|---:|---|
| `README.md` |19,184|`8394E8FA6CFA397F93FF3873099ED85B86A231B17461D0CE960E1941B66A42AF`|
| `ROADMAP.md` |124,194|`00EC13F23852499DEE0B4439B42538B20CF256CA19CAEFE4C2D3D000BF8B8577`|
| `CHANGELOG.md` |154,094|`A6FD6628CF8C0C9654B5BBBDC37F8E6FFB92D87E33C2867F48FBFF3A697A90B3`|
| `native/README.md` |69,650|`0F33B70C28CE321DF0D29B5EFEFC935E7C7880C75E027D6404D5F84C58321573`|
| `native/ARCHITECTURE.md` |70,268|`A457C59239A9D1D77179D03F75C2B47AABA4DCD0D87879953B68F5729A36AD93`|
| `native/VALIDATION.md` |165,514|`50C411C949CF5E37616B9FE5EFBB118EC592B174B149F3E95C446A622018678E`|

Product implementation and .27 acceptance remain unchanged.
