package app

import "github.com/egoist/mygo/ui"

// workbenchStyle separates the editor/control theme from the surrounding chrome.
// Each call owns its Theme so a scoped toolbar theme can be copied independently.
type workbenchStyle struct {
	Theme                                                                    *ui.Theme
	Chrome, Sidebar, Activity, TabBar, TabActive, Status, StatusText, Subtle ui.Color
}

func workbenchAppearance(dark bool) workbenchStyle {
	// flowix 设计令牌移植(text2future/flowix styles/theme):冷蓝灰中性轴
	// (浅色 hue 255 / 深色 hue 265),浅色紫色品牌锚点、深色金黄品牌锚点,
	// 分层纸面(chrome 略浅于卡片)、focus ring 跟随品牌色、8px 圆角。
	// 文本类色值按 OKLCH 亮度自适应到 AA(4.5+)对比度。
	theme := ui.LightTheme()
	theme.Background = ui.Hex("#f8fbff")
	theme.Surface = ui.Hex("#ffffff")
	theme.SurfaceHover = ui.Hex("#ebeef2")
	theme.SurfacePressed = ui.Hex("#e6e9ec")
	theme.Border = ui.Hex("#e6e9ec")
	theme.Text = ui.Hex("#2b2e32")
	theme.TextMuted = ui.Hex("#64686d")
	theme.Danger = ui.Hex("#ba393f")
	theme.Warning = ui.Hex("#8c5f0e")
	theme.Success = ui.Hex("#337538")
	theme.Selection = ui.Hex("#e1e7fd")
	theme.Focus = ui.Hex("#4c5ad3")
	theme.Scrollbar = ui.RGBA(100, 104, 109, 0.4)
	theme.Accent = ui.Hex("#4c5ad3")
	theme.AccentHover = ui.Hex("#3f4bc2")
	theme.AccentPressed = ui.Hex("#363eb5")
	theme.AccentText = ui.Hex("#ffffff")
	theme.InactiveAccent = ui.Hex("#64686d")
	chrome := ui.Hex("#f7fafd")
	sidebar := ui.Hex("#f7fafd")
	if dark {
		theme.Dark = true
		theme.Background = ui.Hex("#0b0d13")
		theme.Surface = ui.Hex("#1c1f26")
		theme.SurfaceHover = ui.Hex("#282c34")
		theme.SurfacePressed = ui.Hex("#31353e")
		theme.Border = ui.Hex("#2a2e35")
		theme.Text = ui.Hex("#c0c4cd")
		theme.TextMuted = ui.Hex("#9fa3ad")
		theme.Danger = ui.Hex("#ff734f")
		theme.Warning = ui.Hex("#f0b135")
		theme.Success = ui.Hex("#1ec55f")
		theme.Selection = ui.Hex("#2b303c")
		theme.Focus = ui.Hex("#e4bf46")
		theme.Scrollbar = ui.RGBA(159, 163, 173, 0.5)
		theme.Accent = ui.Hex("#e4bf46")
		theme.AccentHover = ui.Hex("#f4d05d")
		theme.AccentPressed = ui.Hex("#d5af24")
		theme.AccentText = ui.Hex("#0e1014")
		theme.InactiveAccent = ui.Hex("#7a7d85")
		chrome = ui.Hex("#0e1014")
		sidebar = ui.Hex("#13161c")
	}
	theme.Inverse = theme.Text
	theme.InverseText = theme.Background
	// Focus stays distinct from filled actions; Selection is the quiet tint for
	// navigation. Text and semantic colors stay readable through hover/press.
	theme.FocusRingWidth = 2
	theme.FocusRingOffset = 1
	theme.Radius = 8
	theme.Spacing = 4
	theme.Font = "Microsoft YaHei UI"
	theme.FontSize = 14
	return workbenchStyle{
		Theme: theme, Chrome: chrome, Sidebar: sidebar, Activity: chrome,
		TabBar: chrome, TabActive: theme.Background,
		Status: chrome, StatusText: theme.TextMuted,
		Subtle: theme.TextMuted,
	}
}
