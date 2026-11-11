package modules

import (
	"reflect"
	"testing"

	"levnote.local/native/core/plugins"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/ui/editor"
)

func TestNativeRouteAdapterCapabilitiesTemplateAndSemanticOutline(t *testing.T) {
	for _, fixture := range []struct {
		route                                                     plugins.NativeRoute
		id, format, exportID, sceneID, sceneFormat, title, anchor string
	}{
		{plugins.NativeRouteMarkdown, "markdown", "html", "levnote.markdown-export", "", "", "新文档", "新文档"},
		{plugins.NativeRouteMindmap, "mindmap", "svg", "levnote.mindmap-svg", "levnote.mindmap-preview", "mindmap-scene-v1", "中心主题", ""},
	} {
		t.Run(fixture.id, func(t *testing.T) {
			adapter, ok := NativeRouteAdapter(fixture.route)
			if !ok || adapter.ID != fixture.id || adapter.Create == nil || adapter.Template == nil || adapter.Outline == nil || !reflect.DeepEqual(adapter.Exports, []string{fixture.format}) || adapter.ExportEngine == nil || *adapter.ExportEngine != (editor.ExportEngine{ID: fixture.exportID, Format: fixture.format}) {
				t.Fatalf("route capability = %#v", adapter)
			}
			if fixture.sceneID == "" {
				if adapter.SceneEngine != nil {
					t.Fatal("Markdown invented a scene capability")
				}
			} else if adapter.SceneEngine == nil || adapter.SceneEngine.ID != fixture.sceneID || adapter.SceneEngine.Format != fixture.sceneFormat {
				t.Fatal("Mindmap scene capability missing", adapter.SceneEngine)
			}
			buffer := text.New(adapter.Template())
			before := buffer.String()
			host := editor.Host{Buffer: buffer, Path: "sample.custom-alias", Changed: func() { t.Fatal("observation changed source") }}
			outline := adapter.Outline(host)
			defer outline.Release()
			snapshot, err := outline.OutlineSnapshot()
			if err != nil || snapshot.Truncated || len(snapshot.Items) != 1 || snapshot.Items[0].Text != fixture.title || snapshot.Items[0].AnchorID != fixture.anchor || snapshot.Items[0].Level != 1 {
				t.Fatalf("native template/semantic outline = %#v %v", snapshot, err)
			}
			if fixture.id == "mindmap" && snapshot.Items[0].ID != "root" {
				t.Fatal("native ID replaced by route ID")
			}
			view := adapter.Create(host)
			if view == nil {
				t.Fatal("no native view")
			}
			view.Release()
			if buffer.String() != before || buffer.CanUndo() {
				t.Fatal("template inspection mutated source")
			}
		})
	}
	for _, route := range []plugins.NativeRoute{"", "builtin.unknown", "markdown", "task-notes", "C:/engine.exe"} {
		adapter, ok := NativeRouteAdapter(route)
		if ok || adapter.ID != "" || adapter.Create != nil || adapter.ExportEngine != nil || adapter.SceneEngine != nil {
			t.Fatalf("unknown route %q resolved: %#v", route, adapter)
		}
	}
}

func TestNativeDrawioRouteCapabilitiesAndTemplate(t *testing.T) {
	adapter, ok := NativeRouteAdapter(plugins.NativeRouteDrawio)
	if !ok || adapter.ID != "drawio" || adapter.Name != "原生 draw.io 图纸" ||
		adapter.Create == nil || adapter.Template == nil || adapter.Outline != nil ||
		adapter.SceneEngine != nil || !reflect.DeepEqual(adapter.Exports, []string{"svg"}) ||
		adapter.ExportEngine == nil ||
		*adapter.ExportEngine != (editor.ExportEngine{
			ID: "levnote.drawio-svg", Format: "svg", RequiresView: true,
		}) {
		t.Fatalf("draw.io route capability = %#v", adapter)
	}
	template := adapter.Template()
	document, issues, err := drawio.Parse([]byte(template), drawio.ParseOptions{})
	if err != nil || document == nil || len(document.Pages) != 1 ||
		document.Pages[0].ID != "page-1" || document.Pages[0].Name != "页面 1" {
		t.Fatalf("draw.io template = document:%#v issues:%#v err:%v", document, issues, err)
	}
	for _, issue := range issues {
		if issue.Code != drawio.IssueUnsupportedElement {
			t.Fatalf("draw.io template has unrecognized issue: %#v", issue)
		}
	}
	buffer := text.New(template)
	before := buffer.String()
	view := adapter.Create(editor.Host{Buffer: buffer, Path: "sample.diagramnote"})
	if view == nil {
		t.Fatal("draw.io route did not create a native view")
	}
	view.Release()
	if buffer.String() != before || buffer.CanUndo() {
		t.Fatal("draw.io route template inspection mutated source")
	}
}

func TestNativeRouteAdaptersOwnMutableCapabilities(t *testing.T) {
	for _, route := range []plugins.NativeRoute{
		plugins.NativeRouteMarkdown,
		plugins.NativeRouteMindmap,
		plugins.NativeRouteDrawio,
	} {
		first, _ := NativeRouteAdapter(route)
		originalID, originalFormat := first.ExportEngine.ID, first.Exports[0]
		first.Exports[0], first.ExportEngine.ID = "corrupted", "corrupted"
		if first.SceneEngine != nil {
			first.SceneEngine.ID = "corrupted"
		}
		second, _ := NativeRouteAdapter(route)
		if second.ExportEngine == first.ExportEngine || second.ExportEngine.ID != originalID || second.Exports[0] != originalFormat {
			t.Fatal("route returned shared mutable export capability")
		}
		if second.SceneEngine != nil && (second.SceneEngine == first.SceneEngine || second.SceneEngine.ID != "levnote.mindmap-preview") {
			t.Fatal("route returned shared mutable scene capability")
		}
	}
	manager, err := plugins.NewManager(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"markdown", "mindmap", "drawio"} {
		if err := manager.SetEnabled(id, false); err != nil {
			t.Fatal(err)
		}
	}
	for _, route := range []plugins.NativeRoute{
		plugins.NativeRouteMarkdown,
		plugins.NativeRouteMindmap,
		plugins.NativeRouteDrawio,
	} {
		if adapter, ok := NativeRouteAdapter(route); !ok || adapter.Create == nil || manager.Enabled(adapter.ID) {
			t.Fatal("fixed capability catalog depends on builtin enablement")
		}
	}
}

func TestInstalledSourceOutlineFactoryRemainsSchema2Only(t *testing.T) {
	syntax := plugins.SyntaxManifest{Schema: 2, Outline: []plugins.OutlineRule{{Prefix: "[", Suffix: "]", Level: 1}}}
	factory := InstalledSourceOutlineFactory("sections", syntax)
	if factory == nil {
		t.Fatal("schema2 lost outline")
	}
	syntax.Outline[0].Prefix = "changed"
	outline := factory(editor.Host{Buffer: text.New("[A]\n[B]\n")})
	defer outline.Release()
	snapshot, err := outline.OutlineSnapshot()
	if err != nil || len(snapshot.Items) != 2 || snapshot.Items[0].Text != "A" || snapshot.Items[1].Text != "B" {
		t.Fatal("factory rules were aliased or ignored", snapshot, err)
	}
	for _, schema := range []int{1, 3} {
		syntax.Schema, syntax.NativeRoute = schema, plugins.NativeRouteMarkdown
		if InstalledSourceOutlineFactory("sections", syntax) != nil {
			t.Fatal("non-schema2 gained source outline")
		}
	}
}
