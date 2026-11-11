# Managed fixed-engine runtime — active implementation

2026-11-09. Astra astra_marker_view owns this finite slice. Core store and scene
refresh are accepted. Read the existing runtime-next-stage/preflight once; this
activation supersedes their queued ownership. No app, worker, wire, catalog,
packaging or accepted .8/.7 changes in this slice. No child delegation.

Implement NewManaged(*PackageManager) (*Runner,error), preserving NewBundled.
Reject nil/uninitialized manager without filesystem enumeration. Corrupt load
state remains available for explicit app repair, but every request must refuse
it with the actual error; no bundled fallback. Capture immutable selection and
load status at Export admission, before releasing its admission lock. Keep
request selection fixed even if the Manager later changes. This is bounded
metadata access only; no hashing while holding Runner's admission mutex.

Use the private selection/verifiedInstalled seam for exact selected-package
verification. Hold all package files/ancestors until launch/Wait/cleanup has
finished. Do not verify then close and reopen, accept generic executable paths,
or discard guard ownership when adapting heldPackage to verifiedBundle. Add
Source and PackageDigest (or one consistent equivalent) to EngineInfo, keeping
ID/version/exeSHA separate. Bundled identifies Source=bundled and no invented
package digest. Installed metadata comes from verified bytes and captured digest.
Keep existing cancellation, timeout, single-job admission, Close barrier and
cleanup poisoning semantics. Correct user errors that incorrectly call an
installed engine bundled. Source stays immutable and failed payloads absent.

Publish actual API/source checkpoint promptly. Add meaningful finite tests for
constructor/refusal, admission snapshot after selection changes, installed
integrity/missing file no fallback, identity metadata, handle ownership and
cleanup. Reuse existing store fixtures and process seams. Prefer controlled
cancel after verification for selection tests. If a controlled child is needed
to prove installed launch/Wait ownership, run only that relevant test helper,
never an untrusted imported program and no full resource/process matrix.

Run affected runtime tests plus relevant existing bundled verification tests,
then scoped vet once after implementation. Use fresh evidence with exact args,
real exit codes/raw streams/input hashes. Don't re-run unchanged core/parser
cases just to increase counts. Record any failures and smallest correction.
Final handoff: fixed-engine-package-runtime-result.md. Freeze completed runtime
source; root reviews it and then activates app, without waiting for user input.
