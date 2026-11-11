# .18 bounded SVG release docs result

Status: **COMPLETE, FROZEN FOR ROOT REVIEW** 2026-11-10. The six product
documents now describe the accepted `native-worker.20261110.18` package and
the bounded draw.io SVG contract. `.17` and earlier release records remain
historical and are not rewritten as current.

Only the documents listed below and this result were edited. No product source,
test, evidence script, package, or evidence file changed. No test, vet, build,
launch, benchmark, managed tester, or capture was run or repeated.

## Document changes

- Root `README.md` names `.18` as current and links its acceptance, delivery,
  and 12/12 validator. Its feature summary is user guidance: the 64 MiB output
  cap, no-partial-error behavior, exact-cap success semantics, and the
  distinction between serialized output bytes and process memory. The
  low-level API values and count/fill implementation remain in architecture.
- `ROADMAP.md` updates the current integration and package rows to `.18` and
  adds a bounded draw.io SVG capability row. It preserves the `.17` package and
  direct-evidence identities separately.
- `CHANGELOG.md` adds the `.18` user-facing behavior and package entry ahead of
  the `.17` history.
- `native/README.md` updates the current executable, build example, package
  verification, run commands, draw.io capability boundary, and detailed
  accepted-source section. It records that draw.io export remains host-side
  and the package still has exactly three fixed workers.
- `native/ARCHITECTURE.md` documents the `SVGOptions.MaxBytes` contract, page
  error precedence, count/fill separation, nil-output failures, preserved
  exact-fill behavior, and the memory/performance scope limit. It replaces the
  current package block with `.18` identities and moves `.17` to explicit
  history with its 117-file / 47,784,783-byte direct evidence; `.16` keeps its
  separate 104-file / 47,774,882-byte evidence.
- `native/VALIDATION.md` starts with `.18` package, source, frozen-input,
  hidden-smoke, and scope evidence. It separates the original full
  drawio+graphview green, the later correction of `svg.go` and its test, and
  the final drawio-only 0.641s test plus vet gate over unchanged graphview. It
  also records the final six accepted source/test/oracle hashes, 12/12
  validation, `302` scoped inputs / `3,027,991` bytes, the clean hidden host
  close, and the limits of that smoke.

## Narrow correction pass

Root identified three documentation issues before acceptance:

1. The `.17` historical paragraph had inherited `.16`'s `104` files /
   `47,774,882` bytes. It now attributes `.17`'s actual `117` files /
   `47,784,783` bytes and leaves `.16` with the former values.
2. Validation chronology no longer presents the pre-correction
   drawio+graphview suite as validation of corrected files. It records that
   full-suite green as history, then the later drawio-only test at 0.641s and
   vet at exit 0 after review changes; graphview was unchanged and not rerun.
3. The root README feature paragraph is now user guidance. It keeps cap/no
   partial file/error visibility/success and memory limits, while `MaxBytes`
   normalization and count/fill mechanics live only in architecture.

These were prose corrections only. They did not trigger test, vet, build,
launch, benchmark, capture, or product-identity reruns.

## Accepted source behavior

- Serialized draw.io current-page SVG output has a hard limit of 64 MiB.
- `MaxBytes == 0` selects the 64 MiB default. A lower positive direct-API
  value is honored; a larger positive value clamps to the hard cap; a negative
  value is rejected. Invalid page lookup retains its earlier error precedence.
- The emitter performs one bounded count pass before allocating the
  exact-size output buffer. Count and fill reuse the same formatting/escaping
  rules.
- Exact-fill success preserves the prior SVG bytes, warning order, non-nil
  empty warning slice behavior, and selected-page export. Errors return nil SVG
  and nil issues with no partial output; the existing host export error remains
  the visible UI route.
- The cap is not a process-memory limit. Source/parser structures, page
  indexes, issues, point slices, bounded escape fragments, and the successful
  output buffer still consume memory. The two passes repeat some deterministic
  geometry work.

## Immutable package evidence

- Version: `native-worker.20261110.18`
- Package: 23 files / 23,699,526 bytes
- Inventory identity:
  `4A4F9BD0B6DF2DDCD1E597F026AAC985A9D50711CDD85F3E9310F7DC5B0B2991`
- Host: 14,075,904 bytes, SHA-256
  `BFA6F460F49087390F114CEC2CF2ADC034307B07769D8B64E3E70C78BBDCB44A`
- Manifest: 4,137 bytes / 22 payload entries, SHA-256
  `8CA2016C0064638BE965FDFE74468F8004A37143BE568A189FD7FEDAB2D12FDD`
- Scoped inputs: 302 files / 3,027,991 bytes, including 62
  `native/formats` files
- Inputs before/after identity:
  `B4681E2EB1E761D2D0D8F811E75D24663727C1050B6C4921B5EE6411976B02B1`
- Validation: one package invocation, four Windows amd64 GUI builds, all exit
  0 with empty streams; three worker manifests, four GUI PE images, thirteen
  portable assets, validator 12/12, and aggregate 12/12 all passed
- One exact copied hidden host start and owned `WM_CLOSE`: exit 0, no
  timeout/force/remaining process, empty app streams, zero worker operations

Evidence links:

- [Root package acceptance](drawio-svg-budget-release18-root-acceptance.md)
- [Delivery](drawio-svg-budget-release18-delivery.md)
- [Source acceptance](drawio-svg-budget-root-acceptance.md)
- [Implementation result](drawio-svg-budget-result.md)
- [Independent reconciliation](drawio-svg-budget-independent-reconciliation.md)
- [Validator](../artifacts/acceptance/drawio-svg-budget-release18-20261110-011852/validator.json)
- [Aggregate](../artifacts/acceptance/drawio-svg-budget-release18-20261110-011852/aggregate.json)
- [Hidden host result](../artifacts/acceptance/drawio-svg-budget-release18-20261110-011852/host-smoke/result.json)

## Scope

The docs do not claim measured performance, whole-app memory, physical
UI/IME behavior, packaged SVG interaction, or a dirty recovery. The hidden
smoke establishes startup and clean close only. `draw.io` remains host-side in
`.18`, the package still contains three fixed workers, and concurrent
fourth-worker source development is excluded as future, unpackaged work.

## Final document identities

| File | Bytes | SHA-256 |
|---|---:|---|
| `README.md` | 16,826 | `C8939118DAC3268A215B2A040368CE1C1CA771F0703847F1759A6417BD481433` |
| `ROADMAP.md` | 119,296 | `D67B242BFBD2F76F2F5E71078DF0B9225AE1ED119ED382D1C5E4A569B3FB06A4` |
| `CHANGELOG.md` | 142,070 | `5D20D9938B5092DD9BB85138C157683B33A5845FCD09D1A5932257AA42C833C5` |
| `native/README.md` | 58,346 | `B709F139A9DC0428532D989DC3E23C963A5254D3B34583832F4DC923BC520F87` |
| `native/ARCHITECTURE.md` | 59,129 | `9BBF1A4C414E8A7CB300C1DC6ED17693EF536A61533A705EA2C93368753DBFC4` |
| `native/VALIDATION.md` | 133,192 | `35365FA0010F3B3BC471C141207AAFCC48BBCB8FF70BE993E2C38E627E06BE49` |

## Validation performed

After the correction pass, a scoped Markdown-target scan over the three changed
product documents and this updated result checked 343 local links and found 0
missing paths. The scan ignores external URI schemes and validates filesystem
targets only; it does not validate remote availability or Markdown heading
anchors. A current-label reconciliation over the six product documents finds
`.18` as current and `.17`/earlier entries as historical. No executable or
product validation was rerun for a documentation change.

The six product documents are frozen for root review. This result and those
six documents are the complete documentation-only delta for the task.
