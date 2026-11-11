# Markdown worker full integration — root accepted

2026-11-09 local. Sole integrator `/root/astra_marker_view`; product/tests stayed
frozen. No delegation, new dependency, resource run or OS IME claim.
Root explicitly accepted the exact final aggregate and PASS20/20validator after
reading actual source/logs/oracles/evidence. `root-acceptance.json` separately
records actual creation time2026-11-08T17:55:18.9616705Z and file-derived hashes.
Its validator digest is actual64hex; some tool displays mask that value.

Final immutable metadata is in
`artifacts/acceptance/markdown-worker-integration-20261109-015232-attempt04/`:

| Record | SHA256 |
| --- | --- |
| `aggregate.json` (106,014 bytes) | `FB0468DADF9F526AFD7BAE72BD1E3893CAD7CEE78176870DDDAE5BB59B12A83A` |
| `validator-result.json` (PASS20/20) | `C32CB352659ED708B05A3A0D4E4D4DA01F33523C3D512FA53D0F3381A5040154` |

Actual full test/vet/package ran once in
`markdown-worker-integration-20261109-013547`: test exit0 wall26.778s,
19tested packages+4no-test packages; vet exit0 wall1.822s; checked-in packaging
exit0 wall3.901s. All stderr empty. Both binaries were built from current source
with explicit windows/amd64/v1 and independently checked PE32+/GUI headers.

Candidate payload remains in that first directory's `bundle/`:

| File | Bytes | SHA256 | UTC mtime |
| --- | ---: | --- | --- |
| `levnote-native.exe` | 13,613,568 | `4B5DCECA5E5F5C3D3609336563F138247942E35E0C3BE65B382B32771A80B23C` | `2026-11-08T17:38:51.9981206Z` |
| `engines/markdown-export/levnote-markdown-export.exe` | 3,352,576 | `DB8F02E6E187833FB52B4FC02B076B4AC96E357FB3E554533A698D8F050F6AD0` | `2026-11-08T17:38:49.8028912Z` |
| `engines/markdown-export/manifest.json` | 337 | `64472251BD737D3BB82AFB05C35F648882D2E9D8D4A57BCFD1CDE8CF457CC453` | recorded in aggregate |

Real bundled host-path test in attempt03 has explicit RUN/PASS0.21s. It uses
registered Markdown source/noView capability, real fixed NewBundled worker and
only an injected SaveDialog. Both real exports joined; HTML331bytes,
SHA256`18FBE1F3BE429C7AF1F975E208933F08E154A6C2396FF75CDAB61D516356B758`,
one warning, second same destination refused. Complete source bytes/BOM/encoding,
version/selection/history/dirty/disk baseline stayed unchanged. Independent
heading/task/table/literalHTML/unsupported-link expectations passed. Actual
production worker PID was not observed; cleanup guarantee comes from the
reviewed/tested runtime return contract, without inventing a PID observation.

Separate disposable missing and same-size one-byte-corrupted worker copies
produced actual RUN/PASS0.01s/0.02s, typed unavailable/integrity, no dialog,
fallback or target. Candidate payload hashes stayed unchanged.

Actual hidden native startup is in
`startup-smoke-markdown-worker-20261109-014720-attempt02`: command exit0,
wall4.822s (inner smoke4.267s). Exact held PID19384 queried by
QueryFullProcessImageName, hidden window found, owned WM_CLOSE exited0 with no
timeout/force/survivor. Unrelated held sentinel36776 survived app close, then
exited0 on its own event. Post-close isolated session confirms opened readonly
UTF8BOM Markdown without draft. Three binary/manifest hashes and four fixture/
sentinel hashes match; actual app stdout/stderr empty. No hidden idle-autosave,
interactive-dialog, GPU or OS IME acceptance is claimed.

All174pre-run source/test/contract/build fingerprints remain unchanged, as do
the16bottom-layer and11app checkpoint files. Protected `native/bin/levnote-native.exe`
remains13,461,504bytes/SHA17E014.../mtime16:04:08.3209774Z. Four focused760x480
images were already viewed by owner/root; rendered states unchanged, no rerender.

Preserved harness failures: first `test -c` rejected by existing case-insensitive
`-C` safety guard; attempt02 used `test -run '^$' -o` and ran zero tests while
writing the tester. Attempt02 bare dotted native flags were parsed as `-test`;
attempt03 literal string-array flags ran the intended tests. First smoke
MainModule.FileName was null immediately after launch; held child29112 was
killed+Wait, sentinel36092 exited0. Corrected smoke used exact handle image query
and post-close session oracle. First aggregate552510.../validator19/20 are
preserved: PowerShell7.6 JSON auto-date conversion caused a false timestamp
comparison; attempt04 explicitly compares UTCticks and passed20/20. No product
rebuild or successful-stage rerun occurred for these harness corrections.

Root authorized final docs/clean launch asset copy after exact acceptance.
Clean destination: `native/bin/levnote-native-worker-20261109.1/`, only
accepted host/engines payload plus LICENSE/notices/exact license texts/portable
README; no evidence scripts/tester/logs. Candidate stays immutable, old bin exe
preserved. Program directory is movable; user config remains APPDATA. Full
renderer memory is not bounded by the64MiB post-render result check, final file
publication remains synchronous/noncancelable once started, and SHA agreement
is integrity consistency rather than authentication or sandbox. Packaging
automation follow-up remains separately queued.
