// Local Windows process ownership only. No keyboard, mouse, paint, or IME automation.
// Compiled by the later root-authorized PowerShell run, never during preparation.
using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Text;

namespace LevNoteLocalAcceptance {
    [StructLayout(LayoutKind.Sequential)]
    public struct JobAccounting {
        public long TotalUserTime, TotalKernelTime, ThisPeriodTotalUserTime, ThisPeriodTotalKernelTime;
        public uint TotalPageFaultCount, TotalProcesses, ActiveProcesses, TotalTerminatedProcesses;
    }
    [StructLayout(LayoutKind.Sequential)]
    internal struct IoCounters {
        public ulong ReadOperationCount, WriteOperationCount, OtherOperationCount;
        public ulong ReadTransferCount, WriteTransferCount, OtherTransferCount;
    }
    [StructLayout(LayoutKind.Sequential)]
    internal struct JobLimits {
        public long PerProcessUserTimeLimit, PerJobUserTimeLimit;
        public uint LimitFlags;
        public UIntPtr MinimumWorkingSetSize, MaximumWorkingSetSize;
        public uint ActiveProcessLimit;
        public UIntPtr Affinity;
        public uint PriorityClass, SchedulingClass;
    }
    [StructLayout(LayoutKind.Sequential)]
    internal struct ExtendedLimits {
        public JobLimits BasicLimitInformation;
        public IoCounters IoInfo;
        public UIntPtr ProcessMemoryLimit, JobMemoryLimit, PeakProcessMemoryUsed, PeakJobMemoryUsed;
    }
    [StructLayout(LayoutKind.Sequential)]
    internal struct SecurityAttributes {
        public uint Length;
        public IntPtr SecurityDescriptor;
        [MarshalAs(UnmanagedType.Bool)] public bool InheritHandle;
    }
    [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
    internal struct StartupInfo {
        public uint cb;
        public string Reserved, Desktop, Title;
        public uint X, Y, XSize, YSize, XCountChars, YCountChars, FillAttribute, Flags;
        public ushort ShowWindow, Reserved2Bytes;
        public IntPtr Reserved2, StdInput, StdOutput, StdError;
    }
    [StructLayout(LayoutKind.Sequential)]
    internal struct StartupInfoEx {
        public StartupInfo StartupInfo;
        public IntPtr AttributeList;
    }
    [StructLayout(LayoutKind.Sequential)]
    internal struct ProcessInformation {
        public IntPtr Process, Thread;
        public uint ProcessId, ThreadId;
    }
    [StructLayout(LayoutKind.Sequential)]
    internal struct FileTime { public uint Low, High; }
    public sealed class WindowInfo {
        public long Handle;
        public uint Owner;
        public string Title;
        public bool Visible;
    }
    public sealed class OwnedHost : IDisposable {
        private IntPtr job, process, thread;
        private const uint WaitTimeout = 258, WaitFailed = 0xffffffff;
        public uint ProcessId { get; private set; }
        public int StartAttempts { get; private set; }
        public bool CreatedSuspended { get; private set; }
        public bool AssignedBeforeResume { get; private set; }
        public bool Resumed { get; private set; }
        public bool ForcedCleanup { get; private set; }
        public bool JobClosed { get; private set; }
        public bool ProcessClosed { get; private set; }
        public int ClosePostAttempts { get; private set; }
        public bool ClosePosted { get; private set; }
        public string ImagePath { get; private set; }
        public DateTime StartUtc { get; private set; }
        public long HeldProcessHandle { get { return process.ToInt64(); } }
        public long HeldJobHandle { get { return job.ToInt64(); } }
        public bool HasProcess { get { return process != IntPtr.Zero; } }

        private delegate bool EnumCallback(IntPtr hwnd, IntPtr data);
        [DllImport("kernel32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
        private static extern IntPtr CreateJobObject(IntPtr attributes, string name);
        [DllImport("kernel32.dll", SetLastError=true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool SetInformationJobObject(IntPtr job, int infoClass, ref ExtendedLimits limits, uint length);
        [DllImport("kernel32.dll", SetLastError=true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool AssignProcessToJobObject(IntPtr job, IntPtr process);
        [DllImport("kernel32.dll", SetLastError=true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool QueryInformationJobObject(IntPtr job, int infoClass, out JobAccounting info, uint length, out uint returned);
        [DllImport("kernel32.dll", SetLastError=true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool IsProcessInJob(IntPtr process, IntPtr job, [MarshalAs(UnmanagedType.Bool)] out bool assigned);
        [DllImport("kernel32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool CreateProcess(string application, StringBuilder command, IntPtr processAttributes,
            IntPtr threadAttributes, [MarshalAs(UnmanagedType.Bool)] bool inherit, uint flags, IntPtr environment,
            string cwd, ref StartupInfoEx startup, out ProcessInformation info);
        [DllImport("kernel32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
        private static extern IntPtr CreateFile(string path, uint access, uint share, ref SecurityAttributes attrs,
            uint disposition, uint flags, IntPtr template);
        [DllImport("kernel32.dll", SetLastError=true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool InitializeProcThreadAttributeList(IntPtr list, int count, uint flags, ref UIntPtr bytes);
        [DllImport("kernel32.dll", SetLastError=true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool UpdateProcThreadAttribute(IntPtr list, uint flags, UIntPtr attribute,
            IntPtr value, UIntPtr size, IntPtr previous, IntPtr returned);
        [DllImport("kernel32.dll")]
        private static extern void DeleteProcThreadAttributeList(IntPtr list);
        [DllImport("kernel32.dll", SetLastError=true)]
        private static extern uint ResumeThread(IntPtr thread);
        [DllImport("kernel32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool QueryFullProcessImageName(IntPtr process, uint flags, StringBuilder path, ref uint size);
        [DllImport("kernel32.dll", SetLastError=true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool GetProcessTimes(IntPtr process, out FileTime creation, out FileTime exit, out FileTime kernel, out FileTime user);
        [DllImport("kernel32.dll", SetLastError=true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool GetExitCodeProcess(IntPtr process, out uint code);
        [DllImport("kernel32.dll", SetLastError=true)]
        private static extern uint WaitForSingleObject(IntPtr handle, uint milliseconds);
        [DllImport("kernel32.dll", SetLastError=true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool TerminateJobObject(IntPtr job, uint code);
        [DllImport("kernel32.dll", SetLastError=true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool TerminateProcess(IntPtr process, uint code);
        [DllImport("kernel32.dll", SetLastError=true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool CloseHandle(IntPtr handle);
        [DllImport("user32.dll", SetLastError=true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool EnumWindows(EnumCallback callback, IntPtr data);
        [DllImport("user32.dll")]
        private static extern uint GetWindowThreadProcessId(IntPtr hwnd, out uint owner);
        [DllImport("user32.dll", CharSet=CharSet.Unicode)]
        private static extern int GetWindowText(IntPtr hwnd, StringBuilder text, int capacity);
        [DllImport("user32.dll")]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool IsWindowVisible(IntPtr hwnd);
        [DllImport("user32.dll")]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool IsWindow(IntPtr hwnd);
        [DllImport("user32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool PostMessage(IntPtr hwnd, uint message, IntPtr w, IntPtr l);

        private static Win32Exception LastError(string label) { return new Win32Exception(Marshal.GetLastWin32Error(), label); }
        private static void CloseChecked(ref IntPtr handle) {
            if (handle == IntPtr.Zero || handle == new IntPtr(-1)) return;
            if (!CloseHandle(handle)) throw LastError("CloseHandle");
            handle = IntPtr.Zero;
        }
        private static IntPtr OpenCapture(string path, bool input, ref SecurityAttributes attrs) {
            IntPtr result = CreateFile(path, input ? 0x80000000u : 0x40000000u,
                input ? 3u : 1u, ref attrs, input ? 3u : 1u, 0x80, IntPtr.Zero);
            if (result == new IntPtr(-1)) throw LastError("CreateFile capture");
            return result;
        }
        public static string QuoteArgument(string argument) {
            var result = new StringBuilder("\"");
            int slashes = 0;
            foreach (char c in argument) {
                if (c == '\\') { slashes++; continue; }
                if (c == '"') { result.Append('\\', slashes * 2 + 1); result.Append(c); slashes = 0; continue; }
                result.Append('\\', slashes); slashes = 0; result.Append(c);
            }
            result.Append('\\', slashes * 2); result.Append('"');
            return result.ToString();
        }
        private static IntPtr EnvironmentBlock(string[] environment) {
            var rows = new List<string>(environment);
            rows.Sort(StringComparer.OrdinalIgnoreCase);
            return Marshal.StringToHGlobalUni(string.Join("\0", rows) + "\0\0");
        }
        public void Start(string exe, string[] args, string cwd, string[] environment, string stdin, string stdout, string stderr) {
            if (StartAttempts != 0) throw new InvalidOperationException("Single host attempt only.");
            StartAttempts++;
            job = CreateJobObject(IntPtr.Zero, null); // No SECURITY_ATTRIBUTES: Job handle is non-inheritable.
            if (job == IntPtr.Zero) throw LastError("CreateJobObject");
            var limits = new ExtendedLimits();
            limits.BasicLimitInformation.LimitFlags = 0x2000; // JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE; no breakaway flags.
            if (!SetInformationJobObject(job, 9, ref limits, (uint)Marshal.SizeOf<ExtendedLimits>())) throw LastError("SetInformationJobObject");
            IntPtr input = IntPtr.Zero, output = IntPtr.Zero, error = IntPtr.Zero;
            IntPtr attributes = IntPtr.Zero, handles = IntPtr.Zero, env = IntPtr.Zero;
            bool attributesReady = false;
            Exception startupFailure = null;
            var closeFailures = new List<Exception>();
            try {
                var security = new SecurityAttributes { Length=(uint)Marshal.SizeOf<SecurityAttributes>(), InheritHandle=true };
                input = OpenCapture(stdin, true, ref security);
                output = OpenCapture(stdout, false, ref security);
                error = OpenCapture(stderr, false, ref security);
                UIntPtr bytes = UIntPtr.Zero;
                InitializeProcThreadAttributeList(IntPtr.Zero, 1, 0, ref bytes);
                if (bytes == UIntPtr.Zero) throw LastError("Attribute list size");
                attributes = Marshal.AllocHGlobal(checked((int)bytes.ToUInt64()));
                if (!InitializeProcThreadAttributeList(attributes, 1, 0, ref bytes)) throw LastError("InitializeProcThreadAttributeList");
                attributesReady = true;
                handles = Marshal.AllocHGlobal(IntPtr.Size * 3);
                Marshal.WriteIntPtr(handles, 0, input); Marshal.WriteIntPtr(handles, IntPtr.Size, output); Marshal.WriteIntPtr(handles, IntPtr.Size * 2, error);
                // Only the three stdio handles inherit. Never the Job/process/thread handles.
                if (!UpdateProcThreadAttribute(attributes, 0, new UIntPtr(0x20002), handles, new UIntPtr((uint)(IntPtr.Size * 3)), IntPtr.Zero, IntPtr.Zero)) throw LastError("Handle list");
                env = EnvironmentBlock(environment);
                var startup = new StartupInfoEx();
                startup.StartupInfo.cb = (uint)Marshal.SizeOf<StartupInfoEx>();
                startup.StartupInfo.Flags = 0x101; // STARTF_USESHOWWINDOW | STARTF_USESTDHANDLES.
                startup.StartupInfo.ShowWindow = 0; // SW_HIDE: observable expectation, not an input intervention.
                startup.StartupInfo.StdInput=input; startup.StartupInfo.StdOutput=output; startup.StartupInfo.StdError=error;
                startup.AttributeList=attributes;
                var command = new StringBuilder(QuoteArgument(exe));
                foreach (string arg in args) { command.Append(' '); command.Append(QuoteArgument(arg)); }
                ProcessInformation info;
                if (!CreateProcess(exe, command, IntPtr.Zero, IntPtr.Zero, true,
                    0x00080000u | 0x00000400u | 0x00000004u, env, cwd, ref startup, out info)) throw LastError("CreateProcessW suspended");
                process=info.Process; thread=info.Thread; ProcessId=info.ProcessId; CreatedSuspended=true;
                if (!AssignProcessToJobObject(job, process)) throw LastError("AssignProcessToJobObject before resume");
                bool assigned;
                if (!IsProcessInJob(process, job, out assigned) || !assigned) throw LastError("Verify owned Job membership");
                AssignedBeforeResume=true;
                ImagePath=ReadImagePath();
                if (!StringComparer.OrdinalIgnoreCase.Equals(ImagePath, exe)) throw new InvalidOperationException("Held process image path mismatch.");
                FileTime creation, exit, kernel, user;
                if (!GetProcessTimes(process, out creation, out exit, out kernel, out user)) throw LastError("GetProcessTimes");
                StartUtc=DateTime.FromFileTimeUtc(unchecked((long)(((ulong)creation.High << 32) | creation.Low)));
                var accounting=Accounting();
                if (accounting.TotalProcesses != 1 || accounting.ActiveProcesses != 1) throw new InvalidOperationException("Owned Job must contain exactly suspended host.");
                uint count=ResumeThread(thread);
                if (count == 0xffffffff) throw LastError("ResumeThread");
                if (count != 1) throw new InvalidOperationException("Unexpected primary-thread suspension count.");
                Resumed=true;
                CloseChecked(ref thread);
            } catch (Exception e) { startupFailure = e; }
            finally {
                if (attributesReady) DeleteProcThreadAttributeList(attributes);
                if (attributes != IntPtr.Zero) Marshal.FreeHGlobal(attributes);
                if (handles != IntPtr.Zero) Marshal.FreeHGlobal(handles);
                if (env != IntPtr.Zero) Marshal.FreeHGlobal(env);
                try { CloseChecked(ref input); } catch (Exception e) { closeFailures.Add(e); }
                try { CloseChecked(ref output); } catch (Exception e) { closeFailures.Add(e); }
                try { CloseChecked(ref error); } catch (Exception e) { closeFailures.Add(e); }
            }
            if (startupFailure != null) closeFailures.Insert(0, startupFailure);
            if (closeFailures.Count != 0) throw new AggregateException("Suspended launch/std-handle cleanup failed", closeFailures);
        }
        public string ReadImagePath() {
            var path=new StringBuilder(32768); uint size=(uint)path.Capacity;
            if (!QueryFullProcessImageName(process, 0, path, ref size)) throw LastError("QueryFullProcessImageName held handle");
            return path.ToString();
        }
        public bool IsAlive() {
            if (process == IntPtr.Zero) return false;
            uint wait=WaitForSingleObject(process, 0);
            if (wait == WaitFailed) throw LastError("WaitForSingleObject alive");
            return wait == WaitTimeout;
        }
        public bool WaitExit(int milliseconds) {
            uint wait=WaitForSingleObject(process, checked((uint)milliseconds));
            if (wait == WaitFailed) throw LastError("WaitForSingleObject exit");
            return wait == 0;
        }
        public uint ExitCode() {
            if (IsAlive()) throw new InvalidOperationException("Host is still alive.");
            uint code;
            if (!GetExitCodeProcess(process, out code)) throw LastError("GetExitCodeProcess");
            return code;
        }
        public JobAccounting Accounting() {
            JobAccounting result; uint returned;
            if (!QueryInformationJobObject(job, 1, out result, (uint)Marshal.SizeOf<JobAccounting>(), out returned)) throw LastError("Query Job accounting");
            return result;
        }
        public WindowInfo[] Windows(string title) {
            if (!IsAlive()) throw new InvalidOperationException("Cannot enumerate window of exited host.");
            var rows=new List<WindowInfo>();
            if (!EnumWindows(delegate(IntPtr hwnd, IntPtr unused) {
                uint owner; GetWindowThreadProcessId(hwnd, out owner);
                if (owner != ProcessId) return true;
                var text=new StringBuilder(2048); GetWindowText(hwnd, text, text.Capacity);
                if (text.ToString() == title) rows.Add(new WindowInfo { Handle=hwnd.ToInt64(), Owner=owner, Title=text.ToString(), Visible=IsWindowVisible(hwnd) });
                return true;
            }, IntPtr.Zero)) throw LastError("EnumWindows");
            return rows.ToArray();
        }
        public void PostOwnedClose(long window, string title) {
            if (ClosePostAttempts != 0) throw new InvalidOperationException("WM_CLOSE attempt already consumed.");
            if (!IsAlive() || !StringComparer.OrdinalIgnoreCase.Equals(ReadImagePath(), ImagePath)) throw new InvalidOperationException("Host identity not live.");
            var hwnd=new IntPtr(window); uint owner;
            GetWindowThreadProcessId(hwnd, out owner);
            var text=new StringBuilder(2048); GetWindowText(hwnd, text, text.Capacity);
            if (!IsWindow(hwnd) || owner != ProcessId || text.ToString() != title || IsWindowVisible(hwnd)) throw new InvalidOperationException("Window is not exact hidden owned target.");
            ClosePostAttempts++;
            ClosePosted=PostMessage(hwnd, 0x10, IntPtr.Zero, IntPtr.Zero);
            if (!ClosePosted) throw LastError("Single owned PostMessage WM_CLOSE");
        }
        public void ForceCleanup(int milliseconds) {
            if (!HasProcess) return;
            if (IsAlive() || (AssignedBeforeResume && Accounting().ActiveProcesses != 0)) {
                ForcedCleanup=true;
                // Assignment failures stay suspended and terminate by exact held handle.
                if (AssignedBeforeResume) {
                    if (!TerminateJobObject(job, 0xe001)) throw LastError("Terminate owned Job");
                } else if (!TerminateProcess(process, 0xe001)) throw LastError("Terminate exact suspended process");
                if (!WaitExit(milliseconds)) throw new InvalidOperationException("Owned cleanup exit bound exceeded.");
            }
        }
        public void Dispose() {
            var failures=new List<Exception>();
            // Close Job before process handle: kill-on-close still acts if forced cleanup failed.
            try { CloseChecked(ref job); JobClosed=true; } catch (Exception e) { failures.Add(e); }
            try { CloseChecked(ref thread); } catch (Exception e) { failures.Add(e); }
            try { CloseChecked(ref process); ProcessClosed=true; } catch (Exception e) { failures.Add(e); }
            if (failures.Count != 0) throw new AggregateException("Owned handle disposal failed", failures);
        }
    }
}
