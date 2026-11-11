# Draw.io fourth worker, stage 1 progress

## 2026-11-10 first compile milestone

Implementation is underway in the authorized lower layers: fixed catalog identity
and limits, typed page wire, runtime-owned page admission, the new one-shot SVG
worker, schema-2 package migration, and the fourth packaging definition.

The first compile-only command was:

```powershell
& .\native\scripts\go.ps1 -GoArguments @('test','./core/enginecatalog','./core/engineipc','./core/engineruntime','./formats/drawio','./cmd/levnote-drawio-svg','-run','^$')
```

It exited 1. The production packages and new worker compiled, but
`core/engineipc/protocol_test.go` still had three positional `RequestMeta`
literals that no longer compile after the additive `Page` field. The raw output
is preserved in
`native/handoffs/drawio-worker-stage1-evidence/compile-milestone-1.log`.

Next: update those existing test literals to named fields, then add focused wire,
runtime, worker, parser-scope, and package-migration regressions before the one
final affected suite and vet. No build, packaging run, launch, app/UI,
graphview, editor, accepted SVG emitter, or product-doc edit has been made.
