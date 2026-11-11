# Resource harness final preparation validation

ACTIVE2026-11-10 for existing GLM resource owner. Root reviewed source checkpoint
6B546EFC1F19A38DEC603ED9040C183D0ED6F4F69F3B14B3D470FEDC05CC69A6,
40272B and accepts it for bounded preparation validation. No measurement yet.

Keep source frozen. Compile once to a NEW evidence-owned absolute tester path
using native/scripts/go.ps1 -GoArguments; do not overwrite native/drawio-resources.test.exe
or any prior artifact. Run this exact tester once with both gates disabled
(both tests must SKIP), then enable only preflight with a fresh absent output
directory. Capture actual commands, stdout/stderr and exit codes in separate
files; use proper argument arrays and absolute paths to avoid the earlier
PowerShell splitting error. Existing fixture hashes/oracle must remain the
same. The preflight must exercise JSON shape and full12 matrix mapping.

Publish actual final tester/preflight/source identities and result, then
stop. No worker, host, app suite/vet, benchmark or product rebuild. A concrete
validation failure may be corrected with preserved failure evidence; never
silently overwrite or rerun passing product work. Root will schedule the12
real requests only after .22 package/host activity finishes, under a separate
explicit activation. All current accepted package/evidence remains immutable.
