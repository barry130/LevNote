# `.28` docs narrow correction completed

Completed2026-11-10 under `growth-release28-docs-narrow-correction.md`. This
follow-up changed the `CHANGELOG.md` typo from `宿主导出0` to `宿主退出码为0`,
corrected the docs result's description of the original failed checks, and
updated the final documentation identities. No runtime, portable README,
package, accepted evidence, source, host, worker or product artifact changed.
No product check ran.

## Original check preservation

The original consolidated docs check had aggregate exit1. Its raw stdout/stderr
and inline script were not saved, and no reconstructed original log is offered.
The exact six-file pre-edit backup remains at:

`C:\Users\canace\AppData\Local\Temp\levnote-growth28-docs-before-2716ca43304349499ac1f36b5f02e4d3`

The prior result now states that two comparison expressions were over-broad and
one was newline-sensitive. It does not characterize those failures as
deliberate.

## Justified narrow check

One inline correction check compared only the unresolved items against that
backup and the current docs. Raw result:
[growth-release28-docs-narrow-correction-check.json](growth-release28-docs-narrow-correction-check.json),
2,794B/SHA256
`3947219B2FE9BE98D334CEBF267244F71B2F5DB9AFB224BED5342D539230AC20`.

| Narrow assertion | Result |
|---|---|
| Root README `.27` paragraph, before vs after without the added `历史交付` label prefix | PASS, both `0D2A64F58E2000BE2ED2ACE42C634D57F2E0DEE414A2A658211C960F1B075891` |
| Architecture `.21` paragraph, exact before vs after | PASS, both `089DB805FD53F7EADE93C076A0C8F5026C8485ED8B642B53B6D850C153524960` |
| Delivery phrase after normalized whitespace | PASS: old `mixed-session recovery` absent; `ordered command-line open/session publication` and no-second-restore wording present |
| CHANGELOG exit-code typo | PASS: `宿主导出0` absent; `宿主退出码为0` present |

The narrow checker's inline script was not saved; its JSON result was saved
without reconstruction. This was a scoped correction needed to resolve the
original exit1, not a second broad docs check.

## Final identities

| Path | Bytes | SHA256 |
|---|---:|---|
| `README.md` |20,885|`C877577DF62F36D914BF3C998CDDB7A737C12E386C08D8BBF287FFA6371A81DE`|
| `ROADMAP.md` |125,180|`C0C5B894080D7078D89B5764E55EA9A0A86EF1932B703626EDA38A34712D1F01`|
| `CHANGELOG.md` |156,129|`A40AF07961BF84D3766188927862BA77EFFEA764B62A605FBBCC856AC391D098`|
| `native/README.md` |72,429|`B3E26A0BC6546608231F2E32E348ADE053848F91CCFE3DB8FA44EDCCF8F01DD0`|
| `native/ARCHITECTURE.md` |72,086|`569C3809F8B48C813101269E124C3E5587F9619E50CE1DF3AD6C5DA08501B578`|
| `native/VALIDATION.md` |171,577|`63FCF0BAC61101ACA62C14540DFE9125082EE79A217AF7F61240113C7BED4DB7`|
| `native/handoffs/growth-release28-delivery.md` |5,576|`1A718D7BD144B4AF1911768FEF72B60C8B4151F9C408469581DA1719BEDF874C`|

The `.28` candidate and both closed evidence roots remain frozen. The delivered
host claim remains ordered command-line open/session publication plus normal
close, with all46 recorded checks; it is not second-launch recovery. Root
acceptance has not yet been written.
