package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"levnote.local/native/core/document"
)

func (a *App) exportCurrent() {
	t := a.Current()
	if t == nil {
		return
	}
	if t.isBinary() {
		a.Message = "图片没有文档导出；请使用另存副本保存原始字节"
		return
	}
	if a.exportJob != nil {
		a.Message = "已有导出正在完成或清理，请稍候"
		return
	}
	if t.Adapter != nil && t.Adapter.ExportEngine != nil {
		if a.refuseUnpreparedViewExport(t, *t.Adapter.ExportEngine) {
			return
		}
		a.startEngineExport(t)
		return
	}
	if t.View == nil || t.Adapter == nil || len(t.Adapter.Exports) == 0 {
		if t.Adapter == nil && t.MatchedPlugin.Declarative {
			a.report(fmt.Errorf("当前声明式格式只提供源码编辑，没有可导出的阅读视图。"))
		}
		return
	}
	if !a.flushTab(t, "导出") {
		return
	}
	defer a.deferExportDelivery()()
	format := t.Adapter.Exports[0]
	data, err := t.View.Export(format)
	if err != nil {
		a.report(err)
		return
	}
	base := t.Name
	if dot := strings.LastIndexByte(base, '.'); dot > 0 {
		base = base[:dot]
	}
	path, err := a.chooseExportPath(mygo.SaveDialogOptions{
		Parent: a.Window, Title: "导出 " + strings.ToUpper(format),
		DefaultPath: base + "." + format,
	})
	if err != nil {
		a.report(err)
		return
	}
	if path == "" {
		return
	}
	if err := writeExportFile(path, data); err != nil {
		a.report(fmt.Errorf("导出失败：%w", err))
		return
	}
	a.Message = "已导出 " + path
}

func writeExportFile(path string, data []byte) error {
	return document.WriteNewFile(path, data)
}
