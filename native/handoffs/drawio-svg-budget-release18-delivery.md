# Draw.io SVG output budget: .18 delivery

Status: **CANDIDATE READY FOR ROOT REVIEW** 2026-11-10. The actual
`native-worker.20261110.18` package, four builds, aggregate validation, and one
owned hidden host smoke have passed. `.17` remains the accepted current package
until root accepts the actual `.18` binary and updates product documentation.

This delivery changed evidence scripts only under a new acceptance directory.
Product source, tests, package scripts, and product docs were not edited, and no
test, vet, benchmark, capture, managed tester, or production worker request was
rerun.

## Candidate and package

`native/bin/levnote-native-worker-20261110.18` contains 23 files / 23,699,526
bytes. Its complete inventory identity, including relative path, byte count,
SHA-256, and recorded modification ticks, is
`4A4F9BD0B6DF2DDCD1E597F026AAC985A9D50711CDD85F3E9310F7DC5B0B2991`.

| Item | Bytes | SHA-256 |
|---|---:|---|
| `levnote-native.exe` | 14,075,904 | `BFA6F460F49087390F114CEC2CF2ADC034307B07769D8B64E3E70C78BBDCB44A` |
| `release-manifest.json` | 4,137 | `8CA2016C0064638BE965FDFE74468F8004A37143BE568A189FD7FEDAB2D12FDD` |

The one package-script invocation exited 0. Each of its four GUI builds exited
0 with zero stdout and stderr bytes:

| Build | Output | Wall time |
|---|---|---:|
| `worker-markdown` | `levnote-markdown-export.exe`, 3,300,352 bytes, `A7C33B9F81FA1836A456C64B580982E7BA1E3350BD13FC14F81BF293923B45B0` | 0.979 s |
| `worker-mindmap` | `levnote-mindmap-svg.exe`, 3,122,688 bytes, `9A35C7B5E454368887350EE378AF97137EF9B431B422298C3494062452E81768` | 0.946 s |
| `worker-mindmap-preview` | `levnote-mindmap-preview.exe`, 3,136,512 bytes, `174A68224FCE7FC140E376982A9B9F437CA861312CDC83B26CD8636CC7526F78` | 0.935 s |
| `host` | `levnote-native.exe`, 14,075,904 bytes, `BFA6F460F49087390F114CEC2CF2ADC034307B07769D8B64E3E70C78BBDCB44A` | 2.117 s |

The complete candidate copy is byte-identical at 23 files / 23,699,526 bytes.
The release manifest names `native-worker.20261110.18` and all 22 payload
entries match exact sizes and hashes. The three worker manifests independently
carry the same version and match their payload size/hash. All four executables
are Windows amd64 GUI PE images (`machine=0x8664`, PE32+ `0x20b`,
`subsystem=2`). All 13 portable assets retain their source-copy hashes, and the
package script checked local authored links.

Raw records:

- [package command](../artifacts/acceptance/drawio-svg-budget-release18-20261110-011852/package-command.json)
- [package result](../artifacts/acceptance/drawio-svg-budget-release18-20261110-011852/package/package-result.json)
- [package-copy result](../artifacts/acceptance/drawio-svg-budget-release18-20261110-011852/package-copy-result.json)

## Hidden host smoke

The smoke copied the complete 23-file bundle to a fresh F: evidence path,
started that exact copy once with `Start-Process -WindowStyle Hidden`, and used
the query path from the exact held process handle. It used an isolated run-owned
APPDATA/LOCALAPPDATA root on the normal C: temp volume and opened one unchanged
generated Markdown source.

The native window title was found while the window remained hidden. One
`WM_CLOSE` was posted to that owned window. The app exited normally with status
0, without timeout or forced cleanup. The source and both candidate/copy
bundles remained unchanged. The isolated post-close session contains one clean
tab with the source path, no draft, and no terminal LF/CR. Both app streams were
empty, no worker operation was requested, and no owned process remains.

Raw records:

- [host result](../artifacts/acceptance/drawio-svg-budget-release18-20261110-011852/host-smoke/result.json)
- [clean session](../artifacts/acceptance/drawio-svg-budget-release18-20261110-011852/host-smoke/session-after-close.json)
- [host outer invocation](../artifacts/acceptance/drawio-svg-budget-release18-20261110-011852/outer-host-smoke-invocation.json)

This smoke establishes startup and clean close only. The accepted SVG source
tests establish emitter behavior. It does not prove SVG export through the
packaged host, physical UI/IME behavior, a dirty Save, process memory, CPU use,
or a resource improvement.

## Validation and frozen inputs

The final aggregate validator passed **12/12**, exit 0:

- [validator.json](../artifacts/acceptance/drawio-svg-budget-release18-20261110-011852/validator.json)
  SHA-256 `0111BF0D6A04A82EBC74A54305DF359E2056FBE10DFF770480D348EC2FFD1D47`.
- [aggregate.json](../artifacts/acceptance/drawio-svg-budget-release18-20261110-011852/aggregate.json)
  SHA-256 `D1066FA721005A7F4DC97184BC8E1E802B8FE86013609B5A2F64975D81E14FF0`.

The scanner observed 302 source/build inputs / 3,027,991 bytes, including 62
`native/formats` files and the referenced portable
`NOTICE-PREAMBLE.md`. The before/after inventory record is identical with
SHA-256 `B4681E2EB1E761D2D0D8F811E75D24663727C1050B6C4921B5EE6411976B02B1`.

The six accepted SVG source/test/oracle identities all matched:

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/formats/drawio/types.go` | 5,089 | `233025C35D1A6C33832E47D8782A07A367954F1EEF70754FB83A5BE436C7D845` |
| `native/formats/drawio/svg.go` | 15,572 | `47F3E584EEF6E2AD981FAE539EE6E2ECA3E93DDBD16F22B7DEC3D5FA086944CF` |
| `native/formats/drawio/drawio_test.go` | 13,581 | `A1A777227507637928A1920FF9B253F75C6E4A1EFEA697BB087F74CB8D18F189` |
| `native/formats/drawio/svg_budget_test.go` | 10,737 | `79C1F18B12D35C238FA76E5F919BCA65FEE734AA62FC40BEB48FE43797261801` |
| `native/formats/drawio/testdata/svg-prechange-oracle.json` | 6,688 | `14D7C7166E047979ACDBCBE670EE6918CF44892D54680AAFAC7B0B81944A6611` |
| `native/modules/graphview/graphview_test.go` | 9,858 | `3F8B264305F77F1D6EF2D1468742E8F62DB9248631EB7AB675A1EE213CDE18A0` |

All 23 accepted SVG gate/handoff/evidence files remained unchanged, and the
complete accepted `.17` package remained 23 files / 23,690,310 bytes with host
SHA-256
`E4E81A30B0CB47F080379BD75EEEBD003F4621B3A8393D344073D32DA94EBC1C`,
manifest SHA-256
`668C94BCDAC492BE88D96AE45A73157B7B5B7427C001FC2F4452DF0BB6E3BD08`,
and inventory identity
`669CAFCBBD6DD24DD4AF5F9084AFE9299A6271CEBE11BCC58A8975F0242D5D87`.
Its direct `.17` evidence also remained 117 files / 47,784,783 bytes with
aggregate SHA-256
`B7094AFE989AD9D35A59CCBB4E7312A4BCDC55440AFF2061A50A08FACD3253ED`
and inventory identity
`41FCCE7072CD36F3955C859DB54E056D453EA2A21BC19552916624601A8367B4`.

The fresh `.18` acceptance directory contains the scripts and raw stage/build/
host records used above. No failed delivery attempt was replaced or removed.

## Scope

The change under delivery bounds serialized SVG output bytes. It does not bound
process memory; parser/source structures, page indexes/issues, temporary
fragments, and the successful output buffer remain outside that byte cap.
Draw.io export stays in the host, and the package still contains the host plus
three fixed workers. The fourth-worker plan is not implemented here.

No physical IME/UI automation, packaged SVG export, dirty recovery/worker run,
process-memory/CPU measurement, or performance claim is made. Root review of the
actual `.18` binary remains the acceptance gate.
