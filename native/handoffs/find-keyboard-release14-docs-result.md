# Find keyboard and `.14`: documentation result

2026-11-09. The six-file documentation follow-up is complete and frozen for
root review. The updated documents describe the root-accepted `.14` source and
binary, keep `.13` search/cache/layout and `.12`/`.11` history intact, and do not
turn synthetic keyboard evidence into physical Windows IME or resource claims.

## Changed files

- [README.md](../../README.md)
- [ROADMAP.md](../../ROADMAP.md)
- [CHANGELOG.md](../../CHANGELOG.md)
- [native/README.md](../README.md)
- [native/ARCHITECTURE.md](../ARCHITECTURE.md)
- [native/VALIDATION.md](../VALIDATION.md)

The accepted
[implementation result](find-keyboard-result.md),
[independent reconciliation](find-keyboard-independent-reconciliation.md),
[source acceptance](find-keyboard-root-acceptance.md),
[`.14` package acceptance](find-keyboard-release14-root-acceptance.md), and
[frozen delivery handoff](find-keyboard-release14-delivery.md) were not
rewritten. The delivery handoff remains the same recorded artifact; this result
links its root acceptance without converting candidate wording retroactively.

## Documented behavior

The current runnable path is
`native/bin/levnote-native-worker-20261109.14/levnote-native.exe`. The documents
lead with the accepted user behavior:

- the first `Ctrl+F` or toolbar Find request opens and focuses the query;
  repeated requests preserve the existing query, replacement and options;
- query Enter/Shift+Enter search next/previous;
- replacement Enter replaces the current result only, while Shift+Enter does
  nothing;
- Escape closes Find and returns to the live source in source/split mode;
  preview does not receive an invisible source focus;
- active composition does not execute Enter/Escape actions or `Ctrl+G`; focus
  leaving a field or Plugins hiding Find reconciles transient
  focus/composition state while preserving values and options;
- generation plus tab/document/editor/buffer identity guards reject stale
  callbacks; replacement Flushes, rechecks identity and uses a fresh match,
  while failed Flush preserves text, history, selection and field focus.

`native/ARCHITECTURE.md` contains the implementation contract and
`native/VALIDATION.md` contains the source/test identities, build counts, raw
final-suite/vet log hashes, hidden-host result and explicit limits.

## Accepted `.14` evidence

The package is `native-worker.20261109.14`, 23 files / 23,673,926 bytes. The
14,050,304-byte host has SHA-256
`D037EF7B3DC700E84AA9484AE9A0A7A706662DB8A834D8D3CC4C4B53A3C4FC9A`;
the 4,137-byte manifest has SHA-256
`A876F6E298DCDFA176178E5B2749383AF0CD5053984355751C0A0F10F9BF5379`.
The package acceptance records one package invocation, four GUI builds, one
hidden exact-copy startup with owned `WM_CLOSE`, exit 0, preserved source and
session, and zero managed/worker requests. The final validator is 12/12 with no
retry.

- [`.14` package acceptance](find-keyboard-release14-root-acceptance.md)
- [`.14` delivery](find-keyboard-release14-delivery.md)
- [`.14` validator](../artifacts/acceptance/find-keyboard-release14-delivery-20261109-215637/validator.json)
- [`.13` package acceptance](search-navigation-release13-root-acceptance.md)
- [`.13` delivery](search-navigation-release13-delivery.md)

The `.13` adjacent/cache/layout implementation and its two 760x480 captures
remain historical facts. The `.11` managed old-release/bundled-reset requests
remain historical integration evidence and were not rerun for `.14`.

## Read-only checks

The six documents plus this result contained 367 local Markdown targets after
the edits; all 367 resolved to existing paths (0 missing). Target paths were
checked; HTTP links and heading-anchor identity were outside that path-existence
scan.

The documented `.14` executable, manifest, package README, source/package
acceptances, delivery, validator and `native/portable/NOTICE-PREAMBLE.md` all
exist. Independent reads confirmed the package's 23 files / 23,673,926 bytes,
the host and manifest sizes/hashes above, and validator values of
`Passed=12`, `Total=12`, `ManagedTesterCompiles=0`, `ManagedTesterRuns=0`,
`ManagedProductionWorkerRequests=0`, and `HiddenAppStarts=1`. Inputs-before and
inputs-after each hash to
`5C7F9787268CACAAD33D79FB904EA9A69F221EF256252E76DBAC01D7E49D108A`.

No tests, builds, launches, package writes, product-source edits or evidence
edits were performed. No physical Windows IME/candidate-window, visible UI
automation, CPU, GPU, RSS, working-set, memory or performance result is claimed.

## Post-review wording correction

2026-11-09. Applied the three root-approved wording corrections in
`CHANGELOG.md`, `native/ARCHITECTURE.md` and `native/README.md`: restored the
missing predicate, limited composition protection to Find actions and `Ctrl+G`
rather than panel switching, and removed the implication that refused actions
are queued. No links changed. No tests, builds, launches, package changes or
evidence changes were performed.
