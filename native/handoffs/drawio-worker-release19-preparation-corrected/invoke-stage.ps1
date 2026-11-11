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
$stdout = Join-Path $evidenceRoot ("stage-{0}.stdout.log" -f $Name)
$stderr = Join-Path $evidenceRoot ("stage-{0}.stderr.log" -f $Name)
$resultPath = Join-Path $evidenceRoot ("stage-{0}-invocation.json" -f $Name)
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
