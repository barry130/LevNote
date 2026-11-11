# Native outline acceptance

## ACCEPTED final checkpoint — 2026-11-08 14:03 +08:00

Root accepted the Markdown/mindmap outline implementation, final executable,
isolated startup evidence, and accepted screenshots. The accepted artifact is
source-frozen; no retest was performed after this acceptance.

Delivered app behavior:

- bounded current-tab Markdown/mindmap sidebar outline with runtime-only
  file/outline toggle, virtual rows, keyboard and far-row activation, and
  explicit unsupported/error/empty/partial states;
- fallible Flush plus tab/adapter/path/mode/buffer/controller-generation
  revalidation before resolver and navigation;
- Markdown old-ID resolution through the source journal, UTF-16 source reveal,
  mounted-preview navigation, split navigation, and distinct anchor resolution;
- mindmap source collection without canvas construction, hidden writable-node
  expansion as one Undo transaction, unknown metadata preservation, readonly
  refusal without mutation, and split navigation preserving the source
  selection;
- a real split hidden-node test with a nonzero reversed source range followed
  through expansion, Undo, and Redo;
- 2048-item/512-byte-label bounds, unchanged-frame cache reuse, lifecycle
  release, and a visible partial state beyond the count cap;
- narrow readonly graph layout now reserves tab/mode rows and has a geometry
  regression for the full filename, close control, mode controls, canvas, and
  inspector.

Final accepted checks:

```powershell
native\tools\go\bin\gofmt.exe -w native\app
native\scripts\go.ps1 test ./... -count=1 -timeout=300s
native\scripts\go.ps1 vet ./...
```

The full suite passed in 8.083s wall time, including `dependency-patches`.
Full vet emitted no diagnostics.

Final executable:

- path `F:\AIproject\LevNote\native\bin\levnote-native.exe`;
- bytes `13,143,040`;
- SHA-256
  `D77BC3F45114957416BE83B361873AC3648708E23B308C3395A35D099FF15925`;
- UTC mtime `2026-11-08T06:02:54.6755956Z`;
- `go1.27.3`, Windows amd64/v1, CGO enabled, trimpath, MyGo `v0.8.3`;
- PE32+ x64 Windows GUI subsystem.

Build command:

```powershell
native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

Isolated smoke:
`native\artifacts\acceptance\startup-smoke-outline-20261108-140315`.
The native window opened, graceful close was requested and accepted, exit code
was `0`, and unrelated `.unrelated-data.old-999999` plus
`.unrelated-directory.old-999999\keep.txt` survived with exact contents.

Accepted images under `native\artifacts\acceptance\`:

- `outline-markdown-preview-1180x780.png`
- `outline-markdown-source-1180x780.png`
- `outline-markdown-split-1180x780.png`
- `outline-markdown-preview-dark-1180x780.png`
- `outline-markdown-split-dark-760x480.png`
- `outline-far-heading-760x480.png`
- `outline-partial-760x480.png`
- `outline-mindmap-hidden-1180x780.png`
- `outline-mindmap-hidden-dark-1180x780.png`
- `outline-mindmap-readonly-760x480.png`
- `outline-drawio-unsupported-760x480.png`

The far image shows chapter 90 after real activation, with matching content and
status. Per-mode Markdown images show the activated `今天的记录` target.
The regenerated readonly image has one readonly notice, full tab/close/mode
controls, and a usable canvas/inspector.

`CurrentOutlineItem` is a snapshot/navigation hint, not continuous caret
tracking. The outline/file toggle is runtime-only and is not persisted.

Limitations: MyGo tester Compose verifies preedit transport, not a real OS IME
candidate/language-switch UI; no runtime network feature was packet-capture
tested; the bundled Windows race runner is unavailable and is not claimed.
The 12:55 binary and resource samples remain historical and do not measure this
outline executable.

## Next-stage transfer acknowledgement

The accepted app/plugin outline checkpoint is frozen. For the inactive
schema2 installed-source-outline stage:

- GLM may take `core/plugins/**` schema2 validation/loading, nested rule copy
  tests, and `examples/plugins/**` samples after root activates the stage.
- Astra retains editor contracts and owns the neutral source-outline scanner
  and provider factory.
- DeepSeek retains `app/**` and `modules/**` wiring, including a separate
  `Tab.OutlineFactory` lifecycle path for installed source formats.

An installed schema2 source format keeps `Tab.Adapter == nil`; it must never
enter preview/split/export paths that require `Adapter.Create`. App composition
will convert core plugin rule data to Astra's neutral rule type, so
`modules/sourceview` must not import `core/plugins`. Exact scanner function,
provider factory, and Tab wiring signatures require root/Astra/GLM/DeepSeek
agreement before implementation.

No product source was changed for this design-only stage.
