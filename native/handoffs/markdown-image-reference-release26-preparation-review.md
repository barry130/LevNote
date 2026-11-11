# .26 preparation review: retain the accepted strict lifecycle

2026-11-10. Root read the16:20checkpoint and current helper source. The
checkpoint says pre-close state became nullable/descriptive. This change is
outside the active task and reverses the accepted .24/.25 autosave verification.
The preparation task explicitly requires the existing strict pre-close mixed
session check and single normal close. There is no observed .26 failure or
source change warranting this weaker predicate.

Restore the exact accepted .25 run-host-smoke/aggregate pre-close requirements
in the new .26 copies: after5000ms alive settling, capture a freshly published
strict ordered text+PNG session before the single owned WM_CLOSE, then normal
close must publish the same valid session with a newer timestamp. Retain exact
DateTimeOffset conversions and all owned process/Job cleanup. Keep .26 paths,
source identities, protected .25 inputs and updated portable wording.

Do not modify accepted .25 scripts. Reparse only the corrected new scripts once
because this is a concrete source correction after the initial static parse;
preserve the original parse as historical. Update provenance and preparation
result to explain the corrected policy drift. Root has not activated any build
or host execution yet. If a real product failure occurs later, preserve/report
it instead of weakening acceptance or repeating launches.
