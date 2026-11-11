$ErrorActionPreference = 'Stop'
$evidenceRoot = [IO.Path]::GetFullPath($PSScriptRoot)
. (Join-Path $evidenceRoot 'delivery-lib.ps1')
$ctx = Read-DeliveryContext $evidenceRoot
Assert-DeliveryExecution $ctx

if (-not (Test-Path -LiteralPath (Join-Path $evidenceRoot 'package-result.json') -PathType Leaf)) {
    throw 'Run the package stage first.'
}
$smokeRoot = Join-Path $evidenceRoot 'host-smoke'
if (Test-Path -LiteralPath $smokeRoot) {
    throw "Fresh hidden host smoke directory already exists: $smokeRoot"
}
$bin = Join-Path $smokeRoot 'bundle'
$workspace = Join-Path $smokeRoot 'workspace'
$normalTemp = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
if (-not [string]::Equals(
        [IO.Path]::GetPathRoot($normalTemp),
        'C:\',
        [StringComparison]::OrdinalIgnoreCase)) {
    throw "Hidden host smoke requires the normal default TEMP path on C:, resolved to $normalTemp."
}
$configRoot = Join-Path $normalTemp (
    'levnote-native-release21-route-smoke-' + [Guid]::NewGuid().ToString('N'))
$roaming = Join-Path $configRoot 'appdata/Roaming'
$local = Join-Path $configRoot 'appdata/Local'
foreach ($path in @($smokeRoot, $bin, $workspace, $configRoot, $roaming, $local)) {
    if (Test-Path -LiteralPath $path) {
        throw "Hidden host smoke path must be fresh: $path"
    }
    New-Item -ItemType Directory -Path $path -ErrorAction Stop | Out-Null
}

$candidateBefore = @(Get-DirectoryInventory $ctx.candidateDirectory)
if ($candidateBefore.Count -ne [int]$ctx.expectedCandidateFiles) {
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

$fixture = Join-Path $workspace 'route-gate.diagramnote'
$fixtureText = @'
<?xml version="1.0" encoding="UTF-8"?>
<mxfile host="LevNoteRouteGate" modified="2026-11-10T00:00:00.000Z" agent="fixture" version="24.0.0" customUnknown="root-keep">
  <diagram id="route-page" name="Route Page" customUnknown="diagram-keep">
    <mxGraphModel dx="800" dy="600" grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" fold="1" page="1" pageScale="1" pageWidth="827" pageHeight="1169" math="0" shadow="0" customUnknown="model-keep">
      <root>
        <mxCell id="0" customUnknown="cell-keep"/>
        <mxCell id="1" parent="0"/>
        <mxCell id="2" value="Route test" style="rounded=0;whiteSpace=wrap;html=1;" vertex="1" parent="1" customUnknown="vertex-keep">
          <mxGeometry x="80" y="80" width="180" height="60" as="geometry" customUnknown="geometry-keep"/>
        </mxCell>
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>
'@
$fixtureText = $fixtureText.Replace("`r`n", "`n").Replace("`n", "`r`n")
[IO.File]::WriteAllText($fixture, $fixtureText, [Text.UTF8Encoding]::new($true))
$sourceBefore = Get-FileFingerprint $fixture
if ($fixtureText.Contains("`n") -and -not $fixtureText.Contains("`r`n")) {
    throw 'Fixture did not retain CRLF.'
}

$pluginDirectory = Join-Path $roaming 'LevNoteNative/plugins'
New-Item -ItemType Directory -Path $pluginDirectory -ErrorAction Stop | Out-Null
$installedPlugin = Join-Path $pluginDirectory 'diagram-notes.json'
Copy-Item -LiteralPath $ctx.reviewedDiagramNotesPath -Destination $installedPlugin -ErrorAction Stop
$pluginBefore = Get-FileFingerprint $installedPlugin
if ($pluginBefore.Bytes -ne [Int64]$ctx.reviewedDiagramNotesBytes -or
    $pluginBefore.SHA256 -cne $ctx.reviewedDiagramNotesSHA256) {
    throw 'Seeded installed plugin differs from the reviewed source example.'
}

Add-Type -TypeDefinition @'
using System;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Text;

public struct Release21SmokeWindow {
    public IntPtr Handle;
    public string Title;
    public bool Visible;
}

[StructLayout(LayoutKind.Sequential)]
public struct Release21JobAccounting {
    public long TotalUserTime;
    public long TotalKernelTime;
    public long ThisPeriodTotalUserTime;
    public long ThisPeriodTotalKernelTime;
    public uint TotalPageFaultCount;
    public uint TotalProcesses;
    public uint ActiveProcesses;
    public uint TotalTerminatedProcesses;
}

[StructLayout(LayoutKind.Sequential)]
public struct Release21IoCounters {
    public ulong ReadOperationCount;
    public ulong WriteOperationCount;
    public ulong OtherOperationCount;
    public ulong ReadTransferCount;
    public ulong WriteTransferCount;
    public ulong OtherTransferCount;
}

[StructLayout(LayoutKind.Sequential)]
public struct Release21JobLimitInformation {
    public long PerProcessUserTimeLimit;
    public long PerJobUserTimeLimit;
    public uint LimitFlags;
    public UIntPtr MinimumWorkingSetSize;
    public UIntPtr MaximumWorkingSetSize;
    public uint ActiveProcessLimit;
    public UIntPtr Affinity;
    public uint PriorityClass;
    public uint SchedulingClass;
}

[StructLayout(LayoutKind.Sequential)]
public struct Release21JobExtendedLimit {
    public Release21JobLimitInformation BasicLimitInformation;
    public Release21IoCounters IoInfo;
    public UIntPtr ProcessMemoryLimit;
    public UIntPtr JobMemoryLimit;
    public UIntPtr PeakProcessMemoryUsed;
    public UIntPtr PeakJobMemoryUsed;
}

public static class Release21SmokeWin32 {
    private delegate bool Callback(IntPtr hwnd, IntPtr data);

    [DllImport("user32.dll")]
    private static extern bool EnumWindows(Callback callback, IntPtr data);
    [DllImport("user32.dll")]
    private static extern uint GetWindowThreadProcessId(IntPtr hwnd, out uint owner);
    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    private static extern int GetWindowText(IntPtr hwnd, StringBuilder text, int count);
    [DllImport("user32.dll")]
    private static extern bool IsWindowVisible(IntPtr hwnd);
    [DllImport("user32.dll")]
    public static extern bool PostMessage(IntPtr hwnd, uint message, IntPtr w, IntPtr l);
    [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    private static extern bool QueryFullProcessImageName(
        IntPtr process, uint flags, StringBuilder name, ref uint size);
    [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    private static extern IntPtr CreateJobObject(IntPtr attributes, string name);
    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool SetInformationJobObject(
        IntPtr job, int infoClass, ref Release21JobExtendedLimit info, uint length);
    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool AssignProcessToJobObject(IntPtr job, IntPtr process);
    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool QueryInformationJobObject(
        IntPtr job, int infoClass, out Release21JobAccounting info,
        uint length, out uint returned);
    [DllImport("kernel32.dll", SetLastError = true)]
    public static extern bool CloseHandle(IntPtr handle);

    public static string ImagePath(IntPtr process) {
        var name = new StringBuilder(32768);
        uint size = (uint)name.Capacity;
        if (!QueryFullProcessImageName(process, 0, name, ref size)) {
            throw new Win32Exception(Marshal.GetLastWin32Error());
        }
        return name.ToString();
    }

    public static Release21SmokeWindow FindWindow(uint ownerId) {
        Release21SmokeWindow result = new Release21SmokeWindow();
        EnumWindows(delegate(IntPtr hwnd, IntPtr unused) {
            uint owner;
            GetWindowThreadProcessId(hwnd, out owner);
            if (owner != ownerId) { return true; }
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

    public static IntPtr CreateKillOnCloseJob() {
        IntPtr job = CreateJobObject(IntPtr.Zero, null);
        if (job == IntPtr.Zero) {
            throw new Win32Exception(Marshal.GetLastWin32Error());
        }
        var info = new Release21JobExtendedLimit();
        info.BasicLimitInformation.LimitFlags = 0x00002000;
        uint length = (uint)Marshal.SizeOf<Release21JobExtendedLimit>();
        if (!SetInformationJobObject(job, 9, ref info, length)) {
            int error = Marshal.GetLastWin32Error();
            CloseHandle(job);
            throw new Win32Exception(error);
        }
        return job;
    }

    public static void Assign(IntPtr job, IntPtr process) {
        if (!AssignProcessToJobObject(job, process)) {
            throw new Win32Exception(Marshal.GetLastWin32Error());
        }
    }

    public static uint TotalProcesses(IntPtr job) {
        Release21JobAccounting info;
        uint returned;
        uint length = (uint)Marshal.SizeOf<Release21JobAccounting>();
        if (!QueryInformationJobObject(job, 1, out info, length, out returned)) {
            throw new Win32Exception(Marshal.GetLastWin32Error());
        }
        return info.TotalProcesses;
    }
}
'@

$started = [DateTime]::UtcNow
$watch = [Diagnostics.Stopwatch]::StartNew()
$oldAppData = [Environment]::GetEnvironmentVariable('APPDATA', 'Process')
$oldLocal = [Environment]::GetEnvironmentVariable('LOCALAPPDATA', 'Process')
$process = $null
$job = [IntPtr]::Zero
$window = $null
$appIdentity = $null
$closePosted = $false
$timedOut = $false
$forced = $false
$errorText = $null
$sessionObserved = $false
$sessionEvidence = $null
$sessionHasTerminalNewline = $true
$jobProcessesAfterAssignment = 0
$jobProcessesAfterExit = 0
$sessionPath = Join-Path $roaming 'LevNoteNative/session.json'
$appStdout = Join-Path $smokeRoot 'app.stdout.log'
$appStderr = Join-Path $smokeRoot 'app.stderr.log'
$exe = Join-Path $bin 'levnote-native.exe'

try {
    [Environment]::SetEnvironmentVariable('APPDATA', $roaming, 'Process')
    [Environment]::SetEnvironmentVariable('LOCALAPPDATA', $local, 'Process')
    $process = Start-Process -FilePath $exe `
        -ArgumentList @(('"' + $fixture + '"')) `
        -WorkingDirectory $workspace -WindowStyle Hidden `
        -RedirectStandardOutput $appStdout -RedirectStandardError $appStderr -PassThru
    $heldHandle = $process.Handle
    $appIdentity = [ordered]@{
        Id = $process.Id
        Path = [Release21SmokeWin32]::ImagePath($heldHandle)
        StartTimeUtc = $process.StartTime.ToUniversalTime().ToString('o')
        HeldHandle = $heldHandle.ToInt64()
        PathMethod = 'QueryFullProcessImageName(exact held handle)'
    }
    if (-not [string]::Equals(
            $appIdentity.Path, $exe, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Exact copied .21 host image path mismatch.'
    }

    $job = [Release21SmokeWin32]::CreateKillOnCloseJob()
    [Release21SmokeWin32]::Assign($job, $heldHandle)
    $jobProcessesAfterAssignment = [Release21SmokeWin32]::TotalProcesses($job)
    if ($jobProcessesAfterAssignment -ne 1) {
        throw "Host job contains $jobProcessesAfterAssignment processes before interaction; expected only the host."
    }

    $deadline = [DateTime]::UtcNow.AddSeconds(20)
    do {
        if ($process.HasExited) {
            throw "Host exited before readiness: $($process.ExitCode)"
        }
        $window = [Release21SmokeWin32]::FindWindow([uint32]$process.Id)
        if ($window.Handle -ne [IntPtr]::Zero) { break }
        Start-Sleep -Milliseconds 100
    } while ([DateTime]::UtcNow -lt $deadline)
    if ($window.Handle -eq [IntPtr]::Zero) {
        throw 'Exact host native window was not found within the readiness bound.'
    }
    if ($window.Visible) {
        throw 'Native window was visible despite hidden launch.'
    }

    Start-Sleep -Milliseconds 1200
    if ($process.HasExited) {
        throw 'Host exited during the queued-open delivery interval.'
    }
    $closePosted = [Release21SmokeWin32]::PostMessage(
        $window.Handle, 0x0010, [IntPtr]::Zero, [IntPtr]::Zero)
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
    $jobProcessesAfterExit = [Release21SmokeWin32]::TotalProcesses($job)
    if ($jobProcessesAfterExit -ne 1) {
        throw "Host job recorded $jobProcessesAfterExit total processes; a worker or helper was requested."
    }

    if (-not (Test-Path -LiteralPath $sessionPath -PathType Leaf)) {
        throw 'Normal close did not produce an isolated session.'
    }
    $sessionBytes = [IO.File]::ReadAllBytes($sessionPath)
    $sessionHasTerminalNewline = $sessionBytes.Count -gt 0 -and
        ($sessionBytes[$sessionBytes.Count - 1] -eq 10 -or
            $sessionBytes[$sessionBytes.Count - 1] -eq 13)
    if ($sessionHasTerminalNewline) {
        throw 'Post-close session ends with a terminal newline.'
    }
    $session = Read-JsonFile $sessionPath
    $tabs = @($session.tabs)
    if ($tabs.Count -ne 1) {
        throw "Isolated session has $($tabs.Count) tabs; expected exactly one."
    }
    $tab = $tabs[0]
    $tabProperties = @($tab.PSObject.Properties.Name)
    if ($tab.path -cne $fixture -or
        $tab.pluginId -cne 'diagram-notes' -or
        $tab.view -cne 'preview' -or
        $tab.encoding -cne 'UTF-8' -or
        $tab.bom -ne $true -or
        $tabProperties -contains 'draft') {
        throw "Post-close route/session identity differs: path=$($tab.path); pluginId=$($tab.pluginId); view=$($tab.view); encoding=$($tab.encoding); bom=$($tab.bom); draft=$($tabProperties -contains 'draft')"
    }
    $sessionObserved = $true
    $sessionEvidence = [ordered]@{
        Path = $tab.path
        PluginID = $tab.pluginId
        View = $tab.view
        Encoding = $tab.encoding
        BOM = [bool]$tab.bom
        HasDraft = $false
    }
    Copy-Item -LiteralPath $sessionPath `
        -Destination (Join-Path $smokeRoot 'session-after-close.json') -ErrorAction Stop
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
    if ($job -ne [IntPtr]::Zero) {
        [Release21SmokeWin32]::CloseHandle($job) | Out-Null
        $job = [IntPtr]::Zero
    }
}

$candidateAfter = @(Get-DirectoryInventory $ctx.candidateDirectory)
$copiedAfter = @(Get-DirectoryInventory $bin)
$sourceAfter = Get-FileFingerprint $fixture
$pluginAfter = Get-FileFingerprint $installedPlugin
$bundlePreserved = $true
try {
    Assert-InventoryContainsExact $candidateBefore $candidateAfter `
        'Candidate during hidden host smoke' $false
    Assert-InventoryContainsExact $copiedBefore $copiedAfter `
        'Copied host bundle during hidden host smoke' $false
} catch {
    $bundlePreserved = $false
    if ($null -eq $errorText) {
        $errorText = $_.Exception.Message
    }
}
if (-not (Test-FingerprintEqual $sourceBefore $sourceAfter) -and
    $null -eq $errorText) {
    $errorText = 'The unchanged draw.io fixture changed during hidden startup/close.'
}
if (-not (Test-FingerprintEqual $pluginBefore $pluginAfter) -and
    $null -eq $errorText) {
    $errorText = 'The installed diagram-notes JSON changed during hidden startup/close.'
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
    InstalledPluginPath = $installedPlugin
    StartedHidden = $true
    LaunchMethod = 'Start-Process -WindowStyle Hidden'
    AppArguments = @($fixture)
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
    AppExitCode = $(if ($null -ne $process -and $process.HasExited) {
            $process.ExitCode
        } else {
            $null
        })
    OwnedAppStillLive = $null -ne $process -and -not $process.HasExited
    SessionObserved = $sessionObserved
    SessionHasTerminalNewline = $sessionHasTerminalNewline
    OpenedDocument = $sessionEvidence
    FixtureBefore = $sourceBefore
    FixtureAfter = $sourceAfter
    FixturePreserved = Test-FingerprintEqual $sourceBefore $sourceAfter
    InstalledPluginBefore = $pluginBefore
    InstalledPluginAfter = $pluginAfter
    InstalledPluginPreserved = Test-FingerprintEqual $pluginBefore $pluginAfter
    CandidateAfter = $candidateAfter
    CopiedAfter = $copiedAfter
    BundlePreserved = $bundlePreserved
    JobProcessesAfterAssignment = $jobProcessesAfterAssignment
    JobTotalProcessesAfterExit = $jobProcessesAfterExit
    RequestedWorkerOperations = [Math]::Max(0, $jobProcessesAfterExit - 1)
    AppStdoutBytes = $(if (Test-Path -LiteralPath $appStdout) {
            [Int64](Get-Item -LiteralPath $appStdout).Length
        } else {
            [Int64]0
        })
    AppStderrBytes = $(if (Test-Path -LiteralPath $appStderr) {
            [Int64](Get-Item -LiteralPath $appStderr).Length
        } else {
            [Int64]0
        })
    ConfigFilesAfterClose = @(Get-DirectoryInventory $configRoot)
    NoProcessNameScanOrKill = $true
    Scope = 'One hidden start of an exact copied .21 host with isolated C: APPDATA/LOCALAPPDATA, one reviewed diagram-notes JSON seed, one UTF8+BOM/CRLF custom-suffix draw.io XML fixture with unknown metadata, one owned WM_CLOSE and normal exit. This establishes packaged configuration/path routing and clean lifecycle only; it does not establish visible rendering, physical input/IME, interactive SVG export, dirty or process memory or CPU use.'
    Error = $errorText
}
Write-JsonFile (Join-Path $smokeRoot 'result.json') $status 16
$status | ConvertTo-Json -Depth 9
if ($null -ne $process) {
    $process.Dispose()
}
if ($null -ne $errorText) {
    exit 1
}
