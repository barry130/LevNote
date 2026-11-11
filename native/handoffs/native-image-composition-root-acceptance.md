# Image registry and neutral composition source accepted

ACCEPTED 2026-11-10 after root read the actual composition/registry changes,
focused tests, handoff and recorded suite/vet output. All four scoped identities
in native-image-composition-result.md match the workspace. No root rerun.
Final suite records plugins PASS 2.387s and modules PASS 0.822s, exit 0; vet exit 0.

Accepted: builtin `image` before code with readonly read/source capabilities,
enable/disable/persistence through the existing manager, and unaltered schema3
route catalog. Builtins now composes the accepted imageview. DecodeImage takes
context, ReaderAt and size, calls accepted raster.Decode once, passes through
errors, maps PNG/JPEG metadata and transfers the owned RGBA pointer directly.
Real small PNG/JPEG and cancellation/rejection exercises passed. Separate-call
allocation tests establish independent outputs; direct pointer transfer is also
verified by reading the simple composition source, not by overstating that test.

App owns input authority, scheduling, cancellation and UI delivery. This source
acceptance does not establish the still-in-progress application lifecycle or
portable image support. Registry/modules source is frozen while the owner
performs the separately assigned read-only app integration review.
