# Fourth-worker portable documentation: preparation result

Status: portable README prepared and frozen for later `.19` candidate delivery.
Final Stage2 UI wording and behavior still require root confirmation before
packaging. This is not a `.19` package or acceptance claim.

Active task:
[`drawio-worker-portable-docs-task.md`](drawio-worker-portable-docs-task.md).

## Sources used

The wording is based on the accepted stage1 lower-layer implementation and
acceptance, the accepted neutral editor/graphview contract, and the specified
Stage2 behavior. These records establish the fourth fixed engine identity,
exact selected-page index and ID, schema1 loading/schema2 persistence, 32 MiB
source limit, 64 MiB SVG output limit, 30-second runner timeout, and warning
limits.

The changing Stage2 App was not treated as final implementation. Exact visible
button/tooltip text, engine-panel labels, and reset wording remain pending the
frozen Stage2 result. In particular, the prepared README uses the functional
phrase “重置全部四个选择”; root must reconcile it with the final UI wording.

## Changed product documentation

Only `native/portable/README.md` changed:

```text
3A1CE0C55572339BA857925E27D69F1B93C69090F93E8FAE550F96D96D1E52BC  11423  native/portable/README.md
```

It now describes:

- four fixed engines, including `levnote.drawio-svg`;
- one one-shot draw.io SVG worker for the exact selected page, with no silent
  first-page fallback;
- source-mode refusal and the instruction to use a rendered preview or split
  view with a selected page;
- host-owned editable mindmap/draw.io behavior versus worker-backed read-only
  mindmap presentation and draw.io SVG export;
- an immutable local source snapshot, untouched original files, and
  non-overwriting final publication;
- 8/16/32 MiB source budgets for Markdown/mindmap/draw.io, 64 MiB output
  budgets, the default 30-second timeout, and draw.io's 256-warning plus
  metadata byte limits;
- explicit import, activation, and reset behavior for all four fixed engines;
- schema1 old-three load without rewrite, draw.io defaulting to bundled on
  load, and schema2 persistence with exactly four engines after an explicit
  selection;
- per-engine retention of 12 packages, 48 total, with no deletion UI;
- preserved cleanup-poison, whole-directory distribution, integrity-only
  hashes, no-sandbox, and fixed-first-party-engine limitations;
- no whole-application memory or process-peak claim.

The exact template placeholder remains `{{VERSION}}`, present once. No `.19`
version is hardcoded.

The provenance scope did not change. The fourth worker is first-party code and
introduces no dependency, so these files were intentionally preserved:

```text
68895470CBF23E7D81AB619771953254E445E2B32DFF143A84B62A549EDDBB81   492  native/portable/NOTICE-PREAMBLE.md
41A9F7B6526B85576F5E44263211A2EA61B3371A112A66F772A7A62A7A1BA8B2  6390  native/THIRD_PARTY_NOTICES.md
```

## Scoped link check

One read-only link check resolved all five relative README targets against the
existing accepted `.18` output layout:

```text
attribution/
LICENSE
licenses/
release-manifest.json
THIRD_PARTY_NOTICES.md
targets=5 missing=0
```

Exit was `0` and stderr was empty. This check did not read or modify the
generated package copies. Raw command, streams, and exit are in
`native/handoffs/drawio-worker-portable-docs-evidence/`; the evidence manifest
is:

```text
C96EA8AB83B686EE3BFC8A2C8E86EEFF308B1B9DB6194A1EFE595CEAB62D8E69   395  evidence-manifest.sha256
```

## Limits and freeze

No tests, build, packaging, worker/package invocation, host launch, UI action,
or resource measurement ran. The accepted `.18` package is unchanged, and no
fourth-engine package is claimed.

The README is frozen for root review alongside the pending Stage2 result. After
Stage2 is accepted, root must reconcile the final UI wording and any actual
behavior differences before using these assets to build `.19`.
