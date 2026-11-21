package dependencypatches

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestBothPatchFilesAndAllVendoredChangesMatchPinnedManifest(t *testing.T) {
	manifestBytes, err := os.ReadFile("mygo-v0.8.3-patch-files.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Schema int    `json:"schema"`
		Module string `json:"module"`
		Files  []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != 1 || manifest.Module != mygoVersion {
		t.Fatal("wrong patch manifest pin")
	}
	wrapper, err := os.ReadFile(filepath.Join("..", "scripts", "go.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(manifestBytes)
	if !strings.Contains(string(wrapper), "$patchManifestSHA256 = '"+hex.EncodeToString(sum[:])+"'") {
		t.Fatal("wrapper does not pin exact patch manifest")
	}
	// Actual diff headers independently require every modified source/test file
	// in both patches to be inventoried exactly once.
	want := map[string]bool{}
	for _, name := range []string{"mygo-v0.8.3-no-startup-cleanup.patch", "mygo-v0.8.3-bitmap-dispose.patch"} {
		patch, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		want["dependency-patches/"+name] = true
		count := 0
		for _, line := range strings.Split(string(patch), "\n") {
			if path, ok := strings.CutPrefix(line, "+++ b/"); ok {
				if !strings.HasPrefix(path, "vendor/github.com/egoist/mygo/") || want[path] {
					t.Fatalf("unexpected/duplicate patch path %s", path)
				}
				want[path] = true
				count++
			}
		}
		if count == 0 {
			t.Fatalf("empty patch: %s", name)
		}
	}
	for _, file := range manifest.Files {
		if !want[file.Path] || filepath.IsAbs(file.Path) || strings.Contains(file.Path, "..") {
			t.Fatalf("unexpected/duplicate manifest path %s", file.Path)
		}
		delete(want, file.Path)
		data, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(file.Path)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != file.SHA256 {
			t.Fatalf("incomplete MyGo patch: changed %s", file.Path)
		}
	}
	if len(want) != 0 {
		t.Fatalf("patch files absent from manifest: %v", want)
	}
}

const (
	mygoVersion     = "github.com/egoist/mygo v0.8.3"
	unsafeStartCall = "go cleanUpdateLeftovers()"
	patchMarker     = "// LEVNOTE PATCH: this app does not enable MyGo updates"
)

func TestVendoredMyGoStartupCleanupPatchIsPresent(t *testing.T) {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("..", "vendor", "github.com", "egoist", "mygo", "app.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if strings.Contains(text, unsafeStartCall) {
		t.Fatalf("vendored %s still starts %s; rerun dependency-patches/refresh.ps1", mygoVersion, unsafeStartCall)
	}
	if !strings.Contains(text, patchMarker) {
		t.Fatalf("vendored %s is missing the LevNote startup-safety marker; rerun dependency-patches/refresh.ps1", mygoVersion)
	}
}

func TestVendorManifestPinsMyGoAndLicensesRemain(t *testing.T) {
	t.Helper()
	modules, err := os.ReadFile(filepath.Join("..", "vendor", "modules.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(modules), "# "+mygoVersion) {
		t.Fatalf("vendor/modules.txt does not pin %s", mygoVersion)
	}
	for _, relative := range []string{
		filepath.Join("github.com", "egoist", "mygo", "LICENSE"),
		filepath.Join("github.com", "egoist", "mygo", "README.md"),
	} {
		if _, err := os.Stat(filepath.Join("..", "vendor", relative)); err != nil {
			t.Fatalf("vendored MyGo notice %s is missing: %v", relative, err)
		}
	}
}

const nativeInputPatchPath = "dependency-patches/mygo-v0.8.3-native-input.patch"

var nativeInputExpectedPaths = []string{
	nativeInputPatchPath,
	"vendor/github.com/egoist/mygo/internal/platform/text_input.go",
	"vendor/github.com/egoist/mygo/internal/windows/surface.go",
	"vendor/github.com/egoist/mygo/internal/windows/surface_input.go",
	"vendor/github.com/egoist/mygo/ui/headless.go",
	"vendor/github.com/egoist/mygo/ui/text_input_cancel_test.go",
	"vendor/github.com/egoist/mygo/ui/text_input_client.go",
}

type nativeInputManifestEntry struct {
	Path string `json:"path"`
	SHA256 string `json:"sha256"`
}
type nativeInputManifest struct {
	Schema int `json:"schema"`
	Module string `json:"module"`
	Files []nativeInputManifestEntry `json:"files"`
}

func nativeInputSHA(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func nativeInputCleanPath(name string) bool {
	if name == "" || strings.HasPrefix(name,"/") || strings.ContainsAny(name,"\\:\x00\r\n\t") || path.Clean(name) != name { return false }
	for _, part := range strings.Split(name,"/") { if part == "" || part == "." || part == ".." { return false } }
	return true
}

func nativeInputReadRegular(root, name string) ([]byte,error) {
	if !nativeInputCleanPath(name) { return nil,fmt.Errorf("unsafe member path %q",name) }
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 { return nil,fmt.Errorf("native-input root must be a real directory") }
	target := root
	parts := strings.Split(name,"/")
	for i, part := range parts {
		target = filepath.Join(target,part)
		info, err := os.Lstat(target)
		if err != nil { return nil,fmt.Errorf("member missing %s: %w",name,err) }
		if info.Mode()&os.ModeSymlink != 0 || i < len(parts)-1 && !info.IsDir() || i == len(parts)-1 && !info.Mode().IsRegular() { return nil,fmt.Errorf("member path is not regular %s",name) }
	}
	return os.ReadFile(target)
}

var nativeInputHunk = regexp.MustCompile(`^@@ -([0-9]+)(?:,([0-9]+))? \+([0-9]+)(?:,([0-9]+))? @@(?:.*)$`)

// Count only actual file headers, not source lines inside a hunk. Both hunk
// sides are consumed, so a source string resembling +++ cannot bless a member.
func nativeInputPatchMembers(raw []byte) (map[string]bool,error) {
	members, folded := map[string]bool{}, map[string]bool{}
	old, current, declared := "", "", ""
	needNew, hasHunk := false,false
	oldLeft,newLeft := 0,0
	for _, line := range strings.Split(string(raw),"\n") {
		if oldLeft > 0 || newLeft > 0 {
			if strings.HasPrefix(line,"\\ No newline at end of file") { continue }
			if line == "" { return nil,fmt.Errorf("truncated native-input hunk") }
			switch line[0] {
			case ' ': oldLeft--;newLeft--
			case '-': oldLeft--
			case '+': newLeft--
			default:return nil,fmt.Errorf("unexpected native-input hunk line")
			}
			if oldLeft < 0 || newLeft < 0 { return nil,fmt.Errorf("native-input hunk count mismatch") }
			continue
		}
		if line == "" || strings.HasPrefix(line,"\\ No newline at end of file") { continue }
		if strings.HasPrefix(line,"diff --git ") {
			if needNew || declared != "" || current != "" && !hasHunk { return nil,fmt.Errorf("incomplete native-input file") }
			fields := strings.Fields(line)
			if len(fields) != 4 || !strings.HasPrefix(fields[2],"a/vendor/github.com/egoist/mygo/") || fields[3] != "b/"+strings.TrimPrefix(fields[2],"a/") { return nil,fmt.Errorf("invalid native-input diff roots") }
			declared = strings.TrimPrefix(fields[3],"b/")
			if !nativeInputCleanPath(declared) { return nil,fmt.Errorf("unsafe native-input diff path") }
			continue
		}
		if strings.HasPrefix(line,"index ") || line == "new file mode 100644" { continue }
		if strings.HasPrefix(line,"--- ") {
			if needNew || current != "" && !hasHunk { return nil,fmt.Errorf("incomplete native-input file header") }
			old = strings.SplitN(strings.TrimPrefix(line,"--- "),"\t",2)[0]
			if old != "/dev/null" && (!strings.HasPrefix(old,"a/vendor/github.com/egoist/mygo/") || !nativeInputCleanPath(strings.TrimPrefix(old,"a/"))) { return nil,fmt.Errorf("invalid native-input original path") }
			needNew = true
			continue
		}
		if strings.HasPrefix(line,"+++ ") {
			if !needNew { return nil,fmt.Errorf("native-input new header has no original") }
			next := strings.SplitN(strings.TrimPrefix(line,"+++ "),"\t",2)[0]
			if !strings.HasPrefix(next,"b/vendor/github.com/egoist/mygo/") { return nil,fmt.Errorf("native-input changed path outside vendor") }
			name := strings.TrimPrefix(next,"b/")
			if !nativeInputCleanPath(name) || old != "/dev/null" && old != "a/"+name || declared != "" && declared != name || folded[strings.ToLower(name)] { return nil,fmt.Errorf("unsafe, renamed or duplicate native-input file header %s",name) }
			members[name],folded[strings.ToLower(name)] = true,true
			current,hasHunk,needNew,declared = name,false,false,""
			continue
		}
		if strings.HasPrefix(line,"@@ ") {
			match := nativeInputHunk.FindStringSubmatch(line)
			if current == "" || needNew || match == nil { return nil,fmt.Errorf("native-input hunk without valid file headers") }
			counts := [2]int{1,1}
			for i,index := range []int{2,4} {
				if match[index] != "" { n,err := strconv.Atoi(match[index]);if err != nil { return nil,fmt.Errorf("native-input hunk overflow") };counts[i]=n }
			}
			if counts[0] == 0 && counts[1] == 0 || old == "/dev/null" && counts[0] != 0 { return nil,fmt.Errorf("empty or invalid native-input addition hunk") }
			oldLeft,newLeft,hasHunk = counts[0],counts[1],true
			continue
		}
		return nil,fmt.Errorf("unsupported native-input diff metadata %q",line)
	}
	if needNew || declared != "" || oldLeft != 0 || newLeft != 0 || current == "" || !hasHunk || len(members) == 0 { return nil,fmt.Errorf("empty/incomplete native-input patch") }
	return members,nil
}

func validateNativeInputSupplement(root string, manifestRaw, oldRaw, patchRaw []byte) error {
	var manifest,old nativeInputManifest
	if err := json.Unmarshal(manifestRaw,&manifest); err != nil { return fmt.Errorf("invalid native-input manifest: %w",err) }
	if err := json.Unmarshal(oldRaw,&old); err != nil { return fmt.Errorf("invalid preserved manifest: %w",err) }
	if manifest.Schema != 1 || manifest.Module != mygoVersion || len(manifest.Files) != 7 || old.Schema != 1 || old.Module != mygoVersion || len(old.Files) != 26 { return fmt.Errorf("native-input/preserved manifest schema, module or cardinality mismatch") }
	changed,err := nativeInputPatchMembers(patchRaw)
	if err != nil { return err }
	if len(changed) != 6 { return fmt.Errorf("native-input patch must have exactly six vendor headers") }
	expected := map[string]bool{}
	for _,name := range nativeInputExpectedPaths { expected[name]=true }
	for name := range changed { if !expected[name] { return fmt.Errorf("unrelated native-input patch member %s",name) } }
	if len(expected) != len(changed)+1 { return fmt.Errorf("native-input source inventory incomplete") }
	oldPaths := map[string]bool{}
	for _,entry := range old.Files { oldPaths[strings.ToLower(entry.Path)] = true }
	seen := map[string]bool{}
	for _,entry := range manifest.Files {
		fold := strings.ToLower(entry.Path)
		if !nativeInputCleanPath(entry.Path) || !expected[entry.Path] || seen[fold] || oldPaths[fold] { return fmt.Errorf("unsafe, unrelated, duplicate or overlapping native-input manifest member %s",entry.Path) }
		if len(entry.SHA256) != 64 || strings.ToLower(entry.SHA256) != entry.SHA256 { return fmt.Errorf("native-input hash syntax %s",entry.Path) }
		if _,err := hex.DecodeString(entry.SHA256); err != nil { return fmt.Errorf("native-input hash syntax %s",entry.Path) }
		seen[fold]=true
		data,err := nativeInputReadRegular(root,entry.Path)
		if err != nil { return err }
		if nativeInputSHA(data) != entry.SHA256 { return fmt.Errorf("native-input changed member %s",entry.Path) }
		if entry.Path == nativeInputPatchPath && !bytes.Equal(data,patchRaw) { return fmt.Errorf("native-input header input differs from actual patch member") }
	}
	if len(seen) != len(expected) { return fmt.Errorf("native-input manifest missing changed member") }
	return nil
}

func TestNativeInputSupplementExactInventoryAndWrapperPins(t *testing.T) {
	old,err := os.ReadFile("mygo-v0.8.3-patch-files.json");if err != nil { t.Fatal(err) }
	raw,err := os.ReadFile("mygo-v0.8.3-native-input-files.json");if err != nil { t.Fatal(err) }
	patchBytes,err := os.ReadFile("mygo-v0.8.3-native-input.patch");if err != nil { t.Fatal(err) }
	if err := validateNativeInputSupplement("..",raw,old,patchBytes); err != nil { t.Fatal(err) }
	pin := nativeInputSHA(raw)
	windows,err := os.ReadFile(filepath.Join("..","scripts","go.ps1"));if err != nil { t.Fatal(err) }
	linux,err := os.ReadFile(filepath.Join("..","..",".localtools","native-go.sh"));if err != nil { t.Fatal(err) }
	refresh,err := os.ReadFile("refresh.ps1");if err != nil { t.Fatal(err) }
	if !strings.Contains(string(windows),"$nativeInputManifestSHA256 = '"+pin+"'") || !strings.Contains(string(linux),"assert.equal(hash(nativeBytes), '"+pin+"'") || !strings.Contains(string(refresh),"$nativeInputManifestSHA256 = '"+pin+"'") { t.Fatal("both real wrappers and refresh must pin the identical seven-member supplement bytes") }
	for _,name := range nativeInputExpectedPaths {
		for _,source := range [][]byte{windows,linux,refresh} { if !bytes.Contains(source,[]byte("'"+name+"'")) { t.Fatalf("wrapper/refresh expected inventory misses %s",name) } }
	}
}

type nativeInputDataFixture struct {
	root string
	old []byte
	patch []byte
	manifest nativeInputManifest
}

func newNativeInputDataFixture(t *testing.T) *nativeInputDataFixture {
	t.Helper()
	old,err := os.ReadFile("mygo-v0.8.3-patch-files.json");if err != nil { t.Fatal(err) }
	f := &nativeInputDataFixture{root:t.TempDir(),old:old,manifest:nativeInputManifest{Schema:1,Module:mygoVersion}}
	var patch strings.Builder
	for _,name := range nativeInputExpectedPaths {
		if name == nativeInputPatchPath { continue }
		fmt.Fprintf(&patch,"--- a/%s\n+++ b/%s\n@@ -1 +1 @@\n-old\n+new\n",name,name)
		data := []byte("fixture bytes for "+name+"\n")
		p := filepath.Join(f.root,filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p),0700); err != nil { t.Fatal(err) }
		if err := os.WriteFile(p,data,0600); err != nil { t.Fatal(err) }
		f.manifest.Files = append(f.manifest.Files,nativeInputManifestEntry{Path:name,SHA256:nativeInputSHA(data)})
	}
	f.patch = []byte(patch.String())
	f.manifest.Files = append(f.manifest.Files,nativeInputManifestEntry{Path:nativeInputPatchPath})
	return f
}

func (f *nativeInputDataFixture) save(t *testing.T) []byte {
	t.Helper()
	patchFile := filepath.Join(f.root,filepath.FromSlash(nativeInputPatchPath))
	if err := os.MkdirAll(filepath.Dir(patchFile),0700); err != nil { t.Fatal(err) }
	if err := os.WriteFile(patchFile,f.patch,0600); err != nil { t.Fatal(err) }
	for i := range f.manifest.Files { if f.manifest.Files[i].Path == nativeInputPatchPath { f.manifest.Files[i].SHA256=nativeInputSHA(f.patch) } }
	raw,err := json.Marshal(f.manifest);if err != nil { t.Fatal(err) }
	return raw
}

func TestNativeInputSupplementDataPolicyRejectsSameCountSubstitution(t *testing.T) {
	cases := []struct{name string; mutate func(*nativeInputDataFixture)}{
		{"wrong-schema",func(f *nativeInputDataFixture){f.manifest.Schema=2}},
		{"wrong-module",func(f *nativeInputDataFixture){f.manifest.Module="github.com/egoist/mygo v0.8.4"}},
		{"missing-changed-member",func(f *nativeInputDataFixture){f.manifest.Files=f.manifest.Files[1:]}},
		{"same-count-duplicate",func(f *nativeInputDataFixture){f.manifest.Files[1]=f.manifest.Files[0]}},
		{"same-count-unrelated-vendor",func(f *nativeInputDataFixture){f.manifest.Files[0].Path="vendor/github.com/egoist/mygo/ui/unrelated.go"}},
		{"old-pin-overlap",func(f *nativeInputDataFixture){var old nativeInputManifest;if err:=json.Unmarshal(f.old,&old);err!=nil{panic(err)};f.manifest.Files[0].Path=old.Files[0].Path}},
		{"unsafe-parent",func(f *nativeInputDataFixture){f.manifest.Files[0].Path="vendor/../outside.go"}},
		{"unsafe-absolute",func(f *nativeInputDataFixture){f.manifest.Files[0].Path="/vendor/github.com/egoist/mygo/ui/headless.go"}},
		{"unsafe-backslash",func(f *nativeInputDataFixture){f.manifest.Files[0].Path="vendor\\github.com\\egoist\\mygo\\ui\\headless.go"}},
		{"noncanonical-dot",func(f *nativeInputDataFixture){f.manifest.Files[0].Path="vendor/./github.com/egoist/mygo/ui/headless.go"}},
		{"case-alias",func(f *nativeInputDataFixture){f.manifest.Files[0].Path="vendor/github.com/egoist/mygo/ui/HEADLESS.go"}},
		{"wrong-hash",func(f *nativeInputDataFixture){f.manifest.Files[0].SHA256=strings.Repeat("0",64)}},
		{"bad-hash-syntax",func(f *nativeInputDataFixture){f.manifest.Files[0].SHA256="NOT-A-SHA"}},
		{"same-count-duplicate-patch-header",func(f *nativeInputDataFixture){name:=nativeInputExpectedPaths[2];f.patch=bytes.ReplaceAll(f.patch,[]byte(name),[]byte(nativeInputExpectedPaths[1]))}},
		{"unrelated-patch-header",func(f *nativeInputDataFixture){f.patch=bytes.ReplaceAll(f.patch,[]byte(nativeInputExpectedPaths[1]),[]byte("vendor/github.com/egoist/mygo/ui/unrelated.go"))}},
		{"patch-wrong-root",func(f *nativeInputDataFixture){f.patch=bytes.ReplaceAll(f.patch,[]byte("+++ b/vendor/"),[]byte("+++ b/other/"))}},
		{"dangling-diff-header",func(f *nativeInputDataFixture){f.patch=append(f.patch,[]byte("diff --git a/vendor/github.com/egoist/mygo/ui/unrelated.go b/vendor/github.com/egoist/mygo/ui/unrelated.go\n")...)}},
		{"empty-patch",func(f *nativeInputDataFixture){f.patch=nil}},
		{"truncated-hunk",func(f *nativeInputDataFixture){f.patch=[]byte("--- a/vendor/github.com/egoist/mygo/ui/headless.go\n+++ b/vendor/github.com/egoist/mygo/ui/headless.go\n@@ -1,2 +1 @@\n-old\n+new\n")}},
	}
	for _,tc := range cases { t.Run(tc.name,func(t *testing.T){
		f:=newNativeInputDataFixture(t)
		valid:=f.save(t)
		if err:=validateNativeInputSupplement(f.root,valid,f.old,f.patch);err!=nil{t.Fatalf("positive data fixture rejected before mutation: %v",err)}
		tc.mutate(f)
		raw:=f.save(t) // patch self hash follows mutated patch, so header tests reach the real parser
		if err:=validateNativeInputSupplement(f.root,raw,f.old,f.patch);err==nil{t.Fatal("malformed data/inventory was accepted")}
	}) }
}

func TestNativeInputSupplementRejectsMissingChangedAndNonregularMembers(t *testing.T) {
	for _,name:=range nativeInputExpectedPaths { for _,kind:=range []string{"missing","changed","directory"} {
		t.Run(kind+"/"+name,func(t *testing.T){
			f:=newNativeInputDataFixture(t);raw:=f.save(t)
			if err:=validateNativeInputSupplement(f.root,raw,f.old,f.patch);err!=nil{t.Fatal("positive fixture",err)}
			p:=filepath.Join(f.root,filepath.FromSlash(name))
			switch kind {
			case "missing": if err:=os.Remove(p);err!=nil{t.Fatal(err)}
			case "changed": if err:=os.WriteFile(p,[]byte("changed"),0600);err!=nil{t.Fatal(err)}
			case "directory": if err:=os.Remove(p);err!=nil{t.Fatal(err)};if err:=os.Mkdir(p,0700);err!=nil{t.Fatal(err)}
			}
			if err:=validateNativeInputSupplement(f.root,raw,f.old,f.patch);err==nil{t.Fatal("missing/changed/nonregular real member accepted")}
		})
	} }
}
