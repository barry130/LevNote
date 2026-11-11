# `.24` current-launch and usage documentation result

完成时间：2026-11-10。本任务只更新以下六个产品文档，并写入本结果：

- [README.md](../../README.md)
- [ROADMAP.md](../../ROADMAP.md)
- [CHANGELOG.md](../../CHANGELOG.md)
- [native/README.md](../README.md)
- [native/ARCHITECTURE.md](../ARCHITECTURE.md)
- [native/VALIDATION.md](../VALIDATION.md)

## Result

- Root `README.md` now presents `.24` as the accepted runnable delivery, describes
  image keys only when the image canvas has keyboard focus, records explicit
  100% retention across window changes, and distinguishes toolbar activation
  from merely focusing the toolbar. It preserves the image subset, readonly,
  byte-copy, readable-byte fallback and recovery limits.
- `ROADMAP.md` marks the accepted `.24` integration and image keyboard/recovery
  scope with the same focused-canvas and toolbar activation wording; `.23`
  remains the bounded historical image reader.
- `CHANGELOG.md` adds the unpublished `.24` user-facing entry for focused image
  keys, explicit 100%, toolbar focus return and 900ms application-timer
  recovery, with the source-test/one-hidden-host evidence boundary.
- `native/README.md` updates current launch/package references to `.24`,
  retains `.1`–`.23` history, and applies the same focused-canvas, explicit
  100% and enabled-toolbar-button wording in the image usage section.
- `native/ARCHITECTURE.md` keeps the binary/image ownership boundary and adds
  the `.24` application-owned timer paragraph: about 900ms after the latest
  edit, dispatched by an application timer, independent of View drawing;
  pending writes can be delayed by UI work or native dialogs, only recovery
  state is written, and no lossless-crash or exact wall-clock promise is made.
- `native/VALIDATION.md` inserts the sole `.24` identity/evidence section,
  preserves the accepted `.23` section as historical delivery, and changes
  that historical entry from current to historical. The new section records
  package identities, 13/13 validation, the one hidden exact host recovery,
  normal-close republication, evidence limits and metadata-only corrections.

The root final review corrections are applied: aggregate attempt1 is recorded
as 12/13 passing with only one failed item; the corrected regression description
says the imageview explicit-100% regression failed in the first combined test and
was fixed, with no failed recovery-deadline regression; and the architecture
fallback statement is limited to files whose original bytes remain readable.

The root-requested correction is applied consistently: shortcuts are scoped to
the image canvas having keyboard focus; activating an enabled zoom/fit/100%
toolbar button returns keyboard focus to the image canvas, while merely
focusing the toolbar is not an action; explicit 100% retains the chosen manual
zoom on resize, and `0` switches back to fit.

## Scoped check

One scoped check was run after all edits:

```powershell
$files = @('README.md','ROADMAP.md','CHANGELOG.md','native/README.md','native/ARCHITECTURE.md','native/VALIDATION.md')
"`n-- stale wording --"
Select-String -LiteralPath $files -Pattern '离开该模式再调整','工具栏入口聚焦后返回图片视图','工具栏焦点返回图片视图','13项中失败12项','首个组合100%恢复期限回归失败','首个恢复期限回归失败','字节视图仍可检查原文件'
"`n-- corrected facts --"
Select-String -LiteralPath @('native/VALIDATION.md','CHANGELOG.md','native/ARCHITECTURE.md') -Pattern '12/13通过、仅1项失败','首次合并测试中的图片100%模式回归失败','不能解码但仍可读取原始字节'
"`n-- current/history labels --"
Select-String -LiteralPath native/VALIDATION.md -Pattern '^## ', '当前入口', '历史入口'
"`n-- current .24 references --"
Select-String -LiteralPath $files -Pattern '20261110\.24'
"`n-- referenced native handoff files --"
$refs = @('native/handoffs/native-reading-release24-root-acceptance.md','native/handoffs/native-reading-release24-delivery.md','native/handoffs/native-autosave-keyboard-root-acceptance.md','native/handoffs/native-image-release23-root-acceptance.md','native/handoffs/native-image-release23-delivery.md','native/handoffs/native-hidden-autosave-review.md')
foreach($r in $refs){ if(Test-Path -LiteralPath $r){ "OK $r" } else { "MISSING $r" } }
```

Actual results:

- stale wording: no matches.
- `native/VALIDATION.md` top delivery labels: `.24 当前交付` and `.23 历史交付`.
- current `.24` references appear only in root `README.md`, `native/README.md`
  and the new `native/VALIDATION.md` section; the scoped command listed those
  seven path/version lines and no stale `.24` entry was reported.
- all six referenced acceptance/delivery files returned `OK`; no `MISSING`.
- root final-review stale statements: no matches; the three corrected facts were
  found in the intended documents.

## Edited file identities

Actual sizes and SHA-256 values generated from the edited files:

| File | Bytes | SHA-256 |
|---|---:|---|
| `README.md` |16062|399310D768F036AB392153B88CF6CFCF4151BAF2C906A2D6C5F7CC56A3E02DC3|
| `ROADMAP.md` |121701|E183D1AC9669B00AD67C7605DE59E5C48B9237C9AA7F73492E85EF32C7275583|
| `CHANGELOG.md` |149754|EE02FC4F8C901D7C0E7F3353FDB3F98215C4F24BE2C63D509757A45803672C91|
| `native/README.md` |64235|C2D5C4030121AFF6C303D3B98A2FC1BB108E6838248F972DB02B642A96D020C6|
| `native/ARCHITECTURE.md` |66440|C48C1F6AA2ECBA0E0B1E4BA52157ABF502599202F9E2F4D7D01F8B1EEC131247|
| `native/VALIDATION.md` |153944|326397C6D177395102967A1C4A99339DFF275B62EF60CB6538BE754C1593B609|

No source, portable README, accepted package/evidence, tests, builds, launches,
workers, captures, benchmarks or resource operations were modified or executed.
