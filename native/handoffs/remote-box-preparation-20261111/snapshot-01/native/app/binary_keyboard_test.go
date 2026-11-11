package app

import (
	"image/color"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestNativeImageKeyboardByteAndTabBoundaries(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "keyboard.png")
	imageFixture(t, imagePath, "PNG")

	a := newTestApp(t, filepath.Join(root, "config"))
	a.NewDocument("text")
	textTab := a.Current()
	textTab.Editor.ReplaceText(nil, "plain")

	a.Open(imagePath)
	imageTab := a.Current()
	if imageTab == nil || imageTab.Binary == nil || imageTab == textTab {
		t.Fatal("image fixture did not become the active binary tab")
	}
	token := imageTab.Binary.Token()

	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	if a.binaryPreview != nil {
		waitBinaryTask(t, a.binaryPreview.task)
		tester.Frame()
	}
	tester.Frame()
	if imageTab.ImageWidth != 40 || imageTab.ImageHeight != 24 || !tester.HasText("适合窗口") {
		t.Fatalf("image preview did not settle: %dx%d texts=%#v", imageTab.ImageWidth, imageTab.ImageHeight, tester.Texts())
	}

	if err := tester.Click("按百分之百显示"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !tester.Focused("图片查看区域") {
		t.Fatal("enabled 100% action did not return focus to the image canvas")
	}
	canvas, ok := tester.Find("图片查看区域")
	if !ok {
		t.Fatalf("image canvas bounds unavailable: %#v", tester.Texts())
	}
	// At 100% the 40-pixel-wide fixture stops 20 DIPs from its center. A
	// pixel 22 DIPs to the right is therefore backdrop; at 125% it is image.
	// This observes the public rendered frame rather than the labeled zoom text.
	sample := func() color.RGBA {
		frame := tester.Image()
		if frame == nil {
			t.Fatal("Tester returned no rendered frame")
		}
		x := int(canvas.X + canvas.W/2 + 22)
		y := int(canvas.Y + canvas.H/2)
		return frame.RGBAAt(x, y)
	}
	at100 := sample()
	tester.Key(0, ui.KeyEqual)
	tester.Frame()
	at125 := sample()
	if at125 == at100 {
		t.Fatalf("public image geometry did not change after keyboard zoom: pixel=%#v", at100)
	}
	tester.Key(0, ui.Key1)
	tester.Frame()
	if restored := sample(); restored != at100 {
		t.Fatalf("1 key did not restore the 100%% image geometry: before=%#v zoomed=%#v restored=%#v",
			at100, at125, restored)
	}

	if err := tester.Click("查看只读字节"); err != nil {
		t.Fatal(err)
	}
	if imageTab.Mode != "source" || !imageTab.Editor.ReadOnly {
		t.Fatalf("real byte button did not enter the readonly byte editor: mode=%q readonly=%v", imageTab.Mode, imageTab.Editor.ReadOnly)
	}
	tester.Frame()
	if a.binaryPreview != nil {
		waitBinaryTask(t, a.binaryPreview.task)
		tester.Frame()
	}
	if err := tester.Click("只读原始字节"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !tester.Focused("只读原始字节") {
		t.Fatalf("readonly byte editor did not retain public keyboard focus: %#v", tester.Texts())
	}

	bytesBefore := imageTab.Editor.Buffer.String()
	versionBefore := imageTab.Editor.Buffer.Version()
	if bytesBefore == "" || imageTab.Binary.Token() != token || a.heldBinary(imageTab) == nil {
		t.Fatal("byte-view fixture lacks held bytes, token, or inspector content")
	}
	for _, key := range []ui.Key{ui.Key0, ui.Key1, ui.KeyEqual, ui.KeyMinus, ui.KeyLeft, ui.KeyRight, ui.KeyUp, ui.KeyDown} {
		tester.Key(0, key)
	}
	tester.CodeDown(0, ui.CodeNumpadAdd)
	tester.CodeUp(0, ui.CodeNumpadAdd)
	tester.TypeKey(0, ui.KeyEqual, "=")
	tester.Compose("组合", 2)
	tester.Command("paste")
	if imageTab.Mode != "source" ||
		imageTab.Editor.Buffer.String() != bytesBefore ||
		imageTab.Editor.Buffer.Version() != versionBefore ||
		imageTab.Binary.Token() != token {
		t.Fatalf("image keys changed readonly bytes, version, mode, or token: mode=%q version=%d token=%#v",
			imageTab.Mode, imageTab.Editor.Buffer.Version(), imageTab.Binary.Token())
	}

	if held := a.heldBinary(imageTab); held == nil || held.Token() != token {
		t.Fatal("byte view no longer holds the active binary baseline")
	}
	tester.Key(ui.Cmd, ui.KeyTab)
	if a.Current() != textTab || a.binarySnapshot != nil || a.heldBinary(imageTab) != nil {
		t.Fatalf("Ctrl+Tab did not release and transfer active tab authority: active=%v snapshot=%v", a.Current(), a.binarySnapshot)
	}
	if imageTab.Mode != "source" || imageTab.Binary.Token() != token || imageTab.Editor.Buffer.LenBytes() != 0 {
		t.Fatal("leaving the byte tab changed its mode/token or retained held bytes")
	}

	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	textBefore := textTab.Doc.Buffer.String()
	tester.Type(" typed")
	tester.TypeKey(0, ui.KeyEqual, "=")
	if !strings.Contains(textTab.Doc.Buffer.String(), "typed=") || !strings.Contains(textTab.Doc.Buffer.String(), "plain") {
		t.Fatalf("plain text did not receive input after Ctrl+Tab: %q", textTab.Doc.Buffer.String())
	}
	if !strings.Contains(textTab.Doc.Buffer.String(), textBefore) {
		t.Fatalf("text input replaced rather than edited the active document: before=%q after=%q", textBefore, textTab.Doc.Buffer.String())
	}
	if imageTab.Mode != "source" || imageTab.Binary.Token() != token || a.binarySnapshot != nil {
		t.Fatal("image shortcuts leaked across the text tab boundary")
	}

	tester.Key(ui.Cmd|ui.Shift, ui.KeyTab)
	if a.Current() != imageTab {
		t.Fatal("Ctrl+Shift+Tab did not return to the image tab")
	}
	tester.Frame()
	if a.binaryPreview != nil {
		waitBinaryTask(t, a.binaryPreview.task)
		tester.Frame()
	}
	if imageTab.Mode != "source" || !imageTab.Editor.ReadOnly ||
		imageTab.Binary.Token() != token || imageTab.Editor.Buffer.String() != bytesBefore ||
		a.heldBinary(imageTab) == nil {
		t.Fatalf("returning to the byte tab did not restore authority/content: mode=%q readonly=%v token=%#v held=%v",
			imageTab.Mode, imageTab.Editor.ReadOnly, imageTab.Binary.Token(), a.heldBinary(imageTab) != nil)
	}
}
