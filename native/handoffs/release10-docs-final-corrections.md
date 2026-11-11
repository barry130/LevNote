# Final small corrections to .10 docs

2026-11-09. GLM documentation owner only; keep the existing seven-file scope.
Root read the actual files after your completed result. Correct these remaining
factual/wording issues, then amend release10-docs-result.md. No Go commands,
implementation, build, launch or release-directory edits.

1. Root README has a literal doubled separator in the inline configuration path:
   `%APPDATA%\\LevNoteNative`. Use one separator, as native README already does.
2. CHANGELOG's `.8` paragraph still says `当前入口` before its `.8` directory.
   Change that historical sentence to `当时入口`.
3. native README lines around312-314 globally say viewing the plugin panel does
   not scan workers and validation only checks adjacent programs. Distinguish
   the lazy export-result status from the new engine-package inventory, which
   does validate installed package files. Neither listing starts a worker.
   Validation can apply to selected installed or bundled files. Retain the
   honest integrity-versus-authentication statement without duplication.
4. ARCHITECTURE's final local-package paragraph should say an installed package
   needs explicit activation, while bundled engines are selected by default;
   either selected worker starts only on an actual export/preview request.
5. API wording: the manager's operations accept only fixed catalog engine IDs;
   its constructor accepts host/state directories. `不接收任意命令、路径` is too
   broad because import accepts a release directory. State no arbitrary
   executable-path/command overrides. Apply the same precise term where the
   .10 changelog's new package paragraph says `任意路径`.

Use one source-only check of changed links and exact corrected wording. Don't
rerun the seven-file inspection if only a targeted subset changed and existing
checks still apply. Root will accept from the corrected files and result.
