param(
  [string]$Label,
  [string]$ArgsJson
)
$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path
$native = Join-Path $repo 'native'
$wrapper = Join-Path $native 'scripts\go.ps1'
$outDir = Join-Path $native 'handoffs\drawio-source-retention-evidence'
$sourceFiles = @('formats/drawio/parse.go','formats/drawio/edit.go','formats/drawio/types.go','formats/drawio/source_retention_benchmark_test.go')
$argsList = @(ConvertFrom-Json -InputObject $ArgsJson)
$hashesBefore = [ordered]@{}
foreach ($relative in $sourceFiles) {
  $full = Join-Path $native ($relative -replace '/', '\')
  if (Test-Path -LiteralPath $full) { $hashesBefore[$relative] = (Get-FileHash -Algorithm SHA256 -LiteralPath $full).Hash.ToUpperInvariant() }
}
$psi = [System.Diagnostics.ProcessStartInfo]::new()
$psi.FileName = (Get-Command pwsh).Source
$psi.WorkingDirectory = $repo
$psi.UseShellExecute = $false
$psi.RedirectStandardOutput = $true
$psi.RedirectStandardError = $true
$psi.Environment['LEVNOTE_GO_WRAPPER'] = $wrapper
$psi.Environment['LEVNOTE_GO_ARGS_JSON'] = $ArgsJson
$psi.ArgumentList.Add('-NoProfile')
$psi.ArgumentList.Add('-Command')
$psi.ArgumentList.Add('& $env:LEVNOTE_GO_WRAPPER -GoArguments @($env:LEVNOTE_GO_ARGS_JSON | ConvertFrom-Json)')
$started = [DateTimeOffset]::UtcNow
$p = [System.Diagnostics.Process]::new()
$p.StartInfo = $psi
[void]$p.Start()
$outTask = $p.StandardOutput.ReadToEndAsync()
$errTask = $p.StandardError.ReadToEndAsync()
$p.WaitForExit()
$out = $outTask.Result
$err = $errTask.Result
$finished = [DateTimeOffset]::UtcNow
$hashesAfter = [ordered]@{}
foreach ($relative in $sourceFiles) {
  $full = Join-Path $native ($relative -replace '/', '\')
  if (Test-Path -LiteralPath $full) { $hashesAfter[$relative] = (Get-FileHash -Algorithm SHA256 -LiteralPath $full).Hash.ToUpperInvariant() }
}
$outPath = Join-Path $outDir ($Label + '.stdout.log')
$errPath = Join-Path $outDir ($Label + '.stderr.log')
$jsonPath = Join-Path $outDir ($Label + '.invocation.json')
$utf8 = [System.Text.UTF8Encoding]::new($false)
[System.IO.File]::WriteAllText($outPath, $out, $utf8)
[System.IO.File]::WriteAllText($errPath, $err, $utf8)
$record = [ordered]@{
  label = $Label
  command = @('native/scripts/go.ps1','-GoArguments') + $argsList
  workingDirectory = $repo
  startedAtUtc = $started.ToString('o')
  finishedAtUtc = $finished.ToString('o')
  exitCode = $p.ExitCode
  sourceSha256Before = $hashesBefore
  sourceSha256After = $hashesAfter
  stdout = $outPath
  stderr = $errPath
}
[System.IO.File]::WriteAllText($jsonPath, ($record | ConvertTo-Json -Depth 5), $utf8)
$out
if ($err) { [Console]::Error.Write($err) }
exit $p.ExitCode