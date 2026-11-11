param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][string]$ScriptPath
)

$ErrorActionPreference = 'Stop'
$evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
. (Join-Path $evidenceRoot 'delivery-lib.ps1')
$ctx = Read-DeliveryContext $evidenceRoot
$repositoryRoot = Assert-AbsoluteDirectory $ctx.repositoryRoot 'repository root'
$scriptFull = [IO.Path]::GetFullPath($ScriptPath)
$captureRoot = $evidenceRoot
if ($Name -eq 'aggregate') {
    $captureRoot = Join-Path (Split-Path $evidenceRoot -Parent) (
        (Split-Path $evidenceRoot -Leaf) + '-aggregate-capture')
}
if (-not (Test-Path -LiteralPath $captureRoot -PathType Container)) {
    New-Item -ItemType Directory -Path $captureRoot -ErrorAction Stop | Out-Null
}
$stdout = Join-Path $captureRoot ("stage-{0}.stdout.log" -f $Name)
$stderr = Join-Path $captureRoot ("stage-{0}.stderr.log" -f $Name)
$resultPath = Join-Path $captureRoot ("stage-{0}-invocation.json" -f $Name)
foreach ($capturePath in @($stdout, $stderr, $resultPath)) {
    Assert-AbsentAbsolutePath $capturePath 'inner stage capture' | Out-Null
}
$pwsh = (Get-Command pwsh -ErrorAction Stop).Source
$arguments = @('-NoProfile', '-File', ('"' + $scriptFull + '"'))
$started = [DateTime]::UtcNow
$watch = [Diagnostics.Stopwatch]::StartNew()
$process = Start-Process -FilePath $pwsh -ArgumentList $arguments `
    -WorkingDirectory $repositoryRoot -WindowStyle Hidden `
    -RedirectStandardOutput $stdout -RedirectStandardError $stderr -PassThru -Wait
$watch.Stop()
$record = [ordered]@{
    Schema = 1
    Name = $Name
    Command = $pwsh
    Arguments = $arguments
    ScriptPath = $scriptFull
    WorkingDirectory = $repositoryRoot
    StartedAtUtc = $started.ToString('o')
    FinishedAtUtc = [DateTime]::UtcNow.ToString('o')
    WallSeconds = $watch.Elapsed.TotalSeconds
    ExitCode = $process.ExitCode
    StdoutPath = $stdout
    StderrPath = $stderr
    StdoutBytes = [Int64](Get-Item -LiteralPath $stdout).Length
    StderrBytes = [Int64](Get-Item -LiteralPath $stderr).Length
    WindowStyle = 'Hidden'
}
Write-JsonFile $resultPath $record 6
$exitCode = $process.ExitCode
$process.Dispose()
exit $exitCode
