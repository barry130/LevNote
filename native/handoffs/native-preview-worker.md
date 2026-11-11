# Fixed Mindmap preview command

2026-11-09 — command implementation and finite actual test/vet gate complete,
ready for root review. Root accepted the neutral codec directly; no independent
review approval is claimed. This stage creates no runnable bundle.

Owned source is `cmd/levnote-mindmap-preview/main.go` and `scene.go`, with
`main_test.go` and `scene_test.go`. The fixed command reads one preview frame,
parses at the catalog's 16 MiB source limit, uses `NativeLayoutOptions`, preserves
parser/layout/presentation warning order and converts only presentation values
to `enginescene.Scene`. `Encode` enforces its real 32 MiB wire cap before returning
payload. Every refusal has zero payload and bounded constant error text; response
writer failure exits 1 and invalid build version exits 2 before reading input.

Paint normalization uses shared `LiteralColor` admission and expands successful
values to lowercase `#rrggbbaa`; absent/unsupported values remain empty host-theme
roles. Shared `DisplayLabel` keeps its 512-byte prefix plus optional ellipsis.
No source metadata, raw model pointers, image URLs, theme defaults, selection or
host identity enter the DTO. The image warning keeps its code/order/range but
truthfully says this read-only preview does not load or display images.

Thirteen top-level tests passed in the first actual command run:

- Four independently specified layout geometries, edges, inherited colors,
  collapsed hidden children and exact unknown-field warnings; source nonmutation.
- Literal escaped JSON output including Unicode, single/double quote, backslash,
  NUL and newline; empty default paints; all admitted color spellings and
  unsupported/non-string fallbacks.
- Future layout/version and hidden image/label/color warnings in exact order,
  including the truthful image message and whole-source ranges.
- Exact/over-limit labels and IDs; invalid schema and valid excessive depth;
  warning count/code/message/UTF-8/range bounds; cross-engine and source-cap
  refusal before payload reads; invalid frames/build version; short/error writer.
- A legal exact-16-MiB source with independently authored small output.
- Real output-limit refusal: 512 NUL bytes in the root ID, 11,000 leaf children,
  <1 MiB source. Parent IDs alone require `11,000 × 512 × 6 = 33,792,000`
  escaped bytes. Fixture checks no parser/presentation/layout warnings, exact
  `312 × 704032` bounds, valid node/edge counts and local field budgets before
  entering the real command. No injected encoder or artificial cap is used.
- Literal fixed preview descriptor and shared parser/codec limit agreement.

Actual evidence:

- [attempt01 result](../artifacts/acceptance/native-preview-worker-20261109-074600-owned-attempt01/result.json):
  the whole command's 13 top-level tests passed in **1.025 s** (wrapper
  2.422707 s), exit 0 and empty stderr. The real output-limit fixture logged
  **300,077 source bytes / 11,001 nodes / 11,000 edges / 33,792,000-byte lower
  bound**, and returned a coherent zero-payload refusal. Vet exited 1 solely
  because 28 external `Point` literals in the four-layout test omitted X/Y keys;
  the original error stream and result remain intact. All 27 scoped inputs were
  unchanged during the run.
- [attempt02 result](../artifacts/acceptance/native-preview-worker-20261109-074800-owned-attempt02/result.json):
  after adding only those X/Y keys, the affected four-layout test and all four
  subtests passed in **0.576 s** (wrapper 1.6970749 s). Scoped vet passed, exit 0,
  in 0.8816852 s with empty streams. All 27 inputs were unchanged during this
  run. Cross-attempt inventories differ only in `scene_test.go`; production and
  the other twelve passing cases were not rerun.

Both `run.ps1` and `go-check.ps1` are retained beside direct stdout/stderr and
stage command metadata. Formatting preceded the inventories. Commands used the
pinned `scripts/go.ps1` with explicit arrays:

```text
test ./cmd/levnote-mindmap-preview -count=1 -timeout 120s -v
test ./cmd/levnote-mindmap-preview -run ^TestPreviewWorkerFourLayoutsIndependentGeometryPaintAndHiddenChild$ -count=1 -timeout 120s -v
vet ./cmd/levnote-mindmap-preview
```

Final SHA256 fingerprints:

| Item | SHA256 |
| --- | --- |
| Preserved attempt01 result | `58CE9C3567E7EE4FE7FFD407CA5179904744B4BEB43094211E8D9BB5A3B3F010` |
| Final attempt02 result | `3404CE6FC0FA6224928413C917EBEA047BCB4DEA05BADC2C4940489ACF000FD1` |
| Final source inventory | `64093208FED871384A3FF376E1DF1F234CB55460047ACDB79C0662A7535E94C1` |
| main.go | `BD3C58E342E457B94E4A449B8FB36FF276BF115372B0976B77C53A477056AFF5` |
| scene.go | `25EF428B63015B179568083EA24DADCCEA1932BC994E59521F4BF4D3BE0C6FF4` |
| main_test.go | `F1AEC813CA727C3221721B0E984763C1FA4E57B8C1F9A22CBA5720CFFA2015DE` |
| scene_test.go | `B4B001D43AE9FAFA996EA3F5FC05F9189EE9D22AF1C4EA20EC746D48C4131533` |

The inventory includes command files and imported production codec/format/
catalog/IPC sources, module files, wrapper, toolchain executable and guard inputs;
it excludes concurrently authored transport tests. No app/runtime/UI, standalone
worker process, package, or resource verification occurred. The 32 MiB wire cap
is not a total-process-memory guarantee. Accepted `.6` artifacts stay unchanged.
