# Next editable engine boundary — corrected design

Status: DESIGN ONLY, 2026-11-09. This supersedes the previous draft. No
implementation, product/test/script edit, test execution, build, process
launch, package mutation, network research, or child delegation occurred.
Current schema3 app acceptance and .8 packaging do not depend on this file.

## Recommendation

Do not add a fourth executable as the next product step merely to prove an
editable worker. The more useful next slice is a **trusted first-party fixed
engine package boundary**: versioned package identity, installation and
rollback records, explicit user consent, and activation limited to
repository-published first-party capabilities. That advances modular formats
truthfully without pretending that arbitrary third-party executable support
already exists. It does not automatically reduce memory or CPU.

If root wants to retire editable worker semantics first, the smallest honest
pilot remains one explicit Mindmap topic **Apply** operation. It is useful for
proving snapshot identity, stale safety, byte-range publication, pending UI,
cancellation and undo, but it leaves route selection, ordinary edits, outline,
layout and post-commit refresh in the host and temporarily starts another
process. I recommend the package boundary first, then host the rename pilot as
its first capability if root still wants it.

## Explicit Mindmap Apply pilot

The worker path is opt-in and does not replace current Enter or automatic
`Flush` behavior. The user selects a Mindmap node, edits its topic, and presses
an explicit **Apply** button. Enter and the existing synchronous topic commit
remain in-process.

The host captures, on the UI thread:

- the exact buffer and immutable source snapshot;
- the buffer version and a draft generation;
- the selected node ID, at most 512 bytes;
- a SHA-256 digest of the exact old topic; and
- the exact new topic, at most 16 KiB and valid UTF-8.

While the worker is active, the topic input remains editable and shows a
visible pending state. The host does not copy it into the authoritative buffer.
The user can explicitly wait or cancel. Cancel stops the app-owned task and
keeps the draft; because the draft itself is still uncommitted, cancel cannot
promise that save or navigation will proceed afterward. No path silently drops
or discards the text. This is a semantic pilot, not a measured memory or
latency improvement.

## Bounded request envelope

The existing `engineipc.RequestMeta` has only `engine` and `format`; strict
request decoding rejects other keys. Two 16 KiB topic strings plus a 512-byte
ID cannot fit the existing 4 KiB metadata frame after JSON escaping. The old
three operations must keep their unchanged request shape.

Use an operation-specific two-part request payload for the new engine only:

```text
metadata JSON (<= 4 KiB):
  engine, format, nodeID, sourceVersion, oldTopicSHA256

request payload envelope (catalog limit 16 MiB + 16 KiB + 16):
  uint64 sourceLength
  source bytes (<= 16 MiB, exact complete Mindmap document)
  uint64 commandLength
  command bytes (<= 16 KiB, exact UTF-8 new topic)
```

No topic text enters metadata. Admission requires exact metadata keys and
types, a nonempty bounded node ID, valid UTF-8, a 64-byte lowercase hex
digest, distinct old/new values, exact declared lengths, and no trailing
bytes. The runtime checks metadata and both lengths against catalog limits
before it allocates or reads either part. `sourceVersion` is host identity,
not worker authorization.

The old Markdown HTML, Mindmap SVG, and readonly Mindmap Preview requests do
not gain optional fields or a generic command schema.

## Exact token patch

The worker parses the complete source with the existing Mindmap parser, finds
the node by exact ID, hashes the decoded topic, and refuses unless the digest
matches the captured old-topic digest. It resolves the actual `topic` JSON
string token in the source and returns a bounded patch:

```text
uint64 start
uint64 end
uint64 replacementLength
replacement topic token bytes
```

The replacement must be a valid JSON string token encoding the captured new
topic, and the patched source must remain at most 16 MiB and valid UTF-8. The
worker never rewrites or serializes the entire document.

Current `mindmap.Parse` retains each `Member.Raw`, but `mindmap.Node` exposes
neither that raw member nor a source offset for the topic token. A correct
locator is new format work, not something the worker can infer from current
APIs. Add an additive exact `formats/mindmap` topic-token locator that uses the
same strict parser path and records the token's absolute byte range. It must
refuse when it cannot prove identity; it must not guess offsets or accept the
first textually similar value.

On completion, the host does not independently rebuild and serialize the
document to validate the worker. It verifies the captured source identity and
version, old-topic digest, token range, replacement token, resulting size,
valid UTF-8, and that every byte outside `[start,end)` is unchanged. It applies
the one range replacement in one text transaction, so undo remains host-owned
and one-step. After publication, the existing graphview refresh performs its
normal host parse, outline and layout update; the pilot does not claim those
operations moved into the worker.

## Pending state and cancellation

Add a narrow optional contract beside `BeginScene`:

```go
type EditCapability struct{ ID, Format string }
type RenameRequest struct {
    Capability EditCapability
    NodeID, OldDigest, New string
    SourceVersion uint64
}
type RenameResult struct {
    Start, End uint64
    Replacement []byte
    Warnings []SceneWarning
    Engine SceneEngineInfo
}
BeginRename func(owner View, request RenameRequest,
    complete func(RenameResult, error)) (cancel func(), err error)
```

The active app-owned job may retain bounded identity references such as tab ID,
buffer pointer, node ID, draft generation and source version. Runtime
callbacks never read mutable UI state off-thread.

Completion is delivered on the UI pump, exactly once, unless canceled first.
The app first rechecks tab, buffer, adapter, view, node, draft generation and
buffer version; only then can the view inspect and publish the patch. On tab
close, view release, or explicit cancel, the app clears owner and result
references, invalidates completion, cancels the task, and lets the app-owned
lane join and drop the worker result. There is no permanent per-tab process.

Existing save, navigation, switch, export, close and modal paths keep the
synchronous `Flush` barrier. While worker application is pending, `Flush`
reports a visible pending-edit error and the action refuses; the UI offers an
explicit wait or cancel rather than waiting on the child or pretending success.

## First-party and third-party boundaries

Current schema3 routing manifests select only `builtin.markdown` or
`builtin.mindmap`; they do not install executable code. Current
`enginecatalog` and `engineruntime` admit fixed repository IDs and verify a
locally declared SHA-256 for corruption and packaging integrity. That does not
create publisher trust, containment, version migration, or arbitrary engine
installation.

Separate two boundaries:

1. **First-party separately installable packages.** These can use a trusted
   repository publisher identity, pinned operation contracts, package
   manifests, installation and rollback records, explicit user consent, and
   refusal of unknown publishers. This can be useful before any generalized
   third-party ecosystem exists.
2. **Arbitrary third-party executables.** These additionally need an explicit
   trust and containment policy. Publisher signing and OS sandboxing are
   possible policy choices, not requirements imposed by the user. The product
   must also define permissions, resource limits, upgrade, downgrade and
   incident response.

SHA256 is not publisher trust or an OS sandbox. Do not add a third-party
runtime, scripting, networking, or a generic validation framework.

## Implemented facts used

- `native/core/enginecatalog/catalog.go` fixes the three current operations.
- `native/core/engineipc/protocol.go` and `native/core/engineipc/json.go`
  keep source-only requests and reject unknown metadata.
- `native/core/engineruntime/run.go` provides one-shot, one-active process
  execution, bounded output, cancellation and job-object cleanup.
- `native/app/engine_lane.go` serializes preview/export tasks on one lane.
- `native/app/preview_worker.go` invalidates UI ownership while app-owned
  completion still joins the task.
- `native/ui/editor/contract.go` defines the synchronous, fallible `Flush`
  barrier and the existing asynchronous scene seam.
- `native/modules/graphview/draft.go` owns topic drafts, conflict, recovery
  and current synchronous commit behavior.
- `native/formats/mindmap/parse.go` preserves raw members but does not expose
  topic-token source offsets.
- `native/modules/graphview/outline.go` and `sync.go` still perform host
  parsing, outline and layout work.

## Finite implementation stages

1. **Exact topic-token locator.** Add `native/formats/mindmap/token.go` and
   focused tests in `native/formats/mindmap/token_test.go`. Oracles: unique
   node ID, duplicate IDs, escaped JSON, invalid UTF-8, absent topic, empty
   topic, exact raw-token range, and refusal when identity cannot be proved.
2. **Operation envelope and worker.** Extend
   `native/core/enginecatalog/catalog.go`,
   `native/core/engineipc/protocol.go`, `native/core/engineipc/json.go`, and
   `native/core/engineruntime/run.go` for the operation-specific envelope.
   Add `native/cmd/levnote-mindmap-rename/main.go`. Oracles: reject-before-
   allocation, length mismatch, oversized command, digest mismatch, no payload
   on error, and unchanged old request behavior.
3. **Apply UI and publication.** Add `native/app/edit_worker.go`; modify
   `native/app/engine_lane.go`, `native/modules/graphview/draft.go`, and
   `native/modules/graphview/adapter.go`. Oracles: explicit apply only, no-op,
   pending Flush refusal, stale identity, cancel keeps draft, tab close joins
   without publishing, worker failure, one-step undo, and exact unrelated
   bytes.
4. **One fresh deliverable gate.** Run one focused real-bundled-worker app
   HostPath test covering explicit apply, exact token patch, semantic refresh,
   save/navigation refusal, cancel, undo, and cleanup. Reuse existing broad
   runtime and resource evidence where inputs are unchanged; do not create
   another generic validation framework.

Future owners must be assigned by root. The likely file split is: format owner
for `native/formats/mindmap/**`; engine owner for
`native/core/engine{catalog,ipc,runtime}/**` and
`native/cmd/levnote-mindmap-rename/**`; graphview owner for
`native/modules/graphview/**`; app owner for `native/app/**`. Do not begin this
work while Astra's current schema3 delivery or .8 packaging is active.
