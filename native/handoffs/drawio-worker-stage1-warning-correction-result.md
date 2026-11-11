# Fourth worker stage1 warning correction: result

Status: correction implemented, worker-only checks passed, product source/test
and evidence frozen for independent review.

Active task:
[`drawio-worker-stage1-warning-correction-task.md`](drawio-worker-stage1-warning-correction-task.md).

## Concrete finding and correction

The initial stage1 worker conversion path allocated `relevant` with capacity
`len(parsedIssues)` before filtering warnings and before enforcing the IPC limit.
`wireWarnings` similarly allocated from `len(issues)` before refusing an excessive
count. Large warning populations could therefore create a proportional
intermediate slice even though the wire contract can accept at most 256 warnings.

The correction in `native/cmd/levnote-drawio-svg/main.go` now:

- Counts document/selected-page parse warnings first and refuses when they exceed
  the supplied capacity.
- Allocates parse-warning conversion storage only after the relevant count is
  bounded, then converts only those relevant issues in their existing order.
- Gives emitter conversion `WarningsMaxCount-len(parseWarnings)` as its own limit,
  so the combined parse-plus-emitter output is checked before emitter conversion
  storage is allocated.
- Checks `wireWarnings` count bounds before allocating or converting and shares
  the existing strict warning-component validation through `wireWarning`.
- Leaves parser and emitter internal issue allocation unchanged. This bounds the
  new IPC conversion layer; it is not a whole-process memory cap.

Ordered document/selected parse warnings followed by emitter warnings, exact
selected SVG bytes, strict raw component checks, nil payload on failure, and an
empty warning array on failure remain unchanged.

## Tests added or strengthened

`native/cmd/levnote-drawio-svg/main_test.go` now covers:

- 4,096 warnings on an unselected page are filtered without appearing in output.
- 257 relevant parse warnings are refused before conversion allocation.
- Emitter conversion with zero remaining capacity is refused.
- A combined parse-plus-emitter total of exactly 256 is accepted in original
  order, with the two emitter warnings last.
- A combined total of 257 is refused with nil payload and no warnings.
- Existing strict component, type, UTF-8, count, warning-scope, and ordering
  regressions remain in place.

No focused source test failed in this correction run: the bounded implementation
was present before the first worker-only test execution. Historical stage1
failures remain preserved in the original stage1 evidence and are not rewritten
as failures of this correction.

## Verification

Focused worker test:

```powershell
& .\native\scripts\go.ps1 -GoArguments @('test','./cmd/levnote-drawio-svg')
```

Observed output:

```text
ok  	levnote.local/native/cmd/levnote-drawio-svg	0.728s
EXIT_CODE=0
```

Worker vet:

```powershell
& .\native\scripts\go.ps1 -GoArguments @('vet','./cmd/levnote-drawio-svg')
```

Observed result: no output, `EXIT_CODE=0`.

Raw logs:

- [`focused-1.log`](drawio-worker-stage1-warning-correction-evidence/focused-1.log),
  SHA256
  `A46E2B2D3BA0B8318069226FFFF5A3E60E2800583D28DED9E2B684F563197A1C`
- [`worker-vet.log`](drawio-worker-stage1-warning-correction-evidence/worker-vet.log),
  SHA256
  `4CFCB797DD42CAB1453791E3065BCE42F1FE41A94D27EC2719A71C21AA0080B9`
- [`final-hashes.txt`](drawio-worker-stage1-warning-correction-evidence/final-hashes.txt),
  SHA256
  `40FAD07FD4E021647CBF8C7E13ACE0A32F0E7A45E00B855A8F05EBB736C3E998`

## Frozen files

| File | Bytes | SHA256 |
| --- | ---: | --- |
| `native/cmd/levnote-drawio-svg/main.go` | 7,228 | `1B8EA397373D1CD3A2CC6F09D4FA6B9EA27E81B6F2FC0241722E7D7D4B85BE20` |
| `native/cmd/levnote-drawio-svg/main_test.go` | 16,601 | `06007F93A9CCF105A16943B0F780322D5103266D84808145F2BF28FDE03E88E5` |

These hashes match the files at result publication time.

The original frozen stage1 record was not modified:

- `native/handoffs/drawio-worker-stage1-result.md` SHA256
  `C476EF99BBAE3F8E7145274BEE0F1F9A7AC0AAA1FAF321D9C5E453F7308BF3BA`
- `native/handoffs/drawio-worker-stage1-evidence/final-hashes.txt` SHA256
  `072828726C48B3BBE390B80FE5BFD0707955307B1CDFA72038CFA22FC2E2AEBE`

## Scope and freeze

Only `native/cmd/levnote-drawio-svg/main.go`,
`native/cmd/levnote-drawio-svg/main_test.go`, correction evidence, and this result
were changed. No SVG emitter, protocol, app, UI, runtime, format, script, product
doc, packaging, or `.18` artifact change was made. No build, launch, resource
measurement, or full eight-package suite was run.

The corrected product source and tests are frozen for independent review. No
further source, test, or other product edit is active under this task.
