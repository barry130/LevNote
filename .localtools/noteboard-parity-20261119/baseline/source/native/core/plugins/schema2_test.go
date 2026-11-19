package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validSchema2Manifest = `{
	"schema": 2,
	"id": "source-outline",
	"name": "Source outline",
	"version": "1.0.0",
	"apiVersion": 1,
	"extensions": [".src"],
	"outline": [
		{"prefix": "[", "suffix": "]", "level": 1},
		{"prefix": "# ", "suffix": "", "level": 2}
	]
}`

func TestSchema2OutlineDecodesStrictly(t *testing.T) {
	syntax, err := decodeManifest([]byte(validSchema2Manifest))
	if err != nil {
		t.Fatal(err)
	}
	if syntax.Schema != 2 || len(syntax.Outline) != 2 {
		t.Fatalf("schema2 decoded as %#v", syntax)
	}
	want := []OutlineRule{
		{Prefix: "[", Suffix: "]", Level: 1},
		{Prefix: "# ", Suffix: "", Level: 2},
	}
	for i, rule := range want {
		if syntax.Outline[i] != rule {
			t.Fatalf("rule %d = %#v; want %#v", i, syntax.Outline[i], rule)
		}
	}
}

func schema2Manifest(outline string) string {
	return fmt.Sprintf(`{
	"schema": 2,
	"id": "source-outline",
	"name": "Source outline",
	"version": "1.0.0",
	"apiVersion": 1,
	"extensions": [".src"]%s
}`, func() string {
		if strings.TrimSpace(outline) == "" {
			return ""
		}
		return ",\n\t" + outline
	}())
}

func outlineField(rules string) string {
	return fmt.Sprintf(`"outline": [%s]`, rules)
}

func rule(prefix, suffix string, level int) string {
	return fmt.Sprintf(`{"prefix": %q, "suffix": %q, "level": %d}`, prefix, suffix, level)
}

func TestSchemaOutlineCapabilityPresence(t *testing.T) {
	empty := schema2Manifest(outlineField(""))
	if syntax := mustDecode(t, empty); syntax.Schema != 2 || len(syntax.Outline) != 0 {
		t.Fatalf("empty outline decoded as %#v", syntax)
	}
	absent := schema2Manifest("")
	if syntax := mustDecode(t, absent); len(syntax.Outline) != 0 {
		t.Fatalf("absent outline decoded as %#v", syntax)
	}
}

func TestSchema1RejectsOutlineAndSchema2RequiresStrictTypes(t *testing.T) {
	schema1Outline := strings.Replace(validSchema2Manifest, `"schema": 2,`, `"schema": 1,`, 1)
	if _, err := decodeManifest([]byte(schema1Outline)); err == nil || !strings.Contains(err.Error(), outlineSchema1Error) {
		t.Fatalf("schema1 outline error = %v", err)
	}

	tests := []struct {
		name    string
		replace func(string) string
	}{
		{"outline null", func(string) string { return schema2Manifest(`"outline": null`) }},
		{"outline object", func(string) string { return schema2Manifest(`"outline": {}`) }},
		{"unknown top-level", func(s string) string {
			return strings.Replace(s, `"apiVersion": 1`, `"apiVersion": 1, "future": true`, 1)
		}},
		{"unknown rule", func(s string) string { return strings.Replace(s, `"level": 1}`, `"level": 1, "anchor": true}`, 1) }},
		{"prefix missing", func(s string) string {
			return strings.Replace(s, `{"prefix": "[", "suffix": "]", "level": 1}`, `{"suffix": "]", "level": 1}`, 1)
		}},
		{"prefix null", func(s string) string { return strings.Replace(s, `"prefix": "["`, `"prefix": null`, 1) }},
		{"suffix null", func(s string) string { return strings.Replace(s, `"suffix": "]"`, `"suffix": null`, 1) }},
		{"prefix invalid utf16 escape", func(s string) string { return strings.Replace(s, `"prefix": "["`, `"prefix": "\ud800"`, 1) }},
		{"suffix invalid utf16 escape", func(s string) string { return strings.Replace(s, `"suffix": "]"`, `"suffix": "\ud800"`, 1) }},
		{"prefix control", func(s string) string { return strings.Replace(s, `"prefix": "["`, `"prefix": "\u0001"`, 1) }},
		{"suffix newline", func(s string) string { return strings.Replace(s, `"suffix": "]"`, `"suffix": "\n"`, 1) }},
		{"prefix empty", func(s string) string { return strings.Replace(s, `"prefix": "["`, `"prefix": ""`, 1) }},
		{"prefix over max", func(s string) string { return strings.Replace(s, `"prefix": "["`, `"prefix": "0123456789abcdefg"`, 1) }},
		{"suffix over max", func(s string) string { return strings.Replace(s, `"suffix": "]"`, `"suffix": "0123456789abcdefg"`, 1) }},
		{"level missing", func(string) string { return schema2Manifest(outlineField(`{"prefix": "#", "suffix": ""}`)) }},
		{"level null", func(s string) string { return strings.Replace(s, `"level": 1`, `"level": null`, 1) }},
		{"level float", func(s string) string { return strings.Replace(s, `"level": 1`, `"level": 1.2`, 1) }},
		{"level string", func(s string) string { return strings.Replace(s, `"level": 1`, `"level": "1"`, 1) }},
		{"level low", func(s string) string { return strings.Replace(s, `"level": 1`, `"level": 0`, 1) }},
		{"level high", func(s string) string { return strings.Replace(s, `"level": 1`, `"level": 7`, 1) }},
		{"duplicate markers", func(s string) string {
			return strings.Replace(s, `{"prefix": "# ", "suffix": "", "level": 2}`, `{"prefix": "[", "suffix": "]", "level": 3}`, 1)
		}},
		{"nine rules", func(string) string {
			rules := make([]string, 0, 9)
			for i := 0; i < 9; i++ {
				rules = append(rules, fmt.Sprintf(`{"prefix": "%d", "suffix": "", "level": 1}`, i))
			}
			return schema2Manifest(outlineField(strings.Join(rules, ",")))
		}},
		{"rule null", func(string) string { return schema2Manifest(outlineField(`null`)) }},
		{"rule number", func(string) string { return schema2Manifest(outlineField(`1`)) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := test.replace(validSchema2Manifest)
			if _, err := decodeManifest([]byte(data)); err == nil {
				t.Fatalf("invalid manifest was accepted: %s", data)
			}
		})
	}
}

func TestLiteralReplacementRuneIsAllowedButIllFormedSurrogateEscapeIsRejected(t *testing.T) {
	literal := strings.Replace(validSchema2Manifest, `"prefix": "["`, `"prefix": "�"`, 1)
	syntax, err := decodeManifest([]byte(literal))
	if err != nil {
		t.Fatal(err)
	}
	if syntax.Outline[0].Prefix != "�" {
		t.Fatalf("literal replacement-rune marker decoded as %#v", syntax.Outline[0])
	}

	unpaired := strings.Replace(validSchema2Manifest, `"prefix": "["`, `"prefix": "\ud800"`, 1)
	if _, err := decodeManifest([]byte(unpaired)); err == nil {
		t.Fatal("ill-formed surrogate escape was accepted")
	}

	paired := strings.Replace(validSchema2Manifest, `"prefix": "["`, `"prefix": "\ud83d\ude00"`, 1)
	pairedSyntax, err := decodeManifest([]byte(paired))
	if err != nil {
		t.Fatal(err)
	}
	if pairedSyntax.Outline[0].Prefix != "😀" {
		t.Fatalf("paired surrogate escape decoded as %#v", pairedSyntax.Outline[0])
	}

	absentSuffix := strings.Replace(validSchema2Manifest, `"prefix": "[", "suffix": "]", "level": 1`, `"prefix": "[", "level": 1`, 1)
	syntax, err = decodeManifest([]byte(absentSuffix))
	if err != nil {
		t.Fatal(err)
	}
	if syntax.Outline[0].Suffix != "" {
		t.Fatalf("absent suffix decoded as %#v", syntax.Outline[0])
	}
}

func TestJSONEscapeParityDoesNotRejectLiteralBackslashSequences(t *testing.T) {
	// JSON source is "\\ud800": a literal backslash followed by u/d/8/0/0.
	// The decoded marker is `\ud800`, not a Unicode surrogate escape.
	literalBackslash := strings.Replace(validSchema2Manifest, `"prefix": "["`, `"prefix": "\\ud800"`, 1)
	syntax, err := decodeManifest([]byte(literalBackslash))
	if err != nil {
		t.Fatal(err)
	}
	if syntax.Outline[0].Prefix != `\ud800` {
		t.Fatalf("literal backslash marker decoded as %#v", syntax.Outline[0])
	}

	lowSurrogate := strings.Replace(validSchema2Manifest, `"prefix": "["`, `"prefix": "\udc00"`, 1)
	if _, err := decodeManifest([]byte(lowSurrogate)); err == nil {
		t.Fatal("lone low surrogate escape was accepted")
	}

	escapedQuote := strings.Replace(validSchema2Manifest, `"suffix": "]"`, `"suffix": "\\\"\ud800"`, 1)
	if _, err := decodeManifest([]byte(escapedQuote)); err == nil {
		t.Fatal("escaped quote followed by a surrogate escape was accepted")
	}
}

func TestDirectInvalidUTF8ManifestAndEscapedUnicodeAreRejected(t *testing.T) {
	data := []byte(validSchema2Manifest)
	index := strings.Index(string(data), `"prefix": "["`)
	if index < 0 {
		t.Fatal("fixture marker missing")
	}
	data[index+len(`"prefix": "`)] = 0xff
	if _, err := decodeManifest(data); err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
		t.Fatalf("direct invalid UTF-8 error = %v", err)
	}

	validUnicode := strings.Replace(validSchema2Manifest, `"prefix": "["`, `"prefix": "\u4e2d"`, 1)
	syntax, err := decodeManifest([]byte(validUnicode))
	if err != nil {
		t.Fatal(err)
	}
	if syntax.Outline[0].Prefix != "中" {
		t.Fatalf("escaped Unicode decoded as %#v", syntax.Outline[0])
	}
}

func TestOutlineRuleCountAndUTF8ByteBounds(t *testing.T) {
	rules := make([]string, 0, 8)
	for i := 0; i < 8; i++ {
		rules = append(rules, fmt.Sprintf(`{"prefix": "%d", "suffix": "", "level": 1}`, i))
	}
	exactEight := schema2Manifest(outlineField(strings.Join(rules, ",")))
	if syntax := mustDecode(t, exactEight); len(syntax.Outline) != 8 {
		t.Fatalf("exact eight-rule manifest decoded as %#v", syntax)
	}

	// One CJK rune is three UTF-8 bytes; four runes plus two two-byte runes
	// equal the exact 16-byte marker cap.
	exactUnicodeMarker := strings.Repeat("中", 4) + "éé"
	validUnicodeSize := schema2Manifest(outlineField(rule(exactUnicodeMarker, "", 1)))
	if syntax := mustDecode(t, validUnicodeSize); syntax.Outline[0].Prefix != exactUnicodeMarker {
		t.Fatalf("exact 16-byte UTF-8 marker decoded as %#v", syntax.Outline[0])
	}

	sevenCJPRunes := strings.Repeat("中", 7)
	oversizeUnicode := schema2Manifest(outlineField(rule(sevenCJPRunes, "", 1)))
	if _, err := decodeManifest([]byte(oversizeUnicode)); err == nil {
		t.Fatal("over-bound UTF-8 marker was accepted")
	}
}

func TestSchema2ManagerLoadInstallAndSyntaxCopySemantics(t *testing.T) {
	root := t.TempDir()
	installRoot := filepath.Join(root, "installed")
	source := filepath.Join(root, "source-outline.json")
	if err := os.WriteFile(source, []byte(validSchema2Manifest), 0600); err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(filepath.Join(root, "settings"), installRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.InstallFile(source); err != nil {
		t.Fatal(err)
	}
	syntax, ok := manager.Syntax("source-outline")
	if !ok || len(syntax.Outline) != 2 || syntax.Outline[0].Prefix != "[" || syntax.Outline[1].Level != 2 {
		t.Fatalf("installed schema2 syntax = %#v, %v", syntax, ok)
	}
	syntax.Extensions[0] = ".mutated"
	syntax.Keywords = append(syntax.Keywords, "mutated")
	syntax.Outline[0] = OutlineRule{Prefix: "mutated", Suffix: "mutated", Level: 9}
	again, _ := manager.Syntax("source-outline")
	if again.Extensions[0] != ".src" || len(again.Keywords) != 0 || again.Outline[0].Prefix != "[" || again.Outline[0].Level != 1 {
		t.Fatalf("caller mutation changed manager syntax: %#v", again)
	}
	if match, err := manager.Match("example.src"); err != nil || match.ID != "source-outline" {
		t.Fatalf("schema2 install did not match: %#v %v", match, err)
	}
}

func TestBundledSchema2ExamplesAreAccepted(t *testing.T) {
	for _, name := range []string{"ini-outline.json", "config-headings.json"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "plugins", name))
		if err != nil {
			t.Fatal(err)
		}
		syntax, err := decodeManifest(data)
		if err != nil {
			t.Fatalf("%s is invalid: %v", name, err)
		}
		if syntax.Schema != 2 || len(syntax.Outline) == 0 {
			t.Fatalf("%s decoded as %#v", name, syntax)
		}
	}
}

func mustDecode(t *testing.T, data string) SyntaxManifest {
	t.Helper()
	syntax, err := decodeManifest([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	return syntax
}
