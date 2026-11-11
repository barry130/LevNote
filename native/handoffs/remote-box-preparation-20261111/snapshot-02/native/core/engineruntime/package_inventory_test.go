//go:build windows

package engineruntime

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
)

const inventoryFixture = `{
  "schema": 1,
  "version": "native-worker.20261109.8",
  "goos": "windows",
  "goarch": "amd64",
  "goamd64": "v1",
  "excludes": ["release-manifest.json"],
  "scope": "Every shipped file except this manifest",
  "configuration": "%APPDATA%/LevNoteNative",
  "files": [
    {"path":"attribution/SOURCE_THIRD_PARTY_NOTICES.txt","size":6390,"sha256":"41A9F7B6526B85576F5E44263211A2EA61B3371A112A66F772A7A62A7A1BA8B2"},
    {"path":"engines/markdown-export/levnote-markdown-export.exe","size":3300352,"sha256":"7487710e2882099f522c4f8a14e61c03a3f1086fa5c70e5a073daa83c1d85f4c"},
    {"path":"harmless/unrelated.txt","size":2,"sha256":"0000000000000000000000000000000000000000000000000000000000000000"}
  ]
}`

func TestPackageInventoryParsesLiteralShapeAndNormalizesHash(t *testing.T) {
	inventory, err := parseReleaseInventory([]byte(inventoryFixture))
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Version != "native-worker.20261109.8" || len(inventory.Files) != 3 {
		t.Fatal("literal fixture was not decoded", inventory)
	}
	source := inventory.Files["attribution/SOURCE_THIRD_PARTY_NOTICES.txt"]
	if source.Size != 6390 || source.SHA256 != strings.ToLower("41A9F7B6526B85576F5E44263211A2EA61B3371A112A66F772A7A62A7A1BA8B2") {
		t.Fatal("source file was not decoded and normalized", source)
	}
	if inventory.Files["harmless/unrelated.txt"].Size != 2 {
		t.Fatal("harmless unrelated entry was rejected")
	}
	if _, listed := inventory.Files["release-manifest.json"]; listed {
		t.Fatal("inventory listed itself")
	}
}

func TestPackageInventoryRejectsMalformedObjects(t *testing.T) {
	for _, mode := range []string{
		"missing-root", "missing-file", "wrongcase-root", "wrongcase-file",
		"duplicate-root", "duplicate-file", "null-root", "null-file",
		"unknown-root", "unknown-file", "noninteger-size", "float-size",
		"overflow-size", "negative-size", "empty-files", "extra-exclude",
		"empty-exclude", "scope-null", "scope-empty", "scope-too-long",
		"configuration-empty", "version-empty", "hash-short", "hash-invalid",
		"trailing-object",
	} {
		t.Run(mode, func(t *testing.T) {
			data := []byte(inventoryFixture)
			switch mode {
			case "missing-root":
				data = bytes.Replace(data, []byte(`"goamd64": "v1",`), []byte(""), 1)
			case "missing-file":
				data = bytes.Replace(data, []byte(`"size":3300352,`), []byte(""), 1)
			case "wrongcase-root":
				data = bytes.Replace(data, []byte(`"goos"`), []byte(`"Goos"`), 1)
			case "wrongcase-file":
				data = bytes.Replace(data, []byte(`"sha256":"7487`), []byte(`"SHA256":"7487`), 1)
			case "duplicate-root":
				data = bytes.Replace(data, []byte(`"schema": 1,`), []byte(`"schema": 1,"schema": 1,`), 1)
			case "duplicate-file":
				data = bytes.Replace(data, []byte(`"size":2,`), []byte(`"size":2,"size":2,`), 1)
			case "null-root":
				data = bytes.Replace(data, []byte(`"goarch": "amd64",`), []byte(`"goarch":null,`), 1)
			case "null-file":
				data = bytes.Replace(data, []byte(`"size":3300352,`), []byte(`"size":null,`), 1)
			case "unknown-root":
				data = bytes.Replace(data, []byte(`"files": [`), []byte(`"extra":1,"files": [`), 1)
			case "unknown-file":
				data = bytes.Replace(data, []byte(`"path":"harmless`), []byte(`"mode":"x","path":"harmless`), 1)
			case "noninteger-size":
				data = bytes.Replace(data, []byte(`"size":3300352,`), []byte(`"size":"3300352",`), 1)
			case "float-size":
				data = bytes.Replace(data, []byte(`"size":3300352,`), []byte(`"size":3300352.0,`), 1)
			case "overflow-size":
				data = bytes.Replace(data, []byte(`"size":3300352,`), []byte(`"size":9223372036854775808,`), 1)
			case "negative-size":
				data = bytes.Replace(data, []byte(`"size":3300352,`), []byte(`"size":-1,`), 1)
			case "empty-files":
				data = bytes.Replace(data, []byte(inventoryFixture[strings.Index(inventoryFixture, `"files": [`):]), []byte(`"files": []}`), 1)
			case "extra-exclude":
				data = bytes.Replace(data, []byte(`["release-manifest.json"]`), []byte(`["release-manifest.json","other"]`), 1)
			case "empty-exclude":
				data = bytes.Replace(data, []byte(`["release-manifest.json"]`), []byte(`[]`), 1)
			case "scope-null":
				data = bytes.Replace(data, []byte(`"scope": "Every shipped file except this manifest",`), []byte(`"scope":null,`), 1)
			case "scope-empty":
				data = bytes.Replace(data, []byte(`"scope": "Every shipped file except this manifest",`), []byte(`"scope": "",`), 1)
			case "scope-too-long":
				data = bytes.Replace(data, []byte(`"scope": "Every shipped file except this manifest",`), []byte(`"scope": "`+strings.Repeat("x", 1025)+`",`), 1)
			case "configuration-empty":
				data = bytes.Replace(data, []byte(`"configuration": "%APPDATA%/LevNoteNative",`), []byte(`"configuration": "",`), 1)
			case "version-empty":
				data = bytes.Replace(data, []byte(`"version": "native-worker.20261109.8",`), []byte(`"version": "",`), 1)
			case "hash-short":
				data = bytes.Replace(data, []byte(`"sha256":"7487710e2882099f522c4f8a14e61c03a3f1086fa5c70e5a073daa83c1d85f4c"`), []byte(`"sha256":"7487710e"`), 1)
			case "hash-invalid":
				data = bytes.Replace(data, []byte(`"sha256":"7487710e2882099f522c4f8a14e61c03a3f1086fa5c70e5a073daa83c1d85f4c"`), []byte(`"sha256":"`+strings.Repeat("g", 64)+`"`), 1)
			case "trailing-object":
				data = append(data, []byte(` {"ignored":true}`)...)
			}
			if _, err := parseReleaseInventory(data); err == nil {
				t.Fatal("malformed inventory accepted", mode)
			}
		})
	}
}

func TestPackageInventoryRejectsJSONBytesAndCaps(t *testing.T) {
	for _, mode := range []string{
		"utf8", "lone-surrogate", "valid-surrogate-pair", "oversize", "entry-count",
		"whitespace-cap", "not-object", "array-root", "hash-uppercase-only-valid",
	} {
		t.Run(mode, func(t *testing.T) {
			data := []byte(inventoryFixture)
			switch mode {
			case "utf8":
				data = append(data, 0xff)
			case "lone-surrogate":
				data = bytes.Replace(data, []byte(`"scope": "Every shipped file except this manifest",`),
					[]byte(`"scope": "unexpected \uDC00",`), 1)
			case "valid-surrogate-pair":
				data = bytes.Replace(data, []byte(`"scope": "Every shipped file except this manifest",`),
					[]byte(`"scope": "valid \uD83E\uDDF0",`), 1)
				if _, err := parseReleaseInventory(data); err != nil {
					t.Fatal("valid surrogate pair was rejected", err)
				}
				return
			case "oversize":
				data = append(data, bytes.Repeat([]byte(" "), releaseInventoryLimit)...)
			case "whitespace-cap":
				data = append(data, bytes.Repeat([]byte(" "), releaseInventoryLimit-len(data)+1)...)
			case "entry-count":
				data = bytes.Replace(data, []byte(inventoryFixture[strings.Index(inventoryFixture, `"files": [`):]), []byte(makeInventoryManyFiles()), 1)
			case "not-object":
				data = []byte(`null`)
			case "array-root":
				data = []byte(`[]`)
			case "hash-uppercase-only-valid":
				if _, err := parseReleaseInventory(data); err != nil {
					t.Fatal("uppercase hexadecimal hash was rejected")
				}
				return
			}
			if _, err := parseReleaseInventory(data); err == nil {
				t.Fatal("invalid JSON data or cap accepted", mode)
			}
		})
	}
}

func TestPackageInventoryRejectsInvalidPathsAndAliases(t *testing.T) {
	badPaths := []string{
		"", "/absolute.txt", "C:/absolute.txt", "drive:relative.txt",
		"back\\slash.txt", "dot/./file.txt", "dot/../file.txt", "double//file.txt",
		"trailing/file.", "trailing/file ", "control\x01.txt", "embedded\x00.txt", "nul.txt",
		"angle</file.txt", "greater>/file.txt", "quote\"/file.txt", "pipe|/file.txt",
		"question?/file.txt", "star*/file.txt", "conin$/file.txt", "conout$/file.txt",
		"CON /file.txt", "CON .txt", "conin$", "conout$", "com1/file.txt", "com1.txt", "com¹/file.txt", "com²/file.txt", "com³/file.txt",
		"lpt1/file.txt", "lpt¹/file.txt", "lpt²/file.txt", "lpt³/file.txt",
		"release-manifest.json", "RELEASE-MANIFEST.JSON",
		"attribution/SOURCE_THIRD_PARTY_NOTICES.txt",
	}
	for _, path := range badPaths {
		t.Run(path, func(t *testing.T) {
			encoded, err := json.Marshal(path)
			if err != nil {
				t.Fatal(err)
			}
			data := bytes.Replace([]byte(inventoryFixture), []byte(`"path":"harmless/unrelated.txt"`), append([]byte(`"path":`), encoded...), 1)
			if _, err := parseReleaseInventory(data); err == nil {
				t.Fatal("invalid path accepted", path)
			}
		})
	}
	if !inventoryValidPath("engines/markdown-export/levnote-markdown-export.exe") {
		t.Fatal("valid catalog path was rejected")
	}
	if !inventoryValidPath("attribution/SOURCE_THIRD_PARTY_NOTICES.txt") {
		t.Fatal("valid notice path was rejected")
	}
	if inventoryValidPath("com1") || inventoryValidPath("com¹") || inventoryValidPath("com²") ||
		inventoryValidPath("com³") || inventoryValidPath("lpt¹") {
		t.Fatal("reserved device number was accepted")
	}
	if !inventoryValidPath("COM3D.txt") || !inventoryValidPath("COM10.txt") || !inventoryValidPath("LPT10.txt") {
		t.Fatal("ordinary COM/LPT-prefixed filename was rejected")
	}
}

func TestPackageInventoryRejectsEqualFoldPathAliases(t *testing.T) {
	fixture := `{"schema":1,"version":"v1","goos":"windows","goarch":"amd64","goamd64":"v1",
"excludes":["release-manifest.json"],"scope":"s","configuration":"c",
"files":[{"path":"K.txt","size":0,"sha256":"` + strings.Repeat("0", 64) + `"},
{"path":"\u212A.txt","size":0,"sha256":"` + strings.Repeat("1", 64) + `"}]}`
	if _, err := parseReleaseInventory([]byte(fixture)); err == nil {
		t.Fatal("Unicode simple-fold path alias was accepted")
	}
}

func makeInventoryManyFiles() string {
	var builder strings.Builder
	builder.WriteString(`"files": [`)
	for i := 0; i < releaseInventoryEntryLimit+1; i++ {
		if i != 0 {
			builder.WriteByte(',')
		}
		_, _ = builder.WriteString(`{"path":"file-` + strconv.Itoa(i) + `.txt","size":0,"sha256":"` + strings.Repeat("0", 64) + `"}`)
	}
	builder.WriteString(`]}`)
	return builder.String()
}

func TestPackageInventoryErrorsUseExistingFailureType(t *testing.T) {
	_, err := parseReleaseInventory([]byte(`{}`))
	var failed *Failure
	if !errors.As(err, &failed) || failed.Kind != FailureIntegrity {
		t.Fatal("decoder did not use existing failure type", err)
	}
}
