# Markdown image-reference .26 product documents

Completed2026-11-10 for
[.26 docs task](markdown-image-reference-release26-docs-task.md) after
[.26 root acceptance](markdown-image-reference-release26-root-acceptance.md).
Only the six owned product documents were edited; no product run, build, test,
vet, package or portable/evidence file was executed or changed.

The documents now present the accepted `.26` package as current and preserve
the complete `.25` and `.24` validation/evidence sections as historical records.
The candidate uses the root-accepted identity
`7DB6E2660206EABC4D689894A0B1D50400C6DFA8BAA71CBD3689C6055EE1FB64`, not the
delivery prose's transposed value. The current entry is
`bin/levnote-native-worker-20261110.26/levnote-native.exe`, version
`native-worker.20261110.26`; the package is26 files/27,281,492bytes, host
14,455,808bytes SHA256
`B219024B7C5B444DA0307579C1500821A6B24FDCDD9B4D47E0FC14E863A51B67`, manifest
4,674bytes/25payload entries SHA256
`EFF8DD2FAF05EC620AB5976C2A72E647F12570876B15B13606062B7DC9C4E20E`.

The docs preserve the actual repair/closure history: the sole package call
exited1 after all five GUI builds exited0 with empty streams; the portable link
checker misread an inline code example; only the exact source README example
and candidate README/manifest were repaired; all five executables, four engine
manifests and remaining assets stayed byte-identical. Original failure context,
inventories, streams, missing package-result files and repair parse history are
retained. Aggregate attempt1 exited1 before validator/aggregate creation because
the actual HostSHA256 was missing, so no12/13 verdict exists; attempt2 passed
13/13. No product operation was repeated.

The `.26` feature description remains bounded: explicit `![label](target)`
remains raw text; Ctrl/Cmd+click or Tab+Enter can explicitly open a local target;
current open/readers/byte fallback are reused; PNG/JPEG limits and plugin
enable/disable are unchanged; there is no automatic inline bitmap decoding,
image cache, title form or reference form; HTML preserves escaped literal
Markdown syntax. Targets are1..4096UTF-8bytes excluding ASCII whitespace,
control characters, angle brackets and parentheses. Raw targets preserve local
relative paths, percent encoding and Windows backslashes; existing explicit
`file:` URLs and anchors retain existing host rules. The separately implementing
closed-file workflow is not claimed as part of `.26`.

The one focused docs/link/wording consistency check passed after correcting the
checker's inline-code and angle-bracket handling. It verified stale current
`.25` references, concrete image-reference boundaries, local markdown link
targets and balanced code fences across the six edited documents. No broader
product checks were run.

Final six identities:

| Document | Bytes | SHA256 |
|---|---:|---|
| README.md |17,850|221691AD56CAE9F8405DF02302530FC8506635469B48FA485C853164C9E5723C|
| ROADMAP.md |123,108|A55C374591DA7CDEB83E76DC00EEB1F638F0794375299D9819C9E2D555EA7ACD|
| CHANGELOG.md |151,981|A0C7FE74AE4239C2519C25FD099DAD16A9606378C822F60B29E78490DA693949|
| native/README.md |66,579|438E01C4F549BA1CE8788BE1BB9F499005D0CC60A00A0AB2D20A1A07900E36F7|
| native/ARCHITECTURE.md |68,234|75313F9BFE06DB346BE0116B1D2553519E67D4DC269A95087E90666439EB97D9|
| native/VALIDATION.md |160,242|935F5C1EF8154AC8863CDB112D03C3CDD33E7DD36A6A6CC2B85681FCD534BF5E|

This result intentionally does not embed its own final hash. The actual result
identity is reported separately to root when the file is frozen.
