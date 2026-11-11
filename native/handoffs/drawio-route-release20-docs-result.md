# Draw.io route `.20` current documentation

Status: **CANDIDATE READY FOR ROOT REVIEW** 2026-11-10.

Only the eight task-owned product documentation files were edited. No product
source, package, portable asset, manifest, evidence, or historical acceptance
artifact was changed; no Go test, build, launch, capture, or resource
measurement was run.

## Scope

- [README.md](../../README.md): current launch path is `.20`; schema3 now has
  exactly `builtin.markdown`, `builtin.mindmap`, and `builtin.drawio`.
- [ROADMAP.md](../../ROADMAP.md): current plugin/integration row now names the
  three schema3 routes and accepted `.20` identity/gate; `.19` worker evidence
  remains inherited runtime evidence.
- [CHANGELOG.md](../../CHANGELOG.md): added the user-facing `.20` route and
  package entry; retained `.19` as dated history.
- [native/README.md](../README.md): current launch/build/run guidance is
  `.20`; corrected four-engine/single-lane/current-page-worker prose; updated
  route capabilities, examples, and accepted identity.
- [native/ARCHITECTURE.md](../ARCHITECTURE.md): added `builtin.drawio` to the
  closed schema3 route set and updated current `.20` package/evidence facts.
- [native/VALIDATION.md](../VALIDATION.md): added the accepted `.20` route
  gate and package identities; retained `.19` as explicit history and inherited
  installed-worker evidence.
- [native/formats/API.md](../formats/API.md): added
  `NativeRouteDrawio = "builtin.drawio"`, three exact routes, the draw.io
  example, capability truth, and `.20` evidence links.
- [native/examples/plugins/README.md](../examples/plugins/README.md): all
  three schema3 examples are supported by `.20`; removed the later-build
  wording.

The `.diagramnote` route is described as native draw.io editing/preview plus
current-page SVG with no Outline or Scene capability. It does not convert
Markdown, mindmap, or other file bodies; SaveAs only rematches by the new
path. The documentation does not attribute the parallel warning or
source-retention source increments to `.20`.

## Corrected stale guidance

- `native/README.md` no longer says Markdown and mindmap exports alone share
  the slot, that there are three workers, or that draw.io SVG is host-generated.
  Current prose says four fixed engines/workers share one task slot and lane;
  draw.io current-page SVG runs in the fourth fixed worker.
- Current two-route schema3 references now list all three exact data routes.
- Current `.19` launch/acceptance language was changed to `.20`. Genuine dated
  `.19` and earlier histories were preserved rather than globally replaced.
- Root review found one remaining current pointer at
  `native/formats/API.md` saying “当前`.19`交付”; it now says
  “当前`.20`交付” while the surrounding historical evidence is retained.
- Final root note separated the export slot from the execution lane: the three
  export engines share one export job slot, while all four engine operations
  share one execution queue.

## Validation

One scoped local check was run after edits:

- Parsed all eight touched Markdown files and checked every ordinary local
  relative link target for existence.
- Parsed the three schema3 example JSON files with PowerShell
  `ConvertFrom-Json`.

Final result: all three JSON examples PASS and all checked local links exist.
During drafting, the check initially found two newly added links to a
nonexistent `drawio-route-result.md`; those references were corrected to the
actual
[drawio-route-implementation-result.md](drawio-route-implementation-result.md).
After root's stale-pointer correction, the same scoped link check was rerun once
and passed. It deliberately skips absolute web/mail URLs and pre-existing
angle-bracket path literals in historical VALIDATION prose that contain spaces.

No Go tests, builds, launchers, captures, worker requests, or resource
measurements were run.

## Final touched files

| File | Bytes | SHA-256 |
|---|---:|---|
| `README.md` | 18,373 | `8B04A086A9766CECE66D92D36114EF7720E7116CF95D0A21E41B3D6A05C60008` |
| `ROADMAP.md` | 119,816 | `38F3CD0E0920CE1910E0943121359F77F05C0D866C601BCE9465D5F808889A48` |
| `CHANGELOG.md` | 145,338 | `E16E0F8770ABA7F67353E9BDA78551592980C50881882DA5A7CD2D20C8C4AC5D` |
| `native/README.md` | 59,948 | `2E5C6E6339038C282EBA5DDE203B340483474A917B7649717C5083AE974919FD` |
| `native/ARCHITECTURE.md` | 62,256 | `D35EFDE76FCA7E76DB18910501F4DCB37540FC616147155F5ACD23BC6DAF0010` |
| `native/VALIDATION.md` | 140,691 | `F7A17D2D9A045E2824F1552DD7D730BF84B77199A2B519DB44AD28315B88118A` |
| `native/formats/API.md` | 53,810 | `0DBFADEB41661904FE3EA02BBDE60D2B0470FEF388992B370B3AC9E55339998C` |
| `native/examples/plugins/README.md` | 4,806 | `43F280AB5E5E1F4C550FD5D85DDD67FAEFF19B290CB70CCC91FE2029A0537573` |

## Acceptance boundary

This handoff does not self-accept the documentation. Freeze it for root review.
