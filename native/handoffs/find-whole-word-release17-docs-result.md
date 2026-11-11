# Whole-word Find `.17` documentation result

2026-11-10. The six release documents now present
`native-worker.20261110.17` as the current runnable package, retain the
accepted `.15` and `.16` history, and describe the app-only WholeWord option
without claiming a core/search, cache-structure or session-schema change.
No source, test, package, evidence or delivery handoff was edited. This task
ran no tests, vet, build, package, worker, launch or capture command.

## Documentation changes

- `README.md` keeps the current `.17` path and `全词` usage practical, then
  delegates historical encoder, commit-point and evidence detail to the
  existing native architecture, validation and changelog links.
- `ROADMAP.md` makes `.17` the current integration/package entry, preserves
  `.16` as prior accepted history, and records the bounded WholeWord contract.
- `CHANGELOG.md` adds the `.17` user-facing entry before the retained `.16`,
  `.15` and older history.
- `native/README.md` updates the runnable path/version, usage table, capability
  boundary and source/package evidence links.
- `native/ARCHITECTURE.md` documents `App.WholeWord`, the single
  `findOptions()` mapping, shared consumers, cache-option identity, Unicode
  boundary semantics and unchanged search/cache/session contracts. Its
  delivery paragraph now records `.17` as current and labels `.16` as
  preceding history.
- `native/VALIDATION.md` records the `.17` source/package identities and
  distinguishes actual Tester checkbox checks from method-level option tests
  and startup/clean-close smoke.

The option is documented as a runtime-only control shared by document
navigation, Replace Current/All and workspace search. It uses Unicode
letter/number/mark/underscore boundaries: `cat` matches standalone `cat`, but
not `category`, `_cat` or `cat_`. It is not Chinese linguistic segmentation.
Composition/focus lifecycle, Flush-before-replace, one-undo-group Replace All
and failed-Flush preservation remain explicit.

The accepted two `760x480` PNGs are described as MyGo Tester captures. Actual
checkbox interaction covers navigation/wrap and layout/Tab/Escape;
Unicode/case/regex, workspace and replacement/lifecycle tests assign runtime
options and invoke app methods. The documents state that the initial affected
suite and vet preceded two test-only corrections and that focused reruns
exercised those corrections. The `.17` hidden host smoke is limited to startup
and clean close; it does not prove packaged Find interaction, physical
Windows IME/UI, worker execution, memory, CPU or performance.

## Read-only verification

- The final correction performed one local target scan across the affected
  `README.md`, `native/ARCHITECTURE.md` and this result; every local target
  existed.
- The five frozen source/test identities, both accepted captures, the current
  `.17` host and the release manifest match the root-accepted values recorded
  in `find-whole-word-root-acceptance.md`; the package is exactly 23 files and
  23,690,310 bytes.
- `git diff --check` reported no whitespace errors. Git noted the existing
  README line-ending normalization warning; it was not treated as an error.

## Final document hashes

| File | Bytes | SHA-256 |
|---|---:|---|
| `README.md` | 16,465 | `6861049D370F92BD16355936396B60DE35A4309C2DA97A0D5D6FB1388A0A4AAE` |
| `ROADMAP.md` | 118,467 | `0C86381C6134E7A420CFA38464253D6D7A163CFDFC9BB81C83627D9ADBAED540` |
| `CHANGELOG.md` | 140,523 | `380463DB180C40368C812CBB16D5548DD0CB1B4D3310CF54AD27A4AE5CAF451B` |
| `native/README.md` | 56,220 | `34A5C444BB080850AAC9143981AD668DD554CCAF2FC43D12DD999C3A426795B1` |
| `native/ARCHITECTURE.md` | 56,637 | `3D94616ED53C9FA6D8306184963C6BB2659775D9073577FD574C1F810506C873` |
| `native/VALIDATION.md` | 127,590 | `5C528C65B6A46FA698E21F1BC04D4DA3FB91F184292B08554E39359B810BDEC0` |

The corrected documentation and result are frozen for root review.
