param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('baseline', 'package', 'managed-smoke', 'host-smoke', 'aggregate')]
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
    'managed-smoke' = 'run-managed-smoke.ps1'
    'host-smoke' = 'run-host-smoke.ps1'
    aggregate = 'aggregate.ps1'
}
$invokeStage = Join-Path $evidenceRoot 'invoke-stage.ps1'
$stageScript = Join-Path $evidenceRoot $scriptMap[$Name]
$pwsh = (Get-Command pwsh -ErrorAction Stop).Source
$stdout = Join-Path $evidenceRoot ('outer-' + $Name + '.stdout.log')
$stderr = Join-Path $evidenceRoot ('outer-' + $Name + '.stderr.log')
$recordPath = Join-Path $evidenceRoot ('outer-' + $Name + '-invocation.json')
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
