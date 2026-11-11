# Whole-word review corrections: second pass

2026-11-10. Second narrow reconciliation after Root identified that the prior
inside-word fixture selected `[1,4)` (`ate`) instead of `cat` at `[0,3)`.
Production, UI, captures, the prior implementation result and the first
correction handoff/log are preserved unchanged. No build, launch, full suite,
vet or capture rerun was performed.

## Corrected replacement proof

`TestFindWholeWordReplacementAndGroupedUndo` now has paired independent
`category` fixtures. Both first select `[0,3)`, read the selected text with
`Buffer.SliceUTF16(0,3)`, and require it to equal the query `cat`.

- `WholeWord=false`: Replace Current with `dog` changes `category` to
  `dogegory`. This proves the selected candidate is a real matching occurrence
  and that it is replaceable when the boundary filter is disabled.
- `WholeWord=true`: a fresh fixture with the same exact selection invokes
  Replace Current and must preserve the text, buffer version and selection
  exactly. This is the non-vacuous refusal proof.

The existing standalone Replace Current success and Replace All one-undo-group
tests are unchanged.

## Focused verification

One focused test was run after the correction:

```powershell
& .\native\scripts\go.ps1 -GoArguments @('test','-count=1','-run','^TestFindWholeWordReplacementAndGroupedUndo$','./app')
```

```text
ok      levnote.local/native/app    1.740s
EXIT_CODE=0
```

Raw log:
`native/handoffs/find-whole-word-evidence/review-corrections-2-focused.log`

SHA-256 `5F50F57B160F074F46D04688300C2AC68AE903758F14A1E40CC96021E0F51065`.

## Preserved prior correction evidence

- `native/handoffs/find-whole-word-review-corrections.md`

  SHA-256 `7BDBF484D5EB6C8061B6D6317E6F0103DF0BBC9419918C9A7CEEAFAFADD7558F`

- `native/handoffs/find-whole-word-evidence/review-corrections-focused.log`

  SHA-256 `C61637F38CB83B574D59EC7F5B6833C7EBDC9F27A91B264454F315C6416C0D77`

## Current hashes

| Artifact | SHA-256 |
|---|---|
| `native/app/app.go` | `58B6A783968C8860BBE6E8EF87E74CE9C6BEA3560BD4C19A569FC44465557931` |
| `native/app/state.go` | `023C76A143CB572D0A2B45D783A20E105996FF570615357BB1C3F31032E79984` |
| `native/app/find_whole_word_test.go` | `BEBFEBC560AA78D8A47B1390A2CA4E29E956C8328CB4AFEE0B080BF81328E744` |
| `native/app/search_navigation_test.go` | `FA8EF3C1DAF50B67AA03F7ADE032F465EB0535202FAB4328EB7C28DFC4E72A40` |
| `native/app/findbar_layout_test.go` | `662C167BFFA6CC344EF2E2A73E3566824E03B0DA3BA3D19FA463279FFCF12180` |

The production and UI hashes still match the reviewed implementation. Only
the replacement test changed in this pass. The corrected test and evidence are
frozen and ready for independent reconciliation; this handoff does not
self-accept the feature.
