package enginescene

import (
	"bytes"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

// These authored literals are independent of the encoder. They deliberately
// keep all required zero/empty/false values visible in the wire contract.
const literalNode = `{"id":"root","label":"Hello","x":-12.5,"y":0,"width":92,"height":40,"shape":"rounded","fill":"","stroke":"#102030ff","text":"","root":true,"collapsed":false}`
const literalScene = `{"schema":1,"layout":"logicalStructureRight","nodeCount":1,"edgeCount":0,"bounds":{"x":-24,"y":0,"width":196,"height":144},"nodes":[` + literalNode + `],"edges":[]}`

func expectedScene() Scene {
	return Scene{
		Schema: 1, Layout: "logicalStructureRight", Bounds: Rect{-24, 0, 196, 144},
		Nodes: []Node{{ID: "root", Label: "Hello", X: -12.5, Width: 92, Height: 40,
			Shape: "rounded", Stroke: "#102030ff", Root: true}},
		Edges: []Edge{},
	}
}

func requireDecodeError(t *testing.T, input []byte, want error) {
	t.Helper()
	got, err := Decode(input)
	if !errors.Is(err, want) || !reflect.DeepEqual(got, Scene{}) {
		t.Fatalf("decode error=%v scene=%+v; want %v and zero scene", err, got, want)
	}
}

func requireEncodeError(t *testing.T, scene Scene, want error) {
	t.Helper()
	data, err := Encode(scene)
	if !errors.Is(err, want) || data != nil {
		t.Fatalf("encode bytes=%d error=%v; want nil and %v", len(data), err, want)
	}
}

func TestSceneLiteralDeterministicAndOwned(t *testing.T) {
	input := []byte(literalScene)
	decoded, err := Decode(input)
	if err != nil || !reflect.DeepEqual(decoded, expectedScene()) {
		t.Fatalf("literal decode=%+v err=%v", decoded, err)
	}
	clear(input)
	if !reflect.DeepEqual(decoded, expectedScene()) {
		t.Fatal("decoded scene aliases caller payload")
	}
	before := expectedScene()
	for i := 0; i < 2; i++ {
		data, err := Encode(decoded)
		if err != nil || string(data) != literalScene || len(data) != cap(data) {
			t.Fatalf("literal encoding=%s len/cap=%d/%d err=%v", data, len(data), cap(data), err)
		}
		data[0] = '!'
		if !reflect.DeepEqual(decoded, before) {
			t.Fatal("encoder modified caller scene")
		}
	}
}

func TestSceneLiteralArbitraryOrderEscapesAndTree(t *testing.T) {
	const input = ` 
{"edges":[{"stroke":"","toPoint":{"y":-2,"x":3},"fromPoint":{"y":0,"x":1},"to":"kid","from":"r\u006fot"}],
"nodes":[{"collapsed":false,"root":true,"text":"#ffffffff","stroke":"","fill":"#aabbcc00","shape":"rounded","height":40,"width":92,"y":0,"x":-12.5,"label":"\u4e2d\ud83d\ude80\n\t\b\f\r\"\\\/\u0000<>&","id":"root"},
{"label":"…","id":"kid","root":false,"collapsed":true,"text":"","stroke":"","fill":"","shape":"rounded","width":1,"height":1,"x":3,"y":-2}],
"bounds":{"height":144,"width":196,"y":0,"x":-24},"edgeCount":1.0,"nodeCount":2e0,"layout":"orgChart","schema":1e0} 	
`
	want := Scene{Schema: 1, Layout: "orgChart", Bounds: Rect{-24, 0, 196, 144},
		Nodes: []Node{
			{ID: "root", Label: "中🚀\n\t\b\f\r\"\\/\x00<>&", X: -12.5, Width: 92, Height: 40, Shape: "rounded", Fill: "#aabbcc00", Text: "#ffffffff", Root: true},
			{ID: "kid", Label: "…", X: 3, Y: -2, Width: 1, Height: 1, Shape: "rounded", Collapsed: true},
		},
		Edges: []Edge{{FromID: "root", ToID: "kid", From: Point{1, 0}, To: Point{3, -2}}},
	}
	decoded, err := Decode([]byte(input))
	if err != nil || !reflect.DeepEqual(decoded, want) {
		t.Fatalf("arbitrary order/escape tree=%+v err=%v", decoded, err)
	}
	const expected = `{"schema":1,"layout":"orgChart","nodeCount":2,"edgeCount":1,"bounds":{"x":-24,"y":0,"width":196,"height":144},"nodes":[{"id":"root","label":"中🚀\n\t\b\f\r\"\\/\u0000<>&","x":-12.5,"y":0,"width":92,"height":40,"shape":"rounded","fill":"#aabbcc00","stroke":"","text":"#ffffffff","root":true,"collapsed":false},{"id":"kid","label":"…","x":3,"y":-2,"width":1,"height":1,"shape":"rounded","fill":"","stroke":"","text":"","root":false,"collapsed":true}],"edges":[{"from":"root","to":"kid","fromPoint":{"x":1,"y":0},"toPoint":{"x":3,"y":-2},"stroke":""}]}`
	encoded, err := Encode(want)
	if err != nil || string(encoded) != expected {
		t.Fatalf("independent escaped encoding=%s err=%v", encoded, err)
	}
}

func TestSceneLayoutsAndExactStringBudgets(t *testing.T) {
	for _, layout := range []string{"logicalStructureLeft", "logicalStructureRight", "logicalStructure", "orgChart"} {
		t.Run(layout, func(t *testing.T) {
			wire := strings.Replace(literalScene, "logicalStructureRight", layout, 1)
			got, err := Decode([]byte(wire))
			if err != nil || got.Layout != layout {
				t.Fatalf("layout=%q err=%v", got.Layout, err)
			}
		})
	}
	for _, layout := range []string{"", "logicalstructure", "LogicalStructure", "unknown"} {
		requireDecodeError(t, []byte(strings.Replace(literalScene, "logicalStructureRight", layout, 1)), ErrInvalidScene)
	}
	for _, label := range []string{strings.Repeat("a", 512) + "…", strings.Repeat("中", 171) + "ab", ""} {
		scene := expectedScene()
		scene.Nodes[0].ID = strings.Repeat("中", 170) + "ab"
		scene.Nodes[0].Label = label
		data, err := Encode(scene)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Decode(data)
		if err != nil || got.Nodes[0].ID != scene.Nodes[0].ID || got.Nodes[0].Label != label {
			t.Fatalf("exact string boundaries err=%v", err)
		}
	}
	for _, change := range []struct{ old, value string }{
		{"Hello", strings.Repeat("x", MaxLabelBytes+1)},
		{"root", strings.Repeat("i", MaxIDBytes+1)},
		{"logicalStructureRight", strings.Repeat("l", MaxLayoutBytes+1)},
	} {
		requireDecodeError(t, []byte(strings.Replace(literalScene, change.old, change.value, 1)), ErrLimitExceeded)
	}
	for _, paint := range []string{"#00000000", "#abcdef09", "#ffffffff", ""} {
		scene := expectedScene()
		scene.Nodes[0].Fill, scene.Nodes[0].Stroke, scene.Nodes[0].Text = paint, paint, paint
		if _, err := Encode(scene); err != nil {
			t.Fatalf("valid paint %q: %v", paint, err)
		}
	}
	for _, paint := range []string{"#ABCDEF09", "#fff", "red", "#abcdex00", " #12345678", "#12345678 "} {
		wire := strings.Replace(literalScene, "#102030ff", paint, 1)
		want := ErrInvalidScene
		if len(paint) > 9 {
			want = ErrLimitExceeded
		}
		requireDecodeError(t, []byte(wire), want)
	}
}

func TestSceneInvalidSchemasAndShapes(t *testing.T) {
	cases := map[string]string{
		"unknown":             strings.Replace(literalScene, `"schema":1`, `"extra":0,"schema":1`, 1),
		"duplicate":           strings.Replace(literalScene, `"schema":1`, `"schema":1,"schema":1`, 1),
		"escaped-duplicate":   strings.Replace(literalScene, `"schema":1`, `"schema":1,"sch\u0065ma":1`, 1),
		"case":                strings.Replace(literalScene, `"schema":1`, `"Schema":1`, 1),
		"missing":             strings.Replace(literalScene, `"layout":"logicalStructureRight",`, "", 1),
		"schema-version":      strings.Replace(literalScene, `"schema":1`, `"schema":2`, 1),
		"schema-rounding":     strings.Replace(literalScene, `"schema":1`, `"schema":1.00000000000000001`, 1),
		"node-count-mismatch": strings.Replace(literalScene, `"nodeCount":1`, `"nodeCount":2`, 1),
		"edge-count-mismatch": strings.Replace(literalScene, `"edgeCount":0`, `"edgeCount":1`, 1),
		"node-null":           strings.Replace(literalScene, literalNode, "null", 1),
		"nodes-null":          strings.Replace(literalScene, "["+literalNode+"]", "null", 1),
		"edges-null":          strings.Replace(literalScene, `"edges":[]`, `"edges":null`, 1),
		"bounds-null":         strings.Replace(literalScene, `"bounds":{"x":-24,"y":0,"width":196,"height":144}`, `"bounds":null`, 1),
		"paint-null":          strings.Replace(literalScene, `"fill":""`, `"fill":null`, 1),
		"paint-missing":       strings.Replace(literalScene, `"fill":"",`, "", 1),
		"node-duplicate":      strings.Replace(literalScene, `"id":"root"`, `"id":"root","id":"root"`, 1),
		"node-case":           strings.Replace(literalScene, `"root":true`, `"Root":true`, 1),
		"node-unknown":        strings.Replace(literalScene, `"root":true`, `"root":true,"image":0`, 1),
		"node-nested":         strings.Replace(literalScene, `"label":"Hello"`, `"label":{"label":"Hello"}`, 1),
		"bounds-duplicate":    strings.Replace(literalScene, `"x":-24`, `"x":-24,"x":0`, 1),
		"bounds-missing":      strings.Replace(literalScene, `"x":-24,`, "", 1),
		"bounds-unknown":      strings.Replace(literalScene, `"x":-24`, `"z":0,"x":-24`, 1),
		"bool-number":         strings.Replace(literalScene, `"root":true`, `"root":1`, 1),
		"empty-id":            strings.Replace(literalScene, `"id":"root"`, `"id":""`, 1),
		"shape":               strings.Replace(literalScene, `"shape":"rounded"`, `"shape":"ellipse"`, 1),
		"trailing-comma":      strings.Replace(literalScene, `"edges":[]}`, `"edges":[],}`, 1),
		"array-object":        strings.Replace(literalScene, `"edges":[]`, `"edges":{}`, 1),
		"second-object":       literalScene + "{}",
		"BOM":                 "\ufeff" + literalScene,
		"non-json-space":      literalScene + "\v",
	}
	for name, wire := range cases {
		t.Run(name, func(t *testing.T) { requireDecodeError(t, []byte(wire), ErrInvalidScene) })
	}
	for i := 0; i < len(literalScene); i++ {
		requireDecodeError(t, []byte(literalScene[:i]), ErrInvalidScene)
	}
	for _, label := range []string{`\ud800`, `\udfff`, `\ud800\u0041`, `\ud800x`, `\u12xz`, `\q`, "raw\nline", string([]byte{0xff}), string([]byte{0xc0, 0x80})} {
		requireDecodeError(t, []byte(strings.Replace(literalScene, "Hello", label, 1)), ErrInvalidScene)
	}
	for _, input := range []string{`null`, `[]`, `"scene"`, `{`, ``, `{"schema":[[[[[[[[[[`} {
		requireDecodeError(t, []byte(input), ErrInvalidScene)
	}
}

func treeScene() Scene {
	scene := expectedScene()
	scene.Nodes = append(scene.Nodes, Node{ID: "a", Width: 1, Height: 1, Shape: "rounded"}, Node{ID: "b", Width: 1, Height: 1, Shape: "rounded"})
	scene.Edges = []Edge{{FromID: "root", ToID: "a"}, {FromID: "a", ToID: "b"}}
	return scene
}

func TestSceneTreeAndProgrammaticRefusals(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Scene)
		want   error
	}{
		{"zero", func(s *Scene) { *s = Scene{} }, ErrInvalidScene},
		{"duplicate-id", func(s *Scene) { s.Nodes[2].ID = "a" }, ErrInvalidScene},
		{"two-roots", func(s *Scene) { s.Nodes[1].Root = true }, ErrInvalidScene},
		{"no-root", func(s *Scene) { s.Nodes[0].Root = false }, ErrInvalidScene},
		{"missing-from", func(s *Scene) { s.Edges[0].FromID = "missing" }, ErrInvalidScene},
		{"missing-to", func(s *Scene) { s.Edges[0].ToID = "missing" }, ErrInvalidScene},
		{"self", func(s *Scene) { s.Edges[0].ToID = "root" }, ErrInvalidScene},
		{"root-parent", func(s *Scene) { s.Edges[0] = Edge{FromID: "a", ToID: "root"} }, ErrInvalidScene},
		{"two-parents", func(s *Scene) { s.Edges[1].ToID = "a" }, ErrInvalidScene},
		{"disconnected-cycle", func(s *Scene) { s.Edges = []Edge{{FromID: "a", ToID: "b"}, {FromID: "b", ToID: "a"}} }, ErrInvalidScene},
		{"missing-edge", func(s *Scene) { s.Edges = s.Edges[:1] }, ErrInvalidScene},
		{"extra-edge", func(s *Scene) { s.Edges = append(s.Edges, Edge{}) }, ErrInvalidScene},
		{"nan", func(s *Scene) { s.Bounds.X = math.NaN() }, ErrInvalidScene},
		{"inf", func(s *Scene) { s.Edges[0].From.Y = math.Inf(1) }, ErrInvalidScene},
		{"zero-bound-width", func(s *Scene) { s.Bounds.Width = 0 }, ErrInvalidScene},
		{"negative-height", func(s *Scene) { s.Nodes[0].Height = -1 }, ErrInvalidScene},
		{"coordinate-limit", func(s *Scene) { s.Nodes[0].X = MaxCoordinateAbs + 1 }, ErrLimitExceeded},
		{"dimension-limit", func(s *Scene) { s.Bounds.Height = MaxDimension + 1 }, ErrLimitExceeded},
		{"id-limit", func(s *Scene) { s.Nodes[0].ID = strings.Repeat("i", MaxIDBytes+1) }, ErrLimitExceeded},
		{"label-limit", func(s *Scene) { s.Nodes[0].Label = strings.Repeat("l", MaxLabelBytes+1) }, ErrLimitExceeded},
		{"layout-limit", func(s *Scene) { s.Layout = strings.Repeat("l", MaxLayoutBytes+1) }, ErrLimitExceeded},
		{"bad-utf8", func(s *Scene) { s.Nodes[0].Label = string([]byte{0xff}) }, ErrInvalidScene},
		{"shape", func(s *Scene) { s.Nodes[0].Shape = "ellipse" }, ErrInvalidScene},
		{"paint", func(s *Scene) { s.Edges[0].Stroke = "#ABCDEFff" }, ErrInvalidScene},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scene := treeScene()
			tc.mutate(&scene)
			requireEncodeError(t, scene, tc.want)
			// For JSON-representable invalid inputs, bypass validation only
			// in the test to prove public Decode's independent refusal too.
			if tc.name == "nan" || tc.name == "inf" || tc.name == "bad-utf8" {
				return
			}
			wire := sceneWriter{limit: MaxPayloadBytes, data: make([]byte, 8192)}
			emitScene(&wire, scene)
			if wire.err != nil || wire.written > len(wire.data) {
				t.Fatal("invalid-fixture serialization unexpectedly failed")
			}
			requireDecodeError(t, wire.data[:wire.written], tc.want)
		})
	}
	// Sibling order and opaque Unicode/control IDs are not identity normalization.
	scene := treeScene()
	scene.Nodes[0], scene.Nodes[2] = scene.Nodes[2], scene.Nodes[0]
	data, err := Encode(scene)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if err != nil || got.Nodes[2].ID != "root" {
		t.Fatalf("root need not be first: %v", err)
	}
	scene = expectedScene()
	scene.Nodes[0].ID = "\x00中/.. opaque"
	data, err = Encode(scene)
	if err != nil || !bytes.Contains(data, []byte(`"id":"\u0000中/.. opaque"`)) {
		t.Fatalf("opaque ID changed: %q %v", data, err)
	}
}
