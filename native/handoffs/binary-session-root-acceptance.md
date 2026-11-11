# Binary path-only session source accepted

ACCEPTED 2026-11-10 by root after source, tests and recorded evidence review.
Root performed no test/build/launch/resource run. Current portable .22 remains
unchanged; image tabs still require the separate app integration stage.

Reviewed binary.go, builder.go, load.go, output.go, budget.go, session.go and
both new test files. The eight scoped SHA256 values in
binary-session-implementation-result.md match the actual workspace files.
Final session suite output records PASS in 1.538s, exit 0; vet records exit 0.
Those passed checks are not to be repeated in isolation without a new concern.

Accepted: a separate binary wire object in the ordered Version 1 tabs array,
required explicit discriminator/kind and absolute path, no pixel/source payload,
strict rejection of unknown binary fields and forbidden text fields by presence,
pure RestoreBinary, and explicit ErrBinaryDispatchRequired from RestoreTab.
Invalid entries leave neighboring drafts and valid active-index mapping intact.
Text-only serialization retains its existing JSON encoder and output bytes;
mixed output projects small binary records and unchanged text values without a
per-text MarshalJSON copy. Load field values slice the already bounded input.
Builder clone, exact JSON-length budget and pre-filesystem rejection regressions
cover the new branch.

Integration must call RestoreBinary before text restoration, retain only metadata
for inactive image tabs, and serialize via session.Save rather than marshaling
the public Snapshot directly (binary authority intentionally has a distinct save
projection). This acceptance establishes the session contract, not app recovery
or whole-process memory behavior. GLM moves to image registry/composition; session
source is frozen unless a concrete integration defect requires an owned repair.
