# Snapshot scope review — corrected input set 02

2026-11-11. Static source inspection only. No Go/list/test/vet/build, syntax
check, application, screenshot, worker or resource run was performed. The
archive creators only copied file bytes, made ZIPs and inspected hashes.

## What the snapshot covers

The 02 snapshot contains 1,208 repository-relative files / 17,045,848 bytes.
It conservatively includes complete native app, cmd, core, dependency-patches,
examples, formats, licenses, modules, portable, scripts, ui and vendor trees,
except the precise historical capture-output directory described below. Seven
additional files are LICENSE, README.md, native/README.md,
native/THIRD_PARTY_NOTICES.md, native/go.mod, native/go.sum and the existing
picker key-correction source-frozen.json. This is an included superset, not a
claim that every included package/test is selected by `go test ./app`.

Current working-tree bytes are copied directly. Git, checkout, generated
replacement source, dependency downloads and re-vendoring are not involved.
BOMs, line endings and binary fixture contents are preserved. The manifest
uses repository-root-relative forward-slash Path, decimal Length and SHA256.
Windows ACLs or cross-platform filesystem semantics are not a byte-identity
claim. Remote extraction assigns ordinary source-file permissions.

## Actual static references reviewed

- `app/app.go` imports native document, enginecatalog, engineruntime, plugins,
  search, session, text, modules and ui/editor plus vendored MyGo packages.
  All these complete implementation trees are included. No app test/helper
  import or embed requiring the old React/Tauri source was found.
- `app/app_test.go:newTestApp` uses caller-supplied test temp configuration and
  cleanup. App text, graph, image, encoding, recovery and picker tests construct
  fixtures in source literals/byte slices and t.TempDir paths. Generated PNGs,
  temp documents, temp configs and inert engine package fixtures are outputs
  of future tests, not missing repository fixtures.
- `app/acceptance_test.go:writeAcceptanceFixture` constructs all its fixture
  files from in-source constants. Its runtime fixture and screenshot tests
  already require LEVNOTE_ACCEPTANCE_RUNTIME_DIR / SCREENSHOT_DIR opt-ins.
- `app/export_worker_test.go:controlledExporter` and related app worker tests
  use the controlled runner seam. Real bundled worker tests require existing
  LEVNOTE_WORKER_BUNDLE_ACCEPTANCE=1. The separate refusal test in
  export_worker_bundle_test.go requires nonempty LEVNOTE_WORKER_BUNDLE_REFUSAL.
  Both are disabled by the clean environment, preserving existing test policy;
  no new skip or assertion change was introduced. Historical host/worker
  binaries are not prerequisites for this default app gate.
- Full in-package app test helpers are included, including Windows-tagged
  `engine_packages_store_test.go`, whose fixture writes inert executable text
  and attribution files into t.TempDir; it does not read a historical package.
- Conservative dependency-test fixture retention includes
  `formats/drawio/testdata/svg-prechange-oracle.json`,
  `formats/mindmap/testdata/presentation/*` (four JSON/mindmap/SVG triplets and
  README), and `examples/plugins/*`. References include drawio/svg_budget_test,
  mindmap/svg_test, graphview/mindmap_presentation_test and plugins tests.
  Those dependency package tests are NOT run by `go test ./app`; their fixtures
  are retained to avoid accidentally splitting source from its authored data.
- Vendored non-Go assets are included: MyGo bridge.js, GL/HLSL/Metal shaders,
  Windows keys.js, l10n JSON/messages and x/net/publicsuffix data. Genuine embed
  directives were distinguished from example comments in API documentation.
- All seven native scripts and six dependency-patch files are retained as
  original bytes, including go.ps1/bootstrap/refresh/package helpers. Only the
  new independent Linux wrapper is proposed for later execution; the original
  Windows helpers are not run or changed.

## Exclusions and correction history

Toolchains, build caches, native/bin, native/artifacts, the stray historical
native/native/handoffs tree, drawio-resources.test.exe, other historical handoffs,
old large validation documents, Git/private configuration and the old web/Tauri
implementation are excluded. All new preparation artifacts, current AGENTS
coordination and the independently written remote preflight report are outside
the input manifest. Only the old 748-byte picker authority is retained from
handoffs; it is an identity dependency, not an execution output for this stage.

01 was byte-correct at 1,212 files / 17,166,862 bytes, but broad module-tree copy
also included four old PNGs in exactly:
`native/modules/markdownview/artifacts/acceptance/markdown-cell-view-20261108-final`.
`table_cell_test.go:TestTableCellScreenshotsLightDark` writes them with os.Create
under an existing LEVNOTE_TABLE_SCREENSHOT_DIR opt-in; it never reads them as
fixtures. Root requested this precise exclusion. All 01 files/ZIP/manifests/
streams and wrapper draft remain unchanged. 02 removes only those four output
PNGs (121,014 uncompressed bytes), not a general artifacts glob. The early
message estimate of 1,305 files was an arithmetic mistake; actual manifests
were 1,212 and then 1,208. No file count was repaired by deleting source.

Each snapshot independently reconciles original-before, copied files, every
ZIP member and original-after by Path/Length/Sha256. Both creator invocations
exited 0. 01 is superseded for scope, not a product test failure. The separate
scope-correction-02.json records the exact four-path difference.

## Linux configuration and limits

The draft fixes GOOS=linux, GOARCH=amd64, GOAMD64=v1, CGO_ENABLED=0. Root accepted
this explicit Linux configuration after reading purego/nocgo.go,
dlfcn_nocgo_linux.go and internal/fakecgo tags. MyGo internal/linux/ffi.go states
its runtime native libraries use purego without cgo. This does not establish
parity with Windows or default Linux CGO selection; no Go file-selection or
compilation command has been run. Actual compatibility remains unverified.

Pango/Cairo remain required. The Linux source's newEngine falls back to stub
when newPango fails; mandatory symbol loading and font-map creation are real
failure routes. The draft records resolved runtime-library bytes and separate
ctypes symbol/font-map/context prerequisites, plus fc-match executable/font
bytes. That is separate-process prerequisite evidence, not MyGo backend,
shaping, rendering, physical keyboard or IME observation. Nonverbose successful
`go test` may suppress logs; absence of a fallback message cannot prove backend.

Known retained platform issues include Windows C: path expectations in
app/reference_test.go, and unguarded uppercase reopening of a real temp path in
TestReopenClosedBoundDedupAndShutdownClearBacking. Four app test files explicitly
use `//go:build windows` (engine_packages_store/lifecycle/capture and drawio
stage2 capture). Linux may exclude these or fail existing behavior assertions.
The draft must retain failures; no skip, source edit or automatic retry is allowed.

The complete source-input manifest is separate from observed system dependencies.
It is not a hermetic closure over Linux, Python, shared libraries or all fonts.
No local/remote product execution or transport has occurred in this preparation.