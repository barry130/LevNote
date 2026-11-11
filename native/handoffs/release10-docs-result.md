# Release 10 product documentation result

2026-11-09. Published after root accepted `.10` with 14/14 delivery checks.
The seven assigned product documentation files were updated to the actual
accepted delivery. No implementation, test, build, launch, bundled README
template, notice, release directory or source file was changed.

## Changed files

- `README.md`
- `ROADMAP.md`
- `CHANGELOG.md`
- `native/README.md`
- `native/ARCHITECTURE.md`
- `native/VALIDATION.md`
- `native/formats/API.md`

## Published content

- Root native preface reduced to a concise entry point (35 lines before the
  legacy heading), with the `.10` launch command, portable-directory/config
  guidance, finite capability scope, and links to native details.
- `.10` identities and evidence propagated from the accepted delivery:
  `native-worker.20261109.10`, 23 files / 23,622,214 bytes, host 13,998,592
  bytes / SHA256
  `4EB096FAEA7CA79E71F0C9FF30E659478C0E2AA135803DDBF7C0AEE47A0AEF6C`.
- Local engine-package workflow documented across README, native README,
  architecture, API, roadmap and changelog: fixed three-engine catalog,
  non-activating import, exact-digest reuse, explicit activation/reset,
  12-version retention, background/cancelable work, installed store location,
  and no arbitrary path/command/engine plugin API.
- Selection/package failure semantics documented precisely: old work joins
  before persistence; writable drafts and undo remain; cleanup poison blocks
  execution/transitions while import/list remain available; corrupt selected
  package refuses execution without fallback while healthy alternatives/reset
  remain usable; corrupt selection state requires explicit all-three reset.
- Runtime/API snippets now include `EngineInfo.Source`,
  `EngineInfo.PackageDigest` and `NewManaged`, and distinguish bundled host
  paths from explicitly selected installed package paths.
- `.9` volume-root failure and focused open-first correction retained as
  history. `.10` managed request evidence records the in-memory 20-byte source
  and exact package/output identities; hidden host evidence records the
  preserved 60-byte file. `.8` wording uses “当时入口” and remains historical.
- Real-window UI pilot limitation recorded in `native/VALIDATION.md` and
  `ROADMAP.md`: three bounded Computer Use helper probes failed before launch;
  no package copy, fixture, app launch, real-window observation/input, or
  resource measurement occurred. It is a helper availability limitation, not a
  product UI failure.

## Source-only consistency/link check

Command scope: a PowerShell source-only Markdown target/anchor inspection over
the seven changed files. It resolved relative links and normalized heading
anchors; no Go test, build, application launch, or product execution ran.

Result:

```text
Link check: PASS=7/7
```

Additional source checks found:

- zero occurrences of the incorrect host hash prefix
  `4EB096FAEA7CA79E71F5C9FF`;
- zero literal doubled Windows backslashes in launch-code paths;
- no remaining “current `.8`” wording;
- exact accepted host/package/output identities present;
- helper limitation present only in the requested `VALIDATION.md`/`ROADMAP.md`;
- root native preface is 35 lines before “## 旧 Tauri 版（legacy）”.

## Final root-requested corrections

After the first result, root requested five narrow factual/wording corrections
in the same seven-file scope. All were applied:

- Root README now uses one separator in `%APPDATA%\LevNoteNative` and says there
  is no arbitrary executable-path or command override or engine plugin API.
- CHANGELOG's historical `.8` paragraph now says “当时入口”, and its `.10`
  package paragraph uses the precise executable-path/command-override wording.
- native README distinguishes the lazy plugin export-result status from the
  engine-package inventory: installed-list/import operations validate package
  files, bundled files are verified on actual requests, and no listing starts a
  worker. The integrity-versus-authentication statement is retained without
  duplicating it.
- ARCHITECTURE states installed packages require explicit activation, bundled
  engines are selected by default, and either selected worker starts only on an
  actual export/preview request.
- API states `NewPackageManager` accepts host/state directories while manager
  operations accept only fixed catalog engine IDs; import accepts a release
  directory, and there is no arbitrary executable-path or command override.

Targeted source-only check after these corrections:

```text
Correct APPDATA separator: PASS=True
Historical .8 entry wording: PASS=True
Native README inventory validation: PASS=True
Architecture selection semantics: PASS=True
API fixed-catalog wording: PASS=True
Changed-link check: PASS=True/5
```

The checker normalized whitespace for wrapped phrases. It checked links only in
the five changed files, as requested; the prior 7/7 source-only inspection still
covers the unchanged docs. No Go command, implementation, build, launch, or
release-directory edit was performed.
