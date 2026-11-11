# Portable engine-package instructions — active docs only

2026-11-09. GLM read_native_handoffs_editable_engine_next_design_task_md owns
ONLY native/portable/README.md and a handoff. Its managed_bundle_test.go is
accepted source-ready and frozen; no tests/Go/script/package-file edits now.
No build/process/browser/subagents. Root reviews while Astra tests app.

Update portable README's affected paragraphs for actual current app source
(engine_packages.go/UI, pluginPanel, managed runtime) before .9 packaging:

- Plugins has Format configurations and Engine packages, not the old single
  Plugins->Install wording alone. JSON remains nonexecuting schema1/2/3 data.
- Fixed3 supported engines: MarkdownHTML, mindmapSVG and readonly scene;
  local complete release-directory import copies selected executable+required
  notices; import never activates; choose exact installed version/digest to
  activate, restore bundled explicitly. Unknown IDs/extra capabilities refused.
- Show three action semantics in simple Chinese: import, enable, restore;
  what the user will see during background/cancel. Corrupt state explicit ALL3
  reset preserves installed packages. No silent fallback when selected files
  fail verification. Old packages retained,12/engine and no delete UI.
- State private store location %APPDATA%\LevNoteNative\engine-packages, separate
  from moved program directory; configuration moves are not automatic. Startup
  reads bounded preferences, installed scans happen on explicit check; don't
  overpromise every I/O is asynchronous since startup metadata and final export
  publication have existing bounded/synchronous portions.
- Source/digest vs executable checksum: receipt consistency only, no publisher
  authentication or sandbox. Local executable activation disclosure is concise.
  Repository first-party fixed engines are supported; not arbitrary executable
  plugins. No legal immunity or new memory/performance/OSIME proof claim.
- What remains host-native: editable document models/layout, outline, drawio;
  readonly mindmap scene is separate worker, JSON routes use existing formats.
  Engine switch preserves writable drafts/undo and invalidates readonly scene.

Keep {{VERSION}}, exact licenses/local links and unrelated editor instructions.
Do not replace historical source README launch docs until actual .9 acceptance.
Publish handoffs/fixed-engine-package-portable-docs.md with changed sections and
source-only status; no commands/checks needed for a small Markdown change.
