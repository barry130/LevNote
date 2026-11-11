# Composition coalescing `.29` delivery

Delivered2026-11-10 by
`read_native_handoffs_growth_release28_preparation_task_md`.
Status: **AWAITING ROOT ACCEPTANCE**. The `.29` candidate and closed evidence
are frozen by the identities below. Product documentation and picker work
remain planned until root acceptance.

The authorized sequence ran through initialization, baseline, one package
invocation, one copied hidden host, aggregate attempt1 and external closure.
No source test, vet, full parser parse, build, package, host, aggregate or
closure was repeated.

## Accepted source and portable inputs

Baseline passed with the `.29` candidate absent. The root source acceptance is
[active-composition-coalescing-source-root-acceptance.md](active-composition-coalescing-source-root-acceptance.md),
2,775B / SHA256
`59F8AF4AEF765C9E0C725E6759DA2E96C6DC03171998774CA50B9EFD4CD3B7BF`.
The accepted source manifest is
[source-frozen.json](active-composition-coalescing-final-evidence-20261110/source-frozen.json),
628B / SHA256
`5673AB5368E21B7F81631418448C0D4B20B8ECC3A274906E6BE3EBF77EB47213`.
Both source inputs below stayed byte-identical from initialization through
aggregate.

| Accepted source input | Bytes | SHA256 |
| --- | ---: | --- |
| `native/core/text/buffer.go` | 23,353 | `D02837C2AD8EC739E8058B4ECFC479A3C537B85233653902C4E1069079ECE057` |
| `native/ui/editor/editor.go` | 24,397 | `B5D98594BF9213C0D2B0EF2B69C1C00040ED5E365FF133A11BF813AABFD9366A` |
| `native/core/text/composition_test.go` | 12,624 | `C57A16E814798699705F7C3F1062967E6E37E4434C80603E133E3C792F88A3B3` |
| `native/ui/editor/composition_test.go` | 7,948 | `CDACCCD4EBDE965CD3CCD2B8366B843FD483FDFB39B94AC8EB63B87D39ACFF7D` |

The scoped source/build input set was401 files /4,355,553B, identity
`09A051CD60CCC8911EF93FE06DA00CA82A02199FD9EC1C232DF910C4090B4C0A`.
Baseline and aggregate also retained17 root source gates,35 frozen bitmap
files with identity
`2B1D185CB3302427E692B0FE955CE2D0AAC0BAB8F559775B8E522B56BCA10E8A`,
and26 pinned patch-manifest entries.

The portable [README](../portable/README.md) stayed at19,152B / SHA256
`5C209387A537E1307467792B68F08E5FB756178034E89F226D70E4C5DD3FC6B7`.
Its composition paragraph describes endpoint retention for consecutive exact
full-span updates, ordinary grouped undo/cancel/redo, and unbounded suffix
history for irregular explicit ranges. It preserves the inherited `.28` size,
IME retry/Escape and Save/Select/Close barriers.

## Initialization failure and narrow correction

Initialization attempt1 is a preserved harness failure, not a product failure.
It exited1 before delivery context or product execution because the helper
required one contiguous README literal even though the source wrapped the
phrase as `不规则的显式区间` followed by `会安全地追加后缀`. Stdout was empty;
stderr was336B.

The original command, invocation, streams and summary remain under
`native/artifacts/acceptance/composition-release29-20261110/initialize-attempt-1/`:

| Record | Bytes | SHA256 |
| --- | ---: | --- |
| `initialize-command.ps1` | 175 | `77ECE70648A59D922541817008405D38784D7A834F0D849EFC7DA77CE9AE4EE0` |
| `initialize-invocation.json` | 1,008 | `AC70B0B0B8CCD47DE6146383435C301AC8523F82B7D6CE09ADF5A339448BF387` |
| `initialize.stdout.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `initialize.stderr.log` | 336 | `C7D4290EDD7AE03BB57A10827410DEBF73976AB16E2331C234EEB3C22C207EE6` |
| `attempt-summary.json` | 2,126 | `12407E0DA5BD1A5B9064039C374B86384AE4690BD52E0F7C4EB2EFA5B51C08CE` |

The correction changed only that README check: the required phrase became two
actual literals, preserving both checks. No runtime or portable input changed.
The helper moved from30,040B / SHA256
`6E810172D72C0EEF23A5F357A7BB8ED833B0F13F9742E9E51D03DCAF78FAD7BB`
to30,044B / SHA256
`DF304AC2D949F18B59BE3B22ADE06E92964DBB0807F68B58765B592C01719F11`.
One syntax-only parse of the changed helper exited0 with zero errors; its
684B result has SHA256
`5A97E8C4668E8B8E723F76EF18F971D45C82660B8E6A0013687BEFAC370698B9`.

Separate initialization attempt2 passed, exit0. Its invocation is1,008B /
SHA256 `CB9A89BC26FC4C79727602E52799F0B2A31384350BB2E2CE5BE892DDAF741D6D`;
stdout is766B / SHA256
`AA1F83F9E5A6375B3F1231699A04B7CC569114FD5DCEFAA3E288A8286C913C45`;
stderr is the empty stream. Canonical successful records are at the normal
names, with a verified byte-identical copy under `initialize-attempt-2/`.

The previously saved preparation-time static parse covered14 scripts with zero
errors and was not repeated. The preparation report received only
reporting-link correction and its final identity is7,963B / SHA256
`49F090DD9D2C56FC6431433023AD5FC4B94C57E78D8E8506577F090C18E38B45`.

## Package and host

The runnable candidate is
`F:\AIproject\LevNote\native\bin\levnote-native-worker-20261110.29`,
version `native-worker.20261110.29`. Its executable is
`F:\AIproject\LevNote\native\bin\levnote-native-worker-20261110.29\levnote-native.exe`.

| Artifact | Bytes | SHA256 / inventory identity |
| --- | ---: | --- |
| 26-file candidate package | 27,408,380 | `9FAEE40604A7DBA31ED14003011938911661E8BBBE0CCBDC5D9EBCDAA640962B` |
| `levnote-native.exe` | 14,580,224 | `9462194DDC7A310834D564A0A5DE2281A7C799BCE323989D1755CE698CD4CFC6` |
| `release-manifest.json` | 4,674 | `8CCFC58E3D1A19847EF7A55289DE947C7C35D493DCEB1681772250D07AB00722` |

Package passed with one package invocation and five Windows amd64 GUI builds,
each exit0 with empty build streams. The manifest is schema1 with25 payload
entries; the package contains four engines and14 actual portable source copies.
The copied package is byte-identical to the candidate.

Host smoke is PASS with exactly one run and one hidden start attempt. It
executed the byte-identical copied bundle with an isolated `C:` config and
installed local diagram plugin. It opened the accepted UTF-8+BOM/CRLF text and
PNG as ordered CLI arguments, remained alive through the requested5,000ms
settling period (5,009.7743ms observed), and produced a fresh strict two-tab
session with active index1 before close. One owned `WM_CLOSE` produced newer
normal-close publication, exit0, one process in the Job while running and zero
active afterward, with no forced cleanup, process-name scan/kill or worker
request. The process, Job and handles were disposed. The isolated config was
intentionally retained as evidence; no app process remains.

Session bytes and SHA256 were653B /
`B81A6532CA60C86C601E9CD0CB458B2D1899380B4B96AAB511126F0CE5F674D4`
both before and after close. This proves ordered CLI open/session publication
and normal close, not a second-launch restore.

## Aggregate and closed evidence

Aggregate attempt1 is PASS,13/13 checks. The successful attempt is1; no retry
or repair occurred. It checked157 evidence records and retained the single
hidden start, five builds, four engines, zero worker operations and the exact
source, package and `.28` protection identities.

| Aggregate object | Bytes | SHA256 |
| --- | ---: | --- |
| top-level `aggregate.json` | 51,587 | `E5D72F92DE5486118D85A8FDC048E955615FB20FF2E0BB34EF6B9173EEEF6C47` |
| attempt aggregate | 50,910 | `7B2E2D10184FEFC2F4B8EE51DC1EB30CDD7B09548028578CEAA1E21AD0ED0290` |
| attempt validator | 10,529 | `529ADD2C88D89142D5068AE1B37B4E190AD466829A2A4DBC03AC1EB798C9DB7D` |

Stage and outer aggregate exits were0; stage stderr was empty. External closure
ran after all writers exited and recorded all stage invocations as exit0.

| Closed root | Files | Bytes | Inventory identity |
| --- | ---: | ---: | --- |
| `native\artifacts\acceptance\composition-release29-20261110` | 158 | 55,825,749 | `0F46070C6DFDD5CD30824376C2E6D4D9E459849DB45F9353366093EDF8388F89` |
| `native\artifacts\acceptance\composition-release29-20261110-aggregate-capture-attempt-1` | 6 | 3,243 | `1F4E2193425B76BBD06BF5C929B7662BF43758E20EF4236D65D8B643EEA81242` |

The external [closed inventory](../artifacts/acceptance/composition-release29-20261110-closed-inventory.json)
contains164 files /55,828,992B and is69,492B / SHA256
`5BB1C746F4590C1DC40A01F255E4F88E1D5E1C24F4D230047A4BA9A73AA78854`.
All157 aggregate evidence records and the complete `.28` protection set were
unchanged at closure.

## Protection and limits

Baseline and aggregate retained the accepted `.28` package and evidence
byte-for-byte:

| Protected `.28` object | Files | Bytes | Identity / SHA256 |
| --- | ---: | ---: | --- |
| package | 26 | 27,395,153 | `B2BB8C7EB09C2D917882FBFA424D8B921F4504619E37A5DE17E3B10B18E5D8CB` |
| direct evidence | 146 | 55,786,384 | `854427F529599D5912F316C61CD313147D0D15F1105EA9E78ED0C79394F96075` |
| aggregate capture | 6 | 3,193 | `504D8519A9A4DC2BAFFA9EA86F75B57BE2C0B5A427B711C6A8310669E3541A7E` |
| external closed inventory | 1 | 63,713 | `7ADEE2D137C4FC9CB61DCE6FC5C5B3BAF306F3C587B8E4D0700D7973C3FAEA46` |

The accepted source tests establish endpoint transaction retention for
consecutive exact full marked-span updates while ordinary undo/cancel/redo
remains one group. They also preserve the `.28` sticky size refusal and IME
retry/Escape/Save/Select/Close barriers. Irregular explicit ranges keep ordinary
suffix edits and are not universally bounded.

Delivery did not validate physical Windows IME input, second-launch restore,
PNG rendering, worker execution, dirty recovery, screenshots or any resource
measurement. No whole-process RSS/heap saving or universal history bound is
claimed. The hidden host proves the bounded CLI open/session publication and
normal-close lifecycle only.

Root acceptance has not yet been invented. The `.29` package and closed
evidence are frozen pending root review; product documentation or the next
picker implementation require a separate activation after acceptance.
