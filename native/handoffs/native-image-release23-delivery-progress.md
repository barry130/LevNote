# Native image `.23` delivery progress

2026-11-10. Delivery is active in
`native/artifacts/acceptance/native-image-release23-20261110/`. Ten reviewed
helper scripts were copied byte-identically from the accepted preparation
directory; `helper-copy.json` records the source/destination identities.

Initialization attempt 1 failed before creating `context.json`: PowerShell
`-File` did not bind the eight path-array arguments. Its command, stdout,
stderr, and exit are preserved as `initialization-*`.

Numbered attempt 2 reached the copied helper through an encoded in-process
command, but stopped before context creation because `delivery-lib.ps1`
expected 25 pinned patch-manifest entries while the root-verified current
manifest has 26. Its command, stdout, stderr, and exit are preserved as
`initialization-attempt-2-*`.

Root note 2 authorizes changing only the two copied helper literals:
`delivery-lib.ps1` and `aggregate.ps1` must accept the accepted 26-entry
manifest. Preparation remains unchanged, as does `.22`'s actual 25-file
package count. A scoped parse will run before initialization attempt 3.

The two copied helper checks were corrected to 26 after preserving the original
files and identities under
`native/artifacts/acceptance/native-image-release23-20261110/helper-corrections/pre-manifest-count/`.
The correction record includes before/after hashes, the accepted 4,463-byte
manifest identity, and the verified 26-file property. One scoped static parse
then passed all 10 copied helper scripts with zero parse errors (exit 0).
Its command, stdout, stderr, exit, and result are separately named under
`helper-corrections/`.

Initialization attempt 3 used a saved in-process wrapper with an eight-string
quoted acceptance array and a splatted argument table. It passed with exit 0
and created the execution-authorized delivery context: eight source gates,
26 patch-manifest entries, and 377 frozen source inputs were bound.

Baseline then passed with exit 0. It confirmed the candidate was absent before
packaging and preserved the accepted source, patch, fixture, guarded-file and
`.22` inventories.

The one authorized package stage passed with exit 0. It invoked the existing
package script once, completed all five PE GUI builds, produced a 26-file
candidate totaling 27,214,997 bytes, and copied that exact candidate into the
delivery evidence. The host is 14,400,000 bytes with SHA-256
`837A29744F0C833308B5ACC668B0F5FB2628B0710A816BF8D74DB0B516742C86`;
the release manifest is 4,674 bytes with SHA-256
`B0650A1E9A90ACAC7A68457252DA559938B1CB6D33E4B1279DE23E2688325FD6`.
All four engine directories, five PE headers, copied notices/patches, and
the 35-file frozen set passed.

The single host stage then passed with exit 0. It ran the exact copied bundle
twice, in order, against one isolated C: config during distinct sequential
lifetimes. Run 1 opened the custom-route text and PNG paths and published the
expected two-record, active-index-1 session; run 2 used no arguments, restored
the exact saved baseline, republished a strict fresh session, and closed with one
owned `WM_CLOSE`. Both processes were hidden, alive through the 5,000 ms
settling check, exited with code 0, disposed their handles, requested no worker
operation, and left no owned process live. The two close hashes match, while
their publication times differ.

Aggregate attempt 1 failed with exit 1 before writing `aggregate.json`: the
copied metadata validator attempted to add an `Actual` property directly to a
deserialized JSON object. Its command, streams, exit, and empty attempt directory
are preserved. The copied aggregate was saved separately and corrected once to
attach the field with `Add-Member`; no product, package, or host stage was
rerun. Metadata-only aggregate attempt 2 then passed all 13 checks with exit 0.
It recorded no failed checks and preserved the failed attempt. Current launch
docs remain unchanged.

Final checkpoint: the direct evidence root contains 166 files / 55,370,832
bytes (identity `0C450DCC8B12E330B2B1384B4CCEDB6390C84FCDFE3A206CA41A91328FC689AE`);
all 165 aggregate-listed files were rechecked unchanged. The two aggregate
attempt captures remain separate at 6 files each. The external closed inventory
covers 178 files / 55,376,774 bytes and is 75,450 bytes with SHA-256
`E740A6FB3E17901C2BBB74F6A2E6CAA6CF2E1952726FDCCA083194A854317E8A`.
Final delivery handoff is `native-image-release23-delivery.md`. Package and
evidence are frozen for root acceptance; launch documentation remains `.22`.
