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
    'levnote-native-release23-image-smoke-' + [Guid]::NewGuid().ToString('N'))
$roaming = Join-Path $configRoot 'appdata/Roaming'
$local = Join-Path $configRoot 'appdata/Local'
foreach ($path in @($smokeRoot, $bin, $workspace, $configRoot, $roaming, $local)) {
    if (Test-Path -LiteralPath $path) {
        throw "Hidden host smoke path must be fresh: $path"
    }
    New-Item -ItemType Directory -Path $path -ErrorAction Stop | Out-Null
}

$candidateBefore = @(Get-DirectoryInventory $ctx.candidateDirectory)
foreach ($source in Get-ChildItem -LiteralPath $ctx.candidateDirectory -File -Recurse) {
    $relative = [IO.Path]::GetRelativePath($ctx.candidateDirectory, $source.FullName)
    $destination = Join-Path $bin $relative
    $parent = Split-Path $destination -Parent
    if (-not (Test-Path -LiteralPath $parent -PathType Container)) {
        New-Item -ItemType Directory -Path $parent -ErrorAction Stop | Out-Null
    }
    Copy-Item -LiteralPath $source.FullName -Destination $destination -ErrorAction Stop
}
$bundleBefore = @(Get-DirectoryInventory $bin)
Assert-InventoryContainsExact $candidateBefore $bundleBefore `
    'One byte-identical hidden host bundle copy' $false
$packageResult = Read-JsonFile (Join-Path $evidenceRoot 'package-result.json')
if ($candidateBefore.Count -ne [int]$packageResult.CandidateFileCount -or
    (Get-InventoryIdentity $candidateBefore) -cne
        [string]$packageResult.CandidateInventoryIdentity) {
    throw 'The hidden host candidate differs from the packaged candidate identity.'
}

$mixedFixtureName = 'route-gate.diagramnote'
$imageFixtureName = 'image-fixture.png'
$mixedFixture = Join-Path $workspace $mixedFixtureName
$imageFixture = Join-Path $workspace $imageFixtureName
Copy-Item -LiteralPath $ctx.MixedTextFixturePath -Destination $mixedFixture -ErrorAction Stop
Copy-Item -LiteralPath $ctx.ImageFixturePath -Destination $imageFixture -ErrorAction Stop
$mixedBefore = Get-FileFingerprint $mixedFixture
$imageBefore = Get-FileFingerprint $imageFixture
if (-not (Test-FileContentEqual ([pscustomobject]$ctx.MixedTextFixture) $mixedBefore) -or
    -not (Test-FileContentEqual ([pscustomobject]$ctx.ImageFixture) $imageBefore)) {
    throw 'Workspace fixture copy differs from its accepted source identity.'
}
$workspaceBefore = @(Get-DirectoryInventory $workspace)

$pluginDirectory = Join-Path $roaming 'LevNoteNative/plugins'
New-Item -ItemType Directory -Path $pluginDirectory -ErrorAction Stop | Out-Null
$installedPlugin = Join-Path $pluginDirectory 'diagram-notes.json'
Copy-Item -LiteralPath $ctx.ReviewedDiagramNotesPath -Destination $installedPlugin -ErrorAction Stop
$pluginBefore = Get-FileFingerprint $installedPlugin
if (-not (Test-FileContentEqual ([pscustomobject]$ctx.ReviewedDiagramNotes) $pluginBefore)) {
    throw 'Seeded installed plugin differs from the reviewed source example.'
}
$sessionPath = Join-Path $roaming 'LevNoteNative/session.json'
$exe = Join-Path $bin 'levnote-native.exe'

Add-Type -TypeDefinition @'
using System;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Text;

public struct Release23SmokeWindow {
    public IntPtr Handle;
    public string Title;
    public bool Visible;
}

[StructLayout(LayoutKind.Sequential)]
public struct Release23JobAccounting {
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
public struct Release23IoCounters {
    public ulong ReadOperationCount;
    public ulong WriteOperationCount;
    public ulong OtherOperationCount;
    public ulong ReadTransferCount;
    public ulong WriteTransferCount;
    public ulong OtherTransferCount;
}

[StructLayout(LayoutKind.Sequential)]
public struct Release23JobLimitInformation {
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
public struct Release23JobExtendedLimit {
    public Release23JobLimitInformation BasicLimitInformation;
    public Release23IoCounters IoInfo;
    public UIntPtr ProcessMemoryLimit;
    public UIntPtr JobMemoryLimit;
    public UIntPtr PeakProcessMemoryUsed;
    public UIntPtr PeakJobMemoryUsed;
}

public static class Release23SmokeWin32 {
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
        IntPtr job, int infoClass, ref Release23JobExtendedLimit info, uint length);
    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool AssignProcessToJobObject(IntPtr job, IntPtr process);
    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool QueryInformationJobObject(
        IntPtr job, int infoClass, out Release23JobAccounting info,
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

    public static Release23SmokeWindow FindWindow(uint ownerId) {
        Release23SmokeWindow result = new Release23SmokeWindow();
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
        var info = new Release23JobExtendedLimit();
        info.BasicLimitInformation.LimitFlags = 0x00002000;
        uint length = (uint)Marshal.SizeOf<Release23JobExtendedLimit>();
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

    public static Release23JobAccounting Accounting(IntPtr job) {
        Release23JobAccounting info;
        uint returned;
        uint length = (uint)Marshal.SizeOf<Release23JobAccounting>();
        if (!QueryInformationJobObject(job, 1, out info, length, out returned)) {
            throw new Win32Exception(Marshal.GetLastWin32Error());
        }
        return info;
    }
}
'@

function Invoke-HiddenHostRun(
    [string]$Name,
    [string]$Kind,
    [string[]]$LaunchPath,
    [ValidateRange(1, 10000)][int]$SettlingMilliseconds,
    [object]$ExpectedSessionBaseline
) {
    $runRoot = Join-Path $smokeRoot $Name
    if (Test-Path -LiteralPath $runRoot) {
        throw "Fresh hidden host run directory already exists: $runRoot"
    }
    New-Item -ItemType Directory -Path $runRoot -ErrorAction Stop | Out-Null
    $appStdout = Join-Path $runRoot 'app.stdout.log'
    $appStderr = Join-Path $runRoot 'app.stderr.log'
    $sessionCopy = Join-Path $runRoot 'session-after-close.json'
    $sessionBeforeRunPath = Join-Path $runRoot 'session-before-run.json'
    $sessionBeforeRunCopy = Join-Path $runRoot 'session-before-run-copy.json'
    $resultPath = Join-Path $runRoot 'result.json'
    $started = [DateTime]::UtcNow
    $watch = [Diagnostics.Stopwatch]::StartNew()
    $oldAppData = [Environment]::GetEnvironmentVariable('APPDATA', 'Process')
    $oldLocal = [Environment]::GetEnvironmentVariable('LOCALAPPDATA', 'Process')
    $process = $null
    $processDisposed = $false
    $appExitCode = $null
    $ownedAppStillLive = $null
    $job = [IntPtr]::Zero
    $window = $null
    $appIdentity = $null
    $closePosted = $false
    $timedOut = $false
    $forced = $false
    $errorText = $null
    $sessionFacts = $null
    $sessionBeforeRun = $null
    $sessionBeforeRunCopyPath = $null
    $sessionBeforeCloseFingerprint = $null
    $sessionBeforeCloseFacts = $null
    $preCloseSessionObservationAttempted = $false
    $preCloseSessionStrictShapeObserved = $false
    $preCloseSessionObservationError = $null
    $sessionAfterCloseFingerprint = $null
    $freshSessionPublication = $false
    $closePublicationAfterProcessStart = $false
    $settlingCompleted = $false
    $processAliveAfterSettling = $false
    $settlingAliveChecks = 0
    $settlingWatch = $null
    $jobAfterAssignment = $null
    $jobAfterExit = $null
    $arguments = @($LaunchPath)
    $argumentString = (@($arguments | ForEach-Object { '"' + $_ + '"' }) -join ' ')

    try {
        [Environment]::SetEnvironmentVariable('APPDATA', $roaming, 'Process')
        [Environment]::SetEnvironmentVariable('LOCALAPPDATA', $local, 'Process')
        $sessionBeforeRun = Get-OptionalFileState $sessionPath
        if ($sessionBeforeRun.Present) {
            Copy-Item -LiteralPath $sessionPath -Destination $sessionBeforeRunCopy -ErrorAction Stop
            $sessionBeforeRunCopyPath = $sessionBeforeRunCopy
            [IO.File]::SetLastWriteTimeUtc(
                $sessionBeforeRunCopyPath,
                [DateTime]::new(
                    [Int64]$sessionBeforeRun.LastWriteUtcTicks,
                    [DateTimeKind]::Utc))
            if (-not (Test-FileStateContentEqual $sessionBeforeRun (
                    Get-OptionalFileState $sessionBeforeRunCopyPath))) {
                throw 'Captured pre-run session copy differs from its baseline identity.'
            }
        }
        if ($Kind -eq 'mixed-cli-open' -and [bool]$sessionBeforeRun.Present) {
            throw 'Mixed CLI run requires an absent pre-run session in the fresh isolated config.'
        }
        if ($Kind -eq 'path-only-session-restore' -and -not [bool]$sessionBeforeRun.Present) {
            throw 'Session-restore run requires a captured pre-run session baseline.'
        }
        if ($null -ne $ExpectedSessionBaseline -and
            -not (Test-FileStateContentEqual $ExpectedSessionBaseline $sessionBeforeRun)) {
            throw 'Captured pre-run session differs from the preceding run publication.'
        }
        Write-JsonFile $sessionBeforeRunPath ([ordered]@{
            Schema = 1
            CapturedAtUtc = [DateTime]::UtcNow.ToString('o')
            CapturedBeforeProcessStart = $true
            Session = $sessionBeforeRun
            ByteCopyPath = $sessionBeforeRunCopyPath
        }) 8
        $startParams = @{
            FilePath = $exe
            WorkingDirectory = $workspace
            WindowStyle = 'Hidden'
            RedirectStandardOutput = $appStdout
            RedirectStandardError = $appStderr
            PassThru = $true
        }
        if ($argumentString -ne '') {
            $startParams.ArgumentList = @($argumentString)
        }
        $process = Start-Process @startParams
        $heldHandle = $process.Handle
        $appIdentity = [ordered]@{
            Id = $process.Id
            Path = [Release23SmokeWin32]::ImagePath($heldHandle)
            StartTimeUtc = $process.StartTime.ToUniversalTime().ToString('o')
            HeldHandle = $heldHandle.ToInt64()
            PathMethod = 'QueryFullProcessImageName(exact held handle)'
        }
        if (-not [string]::Equals(
                $appIdentity.Path, $exe, [StringComparison]::OrdinalIgnoreCase)) {
            throw 'Exact copied .23 host image path mismatch.'
        }

        $job = [Release23SmokeWin32]::CreateKillOnCloseJob()
        [Release23SmokeWin32]::Assign($job, $heldHandle)
        $jobAccounting = [Release23SmokeWin32]::Accounting($job)
        $jobAfterAssignment = [ordered]@{
            TotalProcesses = [uint32]$jobAccounting.TotalProcesses
            ActiveProcesses = [uint32]$jobAccounting.ActiveProcesses
            TotalTerminatedProcesses = [uint32]$jobAccounting.TotalTerminatedProcesses
        }
        if ($jobAfterAssignment.TotalProcesses -ne 1 -or
            $jobAfterAssignment.ActiveProcesses -ne 1) {
            throw "Host job contains $($jobAfterAssignment.TotalProcesses) total/$($jobAfterAssignment.ActiveProcesses) active processes before interaction; expected only the host."
        }

        $deadline = [DateTime]::UtcNow.AddSeconds(20)
        do {
            if ($process.HasExited) {
                throw "Host exited before readiness: $($process.ExitCode)"
            }
            $window = [Release23SmokeWin32]::FindWindow([uint32]$process.Id)
            if ($window.Handle -ne [IntPtr]::Zero) { break }
            Start-Sleep -Milliseconds 100
        } while ([DateTime]::UtcNow -lt $deadline)
        if ($window.Handle -eq [IntPtr]::Zero) {
            throw 'Exact host native window was not found within the readiness bound.'
        }
        if ($window.Visible) {
            throw 'Native window was visible despite hidden launch.'
        }

        # A hidden MyGo Surface is not guaranteed to draw after its first frame,
        # while this app persists from its View cycle. Let the owned process
        # settle without assigning semantic readiness to a session poll, then
        # close it exactly once and require a fresh publication after exit.
        $settlingWatch = [Diagnostics.Stopwatch]::StartNew()
        $settlingDeadline = [DateTime]::UtcNow.AddMilliseconds(
            $SettlingMilliseconds)
        while ([DateTime]::UtcNow -lt $settlingDeadline) {
            if ($process.HasExited) {
                throw "Host exited during the $SettlingMilliseconds ms settling period: $($process.ExitCode)"
            }
            $settlingAliveChecks++
            $remainingMilliseconds = [Math]::Max(
                1,
                [int][Math]::Ceiling(
                    ($settlingDeadline - [DateTime]::UtcNow).TotalMilliseconds))
            Start-Sleep -Milliseconds ([Math]::Min(100, $remainingMilliseconds))
        }
        if ($process.HasExited) {
            throw "Host exited during the $SettlingMilliseconds ms settling period: $($process.ExitCode)"
        }
        $settlingWatch.Stop()
        $settlingCompleted = $true
        $processAliveAfterSettling = $true

        $preCloseSessionObservationAttempted = $true
        try {
            $sessionBeforeCloseFingerprint = Get-OptionalFileState $sessionPath
            if ([bool]$sessionBeforeCloseFingerprint.Present) {
                try {
                    $sessionBeforeCloseFacts = Get-ReleaseSessionFacts `
                        $sessionPath $mixedFixture $imageFixture
                    $preCloseSessionStrictShapeObserved = $true
                } catch {
                    $preCloseSessionObservationError = $_.Exception.Message
                }
            }
        } catch {
            $preCloseSessionObservationError = $_.Exception.Message
        }

        $closePosted = [Release23SmokeWin32]::PostMessage(
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
        $jobAccounting = [Release23SmokeWin32]::Accounting($job)
        $jobAfterExit = [ordered]@{
            TotalProcesses = [uint32]$jobAccounting.TotalProcesses
            ActiveProcesses = [uint32]$jobAccounting.ActiveProcesses
            TotalTerminatedProcesses = [uint32]$jobAccounting.TotalTerminatedProcesses
        }
        if ($jobAfterExit.TotalProcesses -ne 1 -or
            $jobAfterExit.ActiveProcesses -ne 0) {
            throw "Host job recorded total=$($jobAfterExit.TotalProcesses), active=$($jobAfterExit.ActiveProcesses); a worker or helper was requested."
        }

        $sessionFacts = Get-ReleaseSessionFacts $sessionPath $mixedFixture $imageFixture
        Copy-Item -LiteralPath $sessionPath -Destination $sessionCopy -ErrorAction Stop
        $sessionAfterCloseFingerprint = Get-OptionalFileState $sessionPath
        $sessionStartTicks = [DateTimeOffset]::Parse(
            [string]$appIdentity.StartTimeUtc,
            [Globalization.CultureInfo]::InvariantCulture,
            [Globalization.DateTimeStyles]::RoundtripKind
        ).UtcDateTime.Ticks
        $baselineChanged = -not [bool]$sessionBeforeRun.Present -or
            -not (Test-FileStateContentEqual $sessionBeforeRun $sessionAfterCloseFingerprint)
        $closePublicationAfterProcessStart = [bool]$sessionAfterCloseFingerprint.Present -and
            [Int64]$sessionAfterCloseFingerprint.LastWriteUtcTicks -ge $sessionStartTicks
        $freshSessionPublication = $baselineChanged -and $closePublicationAfterProcessStart
        if (-not $freshSessionPublication) {
            throw 'Post-close session was not freshly published after this run started relative to its captured baseline.'
        }
    } catch {
        $errorText = $_.Exception.Message
    } finally {
        [Environment]::SetEnvironmentVariable('APPDATA', $oldAppData, 'Process')
        [Environment]::SetEnvironmentVariable('LOCALAPPDATA', $oldLocal, 'Process')
        try {
            if ($null -ne $process -and -not $process.HasExited) {
                $forced = $true
                $process.Kill()
                if (-not $process.WaitForExit(10000) -and $null -eq $errorText) {
                    $errorText = 'Owned app forced cleanup failed.'
                }
            }
        } catch {
            if ($null -eq $errorText) {
                $errorText = "Owned app cleanup failed: $($_.Exception.Message)"
            }
        } finally {
            if ($job -ne [IntPtr]::Zero) {
                try {
                    [Release23SmokeWin32]::CloseHandle($job) | Out-Null
                    $job = [IntPtr]::Zero
                } catch {
                    if ($null -eq $errorText) {
                        $errorText = "Owned job cleanup failed: $($_.Exception.Message)"
                    }
                }
            }
            if ($null -ne $process) {
                try {
                    $ownedAppStillLive = -not $process.HasExited
                    if ($process.HasExited) {
                        $appExitCode = $process.ExitCode
                    }
                } finally {
                    try {
                        $process.Dispose()
                        $processDisposed = $true
                    } catch {
                        if ($null -eq $errorText) {
                            $errorText = "Owned process Dispose failed: $($_.Exception.Message)"
                        }
                    }
                    $process = $null
                }
            }
        }
    }
    $watch.Stop()
    $result = [ordered]@{
        Schema = 1
        Name = $Name
        Kind = $Kind
        Status = $(if ($null -eq $errorText) { 'PASS' } else { 'FAIL' })
        Error = $errorText
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
        AppArguments = $arguments
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
        AppExitCode = $appExitCode
        OwnedAppStillLive = $ownedAppStillLive
        ProcessDisposed = $processDisposed
        JobAfterAssignment = $jobAfterAssignment
        JobAfterExit = $jobAfterExit
        RequestedWorkerOperations = $(
            if ($null -eq $jobAfterExit) { $null }
            else { [Math]::Max(0, [int]$jobAfterExit.TotalProcesses - 1) }
        )
        SessionObserved = $null -ne $sessionFacts
        SessionBeforeRun = $sessionBeforeRun
        SessionBeforeRunRecordPath = $sessionBeforeRunPath
        SessionBeforeRunCopyPath = $sessionBeforeRunCopyPath
        SessionSettlingMillisecondsRequested = $SettlingMilliseconds
        SessionSettlingMillisecondsActual = $(
            if ($null -eq $settlingWatch) { 0 }
            else { $settlingWatch.Elapsed.TotalMilliseconds }
        )
        SessionSettlingAliveChecks = $settlingAliveChecks
        SessionSettlingCompleted = $settlingCompleted
        ProcessAliveAfterSettling = $processAliveAfterSettling
        CloseIssuedAfterSettling = $settlingCompleted -and $closePosted
        PreCloseSessionObservationAttempted = $preCloseSessionObservationAttempted
        PreCloseSessionStrictShapeObserved = $preCloseSessionStrictShapeObserved
        PreCloseSessionObservationError = $preCloseSessionObservationError
        SessionFactsBeforeClose = $sessionBeforeCloseFacts
        SessionBeforeCloseFingerprint = $sessionBeforeCloseFingerprint
        SessionFactsAfterClose = $sessionFacts
        SessionAfterCloseFingerprint = $sessionAfterCloseFingerprint
        FreshSessionPublication = $freshSessionPublication
        ClosePublicationAfterProcessStart = $closePublicationAfterProcessStart
        SessionCopyPath = $sessionCopy
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
        FixturePaths = [ordered]@{
            Text = $mixedFixture
            Image = $imageFixture
        }
        NoProcessNameScanOrKill = $true
        Scope = $(if ($Kind -eq 'mixed-cli-open') {
                'One hidden start of an exact copied .23 host with two CLI paths: accepted custom-route text then PNG. Session proves ordered saved-absolute-path records and active mapping only; it does not prove PNG decode or physical rendering.'
            } else {
                'One hidden restart of the same exact copied .23 host and isolated config with no CLI paths. A fresh close-side publication proves restoration was saved after this run started, while the observed pre-close shape alone is not restoration proof; it does not prove PNG decode or physical rendering.'
            })
    }
    Write-JsonFile $resultPath $result 18
    if ($null -ne $errorText) {
        throw "Hidden host run '$Name' failed: $errorText"
    }
    return $result
}

try {
    $run1 = Invoke-HiddenHostRun `
        -Name 'run-1-mixed-cli-open' `
        -Kind 'mixed-cli-open' `
        -LaunchPath @($mixedFixture, $imageFixture) `
        -SettlingMilliseconds 5000
    $pluginAfterRun1 = Get-FileFingerprint $installedPlugin
    if (-not (Test-FingerprintEqual $pluginBefore $pluginAfterRun1)) {
        throw 'Installed plugin JSON changed during mixed CLI opening.'
    }
    $run2 = Invoke-HiddenHostRun `
        -Name 'run-2-path-only-restore' `
        -Kind 'path-only-session-restore' `
        -LaunchPath @() `
        -SettlingMilliseconds 5000 `
        -ExpectedSessionBaseline $run1.SessionAfterCloseFingerprint
} finally {
    # All state has already been recorded per run; cleanup cannot hide a failure.
}

$candidateAfter = @(Get-DirectoryInventory $ctx.candidateDirectory)
$bundleAfter = @(Get-DirectoryInventory $bin)
$workspaceAfter = @(Get-DirectoryInventory $workspace)
$pluginAfter = Get-FileFingerprint $installedPlugin
$candidatePreserved = $true
try {
    Assert-InventoryContainsExact $candidateBefore $candidateAfter `
        'Candidate during two hidden host runs' $true
    Assert-InventoryContainsExact $bundleBefore $bundleAfter `
        'Copied bundle during two hidden host runs' $true
    Assert-InventoryContainsExact $workspaceBefore $workspaceAfter `
        'Fixture workspace during two hidden host runs' $true
} catch {
    $candidatePreserved = $false
}
if (-not $candidatePreserved) {
    throw 'Candidate, copied bundle, or fixture workspace changed during hidden host execution.'
}
if (-not (Test-FingerprintEqual $pluginBefore $pluginAfter)) {
    throw 'Installed plugin JSON changed during hidden host execution.'
}

$run1Creation = [DateTimeOffset]::Parse(
    [string]$run1.App.StartTimeUtc,
    [Globalization.CultureInfo]::InvariantCulture,
    [Globalization.DateTimeStyles]::RoundtripKind)
$run1Completion = [DateTimeOffset]::Parse(
    [string]$run1.FinishedAtUtc,
    [Globalization.CultureInfo]::InvariantCulture,
    [Globalization.DateTimeStyles]::RoundtripKind)
$run2Creation = [DateTimeOffset]::Parse(
    [string]$run2.App.StartTimeUtc,
    [Globalization.CultureInfo]::InvariantCulture,
    [Globalization.DateTimeStyles]::RoundtripKind)
$distinctSequentialRunLifetimes = $run1Creation -lt $run2Creation -and
    $run1Completion -lt $run2Creation

$summary = [ordered]@{
    Schema = 1
    CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
    Status = 'PASS'
    RunCount = 2
    RunOrder = @($run1.Name, $run2.Name)
    SameCopiedBundle = [string]$run1.ExecutedByteIdenticalBundle -ceq
        [string]$run2.ExecutedByteIdenticalBundle
    SameConfigRoot = [string]$run1.RunOwnedConfigRoot -ceq
        [string]$run2.RunOwnedConfigRoot
    DistinctSequentialRunLifetimes = $distinctSequentialRunLifetimes
    Run1CompletedBeforeRun2Started = $run1Completion -lt $run2Creation
    BothRunsSettledAliveForBoundedPeriod =
        $run1.SessionSettlingCompleted -and $run1.ProcessAliveAfterSettling -and
        $run2.SessionSettlingCompleted -and $run2.ProcessAliveAfterSettling
    BothRunsRecordedOptionalPreCloseSessionObservation =
        $run1.PreCloseSessionObservationAttempted -and
        $run2.PreCloseSessionObservationAttempted
    BothRunsPublishedFreshSessionAfterStart = $run1.FreshSessionPublication -and
        $run2.FreshSessionPublication
    BothRunsClosedOnlyAfterSettling = $run1.CloseIssuedAfterSettling -and
        $run2.CloseIssuedAfterSettling
    Run1 = $run1
    Run2 = $run2
    CandidatePreserved = $true
    CopiedBundlePreserved = $true
    FixturesPreserved = $true
    InstalledPluginPreserved = $true
    ConfigRoot = $configRoot
    ConfigSessionPath = $sessionPath
    InstalledPluginPath = $installedPlugin
    ConfigFilesAfterRuns = @(Get-DirectoryInventory $configRoot)
    Scope = 'Two hidden exact-copied-host lifecycle checks share one isolated C: config: mixed CLI open then saved-absolute-path mixed restore. Each run checks that its owned host remains alive through one bounded 5000 ms settling period, records any pre-close session observation descriptively, posts exactly one WM_CLOSE, and requires a fresh ordered mixed-session publication after this run started. Run 2 additionally requires its captured pre-run baseline to match run 1 and the post-close publication to differ from that baseline. Neither session nor window evidence proves successful PNG decode/rendering. No process-name scan/kill, worker request, physical input/IME, resource measurement, or packaged source claim.'
}
Write-JsonFile (Join-Path $smokeRoot 'result.json') $summary 18
$summary | ConvertTo-Json -Depth 8
