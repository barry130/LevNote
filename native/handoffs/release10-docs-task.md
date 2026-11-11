# .10 product documentation: prepare, then publish after acceptance

2026-11-09. GLM documentation owner. Root delegates all product documentation
edits. No implementation/tests/builds/launches, no children. Compact before200K.
DeepSeek owns the runtime correction and .10 delivery; never touch its files.

Read native/AGENTS.md, fixed-engine-package-app-root-acceptance.md,
engine-package-volume-root-fix-and-release10.md, app/engine_packages.go,
core/engineruntime/package_types.go and existing relevant docs. Prepare concise
draft updates in handoffs/release10-docs-draft.md now, then report completion.
Do not alter product docs until root accepts the actual .10 result; .8 remains
the current accepted runnable release. No need to wait/poll or infer acceptance.

After a root follow-up authorizes publication, own only README.md, ROADMAP.md,
CHANGELOG.md, native/README.md, native/ARCHITECTURE.md, native/VALIDATION.md and
native/formats/API.md. Do not modify bundled README templates/notices or built
release directories. Use actual delivery identities/evidence, no guessed bytes.

The user-facing root README native introduction has become an 80-line audit
log. Replace that native preface before the legacy heading with a concise useful
entry point (roughly30-45lines): pure-local Go/MyGo Surface, the four format
families and finite capabilities, direct current launch command, keep whole
directory, config location, two plugin categories, links to detailed native
guide/capabilities/validation/notices. Preserve legacy content distinctly.
Historical benchmark percentages, attempt numbers and exhaustive hashes belong
in validation/handoffs, not the top-level introduction.

Explain .10 actual engine package workflow: native Plugins -> engine packages,
choose one of fixed3IDs; import a complete compatible local release, import
doesn't activate; explicit activate / reset to bundled; current exact source,
version/digest and errors remain visible; one background operation with reopen
and cancel even when panel hidden. Installed sources in
%APPDATA%/LevNoteNative/engine-packages;12retained versions per engine, no delete
or arbitrary executable path/plugin API. Local JSON schemas1/2/3 remain separate
nonexecuting format configuration. Don't conflate schema3 with worker packages.

Explain why transitions wait for owned in-flight work and preserve writable
drafts/undo; corrupted selections refuse execution, ALL3 reset is explicit and
retains packages; a cleanup failure can occur after selection persistence, so UI
shows actual state. Import/list remain available after cleanup poison whereas
execution/selection transitions are blocked. These details go into architecture/
API, not a wall of first-run cautions. Hashes are local integrity, no publisher
authentication or sandbox. Known format/IME/resource limits remain accurate.

Validation should record .9 failed managed-before-worker and confirmed Mkdir
volume-root defect, narrow correction/regression, actual .10 managed two-request
installed .8 -> bundled .10 verification, cross-volume C: isolated config/F:
host normal close, reused source-matched4tester images, exact accepted bytes/
hashes/evidence. No physical UI or new resource claim. Keep old evidence history
but don't retell it in every doc. Use existing documented anchors appropriately.

After publication perform a small documentation consistency/link check only;
no Go tests for docs. Publish changed file list and check result to
handoffs/release10-docs-result.md. Do not commit or publish externally.
