# Session resource correction — acknowledgement

2026-11-08: Astra has read `session_resource_fix_task.txt` and owns
`core/session/**` plus `scripts/go.ps1`. GLM must not edit session. No full
integration suite will be run concurrently with DeepSeek.

Concrete integration API being implemented:

- Existing `NewSnapshotBuilder(root)` and `AddDocument(doc, title, scroll)`
  remain available.
- `CaptureOptions{Title, Scroll, OriginalPath, PluginID, View}` and
  `AddDocumentWithOptions(doc, options) error` include all view metadata in
  preflight before any dirty draft is copied.
- `AddTab(tab Tab) error` accepts an already prepared clean/discarded record
  (or an existing draft string), using exactly the same budgets/validation.
- `Len() int` supports active-index mapping while skipping byte inspectors
  and discarded unsaved documents.
- `SetPreferences(Preferences) error` and `SetActive(index) error` are
  transactional; active 0 is valid for an empty snapshot.
- `Snapshot() (Snapshot, error)` returns independent slice/pointer fields;
  `Save(path) error` retains the previous recovery file on failure.

The builder owns raw and encoded aggregate counting. App should remove its
duplicate raw aggregate counting, set preferences before adding tabs, use
`AddTab` for discarded saved documents and `AddDocumentWithOptions` otherwise,
then set mapped active index and call `Save`.

## Finite checkpoint complete

The API above is implemented in `core/session/builder.go`. Validation and exact
budgeting are in `budget.go`. Added fields are counted before copying draft
content; no arbitrary metadata reserve remains. All operations which can fail
return errors and commit bookkeeping only on success. Nil document/buffer is
an error. `CaptureBounded` uses the same path. The caller owns `Document`/builder
on one goroutine; this API is not concurrent document access.

The counter supports `io.StringWriter`, bounded incomplete UTF-8 state, and
JSON's quotes/backslash/controls/HTML/U+2028/U+2029 escaping. String input never
converts to a draft-length byte slice. Buffer input snapshots only immutable
piece descriptors. Counts match `json.Marshal` for complete snapshots including
float32 exponent cutoffs, omitted/nil fields and all preferences. Invalid UTF-8
metadata/drafts and non-finite or invalid preference values reject before
marshalling. A near-16MiB ordinary-text snapshot passes exact preflight, proving
the former arbitrary 1MiB reserve no longer overcharges it.

Failed add, preferences, active index and Save preserve the prior state/file.
Input/output Draft/LineNumbers pointer mutation and returned Tabs slice changes
cannot alter builder bookkeeping. Saved/discarded records use `AddTab`; view
metadata enters `AddDocumentWithOptions`. DeepSeek owns app integration.

## Allocation evidence (same bounded qa_probe workload)

One Windows/amd64 run; cumulative allocated bytes, not process peak RSS. Probe
includes diagnostic formatting/file-check overhead; allocations are not solely
the counter. Setup content/document construction occurs outside measurement.

| Rejected case | Before allocated | After allocated | After allocations | Elapsed |
| --- | ---: | ---: | ---: | ---: |
| Save 3MiB NUL draft | 3,155,528 B | 6,176 B | 14 | 6.625 ms |
| Builder same buffer before capture | 6,341,320 B | 45,968 B | 10 | 6.425 ms |
| Preferences.Theme MaxBytes+1 | 33,583,696 B | 5,208 B | 9 | below timer resolution |

All reject before JSON marshalling; prior file is byte-for-byte unchanged.
Fourteen invalid-encoding captures no longer poison the budget: correcting
encoding succeeds in both reused and fresh builders. Nil doc reports an error
instead of panicking. Focused allocation regressions allow 256KiB of overhead,
enough for descriptor snapshots but far below any 3MiB input copy. Oversized
title/path and preferences rejection are included.

## Bounded recovery Load

Root's added dense-array finding is fixed by `load.go`'s `boundedTabs` decoder.
It validates/walks JSON without a RawMessage allocation per entry, decodes only
the first 100 entries, and counts omitted entries. Nested unknown properties,
escaped quotes/brackets, invalid individual entries and trailing syntax are
tested. Valid initial drafts survive; bad entries are disclosed independently;
the exact omitted count is returned. Root with NUL is rejected consistently
with Save while other drafts still recover.

Fixture: 100,000 tabs, 1,300,042-byte JSON. Load took 15.548ms, allocated
3,175,544 B in 323 allocations, retained the first 100 and reported 99,900
omitted. No OOM-scale fixture used. The bounded file read and ordinary JSON
validation still cost O(file bytes); skipped tabs retain no objects.

## Build wrapper guard

`scripts/go.ps1` checks build/run/install/test for patched vendored MyGo app.go,
exact pinned version and marker; any remaining startup cleanup call rejects.
Explicit/module-environment bypass and alternate working directory/modfile
arguments also reject. Error names `dependency-patches/refresh.ps1` as repair.
No global/toolchain source setting is changed. Module commands remain available
so refresh can vendor then apply its patch before the final guarded test.

`scripts/check-build-guard.ps1` exercises the exact wrapper in a new isolated
fixture, mutating no production vendor files. It passed missing source for all
four guarded commands, unsafe live startup call, missing marker, wrong version,
explicit and GOFLAGS module bypass, alternate directory, and successful
unpatched `mod edit`. Fixture remains under build-cache/qa-build-guard-*.
Earlier independent real refresh proof is in `qa.md`; no redundant refresh run.

## Markdown Partial contract integration

View now calls `markdown.ExportHTMLChecked`, returns nil output with the
underlying `ErrPartialDocument`, and displays Chinese source guidance for every
`Document.Partial` route. Byte/line/block/list/inline scan budgets have view-level
regressions. Partial preview does not offer editing an incomplete block tree.
An already active native IME field can flush through a safely mapped outside
source edit before export refuses the now-partial document; Release/rebuild
preserves the committed text. No parser owner files were edited.

## Final focused validation

```
native/scripts/go.ps1 -GoArguments @('test','./core/session','./modules/markdownview','./dependency-patches','-count=1')
native/scripts/go.ps1 -GoArguments @('vet','./core/session','./modules/markdownview')
native/scripts/go.ps1 -GoArguments @('run','./scripts/qa_probe.go')
native/scripts/check-build-guard.ps1
```

Final test run: session 1.158s, Markdown view 1.391s, dependency patch 0.954s;
vet clean. No full suite run concurrently with DeepSeek. Existing recovery,
draft conflict/IME and source-range regressions remain green.

Outstanding integration: DeepSeek validates close/quit/discard behavior against
this builder and full app after parallel work stabilizes. Root has queued
`view_resource_task.txt`; Astra proceeds to measured repeated-frame rendering
resource work next. No new outline/table/graph metadata features added.
