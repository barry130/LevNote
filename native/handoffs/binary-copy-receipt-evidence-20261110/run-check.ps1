param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('format', 'focused', 'final-test', 'final-vet')]
    [string]$Step
)
$ErrorActionPreference = 'Stop'
$repoRoot = 'F:\AIproject\LevNote'
$pwshPath = 'C:\Users\canace\.cache\codex-runtimes\codex-primary-runtime\dependencies\native\powershell\pwsh.exe'
$evidenceRoot = $PSScriptRoot
$goArgs = switch ($Step) {
    'format' { @('run', 'cmd/gofmt', '-w', 'core/document/binary.go', 'core/document/binary_copy.go', 'core/document/binary_test.go') }
    'focused' { @('test', './core/document', '-run', '^TestBinaryCopy', '-count=1', '-v') }
    'final-test' { @('test', './core/document', '-count=1') }
    'final-vet' { @('vet', './core/document') }
}
$outputPath = Join-Path $evidenceRoot "$Step.output.txt"
$commandPath = Join-Path $evidenceRoot "$Step.command.ps1"
$resultPath = Join-Path $evidenceRoot "$Step.result.json"
foreach ($path in @($outputPath, $commandPath, $resultPath)) {
    if (Test-Path -LiteralPath $path) { throw "Preserve prior evidence; step already exists: $path" }
}
$quotedArgs = ($goArgs | ForEach-Object { "'" + $_.Replace("'", "''") + "'" }) -join ', '
$commandText = "Set-Location -LiteralPath 'F:\AIproject\LevNote'`r`n& 'F:\AIproject\LevNote\native\scripts\go.ps1' -GoArguments @($quotedArgs)`r`n"
[IO.File]::WriteAllText($commandPath, $commandText, [Text.UTF8Encoding]::new($false))
$started = [DateTimeOffset]::UtcNow
Set-Location -LiteralPath $repoRoot
& $pwshPath -NoLogo -NoProfile -File $commandPath 2>&1 | Tee-Object -FilePath $outputPath
$checkExitCode = $LASTEXITCODE
$finished = [DateTimeOffset]::UtcNow
$result = [ordered]@{
    step = $Step
    commandFile = $commandPath
    outputFile = $outputPath
    goArguments = @($goArgs)
    startedUtc = $started.ToString('o')
    finishedUtc = $finished.ToString('o')
    durationSeconds = [Math]::Round(($finished - $started).TotalSeconds, 3)
    exitCode = $checkExitCode
}
[IO.File]::WriteAllText($resultPath, ($result | ConvertTo-Json -Depth 4) + "`r`n", [Text.UTF8Encoding]::new($false))
exit $checkExitCode
