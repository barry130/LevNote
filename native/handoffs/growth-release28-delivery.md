# `.28` document-growth delivery

Delivered2026-11-10 by
`read_native_handoffs_growth_release28_preparation_task_md`. The prepared
sequence ran through initialization, baseline, package, host smoke, aggregate
attempt1 and external closure. No source test, vet, parser, screenshot,
benchmark, worker request or resource measurement was repeated.

## Package and host

The runnable candidate is
`F:\AIproject\LevNote\native\bin\levnote-native-worker-20261110.28`, version
`native-worker.20261110.28`.

| Artifact | Bytes | SHA256 / inventory identity |
|---|---:|---|
| 26-file candidate package | 27,395,153 | `B2BB8C7EB09C2D917882FBFA424D8B921F4504619E37A5DE17E3B10B18E5D8CB` |
| `levnote-native.exe` | 14,567,424 | `8DB245432AB4C2358667B837A8D0CA7F164EA6CC8696C09D585AF194A2F04C64` |
| `release-manifest.json` | 4,674 | `999B4EAB52C6BC2E54D9ED74121CA3DBD4DFEE85CE9AAE678EF0C6297C895E24` |

The release manifest is schema1 with25 payload entries. Package result is PASS:
one package invocation, five successful GUI builds, four valid engines, byte
identical package-copy verification, expected candidate file set and zero
package stderr.

Host smoke is PASS. It executed one byte-identical copied bundle, with one
hidden start attempt and one observed app start under an isolated `C:` config
and installed plugin. It opened the accepted ordered UTF8+BOM/CRLF
`.diagramnote` followed by the PNG, remained alive through the5000ms settling
window, and captured a fresh two-tab session with active index1 before close.
One owned `WM_CLOSE` produced fresh normal-close publication, app exit0, and
exact process/Job/handle cleanup. Requested worker operations were0.

## Source acceptance and aggregate

Baseline passed with the candidate absent. It consumed the root-accepted,
11-file document-growth source manifest rather than the earlier implementation
manifest. The scoped source set was399 files /4,328,941B with identity
`F01EED43E83BF3B54B295EBB575C953D537F9D1D155619F7D3818F526D4AC875`.
The root acceptance is
`document-growth-budget-source-root-acceptance.md`,2,919B/SHA256
`6FAAA925ED7EED62604DAF96D2AAE51B7C8E5B2458BC651F2FC6D3A61D08B111`;
the final source manifest is `source-frozen.json`,1,727B/SHA256
`B0A212FF61660DDADD9C5A040289FA6057E52B36E3A030B91BA32196A7C8D275`.

Aggregate attempt1 is PASS,13/13 checks. It confirms five GUI builds, four
engines, one hidden host start, zero worker operations,26 candidate files and
the frozen source identity above. Its attempt aggregate is46,305B/SHA256
`3F4983C85DD9B4A6E4EEE8080C33631F569181E705EA72B28A04B780AD07C478`.

The accepted source tests cover size admission and input behavior: ordinary
UTF-8 live content is capped at128MiB; initial or explicitly reloaded decoded
seeds remain available in full; refused replacement/ReplaceAll/indent work is
visible and non-partial; rejected rich-format flushes retain their local draft;
and a refused IME preedit remains pending until a valid retry or explicit
Escape. These are source-test results, not rerun by delivery.

## Closed evidence

Writers finished before the external inventory. The closure contains152 files
/55,789,577B:

| Root | Files | Bytes | Inventory identity |
|---|---:|---:|---|
| `native\artifacts\acceptance\growth-release28-20261110` | 146 | 55,786,384 | `854427F529599D5912F316C61CD313147D0D15F1105EA9E78ED0C79394F96075` |
| `native\artifacts\acceptance\growth-release28-20261110-aggregate-capture-attempt-1` | 6 | 3,193 | `504D8519A9A4DC2BAFFA9EA86F75B57BE2C0B5A427B711C6A8310669E3541A7E` |

The external closed inventory is
`native\artifacts\acceptance\growth-release28-20261110-closed-inventory.json`,
63,713B/SHA256
`7ADEE2D137C4FC9CB61DCE6FC5C5B3BAF306F3C587B8E4D0700D7973C3FAEA46`.
The package and both closed evidence roots are frozen. Publication of this
report does not promote the current `.27` product documentation.

## Protection and limits

Baseline and aggregate retained and verified the accepted `.27` package,
direct evidence, aggregate capture and external closed inventory by these exact
identities:

| Protected `.27` object | Files | Bytes | Inventory identity / SHA256 |
|---|---:|---:|---|
| package | 26 | 27,382,619 | `04740A5B7ABECA69FE5CBDE224B85C26EBC93C9BE418E95B0DC9B0D95AB5B247` |
| direct evidence | 147 | 55,816,795 | `EE8F9D01AC05AAD418C7858E143E8AAF56D0B0B53C3B4B1BC2363BA085C75342` |
| capture | 6 | 3,264 | `B0AC3622B591A12BB7F1A7EA42D3575EFF18D9FF627F18A912A0ACFD19255F87` |
| external inventory | 1 | 65,251 | `E4F9FEE38A2F557E954A3E9F2A453B6A55789E2B78E4230668C15328582BD978` |

The portable source README remained frozen at18,725B/SHA256
`7B25317E6480956154164323D77057883DF1AB17BBB61A868F6FFA8BABDEBFCB`;
the package copy added only its generated package path/version line. The prior
one-time static parse covered14 scripts with0 parse errors and was not repeated.

A preliminary command-dispatch attempt used the unavailable WSL shell and
returned exit1 before the package PowerShell script could start. It created no
stage output and caused no product execution. The package stage was then
invoked once through PowerShell and recorded exit0 with zero stderr. No product
stage failed.

The hidden host run proves startup, ordered command-line open/session
publication and normal close lifecycle only. It did not perform a second
restore launch. It does not validate physical IME behavior, exercise a runtime
size refusal, render or compare the PNG, request a worker, or measure whole-app
memory. The growth and IME behavior above comes from the accepted source tests.
No memory-improvement claim is made.
