# Final wording correction only

2026-11-10 root read actual eight files and verified all handoff hashes. One
precise current sentence still conflates exports with preview:
native/README.md says 四个固定引擎共用一个导出任务槽. The three EXPORT engines
share the export job slot; all four engine operations share the execution lane.
Replace with e.g. Markdown HTML、导图SVG和draw.io SVG共用一个导出任务槽.
Keep next sentence 四个worker共用单执行队列. No other change needed. Update only
that README's final bytes/hash and this result's affected description. Freeze.
No link/example recheck needed: no link, example or runtime change.
