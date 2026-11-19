package engineruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"levnote.local/native/core/engineipc"
)

const (
	requestTimeout = 30 * time.Second
	pipeWaitDelay  = 2 * time.Second
	maxStderrBytes = 16 << 10
)

type operation struct {
	cancel context.CancelCauseFunc
}

// Runner admits one operation. Its zero value is closed/unavailable; callers
// should use NewBundled or NewManaged, neither of which hashes engine files.
type Runner struct {
	mu        sync.Mutex
	hostDir   string
	manager   *PackageManager
	active    *operation
	closed    bool
	closeDone chan struct{}
	closeErr  error
	timeout   time.Duration
	waitDelay time.Duration
	hooks     *testHooks
}

// Private seams never accept public executable/configuration input.
type testHooks struct {
	afterAdmission func()
	afterVerify    func(*verifiedBundle)
	afterStart     func(*os.Process)
	beforeAttach   func() error
	afterAttach    func(*processJob)
	afterWait      func()
	observeStdout  func([]byte)
	workerArgs     []string
}

func NewBundled() (*Runner, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, failure(FailureUnavailable, "无法定位应用程序目录", err)
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return nil, failure(FailureUnavailable, "无法定位应用程序目录", err)
	}
	return newRunner(filepath.Dir(exe)), nil
}

// NewManaged binds the already initialized local store, not an executable path.
// A corrupt selection state is intentionally retained for explicit repair; each
// request refuses it until repaired. Construction performs no filesystem I/O.
func NewManaged(manager *PackageManager) (*Runner, error) {
	if manager == nil || manager.gate == nil || manager.root == "" || manager.hostDir == "" {
		return nil, failure(FailureUnavailable, "引擎包仓库未初始化", nil)
	}
	r := newRunner(manager.hostDir)
	r.manager = manager
	return r, nil
}

func newRunner(hostDir string) *Runner {
	return &Runner{hostDir: hostDir, closeDone: make(chan struct{}), timeout: requestTimeout, waitDelay: pipeWaitDelay}
}

func (r *Runner) Export(ctx context.Context, id engineipc.EngineID, src Source) (Result, error) {
	if r == nil || ctx == nil {
		return Result{}, failure(FailureUnavailable, "导出运行器不可用", nil)
	}
	// Admission owns the page value before any hook, package lookup, process
	// creation or source write. The caller may mutate its original pointer once
	// Export has begun without changing this operation.
	if src.Page != nil {
		page := *src.Page
		src.Page = &page
	}
	r.mu.Lock()
	if r.closed || r.closeDone == nil {
		r.mu.Unlock()
		return Result{}, failure(FailureClosed, "导出运行器已关闭", nil)
	}
	if r.active != nil {
		r.mu.Unlock()
		return Result{}, failure(FailureBusy, "已有导出正在完成或清理", nil)
	}
	descriptor, admissionErr := admittedEngine(id)
	if admissionErr != nil {
		r.mu.Unlock()
		return Result{}, admissionErr
	}
	if _, validationErr := engineipc.EncodeRequestPrefix(engineipc.RequestMeta{
		Engine: id, Format: descriptor.Format, Page: src.Page,
	}, 0); validationErr != nil {
		r.mu.Unlock()
		return Result{}, failure(FailureProtocol, "导出请求选项无效", validationErr)
	}
	selection := selectedPackage{PackageStatus: PackageStatus{Engine: id, Source: PackageBundled}, hostDir: r.hostDir}
	if r.manager != nil {
		var selectionErr error
		selection, selectionErr = r.manager.selection(id)
		if selectionErr != nil {
			r.mu.Unlock()
			return Result{}, selectionErr
		}
	}
	deadlineCtx, stopTimer := context.WithTimeout(ctx, r.timeout)
	runCtx, cancel := context.WithCancelCause(deadlineCtx)
	r.active = &operation{cancel: cancel}
	r.mu.Unlock()
	if r.hooks != nil && r.hooks.afterAdmission != nil {
		r.hooks.afterAdmission()
	}

	result, err, cleanupErr := r.run(runCtx, cancel, id, src, selection)
	// The child, writers and handles are already joined. Preserve the first
	// cancellation/failure cause and never return publishable bytes on failure.
	r.mu.Lock()
	// Completion linearizes with admission/Close under this same mutex. Close
	// accepted before this point must win over an otherwise valid result.
	if canceled := contextFailure(runCtx); canceled != nil {
		err = canceled
	}
	if cleanupErr != nil {
		err = failure(FailureCleanup, "导出进程清理失败", cleanupErr)
		// A failed cleanup cannot be treated as a reusable idle runner. Preserve
		// its outcome for Close even when Close is first called after Export.
		r.closed = true
		r.closeErr = cleanupErr
	}
	if err != nil {
		result.Payload = nil
	}
	cancel(context.Canceled)
	stopTimer()
	r.active = nil
	if r.closed {
		r.closeErr = cleanupErr
		close(r.closeDone)
	}
	r.mu.Unlock()
	return result, err
}

// Close is safe concurrently with Export and other Close calls. Every caller
// waits for the same cleanup barrier; the closed flag alone is not completion.
func (r *Runner) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	if r.closeDone == nil {
		r.closeDone = make(chan struct{})
	}
	if !r.closed {
		r.closed = true
		if r.active != nil {
			r.active.cancel(context.Canceled)
		} else {
			close(r.closeDone)
		}
	}
	done := r.closeDone
	r.mu.Unlock()
	<-done
	r.mu.Lock()
	err := r.closeErr
	r.mu.Unlock()
	return err
}

func contextFailure(ctx context.Context) error {
	cause := context.Cause(ctx)
	if cause == nil {
		return nil
	}
	var known *Failure
	if errors.As(cause, &known) {
		return known
	}
	if errors.Is(cause, context.DeadlineExceeded) {
		return failure(FailureTimeout, "导出超过时间限制", context.DeadlineExceeded)
	}
	return failure(FailureCanceled, "导出已取消", context.Canceled)
}
