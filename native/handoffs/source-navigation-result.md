# Native source navigation: implementation result

2026-11-09. Implementation and scoped verification are complete under
`source-navigation-implementation-task.md`. Production source is frozen pending
root review. No package build, release, product-doc update, helper retry,
resource matrix or physical UI/IME test was run. `.11` is unaffected.

## Delivered behavior

- `Ctrl+G` and the current line/column status button open one compact source
  line panel. The numeric field is focused once with `AutoFocus`, is limited to
  32 UTF-16 code units, and keeps the raw query on local errors.
- Input syntax is exactly positive ASCII `line` or `line:column`; surrounding
  whitespace only is removed. Omitted `:column` means one-based column 1.
  Leading zeroes, signs, zero, empty fields, extra separators and overflow are
  rejected. There is no clamping.
- Submission parses first, verifies the captured tab/doc/editor/buffer, calls
  `Flush` once, verifies identity again, and validates the final flush buffer
  via bounded `LineRange` plus an empty `SliceUTF16` check. No whole-document
  conversion is used. Inside-surrogate columns are rejected before
  `SetPosition`.
- Coordinates are one-based UTF-16 code-unit positions. CR/LF are not columns,
  empty final lines are valid, and columns between combining code points remain
  addressable. This is not a grapheme-cluster guarantee.
- A flush refusal retains the panel, raw query, target references, mode,
  selection and history. A successful flush followed by an invalid target
  reports that text was synchronized. Source and split stay in their current
  mode; preview switches to source only after final validation. Every
  successful source/split/preview navigation requests position persistence.
- Readonly source navigation and the displayed read-only hexadecimal byte-view
  lines work. The help explicitly says when the target is the byte view.
- Opening is refused while the source editor or either neutral rich view reports
  active composition. The optional `editor.Composer` contract is implemented by
  Markdown and graph views without widget-tree inspection.
- The global Escape branch is suppressed while the panel is visible. Input
  Escape and Enter are handled through `OnTextKey`; while a synthetic IME
  composition is active they are left to the IME. Escape on either focused
  panel button is handled by that button's scoped shortcut. Pointer submit keeps
  focus in the field and is blocked during composition.
- A panel generation guards every field/button/observer callback. Closing or
  reopening cannot let an old callback act on the new target. Tab selection,
  new/recovered tabs, committed removal, reload, encoding rematch and plugin
  reconfiguration clear the panel and all captured pointers; failed tab switch
  or close retains them with the original tab. The input element key also
  includes the generation, so MyGo cannot retain a closed field's internal
  TextInput state across close/reopen. Plugin panel opening also closes source
  navigation.

## First focused run checkpoint

Exact command:

```text
.\native\scripts\go.ps1 -GoArguments @("test","./app","-run","SourceNavigation|ParseSourceLine")
```

Exit code: `1`.

The first feature test
`TestSourceNavigationActualShortcutFocusEnterAndStatusAction` opened Ctrl+G and
then panicked while building `app/source_navigation.go`:

```text
panic: ui: Key on a TextInput, whose state was initialized: use Context.Key before construction
```

The panic points to the panel field construction at
`native/app/source_navigation.go:205`: `ui.TextInput(c, ...)` is constructed
before `.Key("gotoline-input")`. The focused run stopped at this first real
lifecycle error, so later tests did not execute.

Corrections applied after this checkpoint:

- initialize the field with `Context.Key` before TextInput construction; the
  final key is generation-scoped, not the initial constant key;
- correct the inverted invalid-syntax assertion;
- verify the editor still owns the captured buffer;
- keep the raw query after an empty/invalid submission;
- mark position persistence pending after every successful navigation;
- state the omitted-column and 32-UTF-16-unit limits in the visible help;
- add routed Escape coverage with a synthetic composition;
- add the post-Flush changed-coordinate and buffer-identity regressions.

## Corrected focused checkpoint

The construction order was changed to
`ui.TextInput(c.Key("gotoline-input"), ...)`, and the corrections above were
implemented within the authorized files. The corrected focused run first
reached the synthetic composition test and found one test expectation error:
`123` correctly resolves to the zero-based `LineColumn` pair `(122, 0)`, not
`(122, 1)`. Pointer submission remained blocked while composition was active,
and the post-commit query resolved the complete line.

After correcting that assertion, the then-current focused command exited `0`
with:

```text
ok  	levnote.local/native/app	2.032s
```

Root's final source review then added four concrete requirements: generation-
scoped field identity, explicit leading-zero help, pointer cancel during query
composition, and a real Markdown draft `Flush`. The refreshed focused gate
exposed one test assumption: normal rich-draft text is synchronized through its
`OnChange` callback immediately, so it is not still pending when the focused
menu opens:

```text
--- FAIL: TestSourceNavigationFlushesRealMarkdownDraft
    source_navigation_test.go:332: committed rich draft reached the host buffer before source navigation flushed it
FAIL	levnote.local/native/app	2.082s
```

The regression now wraps the actual Markdown view and proves source navigation
invokes its real `Flush` exactly once. Active composition remains blocked at
open, and the deterministic fake Flusher separately proves coordinates are
resolved against a successful Flush that changes the source.

The corrected final focused gate exited `0`:

```text
ok  	levnote.local/native/app	2.294s
```

## Verification

Final focused feature command:

```text
.\native\scripts\go.ps1 -GoArguments @("test","./app","-run","SourceNavigation|ParseSourceLine","-count=1")
ok  	levnote.local/native/app	2.294s
```

Final capture command:

```text
$env:LEVNOTE_SOURCE_NAVIGATION_SCREENSHOT_DIR='F:\AIproject\LevNote\native\handoffs\source-navigation-evidence'
.\native\scripts\go.ps1 -GoArguments @("test","./app","-run","^TestSourceNavigationNarrowErrorCapture$","-count=1")
ok  	levnote.local/native/app	1.546s
```

The first two capture-only attempts exited `1` because the test asserted the
localized error body through `Tester.Texts`; that toolkit accessor reports the
error Text's explicit accessible label, not its visual body. The final test
retains an explicit assertion on `GoToLineError` and checks the accessible
label, leading-zero help, and geometry separately.

The single final affected suite exited `0`:

```text
.\native\scripts\go.ps1 -GoArguments @("test","./app","./ui/editor","./modules/markdownview","./modules/graphview","-count=1")
ok  	levnote.local/native/app	23.694s
ok  	levnote.local/native/ui/editor	5.533s
ok  	levnote.local/native/modules/markdownview	7.713s
ok  	levnote.local/native/modules/graphview	19.048s
```

Scoped vet exited `0` with no output:

```text
.\native\scripts\go.ps1 -GoArguments @("vet","./app","./ui/editor","./modules/markdownview","./modules/graphview")
```

Exact current-source order was: final focused pass, refreshed capture pass,
single combined affected suite, then scoped vet. No source, test or evidence
file changed after that sequence. Only this result document was updated.

## Tester captures

Both images were visually inspected. The input, translated `0`, local
`行号和列号必须是正整数` error, compact `转到` / `取消` buttons, and full
accessible labels remain inside the main pane and above the fixed status row.
The complete help text includes one-based coordinates, UTF-16 units, omitted
column, rejection of leading zero, and the 32-unit limit; it wraps to two
readable lines. Coverage includes left/right horizontal bounds, top/bottom
vertical bounds, and the 29-DIP status row at 760x480.

| evidence | geometry | bytes | SHA256 |
|---|---:|---:|---|
| `source-navigation-evidence/source-navigation-error-760x480-light.png` | 760x480 | 35,671 | `2932E7800A6C9062636DDDA1180F4A876ACBCDA1B7769F7F413427B614F0229A` |
| `source-navigation-evidence/source-navigation-error-760x480-dark.png` | 760x480 | 35,805 | `17E75F906CF7B4C54F6D1708E46EF9EBEE714B576B2FC84799F542D361960D7B` |

These are MyGo tester captures. They do not establish physical OS IME behavior,
native candidate-window behavior, font shaping outside the tester, or process
resource usage. Composition cancel/submit and the Markdown composition barrier
are synthetic tester evidence. Normal Markdown draft text synchronizes on
change; the real-view regression proves the actual `Flush` call is made exactly
once, while a deterministic Flusher covers successful mutation followed by
final-coordinate validation.

## Changed files and frozen hashes

| file | bytes | SHA256 |
|---|---:|---|
| `native/app/app.go` | 32,580 | `E0CD0C67A6FE6C511924BD24B5E1F1853F30BED4C1217D1B64906E12BFAFDDE9` |
| `native/app/plugins.go` | 10,731 | `F880725B90C4C7602A9A1E0003861805DB723C391CF513F2FFBC2E662E968AD9` |
| `native/app/view.go` | 20,541 | `5961F85561A58FACFFB1DFDE1CBD9AC6BED8B231F48523D0253B9EE95FE01F9C` |
| `native/app/source_navigation.go` | 8,042 | `E7E32D5230F94AD4F12C3D9CADD2A1B27247F8D5B2AEF1B21A6CBA2A4AA79699` |
| `native/app/source_navigation_test.go` | 27,647 | `D9C1EFD573E413359CE500C6E1C806D9C0EF09AD900719A70D7C9786230C4998` |
| `native/ui/editor/contract.go` | 3,954 | `42BCA016681249C8EBCB52C4E6C9B3A416624663313B24FAAA5098748EC7593C` |
| `native/modules/markdownview/view.go` | 17,982 | `B2B7D170206C84A0DFDACB9C3B33B45985439E95FC414D4FFAB0D4C7431E1BC4` |
| `native/modules/graphview/adapter.go` | 8,746 | `E5DE14F50D4AB45EE4D02E2E93223E0DCC4BD0D4E61148BB5DDF69533BB260C6` |

No core/text, module format semantics, build script, dependency, release
package, product documentation or `.11` artifact was changed.
