# Preview packaging source checkpoint

2026-11-09. Immediate three-file source ownership is active. No build, package
creation, worker launch, test or other execution has occurred.

## Saved first source

`native/scripts/package-markdown-worker.ps1` now has three fixed repository
definitions:

- Markdown export: `levnote.markdown-export`, `html`, 8 MiB source, 64 MiB
  payload, existing path and command.
- Mindmap SVG export: `levnote.mindmap-svg`, `svg`, 16 MiB source, 64 MiB
  payload, existing path and command.
- Readonly Mindmap scene preview: `levnote.mindmap-preview`,
  `mindmap-scene-v1`, `engines/mindmap-preview`,
  `levnote-mindmap-preview.exe`, `./cmd/levnote-mindmap-preview`, 16 MiB
  source and 32 MiB payload.

`PayloadMaxBytes` is declared per definition and used for aggregate IPC output
metadata. The worker executable guard remains the separate fixed runtime limit
of 64 MiB for all engines. Existing entry parameters, fresh-directory guards,
build/PE/manifest sharing and release scope wording are preserved, except for
the truthful three-engine scope string.

## Other owned source

`native/portable/README.md` now describes three fixed one-shot workers: two
export engines and one readonly Mindmap scene engine. It distinguishes the SVG
export's “图” image placeholder from readonly scene preview, which loads and
displays neither images nor that badge while preserving references in source. It
states the actual boundary: MyGo native UI renders in the host; editable
Mindmap parsing/layout and source-outline parsing remain in the host; readonly
Mindmap scene parser/layout runs in the fixed worker; and missing preview is
visible with source mode available, never a host-parser/layout fallback. It adds
concise readonly inspector/source-only behavior and the writable Save As
transition back to the editable inspector, and clarifies that the existing tag
editing section applies to editable Mindmap. Preview budgets are 16 MiB source
and 32 MiB DTO.
Version substitution, `%APPDATA%`, local links, actual licenses, no-WebView and
unaccepted-build wording are retained.

`native/scripts/check-portable-package.ps1` now creates nonexecuted fixture
payloads and manifests for all three fixed engine paths. It independently lists
all 13 required attribution/license relative paths, verifies exact path coverage
and compares actual bytes/digests. Total manifest counts are derived from actual
inventory rather than stale prose. It still copies the real packaging script
only to reject inputs before execution/compilation; fixture worker bytes are
never executed.

No build, test, package creation, worker or product launch has occurred. Static
source review found no need to change `portable-package-lib.ps1`.

## Corrected captured source inventory

Captured after the three owned source edits and review corrections:

- `native/scripts/package-markdown-worker.ps1` — 9,067 bytes, SHA-256
  `FD5C6E18394135759EF8F485209F9895CDEB6C733C18571014C679C245F47F28`.
- `native/scripts/check-portable-package.ps1` — 10,831 bytes, SHA-256
  `5A4ACE56339440446BFCB0B3F1306F0F6EB021A84E6EDB6679428D46A2496736`.
- `native/portable/README.md` — 5,702 bytes, SHA-256
  `E895D1AFDC640B5A95782780651208159E7E50869C6B9D3A9833C92C07C02FB0`.

No pre-edit hashes were retained for these files. This is a source checkpoint
only, not a validation result.

## Asset-only verification

Root accepted the fixed caps and independent 13-path oracle. The remaining
readonly paragraph was corrected to the exact semantics: scene preview supports
zoom/pan/selection/outline navigation; properties, tags and image references
remain in source; writable Save As is required before editing; source mode does
not clear readonly; SVG export remains directly available; and the tag section
is explicitly scoped to editable Mindmap.

Final captured input hashes after that correction:

- `native/scripts/package-markdown-worker.ps1` — 9,067 bytes, SHA-256
  `FD5C6E18394135759EF8F485209F9895CDEB6C733C18571014C679C245F47F28`.
- `native/scripts/check-portable-package.ps1` — 10,831 bytes, SHA-256
  `5A4ACE56339440446BFCB0B3F1306F0F6EB021A84E6EDB6679428D46A2496736`.
- `native/portable/README.md` — 5,823 bytes, SHA-256
  `347B8D069BC0AB376606F837437F1D765DFCF93843AB9D2749C55BBB63B830A3`.
- `native/scripts/portable-package-lib.ps1` (read-only fixture input) —
  6,982 bytes, SHA-256
  `2C94A6BE5F8B619406C02C2AC88E1464EC192B91BDFF80A0CF4FD7A56D81A508`.

One authorized asset/path fixture run completed on 2026-11-09 at
08:16:16.0979496–08:16:17.0675476 +08:00:

`pwsh -NoProfile -File native\scripts\check-portable-package.ps1 -EvidenceDirectory F:\AIproject\LevNote\native\handoffs\evidence\preview-packaging-fixture-20261109\fixture`

Exit code was 0. The fixture result was `PASS`, 25/25 checks; stdout was 4,747
bytes and stderr was 0 bytes. Raw command metadata, streams, fixture
`result.json`, before/after hashes for the four owned/fixture inputs and all
13 copied project/attribution/license assets are retained under:

`native/handoffs/evidence/preview-packaging-fixture-20261109/`

Fixture worker payloads are explicit nonexecuted byte strings. No Go command,
product build, worker launch, host UI/resource launch, or prior bundle/evidence
mutation occurred. Packaging source ownership is relinquished after this
checkpoint for later Astra delivery.
