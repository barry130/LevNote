# `.23` product documentation result

Completed `native/handoffs/native-image-release23-docs-task.md` on 2026-11-10.
Only the six owned product Markdown files were changed. Root-requested narrow
wording corrections were applied after review: the `.22` VALIDATION subsection
now says `历史入口`; byte fallback is limited to files that cannot decode but
can still be read, while read failures are reported truthfully; architecture notes
known transient old/new baseline coexistence during open/reload/SaveAs.

## Changes

- `README.md`: current package/acceptance references moved to `.23`; added concise
  user-facing PNG/JPEG reading, original-byte copy, byte fallback, plugin switching,
  mixed-session restore, support limits, and the `.23` acceptance/delivery links.
- `ROADMAP.md`: current integration/package rows moved to `.23`; added the accepted
  limited PNG/JPEG reader row; changed the fixed12 matrix row from pending to its
  truthful mixed 12/12 descriptive local result.
- `CHANGELOG.md`: added the unpublished `.23` image-reader/delivery entry, including
  the 13/13 acceptance and evidence boundary.
- `native/README.md`: current path, examples, package paragraph, and accepted-version
  wording moved to `.23`; added a local image-reading section and capability row with
  exact support limits and no internal queue/implementation labels.
- `native/ARCHITECTURE.md`: added the current binary/image section covering immutable
  accepted state, separately owned immutable snapshots, safe reader access, serialized
  ownership-checked work, active-only decoded presentation, cancellation/barriers, and
  resource-release boundaries; moved the current-delivery wording to `.23`.
- `native/VALIDATION.md`: added the `.23` current-delivery section once with exact
  package/host/manifest identities, 26-file package size, 25-entry self-excluding
  manifest, 35 verified guarded source/build inputs (not shipped assets), 14 portable
  copies, 13/13 gates, evidence boundary, and immutable autosave limitation; retained
  `.22` below it as dated history.

## Scoped check

Command:

```powershell
rg -n "JJPEG|PNG/JJPEG|J/JPEG|当前.*\.22|\.22.*当前|20261110\.22\\levnote-native\.exe|20261110\.22/README|tab-workflow-release22-root-acceptance|tab-workflow-release22-delivery|本地PNG/JPEG阅读|draw\.io worker资源矩阵|path-only|single-lane|单车道" README.md ROADMAP.md CHANGELOG.md native/README.md native/ARCHITECTURE.md native/VALIDATION.md
```

Actual output:

```text
ROADMAP.md:26:| 本地PNG/JPEG阅读 | ✅ `.23`有限子集 | ... [包验收](native/handoffs/native-image-release23-root-acceptance.md)/[验证记录](native/VALIDATION.md) |
ROADMAP.md:42:| 宿主＋四个引擎目录交付 | ... 当前`.23`已接受，`.22`及更早历史包保持不变；...
ROADMAP.md:60:| draw.io worker资源矩阵 | ✅ 12/12描述性证据 | ...
native/VALIDATION.md:40:[bin/levnote-native-worker-20261110.22](bin/levnote-native-worker-20261110.22/README.md)，
native/VALIDATION.md:91:[包验收](handoffs/tab-workflow-release22-root-acceptance.md)、
native/VALIDATION.md:92:[交付](handoffs/tab-workflow-release22-delivery.md)和
```

Result: no typo/internal-label match; no stale `.22` path/current wording. The
three ROADMAP matches are the accepted `.23` rows; the three VALIDATION matches
are in the retained `.22` history section.

Link verification actual output:

```text
native/handoffs/native-image-release23-root-acceptance.md  EXISTS
native/handoffs/native-image-release23-delivery.md  EXISTS
native/handoffs/native-hidden-autosave-review.md  EXISTS
native/handoffs/drawio-worker-resources-fixed12-root-acceptance.md  EXISTS
```

## Final identities

| File | Bytes | SHA-256 |
|---|---:|---|
| `README.md` |15,340|`678BBFD3A30E336C72184267C87D29F01D966C8B13108CD943AE47D10110BDF8`|
| `ROADMAP.md` |121,001|`C6719946CB4D9622098E55E5F7FB55487EA814E7CDFE92EA7FA61C8217DAC891`|
| `CHANGELOG.md` |148,384|`FCBE997BBBABEF5921E8AE27C8F0986EEEEE6AD34883323C16DA669672F0301D`|
| `native/README.md` |63346|`10E28C23BC86D58D0BE865E6C51ADF4CBBFAF8F0D7D6F6015506909BEC6FB2FA`|
| `native/ARCHITECTURE.md` |65,967|`FE484438A2D0FFB1AC5B3B965C57201B20E6AFFDA35AE6D1F71496B526560764`|
| `native/VALIDATION.md` |151,417|`49C4379EB51D3FBF90C477697761A26F2C71FBDAA316152DA8A2E7CA29B4BD95`|

No product source, portable README, accepted bundle/evidence, tests, builds,
launches, workers, captures, resource tools, or other runtime operations were run
or modified.
