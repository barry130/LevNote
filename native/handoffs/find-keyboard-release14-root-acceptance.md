# .14 Find keyboard package: root acceptance

2026-11-09. Root accepts `native-worker.20261109.14` as the current runnable
native package. The `.13` package, its complete evidence and all earlier
releases remain immutable historical records.

Candidate:

```text
F:\AIproject\LevNote\native\bin\levnote-native-worker-20261109.14
```

Version `native-worker.20261109.14`; 23 files / 23,673,926 bytes. Host is
14,050,304 bytes with SHA-256
`D037EF7B3DC700E84AA9484AE9A0A7A706662DB8A834D8D3CC4C4B53A3C4FC9A`.
Release manifest is 4,137 bytes with SHA-256
`A876F6E298DCDFA176178E5B2749383AF0CD5053984355751C0A0F10F9BF5379`.
Root independently read and hashed the actual candidate files.

The delivery used one existing package-script invocation and exactly four
hidden Windows amd64 GUI builds; every build and outer stage exited 0 with empty
streams. The aggregate validator passed 12/12 without retry. Payload manifests,
three engine files, PE type, 13 portable assets and the byte-identical evidence
copy match. Scoped inputs were 293 files / 2,962,619 bytes and were unchanged;
the complete `.13` package (23 files / 23,655,494 bytes) and its 102-file
evidence inventory also remain unchanged.

Root inspected `validator.json`, `aggregate.json`, the package build records,
the hidden host result and the retained session. One exact copied bundle ran from
`F:` with run-owned configuration on `C:`; PID 17036 held handle 2288, the
owned window received WM_CLOSE, exit code was 0, no forced cleanup or owned
process remained, and stdout/stderr were empty. The 60-byte UTF-8 Markdown
source retained its bytes/hash/timestamp. The resulting session contains one
UTF-8 tab without a draft (421 bytes, SHA-256
`1957AE8D2AA6CA68C014FBE16E82EF8AC9A05D479F5B0640919AAE1EBCA723E4`).

The accepted product change is the bounded Find keyboard flow and focus-leave
correction already accepted in
[find-keyboard-root-acceptance.md](find-keyboard-root-acceptance.md), with
synthetic App/Tester evidence and final app/editor tests and vet. This package
does not add physical Windows IME, visible UI automation, worker execution,
resource, CPU, GPU, memory or performance evidence. The `.13` layout captures
remain historical; no new capture was needed.

Delivery evidence:

```text
native/artifacts/acceptance/find-keyboard-release14-delivery-20261109-215637
```

Handoff SHA-256: `895FEE985CBC0DA4A18C7FDEA6DFAA6A5B4795F0BF7399E39D7D51799BC1DDA0`.

The six-document follow-up is now active. It may update only the root README,
ROADMAP, CHANGELOG, native README/ARCHITECTURE/VALIDATION, and its own handoff;
it must preserve `.13` history and state the accepted `.14` facts and limits.

Six-doc follow-up accepted after root review and three wording corrections:
active composition actions are refused, not queued; Plugins/tab focus transfer
remains possible; CHANGELOG's missing predicate restored. Owner verified367
local target paths, none missing. Original delivery evidence/handoff untouched.
Final docs result SHA256:
7DE19644172961F344E347F019850628F1AC74C3A3336064C9217456816FD3A0.
No tests/builds/launches for text-only corrections. Next UTF8 save source work
is separate and not part of this immutable `.14` package.
