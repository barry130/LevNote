# Draw.io route examples and portable guidance — complete

Status: **FROZEN FOR ROOT REVIEW** 2026-11-10. The source example, examples
README and future-build portable README now describe the three schema3 native
routes. Production source integration and manifest admission remain owned by
the parallel implementation task.

The accepted `.19` package and direct evidence were not changed. This work did
not run a Go suite, build, product launch, worker, host or screenshot.

## Delivered files

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/examples/plugins/diagram-notes.json` | 179 | `317FB43C43863A092FF7AA346C1A436A26C75A8BE41DC5E74C7F184AAE529A07` |
| `native/examples/plugins/README.md` | 5,024 | `C8EC7A85A5F1A64607453D8EF7D204469546321B4234C02C2943FC8899D103E9` |
| `native/portable/README.md` | 12,469 | `8398D4B5D4821204EEF456D7F392887A06DCA8D27059573EC14D58622FF67AB8` |

`diagram-notes.json` is a complete schema3/API1 manifest with ID
`diagram-notes`, extension `.diagramnote`, and exact route `builtin.drawio`.
It contains no command, executable path, network access or custom syntax.

The two READMEs now:

- show Markdown, mindmap and draw.io as the three source routes;
- include the complete `diagram-notes.json` inline in both future-build views;
- state that the accepted `.19` bundle supports only Markdown and mindmap
  schema3 routes and that draw.io suffix routing needs a later build;
- explain that the file must already contain draw.io XML and that a suffix
  change does not convert another format;
- describe native draw.io editing and selected-page SVG export without
  claiming outline or scene capabilities;
- send source-mode export through the visible reading/editing/split view and
  selected-page identity checks;
- state the existing supported subset, visible warnings, exact no-edit bytes,
  and preservation of unrelated/unknown XML metadata for supported edits;
- retain the separate configuration enable state and normal pre-operation
  native draft Flush rules;
- distinguish installed JSON configuration data from imported worker packages.

The portable README remains explicitly a future-build template. It retains its
single `{{VERSION}}` placeholder and all five existing local package/notice
links. No `.20` release or package is claimed.

## Narrow validation

The retained validator ran once through PowerShell 7:

```text
pwsh.exe -NoProfile -ExecutionPolicy Bypass \
  -File native/handoffs/drawio-route-examples-evidence/validate.ps1
```

It exited `0` in 0.789 s with 1,470 stdout bytes and zero stderr bytes. All
**7/7** checks passed:

1. All six example manifests are strict UTF-8 and structurally parse.
2. `diagram-notes.json` has the exact schema3/API1 identity, one extension and
   `builtin.drawio` route.
3. The portable template retains exactly one `{{VERSION}}` marker.
4. Its three inline JSON blocks parse and route to Markdown, mindmap and
   draw.io in order.
5. All three examples README relative links resolve.
6. All five existing portable package/notice links resolve in the frozen `.19`
   package.
7. The portable template explicitly separates accepted `.19` support from the
   pending later-build draw.io route.

Evidence:

| File | Bytes | SHA-256 |
|---|---:|---|
| [validate.ps1](drawio-route-examples-evidence/validate.ps1) | 5,214 | `ACF667FD4FB468F0353DD9BB40248A7C63E349CC7DD8EF0B504C5AC691189C17` |
| [validation stdout](drawio-route-examples-evidence/validation.stdout.log) | 1,470 | `0A5D9A0518761F196ED23CBDBE68B7F96205A222291DA910687C2C793E6E3369` |
| [validation stderr](drawio-route-examples-evidence/validation.stderr.log) | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| [validation invocation](drawio-route-examples-evidence/validation-invocation.json) | 764 | `4E46981FA4A870F7AB6263348634E111B08BFB358831E190DC98247CAA688994` |

This check proves structural JSON validity, UTF-8 decoding, the version-marker
count and local authored-link presence only. It is not the product's strict
schema3 admission path and cannot establish runtime routing.

## Pending integration boundary

The parallel implementation owner must add and test the production
`builtin.drawio` allowlist, fresh adapter resolution and route lifecycle.
Their accepted source must still establish the documented template, native
editing, selected-page export, source-mode refusal, enable/rematch behavior,
draft Flush safety and no-outline/no-scene capability through the project's
focused suites.

These examples and portable guidance must not be attributed to the accepted
`.19` binary. Packaging or shipping this template requires the later source
acceptance and a separately authorized build. The source files in this handoff
are frozen for root review; no further wording edit was made after validation.
