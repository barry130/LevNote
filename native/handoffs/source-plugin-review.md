# Source-plugin integration review

Reviewer: GLM. Mode: read-only review of Astra's scanner/provider and DeepSeek's
app integration. No scanner, provider, app, engine, contract, or test files were
edited during the review.

Status: final and frozen. The baseline-highlight concern was rechecked against
the current app and lexer code after Root's review message: schema1 and schema2
manifests without configured comment/keyword metadata still receive baseline
string/number highlighting, and the capability text now distinguishes baseline
highlighting from configured inline rules. This correction was already present
in the reviewed product files and covered by their current tests; the review
records it, not a GLM product edit. No additional tests were run after the
README-only sample-doc correction. Future Go checks should use
`native/scripts/go.ps1` as the standard workspace command.

## Verdict

No material behavioral defect was found. The implementation follows the approved
contract and the already-observed lifecycle invariants:

- `formats/sourceoutline.Scan` is finite and bounded: it interprets at most
  2 MiB, consumes at most 2 MiB+1, stops at 65,536 physical lines or 2,048
  entries, keeps labels to 512 bytes, accepts at most8 rules and16-byte
  markers, and never interprets prefetched bytes after a budget is reached.
- Marker handling is line-local and literal: exactly one terminal CR is removed
  before LF/EOF, internal CRs remain, leading ASCII space/tab is skipped, and
  the first full marker match owns a line. A budget-cut line is never emitted.
- Provider IDs include buffer identity, source version, ruleset fingerprint,
  and the full UTF-16 line range. Changes to rules, plugin identity, target
  line, buffer, or history invalidate an old ID rather than reinterpreting it.
- App navigation captures and rechecks tab, adapter/factory generation, path,
  mode, buffer, outline-controller generation, and buffer version around the
  fallible Flush/snapshot/resolve/navigation sequence.
- Failed Flush preserves the source draft and controller state. Factory changes
  during Flush reject the captured controller.
- Installed schema2 formats stay source-only: `Adapter` remains nil, preview
  and split are rejected, export explains that only source editing exists, and
  navigation reveals source coordinates without mounting a rich view.
- The declarative app setup keeps baseline string/number highlighting even when
  schema1 or schema2 has no configured comment/keyword metadata. This matches
  the actual lexer behavior at `native/formats/code/lexer.go:71`: quotes and
  numbers are recognized independently of manifest metadata. Current app tests
  assert both the actual spans and truthful capability text in
  `native/app/source_plugin_outline_test.go:248` and
  `native/app/declarative_outline_test.go:10`.

## Reviewed seams

- `native/formats/sourceoutline/scan.go:47` defensive rule validation and
  `:76` bounded `Scan`.
- `native/formats/sourceoutline/line.go:31` exact CR/line and marker matching;
  `:88` bounded label/tail state.
- `native/modules/sourceview/outline.go:36` immutable config copying;
  `:79` rule-sensitive identity; `:83` version-checked refresh;
  `:123` stale-ID mapping; `:159` bounded UTF-16 reader windows.
- `native/modules/modules.go:22` the sole plugin-DTO-to-neutral-rule composition
  seam.
- `native/app/plugins.go:42` factory/generation reset and source-only routing.
- `native/app/outline.go:291` navigation captures/rechecks context around all
  fallible operations.
- `native/app/view.go:470` source-only rendering and `:534` rich-mode rejection.
- `native/app/export.go:11` source-only export rejection.

Relevant focused tests were also read in `formats/sourceoutline/scan_test.go`,
`modules/sourceview/outline_test.go`, and `app/source_plugin_outline_test.go`.
They cover Unicode/CRLF/fragmentation, exact budget boundaries, reader errors,
empty first-marker ownership, stale identity cases, real plugin installation,
pending Compose/Flush, factory invalidation, readonly/far-row navigation,
source-only mode/export rejection, and bounded empty-partial UI guidance.

One accepted degenerate case was checked and is not reported as a defect: a
non-empty suffix consisting entirely of ASCII space/tab cannot occur at the end
of the body after trailing-horizontal trimming. This follows directly from the
approved rule (`native/handoffs/source-plugin-contract.md:91`), is not a
reachable positive marker shape, and does not alter valid rule behavior.

## GLM owned-loader focused checkpoint

The schema2 loader and samples remain frozen. The requested sample-doc wording
correction was made only in `native/examples/plugins/README.md`: the
schema/apiVersion sentence is now scoped to the schema1 `toml.json` sample;
the schema2 section continues to state `apiVersion: 1` and that `suffix` is
optional.

Validation from `native`:

```powershell
.\tools\go\bin\go.exe test ./core/plugins -count=1 -timeout=180s
.\tools\go\bin\go.exe vet ./core/plugins
git diff --check -- native/core/plugins native/examples/plugins
```

Results:

```text
ok  	levnote.local/native/core/plugins	3.998s
```

`go vet ./core/plugins` and `git diff --check` produced no diagnostics. No
broader app, scanner, or build test was run or claimed during this review.
