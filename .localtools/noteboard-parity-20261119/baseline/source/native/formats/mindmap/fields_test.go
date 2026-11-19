package mindmap

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
)

const fieldMindmap = `{
  "format":"levnote.mindmap",
  "version":1,
  "futureTop": 123456789012345678901234567890,
  "root":{
    "id":"root",
    "topic":"根",
    "note":"",
    "hyperLink":"notes/a.mindmap",
    "unknownNode":{"kept":1.00},
    "children":[
      {"id":"child","topic":"子","children":[]}
    ]
  },
  "layout":"logicalStructure"
}`

const markerMindmap = `{
  "format":"levnote.mindmap",
  "version":1,
  "root":{
    "id":"root",
    "topic":"根",
    "icons":["⭐","自定义","⭐","😀"],
    "tags":["plain",{"text":"风险","className":"风险样式","style":{"color":"红"},"riskLevel":2}],
    "unknownNode":{"kept":1.00},
    "children":[
      {"id":"empty","topic":"空","icons":[],"children":[]},
      {"id":"plain","topic":"无标记","children":[]}
    ]
  },
  "layout":"logicalStructure"
}`

func parseFieldMindmap(t *testing.T) *Document {
	t.Helper()
	document, _, err := Parse([]byte(fieldMindmap), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func serializeFieldMindmap(t *testing.T, document *Document) []byte {
	t.Helper()
	serialized, err := document.Serialize(SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return serialized
}

func TestFieldReadsOptionalStateWithoutDirtying(t *testing.T) {
	document := parseFieldMindmap(t)
	note, present, err := document.Field("root", FieldNote)
	if err != nil || note != "" || !present {
		t.Fatalf("note = %q, present = %v, err = %v", note, present, err)
	}
	link, present, err := document.Field("root", FieldHyperLink)
	if err != nil || link != "notes/a.mindmap" || !present {
		t.Fatalf("link = %q, present = %v, err = %v", link, present, err)
	}
	absent, present, err := document.Field("child", FieldNote)
	if err != nil || absent != "" || present {
		t.Fatalf("child note = %q, present = %v, err = %v", absent, present, err)
	}
	if !bytes.Equal(serializeFieldMindmap(t, document), []byte(fieldMindmap)) {
		t.Fatal("field read changed original bytes")
	}
	if _, _, err := document.Field("root", FieldKind("topic")); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("invalid field error = %v", err)
	}
	if _, _, err := document.Field("missing", FieldNote); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("missing node error = %v", err)
	}
}

func TestSetNotePreservesExactOptionalAndUnknownState(t *testing.T) {
	document := parseFieldMindmap(t)
	exact := "中文\r\n备注  \U0001F600"
	if err := document.SetNote("root", exact); err != nil {
		t.Fatal(err)
	}
	got, present, err := document.Field("root", FieldNote)
	if err != nil || !present || got != exact {
		t.Fatalf("got %q, present=%v, err=%v", got, present, err)
	}
	if err := document.SetNote("root", exact); err != nil {
		t.Fatal(err)
	}

	serialized := serializeFieldMindmap(t, document)
	text := string(serialized)
	for _, expected := range []string{
		`"note": "中文\r\n备注  😀"`,
		`"hyperLink": "notes/a.mindmap"`,
		`"unknownNode": {`,
		`"kept":1.00`,
		`123456789012345678901234567890`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("serialized output lost %q:\n%s", expected, serialized)
		}
	}

	if err := document.ClearField("root", FieldNote); err != nil {
		t.Fatal(err)
	}
	_, present, err = document.Field("root", FieldNote)
	if err != nil || present {
		t.Fatalf("cleared note present=%v, err=%v", present, err)
	}
	serialized = serializeFieldMindmap(t, document)
	if strings.Contains(string(serialized), `"note"`) {
		t.Fatalf("cleared note remained serialized:\n%s", serialized)
	}
}

func TestFieldSemanticNoOpsPreserveOriginalBytesAndDirtyState(t *testing.T) {
	document := parseFieldMindmap(t)
	if err := document.SetNote("root", ""); err != nil {
		t.Fatal(err)
	}
	if err := document.SetHyperLink("root", "notes/a.mindmap"); err != nil {
		t.Fatal(err)
	}
	if err := document.ClearField("child", FieldNote); err != nil {
		t.Fatal(err)
	}
	if err := document.ClearField("child", FieldHyperLink); err != nil {
		t.Fatal(err)
	}
	if document.Dirty() {
		t.Fatal("semantic no-op dirtied document")
	}
	if !bytes.Equal(serializeFieldMindmap(t, document), []byte(fieldMindmap)) {
		t.Fatal("semantic no-op changed original bytes")
	}
}

func TestHyperLinkPresenceAndLiteralValue(t *testing.T) {
	document := parseFieldMindmap(t)
	exact := " docs/中文 引用.md "
	if err := document.SetHyperLink("root", exact); err != nil {
		t.Fatal(err)
	}
	got, present, err := document.Field("root", FieldHyperLink)
	if err != nil || !present || got != exact {
		t.Fatalf("got %q, present=%v, err=%v", got, present, err)
	}
	if err := document.ClearField("root", FieldHyperLink); err != nil {
		t.Fatal(err)
	}
	_, present, err = document.Field("root", FieldHyperLink)
	if err != nil || present {
		t.Fatalf("cleared link present=%v, err=%v", present, err)
	}
	serialized := serializeFieldMindmap(t, document)
	if strings.Contains(string(serialized), `"hyperLink"`) {
		t.Fatalf("cleared link remained serialized:\n%s", serialized)
	}
	if strings.Contains(string(serialized), `"note": ""`) == false {
		t.Fatalf("note optional state was lost:\n%s", serialized)
	}
}

func TestSetNoteOnAbsentFieldStoresPresentEmpty(t *testing.T) {
	document := parseFieldMindmap(t)
	if _, present, err := document.Field("child", FieldNote); err != nil || present {
		t.Fatalf("child note unexpectedly present=%v, err=%v", present, err)
	}
	if err := document.SetNote("child", ""); err != nil {
		t.Fatal(err)
	}
	if !document.Dirty() {
		t.Fatal("absent-to-present-empty note was not dirty")
	}
	serialized := serializeFieldMindmap(t, document)
	if !strings.Contains(string(serialized), `"note": ""`) {
		t.Fatalf("present-empty note was not serialized:\n%s", serialized)
	}
	parsed, _, err := Parse(serialized, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	value, present, err := parsed.Field("child", FieldNote)
	if err != nil || !present || value != "" {
		t.Fatalf("reparsed child note = %q, present=%v, err=%v", value, present, err)
	}
}

func TestFieldMutatorsRejectBeforePartialMutation(t *testing.T) {
	document := parseFieldMindmap(t)
	before := serializeFieldMindmap(t, document)

	invalid := []struct {
		name string
		call func() error
	}{
		{"empty hyperlink", func() error { return document.SetHyperLink("root", "") }},
		{"invalid utf8 note", func() error { return document.SetNote("root", "a\xffb") }},
		{"invalid utf8 link", func() error { return document.SetHyperLink("root", "a\xffb") }},
		{"missing node set", func() error { return document.SetNote("missing", "x") }},
		{"missing node clear", func() error { return document.ClearField("missing", FieldNote) }},
		{"invalid clear kind", func() error { return document.ClearField("root", FieldKind("topic")) }},
	}
	for _, test := range invalid {
		if err := test.call(); err == nil {
			t.Fatalf("%s was accepted", test.name)
		}
		if got := serializeFieldMindmap(t, document); !bytes.Equal(got, before) {
			t.Fatalf("%s partially mutated document:\n%s", test.name, got)
		}
		if document.Dirty() {
			t.Fatalf("%s dirtied document", test.name)
		}
	}
}

func TestFieldMutationRollsBackOnDocumentValidationFailure(t *testing.T) {
	document := parseFieldMindmap(t)
	originalNote, present, err := document.Field("root", FieldNote)
	if err != nil || !present {
		t.Fatalf("read note present=%v, err=%v", present, err)
	}

	// Create an invalid candidate state outside the setter. A field edit must
	// not be applied when whole-document validation rejects that state.
	oldColor := document.Root.BranchColor
	empty := ""
	document.Root.BranchColor = &empty
	if err := document.SetNote("root", "candidate"); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("expected validation error, got %v", err)
	}
	got, present, err := document.Field("root", FieldNote)
	if err != nil || !present || got != originalNote {
		t.Fatalf("note rolled back to %q, present=%v, err=%v", got, present, err)
	}
	if document.dirty {
		t.Fatal("rejected candidate set dirty")
	}

	document.Root.BranchColor = oldColor
	if got := serializeFieldMindmap(t, document); !bytes.Equal(got, []byte(fieldMindmap)) {
		t.Fatalf("rejected candidate changed serialization:\n%s", got)
	}
}

func TestFieldRollbackKeepsAlreadyDirtyPointerAndValue(t *testing.T) {
	tests := []struct {
		name  string
		apply func(*Document) error
		read  func(*Document) (**string, string)
	}{
		{
			name: "clear hyperlink",
			apply: func(document *Document) error {
				return document.ClearField("root", FieldHyperLink)
			},
			read: func(document *Document) (**string, string) {
				return &document.Root.HyperLink, *document.Root.HyperLink
			},
		},
		{
			name: "set hyperlink",
			apply: func(document *Document) error {
				return document.SetHyperLink("root", "replacement.mindmap")
			},
			read: func(document *Document) (**string, string) {
				return &document.Root.HyperLink, *document.Root.HyperLink
			},
		},
		{
			name: "clear note",
			apply: func(document *Document) error {
				return document.ClearField("root", FieldNote)
			},
			read: func(document *Document) (**string, string) {
				return &document.Root.Note, *document.Root.Note
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := parseFieldMindmap(t)
			if err := document.SetNote("child", "unrelated real edit"); err != nil {
				t.Fatal(err)
			}
			field, originalValue := test.read(document)
			originalPointer := *field
			if !document.dirty {
				t.Fatal("precondition: unrelated real edit did not set dirty")
			}

			invalidColor := ""
			document.Root.BranchColor = &invalidColor
			if err := test.apply(document); !errors.Is(err, ErrInvalidDocument) {
				t.Fatalf("expected validation error, got %v", err)
			}
			if *field != originalPointer || (*field == nil || **field != originalValue) {
				t.Fatalf("field pointer/value changed on rollback: %#v", *field)
			}
			if !document.dirty {
				t.Fatal("rollback cleared an already-dirty document")
			}

			document.Root.BranchColor = nil
			if got, present, err := document.Field("child", FieldNote); err != nil || !present || got != "unrelated real edit" {
				t.Fatalf("unrelated dirty edit changed on rollback: %q, present=%v, err=%v", got, present, err)
			}
			serialized := serializeFieldMindmap(t, document)
			parsed, _, err := Parse(serialized, ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if _, present, err := parsed.Field("child", FieldNote); err != nil || !present {
				t.Fatalf("unrelated edit was lost after rollback: present=%v, err=%v", present, err)
			}
		})
	}
}

func parseMarkerMindmap(t *testing.T) *Document {
	t.Helper()
	document, _, err := Parse([]byte(markerMindmap), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func TestIconsReadsOptionalStateWithoutDirtying(t *testing.T) {
	document := parseMarkerMindmap(t)
	want := []string{"⭐", "自定义", "⭐", "😀"}
	got, present, err := document.Icons("root")
	if err != nil || !present || !slices.Equal(got, want) {
		t.Fatalf("icons = %#v, present=%v, err=%v", got, present, err)
	}

	got[0] = "mutated"
	gotAgain, present, err := document.Icons("root")
	if err != nil || !present || !slices.Equal(gotAgain, want) {
		t.Fatalf("returned copy aliases document: %#v, err=%v", gotAgain, err)
	}

	empty, present, err := document.Icons("empty")
	if err != nil || !present || empty == nil || len(empty) != 0 {
		t.Fatalf("present-empty icons = %#v, present=%v, err=%v", empty, present, err)
	}
	absent, present, err := document.Icons("plain")
	if err != nil || present || absent != nil {
		t.Fatalf("absent icons = %#v, present=%v, err=%v", absent, present, err)
	}
	if document.Dirty() {
		t.Fatal("icons read dirtied document")
	}
	if got := serializeMarker(t, document); !bytes.Equal(got, []byte(markerMindmap)) {
		t.Fatal("icons read changed original bytes")
	}
}

func TestSetIconsStoresPresentEmptyAndPreservesOptionalState(t *testing.T) {
	document := parseMarkerMindmap(t)
	if err := document.SetIcons("root", nil); err != nil {
		t.Fatal(err)
	}
	if !document.dirty {
		t.Fatal("clearing existing markers did not dirty document")
	}
	got, present, err := document.Icons("root")
	if err != nil || !present || got == nil || len(got) != 0 {
		t.Fatalf("cleared icons = %#v, present=%v, err=%v", got, present, err)
	}

	serialized := serializeMarker(t, document)
	if !strings.Contains(string(serialized), `"icons": []`) {
		t.Fatalf("present-empty icons were not serialized:\n%s", serialized)
	}
	parsed, _, err := Parse(serialized, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got, present, err := parsed.Icons("root"); err != nil || !present || got == nil || len(got) != 0 {
		t.Fatalf("reparsed icons = %#v, present=%v, err=%v", got, present, err)
	}

	// Present-empty cloning and AddNode must retain presence, including through
	// the current nil inner slice produced by Node.Clone.
	emptyChild := NewNode("empty-copy", "copy")
	empty := []string{}
	emptyChild.Icons = &empty
	if _, err := document.AddNode("root", -1, emptyChild); err != nil {
		t.Fatal(err)
	}
	clone := document.Root.Clone()
	if clone.Icons == nil {
		t.Fatal("Clone lost present-empty icons field")
	}
	temp := &Document{Root: clone, Layout: LayoutBoth, Version: Version}
	cloneSerialized, err := temp.Serialize(SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cloneSerialized), `"icons": null`) || !strings.Contains(string(cloneSerialized), `"icons": []`) {
		t.Fatalf("Clone/AddNode did not serialize present-empty icons:\n%s", cloneSerialized)
	}
}

func TestSetIconsSemanticNoOpPreservesPointerBytesAndDirtyState(t *testing.T) {
	document := parseMarkerMindmap(t)
	before := document.Root.Icons
	same := []string{"⭐", "自定义", "⭐", "😀"}
	if err := document.SetIcons("root", same); err != nil {
		t.Fatal(err)
	}
	if document.Root.Icons != before {
		t.Fatal("same icons list replaced the old pointer")
	}
	if document.dirty {
		t.Fatal("same icons list dirtied document")
	}
	if got := serializeMarker(t, document); !bytes.Equal(got, []byte(markerMindmap)) {
		t.Fatal("same icons list changed original bytes")
	}

	// A same-list no-op must also preserve an earlier real edit and its dirty
	// state, not merely avoid setting the dirty flag from a clean document.
	if err := document.SetNote("plain", "已经编辑"); err != nil {
		t.Fatal(err)
	}
	if !document.dirty {
		t.Fatal("precondition: unrelated real edit did not set dirty")
	}
	before = document.Root.Icons
	if err := document.SetIcons("root", same); err != nil {
		t.Fatal(err)
	}
	if document.Root.Icons != before || !document.dirty {
		t.Fatal("already-dirty same-list no-op replaced pointer or cleared dirty")
	}
	if note, present, err := document.Field("plain", FieldNote); err != nil || !present || note != "已经编辑" {
		t.Fatalf("prior note edit changed: %q, present=%v, err=%v", note, present, err)
	}

	emptyBefore := document.Root.Children[0].Icons
	emptyEquivalent := []string{}
	if err := document.SetIcons("empty", emptyEquivalent); err != nil {
		t.Fatal(err)
	}
	if document.Root.Children[0].Icons != emptyBefore || !document.dirty {
		t.Fatal("nil and empty were not treated as the same present-empty list")
	}

	if err := document.SetIcons("plain", nil); err != nil {
		t.Fatal(err)
	}
	if !document.dirty {
		t.Fatal("absent-to-present-empty icons did not dirty document")
	}
	got, present, err := document.Icons("plain")
	if err != nil || !present || got == nil || len(got) != 0 {
		t.Fatalf("absent-to-present-empty icons = %#v, present=%v, err=%v", got, present, err)
	}
}

func TestSetIconsClonesInputAndPreservesArbitraryOrderDuplicatesAndMetadata(t *testing.T) {
	document := parseMarkerMindmap(t)
	input := []string{"😀", "⭐", "自定义", "⭐", "😀"}
	if err := document.SetIcons("root", input); err != nil {
		t.Fatal(err)
	}
	input[0] = "mutated"
	got, present, err := document.Icons("root")
	if err != nil || !present || !slices.Equal(got, []string{"😀", "⭐", "自定义", "⭐", "😀"}) {
		t.Fatalf("input was not cloned: %#v, present=%v, err=%v", got, present, err)
	}

	tag := (*document.Root.Tags)[1]
	if !tag.Object || tag.Text != "风险" || tag.ClassName != "风险样式" || !tag.ClassNameSet {
		t.Fatalf("tag scalar metadata changed: %#v", tag)
	}
	if !tag.StyleSet || len(tag.Style.Members) != 1 || tag.Style.Members[0].Key != "color" || string(tag.Style.Members[0].Raw) != `"红"` {
		t.Fatalf("tag style changed: %#v", tag.Style)
	}
	if len(tag.Extra.Members) != 1 || tag.Extra.Members[0].Key != "riskLevel" || string(tag.Extra.Members[0].Raw) != "2" {
		t.Fatalf("tag extra changed: %#v", tag.Extra)
	}
	if len(document.Root.Extra.Members) != 1 || document.Root.Extra.Members[0].Key != "unknownNode" {
		t.Fatalf("unknown node metadata changed: %#v", document.Root.Extra)
	}
	unknown, ok := document.Root.Extra.Get("unknownNode")
	if !ok || string(unknown) != `{"kept":1.00}` {
		t.Fatalf("unknown node metadata changed: %s", unknown)
	}

	serialized := serializeMarker(t, document)
	text := string(serialized)
	for _, expected := range []string{
		`"icons": [`,
		`"😀"`,
		`"自定义"`,
		`"riskLevel": 2`,
		`"className": "风险样式"`,
		`"style": {`,
		`"color": "红"`,
		`"unknownNode": {`,
		`"kept":1.00`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("real icon edit lost %q:\n%s", expected, serialized)
		}
	}
	parsed, _, err := Parse(serialized, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got, present, err := parsed.Icons("root"); err != nil || !present || !slices.Equal(got, []string{"😀", "⭐", "自定义", "⭐", "😀"}) {
		t.Fatalf("reparsed icons = %#v, present=%v, err=%v", got, present, err)
	}
}

func TestSetIconsRejectsInvalidInputAndMissingNode(t *testing.T) {
	document := parseMarkerMindmap(t)
	before := serializeMarker(t, document)
	if err := document.SetIcons("root", []string{"ok", "a\xffb"}); err == nil {
		t.Fatal("invalid UTF-8 was accepted")
	}
	if err := document.SetIcons("missing", []string{"ok"}); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("missing node error = %v", err)
	}
	if got := serializeMarker(t, document); !bytes.Equal(got, before) {
		t.Fatalf("rejected icon input changed serialization:\n%s", got)
	}
	if document.dirty {
		t.Fatal("rejected icon input dirtied document")
	}
}

func TestSetIconsRollsBackPointerValueAndDirtyStateOnValidationFailure(t *testing.T) {
	document := parseMarkerMindmap(t)
	originalPointer := document.Root.Icons
	originalValue := append([]string(nil), (*originalPointer)...)

	invalidColor := ""
	document.Root.BranchColor = &invalidColor
	if err := document.SetIcons("root", []string{"candidate"}); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if document.Root.Icons != originalPointer || !slices.Equal(*document.Root.Icons, originalValue) {
		t.Fatalf("icons were not restored on clean rollback: %#v", document.Root.Icons)
	}
	if document.dirty {
		t.Fatal("clean rollback set dirty")
	}
	document.Root.BranchColor = nil

	if err := document.SetNote("empty", "unrelated real edit"); err != nil {
		t.Fatal(err)
	}
	if !document.dirty {
		t.Fatal("unrelated edit did not set dirty")
	}
	originalPointer = document.Root.Icons
	originalValue = append([]string(nil), (*originalPointer)...)
	document.Root.BranchColor = &invalidColor
	if err := document.SetIcons("root", []string{"candidate"}); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("expected dirty validation error, got %v", err)
	}
	if document.Root.Icons != originalPointer || !slices.Equal(*document.Root.Icons, originalValue) {
		t.Fatalf("icons were not restored on dirty rollback: %#v", document.Root.Icons)
	}
	if !document.dirty {
		t.Fatal("dirty rollback cleared an already-dirty document")
	}
}

func TestSetIconsAcceptsSixtyFiveValidIcons(t *testing.T) {
	document := parseMarkerMindmap(t)
	values := make([]string, 65)
	for i := range values {
		values[i] = "😀"
	}
	if err := document.SetIcons("root", values); err != nil {
		t.Fatal(err)
	}
	got, present, err := document.Icons("root")
	if err != nil || !present || len(got) != 65 {
		t.Fatalf("65 icons = %d entries, present=%v, err=%v", len(got), present, err)
	}
}

func serializeMarker(t *testing.T, document *Document) []byte {
	t.Helper()
	serialized, err := document.Serialize(SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return serialized
}
