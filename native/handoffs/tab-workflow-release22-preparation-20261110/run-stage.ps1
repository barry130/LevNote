param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('baseline', 'package', 'host-smoke', 'aggregate')]
    [string]$Name
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'delivery-lib.ps1')
$evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
$ctx = Read-DeliveryContext $evidenceRoot
$repositoryRoot = Assert-AbsoluteDirectory $ctx.repositoryRoot 'repository root'
$scriptMap = @{
    baseline = 'baseline.ps1'
    package = 'run-package.ps1'
    'host-smoke' = 'run-host-smoke.ps1'
    aggregate = 'aggregate.ps1'
}
$invokeStage = Join-Path $evidenceRoot 'invoke-stage.ps1'
$stageScript = Join-Path $evidenceRoot $scriptMap[$Name]
$pwsh = (Get-Command pwsh -ErrorAction Stop).Source
$captureRoot = $evidenceRoot
if ($Name -eq 'aggregate') {
    $captureRoot = Join-Path (Split-Path $evidenceRoot -Parent) (
        (Split-Path $evidenceRoot -Leaf) + '-aggregate-capture')
}
if (-not (Test-Path -LiteralPath $captureRoot -PathType Container)) {
    New-Item -ItemType Directory -Path $captureRoot -ErrorAction Stop | Out-Null
}
$stdout = Join-Path $captureRoot ('outer-' + $Name + '.stdout.log')
$stderr = Join-Path $captureRoot ('outer-' + $Name + '.stderr.log')
$recordPath = Join-Path $captureRoot ('outer-' + $Name + '-invocation.json')
foreach ($capturePath in @($stdout, $stderr, $recordPath)) {
    Assert-AbsentAbsolutePath $capturePath 'outer stage capture' | Out-Null
}
$arguments = @(
    '-NoProfile',
    '-File',
    ('"' + $invokeStage + '"'),
    '-Name',
    $Name,
    '-ScriptPath',
    ('"' + $stageScript + '"')
)
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
Write-JsonFile $recordPath $record 6
$exitCode = $process.ExitCode
$process.Dispose()
exit $exitCode
