# Immediate packaging source checkpoint

You were interrupted after no source/output progress was visible. Resume with
only this narrow action; do not repeat broad exploration.

Read native/scripts/package-markdown-worker.ps1. In its fixed engineDefinitions,
add PayloadMaxBytes=64MB to Markdown and Mindmap SVG, add the third exact preview
definition from core/enginecatalog/catalog.go with16MB source and32MB payload.
Use definition.PayloadMaxBytes in aggregate instead of64MB; scope says two export
engines and one readonly scene engine. No other behavior change. Save this source
first, send root its path immediately, then complete portable README and asset
fixture edits from native-preview-packaging-task.md.

No test/build/execution. Exact three-file ownership unchanged. If unavailable or
blocked, write one line to native/handoffs/native-preview-packaging-source.md
explaining actual cause and stop. Do not spend more time on design or history.
