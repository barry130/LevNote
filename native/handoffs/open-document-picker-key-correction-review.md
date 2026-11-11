# Open-document picker raw-key correction review

2026-11-10. Narrow read-only review of the frozen query-key correction. This
reviewer ran no test, build, launch or benchmark and edited only this handoff.
The prior blocker report,
`open-document-picker-final-review.md`, remains unchanged at SHA-256
`31512A2A1C1562671F9B69322E236D8B80D2E65AE61617B8871484E7805567F6`.

## Verdict

No concrete blocker remains in the correction. The source now prevents every
non-text key delivered through the query's public `OnTextKey` route from
falling into MyGo's `editKey` path while composition is active. Native
Compose/Text observation remains the explicit commit/cancel route, and the
ordinary non-composing navigation/editing branch is unchanged.

Independent review ownership is released for the final affected app test and
vet.

## Exact frozen identities

The actual five files match the corrected manifest:

| Native-relative source | Bytes | SHA-256 |
|---|---:|---|
| `app/open_documents.go` | 13882 | `BF1789ACC0A9B885F358CD46F62251C5E2971A6DD351AC5209A3C82BC15B7DA6` |
| `app/open_documents_test.go` | 30122 | `A70444047DED384477F2DFDE2E83FE98AC53F06304633F4F65304A19CCC5444C` |
| `app/app.go` | 39003 | `FAD0EDE4C480E0C8F5C56F4E81CF2499BEF0007BA6EAE877FC516623BFFEA050` |
| `app/view.go` | 23425 | `64DCDC194510C6426112A7B6AAFE57A84DDE68E630A20D53C994D77CC34102C0` |
| `app/workspace.go` | 6739 | `7A9ACA4F8E807D7A0563B5595976839AA7DED706EFDC5171B259079D68A16DA4` |

`source-frozen.json` is 748 bytes with SHA-256
`AB90451D3003E713E6A5260087FA36B0E32DC2B3EDA23DD517F8B0502E9B59F6`.
The three files outside the assigned correction remain byte-identical to the
initial freeze. The 01 pre/post manifests are both that same corrected
manifest, and `source-comparison.json` reports all five unchanged across the
focused run.

## Source assessment

At `app/open_documents.go:301-310`, an active picker composition now returns
`true` for every non-text key delivered to `OnTextKey`. The public contract at
`vendor/github.com/egoist/mygo/ui/textinput.go:143-160` limits that callback to
keys that type no text; a key taken by the callback does not continue to the
editor, focus traversal, shortcut or menu path. This closes the prior
Left/Right route and the same route for every other raw non-text key without
vendor changes.

Compose/Text still reaches the dialog observer at
`app/open_documents.go:358-367`: non-empty Compose marks composition active,
empty Compose clears it, and Text clears it as a commit. Those updates occur
before the queued native editor operation, so the callback policy does not
block the intended native commit/cancel event and does not leave a stale app
flag. Once composition is inactive, the unchanged branch handles Escape,
Up/Down/Enter picker actions and leaves ordinary text-input keys to MyGo.

The regression at `app/open_documents_test.go:87-177` runs both commit and
cancel cases through 32 public key/modifier combinations. It includes
Left/Right, Shift and Ctrl/Shift selection or word movement, Home/End and their
Shift forms, Up/Down, Page Up/Down, Backspace/Delete and Ctrl deletion,
select-all/copy/cut/paste/undo/redo/Shift redo, Enter/Shift+Enter,
Escape, Tab and Shift+Tab. After each key it checks the native `Composing()` and
IME caret, query focus/value/filter/selection, pending actions, picker state,
active tab, both document snapshots/version/history/selections and clipboard.
It then verifies Text commit and Enter activation, or empty Compose and Escape
restoration. This directly covers the former blocker and the broader raw-key
risk without private toolkit state.

The new ordinary-input test at `app/open_documents_test.go:179-231` uses the
same real query to insert via Left/Right caret movement, replace a Shift
selection, Home/Delete and End/Backspace, then exercise select-all,
copy/cut/paste and undo/redo before Enter activates the filtered destination.
It also checks both document buffers remain unchanged. This establishes that
the composition guard does not disable normal editing after preedit ends.

## Evidence

The saved command is the required wrapper from `native`:

```text
scripts/go.ps1 -GoArguments @('test','./app','-run','^TestOpenDocuments','-count=1','-v')
```

The run started at `2026-11-10T12:29:07.8251921Z`; Go finished at
`12:29:13.5343644Z` with exit `0`, and the owned runner finished at
`12:29:13.7385452Z` with shell exit `0`. The app package reports `2.619s`;
stderr is zero bytes. All 13 behavior top-level tests passed. The optional
capture skipped because the runner explicitly removed its opt-in environment
variable.

The correction evidence inventory closes 12 other files totaling 11012 bytes
with zero identity mismatches; the inventory itself has SHA-256
`0BD66E90BCC216B651A44AD5D44F3E931642ABAF3A16245D9168AB4EA2F77F2C`.
The original closed picker evidence was independently rechecked at 47 files /
123554 bytes with zero mismatches and inventory hash
`338A86C313FFEB268C71E57F9DFF5C92D8A3F593E8F7E39FBA5926B9F421764E`.
Its earlier captures, failed attempts and original manifests remain in place.

Earlier accepted picker source reasoning and the 08 layout evidence remain
accepted by reference. No capture was repeated because this correction changes
only key routing and tests.

## Limits

This is synthetic public Tester keyboard/Compose/Text evidence, not physical
Windows IME delivery. The raw-key callback policy is separate from external
native menu commands, which it does not intercept or claim. This review also
does not establish the full app suite, vet, build, package, host, worker,
resource or whole-process memory behavior. Those final checks remain for the
next assigned stage.
