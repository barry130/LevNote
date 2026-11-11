# Whole-word review corrections

2026-11-10. Narrow reconciliation after Root review. Production wiring, UI,
captures, the prior implementation result and accepted `.16` content are
unchanged. No build, launch, full suite, vet or screenshot rerun was performed.

## Corrections

1. `TestFindCacheReusesColdOffsetsAndTracksSourceVersion` now uses
   `needle needless needle`, so the fixture contains both whole-word and
   inside-word candidates.

   Before `WholeWord`, the cached regex offsets are exactly:

   ```text
   [0,6) [7,13) [16,22)
   ```

   After enabling `WholeWord`, one new cold scan stores exactly:

   ```text
   [0,6) [16,22)
   ```

   A following Find action reuses that filtered warm cache without another
   cold scan. Existing version/edit/undo/restore/new-buffer checks remain.

2. `TestFindWholeWordReplacementAndGroupedUndo` retains the successful
   standalone Replace Current case and the Replace All one-undo-group proof.
   It now also selects `cat` inside `category`, invokes Replace Current with
   `WholeWord`, and requires the buffer text, version and selection to remain
   exactly unchanged.

3. Evidence wording is corrected as follows:

   - The navigation test uses the actual visible `全词` checkbox and
     `ui.Tester` Next/Previous actions to prove toggling, navigation, identifier
     rejection and wrap.
   - Layout tests use the actual control for bounds, tab order, Escape and one
     toggle assertion.
   - The Unicode/regex/case table assigns the runtime flags directly and calls
     the app Find method. It proves app API wiring and search semantics; it does
     not prove those combinations through synthetic control events.
   - The workspace test assigns `a.WholeWord` directly and runs the real
     `a.searchWorkspace()` entry point. It proves shared-option forwarding into
     workspace search, not checkbox-to-workspace UI interaction.
   - Replacement and lifecycle tests set runtime state directly and call
     `a.replace`; they do not claim pointer/button event coverage.

The prior implementation result remains preserved at
`native/handoffs/find-whole-word-implementation-result.md` with SHA-256
`8B92E45A89AE7285F4AB9EB76044A0F2CC5FBEFBB1DB7183AEF18341C6366EA2`.
This correction handoff supersedes its overbroad combined UI wording.

## Focused verification

Only the two corrected tests were run:

```powershell
& .\native\scripts\go.ps1 -GoArguments @('test','-count=1','-run','^(TestFindCacheReusesColdOffsetsAndTracksSourceVersion|TestFindWholeWordReplacementAndGroupedUndo)$','./app')
```

```text
ok      levnote.local/native/app    1.581s
EXIT_CODE=0
```

Raw log:
`native/handoffs/find-whole-word-evidence/review-corrections-focused.log`

SHA-256 `C61637F38CB83B574D59EC7F5B6833C7EBDC9F27A91B264454F315C6416C0D77`.

## Final hashes

| Artifact | SHA-256 |
|---|---|
| `native/app/app.go` | `58B6A783968C8860BBE6E8EF87E74CE9C6BEA3560BD4C19A569FC44465557931` |
| `native/app/state.go` | `023C76A143CB572D0A2B45D783A20E105996FF570615357BB1C3F31032E79984` |
| `native/app/find_whole_word_test.go` | `3362BA0646ED304D45D5F76DCD13655468FCB6C1948FAEFE9DE52F2CBCB06674` |
| `native/app/search_navigation_test.go` | `FA8EF3C1DAF50B67AA03F7ADE032F465EB0535202FAB4328EB7C28DFC4E72A40` |
| `native/app/findbar_layout_test.go` | `662C167BFFA6CC344EF2E2A73E3566824E03B0DA3BA3D19FA463279FFCF12180` |
| light capture | `68D6F293F5E3F95CD9AFA4032E57916AF8C33494C3875E1FCE665D8FFECE1FAC` |
| dark capture | `C72BF002E237A3D8E43F843D88236EA4F634326189A25232978EF8FAB502CD3E` |

The two production hashes and the layout/capture hashes match the prior result,
confirming no production or UI change during this correction. The corrected
tests and evidence are frozen and ready for independent reconciliation. This
handoff does not self-accept the feature.
