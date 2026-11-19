package app

import "github.com/egoist/mygo/ui"

// The workbench uses local, monochrome SVGs. They follow the theme's text
// color and never depend on a symbol font, a network asset or an icon package.
func workbenchSVG(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var (
	workbenchBrandIcon       = workbenchSVG(`<path d="M6 3h9l4 4v12a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2M14 3v5h5M8.5 12h7M8.5 16h4"/>`)
	workbenchFilesIcon       = workbenchSVG(`<path d="M9 3h8a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2M4 7v13a1 1 0 0 0 1 1h9"/>`)
	workbenchOutlineIcon     = workbenchSVG(`<path d="M9 5h11M9 12h8M9 19h11"/><circle cx="4" cy="5" r="1"/><circle cx="4" cy="12" r="1"/><circle cx="4" cy="19" r="1"/>`)
	workbenchSearchIcon      = workbenchSVG(`<circle cx="10.5" cy="10.5" r="6.5"/><path d="m15.5 15.5 4.5 4.5"/>`)
	workbenchPluginsIcon     = workbenchSVG(`<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/><path d="M17.5 3v7M14 6.5h7"/>`)
	workbenchSettingsIcon    = workbenchSVG(`<path d="M3 6h4m4 0h10M3 12h10m4 0h4M3 18h4m4 0h10"/><circle cx="9" cy="6" r="2"/><circle cx="15" cy="12" r="2"/><circle cx="9" cy="18" r="2"/>`)
	workbenchFolderIcon      = workbenchSVG(`<path d="M3 8V6a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8h18"/>`)
	workbenchDocumentIcon    = workbenchSVG(`<path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8l-5-5v5h5M9 12h6M9 16h4"/>`)
	workbenchMarkdownIcon    = workbenchSVG(`<rect x="2.5" y="5" width="19" height="14" rx="2"/><path d="M6 15V9l3 3 3-3v6M17 9v6m-2-2 2 2 2-2"/>`)
	workbenchCodeIcon        = workbenchSVG(`<path d="m8 7-5 5 5 5m8-10 5 5-5 5M14 4l-4 16"/>`)
	workbenchDiagramIcon     = workbenchSVG(`<rect x="8" y="3" width="8" height="5" rx="1.5"/><rect x="2" y="16" width="8" height="5" rx="1.5"/><rect x="14" y="16" width="8" height="5" rx="1.5"/><path d="M12 8v4M6 16v-4h12v4"/>`)
	workbenchChevronIcon     = workbenchSVG(`<path d="m9 6 6 6-6 6"/>`)
	workbenchNoticeIcon      = workbenchSVG(`<circle cx="12" cy="12" r="8.5"/><path d="M12 7.5v5m0 4v.1"/>`)
	workbenchPlusIcon        = workbenchSVG(`<path d="M12 5v14M5 12h14"/>`)
	workbenchSaveIcon        = workbenchSVG(`<path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2M17 21v-8H7v8M7 3v5h8"/>`)
	workbenchShapesIcon      = workbenchSVG(`<path d="m8 3 5 9H3L8 3M18.5 21a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7M3 15h10v7H3z"/>`)
	workbenchCommandIcon     = workbenchSVG(`<path d="M9 9V6.5A2.5 2.5 0 1 0 6.5 9H9zm0 0v6m0-6h6m-6 6v2.5A2.5 2.5 0 1 1 6.5 15H9zm6-6h2.5A2.5 2.5 0 1 0 15 6.5V9zm0 6v2.5a2.5 2.5 0 1 0 2.5-2.5H15zm0-6v6"/>`)
	workbenchChevronDownIcon = workbenchSVG(`<path d="m6 9 6 6 6-6"/>`)
)
