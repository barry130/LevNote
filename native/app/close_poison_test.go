package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineruntime"
)

// causes holds a []error. Even though the struct type is comparable, an
// interface value containing this particular struct cannot be compared safely.
type closePoisonValueError struct {
	message string
	causes  any
}

func (e closePoisonValueError) Error() string   { return e.message }
func (e closePoisonValueError) Unwrap() []error { return e.causes.([]error) }

// Inject a package cleanup error at the real terminal result slot. The test
// worker cannot publish that result until the close cancels the package work.
func installClosePoisonPackageFault(t *testing.T, a *App, failure error) *packageJob {
	t.Helper()
	if a.packageJob != nil {
		t.Fatal("previous package job was not consumed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	job := &packageJob{manager: a.packageManager, action: packageImportAction, cancel: cancel, done: make(chan struct{})}
	a.packageJob = job
	go func() {
		<-ctx.Done()
		job.mu.Lock()
		job.cleanupErr, job.ready = failure, true
		close(job.done)
		job.mu.Unlock()
	}()
	return job
}

type closePoisonSize struct{ nodes, textBytes, issueLineBytes int }

func measureClosePoison(t *testing.T, a *App) closePoisonSize {
	t.Helper()
	if a.enginePoison == nil {
		t.Fatal("close did not retain cleanup errors")
	}
	// Visit each edge, even if two branches point at the same node. Error
	// rendering and an unsuccessful errors.Is search pay for both branches.
	stack := []error{a.enginePoison}
	var size closePoisonSize
	for len(stack) != 0 {
		last := len(stack) - 1
		err := stack[last]
		stack = stack[:last]
		if err == nil {
			continue
		}
		size.nodes++
		if size.nodes > 32768 {
			t.Fatal("close error graph exceeded the test traversal budget")
		}
		switch err := err.(type) {
		case interface{ Unwrap() []error }:
			stack = append(stack, err.Unwrap()...)
		case interface{ Unwrap() error }:
			stack = append(stack, err.Unwrap())
		}
	}
	size.textBytes = len(a.enginePoison.Error())
	for i := len(a.Issues) - 1; i >= 0; i-- {
		if strings.HasPrefix(a.Issues[i], "关闭未完成：导出进程清理失败") {
			size.issueLineBytes = len(a.Issues[i])
			return size
		}
	}
	t.Fatal("close failure was missing from Issues")
	return size
}

func logClosePoison(t *testing.T, attempt int, size closePoisonSize, a *App) {
	t.Helper()
	issueBytes := 0
	for _, issue := range a.Issues {
		issueBytes += len(issue)
	}
	t.Logf("poison sample attempts=%d nodes=%d textBytes=%d issueLineBytes=%d issues=%d issueBytes=%d",
		attempt, size.nodes, size.textBytes, size.issueLineBytes, len(a.Issues), issueBytes)
}

func TestClosePoisonRetryKeepsBoundedCleanupAndRecovery(t *testing.T) {
	a, tab, runner, queue, _ := workerApp(t)
	tab.Editor.ReplaceText(nil, "live dirty draft🙂")
	a.Persist()
	filename := filepath.Join(a.configDir, "session.json")
	before := mustReadFile(t, filename)
	liveText, history, selection := tab.Doc.Buffer.String(), tab.Doc.Buffer.History(), tab.Doc.Buffer.Selection()
	a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }

	a.exportCurrent()
	export, lane := a.exportJob, a.engineLane
	call := awaitExportCall(t, runner)
	initialPackage := errors.New("initial package cleanup refused")
	packageFault := &os.PathError{Op: "join", Path: "package", Err: initialPackage}
	packageJob := installClosePoisonPackageFault(t, a, packageFault)
	runnerRoot := errors.New("runner handle refused cleanup")
	const sharedMessage = "owned cleanup refused"
	runner.closeErr = closePoisonValueError{message: sharedMessage, causes: []error{runnerRoot}}
	runnerClosed := false
	runner.onClose = func() {
		runnerClosed = true
		select {
		case <-packageJob.done:
		default:
			t.Error("runner closed before package work was joined")
		}
	}

	refuseClose := func() closePoisonSize {
		t.Helper()
		if a.CanClose() {
			t.Fatal("cleanup poison accepted close")
		}
		return measureClosePoison(t, a)
	}
	stable := refuseClose()
	logClosePoison(t, 1, stable, a)
	if !runnerClosed || call.ctx.Err() == nil || a.exportJob != nil || a.packageJob != nil || a.engineLane != lane || a.exporter != runner {
		t.Fatal("close failed to cancel and join its work or changed cleanup ownership")
	}
	select {
	case <-lane.done:
	default:
		t.Fatal("close did not join the lane loop")
	}
	awaitExportDone(t, export)
	drainOneExportCallback(t, queue)
	if export.result.Payload != nil || export.err != nil || !strings.Contains(a.exportStatuses[export.capability.ID].outcome, sharedMessage) {
		t.Fatal("close did not clear the export slot and expose its cleanup status")
	}
	var gotPackage *os.PathError
	var gotRunner closePoisonValueError
	if !errors.Is(a.enginePoison, initialPackage) || !errors.Is(a.enginePoison, runnerRoot) ||
		!errors.As(a.enginePoison, &gotPackage) || gotPackage != packageFault ||
		!errors.As(a.enginePoison, &gotRunner) || !errors.Is(gotRunner, runnerRoot) {
		t.Fatal("first package and runner causes did not survive Is/As")
	}
	if got := a.enginePoison.Error(); !strings.Contains(got, packageFault.Error()) || !strings.Contains(got, sharedMessage) {
		t.Fatal("first close hid one of the cleanup causes")
	}
	requireOldCloseRecovery(t, a, filename, before)

	// The old two-branch feedback is measured for twelve closes. Refuse to
	// enter the longer loop if it grew, keeping the failing case cheap.
	const redAttempts = 12
	var sample closePoisonSize
	var firstChange int
	for attempt := 2; attempt <= redAttempts; attempt++ {
		sample = refuseClose()
		if sample != stable && firstChange == 0 {
			firstChange = attempt
		}
	}
	logClosePoison(t, redAttempts, sample, a)
	if firstChange != 0 {
		t.Fatalf("retry changed a fixed set of causes at attempt %d: first=%+v after %d=%+v", firstChange, stable, redAttempts, sample)
	}

	// This independently completed package error has the same visible text
	// and same noncomparable type as the cached runner error.
	lateRoot := errors.New(runnerRoot.Error())
	installClosePoisonPackageFault(t, a, closePoisonValueError{message: sharedMessage, causes: []error{lateRoot}})
	withLate := refuseClose()
	if withLate.nodes <= stable.nodes || !errors.Is(a.enginePoison, lateRoot) ||
		!errors.Is(a.enginePoison, initialPackage) || !errors.Is(a.enginePoison, runnerRoot) {
		t.Fatal("joining a later package fault did not retain each independent cause")
	}
	logClosePoison(t, redAttempts+1, withLate, a)
	const attempts = issueHistoryLimit + 64
	for attempt := redAttempts + 2; attempt <= attempts; attempt++ {
		if got := refuseClose(); got != withLate {
			t.Fatalf("retry %d changed cleanup graph or diagnostics: got=%+v want=%+v", attempt, got, withLate)
		}
	}
	logClosePoison(t, attempts, withLate, a)
	issueBytes := 0
	for _, issue := range a.Issues {
		issueBytes += len(issue)
	}
	if len(a.Issues) != issueHistoryLimit || a.IssuesDropped == 0 || issueBytes > issueHistoryLimit*withLate.issueLineBytes {
		t.Fatal("closing repeatedly did not settle at the issue history bound")
	}
	if a.Current() != tab || tab.Doc.Buffer.String() != liveText || tab.Doc.Buffer.History() != history || tab.Doc.Buffer.Selection() != selection || a.persistPending {
		t.Fatal("failed close changed the live tab or rearmed persistence")
	}
	requireOldCloseRecovery(t, a, filename, before)
	requireNoCloseTemps(t, filename)

	// Delivery of a completed package fault also keeps its cause when UI
	// ownership has changed and the ordinary result will be discarded.
	deliveredRoot := errors.New("delivered package cleanup refused")
	job := installClosePoisonPackageFault(t, a, deliveredRoot)
	job.cancel()
	<-job.done
	job.generation = a.packageGeneration + 1
	a.deliverPackageWork()
	for _, root := range []error{initialPackage, runnerRoot, lateRoot, deliveredRoot} {
		if !errors.Is(a.enginePoison, root) {
			t.Fatalf("close retries and stale package delivery lost cause %v", root)
		}
	}
	if a.packageJob != nil {
		t.Fatal("delivery did not consume the stale package job")
	}

	a.exportCurrent()
	if a.exportJob != nil || len(runner.calls) != 0 || !strings.Contains(a.Message, sharedMessage) {
		t.Fatal("poisoned engine admitted a new export")
	}
	requireClosePoisonPackageAdmission(t, a)
}

func requireClosePoisonPackageAdmission(t *testing.T, a *App) {
	t.Helper()
	if a.packageManager == nil {
		// On platforms without the store, exercise App admission with an
		// uninitialized manager. The store itself still rejects the request.
		a.packageManager, a.packageInitErr = new(engineruntime.PackageManager), nil
	}
	engine := enginecatalog.MarkdownHTML
	for _, action := range []packageAction{packageActivateAction, packageResetAction, packageRepairAction} {
		a.startPackageWork(action, engine, "")
		if a.packageJob != nil || !strings.Contains(a.Message, "无法切换引擎") {
			t.Fatalf("poisoned transition %s was admitted", action)
		}
	}
	for _, action := range []packageAction{packageListAction, packageImportAction} {
		a.startPackageWork(action, engine, t.TempDir())
		job := a.packageJob
		if job == nil || job.action != action {
			t.Fatalf("poison blocked independent package operation %s", action)
		}
		<-job.done
		a.deliverPackageWork()
		if a.packageJob != nil || a.enginePoison == nil {
			t.Fatalf("package operation %s did not finish while retaining poison", action)
		}
	}
}

func TestClosePoisonAcceptsDistinctRunnerFailuresWithEqualText(t *testing.T) {
	a, _, runner, filename, before := closePlanFixture(t)
	first := errors.New("runner cleanup failed")
	second := errors.New(first.Error())
	wrapped := &os.PathError{Op: "close", Path: "runner", Err: first}
	attempt := 0
	closeWith := func(err error) closePoisonSize {
		t.Helper()
		attempt++
		runner.closeErr = err
		if a.CanClose() || a.engineLane != nil {
			t.Fatal("direct runner cleanup unexpectedly changed close outcome or created a lane")
		}
		return measureClosePoison(t, a)
	}
	initial := closeWith(first)
	added := closeWith(second)
	if added.nodes <= initial.nodes || !errors.Is(a.enginePoison, first) || !errors.Is(a.enginePoison, second) {
		t.Fatal("the later runner error was conflated with a different same-message error")
	}
	complete := closeWith(wrapped)
	var typed *os.PathError
	if !errors.As(a.enginePoison, &typed) || typed != wrapped {
		t.Fatal("a distinct typed error wrapping a known root was lost")
	}
	for range 6 {
		for _, err := range []error{first, second, wrapped, nil} {
			if got := closeWith(err); got != complete {
				t.Fatalf("re-observation changed runner causes: got=%+v want=%+v", got, complete)
			}
		}
	}
	logClosePoison(t, attempt, complete, a)
	requireOldCloseRecovery(t, a, filename, before)
	requireNoCloseTemps(t, filename)
}
