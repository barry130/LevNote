# `.11` current-entry documentation result

2026-11-09. GLM `read_native_handoffs_release10_docs_task_md` completed the
narrow source-only update assigned by
[release11-docs-task.md](release11-docs-task.md). This supersedes the prior
`.10` publication task and preserves all `.10` evidence as immutable history.
No product source, API document, templates, notices, Go commands, tests,
builds, package checks or launches were run.

## Changed files

Only the six assigned current product documents changed; this result handoff is
the additional task output:

```text
README.md
ROADMAP.md
CHANGELOG.md
native/README.md
native/ARCHITECTURE.md
native/VALIDATION.md
```

`native/formats/API.md` was not changed.

## Published update

- Current executable and launch commands now name
  `native/bin/levnote-native-worker-20261109.11/levnote-native.exe`;
  version `native-worker.20261109.11`; 23 files / 23,622,214 bytes.
- Current validation records host 13,998,592 bytes, SHA-256
  `F1D1FC76AC1A5649709B6AD8FDB44A1928174280F766625F7226856493E299B1`,
  and release manifest 4,137 bytes, SHA-256
  `A4E37C4E65B38C7B46ACB1FBC80455B604EE4207E7D3072580BD37DE0103E3E6`.
- The accepted aggregate result is 13/13, with one package invocation, four
  GUI builds, stable 286-file build inputs, and the complete `.10` inventory
  unchanged.
- The `.11` managed integration records exactly two Markdown requests:
  imported `.10` then bundled `.11`; both returned 104-byte HTML matching the
  independent literal oracle, kept the 20-byte source unchanged, and closed
  normally.
- The hidden host evidence records the isolated `F:`/`C:` start, exact PID
  8920, held handle path, owned `WM_CLOSE`, exit 0, no forced cleanup, unchanged
  60-byte source/timestamp, and 417-byte session with SHA-256
  `20050A68832F559159910BE058B01743096658783BB784AEBF9A61E33BDA7F74`.
- The two retained-reference fixes are described only as deterministic
  pointer ownership/lifecycle corrections:
  - `Buffer.replace` clears dead piece-tail slots after coalescing while
    preserving logical content, snapshots, and undo/redo.
  - `removeTab` shifts remaining tab pointers, clears the removed tail pointer,
    then shortens the slice while preserving close/recovery semantics.
- The docs explicitly state that `.11` has no measured RSS or working-set
  improvement and no physical UI/IME evidence. The four 760×480 captures remain
  historical `.10` images and were not claimed as recaptures.
- Historical `.10` wording is preserved: its 14/14 validation, installed `.8` →
  bundled `.10` integration, and hidden-host evidence remain factual history.
- Current capability limits and the pending `Ctrl+G` implementation scope were
  not changed.

## Source-only consistency check

One source-only relevant check was run over the six assigned documents:

```powershell
rg -n "当前.*\.10|\.10.*当前|20261109\.10|当前.*\.11|20261109\.11" README.md ROADMAP.md CHANGELOG.md native/README.md native/ARCHITECTURE.md native/VALIDATION.md
rg -n "284|286|13/13|14/14|F1D1FC76|A4E37C4E|977ee81a|20050A688|4EB096FAEA7CA79E71F" README.md ROADMAP.md CHANGELOG.md native/README.md native/ARCHITECTURE.md native/VALIDATION.md
rg -n "RSS|working set|工作集|指针|死槽位|backing array|removeTab|Buffer.replace" README.md ROADMAP.md CHANGELOG.md native/README.md native/ARCHITECTURE.md native/VALIDATION.md
rg -n "20261109\.11|20261109\.10" native/formats/API.md
```

Relevant linked evidence and entry paths were also checked with
`Test-Path -LiteralPath`: `.11` delivery and root acceptance, its evidence
directory, the `.11` package/README, and the `.10` UI-helper result all exist.

The check found:

- current entries consistently name `.11`;
- `.10` references are historical or evidence statements, including the
  preserved `.10` validation section and changelog entry;
- exact `.11` identity and aggregate values are unchanged from root acceptance;
- `.11` current integration is installed `.10` → bundled `.11`, while `.10`
  history remains installed `.8` → bundled `.10`;
- the two fixes are tied to pointer ownership/lifecycle and explicitly excluded
  from RSS/working-set claims;
- `native/formats/API.md` contains no release-entry references and was not
  edited.

No command was run against Go, builds, launches, packages, tests, or product
source.
