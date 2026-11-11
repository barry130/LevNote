package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/session"
)

type admissionView struct {
	flushes  int
	releases int
	onFlush  func() error
}

func (v *admissionView) Draw(*ui.Context) {}
func (v *admissionView) Release() {
	v.releases++
}
func (v *admissionView) Warnings() []string            { return nil }
func (v *admissionView) Export(string) ([]byte, error) { return nil, nil }
func (v *admissionView) Flush() error {
	v.flushes++
	if v.onFlush != nil {
		return v.onFlush()
	}
	return nil
}

func seedAdmissionTabs(a *App, count int) {
	for index := 0; index < count; index++ {
		a.appendTab(document.New(fmt.Sprintf("tab-%d", index)), fmt.Sprintf("tab-%03d.txt", index), "", false, false)
	}
	if len(a.Tabs) > 0 {
		a.Active = 0
	}
}

func nextAdmissionDispatch(t *testing.T, queued <-chan func()) func() {
	t.Helper()
	select {
	case fn := <-queued:
		return fn
	case <-time.After(3 * time.Second):
		t.Fatal("admission test did not receive its next UI phase")
		return nil
	}
}

func waitAdmissionDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("admission open queue did not finish")
	}
}

func TestLiveTabAdmissionBoundaryAndCreationOrigins(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "config")
	a := newTestApp(t, configDir)
	seedAdmissionTabs(a, session.MaxTabs-1)

	templateCalls := 0
	for index := range a.adapters {
		if a.adapters[index].ID != "markdown" {
			continue
		}
		template := a.adapters[index].Template
		a.adapters[index].Template = func() string {
			templateCalls++
			return template()
		}
		break
	}

	a.NewDocument("markdown")
	if len(a.Tabs) != session.MaxTabs || templateCalls != 1 {
		t.Fatalf("99->100 admission failed: tabs=%d templateCalls=%d message=%s", len(a.Tabs), templateCalls, a.Message)
	}
	origin := a.Current()
	owner := &admissionView{}
	origin.View = owner
	nextID := a.nextID
	originSource := origin.Doc.Buffer.String()
	originVersion := origin.Doc.Buffer.Version()
	originHistory := origin.Doc.Buffer.History()
	originSelection := origin.Doc.Buffer.Selection()
	sessionPath := filepath.Join(configDir, "session.json")
	sessionBytes := []byte("{\"preserved\":true}")
	if err := os.WriteFile(sessionPath, sessionBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	a.NewDocument("markdown")
	if len(a.Tabs) != session.MaxTabs || a.nextID != nextID || templateCalls != 1 ||
		owner.flushes != 0 || owner.releases != 0 || origin.Doc.Buffer.String() != originSource ||
		origin.Doc.Buffer.Version() != originVersion || origin.Doc.Buffer.History() != originHistory ||
		origin.Doc.Buffer.Selection() != originSelection || !strings.Contains(a.Message, "标签已达到") {
		t.Fatalf("100->101 template refusal had side effects: tabs=%d nextID=%d templateCalls=%d flush=%d release=%d message=%s",
			len(a.Tabs), a.nextID, templateCalls, owner.flushes, owner.releases, a.Message)
	}

	a.Message = ""
	if a.NewFromPlugin("missing-plugin") || len(a.Tabs) != session.MaxTabs || a.Current() != origin ||
		!strings.Contains(a.Message, "标签已达到") {
		t.Fatalf("plugin creation bypassed capacity: tabs=%d current=%p origin=%p message=%s",
			len(a.Tabs), a.Current(), origin, a.Message)
	}

	a.Message = ""
	if err := a.recoverDraft("conflict.txt", "kept owner draft"); err == nil || !strings.Contains(err.Error(), "标签已达到") {
		t.Fatalf("recovery callback at capacity = %v", err)
	}
	if len(a.Tabs) != session.MaxTabs || a.Current() != origin || origin.Doc.Buffer.String() != originSource ||
		origin.Doc.Buffer.Version() != originVersion || origin.Doc.Buffer.History() != originHistory ||
		origin.Doc.Buffer.Selection() != originSelection || owner.flushes != 0 || owner.releases != 0 || a.Message != "" {
		t.Fatalf("refused recovery changed owner: tabs=%d current=%p source=%q flush=%d release=%d message=%q",
			len(a.Tabs), a.Current(), origin.Doc.Buffer.String(), owner.flushes, owner.releases, a.Message)
	}
	if got, err := os.ReadFile(sessionPath); err != nil || string(got) != string(sessionBytes) {
		t.Fatalf("creation refusals changed previous recovery bytes: %q err=%v", got, err)
	}
}

func TestLiveTabAdmissionOpenManyContinuesAndReusesExistingPath(t *testing.T) {
	root := t.TempDir()
	existingPath := filepath.Join(root, "existing.txt")
	firstNew := filepath.Join(root, "first-new.txt")
	lastNew := filepath.Join(root, "last-new.txt")
	if err := os.WriteFile(existingPath, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(existingPath)
	existing := a.Current()
	seedAdmissionTabs(a, session.MaxTabs-len(a.Tabs))
	a.Active = session.MaxTabs / 2
	a.Issues = nil

	a.OpenMany([]string{firstNew, existingPath, lastNew})
	if len(a.Tabs) != session.MaxTabs || a.Current() != existing || a.Active != 0 {
		t.Fatalf("mixed capped batch lost later duplicate selection: tabs=%d active=%d current=%p existing=%p message=%s",
			len(a.Tabs), a.Active, a.Current(), existing, a.Message)
	}
	issues := strings.Join(a.Issues, "\n")
	if strings.Count(issues, "标签已达到") != 2 {
		t.Fatalf("capped batch did not report both new paths while continuing: %q", issues)
	}
}

func TestLiveTabAdmissionAsyncTextChangeAndLateDuplicate(t *testing.T) {
	for _, mode := range []string{"capacity-change", "late-duplicate"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "late.txt")
			if err := os.WriteFile(path, []byte("late source"), 0o600); err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(root, "config"))
			seedAdmissionTabs(a, session.MaxTabs-1)
			origin := a.Current()
			tracker := &admissionView{}
			origin.View = tracker
			source := origin.Doc.Buffer.String()
			version := origin.Doc.Buffer.Version()
			history := origin.Doc.Buffer.History()
			selection := origin.Doc.Buffer.Selection()
			sessionPath := filepath.Join(root, "config", "session.json")
			sessionBytes := []byte("{\"preserved\":true}")
			if err := os.WriteFile(sessionPath, sessionBytes, 0o600); err != nil {
				t.Fatal(err)
			}
			queued := make(chan func(), 8)
			a.uiDispatch = func(fn func()) { queued <- fn }

			done := a.openRequests([]openRequest{{path: path}})
			nextAdmissionDispatch(t, queued)()
			apply := nextAdmissionDispatch(t, queued)

			var existing *Tab
			switch mode {
			case "capacity-change":
				a.appendTab(document.New("capacity filler"), "capacity.txt", "", false, false)
			case "late-duplicate":
				late, err := document.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				existing = a.appendTab(late, "late.txt", "", false, false)
			}
			apply()
			waitAdmissionDone(t, done)

			if mode == "capacity-change" {
				if len(a.Tabs) != session.MaxTabs || tracker.flushes != 0 || tracker.releases != 0 ||
					a.Current() != origin || origin.Doc.Buffer.String() != source ||
					origin.Doc.Buffer.Version() != version || origin.Doc.Buffer.History() != history ||
					origin.Doc.Buffer.Selection() != selection || !strings.Contains(a.Message, "标签已达到") {
					t.Fatalf("late text capacity refusal adopted or flushed: tabs=%d flush=%d release=%d message=%s",
						len(a.Tabs), tracker.flushes, tracker.releases, a.Message)
				}
				if got, err := os.ReadFile(sessionPath); err != nil || string(got) != string(sessionBytes) {
					t.Fatalf("late text refusal changed previous recovery bytes: %q err=%v", got, err)
				}
				return
			}
			if len(a.Tabs) != session.MaxTabs || a.Current() != existing || existing == nil ||
				existing.Doc.Buffer.String() != "late source" {
				t.Fatalf("late text duplicate was not selected: tabs=%d current=%p existing=%p message=%s",
					len(a.Tabs), a.Current(), existing, a.Message)
			}
		})
	}
}

func TestLiveTabAdmissionPostFlushCreationAndRecoveryOwnership(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	seedAdmissionTabs(a, session.MaxTabs-1)
	origin := a.Current()
	origin.Editor.ReplaceText(nil, "owner authority")
	originSource := origin.Doc.Buffer.String()
	owner := &admissionView{onFlush: func() error {
		return a.recoverDraft("flush-conflict.txt", "kept recovery")
	}}
	origin.View = owner

	if tab := a.add(document.New("must not be adopted"), "outer.txt", ""); tab != nil {
		t.Fatal("post-Flush full capacity still adopted the requested tab")
	}
	if len(a.Tabs) != session.MaxTabs || a.Current() == origin || a.Current().Doc.Buffer.String() != "kept recovery" ||
		origin.Doc.Buffer.String() != originSource || owner.flushes != 1 || owner.releases != 0 ||
		!strings.Contains(a.Message, "标签已达到") {
		t.Fatalf("post-Flush race state: tabs=%d current=%p origin=%p ownerSource=%q recovered=%q flush=%d release=%d message=%s",
			len(a.Tabs), a.Current(), origin, origin.Doc.Buffer.String(), a.Current().Doc.Buffer.String(),
			owner.flushes, owner.releases, a.Message)
	}
}

func TestLiveTabAdmissionRealRecoverCallbackKeepsDraftAtCapacity(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "conflict.md")
	disk := []byte("# 标题\n\n第二块\n")
	if err := os.WriteFile(path, disk, 0o600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	origin := a.Current()
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	beginRealMarkdownComposition(t, origin, tester)
	if err := origin.Doc.Buffer.ReplaceUTF16(2, 4, "外部"); err != nil {
		t.Fatal(err)
	}
	if a.flushTab(origin, "准入测试冲突") {
		t.Fatal("overlapping Markdown draft unexpectedly flushed")
	}
	tester.Frame()

	source := origin.Doc.Buffer.String()
	version := origin.Doc.Buffer.Version()
	history := origin.Doc.Buffer.History()
	selection := origin.Doc.Buffer.Selection()
	view := origin.View
	seedAdmissionTabs(a, session.MaxTabs-len(a.Tabs))

	recoverer, ok := origin.View.(interface{ RecoverDraft() error })
	if !ok {
		t.Fatalf("real Markdown view has no RecoverDraft callback: %T", origin.View)
	}
	if err := recoverer.RecoverDraft(); err == nil || !strings.Contains(err.Error(), "标签已达到") {
		t.Fatalf("capacity-refused real recovery = %v", err)
	}
	if len(a.Tabs) != session.MaxTabs || a.Current() != origin || origin.View != view ||
		origin.Doc.Buffer.String() != source || origin.Doc.Buffer.Version() != version ||
		origin.Doc.Buffer.History() != history || origin.Doc.Buffer.Selection() != selection {
		t.Fatalf("capacity-refused real recovery changed owner state: tabs=%d current=%p origin=%p view=%p source=%q",
			len(a.Tabs), a.Current(), origin, origin.View, origin.Doc.Buffer.String())
	}
	if !tester.HasText("打开未同步副本") {
		t.Fatalf("capacity-refused real recovery did not retain its draft controls: %#v", tester.Texts())
	}
	if err := recoverer.RecoverDraft(); err == nil || !strings.Contains(err.Error(), "标签已达到") {
		t.Fatalf("real recovery did not retain the draft for another refused retry: %v", err)
	}

	a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
	if !a.CloseTab(1) {
		t.Fatalf("close-one recovery setup failed: %s", a.Message)
	}
	if err := recoverer.RecoverDraft(); err != nil {
		t.Fatalf("retried real recovery = %v", err)
	}
	recovered := a.Current()
	if recovered == nil || recovered == origin || recovered.Doc.Buffer.String() != "中文🙂" ||
		len(a.Tabs) != session.MaxTabs || origin.Doc.Buffer.String() != source {
		t.Fatalf("retried real recovery lost the retained draft: current=%#v tabs=%d owner=%q",
			recovered, len(a.Tabs), origin.Doc.Buffer.String())
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != string(disk) {
		t.Fatalf("recovery capacity refusal changed disk: %q err=%v", got, err)
	}
}

func TestLiveTabAdmissionByteFallbackAndClosedReceiptRetry(t *testing.T) {
	root := t.TempDir()
	reopenPath := filepath.Join(root, "reopen.txt")
	bytePath := filepath.Join(root, "legacy.txt")
	refusedPath := filepath.Join(root, "refused.bin")
	for path, data := range map[string][]byte{
		reopenPath:  []byte("disk"),
		bytePath:    {0xc4, 0xe3, 0xba, 0xc3},
		refusedPath: {0x00, 0x01, 0x02, 0x03},
	} {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(reopenPath)
	a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
	if !a.CloseTab(a.Active) || len(a.closedFiles) != 1 {
		t.Fatalf("reopen fixture was not recorded: %v", a.closedFiles)
	}
	entryID := a.closedFiles[0].id

	seedAdmissionTabs(a, session.MaxTabs-1)
	a.OpenMany([]string{bytePath, refusedPath})
	current := a.Current()
	if len(a.Tabs) != session.MaxTabs || current == nil || !current.ByteInspector ||
		!strings.Contains(current.Doc.Buffer.String(), "c4 e3 ba c3") {
		t.Fatalf("99->100 byte fallback failed: tabs=%d current=%#v message=%s", len(a.Tabs), current, a.Message)
	}
	if strings.Count(strings.Join(a.Issues, "\n"), "标签已达到") != 1 {
		t.Fatalf("second byte fallback was not refused explicitly: %q", a.Issues)
	}

	a.Issues = nil
	reopenDone(t, a.reopenClosedFile(entryID))
	a.reapReopen()
	if len(a.Tabs) != session.MaxTabs || len(a.closedFiles) != 1 || a.closedFiles[0].id != entryID {
		t.Fatalf("capacity refusal consumed reopen receipt: tabs=%d closed=%v", len(a.Tabs), a.closedFiles)
	}
	a.Active = 0
	if !a.CloseTab(a.Active) {
		t.Fatalf("close-one setup failed: %s", a.Message)
	}
	reopenDone(t, a.reopenClosedFile(entryID))
	if len(a.Tabs) != session.MaxTabs || len(a.closedFiles) != 0 ||
		a.Current() == nil || a.Current().path() != canonical(reopenPath) {
		t.Fatalf("close-one/retry did not adopt exact receipt: tabs=%d closed=%v current=%#v message=%s",
			len(a.Tabs), a.closedFiles, a.Current(), a.Message)
	}
}

func TestLiveTabAdmissionClosedReceiptReusesOpenPathAtCap(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "already-open.txt")
	if err := os.WriteFile(path, []byte("already open"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	if !a.CloseTab(a.Active) || len(a.closedFiles) != 1 {
		t.Fatalf("reopen fixture was not recorded: %v", a.closedFiles)
	}
	entryID := a.closedFiles[0].id
	a.Open(path)
	existing := a.Current()
	seedAdmissionTabs(a, session.MaxTabs-len(a.Tabs))
	a.Active = session.MaxTabs / 2

	reopenDone(t, a.reopenClosedFile(entryID))
	if len(a.Tabs) != session.MaxTabs || a.Current() != existing || len(a.closedFiles) != 0 {
		t.Fatalf("same-path reuse did not precede cap rejection: tabs=%d current=%p existing=%p closed=%v message=%s",
			len(a.Tabs), a.Current(), existing, a.closedFiles, a.Message)
	}
}

func TestLiveTabAdmissionAsyncBinaryCleanupAndDuplicate(t *testing.T) {
	for _, mode := range []string{"capacity-change", "late-duplicate", "post-flush"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "image.png")
			imageFixture(t, path, "PNG")
			a := newTestApp(t, filepath.Join(root, "config"))
			seedAdmissionTabs(a, session.MaxTabs-1)
			origin := a.Current()
			tracker := &admissionView{}
			if mode == "post-flush" {
				tracker.onFlush = func() error {
					return a.recoverDraft("image-conflict.txt", "kept image recovery")
				}
			}
			origin.View = tracker
			source := origin.Doc.Buffer.String()
			version := origin.Doc.Buffer.Version()
			history := origin.Doc.Buffer.History()
			selection := origin.Doc.Buffer.Selection()
			sessionPath := filepath.Join(root, "config", "session.json")
			sessionBytes := []byte("{\"preserved\":true}")
			if err := os.WriteFile(sessionPath, sessionBytes, 0o600); err != nil {
				t.Fatal(err)
			}
			queued := make(chan func(), 8)
			a.uiDispatch = func(fn func()) { queued <- fn }

			done := a.openRequests([]openRequest{{path: path}})
			nextAdmissionDispatch(t, queued)()
			nextAdmissionDispatch(t, queued)()
			job := a.binaryJob
			if job == nil {
				t.Fatal("binary admission fixture did not reserve its lane")
			}
			apply := nextAdmissionDispatch(t, queued)

			var existing *Tab
			switch mode {
			case "capacity-change":
				a.appendTab(document.New("capacity filler"), "capacity.txt", "", false, false)
			case "late-duplicate":
				file, snapshot, err := document.OpenBinary(context.Background(), path, document.BinaryKindPNG)
				if err != nil {
					t.Fatal(err)
				}
				existing = a.appendBinary(binaryOutput{file: file, snapshot: snapshot}, "image.png", "")
			}
			apply()
			waitAdmissionDone(t, done)

			if len(a.Tabs) != session.MaxTabs || a.binaryJob != nil || a.binarySnapshot != nil {
				t.Fatalf("binary refusal retained state: tabs=%d job=%v snapshot=%v message=%s",
					len(a.Tabs), a.binaryJob, a.binarySnapshot != nil, a.Message)
			}
			if out, ready := job.task.take(); ready || out.file != nil || out.snapshot != nil || out.image.Pixels != nil {
				t.Fatal("refused binary payload remained in its task slot")
			}
			a.binaryLane.mu.Lock()
			reserved := a.binaryLane.reserved
			a.binaryLane.mu.Unlock()
			if reserved {
				t.Fatal("refused binary payload retained its lane reservation")
			}

			switch mode {
			case "capacity-change":
				if tracker.flushes != 0 || tracker.releases != 0 || a.Current() != origin ||
					origin.Doc.Buffer.String() != source || origin.Doc.Buffer.Version() != version ||
					origin.Doc.Buffer.History() != history || origin.Doc.Buffer.Selection() != selection ||
					!strings.Contains(a.Message, "标签已达到") {
					t.Fatalf("capacity-change binary refusal had side effects: flush=%d release=%d message=%s",
						tracker.flushes, tracker.releases, a.Message)
				}
				if got, err := os.ReadFile(sessionPath); err != nil || string(got) != string(sessionBytes) {
					t.Fatalf("late binary refusal changed previous recovery bytes: %q err=%v", got, err)
				}
			case "late-duplicate":
				if existing == nil || a.Current() != existing || !strings.Contains(a.Message, "文件已在标签中打开") {
					t.Fatalf("late binary duplicate was not selected: current=%p existing=%p message=%s",
						a.Current(), existing, a.Message)
				}
			case "post-flush":
				if a.Current() == origin || a.Current().Doc.Buffer.String() != "kept image recovery" ||
					tracker.flushes != 1 || tracker.releases != 0 || !strings.Contains(a.Message, "标签已达到") {
					t.Fatalf("post-Flush binary race state: current=%p origin=%p recovered=%q flush=%d release=%d message=%s",
						a.Current(), origin, a.Current().Doc.Buffer.String(), tracker.flushes, tracker.releases, a.Message)
				}
			}
		})
	}
}

func TestLiveTabAdmissionRestoresExactMaximum(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	binaryPaths := map[int]string{
		33: filepath.Join(root, "first.png"),
		76: filepath.Join(root, "second.png"),
	}
	for _, path := range binaryPaths {
		imageFixture(t, path, "PNG")
	}
	builder := session.NewSnapshotBuilder("")
	for index := 0; index < session.MaxTabs; index++ {
		if path, ok := binaryPaths[index]; ok {
			if err := builder.AddBinaryTab(session.BinaryTab{
				Path: path, Kind: document.BinaryKindPNG, Title: filepath.Base(path), View: "source",
			}); err != nil {
				t.Fatal(err)
			}
			continue
		}
		draft := fmt.Sprintf("restored-%03d", index)
		if err := builder.AddTab(session.Tab{Title: fmt.Sprintf("%03d.txt", index), Draft: &draft, View: "source"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := builder.SetActive(session.MaxTabs - 1); err != nil {
		t.Fatal(err)
	}
	if err := builder.Save(filepath.Join(configDir, "session.json")); err != nil {
		t.Fatal(err)
	}

	a := newTestApp(t, configDir)
	a.Restore()
	if len(a.Tabs) != session.MaxTabs || a.Active != session.MaxTabs-1 ||
		a.Current() == nil || a.Current().Doc == nil ||
		a.Current().Doc.Buffer.String() != fmt.Sprintf("restored-%03d", session.MaxTabs-1) {
		t.Fatalf("maximum restore changed count/order: tabs=%d active=%d current=%#v issues=%v",
			len(a.Tabs), a.Active, a.Current(), a.Issues)
	}
	restoredBinaries := 0
	for index, tab := range a.Tabs {
		if path, ok := binaryPaths[index]; ok {
			restoredBinaries++
			if tab.Binary == nil || tab.Doc != nil || canonical(tab.path()) != canonical(path) {
				t.Fatalf("restored binary tab %d changed authority: %#v", index, tab)
			}
			continue
		}
		if tab.Binary != nil || tab.Doc == nil {
			t.Fatalf("restored text tab %d changed authority: %#v", index, tab)
		}
		if tab.Doc.Buffer.String() != fmt.Sprintf("restored-%03d", index) {
			t.Fatalf("restored tab %d changed: %q", index, tab.Doc.Buffer.String())
		}
	}
	if restoredBinaries != len(binaryPaths) || a.binarySnapshot != nil {
		t.Fatalf("mixed maximum restore published binary payload: decoded=%d want=%d snapshot=%v",
			restoredBinaries, len(binaryPaths), a.binarySnapshot != nil)
	}
}
