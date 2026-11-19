package app

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

const maxGoToLineQuery = 32

type sourceLineTarget struct {
	line, column int
}

func parseSourceLineTarget(query string) (sourceLineTarget, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return sourceLineTarget{}, errors.New("请输入行号，例如 12 或 12:5")
	}
	parts := strings.Split(query, ":")
	if len(parts) > 2 || parts[0] == "" || len(parts) == 2 && parts[1] == "" {
		return sourceLineTarget{}, errors.New("请输入“行”或“行:列”，例如 12 或 12:5")
	}
	line, err := parsePositiveASCIIDecimal(parts[0])
	if err != nil {
		return sourceLineTarget{}, err
	}
	column := 1
	if len(parts) == 2 {
		column, err = parsePositiveASCIIDecimal(parts[1])
		if err != nil {
			return sourceLineTarget{}, err
		}
	}
	return sourceLineTarget{line: line, column: column}, nil
}

func parsePositiveASCIIDecimal(value string) (int, error) {
	if value == "" {
		return 0, errors.New("行号和列号必须是正整数")
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, errors.New("行号和列号必须是正整数")
		}
	}
	if value[0] == '0' {
		return 0, errors.New("行号和列号必须是正整数")
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New("行号或列号过大")
	}
	return n, nil
}

func validateSourceLineTarget(buffer *text.Buffer, target sourceLineTarget) error {
	if buffer == nil {
		return errors.New("当前文档不可用")
	}
	span, err := buffer.LineRange(target.line - 1)
	if err != nil {
		return fmt.Errorf("源文本没有第 %d 行（共 %d 行）", target.line, buffer.Lines())
	}
	maxColumn := span.End() - span.Start() + 1
	if target.column < 1 || target.column > maxColumn {
		return fmt.Errorf("第 %d 行只有 UTF-16 列 1–%d", target.line, maxColumn)
	}
	offset := span.Start() + target.column - 1
	if _, err := buffer.SliceUTF16(offset, offset); errors.Is(err, text.ErrSurrogate) {
		return fmt.Errorf("第 %d 行第 %d 列位于 UTF-16 代理项内部", target.line, target.column)
	} else if err != nil {
		return fmt.Errorf("目标位置不可用：%w", err)
	}
	return nil
}

func (a *App) sourceCompositionActive(t *Tab) bool {
	if t == nil {
		return false
	}
	if _, composing := t.Editor.MarkedRange(); composing {
		return true
	}
	if composer, ok := t.View.(editor.Composer); ok {
		return composer.CompositionActive()
	}
	return false
}

func (a *App) openGoToLine() bool {
	if a.findCompositionActive() {
		a.Message = "输入法正在组合文本；请先完成组合后再转到行号"
		return false
	}
	t := a.Current()
	if t == nil || t.Doc == nil || t.Editor == nil || t.Doc.Buffer == nil {
		a.Message = "没有可转到源码行的文档"
		return false
	}
	if a.sourceCompositionActive(t) {
		a.Message = "输入法正在组合文本；请先完成组合后再转到行号"
		return false
	}
	a.goToLineGeneration++
	a.ShowGoToLine = true
	a.GoToLineQuery = ""
	a.GoToLineError = ""
	a.GoToLineComposing = false
	a.goToLineTab = t
	a.goToLineDoc = t.Doc
	a.goToLineEditor = t.Editor
	a.goToLineBuffer = t.Doc.Buffer
	return true
}

func (a *App) closeGoToLine() {
	a.goToLineGeneration++
	a.ShowGoToLine = false
	a.GoToLineQuery = ""
	a.GoToLineError = ""
	a.GoToLineComposing = false
	a.goToLineTab = nil
	a.goToLineDoc = nil
	a.goToLineEditor = nil
	a.goToLineBuffer = nil
}

func (a *App) goToLineTargetCurrent() bool {
	t := a.Current()
	return t != nil && a.tabIsOpen(t) &&
		t == a.goToLineTab &&
		t.Doc != nil && t.Doc == a.goToLineDoc &&
		t.Doc.Buffer == a.goToLineBuffer &&
		t.Editor != nil && t.Editor == a.goToLineEditor &&
		t.Editor.Buffer == a.goToLineBuffer
}

func (a *App) expireGoToLine() {
	if a.ShowGoToLine && !a.goToLineTargetCurrent() {
		a.closeGoToLine()
	}
}

func (a *App) cancelGoToLine() {
	t := a.goToLineTab
	a.closeGoToLine()
	if t != nil && t == a.Current() && a.tabIsOpen(t) && t.Mode != "preview" {
		t.Editor.Focus()
	}
}

func (a *App) cancelGoToLineAt(generation uint64) {
	if a.ShowGoToLine && a.goToLineGeneration == generation {
		a.cancelGoToLine()
	}
}

func (a *App) submitGoToLine(query string) bool {
	a.GoToLineQuery = query
	target, err := parseSourceLineTarget(query)
	if err != nil {
		a.GoToLineError = err.Error()
		return false
	}
	if !a.goToLineTargetCurrent() {
		a.closeGoToLine()
		a.Message = "转到源码行时文档已变化；未跳转"
		return false
	}
	t := a.goToLineTab
	if !a.flushTab(t, "转到源码行") {
		a.GoToLineError = "文本输入尚未同步；未跳转"
		return false
	}
	if !a.goToLineTargetCurrent() {
		a.closeGoToLine()
		a.Message = "转到源码行时文档已变化；文本输入已同步，但未跳转"
		return false
	}
	if err := validateSourceLineTarget(t.Doc.Buffer, target); err != nil {
		a.GoToLineError = "文本输入已同步，但目标无效：" + err.Error()
		return false
	}
	if err := t.Editor.SetPosition(target.line-1, target.column-1); err != nil {
		a.GoToLineError = "文本输入已同步，但无法定位：" + err.Error()
		return false
	}
	modeChanged := t.Mode == "preview"
	if modeChanged {
		a.releasePresentation(t)
		t.Mode = "source"
	}
	a.markPersistPending()
	t.Editor.Focus()
	a.Message = fmt.Sprintf("已转到源码第 %d 行，第 %d 列", target.line, target.column)
	a.closeGoToLine()
	return true
}

func (a *App) submitGoToLineAt(generation uint64, query string) {
	if !a.ShowGoToLine || a.goToLineGeneration != generation {
		return
	}
	if a.GoToLineComposing {
		a.GoToLineError = "输入法正在组合文本；请先完成组合"
		return
	}
	a.submitGoToLine(query)
}

func (a *App) goToLineBar(c *ui.Context) {
	a.expireGoToLine()
	if !a.ShowGoToLine {
		return
	}
	t := a.goToLineTab
	help := "按源码行定位（包括 JSON/XML；阅读视图也定位源文本）。行、列均从 1 开始，列按 UTF-16 代码单元；省略 :列 时默认为第 1 列，不接受前导 0，输入最多 32 个 UTF-16 代码单元。"
	if t.ByteInspector {
		help = "按当前只读十六进制字节视图的显示行定位。行、列均从 1 开始，列按 UTF-16 代码单元；省略 :列 时默认为第 1 列，不接受前导 0，输入最多 32 个 UTF-16 代码单元。"
	}
	generation := a.goToLineGeneration
	var input ui.Element
	ui.Column(c.Key("gotolinebar")).Shrink(0).Padding(7, 16).Gap(5).
		Background(c.Theme().Surface).Border(1, c.Theme().Border).
		ObserveInput(func(ev ui.InputEvent) {
			if !a.ShowGoToLine || a.goToLineGeneration != generation {
				return
			}
			switch ev.Kind {
			case ui.InputCompose:
				a.GoToLineComposing = ev.Text != ""
			case ui.InputText:
				a.GoToLineComposing = false
			}
		}).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			input = ui.TextInput(c.Key(fmt.Sprintf("gotoline-input-%d", generation)), &a.GoToLineQuery).
				Width(180).
				MaxLength(maxGoToLineQuery).
				AutoFocus().
				Label("转到行号").
				OnTextKey(func(k ui.TextKey) bool {
					if k.Mods != 0 {
						return false
					}
					if !a.ShowGoToLine || a.goToLineGeneration != generation {
						return false
					}
					if a.GoToLineComposing {
						return false
					}
					switch k.Key {
					case ui.KeyEnter:
						a.submitGoToLineAt(generation, k.Text())
						return true
					case ui.KeyEscape:
						a.cancelGoToLineAt(generation)
						return true
					}
					return false
				})
			if ui.PrimaryButton(c, "转到").
				Label("转到源码行").
				Height(30).
				KeepFocus().
				OnShortcut(0, ui.KeyEscape, func() { a.cancelGoToLineAt(generation) }).
				Clicked() {
				a.submitGoToLineAt(generation, a.GoToLineQuery)
			}
			if ui.Button(c, "取消").
				Label("取消转到行号").
				Height(30).
				KeepFocus().
				OnShortcut(0, ui.KeyEscape, func() { a.cancelGoToLineAt(generation) }).
				Clicked() {
				a.cancelGoToLineAt(generation)
			}
		})
		ui.Text(c, help).FontSize(11).MaxLines(2).TextColor(c.Theme().TextMuted)
		if a.GoToLineError != "" {
			ui.Text(c, a.GoToLineError).Label("转到行号错误").
				FontSize(11).MaxLines(2).TextColor(c.Theme().Warning)
		}
	})
	// Pointer focus leave commits preedit without an InputText event. Read
	// the field's actual state so a finished composition cannot block keys.
	if a.GoToLineComposing && !input.Focused() && !input.Composing() {
		a.GoToLineComposing = false
	}
}
