package markdownview_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	publicAdapter "levnote.local/native/modules/markdownview"
	"levnote.local/native/ui/editor"
)

const recoverySource = "# 前文🙂\r\n\r\n| A | B |\r\n|---|---|\r\n| 甲 | 乙 |\r\n\r\n尾部\r\n"
const recoveryFieldOne = "编辑单元格 · 数据 · 第 2 行 · 第 1 列"
const recoveryFieldTwo = "编辑单元格 · 数据 · 第 2 行 · 第 2 列"

func recoveryCell(t *testing.T, tester *ui.Tester, name, field string) {
	t.Helper()
	cell, ok := tester.AccessByRole(ui.RoleCell, name)
	if !ok || cell.Disabled || !cell.Focusable || cell.Offscreen {
		t.Fatalf("FIXTURE: accessible cell missing: %q %#v texts=%q", name, cell, tester.Texts())
	}
	x, y := cell.Bounds.X+cell.Bounds.W/2, cell.Bounds.Y+cell.Bounds.H/2
	tester.ClickAt(x, y)
	tester.Frame()
	tester.ClickAt(x, y)
	tester.Frame()
	recoveryField(t, tester, field)
}

func recoveryField(t *testing.T, tester *ui.Tester, name string) {
	t.Helper()
	field, ok := tester.AccessByRole(ui.RoleTextField, name)
	if !ok || !field.Focused || field.Disabled || field.Offscreen {
		t.Fatalf("FIXTURE: editable focused field missing: %q %#v texts=%q", name, field, tester.Texts())
	}
}

func recoveryType(t *testing.T, tester *ui.Tester, field, value string) {
	t.Helper()
	recoveryField(t, tester, field)
	tester.Command("selectAll")
	tester.Type(value)
	tester.Frame()
}

// The callback recovers one old payload, not any editor opened while it runs.
func TestMarkdownRecoveryIdentityReplacementOwner(t *testing.T) {
	buffer := text.New(recoverySource)
	selection := text.Range{Anchor: buffer.LenUTF16(), Head: 2}
	if err := buffer.SetSelection(selection.Anchor, selection.Head); err != nil {
		t.Fatal(err)
	}
	version, history := buffer.Version(), buffer.History()
	var view *publicAdapter.View
	var tester *ui.Tester
	var copies []string
	changes := 0
	view = publicAdapter.Adapter().Create(editor.Host{
		Buffer: buffer, Path: "fixture.md", Changed: func() { changes++ },
		RecoverDraft: func(title, value string) error {
			if title != "fixture.md-块草稿.txt" {
				t.Fatalf("copy title = %q", title)
			}
			copies = append(copies, value)
			if len(copies) == 1 {
				view.DiscardDraft()
				tester.Frame()
				tester.Advance(time.Second)
				recoveryCell(t, tester, "数据，第 2 行，第 2 列：乙", recoveryFieldTwo)
				recoveryType(t, tester, recoveryFieldTwo, "新草稿🙂乙")
			}
			return nil
		},
	}).(*publicAdapter.View)
	tester = ui.NewTester(view.Draw, 760, 640)
	tester.Frame()
	recoveryCell(t, tester, "数据，第 2 行，第 1 列：甲", recoveryFieldOne)
	recoveryType(t, tester, recoveryFieldOne, "旧草稿🙂甲")
	if err := view.RecoverDraft(); err != nil {
		t.Fatal(err)
	}
	if err := view.RecoverDraft(); err != nil {
		t.Fatal(err)
	}
	if len(copies) != 2 || copies[0] != "旧草稿🙂甲" || copies[1] != "新草稿🙂乙" {
		t.Fatalf("PRODUCT: recovery erased replacement input: copies=%q", copies)
	}
	if buffer.String() != recoverySource || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection || changes != 0 {
		t.Fatalf("recovery changed main source/history/selection: source=%q history=%+v selection=%+v changes=%d", buffer.String(), buffer.History(), buffer.Selection(), changes)
	}
}

func TestMarkdownRecoveryIdentityRefusedCompositionAcceptedBytes(t *testing.T) {
	buffer := text.NewWithLimit(recoverySource, len(recoverySource))
	version, history, selection := buffer.Version(), buffer.History(), buffer.Selection()
	var reports []error
	var copies []string
	changes := 0
	view := publicAdapter.Adapter().Create(editor.Host{
		Buffer: buffer, Changed: func() { changes++ }, Report: func(err error) { reports = append(reports, err) },
		RecoverDraft: func(_, value string) error { copies = append(copies, value); return nil },
	}).(*publicAdapter.View)
	tester := ui.NewTester(view.Draw, 760, 640)
	tester.Frame()
	recoveryCell(t, tester, "数据，第 2 行，第 1 列：甲", recoveryFieldOne)
	tester.Command("selectAll")
	tester.Compose("已接受𠮷🙂", 5)
	tester.Frame()
	if !view.CompositionActive() {
		t.Fatal("FIXTURE: accepted native composition missing")
	}
	tester.Compose("未接受\xff🙂", 4)
	tester.Frame()
	if len(reports) != 1 || !errors.Is(reports[0], text.ErrUTF8) || !view.CompositionActive() {
		t.Fatal("FIXTURE: invalid incoming composition not observably refused", reports)
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 1 || copies[0] != "已接受𠮷🙂" {
		t.Fatal("PRODUCT: recovery copied rejected input instead of accepted literal", err, copies)
	}
	if buffer.String() != recoverySource || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection || changes != 0 || buffer.MaxBytes() != len(recoverySource) {
		t.Fatal("refused composition recovery changed main authority or admission limit")
	}
}

func TestMarkdownRecoveryIdentityErrorDoesNotPoisonReplacement(t *testing.T) {
	buffer := text.New(recoverySource)
	version, history, selection := buffer.Version(), buffer.History(), buffer.Selection()
	failure := errors.New("old draft recovery failure")
	var view *publicAdapter.View
	var tester *ui.Tester
	var copies []string
	var reports []error
	view = publicAdapter.Adapter().Create(editor.Host{
		Buffer: buffer, Report: func(err error) { reports = append(reports, err) },
		RecoverDraft: func(_, value string) error {
			copies = append(copies, value)
			if len(copies) == 1 {
				view.DiscardDraft()
				tester.Frame()
				tester.Advance(time.Second)
				recoveryCell(t, tester, "数据，第 2 行，第 2 列：乙", recoveryFieldTwo)
				recoveryType(t, tester, recoveryFieldTwo, "新🙂未受旧错影响")
				return failure
			}
			return nil
		},
	}).(*publicAdapter.View)
	tester = ui.NewTester(view.Draw, 760, 640)
	tester.Frame()
	recoveryCell(t, tester, "数据，第 2 行，第 1 列：甲", recoveryFieldOne)
	recoveryType(t, tester, recoveryFieldOne, "旧🙂失败副本")
	if err := view.RecoverDraft(); err != failure || len(reports) != 1 || reports[0] != failure {
		t.Fatal("original old-owner recovery error not reported", err, reports)
	}
	tester.Frame()
	recoveryField(t, tester, recoveryFieldTwo)
	if slices.Contains(view.Warnings(), "单元格草稿尚未写回："+failure.Error()) || tester.HasText("单元格操作问题") {
		t.Fatal("PRODUCT: old recovery error poisoned replacement draft")
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 2 || copies[0] != "旧🙂失败副本" || copies[1] != "新🙂未受旧错影响" {
		t.Fatal("replacement input lost after original failure", err, copies)
	}
	if buffer.String() != recoverySource || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection {
		t.Fatal("recovery error replacement changed main authority")
	}
}

func TestMarkdownRecoveryIdentitySameTextNewComposition(t *testing.T) {
	buffer := text.New(recoverySource)
	version, history, selection := buffer.Version(), buffer.History(), buffer.Selection()
	const initial = "相同🙂草稿"
	const final = "最终𠮷🙂文字"
	var view *publicAdapter.View
	var tester *ui.Tester
	var copies []string
	view = publicAdapter.Adapter().Create(editor.Host{Buffer: buffer, RecoverDraft: func(_, value string) error {
		copies = append(copies, value)
		if len(copies) == 1 {
			if view.CompositionActive() {
				t.Fatal("initiating composition was not flushed before copy")
			}
			tester.Command("selectAll")
			tester.Compose(initial, utf8.RuneCountInString(initial))
			tester.Frame()
			if !view.CompositionActive() {
				t.Fatal("FIXTURE: same-text new composition not active")
			}
		}
		return nil
	}}).(*publicAdapter.View)
	tester = ui.NewTester(view.Draw, 760, 640)
	tester.Frame()
	recoveryCell(t, tester, "数据，第 2 行，第 1 列：甲", recoveryFieldOne)
	tester.Command("selectAll")
	tester.Compose(initial, utf8.RuneCountInString(initial))
	tester.Frame()
	if !view.CompositionActive() {
		t.Fatal("FIXTURE: initial native composition not active")
	}
	if err := view.RecoverDraft(); err != nil {
		t.Fatal(err)
	}
	if !view.CompositionActive() {
		t.Fatal("PRODUCT: recovery erased same-text new composition")
	}
	tester.Type(final)
	tester.Frame()
	if view.CompositionActive() {
		t.Fatal("native Unicode commit left composition active")
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 2 || copies[0] != initial || copies[1] != final {
		t.Fatal("new composition final recovery literal", err, copies)
	}
	if buffer.String() != recoverySource || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection {
		t.Fatal("same-text composition recovery changed main authority")
	}
}

func TestMarkdownRecoveryIdentityOriginalErrorAndRetry(t *testing.T) {
	buffer := text.New(recoverySource)
	version, history, selection := buffer.Version(), buffer.History(), buffer.Selection()
	failure := errors.New("recovery fixture failed once")
	var copies []string
	var reports []error
	view := publicAdapter.Adapter().Create(editor.Host{
		Buffer: buffer, Report: func(err error) { reports = append(reports, err) },
		RecoverDraft: func(_, value string) error {
			copies = append(copies, value)
			if len(copies) == 1 {
				return failure
			}
			return nil
		},
	}).(*publicAdapter.View)
	tester := ui.NewTester(view.Draw, 760, 640)
	tester.Frame()
	recoveryCell(t, tester, "数据，第 2 行，第 1 列：甲", recoveryFieldOne)
	recoveryType(t, tester, recoveryFieldOne, "失败🙂仍保留")
	if err := view.RecoverDraft(); err != failure {
		t.Fatalf("original callback error replaced: %v", err)
	}
	tester.Frame()
	recoveryField(t, tester, recoveryFieldOne)
	if len(reports) != 1 || reports[0] != failure || !tester.HasText("单元格操作问题") || !slices.Contains(view.Warnings(), "单元格草稿尚未写回："+failure.Error()) {
		t.Fatal("original recovery failure not visibly reported", reports, tester.Texts())
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 2 || copies[0] != "失败🙂仍保留" || copies[1] != "失败🙂仍保留" {
		t.Fatal("retry lost retained payload", err, copies)
	}
	if buffer.String() != recoverySource || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection {
		t.Fatal("failed recovery/retry changed main authority")
	}
}

func TestMarkdownRecoveryIdentityNormalSuccessAndNoDraft(t *testing.T) {
	buffer := text.New(recoverySource)
	version, history, selection := buffer.Version(), buffer.History(), buffer.Selection()
	var copies []string
	reports, changes := 0, 0
	view := publicAdapter.Adapter().Create(editor.Host{
		Buffer: buffer, Changed: func() { changes++ }, Report: func(error) { reports++ },
		RecoverDraft: func(_, value string) error { copies = append(copies, value); return nil },
	}).(*publicAdapter.View)
	if err := view.RecoverDraft(); err != nil || len(copies) != 0 || reports != 0 {
		t.Fatal("no-draft recovery was not quiet", err, copies, reports)
	}
	tester := ui.NewTester(view.Draw, 760, 640)
	tester.Frame()
	recoveryCell(t, tester, "数据，第 2 行，第 1 列：甲", recoveryFieldOne)
	recoveryType(t, tester, recoveryFieldOne, "正常🙂副本")
	if err := view.RecoverDraft(); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if _, ok := tester.AccessByRole(ui.RoleTextField, recoveryFieldOne); ok {
		t.Fatal("ordinary success left recovered draft open")
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 1 || copies[0] != "正常🙂副本" || reports != 0 {
		t.Fatal("success/no-draft recovery count or text", err, copies, reports)
	}
	if buffer.String() != recoverySource || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection || changes != 0 {
		t.Fatal("ordinary recovery changed main source/history/selection")
	}
}

func TestMarkdownRecoveryIdentitySameOwnerChangedText(t *testing.T) {
	buffer := text.New(recoverySource)
	version, history, selection := buffer.Version(), buffer.History(), buffer.Selection()
	var view *publicAdapter.View
	var tester *ui.Tester
	var copies []string
	view = publicAdapter.Adapter().Create(editor.Host{Buffer: buffer, RecoverDraft: func(_, value string) error {
		copies = append(copies, value)
		if len(copies) == 1 {
			recoveryType(t, tester, recoveryFieldOne, "后来🙂的新输入")
		}
		return nil
	}}).(*publicAdapter.View)
	tester = ui.NewTester(view.Draw, 760, 640)
	tester.Frame()
	recoveryCell(t, tester, "数据，第 2 行，第 1 列：甲", recoveryFieldOne)
	recoveryType(t, tester, recoveryFieldOne, "先前🙂的输入")
	if err := view.RecoverDraft(); err != nil {
		t.Fatal(err)
	}
	if err := view.RecoverDraft(); err != nil {
		t.Fatal(err)
	}
	if len(copies) != 2 || copies[0] != "先前🙂的输入" || copies[1] != "后来🙂的新输入" {
		t.Fatalf("PRODUCT: recovery erased same-owner new text: copies=%q", copies)
	}
	if buffer.String() != recoverySource || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection {
		t.Fatal("recovery changed main source/history/selection")
	}
}

func TestMarkdownRecoveryIdentitySizeRefusedSuccessCloses(t *testing.T) {
	buffer := text.New(recoverySource)
	selection := text.Range{Anchor: buffer.LenUTF16(), Head: 2}
	if err := buffer.SetSelection(selection.Anchor, selection.Head); err != nil {
		t.Fatal(err)
	}
	version, history := buffer.Version(), buffer.History()
	var reports []error
	var copies []string
	changes := 0
	view := publicAdapter.Adapter().Create(editor.Host{
		Buffer: buffer, Changed: func() { changes++ }, Report: func(err error) { reports = append(reports, err) },
		RecoverDraft: func(_, value string) error { copies = append(copies, value); return nil },
	}).(*publicAdapter.View)
	tester := ui.NewTester(view.Draw, 760, 640)
	tester.Frame()
	recoveryCell(t, tester, "数据，第 2 行，第 1 列：甲", recoveryFieldOne)
	tester.Command("selectAll")
	tester.Compose("已接受𠮷🙂", 5)
	tester.Frame()
	if !view.CompositionActive() {
		t.Fatal("FIXTURE: accepted native preedit missing")
	}
	// The rejected string exercises admission only; it must never reach rendering.
	tester.Compose(strings.Repeat("x", (128<<20)+1), 0)
	tester.Frame()
	if len(reports) != 1 || !errors.Is(reports[0], text.ErrSizeLimit) || !view.CompositionActive() {
		t.Fatal("FIXTURE: actual default-limit refusal missing", reports)
	}
	recoveryField(t, tester, recoveryFieldOne)
	if err := view.RecoverDraft(); err != nil || len(copies) != 1 || copies[0] != "已接受𠮷🙂" {
		t.Fatal("accepted literal recovery failed", err, copies)
	}
	tester.Frame()
	_, fieldStillOpen := tester.AccessByRole(ui.RoleTextField, recoveryFieldOne)
	if fieldStillOpen || view.CompositionActive() {
		t.Fatal("PRODUCT: successful recovery retained unchanged refused composition", fieldStillOpen, view.CompositionActive())
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 1 {
		t.Fatal("PRODUCT: recovered refused draft was copied again", err, copies)
	}
	if buffer.String() != recoverySource || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection || changes != 0 || len(reports) != 1 || buffer.MaxBytes() != 128<<20 {
		t.Fatal("recovery changed main authority or original failure report")
	}
}

func TestMarkdownRecoveryIdentitySizeRefusedSameLiteralInput(t *testing.T) {
	buffer := text.New(recoverySource)
	version, history, selection := buffer.Version(), buffer.History(), buffer.Selection()
	const initial = "拒收前𠮷🙂"
	const final = "后续🙂最终文字"
	var view *publicAdapter.View
	var tester *ui.Tester
	var copies []string
	var reports []error
	view = publicAdapter.Adapter().Create(editor.Host{
		Buffer: buffer, Report: func(err error) { reports = append(reports, err) },
		RecoverDraft: func(_, value string) error {
			copies = append(copies, value)
			if len(copies) == 1 {
				if !view.CompositionActive() {
					t.Fatal("FIXTURE: persistent issue did not retain initiating preedit")
				}
				tester.Compose(initial, utf8.RuneCountInString(initial))
				tester.Frame()
			}
			return nil
		},
	}).(*publicAdapter.View)
	tester = ui.NewTester(view.Draw, 760, 640)
	tester.Frame()
	recoveryCell(t, tester, "数据，第 2 行，第 1 列：甲", recoveryFieldOne)
	tester.Command("selectAll")
	tester.Compose(initial, utf8.RuneCountInString(initial))
	tester.Compose(strings.Repeat("x", (128<<20)+1), 0)
	tester.Frame()
	if len(reports) != 1 || !errors.Is(reports[0], text.ErrSizeLimit) || !view.CompositionActive() {
		t.Fatal("FIXTURE: default-limit refused incoming preedit missing", reports)
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 1 || copies[0] != initial {
		t.Fatal("accepted literal first recovery failed", err, copies)
	}
	if !view.CompositionActive() {
		t.Fatal("PRODUCT: recovery erased callback same-literal accepted input")
	}
	recoveryField(t, tester, recoveryFieldOne)
	tester.Type(final)
	tester.Frame()
	if view.CompositionActive() {
		t.Fatal("native final commit left composition active")
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 2 || copies[1] != final {
		t.Fatal("callback input final recovery failed", err, copies)
	}
	tester.Frame()
	if _, open := tester.AccessByRole(ui.RoleTextField, recoveryFieldOne); open {
		t.Fatal("success did not close recovered final field")
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 2 {
		t.Fatal("final recovery was not quiet", err, copies)
	}
	if buffer.String() != recoverySource || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection || len(reports) != 1 {
		t.Fatal("same-literal recovery changed main authority or original report")
	}
}

func TestMarkdownRecoveryIdentitySizeRefusedNewComposition(t *testing.T) {
	buffer := text.New(recoverySource)
	version, history, selection := buffer.Version(), buffer.History(), buffer.Selection()
	const initial = "同字𠮷🙂新组合"
	const final = "新组合🙂最终𠮷"
	var view *publicAdapter.View
	var tester *ui.Tester
	var copies []string
	var reports []error
	view = publicAdapter.Adapter().Create(editor.Host{
		Buffer: buffer, Report: func(err error) { reports = append(reports, err) },
		RecoverDraft: func(_, value string) error {
			copies = append(copies, value)
			if len(copies) == 1 {
				tester.Type(initial)
				tester.Frame()
				if view.CompositionActive() {
					t.Fatal("FIXTURE: accepted commit did not end refused composition")
				}
				tester.Command("selectAll")
				tester.Compose(initial, utf8.RuneCountInString(initial))
				tester.Frame()
				if !view.CompositionActive() {
					t.Fatal("FIXTURE: callback new same-text composition missing")
				}
			}
			return nil
		},
	}).(*publicAdapter.View)
	tester = ui.NewTester(view.Draw, 760, 640)
	tester.Frame()
	recoveryCell(t, tester, "数据，第 2 行，第 1 列：甲", recoveryFieldOne)
	tester.Command("selectAll")
	tester.Compose(initial, utf8.RuneCountInString(initial))
	tester.Compose(strings.Repeat("x", (128<<20)+1), 0)
	tester.Frame()
	if len(reports) != 1 || !errors.Is(reports[0], text.ErrSizeLimit) || !view.CompositionActive() {
		t.Fatal("FIXTURE: persistent default-limit refusal missing", reports)
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 1 || copies[0] != initial {
		t.Fatal("first accepted recovery failed", err, copies)
	}
	if !view.CompositionActive() {
		t.Fatal("PRODUCT: recovery erased new same-text composition after refused commit")
	}
	recoveryField(t, tester, recoveryFieldOne)
	tester.Type(final)
	tester.Frame()
	if view.CompositionActive() {
		t.Fatal("final native Unicode commit remained active")
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 2 || copies[1] != final {
		t.Fatal("new composition final literal recovery failed", err, copies)
	}
	tester.Frame()
	if _, open := tester.AccessByRole(ui.RoleTextField, recoveryFieldOne); open {
		t.Fatal("recovered final draft remained open")
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 2 {
		t.Fatal("recovered final draft was not quiet", err, copies)
	}
	if buffer.String() != recoverySource || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection || len(reports) != 1 {
		t.Fatal("new composition recovery changed main authority or report")
	}
}

func TestMarkdownRecoveryIdentitySizeRefusedReplacementOwner(t *testing.T) {
	buffer := text.New(recoverySource)
	version, history, selection := buffer.Version(), buffer.History(), buffer.Selection()
	var view *publicAdapter.View
	var tester *ui.Tester
	var copies []string
	var reports []error
	view = publicAdapter.Adapter().Create(editor.Host{
		Buffer: buffer, Report: func(err error) { reports = append(reports, err) },
		RecoverDraft: func(_, value string) error {
			copies = append(copies, value)
			if len(copies) == 1 {
				view.DiscardDraft()
				tester.Frame()
				tester.Advance(time.Second)
				recoveryCell(t, tester, "数据，第 2 行，第 2 列：乙", recoveryFieldTwo)
				recoveryType(t, tester, recoveryFieldTwo, "替换🙂新草稿")
			}
			return nil
		},
	}).(*publicAdapter.View)
	tester = ui.NewTester(view.Draw, 760, 640)
	tester.Frame()
	recoveryCell(t, tester, "数据，第 2 行，第 1 列：甲", recoveryFieldOne)
	tester.Command("selectAll")
	tester.Compose("旧拒收𠮷🙂", 5)
	tester.Compose(strings.Repeat("x", (128<<20)+1), 0)
	tester.Frame()
	if len(reports) != 1 || !errors.Is(reports[0], text.ErrSizeLimit) || !view.CompositionActive() {
		t.Fatal("FIXTURE: refused initiating preedit missing", reports)
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 1 || copies[0] != "旧拒收𠮷🙂" {
		t.Fatal("old accepted recovery failed", err, copies)
	}
	recoveryField(t, tester, recoveryFieldTwo)
	if err := view.RecoverDraft(); err != nil || len(copies) != 2 || copies[1] != "替换🙂新草稿" {
		t.Fatal("PRODUCT: refused recovery erased replacement owner", err, copies)
	}
	tester.Frame()
	if _, open := tester.AccessByRole(ui.RoleTextField, recoveryFieldTwo); open || view.CompositionActive() {
		t.Fatal("replacement success remained open or active")
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 2 {
		t.Fatal("replacement success was not quiet", err, copies)
	}
	if buffer.String() != recoverySource || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection || len(reports) != 1 {
		t.Fatal("replacement recovery changed main authority or original report")
	}
}

func TestMarkdownRecoveryIdentityNestedErrorObserverRestored(t *testing.T) {
	buffer := text.New(recoverySource)
	version, history, selection := buffer.Version(), buffer.History(), buffer.Selection()
	failure := errors.New("nested recovery failure")
	const initial = "嵌套𠮷🙂草稿"
	const final = "嵌套后🙂最终字"
	var view *publicAdapter.View
	var tester *ui.Tester
	var copies []string
	var reports []error
	changes := 0
	view = publicAdapter.Adapter().Create(editor.Host{
		Buffer: buffer, Changed: func() { changes++ }, Report: func(err error) { reports = append(reports, err) },
		RecoverDraft: func(_, value string) error {
			copies = append(copies, value)
			switch len(copies) {
			case 1:
				if err := view.RecoverDraft(); err != failure {
					t.Fatal("nested recovery lost original error", err)
				}
				tester.Command("selectAll")
				tester.Compose(initial, utf8.RuneCountInString(initial))
				tester.Frame()
			case 2:
				return failure
			}
			return nil
		},
	}).(*publicAdapter.View)
	tester = ui.NewTester(view.Draw, 760, 640)
	tester.Frame()
	recoveryCell(t, tester, "数据，第 2 行，第 1 列：甲", recoveryFieldOne)
	tester.Command("selectAll")
	tester.Compose(initial, utf8.RuneCountInString(initial))
	tester.Frame()
	if err := view.RecoverDraft(); err != nil || len(copies) != 2 || copies[0] != initial || copies[1] != initial {
		t.Fatal("nested accepted copies failed", err, copies)
	}
	if !view.CompositionActive() || len(reports) != 1 || reports[0] != failure {
		t.Fatal("PRODUCT: nested error restore lost outer new input or original report", reports)
	}
	recoveryField(t, tester, recoveryFieldOne)
	tester.Type(final)
	tester.Frame()
	if view.CompositionActive() {
		t.Fatal("post-callback Unicode commit remained active")
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 3 || copies[2] != final {
		t.Fatal("post-callback restored input failed", err, copies)
	}
	tester.Frame()
	if _, open := tester.AccessByRole(ui.RoleTextField, recoveryFieldOne); open {
		t.Fatal("nested final success remained open")
	}
	if err := view.RecoverDraft(); err != nil || len(copies) != 3 {
		t.Fatal("nested final recovery was not quiet", err, copies)
	}
	if buffer.String() != recoverySource || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection || changes != 0 || len(reports) != 1 {
		t.Fatal("nested callback altered authority or failure report")
	}
}
