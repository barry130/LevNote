//go:build windows

package engineruntime

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	createJobObject     = kernel32.NewProc("CreateJobObjectW")
	setJobInformation   = kernel32.NewProc("SetInformationJobObject")
	assignProcessJob    = kernel32.NewProc("AssignProcessToJobObject")
	terminateJobObject  = kernel32.NewProc("TerminateJobObject")
	queryJobInformation = kernel32.NewProc("QueryInformationJobObject")
	isProcessInJob      = kernel32.NewProc("IsProcessInJob")
)

type jobBasicLimits struct {
	PerProcessUserTime, PerJobUserTime int64
	Flags                              uint32
	MinWorkingSet, MaxWorkingSet       uintptr
	ActiveProcessLimit                 uint32
	Affinity                           uintptr
	PriorityClass, SchedulingClass     uint32
}
type jobExtendedLimits struct {
	Basic                                                      jobBasicLimits
	IOCounters                                                 [6]uint64
	ProcessMemory, JobMemory, PeakProcessMemory, PeakJobMemory uintptr
}
type jobAccounting struct {
	TotalUserTime, TotalKernelTime, ThisPeriodUserTime, ThisPeriodKernelTime int64
	PageFaults, TotalProcesses, ActiveProcesses, TerminatedProcesses         uint32
}

type processJob struct {
	mu     sync.Mutex
	handle syscall.Handle
	// Held process objects establish a stronger cleanup barrier than the job's
	// active count, which can reach zero before those objects become signaled.
	members    map[uint32]syscall.Handle
	captureErr error
}

const maxOwnedJobProcesses = 256

type jobProcessIDs struct {
	Assigned uint32
	Count    uint32
	IDs      [maxOwnedJobProcesses]uintptr
}

func newProcessJob() (*processJob, error) {
	h, _, err := createJobObject.Call(0, 0)
	if h == 0 {
		return nil, err
	}
	limits := jobExtendedLimits{}
	limits.Basic.Flags = 0x2000 // JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE; no breakaway.
	ok, _, setErr := setJobInformation.Call(h, 9, uintptr(unsafe.Pointer(&limits)), unsafe.Sizeof(limits))
	if ok == 0 {
		_ = syscall.CloseHandle(syscall.Handle(h))
		return nil, setErr
	}
	return &processJob{handle: syscall.Handle(h)}, nil
}

func (j *processJob) attach(p *os.Process) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.handle == 0 {
		return errors.New("job already closed")
	}
	var attachErr error
	err := p.WithHandle(func(h uintptr) {
		ok, _, callErr := assignProcessJob.Call(uintptr(j.handle), h)
		if ok == 0 {
			attachErr = callErr
		}
	})
	return errors.Join(err, attachErr)
}

func (j *processJob) terminate() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.handle == 0 {
		return nil
	}
	if err := j.captureMembersLocked(); err != nil {
		j.captureErr = errors.Join(j.captureErr, err)
	}
	ok, _, err := terminateJobObject.Call(uintptr(j.handle), 1)
	if ok == 0 {
		return errors.Join(j.captureErr, err)
	}
	return j.captureErr
}

func (j *processJob) captureMembersLocked() error {
	var ids jobProcessIDs
	ok, _, queryErr := queryJobInformation.Call(uintptr(j.handle), 3, uintptr(unsafe.Pointer(&ids)), unsafe.Sizeof(ids), 0)
	if ok == 0 {
		return errors.Join(errors.New("cannot enumerate bounded owned job members"), queryErr)
	}
	if ids.Assigned > maxOwnedJobProcesses || ids.Count > maxOwnedJobProcesses || ids.Count != ids.Assigned {
		return errors.New("owned job member count exceeds cleanup limit or changed during enumeration")
	}
	if j.members == nil {
		j.members = make(map[uint32]syscall.Handle)
	}
	for _, rawPID := range ids.IDs[:ids.Count] {
		if rawPID == 0 || rawPID > uintptr(^uint32(0)) {
			return errors.New("invalid owned job member identity")
		}
		pid := uint32(rawPID)
		if _, held := j.members[pid]; held {
			continue
		}
		if len(j.members) >= maxOwnedJobProcesses {
			return errors.New("owned process cleanup handle limit exceeded")
		}
		h, err := syscall.OpenProcess(syscall.SYNCHRONIZE|0x1000, false, pid) // QUERY_LIMITED_INFORMATION
		if err == syscall.Errno(87) {                                         // It already exited and the process object no longer exists.
			continue
		}
		if err != nil {
			return err
		}
		var member uint32
		ok, _, memberErr := isProcessInJob.Call(uintptr(h), uintptr(j.handle), uintptr(unsafe.Pointer(&member)))
		if ok == 0 || member == 0 {
			_ = syscall.CloseHandle(h)
			return errors.Join(errors.New("enumerated process no longer belongs to owned job"), memberErr)
		}
		j.members[pid] = h
	}
	return nil
}

func (j *processJob) closeAndWait() error {
	// Root process Wait has completed before this method. Terminate any owned
	// descendants, then observe the Job becoming empty before closing its handle.
	var errs []error
	if err := j.terminate(); err != nil {
		errs = append(errs, err)
	}
	deadline := time.Now().Add(pipeWaitDelay)
	for {
		j.mu.Lock()
		var counts jobAccounting
		ok, _, err := queryJobInformation.Call(uintptr(j.handle), 1, uintptr(unsafe.Pointer(&counts)), unsafe.Sizeof(counts), 0)
		j.mu.Unlock()
		if ok == 0 {
			errs = append(errs, err)
			break
		}
		if counts.ActiveProcesses == 0 {
			break
		}
		if time.Now().After(deadline) {
			errs = append(errs, errors.New("job still reports active processes after termination"))
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Enumeration is bounded and occurs before termination, including cancellation.
	// Never infer process-object completion from a later lookup of a recycled PID.
	j.mu.Lock()
	for _, h := range j.members {
		remaining := time.Until(deadline)
		milliseconds := uint32(0)
		if remaining > 0 {
			milliseconds = uint32((remaining + time.Millisecond - 1) / time.Millisecond)
		}
		state, err := syscall.WaitForSingleObject(h, milliseconds)
		if err != nil || state != syscall.WAIT_OBJECT_0 {
			errs = append(errs, errors.Join(errors.New("owned process did not signal termination within cleanup deadline"), err))
		}
		if err := syscall.CloseHandle(h); err != nil {
			errs = append(errs, err)
		}
	}
	j.members = nil
	j.mu.Unlock()
	j.mu.Lock()
	if j.handle != 0 {
		if err := syscall.CloseHandle(j.handle); err != nil {
			errs = append(errs, err)
		}
		j.handle = 0
	}
	j.mu.Unlock()
	return errors.Join(errs...)
}

func configureCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	// Go-only worker needs no PATH, workspace config, profiling or plugin env.
	cmd.Env = []string{}
	if root := os.Getenv("SystemRoot"); root != "" {
		cmd.Env = append(cmd.Env, "SystemRoot="+root, "WINDIR="+root)
	}
}
