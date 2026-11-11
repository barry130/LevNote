# Command correction

`commands.txt` records only the first test invocation. Its
`-run '^TestPackageInventory$'` argument was incorrectly quoted in the
execution helper, so Go reported exit 0 with `[no tests to run]`. That call is
preserved under `failed-attempts/` and is not the passing gate.

The actual successful focused command, executed as the second test invocation
at 2026-11-09T05:16:50.4508555Z, was:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts\go.ps1 test -run ^TestPackageInventory ./core/engineruntime
```

The vet command listed in `commands.txt` is the actual scoped vet command and
remains correct. No commands were rerun to create this correction. The passing
test result is exit 0 with `ok levnote.local/native/core/engineruntime 0.725s`;
the wrapper is the existing `scripts/go.ps1`, which preserves
`$LASTEXITCODE` via its own exit statement.
