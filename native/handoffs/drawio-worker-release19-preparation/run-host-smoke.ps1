$ErrorActionPreference = 'Stop'
$evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
. (Join-Path $evidenceRoot 'delivery-lib.ps1')
$ctx = Read-DeliveryContext $evidenceRoot
Assert-DeliveryExecution $ctx

if (-not (Test-Path -LiteralPath (Join-Path $evidenceRoot 'managed-smoke/result.json') -PathType Leaf)) {
    throw 'Run the managed smoke stage first.'
}
$smokeRoot = Join-Path $evidenceRoot 'host-smoke'
if (Test-Path -LiteralPath $smokeRoot) {
    throw "Fresh hidden host smoke directory already exists: $smokeRoot"
}
$bin = Join-Path $smokeRoot 'bundle'
$workspace = Join-Path $smokeRoot 'workspace'
$normalTemp = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
if (-not [string]::Equals([IO.Path]::GetPathRoot($normalTemp), 'C:\', [StringComparison]::OrdinalIgnoreCase)) {
    throw "Hidden host smoke requires the normal default TEMP path on C:, resolved to $normalTemp."
}
$configRoot = Join-Path $normalTemp ('levnote-native-release19-host-smoke-' + [Guid]::NewGuid().ToString('N'))
$roaming = Join-Path $configRoot 'appdata/Roaming'
$local = Join-Path $configRoot 'appdata/Local'
foreach ($path in @($smokeRoot, $bin, $workspace, $configRoot, $roaming, $local)) {
    if (Test-Path -LiteralPath $path) {
        throw "Hidden host smoke path must be fresh: $path"
    }
    New-Item -ItemType Directory -Path $path -ErrorAction Stop | Out-Null
}

$candidateBefore = @(Get-DirectoryInventory $ctx.candidateDirectory)
if ($candidateBefore.Count -ne [int]$ctx.ExpectedCandidateFiles) {
    throw "Hidden host smoke requires the complete candidate, found $($candidateBefore.Count) files."
}
foreach ($source in Get-ChildItem -LiteralPath $ctx.candidateDirectory -File -Recurse) {
    $relative = [IO.Path]::GetRelativePath($ctx.candidateDirectory, $source.FullName)
    $destination = Join-Path $bin $relative
    $parent = Split-Path $destination -Parent
    if (-not (Test-Path -LiteralPath $parent -PathType Container)) {
        New-Item -ItemType Directory -Path $parent -ErrorAction Stop | Out-Null
    }
    Copy-Item -LiteralPath $source.FullName -Destination $destination -ErrorAction Stop
}
$copiedBefore = @(Get-DirectoryInventory $bin)
Assert-InventoryContainsExact $candidateBefore $copiedBefore 'Hidden host initial bundle copy' $false

$sample = Join-Path $workspace 'package-smoke.md'
$sourceText = "# Package smoke`r`n`r`nOne ordinary unchanged Markdown source.`r`n"
[IO.File]::WriteAllText($sample, $sourceText, [Text.UTF8Encoding]::new($false))
$sourceBefore = Get-FileFingerprint $sample

Add-Type -TypeDefinition @'
using System;
using System.Text;
using System.Runtime.InteropServices;
public struct Release19SmokeWindow { public IntPtr Handle; public string Title; public bool Visible; }
public static class Release19SmokeWin32 {
    private delegate bool Callback(IntPtr hwnd, IntPtr data);
    [DllImport("user32.dll")] private static extern bool EnumWindows(Callback callback, IntPtr data);
    [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr hwnd, out uint pid);
    [DllImport("user32.dll", CharSet=CharSet.Unicode)] private static extern int GetWindowText(IntPtr hwnd, StringBuilder text, int count);
    [DllImport("user32.dll")] private static extern bool IsWindowVisible(IntPtr hwnd);
    [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr hwnd, uint message, IntPtr w, IntPtr l);
    [DllImport("kernel32.dll", CharSet=CharSet.Unicode, SetLastError=true)] private static extern bool QueryFullProcessImageName(IntPtr process, uint flags, StringBuilder name, ref uint size);
    public static string ImagePath(IntPtr process) {
        var name = new StringBuilder(32768);
        uint size = (uint)name.Capacity;
        if (!QueryFullProcessImageName(process, 0, name, ref size)) {
            throw new System.ComponentModel.Win32Exception(Marshal.GetLastWin32Error());
        }
        return name.ToString();
    }
    public static Release19SmokeWindow Find(uint pid) {
        Release19SmokeWindow result = new Release19SmokeWindow();
        EnumWindows(delegate(IntPtr hwnd, IntPtr unused) {
            uint owner;
            GetWindowThreadProcessId(hwnd, out owner);
            if (owner != pid) { return true; }
            var text = new StringBuilder(512);
            GetWindowText(hwnd, text, text.Capacity);
            if (text.ToString() != "轻笺 · 原生版") { return true; }
            result.Handle = hwnd;
            result.Title = text.ToString();
            result.Visible = IsWindowVisible(hwnd);
            return false;
        }, IntPtr.Zero);
        return result;
    }
}
'@

$started = [DateTime]::UtcNow
$watch = [Diagnostics.Stopwatch]::StartNew()
$oldAppData = [Environment]::GetEnvironmentVariable('APPDATA', 'Process')
$oldLocal = [Environment]::GetEnvironmentVariable('LOCALAPPDATA', 'Process')
$process = $null
$window = $null
$appIdentity = $null
$closePosted = $false
$timedOut = $false
$forced = $false
$errorText = $null
$sessionObserved = $false
$sessionEvidence = $null
$sessionHasTerminalNewline = $true
$sessionPath = Join-Path $roaming 'LevNoteNative/session.json'
$appStdout = Join-Path $smokeRoot 'app.stdout.log'
$appStderr = Join-Path $smokeRoot 'app.stderr.log'
$exe = Join-Path $bin 'levnote-native.exe'

try {
    [Environment]::SetEnvironmentVariable('APPDATA', $roaming, 'Process')
    [Environment]::SetEnvironmentVariable('LOCALAPPDATA', $local, 'Process')
    $process = Start-Process -FilePath $exe -ArgumentList @(('"' + $sample + '"')) `
        -WorkingDirectory $workspace -WindowStyle Hidden `
        -RedirectStandardOutput $appStdout -RedirectStandardError $appStderr -PassThru
    $heldHandle = $process.Handle
    $appIdentity = [ordered]@{
        PID = $process.Id
        Path = [Release19SmokeWin32]::ImagePath($heldHandle)
        StartTimeUtc = $process.StartTime.ToUniversalTime().ToString('o')
        HeldHandle = $heldHandle.ToInt64()
        PathMethod = 'QueryFullProcessImageName(exact held handle)'
    }
    if (-not [string]::Equals($appIdentity.Path, $exe, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Exact copied .19 host image path mismatch.'
    }

    $deadline = [DateTime]::UtcNow.AddSeconds(20)
    do {
        if ($process.HasExited) {
            throw "Host exited before readiness: $($process.ExitCode)"
        }
        $window = [Release19SmokeWin32]::Find([uint32]$process.Id)
        if ($window.Handle -ne [IntPtr]::Zero) { break }
        Start-Sleep -Milliseconds 100
    } while ([DateTime]::UtcNow -lt $deadline)
    if ($window.Handle -eq [IntPtr]::Zero) {
        throw 'Exact PID native window was not found within the readiness bound.'
    }
    if ($window.Visible) {
        throw 'Native window was visible despite hidden launch.'
    }

    Start-Sleep -Milliseconds 1200
    if ($process.HasExited) {
        throw 'Host exited during the queued-open delivery interval.'
    }
    $closePosted = [Release19SmokeWin32]::PostMessage(
        $window.Handle, 0x0010, [IntPtr]::Zero, [IntPtr]::Zero
    )
    if (-not $closePosted) {
        throw 'Exact owned window WM_CLOSE failed.'
    }
    if (-not $process.WaitForExit(15000)) {
        $timedOut = $true
        throw 'Exact owned process failed to exit after WM_CLOSE.'
    }
    if ($process.ExitCode -ne 0) {
        throw "Native app exit=$($process.ExitCode)"
    }
    if (-not (Test-Path -LiteralPath $sessionPath -PathType Leaf)) {
        throw 'Normal close did not produce an isolated session.'
    }
    $sessionBytes = [IO.File]::ReadAllBytes($sessionPath)
    $sessionHasTerminalNewline = $sessionBytes.Count -gt 0 -and
        ($sessionBytes[$sessionBytes.Count - 1] -eq 10 -or $sessionBytes[$sessionBytes.Count - 1] -eq 13)
    if ($sessionHasTerminalNewline) {
        throw 'Post-close session ends with a terminal newline.'
    }
    $session = Read-JsonFile $sessionPath
    if (@($session.tabs).Count -ne 1) {
        throw "Isolated session has $(@($session.tabs).Count) tabs; expected exactly one."
    }
    $tab = @($session.tabs)[0]
    if ($tab.path -ne $sample -or ($tab.PSObject.Properties.Name -contains 'draft')) {
        throw 'Post-close session did not preserve the source path without a draft.'
    }
    $sessionObserved = $true
    $sessionEvidence = [ordered]@{
        Path = $tab.path
        Encoding = $tab.encoding
        BOM = $tab.bom
        HasDraft = $false
    }
    $sessionAfter = Join-Path $smokeRoot 'session-after-close.json'
    Copy-Item -LiteralPath $sessionPath -Destination $sessionAfter -ErrorAction Stop
} catch {
    $errorText = $_.Exception.Message
} finally {
    [Environment]::SetEnvironmentVariable('APPDATA', $oldAppData, 'Process')
    [Environment]::SetEnvironmentVariable('LOCALAPPDATA', $oldLocal, 'Process')
    if ($null -ne $process -and -not $process.HasExited) {
        $forced = $true
        $process.Kill()
        if (-not $process.WaitForExit(10000) -and $null -eq $errorText) {
            $errorText = 'Owned app forced cleanup failed.'
        }
    }
}

$candidateAfter = @(Get-DirectoryInventory $ctx.candidateDirectory)
$copiedAfter = @(Get-DirectoryInventory $bin)
$sourceAfter = Get-FileFingerprint $sample
$bundlePreserved = $true
try {
    Assert-InventoryContainsExact $candidateBefore $candidateAfter 'Candidate during hidden host smoke' $false
    Assert-InventoryContainsExact $copiedBefore $copiedAfter 'Copied host bundle during hidden host smoke' $false
} catch {
    $bundlePreserved = $false
    if ($null -eq $errorText) {
        $errorText = $_.Exception.Message
    }
}
if (-not (Test-FingerprintEqual $sourceBefore $sourceAfter) -and $null -eq $errorText) {
    $errorText = 'The unchanged Markdown source changed during hidden startup/close.'
}
$watch.Stop()
$status = [ordered]@{
    Schema = 1
    Status = $(if ($null -eq $errorText) { 'PASS' } else { 'FAIL' })
    StartedAtUtc = $started.ToString('o')
    FinishedAtUtc = [DateTime]::UtcNow.ToString('o')
    WallSeconds = $watch.Elapsed.TotalSeconds
    CandidateBundle = $ctx.candidateDirectory
    ExecutedByteIdenticalBundle = $bin
    ExecutablePath = $exe
    ExecutableVolume = [IO.Path]::GetPathRoot($exe)
    NormalTempPath = $normalTemp
    NormalTempVolume = [IO.Path]::GetPathRoot($normalTemp)
    RunOwnedConfigRoot = $configRoot
    ConfigVolume = [IO.Path]::GetPathRoot($configRoot)
    ConfigAppDataRoot = $roaming
    ConfigLocalAppDataRoot = $local
    ConfigRootRetained = $true
    ConfigSessionPath = $sessionPath
    StartedHidden = $true
    LaunchMethod = 'Start-Process -WindowStyle Hidden'
    AppArguments = @($sample)
    AppStarts = 1
    StartAttemptCount = 1
    App = $appIdentity
    WindowFound = $null -ne $window -and $window.Handle -ne [IntPtr]::Zero
    WindowHandle = $(if ($null -ne $window) { $window.Handle.ToInt64() } else { 0 })
    WindowTitle = $(if ($null -ne $window) { $window.Title } else { '' })
    WindowVisible = $(if ($null -ne $window) { $window.Visible } else { $null })
    GracefulWMClosePosted = $closePosted
    WMClosePostCount = $(if ($closePosted) { 1 } else { 0 })
    GracefulCloseTimedOut = $timedOut
    ForcedAppCleanup = $forced
    AppExitCode = $(if ($null -ne $process -and $process.HasExited) { $process.ExitCode } else { $null })
    OwnedAppStillLive = $null -ne $process -and -not $process.HasExited
    SessionObserved = $sessionObserved
    SessionHasTerminalNewline = $sessionHasTerminalNewline
    OpenedDocument = $sessionEvidence
    SourceBefore = $sourceBefore
    SourceAfter = $sourceAfter
    SourcePreserved = Test-FingerprintEqual $sourceBefore $sourceAfter
    CandidateAfter = $candidateAfter
    CopiedAfter = $copiedAfter
    BundlePreserved = $bundlePreserved
    AppStdoutBytes = $(if (Test-Path -LiteralPath $appStdout) { [Int64](Get-Item -LiteralPath $appStdout).Length } else { [Int64]0 })
    AppStderrBytes = $(if (Test-Path -LiteralPath $appStderr) { [Int64](Get-Item -LiteralPath $appStderr).Length } else { [Int64]0 })
    ConfigFilesAfterClose = @(Get-DirectoryInventory $configRoot)
    RequestedWorkerOperations = 0
    NoProcessNameScanOrKill = $true
    Scope = 'One hidden start of the exact copied .19 host from F: with isolated run-owned APPDATA/LOCALAPPDATA on the normal C: temp volume, one unchanged Markdown source, one owned WM_CLOSE, normal exit, and post-close clean one-tab session without terminal LF/CR. This is separate from the managed worker smoke and does not establish visible UI, physical IME, dirty Save, process memory, CPU or the fourth worker itself.'
    Error = $errorText
}
Write-JsonFile (Join-Path $smokeRoot 'result.json') $status 16
$status | ConvertTo-Json -Depth 8
if ($null -ne $process) {
    $process.Dispose()
}
if ($null -ne $errorText) {
    exit 1
}
