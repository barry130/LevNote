# Schema3 app source checkpoint

2026-11-09. Focused workflows, three760x480 captures, complete app tests and
scoped vet passed. Exact final evidence: `format-routing-app-result.md`.
Core gate is accepted in `format-routing-core-root-acceptance.md`.

`configureTab` retains the installed manifest identity and declarative source
highlight, selects the fresh `modules.NativeRouteAdapter`, and leaves schema2
OutlineFactory nil. Native semantic outline/export/readonly-scene descriptors
remain adapter-owned. Missing host capability falls back visibly to source;
readonly fallback says view source / Save As writable copy.

`App.NewFromPlugin(id) bool` uses Manager's normalized first suffix and the
shipped template. The common add path has a private post-Flush/pre-release
validation callback; origin tab/document/buffer/configuration generation and
fresh enabled route + actual matching manifest are checked before append.
The receipt also checks the originating Editor and View, retaining a replacement
input/view when a synchronous Flush callback changes ownership.
Shadowed first suffixes are refused rather than replaced with another suffix.
Install rechecks all Flush barriers after the open dialog returns.
SaveAs also rechecks owner/tab/view/mode/route and Flush after its dialog,
before publication. A nested successful native field write reaches disk and
remains undoable; a staged tag/refused draft or replaced owner publishes nothing.

The reviewed narrow core amendment is `plugins.ErrEnabledNotLoaded`, wrapped
only after settings persistence + reload. App reconciles views on that error;
ordinary persistence failure retains the current view and registry. Tests
exercise actual removed/corrupt installed files and a failing settings target.

Normal controls say “使用 Markdown/思维导图”; name/ID and target stay visible in
the plugin panel, with schema/route in tooltip details. Enabled native-route
rows expose “新建文档”. Two example local JSON files are in examples/plugins/.

Files: app/app.go, plugins.go, view.go; core/plugins/registry.go sentinel;
examples/plugins/task-notes.json and thoughts.json. No engine/runtime/concrete
format/editor/package files changed. Real installed workflows, Flush and
same-ID late completion,760x480 plugin/document captures, full app + scoped vet
are complete. Source/tests/examples are frozen for root review; no real worker,
package build or UI process was launched.
