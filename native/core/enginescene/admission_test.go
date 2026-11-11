package enginescene

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestSceneStringAdmissionBeforeTokenCopy(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		consumed   int
	}{
		{"raw", `"` + strings.Repeat("x", 1<<20) + `"`, 1 + MaxLabelBytes + 1},
		{"escaped", `"` + strings.Repeat(`\u0061`, 100_000) + `"`, 1 + (MaxLabelBytes+1)*6},
		{"astral", `"` + strings.Repeat(`\ud83d\ude80`, 100_000) + `"`, 1 + (MaxLabelBytes/4+1)*12},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := sceneReader{data: []byte(tc.text)}
			value, err := reader.text(MaxLabelBytes)
			if value != "" || !errors.Is(err, ErrLimitExceeded) || reader.pos != tc.consumed {
				t.Fatalf("admission valueLen%d err=%v consumed%d want%d", len(value), err, reader.pos, tc.consumed)
			}
			wire := strings.Replace(literalScene, `"Hello"`, tc.text, 1)
			reader = sceneReader{data: []byte(wire)}
			if err := reader.read(); !errors.Is(err, ErrLimitExceeded) || len(reader.scene.Nodes) != 0 || reader.pos >= len(wire)/2 {
				t.Fatalf("oversize node retained/scanned: nodes%d pos%d err=%v", len(reader.scene.Nodes), reader.pos, err)
			}
			requireDecodeError(t, []byte(wire), ErrLimitExceeded)
		})
	}
	reader := sceneReader{data: []byte(`{"` + strings.Repeat("k", 1<<20) + `":0}`)}
	if err := reader.read(); !errors.Is(err, ErrLimitExceeded) || reader.pos != 12 {
		t.Fatalf("unknown key was not locally bounded: pos%d err=%v", reader.pos, err)
	}
	for _, text := range []string{`"` + strings.Repeat(`\u0061`, MaxLabelBytes) + `"`, `"` + strings.Repeat("a", 512) + `…"`} {
		reader := sceneReader{data: []byte(text)}
		value, err := reader.text(MaxLabelBytes)
		if err != nil || len(value) != MaxLabelBytes || reader.pos != len(text) {
			t.Fatalf("exact decoded bytes were refused: %d %v", len(value), err)
		}
	}
}

func TestSceneIncrementalElementAdmissionAndLateCounts(t *testing.T) {
	for _, tc := range []struct {
		name, prefix, element string
		limit                 int
	}{
		{"nodes", `{"nodes":[`, literalNode, MaxNodes},
		{"edges", `{"edges":[`, `{"from":"a","to":"b","fromPoint":{"x":0,"y":0},"toPoint":{"x":1,"y":1},"stroke":""}`, MaxEdges},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Counts are intentionally absent/late; an invalid excess element
			// must be refused before its opening byte is consumed.
			var text strings.Builder
			text.WriteString(tc.prefix)
			for i := 0; i < tc.limit; i++ {
				if i > 0 {
					text.WriteByte(',')
				}
				text.WriteString(tc.element)
			}
			text.WriteByte(',')
			boundary := text.Len()
			text.WriteString(`{"mustNotParse":[[[[[[[[[[[`)
			wire := []byte(text.String())
			reader := sceneReader{data: wire}
			err := reader.read()
			length, capacity := len(reader.scene.Nodes), cap(reader.scene.Nodes)
			if tc.name == "edges" {
				length, capacity = len(reader.scene.Edges), cap(reader.scene.Edges)
			}
			if !errors.Is(err, ErrLimitExceeded) || reader.pos != boundary || length != tc.limit || capacity != tc.limit {
				t.Fatalf("element admission err=%v pos%d/%d len/cap%d/%d", err, reader.pos, boundary, length, capacity)
			}
		})
	}
	for _, count := range []string{`"nodeCount":100001`, `"edgeCount":100000`} {
		reader := sceneReader{data: []byte(`{` + count + `,"nodes":[[[[[`)}
		if err := reader.read(); !errors.Is(err, ErrLimitExceeded) || len(reader.scene.Nodes) != 0 || len(reader.scene.Edges) != 0 || cap(reader.scene.Nodes) != 0 || cap(reader.scene.Edges) != 0 {
			t.Fatalf("count allocated arrays: %v", err)
		}
	}
	// A valid declared count is still not used as an allocation request.
	reader := sceneReader{data: []byte(`{"nodeCount":100000,"unknown":0}`)}
	if err := reader.read(); !errors.Is(err, ErrInvalidScene) || cap(reader.scene.Nodes) != 0 {
		t.Fatalf("valid declared count allocated: %v", err)
	}
}

func TestScenePayloadAdmissionAndExactEncoderCap(t *testing.T) {
	// Whitespace is legal inside the payload cap. One additional byte must
	// refuse before even the first token/array/string is examined.
	padded := make([]byte, MaxPayloadBytes)
	copy(padded, literalScene)
	for i := len(literalScene); i < len(padded); i++ {
		padded[i] = ' '
	}
	if scene, err := Decode(padded); err != nil || scene.Nodes[0].ID != "root" {
		t.Fatal(err)
	}
	oversized := append(padded, ' ')
	reader := sceneReader{data: oversized}
	if err := reader.read(); !errors.Is(err, ErrLimitExceeded) || reader.pos != 0 || !reflect.DeepEqual(reader.scene, Scene{}) {
		t.Fatalf("payload admission happened too late: %v pos%d", err, reader.pos)
	}
	requireDecodeError(t, oversized, ErrLimitExceeded)

	// Construct a real legal tree whose authored short labels can grow by
	// exactly one output byte each. Adjust those bytes to hit the hard cap.
	scene := chainScene(MaxNodes)
	counter := sceneWriter{limit: MaxPayloadBytes}
	emitScene(&counter, scene)
	if counter.err != nil || counter.written >= MaxPayloadBytes {
		t.Fatal("fixture base exceeds cap")
	}
	remaining := MaxPayloadBytes - counter.written
	for i := range scene.Nodes {
		add := min(remaining, MaxLabelBytes)
		scene.Nodes[i].Label = strings.Repeat("a", add)
		remaining -= add
		if remaining == 0 {
			break
		}
	}
	if remaining != 0 {
		t.Fatal("fixture lacks label capacity")
	}
	data, err := Encode(scene)
	if err != nil || len(data) != MaxPayloadBytes || cap(data) != MaxPayloadBytes {
		t.Fatalf("exact output cap len/cap%d/%d err=%v", len(data), cap(data), err)
	}
	decoded, err := Decode(data)
	if err != nil || len(decoded.Nodes) != MaxNodes || len(decoded.Edges) != MaxEdges || decoded.Nodes[MaxNodes-1].ID != "n99999" {
		t.Fatalf("deep tree/exact cap decode nodes%d edges%d err=%v", len(decoded.Nodes), len(decoded.Edges), err)
	}
	// Extra output remains a locally valid label and legal tree, but cannot
	// return a partial result or allocate the result buffer.
	scene.Nodes[MaxNodes-1].Label += "x"
	requireEncodeError(t, scene, ErrLimitExceeded)
	if len(data) != MaxPayloadBytes || data[len(data)-1] != '}' {
		t.Fatal("previous output was modified")
	}
}

func chainScene(count int) Scene {
	scene := Scene{Schema: 1, Layout: "orgChart", Bounds: Rect{Width: 1, Height: 1}, Nodes: make([]Node, count), Edges: make([]Edge, count-1)}
	for i := range scene.Nodes {
		id := "n" + strconv.Itoa(i)
		scene.Nodes[i] = Node{ID: id, Width: 1, Height: 1, Shape: "rounded", Root: i == 0}
		if i > 0 {
			scene.Edges[i-1] = Edge{FromID: scene.Nodes[i-1].ID, ToID: id}
		}
	}
	return scene
}

func TestSceneNestedEdgeRequiredFields(t *testing.T) {
	const edge = `{"from":"root","to":"child","fromPoint":{"x":1,"y":2},"toPoint":{"x":3,"y":4},"stroke":""}`
	for _, bad := range []string{
		strings.Replace(edge, `"from":"root"`, `"from":"root","from":"root"`, 1),
		strings.Replace(edge, `"stroke":""`, `"stroke":null`, 1),
		strings.Replace(edge, `,"stroke":""`, "", 1),
		strings.Replace(edge, `"toPoint"`, `"ToPoint"`, 1),
		strings.Replace(edge, `"x":1`, `"x":1,"x":1`, 1),
		strings.Replace(edge, `"y":2`, `"z":2`, 1),
		strings.Replace(edge, `,"y":2`, "", 1),
		strings.Replace(edge, `"fromPoint":{"x":1,"y":2}`, `"fromPoint":null`, 1),
		strings.Replace(edge, `"x":3`, `"x":{}`, 1),
	} {
		reader := sceneReader{data: []byte(bad)}
		if _, err := reader.edge(); !errors.Is(err, ErrInvalidScene) {
			t.Fatalf("nested edge accepted: %s %v", bad, err)
		}
	}
	scene := expectedScene()
	scene.Nodes = make([]Node, MaxNodes+1)
	requireEncodeError(t, scene, ErrLimitExceeded)
	scene = expectedScene()
	scene.Edges = make([]Edge, MaxEdges+1)
	requireEncodeError(t, scene, ErrLimitExceeded)
}
