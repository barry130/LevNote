# Native image `.23` package and mixed-session delivery

2026-11-10. Delivery is complete and frozen for root acceptance. Aggregate
attempt 2 passed 13/13 checks with exit 0 and no failed check. Current runnable
package and launch documentation remain `.22` until root accepts this result.
No product source, test, resource, launch document, or `.22` artifact was
changed.

## Candidate

Candidate:
`native/bin/levnote-native-worker-20261110.23`, version
`native-worker.20261110.23`.

| Artifact | Bytes | SHA-256 |
|---|---:|---|
| `levnote-native.exe` | 14,400,000 | `837A29744F0C833308B5ACC668B0F5FB2628B0710A816BF8D74DB0B516742C86` |
| `release-manifest.json` | 4,674 | `B0650A1E9A90ACAC7A68457252DA559938B1CB6D33E4B1279DE23E2688325FD6` |
| Complete package copy | 26 files | inventory identity `180786B864A37AAC6423B42C95F079D8AA0AF5B18BE4504F2A124DDECF184040` |

The package copy totals 27,214,997 bytes. The release manifest is schema 1,
contains 25 payload entries plus its excluded self-entry, and matches every
candidate payload byte and SHA-256.

The single authorized invocation of the existing package script completed five
Windows amd64 PE32+ GUI builds, each with exit 0 and empty stdout/stderr. It
shipped four fixed engine directories, their manifests, both MyGo patches,
14 portable asset copies, notices, licenses, the frozen 35-file set, and the
plain Chinese portable README. The source README is 14,185 bytes with SHA-256
`D9096B53268B5D0C15B859DB6DB33F13945CD73D1AD251518979698FECC88141`.
No worker was requested.

## Hidden lifecycle evidence

The single host stage executed one byte-identical copied bundle twice, in order,
against one isolated C: config. Both windows were hidden and each process was
located through its exact held handle.

| Run | Process facts | Result |
|---|---|---|
| 1, mixed CLI open | PID 39500; held handle 2380; start `2026-11-10T05:38:43.2417747Z` | text path then PNG path; active index 1; exit 0 |
| 2, path-only restore | PID 41736; held handle 2528; start `2026-11-10T05:38:48.7711054Z` | no CLI paths; exact baseline restored; fresh close publication; exit 0 |

Run 1 began with no session, remained alive through 5,010.4432 ms of settling,
and published a 655-byte ordered mixed session with SHA-256
`E4CCBD188FC2DACB6677336DF33454591678104547C47CA9B98F1B99BC6FBEDF`.
Run 2 preserved that exact 655-byte session and timestamp as its baseline,
remained alive through 5,015.9752 ms, then republished identical content with a
later timestamp after its own process started. Both sessions have active index
1, the custom-route text with UTF-8/BOM and plugin `diagram-notes`, followed by
the PNG record with plugin `image`, `view=preview`, and no forbidden source,
pixel, bitmap, or encoded payload fields.

Each run received exactly one owned `WM_CLOSE`, exited normally with code 0,
left no owned process or active job member, disposed its process handle, and
recorded empty app stdout/stderr. PID, handle, and window inequality are
descriptive; sequential start/finish ordering establishes distinct lifetimes.
The retained isolated config root is
`C:\Users\canace\AppData\Local\Temp\levnote-native-release23-image-smoke-7c1664d4f85645ea803ef8d8c72399f9`.

## Corrections and preserved failures

Initialization attempt 1 failed before context/package/host work because
`-File` could not bind the acceptance path array. Attempt 2 reached the copied
helper but rejected the accepted patch manifest's actual 26 entries against the
copied stale literal 25. Both attempts and their command/stdout/stderr/exit
evidence are preserved.

Only the two copied helper checks were corrected to 26. Their original copies
and before/after hashes are under
`native/artifacts/acceptance/native-image-release23-20261110/helper-corrections/pre-manifest-count/`.
The accepted manifest is 4,463 bytes with SHA-256
`9113120BD2EBCD9A86842153239CDDFF84D1F1BD3F46F5D4F0F152EB4560E16A`;
`.22`'s actual 25-file package count and the preparation helpers remain
unchanged. One scoped parse then parsed all 10 copied delivery helpers with zero
errors, wrote a 3,175-byte result with SHA-256
`55A78436F826B3F4091BB1AB356575A9DB3058EDA72154E34535EE09D7D6F1A4`,
and exited 0.

Initialization attempt 3 used the required saved in-process wrapper with a
quoted eight-path array and splatted arguments. It exited 0 and bound all eight
source gates, composition/app acceptances, 26 patch entries, and 377 scoped
source/build inputs totaling 4,067,620 bytes. The source identity remained
`1E8F820F7A4239E9CCB43B014297BE681F78DF8AD1904B824F7634747EABC928`.

Aggregate attempt 1 failed with exit 1 before producing its attempt result
because it tried to assign `Actual` directly on a deserialized JSON object.
Its command, streams, exit, and empty attempt directory are preserved. The
original copied aggregate and correction record are under
`helper-corrections/aggregate-attempt-1-property/`. The one permitted
metadata-only correction used `Add-Member`; aggregate attempt 2 then passed.
Neither package nor either host was repeated.

Final corrected helper identities include:

| Helper | Bytes | SHA-256 |
|---|---:|---|
| `delivery-lib.ps1` | 21,192 | `986EE892E74BC52BFF640FB4F6ACFC7329836EC2E12F2211B3F35EA0E5DAFA46` |
| `aggregate.ps1` | 34,156 | `7E6EBCCECF273D653F39EF9EA47B0BD6EACE48019A21F0320B5B01EAF9A7CD73` |

The final `validator.json` is 9,584 bytes with SHA-256
`54E4CFF9D9F92D169B2250E3AEF0260C7B42AEBFBE19979BAFDF4F8FCC491B08`.
The attempt-2 aggregate is 53,870 bytes with SHA-256
`05D4D1D016A012ED5DD70BA33D0A5BAD526B5476ABA44173CF76C84BC8152ADB`.
The final [aggregate](../artifacts/acceptance/native-image-release23-20261110/aggregate.json)
is 54,549 bytes with SHA-256
`C7D9A4281AD6C61DB6E6FD8573C9C513F72A80E053E8001FC45DA71978429FC3`.

## Evidence freeze and protection

The direct [evidence root](../artifacts/acceptance/native-image-release23-20261110)
contains 166 files totaling 55,370,832 bytes with inventory identity
`0C450DCC8B12E330B2B1384B4CCEDB6390C84FCDFE3A206CA41A91328FC689AE`.
The aggregate lists 165 files because it excludes itself; all 165 were checked
unchanged after every stage writer exited.

The two preserved aggregate capture siblings are separate from that root:

- attempt 1 failure capture: 6 files / 2,688 bytes / identity
  `72A933EFEE75DBEAE56224A957E2928672D3469FA05D7BBDF15047156684BD3C`;
- attempt 2 success capture: 6 files / 3,254 bytes / identity
  `72E98E3E02D9076C1EAE462E0699EAB1EBA3B3AE0ACB3035F6828060AABE20ED`.

The external [closed inventory](../artifacts/acceptance/native-image-release23-20261110-closed-inventory.json)
covers all three roots: 178 files / 55,376,774 bytes. The inventory is 75,450
bytes with SHA-256
`E740A6FB3E17901C2BBB74F6A2E6CAA6CF2E1952726FDCCA083194A854317E8A`.

The accepted `.22` package remains 25 files / 26,863,393 bytes / identity
`BA43209FF7F6CFBEDD8287DBEA7B642008AFA839D133FECD8CC5C9D065FD9CF6`;
its 134-file direct evidence, 6-file capture, and closed inventory also remain
byte/time identical.

## Scope and limits

This delivery establishes the .23 package contents and build identities, one
clean mixed CLI open, and one no-argument saved-session restore/close lifecycle.
It does not establish that the image decoded visually or rendered correctly,
physical Windows input or IME behavior, interaction with native dialogs,
clipboard copy, dirty recovery, worker execution, whole-app RSS/VRAM/CPU,
GPU/driver behavior, animation, SVG-image routing, or cross-platform behavior.
The accepted PNG/JPEG subset and admission/lifecycle limits remain those in the
component and app acceptances. No tests, benchmarks, captures, worker requests,
resource runs, process-name scans, or forced process cleanup were performed.
