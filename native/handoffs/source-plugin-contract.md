# Installed source-plugin outline contract — schema2 implementation

Status: IMPLEMENTED AND FROZEN. Root accepted the14:03 outline build before
activating the exact signatures below. DeepSeek transferred core/plugins and
examples to GLM. Astra's engine/provider finite checkpoint and GLM independent
review are complete; DeepSeek's14:43 full test/vet/build/smoke passed. Current
artifact evidence is in integration.md; focused engine evidence is below.

This contract implements the accepted plan while preserving its
boundaries: no plugin process, script, DLL, web runtime, network access, rich
renderer, regex language, or whole-language parser. Installed rules provide
source outlines and source reveal only. File bytes and plaintext fallback remain
unchanged by listing or navigation.

## Schema

Schema1 manifests remain valid and unchanged. A schema1 reader must reject an
`outline` field; schema2 is the only reader that accepts it. There is no silent
upgrade and no `apiVersion` reinterpretation: schema2 uses API1 and preserves
the existing omitted API/version defaults. Samples explicitly write
`apiVersion: 1`; an incompatible explicit version is rejected.

```go
type OutlineRule struct {
    Prefix string
    Suffix string
    Level  int
}
```

Manifest example:

```json
{
  "schema": 2,
  "id": "ini-outline",
  "name": "INI headings",
  "version": "1.0.0",
  "apiVersion": 1,
  "extensions": [".ini"],
  "lineComment": "#",
  "outline": [
    { "prefix": "[", "suffix": "]", "level": 1 },
    { "prefix": "# ", "suffix": "", "level": 2 }
  ]
}
```

Validation is strict:

- Top-level unknown fields and unknown fields inside every outline rule are
  rejected. `outline`, when present, is an array with at most 8 rule objects.
- `prefix` is required, non-null, valid UTF-8, 1–16 bytes, and contains no NUL,
  CR, LF, or other control byte.
- `suffix` is optional, non-null when present, valid UTF-8, 0–16 bytes, and has
  the same control-byte ban. Absence and JSON `""` both decode to the empty
  literal.
- `level` is required, non-null JSON integer in the inclusive range 1–6. No
  floating-point or string coercion is accepted.
- `outline` may be absent or an empty JSON array; both explicitly mean no
  outline capability. A non-array value is invalid.
- At most 8 rule objects are accepted. A JSON `null` value for `outline` is
  invalid because it can hide whether the author intended no capability or
  omitted the field.
- Duplicate rule marker signatures are rejected by exact `prefix` and `suffix`,
  regardless of `level`. Two rules with the same prefix/suffix cannot be
  distinguished by first-match, so assigning different levels is also invalid.
- Existing manifest limits and installation protections are unchanged:
  64 KiB manifest, 256 installed plugins, bounded directory listing, regular
  files only, no symlinks, no overwrite of an existing ID or destination,
  atomic new-file publication, and rollback when the installed file cannot load.
- Public manager results copy every nested slice, including schema2 outline
  rules and their strings. Mutation by a caller cannot alter matching, syntax,
  or outline generation.

`SyntaxManifest` adds `Outline []OutlineRule`. `decodeManifest` decodes schema1
and schema2 with a strict field list; an absent outline is a valid empty
capability, while a mistyped field is rejected. Install/load errors identify an
invalid manifest, schema, or rule.

## Literal line semantics

Rules describe literal source-line shapes; they are not a language parser.

1. A line is a physical line: its bytes begin after the preceding LF or at the
   beginning of the buffer and end before the next LF or at EOF. Exactly one terminal CR before LF or EOF is stripped for matching; internal
   CR is preserved. The identity range still includes physical terminators.
2. Leading horizontal ASCII whitespace (`0x20` space and `0x09` tab) may precede
   the prefix.
3. `prefix` must then match exact bytes. After the prefix, if a rule has a
   non-empty suffix, the line must end with that suffix after trimming trailing
   horizontal ASCII whitespace. The suffix match is exact; it is not a regex.
4. The label is the content between prefix and suffix, trimmed only of leading
   and trailing horizontal ASCII whitespace. Lines with an empty label are
   skipped.
5. Rules are tried in manifest order and the first declared matching rule wins.
   Authors put more specific prefixes before broader prefixes. This ordering is
   visible in schema validation documentation but the engine performs no
   semantic reordering.
6. `Offset` is the UTF-16 start of the matched line's syntax, including any
   leading horizontal whitespace before the prefix. It is the exact source
   editor coordinate to reveal.

For example, with rule `prefix:"[", suffix:"]", level:1`, the line
`  [Network] \r` has label `Network`, level1, and the offset of its first leading
space.

### No language-context guarantee

The lexer is intentionally line-local. These rules do not track multiline
comments, fenced blocks, triple-quoted strings, heredocs, or any other carry
over state. Consequently a rule can match text inside multiline code, strings,
or comments. The plugin UI and schema documentation must state this limitation
prominently. The existing single-line highlight notice remains truthful and
should remain visible for schema2 plugins. Do not suppress a line merely
because highlighting metadata marks a comment: that would be an unfounded
language inference and would hide valid outline entries in some formats.

## Outline identity and resolution

Use the existing `editor.OutlineSource`, `OutlineItem`, and `OutlineSnapshot`
contracts. `AnchorID` remains empty and `ResolveOutlineAnchor` returns an
explicit unsupported error. Navigation uses the existing source-mode flush and
revalidation flow.

An item ID is opaque and in-process only. It should encode all facts needed to
avoid reinterpretation:

- the `*text.Buffer` identity,
- the source version at capture,
- the matched line range in UTF-16 coordinates,
- a compact identity for the installed ruleset, and
- the plugin ID and schema version.

Resolution:

1. Parse the ID and reject unknown schema/plugin ID or a changed ruleset
   fingerprint. Disabling/re-enabling the same unchanged installed file can keep
   its fingerprint, but refresh that replaces or changes rules must invalidate
   captured IDs visibly.
2. Map the old UTF-16 range through the existing 64-edit `MapRangeSince`.
   Overlapping/target edits, `Restore`, journal expiry, or a replaced buffer
   reject with the existing stale error.
3. Recompute the current ruleset line match at the mapped range. If the range is
   no longer a current rule match, reject rather than search by label. If the
   mapped range is a valid current match, return the current item with its
   current ID and label.
4. The caller continues to use buffer pointer and version guards around
   resolution and reveal.

This mirrors the accepted Markdown outline identity policy. It avoids ordinal
IDs, which can retarget after valid prefix insertion, while the ruleset
fingerprint closes plugin-refresh reinterpretation.

## Resource contract

Bounded extraction is mandatory:

- Process at most 2 MiB of source bytes and at most 65,536 physical lines.
  Omitted content at either limit yields a partial snapshot and a concise
  user-visible source-guidance notice; actual EOF exactly at the limit is
  complete unless another bound clipped content. The document is never
  truncated or rewritten.
- Return at most 2,048 items. Label copies are at most 512 UTF-8 bytes and use
  a rune-boundary ellipsis when clipped.
- Never materialize a pathological physical line merely to reject or clip it.
  Prefer scanning the buffer in bounded windows from the line start and keep
  only the bounded label slice. The exact buffer API may be added by the engine
  owner; this contract does not require exposing an unbounded `String()`.
- Keep one incremental UTF-16 cursor while walking LF boundaries; do not rescan
  each line from byte zero or repeatedly search every prefix.
- CRLF contributes one LF-end boundary; exactly one terminal CR is stripped
  for matching, internal CR is preserved. A final line requires actual EOF.
- Cache by buffer identity + version + ruleset fingerprint. Reuse the snapshot
  on an unchanged frame. `Release` drops items, labels and temporary scan state, retaining only the
  tiny immutable Config so the provider can be reused.

The source snapshot must not call `Flush`, `Host.Changed`, selection operations,
layout, or rich view construction. Failed extraction returns an error; a valid
source with no matching lines returns an empty snapshot, and an empty snapshot
caused by limits remains explicitly partial.

## UI and adapter integration

The important integration rule is that schema2 is source-only. It must not be
represented as a rich editor adapter with a nil `Create` callback, because
current app routes assume `Create` can construct a view whenever preview/split
navigation runs.

Use two explicit paths:

1. `Adapter.Outline(editor.Host) editor.OutlineSource` remains the generic
   source-outline factory for a mounted builtin adapter.
2. `Tab.OutlineFactory` is a separate app-owned field with this type:

   ```go
   func(editor.Host) editor.OutlineSource
   ```

   The composition function returns that factory:

   ```go
   func InstalledSourceOutlineFactory(pluginID string, syntax plugins.SyntaxManifest) func(editor.Host) editor.OutlineSource
   ```

   It lives in `modules`, converts the plugin DTO to neutral sourceoutline
   Config/Rule values, and calls `sourceview.NewOutline`. App code depends on
   the generic factory and does not import renderer or scanner internals.

For a schema2 match:

- `tab.Adapter` remains nil. The source editor and plaintext fallback continue
  to own editing and saving.
- `tab.MatchedPlugin` identifies the plugin, and app wiring installs the source
  outline provider only while that plugin is enabled and active.
- Preview, split, rich export, and rich view navigation are not offered. Existing
  routes already guard source mode; preview/split must reject before dereferencing
  a nil `Create`. The capability UI should say source outline and highlight only.
- Outline activation follows the existing source flow: flush the source tab,
  revalidate tab/buffer/controller, resolve the captured ID, then `Reveal(offset,
  offset)`. A failed flush retains content, selection, and outline state.
- Disabling the plugin falls back to the existing code/source behavior and
  releases its outline source. Re-enabling recreates it only if the installed
  plugin still loads; changed rules invalidate old IDs.
- Installed rules never outrank enabled builtin adapters. Specific builtins stay
  first. A user may explicitly disable a builtin and then installed matching can
  operate on that extension.
- The plugin panel should distinguish “schema1: highlight only” from “schema2:
  source outline + line-local highlight,” and repeat the multiline limitation.
  It must not promise comment suppression or a language-aware outline.

## Sample manifests and acceptance

The two local samples are:

- `examples/plugins/ini-outline.json`: schema2 INI section/comment rules,
  including ordered rules and no language-context guarantee.
- `examples/plugins/config-headings.json`: schema2 heading-prefix rules with a
  suffix, useful for duplicate/CRLF/Unicode acceptance.

Focused acceptance should cover:

- strict schema1 rejection of `outline`, strict schema2 acceptance, unknown
  fields, malformed rule types, marker bounds/control bytes, duplicate
  signatures, and nested-copy isolation;
- install/enable/disable/fallback and source-only UI routing;
- exact source reveal for far rows, leading whitespace, duplicates, Unicode
  labels/offsets, CRLF, and a final line without LF;
- captured-ID resolution after prefix insertion, target edit, journal expiry,
  plugin disable/re-enable, and changed-rules refresh;
- failed Flush and pending Compose retention;
- partial snapshots at 2 MiB/65,536-line/2,048-item/512-byte-label limits and a
  huge-line workload that does not allocate the whole line;
- unchanged-frame cache reuse, release on tab/plugin/reload changes, visible
  list rows, narrow/dark layout, and one final focused test/vet/build pass.

## Current ownership

- GLM: `core/plugins` schema2 validation/loading, nested rule copying, and the
  two sample manifests. No app edits.
- Astra: independent source-outline engine/provider, exact buffer-scan API if
  needed, UTF-16 identity/range semantics, and focused engine tests.
- DeepSeek: app/plugin wiring, source-only capability UI, lifecycle/fallback
  integration, install flows, and final integration/build.

Root accepted the preceding outline build and the explicit plugin ownership
transfer before activating this stage. The scanner/provider and loader are now
implemented and frozen; DeepSeek's final app integration passed at14:43.

## Astra concrete engine/provider API — approved and active

This section fixes the generic signatures used by the composition seam. Root
approved these signatures and activated implementation. The14:03 outline
artifact is historical; the14:43 schema2 artifact is recorded in integration.md.

Neutral `formats/sourceoutline` package (no editor/UI/plugin imports):

```go
const (
    MaxScanBytes = 2 << 20
    MaxScanLines = 65536
    MaxItems = 2048
    MaxLabelBytes = 512
    MaxRules = 8
    MaxMarkerBytes = 16
)

type Rule struct {
    Prefix, Suffix string
    Level int
}
type Config struct {
    PluginID string
    Schema int
    Rules []Rule
}
type Entry struct {
    Start, End int // half-open UTF16 physical-line range, including terminator
    Level int
    Text string // owned, bounded display copy
}
type Result struct {
    Entries []Entry
    Truncated bool
    Bytes int // bytes actually interpreted, <=MaxScanBytes
    Lines int // complete physical lines interpreted, <=MaxScanLines
}
func Scan(reader io.Reader, rules []Rule) (Result, error)
```

`Scan` defensively validates the finite Rule values (UTF8, markers, count,
controls, duplicate prefix/suffix and levels); the plugin loader independently
owns strict JSON field/schema validation. Config is a neutral composition DTO;
Scan does not need plugin/UI identity. The provider validates Schema==2 and a
nonempty bounded PluginID, clones <=8 rules, and uses their order as supplied.
GLM's `plugins.OutlineRule` maps into `sourceoutline.Rule` only at app/modules
composition. Neither package imports the other's model.

`modules/sourceview` package:

```go
func NewOutline(host editor.Host, config sourceoutline.Config) editor.OutlineSource
```

No rich Adapter/Create/View/Export is added by this package. A malformed Config
returns a source whose `OutlineSnapshot`/Resolve report a cached configuration
error; no panic/nil ambiguity. It fits the existing no-error factory and normal
fallible snapshot UI. The composition function sets the Tab source factory to:

```go
func(host editor.Host) editor.OutlineSource {
    return sourceview.NewOutline(host, config)
}
```

The Tab source factory is app-owned and separate from rich `tab.Adapter`, which
stays nil for an installed source format. configureTab clears/rebuilds it;
outline controller generation changes on mode/tab/plugin/reload invalidation.
Builtins keep their existing Adapter.Outline. Existing source Flush/resolve/
Reveal lifecycle stays shared; no concrete-engine inspection by app.

### Exact streaming and budget semantics

- Scanner reads into a fixed <=4096-byte scratch buffer through `io.Reader`.
  UTF8 runes split across Read calls use at most3 pending bytes. Invalid UTF8 or
  reader failure returns error, visibly distinct from a valid empty outline.
- A reader may return data with io.EOF. `(0,nil)` is retried up to8 consecutive
  calls, then returns `io.ErrNoProgress`; no unbounded idle loop. Invalid reader
  counts are rejected. Data+non-EOF-error processes only valid received bytes
  then returns that error; the caller does not present it as complete.
- At a byte/line/item boundary, existing prefetched bytes prove omitted content
  without another Read. If none remain and EOF is unknown, one scanner-byte
  probe may fill the fixed4096 scratch (therefore prefetch beyond a line/item
  boundary). It is not interpreted or included in Result.Bytes. The input read
  request is always capped by remaining2MiB+1 consumption, so at the byte limit
  this probe requests at most1 byte. Interpret<=2MiB, consume<=2MiB+1 overall;
  if EOF is already known no probe is needed.
- LF completes a physical line. Strip EXACTLY one terminal CR immediately
  before LF or at EOF for matching, but preserve every internal CR. Start/End
  identity still includes CR/LF terminator units. A final line without LF is
  valid only when actual EOF is known. A line cut by any budget is never emitted.
  Empty buffer has0 lines; terminal LF does not invent an additional EOF line.
- Before prefix matching, skip only leading ASCII space/tab. Before suffix
  matching, trim trailing ASCII space/tab; remove literal suffix, then trim
  label ASCII space/tab. Labels/markers do not gain Unicode normalization.
  The first full marker match owns the line; if its trimmed label is empty the
  line is omitted, not reinterpreted by a broader later rule.
- Retain bounded prefix/label/tail state per rule (<=8 rules,16-byte markers,
 512-byte labels); never buffer an entire physical line. A huge line can still
  match if fully scanned within budget, with a clipped label and Truncated.
  The source provider supplies bounded UTF16 slices (~2048 units), adjusting a
  chunk endpoint backward when it splits a surrogate pair. It never calls
  Buffer.String, Buffer.Line, or a full Snapshot for scanning.
- Both scanner and provider advance UTF16 counts incrementally. Labels are
  cloned/capped; no view, raw source or full line is retained. Snapshot cache
  keys are Buffer pointer+version+ruleset fingerprint. Release drops descriptors
  and scan state; the small immutable Config remains so the source is reusable.

### Identity

Provider generates an opaque ID containing Buffer identity, source version,
complete line UTF16 range and SHA256 of a length-delimited encoding of plugin
ID, schema and ALL ordered rule prefix/suffix/level tuples. No formatting/JSON
serialization ambiguity, no labels or ordinal fallback. Rules or plugin identity
change => stale error even with identical buffer/ranges. Map old range through
MapRangeSince, require the same exact current line/rule-derived entry, then return
its current ID. AnchorID remains empty; ResolveOutlineAnchor is unsupported.
The host also guards current factory generation, so disable/re-enable/reconfigure
cannot rescue an in-flight navigation by coincidental matching IDs.

### Accepted boundary and validation

Root approved NewOutline/Scan and first-full-marker-match empty-label skip.
The fixed4096-byte scratch may prefetch past a line/item limit; this is bounded
read-ahead, not interpreted work. Never read again if prefetched bytes already
prove omitted content; the overall reader cap2MiB+1 remains hard.
The same2MiB/65536/2048/512 budgets have focused malformed/no-progress/huge-line/
CRLF/Unicode/history/rules-change tests, recorded below.

## Astra engine/provider finite checkpoint — implementation complete, focused green

Files: `formats/sourceoutline/{scan,line}.go`, `modules/sourceview/outline.go`.
Public API is exactly Scan(io.Reader,[]Rule)(Result,error) and
sourceview.NewOutline(editor.Host,sourceoutline.Config) editor.OutlineSource;
ValidateRules is the small defensive pure-rule entry. No core/plugins import,
no new Buffer API, no rich view. SliceUTF16 already binary-searches piece
prefixes, so bounded windows do not rescan the whole document prefix.

Contract edge behavior implemented/tested:
- strip exactly1 terminal CR before LF/EOF; preserve internal/repeated CR;
- Unicode rune fragmentation across1-byte reads and CRLF UTF16 offsets;
- first full marker match with empty label skips without later-rule fallback;
- literal16-byte suffix after thousands of horizontal whitespace bytes;
- complete2MiB final line WITHOUT LF accepted when actual EOF is proven;
- a rune/physical line cut by2MiB budget reports partial, emits no partial item;
- exact line/item boundary EOF is complete; omitted prefetched byte marks partial
  without another Read; hard reader consumption<=2MiB+1;
- bounded4096 scratch may prefetch beyond line/item limits, never interpret it;
- provider tiny probe may materialize2UTF16 units (up to6UTF8 bytes), returning
  only requested bytes. This is bounded materialization, not a full-line copy;
- data+EOF, read errors, malformed UTF8, invalid reader counts,8 consecutive
  zero-progress reads, and progress resetting that counter;
- 500 deterministic simple-reference comparisons independently exercise rules;
- SHA256 length-delimited plugin/schema/ordered-rule identity, duplicate insertion,
  changed rule/order/plugin, edited target/EOF extension/expiry/Restore/Buffer
  replacement rejection; outside undo/cancel mapping; release/recreate;
- owned bounded labels/snapshot arrays, no inspection mutation; native tester
  Compose->Flush->oldID resolve->source Reveal; cache and fragmented reader reuse.

Exact focused checks after final boundary additions:
`native/scripts/go.ps1 -GoArguments @('test','./formats/sourceoutline','./modules/sourceview','-count=1','-timeout=120s')`
PASS scanner0.610s/provider1.004s.
`native/scripts/go.ps1 -GoArguments @('vet','./formats/sourceoutline','./modules/sourceview')`
clean. Owned git diff --check clean.
Targeted verbose bounded reader test PASS1.164s:2MiB-cut Unicode huge line uses
664 bounded materialized windows, returns0 items/partial=true; repeated snapshots
reuse them and release clears state.

Headless benchmark, Ryzen7 6800H/windowsamd64,5iterations:
`test ./formats/sourceoutline ./modules/sourceview -run '^$' -bench 'Benchmark(ScanHugeLine|SourceOutlineProvider)$' -benchtime=5x -benchmem -count=1`
-2MiB unmatched line:17.628ms/op,4896B/op,2allocs/op.
-2MiB matched line:40.477ms/op,5456B/op,4allocs/op.
-2000 source headings cold:2.102ms/op,927416B/op,10021allocs/op.
-2000 source headings cached:0.01134ms/op,131072B/op,1alloc/op.
-2000 headings after200 distributed buffer edits cold:1.745ms/op,935001B/op,10023allocs/op.
-Same fragmented cached:0.01084ms/op,131072B/op,1alloc/op.
No speedup claim from the small timing difference; these are workload samples,
not process memory or end-to-end UI performance. Full integration belongs to
DeepSeek; no broad test/build repeated here. Product files now stable/frozen for
app acceptance. Astra proceeds to truthful schema2/API documentation.
