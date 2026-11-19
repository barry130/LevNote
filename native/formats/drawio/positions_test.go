package drawio

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

const positionPair = `<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" vertex="1" parent="1" style="rounded=1;"><mxGeometry x="-10" y="20" width="30" height="10" as="geometry"/></mxCell><mxCell id="b" vertex="1" parent="1" style="ellipse;"><mxGeometry x="40" y="-5" width="10" height="20" as="geometry"/></mxCell></root></mxGraphModel>`

func TestUpdatePositionsChangesOnlyRequestedAxes(t *testing.T) {
	source := `<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" vertex="1" parent="1" future="keep"><mxGeometry x='1e1' y="20.00" width="0030" height='1e1' as="geometry" note="opaque"/></mxCell><mxCell id="b" vertex="1" parent="1"><mxGeometry x="40" y="-5" width="10" height="20" as="geometry"/></mxCell></root></mxGraphModel>`
	want := `<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" vertex="1" parent="1" future="keep"><mxGeometry x='12.5' y="20.00" width="0030" height='1e1' as="geometry" note="opaque"/></mxCell><mxCell id="b" vertex="1" parent="1"><mxGeometry x="40" y="-12.25" width="10" height="20" as="geometry"/></mxCell></root></mxGraphModel>`
	document, _, err := Parse([]byte(source), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := document.UpdatePositions(0, []PositionUpdate{{CellID: "b", X: 40, Y: -12.25}, {CellID: "a", X: 12.5, Y: 20}}); err != nil {
		t.Fatal(err)
	}
	got, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want || !document.Dirty() || !bytes.Equal(document.OriginalBytes(), []byte(source)) {
		t.Fatalf("batch position fidelity: dirty=%v\ngot  %s\nwant %s", document.Dirty(), got, want)
	}
}

func TestPlanAlignmentSixDirectionsUsesSelectionBounds(t *testing.T) {
	cases := []struct {
		name      string
		alignment Alignment
		want      []PositionUpdate
	}{
		{"left", AlignLeft, []PositionUpdate{{CellID: "b", X: -10, Y: -5}}},
		{"center-x", AlignCenterX, []PositionUpdate{{CellID: "a", X: 5, Y: 20}, {CellID: "b", X: 15, Y: -5}}},
		{"right", AlignRight, []PositionUpdate{{CellID: "a", X: 20, Y: 20}}},
		{"top", AlignTop, []PositionUpdate{{CellID: "a", X: -10, Y: -5}}},
		{"center-y", AlignCenterY, []PositionUpdate{{CellID: "a", X: -10, Y: 7.5}, {CellID: "b", X: 40, Y: 2.5}}},
		{"bottom", AlignBottom, []PositionUpdate{{CellID: "b", X: 40, Y: 10}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document := parsePositionTest(t, positionPair, ParseOptions{})
			before := positionSnapshot(t, document)
			if err := document.ValidatePositionTargets(0, []string{"b", "a"}); err != nil {
				t.Fatal(err)
			}
			plan, err := document.PlanAlignment(0, []string{"a", "b"}, tc.alignment)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(plan, tc.want) {
				t.Fatalf("plan=%#v, want %#v", plan, tc.want)
			}
			assertPositionSnapshot(t, document, before)
			reversed, err := document.PlanAlignment(0, []string{"b", "a"}, tc.alignment)
			if err != nil {
				t.Fatal(err)
			}
			for i := range plan {
				if reversed[len(reversed)-1-i] != plan[i] {
					t.Fatal("selection order changed alignment coordinates")
				}
			}
			if err := document.UpdatePositions(0, plan); err != nil {
				t.Fatal(err)
			}
			for _, want := range tc.want {
				node := findNode(document.Pages[0].Model.Nodes, want.CellID)
				if node == nil || node.X != want.X || node.Y != want.Y {
					t.Fatalf("node=%#v, want %#v", node, want)
				}
			}
			after := positionSnapshot(t, document)
			again, err := document.PlanAlignment(0, []string{"b", "a"}, tc.alignment)
			if err != nil || len(again) != 0 {
				t.Fatalf("second plan=%#v, err=%v", again, err)
			}
			if err := document.UpdatePositions(0, again); err != nil {
				t.Fatal(err)
			}
			assertPositionSnapshot(t, document, after)
		})
	}
}

func TestPositionsNoOpPreservesLexemesOmissionsAndDirtyState(t *testing.T) {
	source := `<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" vertex="1" parent="1"><mxGeometry y='-0' width="3e1" height="10.00" as="geometry"/></mxCell><mxCell id="b" vertex="1" parent="1"><mxGeometry x="0.000" y="0e5" width="10" height="20" as="geometry"/></mxCell></root></mxGraphModel>`
	for _, dirty := range []bool{false, true} {
		t.Run(map[bool]string{false: "clean", true: "dirty"}[dirty], func(t *testing.T) {
			document := parsePositionTest(t, source, ParseOptions{})
			if dirty {
				if err := document.UpdateText(0, "a", "changed label"); err != nil {
					t.Fatal(err)
				}
			}
			before := positionSnapshot(t, document)
			if err := document.UpdatePositions(0, nil); err != nil {
				t.Fatal(err)
			}
			if err := document.UpdatePositions(0, []PositionUpdate{{CellID: "a", X: 0, Y: 0}, {CellID: "b", X: 0, Y: 0}}); err != nil {
				t.Fatal(err)
			}
			plan, err := document.PlanAlignment(0, []string{"a", "b"}, AlignLeft)
			if err != nil || len(plan) != 0 {
				t.Fatalf("no-op plan=%#v err=%v", plan, err)
			}
			assertPositionSnapshot(t, document, before)
		})
	}
	document := parsePositionTest(t, source, ParseOptions{})
	if err := document.UpdatePositions(0, []PositionUpdate{{CellID: "a", X: 7.25, Y: 0}}); err != nil {
		t.Fatal(err)
	}
	want := `<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" vertex="1" parent="1"><mxGeometry y='-0' width="3e1" height="10.00" as="geometry" x="7.25"/></mxCell><mxCell id="b" vertex="1" parent="1"><mxGeometry x="0.000" y="0e5" width="10" height="20" as="geometry"/></mxCell></root></mxGraphModel>`
	if got := positionBytes(t, document); got != want {
		t.Fatalf("insert only missing changed axis:\ngot %s\nwant %s", got, want)
	}
}

func TestPositionsUsePrivateSourceAndPreserveOwnership(t *testing.T) {
	input := []byte(positionPair)
	document, _, err := Parse(input, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	input[0] = '!'
	document.Source[0] = '?'
	document.Pages[0].Model.Nodes[0].X = 900
	document.Pages[0].Model.Nodes[0].Width = 999
	before := positionSnapshot(t, document)
	plan, err := document.PlanAlignment(0, []string{"a", "b"}, AlignLeft)
	if err != nil {
		t.Fatal(err)
	}
	wantPlan := []PositionUpdate{{CellID: "b", X: -10, Y: -5}}
	if !reflect.DeepEqual(plan, wantPlan) {
		t.Fatalf("plan used exposed model: %#v", plan)
	}
	assertPositionSnapshot(t, document, before)
	if err := document.UpdatePositions(0, plan); err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(positionPair, `x="40"`, `x="-10"`, 1)
	if got := positionBytes(t, document); got != want {
		t.Fatalf("private source changed unexpectedly: %s", got)
	}
	if string(document.OriginalBytes()) != positionPair {
		t.Fatal("lost original input bytes")
	}
	document.Source[0] = '!'
	exposed, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	exposed[0] = '!'
	original := document.OriginalBytes()
	original[0] = '!'
	if positionBytes(t, document) != want || string(document.OriginalBytes()) != positionPair {
		t.Fatal("exposed bytes alias owned buffers")
	}
}

func TestPositionsBadTargetInEveryRequestSlotIsAtomic(t *testing.T) {
	for _, dirty := range []bool{false, true} {
		for _, ids := range [][]string{{"missing", "a", "b"}, {"a", "missing", "b"}, {"a", "b", "missing"}, {"a", "a", "b"}, {"a", "b", "b"}, {"a", "0", "b"}, {"a", "1", "b"}, {"a", "", "b"}} {
			t.Run(strings.Join(ids, "_")+map[bool]string{false: "_clean", true: "_dirty"}[dirty], func(t *testing.T) {
				document := parsePositionTest(t, positionPair, ParseOptions{})
				if dirty {
					if err := document.UpdateText(0, "a", "prior edit"); err != nil {
						t.Fatal(err)
					}
				}
				before := positionSnapshot(t, document)
				if err := document.ValidatePositionTargets(0, ids); err == nil {
					t.Fatal("invalid targets validated")
				}
				if plan, err := document.PlanAlignment(0, ids, AlignLeft); err == nil || len(plan) != 0 {
					t.Fatalf("invalid plan=%#v err=%v", plan, err)
				}
				updates := make([]PositionUpdate, len(ids))
				for i, id := range ids {
					updates[i] = PositionUpdate{CellID: id, X: 123, Y: 456}
				}
				if err := document.UpdatePositions(0, updates); err == nil {
					t.Fatal("invalid batch committed")
				}
				assertPositionSnapshot(t, document, before)
				if err := document.UpdatePositions(0, []PositionUpdate{{CellID: "a", X: -11, Y: 20}}); err != nil {
					t.Fatalf("valid edit after failure: %v", err)
				}
			})
		}
	}
}

func TestPositionsRejectAmbiguousAndTransformedSource(t *testing.T) {
	geom := `<mxGeometry x="40" y="-5" width="10" height="20" as="geometry"/>`
	cell := `<mxCell id="b" vertex="1" parent="1" style="ellipse;">`
	replace := func(old, new string) string { return strings.Replace(positionPair, old, new, 1) }
	geometry := func(text string) string { return replace(geom, text) }
	badGeometry := func(old, new string) string { return geometry(strings.Replace(geom, old, new, 1)) }
	badCell := func(old, new string) string { return replace(cell, strings.Replace(cell, old, new, 1)) }
	cases := []struct {
		name, source string
		want         error
	}{
		{"missing-geometry", geometry(""), ErrGeometryMissing},
		{"duplicate-geometry", geometry(geom + geom), ErrUnsupported},
		{"empty-x", badGeometry(`x="40"`, `x=""`), ErrInvalidGeometry},
		{"bad-x", badGeometry(`x="40"`, `x="bad"`), ErrInvalidGeometry},
		{"bad-y", badGeometry(`y="-5"`, `y="garbage"`), ErrInvalidGeometry},
		{"nan-x", badGeometry(`x="40"`, `x="NaN"`), ErrInvalidGeometry},
		{"infinite-y", badGeometry(`y="-5"`, `y="+Inf"`), ErrInvalidGeometry},
		{"overflow-x", badGeometry(`x="40"`, `x="1e309"`), ErrInvalidGeometry},
		{"zero-width", badGeometry(`width="10"`, `width="0"`), ErrInvalidGeometry},
		{"negative-height", badGeometry(`height="20"`, `height="-1"`), ErrInvalidGeometry},
		{"missing-width", badGeometry(`width="10"`, ""), ErrInvalidGeometry},
		{"empty-height", badGeometry(`height="20"`, `height=""`), ErrInvalidGeometry},
		{"extent-overflow", geometry(`<mxGeometry x="1e308" y="0" width="1e308" height="20" as="geometry"/>`), ErrInvalidGeometry},
		{"relative", badGeometry(`as="geometry"`, `as="geometry" relative="1"`), ErrUnsupported},
		{"bad-relative", badGeometry(`as="geometry"`, `as="geometry" relative="maybe"`), ErrUnsupported},
		{"wrong-role", badGeometry(`as="geometry"`, `as="offset"`), ErrUnsupported},
		{"offset-point", geometry(strings.TrimSuffix(geom, "/>") + `><mxPoint x="1" y="2" as="offset"/></mxGeometry>`), ErrUnsupported},
		{"alternate-bounds", geometry(strings.TrimSuffix(geom, "/>") + `><mxRectangle x="1" as="alternateBounds"/></mxGeometry>`), ErrUnsupported},
		{"unknown-geometry-child", geometry(strings.TrimSuffix(geom, "/>") + `><future transform="mystery"/></mxGeometry>`), ErrUnsupported},
		{"geometry-transform-attribute", badGeometry(`as="geometry"`, `as="geometry" transform="matrix(1,0,0,1,4,5)"`), ErrUnsupported},
		{"wrapped-geometry", geometry("<object>" + geom + "</object>"), ErrUnsupported},
		{"namespaced-geometry", geometry(strings.Replace(geom, "mxGeometry", `p:mxGeometry xmlns:p="urn:test"`, 1)), ErrUnsupported},
		{"namespaced-x", badGeometry(`x="40"`, `xmlns:p="urn:test" p:x="40"`), ErrUnsupported},
		{"x-local-collision", badGeometry(`x="40"`, `xmlns:p="urn:test" p:x="99" x="40"`), ErrUnsupported},
		{"y-local-collision", badGeometry(`y="-5"`, `xmlns:p="urn:test" y="-5" p:y="99"`), ErrUnsupported},
		{"x-namespace-declaration", badGeometry(`x="40"`, `xmlns:x="urn:test" x="40"`), ErrUnsupported},
		{"namespaced-width", badGeometry(`width="10"`, `xmlns:p="urn:test" p:width="10"`), ErrUnsupported},
		{"namespaced-role", badGeometry(`as="geometry"`, `xmlns:p="urn:test" p:as="geometry"`), ErrUnsupported},
		{"namespaced-id", badCell(`id="b"`, `xmlns:p="urn:test" p:id="b"`), ErrUnsupported},
		{"id-local-collision", badCell(`id="b"`, `xmlns:p="urn:test" id="b" p:id="other"`), ErrUnsupported},
		{"style-local-collision", badCell(`style="ellipse;"`, `xmlns:p="urn:test" p:style="shape=image;" style="ellipse;"`), ErrUnsupported},
		{"namespaced-parent", badCell(`parent="1"`, `xmlns:p="urn:test" p:parent="1"`), ErrUnsupported},
		{"child-vertex", badCell(`parent="1"`, `parent="a"`), ErrUnsupported},
		{"edge-and-vertex", badCell(`vertex="1"`, `vertex="1" edge="1"`), ErrUnsupported},
		{"edge-only", badCell(`vertex="1"`, `edge="1"`), ErrUnsupported},
		{"collapsed", badCell(`vertex="1"`, `vertex="1" collapsed="1"`), ErrUnsupported},
		{"rotation", badCell(`style="ellipse;"`, `style="ellipse;rotation=45;"`), ErrUnsupported},
		{"invalid-rotation", badCell(`style="ellipse;"`, `style="ellipse;rotation=NaN;"`), ErrUnsupported},
		{"flip", badCell(`style="ellipse;"`, `style="ellipse;flipH=1;"`), ErrUnsupported},
		{"swimlane", badCell(`style="ellipse;"`, `style="swimlane;"`), ErrUnsupported},
		{"group", badCell(`style="ellipse;"`, `style="group;"`), ErrUnsupported},
		{"child-layout", badCell(`style="ellipse;"`, `style="ellipse;childLayout=stackLayout;"`), ErrUnsupported},
		{"unknown-shape", badCell(`style="ellipse;"`, `style="shape=hexagon;"`), ErrInvalidShape},
		{"image", badCell(`style="ellipse;"`, `style="shape=image;image=data:fake;"`), ErrInvalidShape},
		{"stencil", badCell(`style="ellipse;"`, `style="shape=mxgraph.custom.foo;"`), ErrInvalidShape},
		{"duplicate-shape-style", badCell(`style="ellipse;"`, `style="shape=rectangle;shape=ellipse;"`), ErrUnsupported},
		{"noncanonical-root", replace(`<mxCell id="0"/>`, `<mxCell id="0" vertex="1"/>`), ErrUnsupported},
		{"noncanonical-layer", replace(`<mxCell id="1" parent="0"/>`, `<mxCell id="1" parent="0" style="group;"/>`), ErrUnsupported},
		{"layer-content", replace(`<mxCell id="1" parent="0"/>`, `<mxCell id="1" parent="0"><future/></mxCell>`), ErrUnsupported},
		{"layer-instruction", replace(`<mxCell id="1" parent="0"/>`, `<mxCell id="1" parent="0"><?extension test?></mxCell>`), ErrUnsupported},
		{"missing-root-cell", replace(`<mxCell id="0"/>`, ""), ErrUnsupported},
		{"inherited-model-namespace", `<mxfile xmlns="urn:test"><diagram>` + positionPair + `</diagram></mxfile>`, ErrUnsupported},
		{"namespaced-root", replace("<root>", `<root xmlns="urn:test">`), ErrUnsupported},
		{"wrapped-child", replace("</root>", `<object><mxCell id="child" parent="b" vertex="1"/></object></root>`), ErrUnsupported},
		{"hidden-duplicate-id", replace("</root>", `<object><mxCell id="b" parent="1" vertex="1"/></object></root>`), ErrDuplicateID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document := parsePositionTest(t, tc.source, ParseOptions{})
			before := positionSnapshot(t, document)
			for _, ids := range [][]string{{"a", "b"}, {"b", "a"}} {
				if err := document.ValidatePositionTargets(0, ids); !errors.Is(err, tc.want) {
					t.Fatalf("validate=%v want %v", err, tc.want)
				}
				if plan, err := document.PlanAlignment(0, ids, AlignCenterX); !errors.Is(err, tc.want) || len(plan) != 0 {
					t.Fatalf("plan=%#v err=%v want %v", plan, err, tc.want)
				}
				updates := []PositionUpdate{{CellID: ids[0], X: 100, Y: 100}, {CellID: ids[1], X: 200, Y: 200}}
				if err := document.UpdatePositions(0, updates); !errors.Is(err, tc.want) {
					t.Fatalf("update=%v want %v", err, tc.want)
				}
				assertPositionSnapshot(t, document, before)
			}
		})
	}
}

func TestPositionsDuplicateRawAttributesStayParserErrors(t *testing.T) {
	for _, pair := range [][2]string{
		{`x="40"`, `x="40" x="99"`},
		{`y="-5"`, `y="-5" y="99"`},
		{`id="b"`, `id="b" id="other"`},
	} {
		source := strings.Replace(positionPair, pair[0], pair[1], 1)
		document, _, err := Parse([]byte(source), ParseOptions{})
		if !errors.Is(err, ErrInvalidXML) || document != nil {
			t.Fatalf("duplicate attribute parse: doc=%v err=%v", document, err)
		}
	}
}

func TestPositionsPreserveUnknownXMLAndCompressedOtherPages(t *testing.T) {
	payload := compressRawDeflate(positionPair)
	prefix := "<?xml version='1.0'?>\r\n" + `<mxfile vendor="opaque"><diagram id="zip" name="Zip">` + payload + `</diagram><diagram id="edit" name="Edit">`
	extension := `<future data="0009"><nested><![CDATA[a < b & c]]></nested></future>`
	editable := strings.Replace(positionPair, `style="rounded=1;">`, `style="rounded=1;whiteSpace=wrap;html=1;" data-extra="000">`+extension, 1)
	editable = strings.Replace(editable, "</root>", `<!-- retained --><mxCell id="edge" parent="1" edge="1" source="a" target="b"><mxGeometry relative="1" as="geometry"/></mxCell><mxCell id="other" parent="1" vertex="1" style="shape=unknown;"><mxGeometry x="garbage" width="0" height="1" as="geometry"/></mxCell>`+extension+"</root>", 1)
	suffix := `</diagram><diagram id="last" name="Other">` + positionPair + `</diagram><unknown flag="unchanged"/></mxfile>` + "\r\n"
	source := prefix + editable + suffix
	document := parsePositionTest(t, source, ParseOptions{})
	before := positionSnapshot(t, document)
	if err := document.ValidatePositionTargets(0, []string{"a"}); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("compressed validate=%v", err)
	}
	if _, err := document.PlanAlignment(0, []string{"a", "b"}, AlignLeft); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("compressed plan=%v", err)
	}
	if err := document.UpdatePositions(0, []PositionUpdate{{CellID: "a", X: 5, Y: 5}}); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("compressed update=%v", err)
	}
	assertPositionSnapshot(t, document, before)
	if err := document.UpdatePositions(1, []PositionUpdate{{CellID: "b", X: 4.5, Y: -5}, {CellID: "a", X: -10, Y: 22.5}}); err != nil {
		t.Fatal(err)
	}
	want := prefix + strings.Replace(strings.Replace(editable, `x="40"`, `x="4.5"`, 1), `y="20"`, `y="22.5"`, 1) + suffix
	if got := positionBytes(t, document); got != want {
		t.Fatalf("non-target XML changed:\ngot %s\nwant %s", got, want)
	}
	reopened := parsePositionTest(t, want, ParseOptions{})
	if len(reopened.Pages) != 3 || !reopened.Pages[0].Compressed || findNode(reopened.Pages[2].Model.Nodes, "b").X != 40 {
		t.Fatal("other pages changed")
	}
}

func TestPositionsCandidateLimitsFailWithoutChangingDocument(t *testing.T) {
	t.Run("bytes", func(t *testing.T) {
		document := parsePositionTest(t, positionPair, ParseOptions{MaxBytes: len(positionPair)})
		before := positionSnapshot(t, document)
		err := document.UpdatePositions(0, []PositionUpdate{{CellID: "a", X: 100000, Y: 20}, {CellID: "b", X: 400000, Y: -5}})
		if !errors.Is(err, ErrLimitExceeded) {
			t.Fatalf("byte limit=%v", err)
		}
		assertPositionSnapshot(t, document, before)
		if err := document.UpdatePositions(0, []PositionUpdate{{CellID: "a", X: -11, Y: 20}}); err != nil {
			t.Fatalf("edit after failure=%v", err)
		}
	})
	t.Run("reparse-attributes", func(t *testing.T) {
		source := strings.Replace(positionPair, `x="-10" y="20" `, "", 1)
		source = strings.Replace(source, `x="40" `, "", 1)
		document := parsePositionTest(t, source, ParseOptions{MaxAttributes: 4})
		before := positionSnapshot(t, document)
		err := document.UpdatePositions(0, []PositionUpdate{{CellID: "b", X: 0, Y: 6}, {CellID: "a", X: 2, Y: 3}})
		if !errors.Is(err, ErrLimitExceeded) {
			t.Fatalf("candidate reparse=%v", err)
		}
		assertPositionSnapshot(t, document, before)
		if err := document.UpdatePositions(0, []PositionUpdate{{CellID: "b", X: 0, Y: 6}}); err != nil {
			t.Fatalf("edit after candidate failure=%v", err)
		}
		want := strings.Replace(source, `y="-5"`, `y="6"`, 1)
		if positionBytes(t, document) != want {
			t.Fatal("failed candidate changed retained offsets")
		}
	})
}

func TestPositionsRejectNonFiniteDestinationsAndSelectionOverflow(t *testing.T) {
	for _, number := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		for _, updates := range [][]PositionUpdate{
			{{CellID: "a", X: number, Y: 20}, {CellID: "b", X: 50, Y: 10}},
			{{CellID: "a", X: 3, Y: 4}, {CellID: "b", X: 40, Y: number}},
		} {
			document := parsePositionTest(t, positionPair, ParseOptions{})
			before := positionSnapshot(t, document)
			if err := document.UpdatePositions(0, updates); !errors.Is(err, ErrInvalidGeometry) {
				t.Fatalf("nonfinite update=%v", err)
			}
			assertPositionSnapshot(t, document, before)
		}
	}
	source := strings.Replace(positionPair, `x="-10"`, `x="-1e308"`, 1)
	source = strings.Replace(source, `x="40"`, `x="1e308"`, 1)
	document := parsePositionTest(t, source, ParseOptions{})
	// Selection overflow is distinct from each individually finite vertex.
	if err := document.ValidatePositionTargets(0, []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	original := positionBytes(t, document)
	for _, alignment := range []Alignment{AlignLeft, AlignCenterX, AlignRight, AlignTop, AlignCenterY, AlignBottom} {
		if plan, err := document.PlanAlignment(0, []string{"a", "b"}, alignment); !errors.Is(err, ErrInvalidGeometry) || len(plan) != 0 {
			t.Fatalf("overflow plan=%#v err=%v", plan, err)
		}
	}
	if positionBytes(t, document) != original || document.Dirty() {
		t.Fatal("overflow plan mutated source")
	}
	source = strings.Replace(positionPair, `width="30"`, `width="1e308"`, 1)
	document = parsePositionTest(t, source, ParseOptions{})
	before := positionSnapshot(t, document)
	if err := document.UpdatePositions(0, []PositionUpdate{{CellID: "b", X: 5, Y: 5}, {CellID: "a", X: 1e308, Y: 20}}); !errors.Is(err, ErrInvalidGeometry) {
		t.Fatalf("destination extent overflow=%v", err)
	}
	assertPositionSnapshot(t, document, before)
}

func TestPositionsValidateArgumentsWithoutMutation(t *testing.T) {
	document := parsePositionTest(t, positionPair, ParseOptions{})
	before := positionSnapshot(t, document)
	if err := document.ValidatePositionTargets(0, nil); err == nil {
		t.Fatal("empty selection validated")
	}
	for _, ids := range [][]string{nil, {"a"}} {
		if plan, err := document.PlanAlignment(0, ids, AlignLeft); err == nil || len(plan) != 0 {
			t.Fatalf("short plan=%#v err=%v", plan, err)
		}
	}
	if _, err := document.PlanAlignment(0, []string{"a", "b"}, Alignment(255)); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("invalid alignment=%v", err)
	}
	for _, index := range []int{-1, 1} {
		if err := document.ValidatePositionTargets(index, []string{"a"}); !errors.Is(err, ErrPageNotFound) {
			t.Fatalf("invalid page validate=%v", err)
		}
		if _, err := document.PlanAlignment(index, []string{"a", "b"}, AlignLeft); !errors.Is(err, ErrPageNotFound) {
			t.Fatalf("invalid page plan=%v", err)
		}
		if err := document.UpdatePositions(index, []PositionUpdate{{CellID: "a", X: 1, Y: 2}}); !errors.Is(err, ErrPageNotFound) {
			t.Fatalf("invalid page update=%v", err)
		}
	}
	var missing *Document
	if err := missing.ValidatePositionTargets(0, []string{"a"}); !errors.Is(err, ErrPageNotFound) {
		t.Fatalf("nil document validate=%v", err)
	}
	if err := missing.UpdatePositions(0, nil); !errors.Is(err, ErrPageNotFound) {
		t.Fatalf("nil document update=%v", err)
	}
	assertPositionSnapshot(t, document, before)
}

type positionDocumentSnapshot struct {
	source, original, exported string
	dirty                      bool
}

func positionSnapshot(t *testing.T, document *Document) positionDocumentSnapshot {
	t.Helper()
	exported, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return positionDocumentSnapshot{positionBytes(t, document), string(document.OriginalBytes()), string(exported), document.Dirty()}
}

func assertPositionSnapshot(t *testing.T, document *Document, want positionDocumentSnapshot) {
	t.Helper()
	if got := positionSnapshot(t, document); got != want {
		t.Fatalf("failure/no-op mutated document:\ngot %#v\nwant %#v", got, want)
	}
}

func parsePositionTest(t *testing.T, source string, options ParseOptions) *Document {
	t.Helper()
	document, _, err := Parse([]byte(source), options)
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func positionBytes(t *testing.T, document *Document) string {
	t.Helper()
	result, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return string(result)
}
