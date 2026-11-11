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
    'levnote-native-release26-image-ref-smoke-' + [Guid]::NewGuid().ToString('N'))
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
$sourceBefore = Get-ReadingSourceFacts $ctx.repositoryRoot
Write-JsonFile (Join-Path $smokeRoot 'inputs-before.json') ([ordered]@{
    Candidate = $candidateBefore
    Bundle = $bundleBefore
    Workspace = $workspaceBefore
    Plugin = $pluginBefore
    AcceptedSource = $sourceBefore
}) 18

Add-Type -TypeDefinition @'
using System;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Text;

public struct Release24SmokeWindow {
    public IntPtr Handle;
    public string Title;
    public bool Visible;
}

[StructLayout(LayoutKind.Sequential)]
public struct Release24JobAccounting {
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
public struct Release24IoCounters {
    public ulong ReadOperationCount;
    public ulong WriteOperationCount;
    public ulong OtherOperationCount;
    public ulong ReadTransferCount;
    public ulong WriteTransferCount;
    public ulong OtherTransferCount;
}

[StructLayout(LayoutKind.Sequential)]
public struct Release24JobLimitInformation {
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
public struct Release24JobExtendedLimit {
    public Release24JobLimitInformation BasicLimitInformation;
    public Release24IoCounters IoInfo;
    public UIntPtr ProcessMemoryLimit;
    public UIntPtr JobMemoryLimit;
    public UIntPtr PeakProcessMemoryUsed;
    public UIntPtr PeakJobMemoryUsed;
}

public static class Release24SmokeWin32 {
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
        IntPtr job, int infoClass, ref Release24JobExtendedLimit info, uint length);
    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool AssignProcessToJobObject(IntPtr job, IntPtr process);
    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool QueryInformationJobObject(
        IntPtr job, int infoClass, out Release24JobAccounting info,
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

    public static Release24SmokeWindow FindWindow(uint ownerId) {
        Release24SmokeWindow result = new Release24SmokeWindow();
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
        var info = new Release24JobExtendedLimit();
        info.BasicLimitInformation.LimitFlags = 0x00002000;
        uint length = (uint)Marshal.SizeOf<Release24JobExtendedLimit>();
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

    public static Release24JobAccounting Accounting(IntPtr job) {
        Release24JobAccounting info;
        uint returned;
        uint length = (uint)Marshal.SizeOf<Release24JobAccounting>();
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
    [ValidateRange(1, 10000)][int]$SettlingMilliseconds
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
    $sessionBeforeCloseCopy = Join-Path $runRoot 'session-before-close.json'
    $sessionBeforeCloseRecord = Join-Path $runRoot 'session-before-close-facts.json'
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
    $closePostCount = 0
    $closeRequestedAtUtc = $null
    $jobDisposed = $false
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
    $preCloseSessionFreshAfterStart = $false
    $preCloseSessionCapturedAtUtc = $null
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
            Path = [Release24SmokeWin32]::ImagePath($heldHandle)
            StartTimeUtc = $process.StartTime.ToUniversalTime().ToString('o')
            HeldHandle = $heldHandle.ToInt64()
            PathMethod = 'QueryFullProcessImageName(exact held handle)'
        }
        if (-not [string]::Equals(
                $appIdentity.Path, $exe, [StringComparison]::OrdinalIgnoreCase)) {
            throw 'Exact copied .26 host image path mismatch.'
        }

        $job = [Release24SmokeWin32]::CreateKillOnCloseJob()
        [Release24SmokeWin32]::Assign($job, $heldHandle)
        $jobAccounting = [Release24SmokeWin32]::Accounting($job)
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
            $window = [Release24SmokeWin32]::FindWindow([uint32]$process.Id)
            if ($window.Handle -ne [IntPtr]::Zero) { break }
            Start-Sleep -Milliseconds 100
        } while ([DateTime]::UtcNow -lt $deadline)
        if ($window.Handle -eq [IntPtr]::Zero) {
            throw 'Exact host native window was not found within the readiness bound.'
        }
        if ($window.Visible) {
            throw 'Native window was visible despite hidden launch.'
        }

        # The new recovery scheduler dispatches without another paint. After
        # bounded settling, capture and validate its fresh mixed session before
        # issuing the single owned close. No repaint or input is injected.
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
            if (-not [bool]$sessionBeforeCloseFingerprint.Present) {
                throw 'Required paint-independent autosave is absent before close.'
            }
            Copy-Item -LiteralPath $sessionPath -Destination $sessionBeforeCloseCopy -ErrorAction Stop
            [IO.File]::SetLastWriteTimeUtc($sessionBeforeCloseCopy,
                [DateTime]::new([Int64]$sessionBeforeCloseFingerprint.LastWriteUtcTicks, [DateTimeKind]::Utc))
            if (-not (Test-FileStateContentEqual $sessionBeforeCloseFingerprint (Get-OptionalFileState $sessionBeforeCloseCopy))) {
                throw 'Pre-close byte copy differs from the observed session.'
            }
            $sessionBeforeCloseFacts = Get-ReleaseSessionFacts `
                $sessionBeforeCloseCopy $mixedFixture $imageFixture
            $preCloseSessionStrictShapeObserved = $true
            $sessionStartTicks = [DateTimeOffset]::Parse(
                [string]$appIdentity.StartTimeUtc,
                [Globalization.CultureInfo]::InvariantCulture,
                [Globalization.DateTimeStyles]::RoundtripKind).UtcDateTime.Ticks
            $preCloseSessionFreshAfterStart = -not [bool]$sessionBeforeRun.Present -and
                [Int64]$sessionBeforeCloseFingerprint.LastWriteUtcTicks -ge $sessionStartTicks
            if (-not $preCloseSessionFreshAfterStart) {
                throw 'Pre-close session is not fresh relative to this exact process start.'
            }
        } catch {
            $preCloseSessionObservationError = $_.Exception.Message
        }
        $preCloseSessionCapturedAtUtc = [DateTime]::UtcNow.ToString('o')
        Write-JsonFile $sessionBeforeCloseRecord ([ordered]@{
            CapturedAtUtc = $preCloseSessionCapturedAtUtc
            CapturedBeforeWMClose = $true
            Fingerprint = $sessionBeforeCloseFingerprint
            ByteCopyPath = $sessionBeforeCloseCopy
            StrictFacts = $sessionBeforeCloseFacts
            FreshAfterProcessStart = $preCloseSessionFreshAfterStart
            Error = $preCloseSessionObservationError
        }) 18

        $closeRequestedAtUtc = [DateTime]::UtcNow.ToString('o')
        $closePostCount++
        $closePosted = [Release24SmokeWin32]::PostMessage(
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
        $jobAccounting = [Release24SmokeWin32]::Accounting($job)
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
        $baselineChanged = -not [bool]$sessionBeforeCloseFingerprint.Present -or
            [Int64]$sessionAfterCloseFingerprint.LastWriteUtcTicks -gt [Int64]$sessionBeforeCloseFingerprint.LastWriteUtcTicks
        $closePublicationAfterProcessStart = [bool]$sessionAfterCloseFingerprint.Present -and
            [Int64]$sessionAfterCloseFingerprint.LastWriteUtcTicks -ge $sessionStartTicks
        $closeTicks = [DateTimeOffset]::Parse($closeRequestedAtUtc).UtcDateTime.Ticks
        $freshSessionPublication = $baselineChanged -and $closePublicationAfterProcessStart -and
            [Int64]$sessionAfterCloseFingerprint.LastWriteUtcTicks -ge $closeTicks
        if (-not $freshSessionPublication) {
            throw 'Post-close session was not freshly published after this run started relative to its captured baseline.'
        }
        if ($null -ne $preCloseSessionObservationError) {
            throw "Pre-close autosave failed; the owned host was closed normally: $preCloseSessionObservationError"
        }
    } catch {
        $errorText = $_.Exception.Message
    } finally {
        [Environment]::SetEnvironmentVariable('APPDATA', $oldAppData, 'Process')
        [Environment]::SetEnvironmentVariable('LOCALAPPDATA', $oldLocal, 'Process')
        try {
            if ($null -ne $process -and -not $process.HasExited) {
                # Early harness failures still attempt one owned normal close
                # before the held process/Job are used for bounded cleanup.
                if ($closePostCount -eq 0 -and $null -ne $appIdentity) {
                    $cleanupWindow = [Release24SmokeWin32]::FindWindow([uint32]$process.Id)
                    if ($cleanupWindow.Handle -ne [IntPtr]::Zero) {
                        $closeRequestedAtUtc = [DateTime]::UtcNow.ToString('o')
                        $closePostCount++
                        $closePosted = [Release24SmokeWin32]::PostMessage(
                            $cleanupWindow.Handle, 0x0010, [IntPtr]::Zero, [IntPtr]::Zero)
                        if ($closePosted) { $process.WaitForExit(15000) | Out-Null }
                    }
                }
            }
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
                    $jobDisposed = [Release24SmokeWin32]::CloseHandle($job)
                    if (-not $jobDisposed) { throw 'CloseHandle(Job) failed.' }
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
        WMClosePostCount = $closePostCount
        CloseRequestedAtUtc = $closeRequestedAtUtc
        GracefulCloseTimedOut = $timedOut
        ForcedAppCleanup = $forced
        AppExitCode = $appExitCode
        OwnedAppStillLive = $ownedAppStillLive
        ProcessDisposed = $processDisposed
        JobDisposed = $jobDisposed
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
        PreCloseSessionFreshAfterStart = $preCloseSessionFreshAfterStart
        PreCloseSessionCapturedAtUtc = $preCloseSessionCapturedAtUtc
        PreCloseSessionCopyPath = $sessionBeforeCloseCopy
        PreCloseSessionRecordPath = $sessionBeforeCloseRecord
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
        Scope = 'One hidden exact copied .26 host opens accepted text then PNG. Mandatory fresh pre-close mixed-session capture checks the real main-thread autosave dispatch; normal-close publication is checked separately. No forced paint/input, dirty-draft/minimized-IME, PNG rendering, worker or resource claim.'
    }
    Write-JsonFile $resultPath $result 18
    if ($null -ne $errorText) {
        throw "Hidden host run '$Name' failed: $errorText"
    }
    return $result
}

$runError = $null
try {
    $run1 = Invoke-HiddenHostRun `
        -Name 'run-1-mixed-cli-open' `
        -Kind 'mixed-cli-open' `
        -LaunchPath @($mixedFixture, $imageFixture) `
        -SettlingMilliseconds 5000
} catch {
    $runError = $_.Exception.Message
    $runResultPath = Join-Path $smokeRoot 'run-1-mixed-cli-open/result.json'
    if (Test-Path -LiteralPath $runResultPath -PathType Leaf) {
        $run1 = Read-JsonFile $runResultPath
    }
}
$candidateAfter = @(Get-DirectoryInventory $ctx.candidateDirectory)
$bundleAfter = @(Get-DirectoryInventory $bin)
$workspaceAfter = @(Get-DirectoryInventory $workspace)
$pluginAfter = Get-FileFingerprint $installedPlugin
$sourceAfter = Get-ReadingSourceFacts $ctx.repositoryRoot
Write-JsonFile (Join-Path $smokeRoot 'inputs-after.json') ([ordered]@{
    Candidate = $candidateAfter
    Bundle = $bundleAfter
    Workspace = $workspaceAfter
    Plugin = $pluginAfter
    AcceptedSource = $sourceAfter
}) 18
Assert-InventoriesEqual @($sourceBefore.Files) @($sourceAfter.Files) 'Accepted source during hidden host'
$candidatePreserved = $true
try {
    Assert-InventoryContainsExact $candidateBefore $candidateAfter `
        'Candidate during one hidden host run' $true
    Assert-InventoryContainsExact $bundleBefore $bundleAfter `
        'Copied bundle during one hidden host run' $true
    Assert-InventoryContainsExact $workspaceBefore $workspaceAfter `
        'Fixture workspace during one hidden host run' $true
} catch {
    $candidatePreserved = $false
}
if (-not $candidatePreserved) {
    throw 'Candidate, copied bundle, or fixture workspace changed during hidden host execution.'
}
if (-not (Test-FingerprintEqual $pluginBefore $pluginAfter)) {
    throw 'Installed plugin JSON changed during hidden host execution.'
}

$summary = [ordered]@{
    Schema = 1
    CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
    Status = $(if ($null -eq $runError) { 'PASS' } else { 'FAIL' })
    Error = $runError
    RunCount = 1
    RunOrder = @($run1.Name)
    CopiedBundle = $bin
    SettledAliveForBoundedPeriod = $run1.SessionSettlingCompleted -and $run1.ProcessAliveAfterSettling
    FreshStrictSessionBeforeClose = $run1.PreCloseSessionStrictShapeObserved -and $run1.PreCloseSessionFreshAfterStart
    FreshNormalClosePublication = $run1.FreshSessionPublication
    Run1 = $run1
    CandidatePreserved = $true
    CopiedBundlePreserved = $true
    FixturesPreserved = $true
    InstalledPluginPreserved = $true
    ConfigRoot = $configRoot
    ConfigSessionPath = $sessionPath
    InstalledPluginPath = $installedPlugin
    ConfigFilesAfterRuns = @(Get-DirectoryInventory $configRoot)
    Scope = 'One hidden exact copied host and isolated C: config. After5000ms alive settling, strict ordered text+PNG active1 recovery must exist before one owned WM_CLOSE. Capture bytes/fingerprint/time before close, then require fresh normal-close publication, exit0 and empty Job. No second restore, worker, forced repaint/input, dirty recovery or physical resource claim.'
}
Write-JsonFile (Join-Path $smokeRoot 'result.json') $summary 18
$summary | ConvertTo-Json -Depth 8
if ($null -ne $runError) { throw $runError }
