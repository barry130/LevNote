# Native reading .24 root acceptance

Accepted 2026-11-10 after reading the final delivery, actual aggregate/validator,
pre-close and normal-close session facts, corrected closed inventory, and
matching the principal artifact hashes. Root performed read-only review only.
Implementation, verification, packaging and host execution were delegated.

Current runnable entry is `native/bin/levnote-native-worker-20261110.24/levnote-native.exe`.
This accepts the image keyboard/explicit100% and paint-independent recovery
increment described in [source acceptance](native-autosave-keyboard-root-acceptance.md).
The final affected app/imageview test and vet passes are inherited, not repeated.

| Artifact | Bytes / count | SHA256 or inventory identity |
|---|---:|---|
| Complete package | 26 files / 27,226,613 B | `C17C919EB5F71C0643CAA9BBF1F5C9CCD509EE2CC91DD8A7FD704F15D06427B5` |
| Host | 14,410,752 B | `A4D679FB0F35DB9258410DCC35995853780DB4E915C928E8D6434A92E6A0BD08` |
| Release manifest | 4,674 B | `B1609AE10D15DBEA63DD40CC7E699E131E49156FC48821D8FD63891E565D935E` |
| Final aggregate | 51,028 B | `D512E569D75D5250F69A6F358D9455BB48E56A8444A439BA53D4B93233E3DA9E` |
| Attempt2 validator | 10,605 B | `07ACBC85744DA61AC6348D156294F7226B6026D148BBE637047B1DC282FFF77F` |
| Corrected closed inventory | 73,016 B | `D9FA85BBC0F9906FF75874B1851ACD0E1046256CD5B5FB99D29976F2CE94328C` |
| Delivery handoff | 7,676 B | `E1D4E1839F805902CFBF5315C582C7A6D51196859B7303014CE9B33AC60D6B23` |

[Delivery](native-reading-release24-delivery.md) links the saved commands and
failures. One existing package invocation produced five GUI executables, all
builds exit0. Final [validator](../artifacts/acceptance/native-reading-release24-20261110/aggregate-attempts/attempt-2/validator.json)
is13/13. Four engine manifests,25 manifest payload entries,14 portable copies,
licenses/patches and380 build inputs matched. Complete accepted .23 package and
its evidence stayed unchanged.

Exactly one hidden copied host in a fresh isolated configuration opened the
UTF8+BOM/CRLF custom-format text followed by PNG. Before its single owned
WM_CLOSE it published the strict659-byte ordered mixed session (active1), SHA
`800045175388CAB4DF70B11A5650D8072CA70CB8A7AE0EF02C943B71C5CE2E04`.
Capture06:42:12.8862702Z preceded close06:42:12.8920834Z. Normal close published
the same bytes with a newer write timestamp. Exit0,Job total1/active0, no forced
cleanup, both owned handles disposed and empty process streams are recorded.
This checks clean-session persistence through the real UI dispatcher; it does
not establish physical dirty editing, minimized IME behavior, rendering or
whole-process resource improvements. No worker requests or repeated source
checks were run for this delivery.

Both metadata defects retain their original evidence: aggregate attempt1 lost
fractional seconds while converting deserialized times; attempt2 corrected the
three time casts only. The first external inventory had zero footer totals;
the corrected file sums the already correct saved per-root records. Neither
correction reran product work or changed closed product evidence.

Closed evidence: direct155files/55,785,316B, identity
`E44D08DF592A38A7BF26309E3E50052F9EE39A61330B54B18ED7FA1146DF36A9`;
capture1 six files/2,851B, identity
`EFE0346259C5BD19709646F30CBDADAFD8484A4FB20BEA03A851746E9A6C7870`;
capture2 six files/3,274B, identity
`F37278A498615CB9821DA29758A0FACBE8FE2479A6FBE0167BBF466183E656C4`.
Total167files/55,791,441B. Package, portable build input and evidence remain
immutable. The workspace source freeze is released only for separately assigned
table-column work; .24 documentation is activated in its existing task.
