# Second fixed export engine: integration plan

Status: IMPLEMENTATION ACTIVE 2026-11-09. Root accepted pure extraction and its
new boundary regression (review P1 withdrawn with pinned stdlib evidence).
Catalog/IPC product/API source reviewed and frozen; GLM finishes finite missing
identity-before-payload oracles and retained logs in its own tests. Astra owns
runtime/app implementation below, a new DeepSeek owns only the Mindmap command.
Dependent final gates wait for the IPC evidence; source work may proceed now.
Current .5 package and accepted evidence are immutable. Root plans/reviews only.

PREPARATION ONLY 2026-11-09: Astra may now inspect the runtime/app source and
write `mindmap-engine-runtime-checkpoint.md` with precise owned-file/API choices,
test reuse and proposed edit order. No runtime/app/product edits or execution
until activation. Pure source is frozen with result3096002E... and under
independent review; catalog/IPC is GLM-owned and still in progress. Its final
contract is fixed ID Lookup/Match, ReadRequestFor expected ID, existing decoder
and writer signatures generalized to known pairs. This preparation does not
authorize builds, process tests or touching another agent's files.

## Outcome

A Mindmap adapter advertises `levnote.mindmap-svg` / `svg`; source mode exports
without creating graphview. A separately packaged one-shot executable parses
the exact immutable snapshot, uses the shared native layout and bounded SVG
API, and returns complete bytes plus explicit warnings. Native preview/editing
remain compiled. No installable arbitrary code, remote resources or sandbox
claim. This is the second independent transformation, not full format plugins.

## Worker scope (active owner: read_mindmap_worker_task)

New cmd/levnote-mindmap-svg main and tests only. Use operation-bound request
reader and fixed wire descriptor. One request, EOF, version link flag, no
filesystem/network/paths/configuration/child processes. The normal serve path
uses hard limits; small output-budget test seam remains private. Mindmap parser
limits16MiB/depth256/nodes100000 are retained; SVG cap64MiB. Reuse pure
PresentationIssues and NativeLayoutOptions; no duplicated label/color/SVG code.

Refuse invalid JSON/unsupported format/limits and layout error-severity issues
with stable existing engine error codes, bounded non-source diagnostics and no
partial bytes. Retain benign parse/layout/presentation warnings in deterministic
order. More than256warnings or oversized metadata cannot be silently truncated.
Map no-location issues to [0,len(source)] with documented whole-document scope;
do not imply node offsets. Error metadata must never include raw huge document
content. Strict expected engine must reject Markdown before source allocation.

In-process serve tests prove independently asserted SVG XML/bytes, nonmutating
exact source, whole-document ranges, default/unknown layout, labels/icons/tags,
paint/image disclosure, invalid/deep input, warning and tiny output caps,
foreign request and failing writer. No actual worker launch/build at this gate;
one later bundled app acceptance exercises the real executable.

## Runtime/app scope (later owner: Astra)

Use the accepted fixed catalog in run.go and bundle.go. Runtime accepts only
known ID, derives format/path/platform/limits from compiled rules, then checks
all existing bounded manifest fields against those rules. Manifest Version,
Size and SHA remain runtime facts. Unknown IDs/platform/source size reject
before source writing or launch. No public executable/path/descriptor injection.

Preserve locked ancestor/manifest/executable handles, no reparse, hash checking,
single operation across BOTH engines, hidden Start, Job attachment and every
successful Start/Wait, context cause priority, stdin closure, stdout/stderr cap,
30s timeout/2s pipe wait, descendant join, cleanup poisoning, Close linearization
and historical verified identity. No redundant process framework.

Rename Result.HTML to Payload coherently across runtime, app and all test
consumers (including resource tests' field reads, without executing resource
probes). No duplicate HTML/Payload field retaining two references. Update
Markdown tests only mechanically where rename requires; preserve their oracles.

In app preflight validate the advertised engine/format pair through catalog,
use operation-specific source cap and truthful name in size/status messages.
Existing status currently says HTML for every operation: SVG must be named
correctly, and declared pair mismatch must not silently choose another format.
Add Mindmap ExportEngine only after worker/runtime API is ready. Keep the
single job slot, Flush/snapshot/identity/modal/stale/close/cancel/publication
guards. Failed advertised worker never calls View.Export as fallback. Drawio
retains its existing view export. Source mode must keep View nil.

Mindmap consumer tests cover source/preview/split and readonly; pending field
Flush and pending tag refusal/recovery; stale source/adapter/capability, no
fallback, two-engine serialization, missing/corrupt status, exact preserved
BOM/source/history/baseline and no-overwrite output. Reuse shared lifecycle
tests rather than cloning their entire matrix for each engine. Unit tests use
controlled exporter/SaveDialog; real bundle gate later.

## Focused acceptance before delivery

Owners preserve streams/commands/exits/times and owned source snapshots. Run
affected packages and vet once after focused regressions. Pure/catalog/IPC
results are reused if inputs remain identical. Actual changed runtime tests
may execute their existing owned fake child processes once; this is justified
by the runtime refactor. No unrelated full-repository/resources/UI repetitions.
Root requests independent source review and reads actual evidence before build.

## Delivery later

Extend the existing fixed packaging script to build host plus BOTH workers,
each in its fixed directory with manifest1. Use shared helpers rather than a
second copied packaging implementation; existing script entry remains usable.
Source/provenance/license assets and all build protections remain. Generated
README describes both workers and exact source/output bounds. New version and
fresh folder will be selected by root after source acceptance; no old overwrite.

Add one opt-in Mindmap HostPath acceptance test beside existing Markdown test.
After packaging compile one tester into a disposable complete copy, run both
actual acceptance entries once; each performs success then exclusive refusal,
four requests total. Missing/tampered Mindmap integration gets only a finite
disposable case if runtime/unit evidence leaves an unresolved consumer concern.
Root reads true RUN/PASS, independently asserted SVG/HTML/source invariants,
engine manifests, PE/digests/inventory/licenses/local links and protected old
files before accepting. Packaging success alone is not product acceptance.

Only afterward update current source launch docs and separate acceptance.
No performance superiority or physical OSIME/GPU claim is added by this phase.
Next architecture gap is a bounded read-only parsed-model contract usable by
native preview without importing concrete format parsing; design it after this
second engine proves the static boundary end to end.
