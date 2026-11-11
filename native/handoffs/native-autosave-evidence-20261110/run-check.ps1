param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][ValidateSet('format','focused','final','vet')][string]$Step
)
$ErrorActionPreference = 'Stop'
$pwshPath = 'C:\Users\canace\.cache\codex-runtimes\codex-primary-runtime\dependencies\native\powershell\pwsh.exe'
$goArgs = switch ($Step) {
    'format' { @('run','cmd/gofmt','-w','app/autosave.go','app/autosave_test.go','app/app.go','app/state.go','app/view.go','app/workspace.go','app/export_worker.go','app/binary_actions.go','app/outline.go','app/plugins.go','app/source_navigation.go') }
    'focused' { @('test','./app','-run','^TestRecoveryAutosave','-count=1','-v') }
    'final' { @('test','./app','./modules/imageview','-count=1') }
    'vet' { @('vet','./app','./modules/imageview') }
}
$commandPath = Join-Path $PSScriptRoot "$Name.command.ps1"
$outputPath = Join-Path $PSScriptRoot "$Name.output.txt"
$resultPath = Join-Path $PSScriptRoot "$Name.result.json"
foreach ($path in @($commandPath,$outputPath,$resultPath)) { if (Test-Path -LiteralPath $path) { throw "Evidence exists: $path" } }
$quotedArgs = ($goArgs | ForEach-Object { "'" + $_.Replace("'", "''") + "'" }) -join ', '
$commandText = "Set-Location -LiteralPath 'F:\AIproject\LevNote'`r`n& 'F:\AIproject\LevNote\native\scripts\go.ps1' -GoArguments @($quotedArgs)`r`nexit `$LASTEXITCODE`r`n"
[IO.File]::WriteAllText($commandPath,$commandText,[Text.UTF8Encoding]::new($false))
[IO.File]::WriteAllText($outputPath,'',[Text.UTF8Encoding]::new($false))
$started = [DateTimeOffset]::UtcNow
& $pwshPath -NoLogo -NoProfile -File $commandPath 2>&1 | Tee-Object -FilePath $outputPath
$checkExitCode = $LASTEXITCODE
$finished = [DateTimeOffset]::UtcNow
$result = [ordered]@{name=$Name;step=$Step;goArguments=@($goArgs);commandFile=$commandPath;outputFile=$outputPath;startedUtc=$started.ToString('o');finishedUtc=$finished.ToString('o');durationSeconds=[Math]::Round(($finished-$started).TotalSeconds,3);exitCode=$checkExitCode}
[IO.File]::WriteAllText($resultPath,($result|ConvertTo-Json -Depth 4)+"`r`n",[Text.UTF8Encoding]::new($false))
exit $checkExitCode
