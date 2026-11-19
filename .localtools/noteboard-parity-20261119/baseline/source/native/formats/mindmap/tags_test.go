package mindmap

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

const tagFixture = "\r\n" + `{
 "format":"levnote.mindmap","version":1,"layout":"logicalStructure",
 "futureTop":{"decimal":1.00},
 "root":{"id":"r","topic":"根🙂","note":"原备注","icons":["⭐"],
 "image":{"url":"local.png","width":10.00,"height":20,"extra":2.000},
 "tags":["dup","",{"text":"dup","className":"","style":{},"first":1.00,"second":{"n":9.000}},"dup",
 {"text":"对象","style":{"z":0.50,"a":{"n":123456789012345678901234567890}},"className":"样式"}],
 "unknownNode":{"n":3.1400},"children":[
 {"id":"absent","topic":"无","children":[]},
 {"id":"empty","topic":"空","tags":[],"children":[]},
 {"id":"one","topic":"一","tags":[""],"children":[]}]}}
`

func tagsDocument(t *testing.T) *Document {
	t.Helper()
	doc, _, err := Parse([]byte(tagFixture), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func tagsSerialized(t *testing.T, doc *Document) []byte {
	t.Helper()
	raw, err := doc.Serialize(SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestTagsReadOwnershipPresenceAndOriginalBytes(t *testing.T) {
	doc := tagsDocument(t)
	tags, present, err := doc.Tags("r")
	if err != nil || !present || len(tags) != 5 || tags[0].Text != "dup" || tags[1].Text != "" || !tags[2].Object || !tags[2].ClassNameSet || !tags[2].StyleSet {
		t.Fatalf("read: %+v %v %v", tags, present, err)
	}
	for _, tc := range []struct {
		id      string
		present bool
	}{{"absent", false}, {"empty", true}} {
		got, present, err := doc.Tags(tc.id)
		if err != nil || present != tc.present || len(got) != 0 || (got != nil) != tc.present {
			t.Fatalf("%s presence = %#v %v %v", tc.id, got, present, err)
		}
	}
	// Mutate every mutable layer of the returned object, including retained Raw.
	tags[0].Text = "changed"
	tags[2].ClassNameSet = false
	tags[2].Raw[0] = '['
	tags[2].Extra.Members[0].Key = "changed"
	tags[2].Extra.Members[0].Raw[0] = '9'
	tags[4].Style.Members[0].Key = "changed"
	tags[4].Style.Members[0].Raw[0] = '9'
	if doc.Dirty() || !bytes.Equal(tagsSerialized(t, doc), []byte(tagFixture)) {
		t.Fatal("reading or caller mutation changed source")
	}
	if _, _, err := doc.Tags("missing"); !errors.Is(err, ErrNodeNotFound) {
		t.Fatal(err)
	}
}

func TestTagsOrdinalAddAndExactSurvivorMetadata(t *testing.T) {
	doc := tagsDocument(t)
	if err := doc.RemoveTag("r", 0); err != nil {
		t.Fatal(err)
	}
	added := " 逗,号🙂 "
	if err := doc.AddTag("r", added); err != nil {
		t.Fatal(err)
	}
	again, _, err := Parse(tagsSerialized(t, doc), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	got, _, _ := again.Tags("r")
	wantText := []string{"", "dup", "dup", "对象", added}
	for i, value := range wantText {
		if len(got) != len(wantText) || got[i].Text != value || got[i].Object != (i == 1 || i == 3) {
			t.Fatalf("ordered survivors = %+v", got)
		}
	}
	if !got[1].ClassNameSet || got[1].ClassName != "" || !got[1].StyleSet || len(got[1].Style.Members) != 0 || !got[3].StyleSet || got[3].ClassName != "样式" {
		t.Fatalf("object presence lost: %+v", got)
	}
	checkMembers := func(members []Member, keys, values []string) {
		t.Helper()
		if len(members) != len(keys) {
			t.Fatalf("members = %+v", members)
		}
		for i, member := range members {
			if member.Key != keys[i] || string(member.Raw) != values[i] {
				t.Fatalf("member %d = %s %s", i, member.Key, member.Raw)
			}
		}
	}
	checkMembers(got[1].Extra.Members, []string{"first", "second"}, []string{"1.00", `{"n":9.000}`})
	checkMembers(got[3].Style.Members, []string{"z", "a"}, []string{"0.50", `{"n":123456789012345678901234567890}`})
	checkMembers(again.Extra.Members, []string{"futureTop"}, []string{`{"decimal":1.00}`})
	checkMembers(again.Root.Extra.Members, []string{"unknownNode"}, []string{`{"n":3.1400}`})
	if again.Root.Image.Width.String() != "10.00" || string(again.Root.Image.Extra.Members[0].Raw) != "2.000" || *again.Root.Note != "原备注" || (*again.Root.Icons)[0] != "⭐" {
		t.Fatal("unrelated metadata changed")
	}
	// Removing the object with equal text must leave the plain duplicate.
	if err := again.RemoveTag("r", 1); err != nil {
		t.Fatal(err)
	}
	remaining, _, _ := again.Tags("r")
	if len(remaining) != 4 || remaining[1].Object || remaining[1].Text != "dup" {
		t.Fatalf("duplicate ordinal removal: %+v", remaining)
	}
}

func TestTagsNoOpAndLastRemovalPresence(t *testing.T) {
	doc := tagsDocument(t)
	for _, id := range []string{"absent", "empty"} {
		if err := doc.ClearTags(id); err != nil {
			t.Fatal(err)
		}
	}
	if doc.Dirty() || !bytes.Equal(tagsSerialized(t, doc), []byte(tagFixture)) {
		t.Fatal("no-op clear rewrote source")
	}
	if err := doc.RemoveTag("one", 0); err != nil {
		t.Fatal(err)
	}
	if err := doc.ClearTags("r"); err != nil {
		t.Fatal(err)
	}
	again, _, err := Parse(tagsSerialized(t, doc), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"one", "r", "empty"} {
		tags, present, err := again.Tags(id)
		if err != nil || !present || tags == nil || len(tags) != 0 {
			t.Fatalf("%s lost explicit []", id)
		}
	}
	if _, present, _ := again.Tags("absent"); present {
		t.Fatal("absent acquired tags")
	}
}

func TestTagsRejectedActionsAndValidationRollback(t *testing.T) {
	doc := tagsDocument(t)
	for _, call := range []func() error{
		func() error { return doc.AddTag("r", "") },
		func() error { return doc.AddTag("r", "a\xff") },
		func() error { return doc.RemoveTag("r", -1) },
		func() error { return doc.RemoveTag("r", 5) },
		func() error { return doc.RemoveTag("absent", 0) },
	} {
		if err := call(); !errors.Is(err, ErrInvalidDocument) {
			t.Fatalf("invalid action = %v", err)
		}
		if doc.Dirty() || !bytes.Equal(tagsSerialized(t, doc), []byte(tagFixture)) {
			t.Fatal("invalid action changed original")
		}
	}
	for _, call := range []func() error{
		func() error { return doc.AddTag("missing", "x") },
		func() error { return doc.RemoveTag("missing", 0) },
		func() error { return doc.ClearTags("missing") },
	} {
		if err := call(); !errors.Is(err, ErrNodeNotFound) {
			t.Fatal(err)
		}
	}
	for _, action := range []string{"add", "remove", "clear"} {
		t.Run(action, func(t *testing.T) {
			doc := tagsDocument(t)
			before, original := *doc.Root.Tags, doc.Root.Tags
			// Public programmatic trees can exceed the parser's depth cap. This
			// makes document validation fail after candidate preparation.
			chain := NewNode("deep", "deep")
			cursor := chain
			for i := 0; i < 257; i++ {
				child := NewNode(strings.Repeat("x", i+1), "")
				cursor.Children = []*Node{child}
				cursor = child
			}
			doc.Root.Children = append(doc.Root.Children, chain)
			var err error
			switch action {
			case "add":
				err = doc.AddTag("r", "new")
			case "remove":
				err = doc.RemoveTag("r", 0)
			case "clear":
				err = doc.ClearTags("r")
			}
			if !errors.Is(err, ErrLimitExceeded) || doc.Root.Tags != original || !reflect.DeepEqual(*doc.Root.Tags, before) || doc.dirty {
				t.Fatalf("failed mutation did not roll back: %v", err)
			}
		})
	}
}

func TestTagsMutationsOwnSurvivorsAndDoNotImposeViewLimits(t *testing.T) {
	for _, remove := range []bool{false, true} {
		doc := tagsDocument(t)
		old := *doc.Root.Tags
		if remove {
			if err := doc.RemoveTag("r", 0); err != nil {
				t.Fatal(err)
			}
		} else if err := doc.AddTag("r", "x"); err != nil {
			t.Fatal(err)
		}
		before := tagsSerialized(t, doc)
		old[2].Raw[0] = '['
		old[2].Extra.Members[0].Raw[0] = '9'
		old[4].Style.Members[0].Key = "wrong"
		old[4].Style.Members[0].Raw[0] = '9'
		if !bytes.Equal(tagsSerialized(t, doc), before) {
			t.Fatal("mutation retained caller-owned survivor aliases")
		}
	}
	doc := tagsDocument(t)
	for range 65 {
		if err := doc.AddTag("absent", strings.Repeat("x", 513)+"\r\n"); err != nil {
			t.Fatal("format API accidentally applied inspector budget", err)
		}
	}
	tags, present, _ := doc.Tags("absent")
	if !present || len(tags) != 65 || tags[0].Text != strings.Repeat("x", 513)+"\r\n" {
		t.Fatal("format API normalized tag input")
	}
}
