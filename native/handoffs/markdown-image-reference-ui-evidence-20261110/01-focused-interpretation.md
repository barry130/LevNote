# First focused run: FAIL, outer exit capture defect preserved

01-focused.stdout.log is authoritative: markdownview PASS2.173s; app FAIL2.616s
at the PNG existing-image-selection assertion. JPEG and refusal/fallback pass.
01-focused-result.json ExitCode0 is only the outer PowerShell process exit:
the command script omitted `exit $LASTEXITCODE` after invoking go.ps1. It is
not a successful Go exit and must not be counted as PASS. No exact numeric Go
exit was captured by that script. Original command/result/streams are retained.

Future command scripts explicitly exit with LASTEXITCODE immediately after
go.ps1. The next run only enriches the one PNG assertion with actual tab/current
identity/path diagnostics; no fixture timing or runtime is changed yet.
