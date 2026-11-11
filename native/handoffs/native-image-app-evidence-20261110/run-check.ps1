param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][ValidateSet('format','focused','final','imageview-final','vet','captures','capture-fixed')][string]$Step
)
$ErrorActionPreference = 'Stop'
$pwshPath = 'C:\Users\canace\.cache\codex-runtimes\codex-primary-runtime\dependencies\native\powershell\pwsh.exe'
$goArgs = switch ($Step) {
    'format' { @('run','cmd/gofmt','-w','app/app.go','app/binary.go','app/binary_lane.go','app/binary_actions.go','app/binary_test.go','app/state.go','app/view.go','app/plugins.go','app/reference.go','app/workspace.go','app/workers.go','app/export.go','app/export_worker.go','app/search_navigation.go','app/outline.go') }
    'focused' { @('test','./app','-run','^TestNativeImage','-count=1','-v') }
    'final' { @('test','./app','./modules/...','./core/plugins','./core/session','./core/document','./formats/raster','./ui/editor','-count=1') }
    'imageview-final' { @('test','./modules/imageview','-count=1') }
    'vet' { @('vet','./app','./modules/...','./core/plugins','./core/session','./core/document','./formats/raster','./ui/editor') }
    'captures' { @('test','./app','-run','^TestNativeImageCaptures$','-count=1','-v') }
    'capture-fixed' { @('test','./app','-run','^TestNativeImageCaptures$','-count=1','-v') }
}
$commandPath = Join-Path $PSScriptRoot "$Name.command.ps1"
$outputPath = Join-Path $PSScriptRoot "$Name.output.txt"
$resultPath = Join-Path $PSScriptRoot "$Name.result.json"
foreach ($path in @($commandPath,$outputPath,$resultPath)) { if (Test-Path -LiteralPath $path) { throw "Evidence exists: $path" } }
$quotedArgs = ($goArgs | ForEach-Object { "'" + $_.Replace("'", "''") + "'" }) -join ', '
$commandText = "Set-Location -LiteralPath 'F:\AIproject\LevNote'`r`n"
if ($Step -in @('captures','capture-fixed')) { $commandText += "`$env:LEVNOTE_IMAGE_CAPTURE_DIR = 'F:\AIproject\LevNote\native\handoffs\native-image-app-evidence-20261110'`r`n" }
if ($Step -eq 'capture-fixed') { $commandText += "`$env:LEVNOTE_IMAGE_CAPTURE_REVISION = 'toolbar-fixed'`r`n" }
$commandText += "& 'F:\AIproject\LevNote\native\scripts\go.ps1' -GoArguments @($quotedArgs)`r`nexit `$LASTEXITCODE`r`n"
[IO.File]::WriteAllText($commandPath,$commandText,[Text.UTF8Encoding]::new($false))
[IO.File]::WriteAllText($outputPath,'',[Text.UTF8Encoding]::new($false))
$started = [DateTimeOffset]::UtcNow
& $pwshPath -NoLogo -NoProfile -File $commandPath 2>&1 | Tee-Object -FilePath $outputPath
$checkExitCode = $LASTEXITCODE
$finished = [DateTimeOffset]::UtcNow
$result = [ordered]@{name=$Name;step=$Step;goArguments=@($goArgs);commandFile=$commandPath;outputFile=$outputPath;startedUtc=$started.ToString('o');finishedUtc=$finished.ToString('o');durationSeconds=[Math]::Round(($finished-$started).TotalSeconds,3);exitCode=$checkExitCode}
[IO.File]::WriteAllText($resultPath,($result|ConvertTo-Json -Depth 4)+"`r`n",[Text.UTF8Encoding]::new($false))
exit $checkExitCode
