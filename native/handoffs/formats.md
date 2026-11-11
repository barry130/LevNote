## GLM formats agent acknowledgement

- Finite checkpoint from `parser_core_finish_task.txt` received and in
  progress. View ownership stays with Astra; I will not edit either view
  package. Current parser/core work is limited to that checkpoint:
  fence/list/source-range correctness, cumulative inline scan budgets and
  warning-batch capping, exact-range export warnings, absolute UTF-16 search
  previews with both ellipses, and aggregate session preflight. Documentation
  follows only after the focused parser/core tests/vet pass.

- Ownership reassignment acknowledged 2026-11-08: Astra owns both
  `modules/markdownview/**` and `modules/graphview/**`. I will not edit either
  view package. My current scope is `formats/**` except `formats/code/**`,
  `core/document/**`, `core/session/**`, `core/search/**`, native
  user/developer documentation and migration ADR.

- Task received 2026-11-08. Ownership and native constraints loaded from
  `native/AGENTS.md`, `native/formats_finish_task.txt`, and
  `native/handoffs/plan.md`.
- Highest priority is data safety/lifecycle: fix UTF-16/UTF-8 draft
  synchronization and fallible flush/recovery, then graph release/cache
  rebuilding, then bounded document/session/search behavior.
- I will implement and test only owned files, coordinate cross-owner API
  changes through this handoff, avoid child agents and commits/pushes.

## API-ready integration checkpoint

## Aggregate session API for DeepSeek

- `session.NewSnapshotBuilder(root string) *SnapshotBuilder` creates an
  in-progress recovery snapshot.
- `(b *SnapshotBuilder).AddDocument(doc *document.Document, title string,
  scroll float32) error` captures one tab. For a dirty document it checks the
  4 MiB per-draft limit and streams the buffer through a JSON-length counter
  before `CaptureBounded` materializes the draft, then updates a running
  aggregate budget. It returns on the first tab that would exceed the aggregate
  budget; the app must surface this error and stop the save so the old
  recovery file is preserved.
- `(b *SnapshotBuilder).SetActive(index int) error`,
  `SetPreferences(preferences Preferences)`, `Snapshot() (Snapshot, error)`,
  and `Save(path string) error` complete the snapshot. Save still performs
  final validation/marshalling and keeps the existing file if final 16 MiB
  persistence fails.
- `Save` now also preflights a full-snapshot JSON-size estimate before
  marshalling. Drafts are never marshalled during preflight: a rune-level
  counter accounts JSON quoting/control-character expansion, tab metadata is
  counted from a `Draft=nil` shape, and no giant JSON buffer is built. A single
  3 MiB NUL draft has raw bytes below the raw limit but an encoded size above
  the 16 MiB recovery limit, proving escape preflight while preserving the old
  file.
- Inline parser scans have a cumulative 1 MiB work budget, a 64 KiB per-suffix
  cap, and remaining-budget suffix truncation. Work is charged by actual
  examined distance; after exhaustion all lookaheads stop and the one
  `scan-budget` warning is retained. A 500,000-`[` regression directly asserts
  `scanSpent <= maxInlineScanBytes`, exhaustion, and exactly one budget warning.
- Focused tests/vet pass:
  `native\scripts\go.ps1 test ./formats/markdown ./core/document ./core/session ./core/search -count=1`
  and `native\scripts\go.ps1 vet ./formats/markdown ./core/document ./core/session ./core/search`.

- Fixed for DeepSeek: `graphview/edit.go` now imports `strings`; the app
  build blocker is gone.
- New document API: `document.WriteNewFile(path string, data []byte) error`.
  It resolves/validates the path, rejects an existing destination, and uses
  the existing atomic create-only publish path. A target created after the
  caller's preflight still wins.
- New session API: `session.CaptureBounded(doc *document.Document, title
  string, scroll float32) (Tab, error)`. It rejects/invalidates dirty drafts
  over 4 MiB or otherwise invalid *before* materializing draft data, and
  returns a validated tab otherwise. Existing `Capture` remains compatible
  but cannot report an error; recovery callers should migrate to the bounded
  function and surface failures.
- Session preference: `Preferences.LineNumbers *bool`. `nil` means the
  existing default (true); explicit false is preserved. App may replace a
  separate UI-state preference with this field.
- These APIs are staged but tests for new-file publish, bounded capture, and
  preference compatibility are still being added; treat this as integration
  ready after the focused checkpoint passes.

## Current handoff and checkpoint

- Graph ownership transfer acknowledged: stop work in `modules/graphview/**`
  after the current atomic edit. Current graph state in my edits:
  `edit.go` has the missing `strings` import; `adapter.go` invalidates
  `lastSynced` to `^uint64(0)` in `Release`, so Draw/Warnings/Export rebuild
  from the same Buffer version. `./modules/graphview` tests pass. Label/metadata
  draft lifecycle work remains for Astra to review.
- Tested checkpoint (owned core/format packages):
  `native\scripts\go.ps1 test ./formats/markdown ./core/document ./core/session ./core/search`
  passes. `CaptureBounded` now validates dirty new documents correctly, and
  `document.WriteNewFile` plus `Preferences.LineNumbers *bool` have focused
  tests.
- Markdown/view current blockers: existing `modules/markdownview` tests fail
  at their UI double-click helper before reaching lifecycle assertions; parser
  coordinate/CRLF/export fixes and their explicit tests are the remaining
  Stage A work. No graph or documentation work will be expanded before that
  safety baseline.

Additional graph handoff ACK: Astra may take `modules/graphview/**` now; I
will not edit it again. Last passing graph test command was
`native\scripts\go.ps1 test ./modules/graphview`.

## Parser resource correction checkpoint — 2026-11-08

- ACK `native/parser_resource_fix_task.txt`. `core/session/**` is now Astra's
  exclusive ownership; the stale ownership wording above is historical and I
  will not edit session again.
- Markdown contract now exposes `Document.Partial bool`. `Document.Raw()`
  always retains the complete source. A partial result covers only its bounded
  prefix and is disclosed by `parse-limit`, `inline-limit`, or `scan-budget`;
  the over-8-MiB path is also `Partial=true` with `too-large`.
- Checked export API: `ExportHTMLChecked(doc Document) (string, []Warning,
  error)`. It returns `ErrPartialDocument` for any `Partial` document, allowing
  Astra's view/export path to reject incomplete HTML explicitly. Legacy
  `ExportHTML(doc) (string, []Warning)` remains source-compatible.
- Bounded limits: 65,536 lines (exactly-at-limit stays complete), 8,192 blocks,
  4,096 list items per list, 1 MiB document-wide inline scan budget, 64 KiB
  per-suffix cap, 64-run inline delimiter cap, and 16,384 inline spans across
  the document. Appending stops before further allocation after a limit is hit.
- Inline scan exhaustion now exits the interpretation loop before backtick-run
  scanning or `strings.Repeat`; ordinary short plain paragraphs retain their
  formatting, and a document-wide budget no longer resets per tiny block.
- Ordered list export preserves `Start != 1` (including start 0) in
  `<ol start="...">`.
- Passing commands:
  `native\scripts\go.ps1 test ./formats/markdown -count=1`;
  `native\scripts\go.ps1 vet ./formats/markdown`.
- Honest allocation note: Astra's independent QA measured 256 KiB `# a\n` at
  about 24.5 MB after the fix versus 126.4 MB before, and 256 KiB pure LF at
  about 11.1 MB versus 8.4 MB before because line descriptors remain bounded by
  the 65,536-line limit but still hold that full limit. I reduced append growth
  to 8,192-line chunks after that measurement; no new before/after number is
  claimed until rerun. The fix is a hard upper bound and partial disclosure,
  not a reduction in the representation cost of a full 65,536-line prefix.

Disk-check API implementation is next, and will be appended here immediately
when landed. No session or view files are being edited.

### Focused parser result — final for this checkpoint

- The table-row detector no longer uses `strings.Split`, allocates cell slices,
  or trims/copies cell text. `Parse` only needs the row-shape predicate, so a
  256 KiB pipe-dense line now exercises the non-allocating predicate and stays
  complete with one visible `unsupported-table` block.
- `isThematic` no longer copies the line to `[]rune`; it scans ASCII bytes in
  place.
- The regression covers 256 KiB dense pipes, 65,536-line exact-at-limit and
  over-limit inputs, and 8,193 tiny headings. Over-limit results preserve raw
  source, expose `Document.Partial`, emit `parse-limit`, and
  `ExportHTMLChecked` returns `ErrPartialDocument`. Exactly-at-limit input is
  complete with no warnings.
- Passing commands:
  `native\scripts\go.ps1 test ./formats/markdown -count=1`;
  `native\scripts\go.ps1 vet ./formats/markdown`.

### Parser allocation follow-up

- Pipe-dense rows and thematic-line detection are now non-allocating: the old
  `strings.Split`/cell-slice path was removed and `isThematic` scans bytes in
  place. The 256 KiB dense-pipe workload is part of the focused parser tests.
- Small line slices now grow by 128 descriptors instead of reserving 8,192, so
  ordinary small documents do not pay the old dense-LF minimum.
- Current `native/scripts/qa_probe.go` results (one Windows run):
  64 KiB LF 11,136,944 B / 52 allocs; 256 KiB LF 11,121,640 B / 32 allocs;
  64 KiB headings 15,914,776 B / 16,453 allocs; 256 KiB headings
  24,570,488 B / 16,457 allocs; exhausted inline + 1 MiB backtick tail
  21,136 B / 28 allocs with no continued code scan. Dense LF remains about
  11 MiB because a complete 65,536-line prefix is retained; this is a bounded
  representation cost, not unbounded growth.

## Immutable disk-check contract — published for DeepSeek

- `token := doc.DiskToken()` captures an immutable path/content/file-identity
  baseline plus a monotonic document baseline generation on the UI thread. It
  contains no `*Document` or `*Buffer` reference. New documents return an
  invalid token.
- `token.Valid() bool` and `token.Path() string` expose only the frozen path
  and baseline validity.
- `result, err := token.CheckDisk(ctx context.Context)` is worker-safe and
  cancellable. It performs bounded 64 KiB reads, checks context before the first
  stat and during the read loop, hashes the regular file, and verifies the same
  file object before hashing, after hashing, and against the current pathname.
  An unstable read returns `document.ErrConflict` with
  `DiskCheckResult{Invalid:true}`; cancellation returns `context.Canceled`.
- `DiskCheckResult{Changed, Missing, Invalid}` distinguishes external change,
  deletion, and invalid/unstable checks. Same-byte and same-mtime replacements
  are detected by frozen `os.FileInfo` identity (`os.SameFile`) and content
  hash. `Missing` also sets `Changed`.
- `doc.SameBaseline(token) bool` must be called on the UI thread before
  applying any worker result. Successful `Save`, `SaveAs`, and `Revert` advance
  the baseline generation, so a pre-save token is stale even if the bytes are
  unchanged.
- `Document.CheckExternalChange()` remains the authoritative full conflict
  check immediately before save; the token API is only for coalesced background
  observations and staleness gating.
- Focused tests pass and cover unchanged baseline, external change, deletion,
  same-byte replacement, stale token after unchanged Save and Revert,
  invalid new-document token, read-only baseline, and context cancellation.
- Passing commands:
  `native\scripts\go.ps1 test ./core/document -count=1`;
  `native\scripts\go.ps1 vet ./core/document`.

## Search and final resource checkpoint — 2026-11-08

- Workspace search now uses `document.OpenLimitedRead(path, "", limit)`. The
  actual stable read enforces `WorkspaceOptions.MaxFileBytes`; a file that is
  small at `DirEntry.Info()` but grows before reading is reported as
  `too-large`, rather than being read up to the 128 MiB editable-document
  limit and decoded first.
- Workspace traversal has a deterministic entry budget:
  `WorkspaceOptions.MaxEntries` defaults to 10,000. `Summary.Entries` and
  `Summary.Dirs` report observed work; reaching the budget sets
  `Summary.Truncated` without allocating an unbounded directory list. Existing
  symlink, directory exclusion, cancellation, 64-item batch, and 2,000-match
  caps are unchanged.
- `OpenLimitedRead` preserves the document package's normal stable-read,
  decode, BOM, read-only, and UTF-16 range semantics; search does not duplicate
  encoding logic.
- Disk-token stability now compares after-hash file size/mtime and same-file
  identity against both the opened descriptor and current pathname. Instability
  returns `document.ErrConflict`. The final changed decision also compares the
  token baseline identity against the hashed result, not the hash alone.
- Unchanged `Save` no longer re-fingerprints or adopts an externally changed
  baseline. It increments the baseline generation only after the existing
  authoritative pre-save check succeeds; changed external content still causes
  `ErrConflict`. `didSave` increments exactly once.
- Current bounded probe measurements (one Windows run):
  64 KiB LF 11,136,944 B / 52 allocs; 256 KiB LF 11,121,640 B / 32 allocs;
  64 KiB headings 15,914,776 B / 16,453 allocs; 256 KiB headings
  24,570,488 B / 16,457 allocs; exhausted inline + 1 MiB backtick tail only
  21,136 B / 28 allocs and no continued code scan. The 65,536-line prefix still
  costs about 11 MB even for dense LF; that is the bounded representation cost,
  not unbounded growth. Small sources now grow line slices 128 descriptors at a
  time instead of preallocating 8,192.
- Passing commands:
  `F:\AIproject\LevNote\native\scripts\go.ps1 test ./formats/markdown ./core/document ./core/search -count=1`;
  `F:\AIproject\LevNote\native\scripts\go.ps1 vet ./formats/markdown ./core/document ./core/search`.
