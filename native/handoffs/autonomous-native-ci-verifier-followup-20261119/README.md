# CI verifier follow-up — inactive static revision

Prepared 2026-11-19. **UNEXECUTED / NOT WINDOWS ACCEPTED / CI DISABLED.**
This is a new verifier source revision for F1/F2 in the read-only follow-up plan.
No PowerShell parser, fixture, Go command, listing, test, vet, build, packaging,
worker, installer or network operation was run to produce it. Static source
inspection and byte/hash/diff checks are the only performed checks. The November
14 handoff, its disabled workflow and runner, portable README, packaging scripts
and product source were not edited by this task.

The requested model identity was `magpie-responses/xingchiapi/gpt-6.1-sol max`.
This agent's supplied runtime identifies `xingchiapi/gpt-6.1-sol`; it has no tool
to switch or independently attest the provider routing and reasoning effort.
The requested full identity is therefore **not independently verified** here.
No subagent was delegated.

## Files and activation boundary

Seven engineering files, counting this README:

- `ci-lib.ps1`: strict local-drive/filesystem, JSON, pin and six-root primitives.
- `markdown-targets.ps1`: dedicated Markdown target parser/resolver and common
  authored-inline-link checker.
- `test-json-reducer.ps1`: independent expected-identity preflight and JSONL reducer.
- `package-verifier.ps1`: standalone five-PE/25-assets/self-excluded-manifest
  verifier, wired to the common Markdown checker. It does not load the old draft.
- `run-native-ci.ps1`: default-inactive **captured-verification-only** entry point.
- `tests/ci-verifier.tests.ps1`: self-contained positive/negative Windows PS7
  fixtures, including inert PE-header bytes. No Pester or Go dependency.
- `README.md`: policy, schemas, limitations and later gate prerequisites.

`-ExecuteReviewed` always throws. This revision contains no Go/process/product
execution lane, workflow YAML or CI-enabling path. The old process-launching lane
was deliberately not carried into an executable path. A later owner must review
any new product execution integration separately; these sources do not authorize
it. `-VerifyCapturedOnly` plus a separately root-reviewed, pinned schema-2
contract permits only offline checking of an already captured run on the reviewed
Windows PS7 runtime. Its status is `VERIFIED_CAPTURE_ONLY`, never a native-CI or
Windows acceptance status. Optional package verification yields `PACKAGED_STATIC`
and `NOT_WINDOWS_ACCEPTED`; it never launches a PE.

## F1: authored Markdown target policy

The parser sees the raw target before joining, normalization or removal of a
separator. It preserves the old non-filesystem classification: a leading `#`
is a fragment; the old case-insensitive `http:`/`https:` prefixes are opaque
external targets. Those categories trigger no filesystem lookup or network
request. External URL validity, anchor existence and query semantics are not
verified. No new schemes are accepted as external.

All other targets must satisfy the raw canonical relative grammar:

1. One or more nonempty components separated by **single forward slashes**.
   Absolute/drive/UNC/device paths, backslashes, repeated separators, empty
   interior components and `.`/`..` components fail before resolution.
2. At most one terminal `/`. Its presence is an **explicit directory intent in
   the target syntax** (`[licenses/](licenses/)` or `[licenses](licenses/)`). A
   label alone never coerces a file into a directory. The optional parser intent
   argument can constrain File/Directory and disagreement fails.
3. Control characters, whitespace, `: * ? " < > | # %`, Windows device basenames,
   and components ending in a dot or space fail. Percent escapes are neither
   decoded nor normalized. This deliberately narrow authored-inline-link policy
   does not add support for filenames with spaces or URL-encoded names.
4. A local `LICENSE#part`, `LICENSE?download=1`, `licenses/#part`, or query-only
   target fails. The implementation never silently drops a local suffix and
   then checks a different file. A **leading** fragment remains a fragment.

The file branch retains `RelativeFile` + `PlainFile` + `PlainChain` + `Within`.
`RelativeFile(root, 'licenses/')` still rejects, including manifest/member call
sites. The directory branch validates the complete raw spelling, removes exactly
one slash with `Substring` (never `TrimEnd`), joins below the package root and
requires `Within` + `PlainDirectory` + `PlainChain` + `Within`. A slashless
`licenses` follows the file branch and fails on a directory; `LICENSE/` follows
the directory branch and fails on a regular file. A filename prefix or existence
alone cannot satisfy a directory link. The selected root, target and every
existing ancestor must be plain; symlinks, Junctions, all reparse-point types,
wrong types and missing components fail on either branch.

`CheckMarkdownLinks` handles the same bounded inline `[label](target)` family
used by the current authored portable README and generated package notices;
image-style inline syntax is encountered too. It fails on empty/malformed inline
links and unsupported nested parentheses/titles rather than silently stripping
parts. It counts every `](` token, including those in code examples, as the old
raw scan did. It is **not** a CommonMark renderer or an auditor for reference
links, HTML, escaped/nested labels, or upstream attribution READMEs. The five-PE
checker invokes it only for the package-root `README.md` and generated
`THIRD_PARTY_NOTICES.md`, not the source notices containing repository-relative
`../LICENSE` or preserved upstream provenance files.

The portable README's `[licenses/](licenses/)` and `[attribution/](attribution/)`
are preserved. The five-PE verifier compares actual generated README bytes to
the generator's UTF-8/no-BOM source-text + `{{VERSION}}` substitution convention;
it then resolves those **same bytes** through the common checker. Fixtures read
the real portable source, check its two literal directory links, make a temporary
version-substituted README, exercise the full checker with inert PE headers and
real read-only attribution inputs, and compare the source's before/after
fingerprint. No source README or package script repair is part of this revision.

Absolute OS input paths may contain ordinary interior spaces such as `Program
Files`; that does not relax the raw relative Markdown/member grammar. Local
integrity checks are not signatures, an adversarial filesystem sandbox, or a
race-proof handle-based resolver. A trusted, quiescent disposable tree remains
a prerequisite. Hard links are not classified as reparse points by this policy.

## F2: independent expected identities

The exact arguments, including order and slash spellings, are immutable:

```text
./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/...
```

The fixed target is `GOOS=windows`, `GOARCH=amd64`, `GOAMD64=v1`,
`goVersion=go1.27.3`, module `levnote.local/native`. `app` is a single selected
package, not a recursive `app/...` root; each other selected root admits itself
and component-bound descendants. No handoff snapshot or `./...` is admitted.
Root arrays and full listing/test command argument arrays are checked in the
contract, independent listing, expected manifest and command receipt. Public
reducer entry points check the supplied roots **before input reads or any output
write**. The wrapper checks all identity inputs before making an evidence root.

`ReduceTestJson(Inputs, Roots, SummaryPath)` reads seven distinct, independently
pinned input paths. Each descriptor has exactly `path` and lowercase 64-hex
`sha256`. No expected package is inferred from the observed test stream:

| Input key | Required identity and purpose |
| --- | --- |
| `sourceInventory` | Schema 1 frozen inventory with `repositoryRoot`, ordinal-sorted unique `files`, optional `scope`. Each file has canonical `path`, integer `bytes`, lowercase `sha256`; optional original `absolutePath`/`lastWriteUtcTicks` remain allowed. `native/go.mod` and every independently listed test filename must be represented. |
| `toolchainIdentity` | Schema 1 `target`, fixed official `archiveSHA256`/`archiveBytes`, `executable: {sha256, bytes}`, `inventory: {sha256, fileCount}`. Nonempty binary/inventory identities required. |
| `rawPackageListing` | Exact independently captured pre-test listing bytes, opaque to this reducer, pinned separately. They must be preserved for independent source review. |
| `packageListing` | Root-reviewed normalized **pre-test** listing, bound to source/toolchain and the raw listing hash, with each package's `TestGoFiles` and `XTestGoFiles`. |
| `expectedManifest` | Separately sealed expected package identities and boolean `HasTests`, bound to source/toolchain and normalized-listing hash. Exact package order/set and classification are compared to the independent listing. |
| `runIdentity` | Pinned completed-command receipt binding exact arguments, roots, target, source/toolchain, expected-manifest hash and raw-JSONL hash, with exit/cleanup facts. |
| `rawJsonl` | Exact captured `go test -json` output bytes, never rewritten by the reducer. |

The official archive identity carried from the historical draft is
`f26081f438319757f8a1b46d7d0b2c5ec7583d4038e485e0aa725cfde7885fff`,
79,010,309 bytes. It is a required identity input, **not a claim that an archive or
Windows Go executable was available or executed in this task**.

Every target object has exactly these keys and string values:

```json
{"GOOS":"windows","GOARCH":"amd64","GOAMD64":"v1","goVersion":"go1.27.3"}
```

The normalized `packageListing` object has exactly:

```text
schema: 1
origin: "independent-package-list-before-tests"
roots: [the six exact strings above]
target: the fixed target object
modulePath: "levnote.local/native"
sourceInventorySHA256: pin of sourceInventory
toolchainIdentitySHA256: pin of toolchainIdentity
generation:
  arguments: ["list", "-json", the six exact strings]
  rawListingSHA256: pin of rawPackageListing
  completedBeforeTest: true
packages: ordinal-sorted array of
  {Package: full import identity,
   RelativeDirectory: "app" or selected recursive-root directory,
   TestGoFiles: [test-file basenames], XTestGoFiles: [test-file basenames]}
```

The `expectedManifest` uses the same schema/origin/roots/target/module/source/
toolchain fields, replaces `generation` with `packageListingSHA256`, and uses
package rows `{Package, RelativeDirectory, HasTests}`. Every field name is case
sensitive. `HasTests` must be a JSON boolean and equal whether the independently
listed test-file arrays are nonempty. Test files are unique basenames ending in
`_test.go`, and must appear in the source inventory. Package identities are
unique (including case-alias rejection), ordinal-sorted, and exactly module +
relative directory. Empty expected sets, unknown keys, duplicate/case-aliased
JSON properties, wrong types and identity mismatches fail closed.

`runIdentity` schema 1 requires `roots`, `target`,
`arguments=["test","-json","-count=1","-timeout=15m", six roots]`,
`sourceInventorySHA256`, `toolchainIdentitySHA256`,
`expectedManifestSHA256`, `rawJsonlSHA256`, integer `exitCode=0`,
booleans `completed=true`, `jobEmptyObserved=true`, `timedOut=false`, and
`cleanupForced=false`. Events cannot upgrade a failed/incomplete command receipt.

**No live expected manifest, Go listing, normalizer execution or Go receipt is
fabricated here.** A future separately authorized owner must independently
capture/decode the pinned Go listing (its raw format can be concatenated JSON
objects), normalize it without deriving anything from test observations, review
it against the exact frozen checkout/target/toolchain, then seal its expected
manifest before any test starts. This revision is a consumer of that independently
reviewed identity set, not a listing producer. The provenance label, boolean and
hashes alone cannot prove chronology, listing completeness, execution environment
or authenticity. The root reviewer must establish those facts independently.
The reducer proves equality of supplied pinned identities; it does not query Go
or rehash a live source/toolchain tree on behalf of the capture producer.

## Event protocol and exact summary

The implementation was informed by a **local static read**, not a Go run:

- `.localtools/go1.27.3/src/cmd/internal/test2json/test2json.go`: initial `start`,
  nested report flush, `?   \t<Package>\t[no test files]\n`, final `skip` handling,
  output chunking, and final package events.
- `.localtools/go1.27.3/src/cmd/go/internal/test/test.go`: no-test selection from
  `TestGoFiles` + `XTestGoFiles` and exact output around lines 1534–1567.
- `.localtools/go1.27.3/src/cmd/test2json/main.go`: documented test event fields.
- `.localtools/go1.27.3/src/cmd/go/internal/load/printer.go`: separate
  `build-output`/`build-fail` JSON schema.

Supported actions are exactly `start`, `run`, `pause`, `cont`, `output`, `pass`,
`skip`, `fail`. `fail` always rejects. Every expected package needs one start and
one terminal, with no events before its start or after its terminal. A
`HasTests=true` package needs package `pass`, at least one observed test run, and
all observed tests terminated. Each full **Package + Test** identity needs one
`run` and one `pass`/`skip`/`fail`; duplicates, orphans, missing terminals and
premature package completion fail. Nested runs require an observed live top-level
parent. Existing observed parent prefixes must remain live; a slash in a child
label does not fabricate an intermediate parent event. Parents finish after all
observed descendants. Pause/continue must alternate in the running/paused state;
a paused test cannot finish. Packages can interleave freely.

A no-test package may emit no Test identities and must terminate with `skip`.
Concatenating its package output chunks must yield exactly
`?   \t<that Package>\t[no test files]\n`, not a substring, wrong package,
`[no tests to run]`, or some unrelated message. Its completion is counted as
**no-test**, never as a tested package pass. A test-level skip needs a matching
run and captured non-whitespace, **non-framing** output/reason; all its output
chunks and their line positions are retained. An identity-less skip, unknown
package, duplicate skip, or framing-only/no-reason skip fails. Thus a legitimate
`SkipNow` without a message is conservatively unsupported in this draft.

Only observed Test identities can be checked. A completely omitted run **and**
terminal pair cannot be detected from a package-only expected manifest; the
reviewed command/capture and future contract controls remain essential. Also,
a package with `_test.go` helper files but no actual runnable tests is rejected
by this draft's at-least-one-run policy until separately reviewed.

Go 1.27.3 source also defines `bench`, `attr`, `artifacts`, and build event
families. They are **explicitly unsupported and rejected**, never guessed into
pass/completion. This may reject legitimate future raw output. Before production
integration, inspect a separately authorized native Windows raw capture and
review any narrowly justified schema/action extension. Do not silently ignore
such records, decode them using PowerShell coercions, substitute a guessed
package identity, or loosen this rule just to make fixtures pass.

Strict JSON parsing uses .NET `JsonDocument` (including duplicate/case-alias
property inspection), then explicitly materializes each JSON value kind into
ordinal dictionaries, arrays, strings, booleans and numbers. It never uses
`ConvertFrom-Json` date/type inference: timestamp-looking JSON strings remain
strings across PS7 versions. Blank lines,
non-objects, malformed JSON, unknown keys/actions and trailing data fail.
The explicit limits are 256 MiB raw capture and 1 MiB per JSONL line. Input hashes
and size/timestamp fingerprints are rechecked after reduction.

Only after expected/observed package equality and all unique lifecycle terminals
are established is a create-only `test-summary.json` written. It records exact
sorted package identities, per-package HasTests/start/terminal/line/output facts,
per-test full identities/run/terminal/line/pause/continue/output facts, separate
package no-test and test-skip rows, counters, input fingerprints and the raw
capture hash/path. Counter-only acceptance is not used. Direct library rejection
writes no success summary and never changes raw bytes. The wrapper first copies
raw bytes create-only to `stdout.raw.jsonl`; event rejection leaves that capture
and a `REJECTED_CAPTURE` result. Bad root/context preflight writes nothing; the
original raw input remains available. A later failed postguard cannot upgrade
an overall rejection even if a valid event-only summary already exists.

## Captured-verifier contract

The wrapper's schema-2 contract has these exact required keys:

```text
schema=2, purpose="offline-captured-verifier-only"
captureVerificationAuthorized=true, helpersReviewed=true, sourceFrozen=true
roots=[six exact roots]
helpers={"ci-lib.ps1":hash,"markdown-targets.ps1":hash,
         "test-json-reducer.ps1":hash,"package-verifier.ps1":hash}
runnerSHA256=hash of run-native-ci.ps1
rootReview={path:absolute local file,sha256:hash}
runtime={path:actual reviewed pwsh.exe,sha256:hash,version:exact version string}
inputs={the seven path/hash descriptors above}
evidenceRoot=absent absolute local directory with an existing plain parent
```

Optional `packageCapture` has exactly `nativeRoot`, `bundle`, `packageEvidence`,
`version`. It enables only offline `CheckFivePEPackage`, using already present
files. The wrapper does not create a source freeze, expected manifest, listing,
command receipt, Go extraction, package, host or worker.

Bootstrap contract/runtime/helper/root-review pins and non-reparse file chains
are checked before any adjacent helper is loaded. The actual runtime must be
PowerShell **7**, native Windows x64 OS/process; x86, ARM64, emulation assumptions
and non-Windows runs do not satisfy this gate. The contract, runner, helpers,
runtime, root-review and input files are postchecked. Contract flags/hashes are
local integrity assertions, not an approval authority or publisher signature.
No valid execution contract is shipped in this untested revision.

## Future Windows PS7 gate, prerequisites and exclusions

1. Fresh root ownership/review and a source freeze of this exact revision; verify
   the frozen file hashes and preserve the old handoff/disabled workflow.
2. Independently establish an ordinary disposable **native Windows x64** machine,
   its actual reviewed PS7 executable path/version/hash and a plain OS temp chain.
   Review .NET JSON API availability and PowerShell binding semantics by an
   explicit parse gate first; this task has not done so.
3. Ensure Junction and file/directory symlink negative controls really can be
   created and have reparse attributes. Missing privilege, unsupported filesystem
   or failed Junction creation is **FAILED setup**, never SKIP or a pass. Do not
   infer Windows reparse protection from a POSIX symlink fixture.
4. Run only `tests/ci-verifier.tests.ps1` in that reviewed PS7 host, capturing its
   actual command, stdout, stderr, exit and the returned OS-temp root. It uses only
   PS7/.NET, read-only repository inputs, and fresh GUID-named OS temp directories;
   no Pester, Go, executable launch or network. Fake PE headers are inert bytes.
   All fixtures and `fixture-result.json` are retained in the OS temp root for
   inspection; the suite performs no workspace-relative writes or cleanup.
   An unsupported host or setup fault records FAILED, not a skipped test.
5. Review every case, including full five-PE README wiring, file/directory
   mismatches, raw grammar, fragments/queries, Junction/symlink chains, nested and
   interleaved lifecycles, no-test exact output, skips, malformed/duplicate JSON,
   independent identity pins, command/target drift, and roots-before-write checks.
6. Only then consider separately authorizing an independent real Windows package
   listing/capture identity producer and a fresh Go run. Inspect actual 1.27.3 raw
   contracts and any unsupported actions before adding a product execution lane.
   Such work needs a new root-reviewed plan/contract and new evidence namespace.

There was no fixture PASS, Windows PASS, limited-CI PASS, product acceptance or
CI enablement in this task. These changes do not close GUI, IME, clipboard,
physical input, worker lifecycle, host/session normal-close/restart/recovery,
resource, response-time, packaging or release acceptance gates. No existing
runnable acceptance is replaced.
