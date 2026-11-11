$ErrorActionPreference = 'Stop'

$commandText = "native/scripts/go.ps1 -GoArguments @('test','./modules/imageview','-run','TestToolbarControlsRemainReachableAtNarrowWidths','-count=1')"
Set-Content -LiteralPath (Join-Path $PSScriptRoot '03-focused-command.txt') -Value $commandText -Encoding utf8

$child = Start-Process `
    -FilePath (Join-Path $PSHOME 'pwsh.exe') `
    -ArgumentList @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', (Join-Path $PSScriptRoot 'run-focused-child.ps1')) `
    -RedirectStandardOutput (Join-Path $PSScriptRoot '03-focused-stdout.txt') `
    -RedirectStandardError (Join-Path $PSScriptRoot '03-focused-stderr.txt') `
    -WindowStyle Hidden `
    -Wait `
    -PassThru

Set-Content -LiteralPath (Join-Path $PSScriptRoot '03-focused-exit.txt') -Value $child.ExitCode -Encoding ascii
exit $child.ExitCode
