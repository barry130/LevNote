import argparse, hashlib, json, os, pathlib, re, shutil, stat, struct, subprocess, time, zipfile
ROOT = pathlib.Path(r"F:\AIproject\LevNote")
NATIVE = ROOT / "native"
VERSION = "native-flowix.20261116.6-preview"
OUT = ROOT / ".localtools/flowix-chrome-preview-20261116-01"
PROOF = ROOT / ".localtools/flowix-chrome-preview-build-20261116-01"
ARCHIVE = ROOT / "native/downloads/LevNote-Flowix-preview-20261116.6-windows-x64.zip"
GO = ROOT / ".localtools/go1.27.3/bin/go.exe"
GO_SHA = "83b7ba66215e696414fa49462d0db2da4f4cf15c7bebd42f0b189fa04866999d"
PATCH_SHA = "9113120bd2ebcd9a86842153239cddff84d1f1bd3f46f5d4f0f152eb4560e16a"
ENGINES = [
    ("markdown-export", "levnote.markdown-export", "html", "levnote-markdown-export"),
    ("mindmap-svg", "levnote.mindmap-svg", "svg", "levnote-mindmap-svg"),
    ("mindmap-preview", "levnote.mindmap-preview", "mindmap-scene-v1", "levnote-mindmap-preview"),
    ("drawio-svg", "levnote.drawio-svg", "svg", "levnote-drawio-svg"),
]
def digest(path):
    with path.open("rb") as f:
        h = hashlib.sha256()
        for chunk in iter(lambda: f.read(1024*1024), b""): h.update(chunk)
        return h.hexdigest()
def put_json(path, data):
    with path.open("x", encoding="utf-8", newline="\n") as f: json.dump(data, f, indent=2, ensure_ascii=False)
def plain_chain(path):
    for p in [path] + list(path.parents):
        if not p.exists(): continue
        st = p.lstat()
        if st.st_file_attributes & stat.FILE_ATTRIBUTE_REPARSE_POINT: raise RuntimeError("Reparse path: " + str(p))
def snapshot():
    rows = {}
    for folder in ["app", "cmd", "core", "formats", "modules", "ui", "vendor", "scripts", "portable", "dependency-patches", "licenses"]:
        for p in sorted((NATIVE / folder).rglob("*")):
            if p.is_file(): rows[p.relative_to(ROOT).as_posix()] = digest(p)
    for p in [NATIVE / "go.mod", NATIVE / "go.sum", NATIVE / "THIRD_PARTY_NOTICES.md", ROOT / "LICENSE", ROOT / ".localtools/gate.sh", GO]:
        rows[p.relative_to(ROOT).as_posix()] = digest(p)
    return rows
def check_inputs():
    assert digest(GO) == GO_SHA
    manifest_path = NATIVE / "dependency-patches/mygo-v0.8.3-patch-files.json"
    assert digest(manifest_path) == PATCH_SHA
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    assert manifest["schema"] == 1 and manifest["module"] == "github.com/egoist/mygo v0.8.3" and len(manifest["files"]) == 26
    for item in manifest["files"]: assert digest(NATIVE / item["path"]) == item["sha256"]
    source = (NATIVE / "vendor/github.com/egoist/mygo/app.go").read_text(encoding="utf-8")
    assert "// LEVNOTE PATCH: this app does not enable MyGo updates" in source
    assert not re.search(r"\bgo\s+cleanUpdateLeftovers\s*\(", source)
    assert "# github.com/egoist/mygo v0.8.3" in (NATIVE / "vendor/modules.txt").read_text(encoding="utf-8")
def pe(path):
    data = path.read_bytes()
    assert data[:2] == b"MZ"
    offset = struct.unpack_from("<I", data, 0x3c)[0]
    assert 64 <= offset < len(data)-94 and data[offset:offset+4] == b"PE\x00\x00"
    machine = struct.unpack_from("<H", data, offset+4)[0]
    magic = struct.unpack_from("<H", data, offset+24)[0]
    subsystem = struct.unpack_from("<H", data, offset+92)[0]
    assert (machine, magic, subsystem) == (0x8664, 0x20b, 2)
    return {"machine":hex(machine), "magic":hex(magic), "subsystem":subsystem}
def build(name, package, relative, linkflags, env):
    target = OUT / relative
    target.parent.mkdir(parents=True, exist_ok=True)
    command = [str(GO), "build", "-trimpath", "-buildvcs=false", "-ldflags", linkflags, "-o", str(target), package]
    start = time.time()
    with (PROOF / (name+".stdout.raw")).open("xb") as stdout, (PROOF / (name+".stderr.raw")).open("xb") as stderr:
        result = subprocess.run(command, cwd=NATIVE, env=env, stdout=stdout, stderr=stderr, timeout=300)
    receipt = {"argv":command, "cwd":str(NATIVE), "exitCode":result.returncode, "seconds":round(time.time()-start,2), "environment":{k:env[k] for k in FIXED_ENV}}
    if result.returncode == 0:
        receipt.update({"path":relative, "size":target.stat().st_size, "sha256":digest(target), "pe":pe(target)})
    put_json(PROOF / (name+".json"), receipt)
    assert result.returncode == 0, receipt
    print(json.dumps({"build":name, "exitCode":result.returncode, "bytes":receipt["size"]}), flush=True)
    return receipt
FIXED_ENV = {
    "GOROOT":str(ROOT / ".localtools/go1.27.3"), "GOCACHE":str(ROOT / ".localtools/cache"), "GOTMPDIR":str(ROOT / ".localtools/tmp"),
    "GOENV":"off", "GOWORK":"off", "GOFLAGS":"-mod=vendor", "GOPROXY":"off", "GOSUMDB":"off", "GOVCS":"*:off",
    "GOTOOLCHAIN":"local", "GOTELEMETRY":"off", "CGO_ENABLED":"0", "GOOS":"windows", "GOARCH":"amd64", "GOAMD64":"v1"
}
cli = argparse.ArgumentParser()
cli.add_argument('--test-attempt', required=True)
cli.add_argument('--vet-attempt', required=True)
cli.add_argument('--audit-attempt')
params = cli.parse_args()
assert params.test_attempt.isdigit() and params.vet_attempt.isdigit() and (params.audit_attempt is None or params.audit_attempt.isdigit())
TESTDIR = ROOT / ('.localtools/flowix-chrome-20261116-test-' + params.test_attempt)
VETDIR = ROOT / ('.localtools/flowix-chrome-20261116-vet-' + params.vet_attempt)
AUDITDIR = ROOT / ('.localtools/flowix-chrome-20261116-final-audit-' + (params.audit_attempt or params.test_attempt))
tested = json.loads((TESTDIR / 'inputs-before.json').read_text(encoding='utf-8'))
for evidence in (TESTDIR, VETDIR):
    record = json.loads((evidence / 'result.json').read_text(encoding='utf-8'))
    assert record['exitCode'] == 0 and record['inputsSame'] and not record['counts'].get('fail', 0)
    assert tested == json.loads((evidence / 'inputs-before.json').read_text(encoding='utf-8'))
    assert tested == json.loads((evidence / 'inputs-after.json').read_text(encoding='utf-8'))
assert json.loads((TESTDIR / 'result.json').read_text(encoding='utf-8'))['runCount'] > 0
audit = json.loads((AUDITDIR / 'audit.json').read_text(encoding='utf-8'))
for name in ('removedProductFiles','missingOldTests','regressedOldTests','newSkips','duplicateRuns','duplicateResults'):
    assert not audit[name], name
assert audit['liveTestInputsEqual']
for name, sha in tested.items(): assert digest(ROOT / name) == sha, name
check_inputs()
for path in [OUT, PROOF, ARCHIVE]:
    assert not path.exists(), "Fresh output required: " + str(path)
    plain_chain(path.parent)
PROOF.mkdir()
OUT.mkdir()
before = snapshot()
put_json(PROOF / "before.json", before)
put_json(PROOF / "validation.json", {"testDirectory":str(TESTDIR),"vetDirectory":str(VETDIR),"auditDirectory":str(AUDITDIR),"testedInputs":len(tested)})
env = os.environ.copy()
env.update(FIXED_ENV)
try:
    receipts = []
    for folder, engine, fmt, executable in ENGINES:
        relative = "engines/"+folder+"/"+executable+".exe"
        record = build(folder, "./cmd/"+executable, relative, "-s -w -H=windowsgui -X main.engineVersion="+VERSION, env)
        assert 0 < record["size"] <= 64*1024*1024
        manifest = {"schema":1, "engine":engine, "engineVersion":VERSION, "protocol":1, "format":fmt, "goos":"windows", "goarch":"amd64", "filename":executable+".exe", "size":record["size"], "sha256":record["sha256"]}
        put_json(OUT / "engines" / folder / "manifest.json", manifest)
        receipts.append(record)
    receipts.append(build("host", "./cmd/levnote", "levnote-native.exe", "-s -w -H=windowsgui", env))
    copies = [(ROOT / "LICENSE", "LICENSE"), (NATIVE / "THIRD_PARTY_NOTICES.md", "attribution/SOURCE_THIRD_PARTY_NOTICES.txt")]
    copies += [(NATIVE / "vendor/github.com/egoist/mygo/README.md", "attribution/mygo-README.md"), (NATIVE / "vendor/github.com/ebitengine/purego/README.md", "attribution/purego-README.md")]
    for name in ["mygo-v0.8.3-no-startup-cleanup.patch", "mygo-v0.8.3-bitmap-dispose.patch"]: copies.append((NATIVE / "dependency-patches" / name, "attribution/"+name))
    copies.append((NATIVE / "dependency-patches/README.md", "attribution/dependency-patches-README.md"))
    for name in ["Go-BSD-3-Clause.txt", "go-text-typesetting.txt", "mygo-MIT.txt", "purego-Apache-2.0.txt", "x-image-BSD-3-Clause.txt", "x-net-BSD-3-Clause.txt", "x-text-BSD-3-Clause.txt"]: copies.append((NATIVE / "licenses" / name, "licenses/"+name))
    for source, relative in copies:
        plain_chain(source)
        target = OUT / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        assert not target.exists()
        shutil.copyfile(source, target)
        assert digest(source) == digest(target)
    readme = (NATIVE / "portable/README.md").read_text(encoding="utf-8-sig").replace("{{VERSION}}", VERSION)
    preview = "# flowix 工作台布局体验版\n\n解压整个文件夹后双击 levnote-native.exe。默认使用深色工作台，已保存的浅色偏好会保留，可在设置切换主题。\n\n版本："+VERSION+"。本包重构顶部动作区和左下新建文档区：48px紧凑顶栏、图标与文字对齐的幽灵按钮、胶囊搜索与快捷键提示，以及左对齐的四类新建入口。保留此前 flowix 主题、draw.io 多选对齐、命令面板、行编辑、Markdown 行内格式及导图颜色功能。属于预览构建，Windows 实体输入法、剪贴板和正常关闭的完整发行验收尚未完成。\n\n---\n\n"
    (OUT / "README.md").write_text(preview+readme, encoding="utf-8", newline="\n")
    notice = (NATIVE / "THIRD_PARTY_NOTICES.md").read_text(encoding="utf-8-sig")
    rewrites = {"[LICENSE](../LICENSE)":"[LICENSE](LICENSE)", "(vendor/github.com/ebitengine/purego/README.md)":"(attribution/purego-README.md)", "(dependency-patches/README.md)":"(attribution/dependency-patches-README.md)"}
    for name in ["mygo-v0.8.3-no-startup-cleanup.patch", "mygo-v0.8.3-bitmap-dispose.patch"]: rewrites["(dependency-patches/"+name+")"] = "(attribution/"+name+")"
    for old, new in rewrites.items(): notice = notice.replace(old, new)
    preamble = (NATIVE / "portable/NOTICE-PREAMBLE.md").read_text(encoding="utf-8-sig")
    (OUT / "THIRD_PARTY_NOTICES.md").write_text(preamble+"\n"+notice, encoding="utf-8", newline="\n")
    files = [{"path":p.relative_to(OUT).as_posix(), "size":p.stat().st_size, "sha256":digest(p)} for p in sorted(OUT.rglob("*")) if p.is_file()]
    put_json(OUT / "release-manifest.json", {"schema":1, "version":VERSION, "goos":"windows", "goarch":"amd64", "goamd64":"v1", "excludes":["release-manifest.json"], "scope":"Every shipped file except this manifest; integrity fingerprints, preview build, not full release acceptance.", "configuration":"%APPDATA%/LevNoteNative", "files":files})
    for name in ["README.md", "THIRD_PARTY_NOTICES.md"]:
        for link in re.findall(r"\]\(([^)]+)\)", (OUT / name).read_text(encoding="utf-8")):
            if re.match(r"^(https?:|#)", link): continue
            target = (OUT / link).resolve()
            assert target.is_relative_to(OUT.resolve()) and target.exists(), (name, link)
    check_inputs()
    after = snapshot()
    put_json(PROOF / "after.json", after)
    assert before == after, "Build inputs changed"
    package_files = sorted(p for p in OUT.rglob("*") if p.is_file())
    with zipfile.ZipFile(ARCHIVE, "x", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        for p in package_files: z.write(p, "LevNote-Flowix-preview/"+p.relative_to(OUT).as_posix())
    with zipfile.ZipFile(ARCHIVE) as z:
        assert z.testzip() is None
        assert len(z.namelist()) == len(package_files) == 26
        for p in package_files: assert hashlib.sha256(z.read("LevNote-Flowix-preview/"+p.relative_to(OUT).as_posix())).hexdigest() == digest(p)
    result = {"status":"PACKAGED_PREVIEW", "version":VERSION, "inputUnchanged":True, "sourceFiles":len(before), "guiPEs":5, "files":len(package_files), "bytes":sum(p.stat().st_size for p in package_files), "zip":str(ARCHIVE), "zipBytes":ARCHIVE.stat().st_size, "zipSHA256":digest(ARCHIVE), "hostSHA256":digest(OUT / "levnote-native.exe"), "productLaunched":False, "physicalDesktopAcceptance":False}
    put_json(PROOF / "result.json", result)
    print(json.dumps(result, ensure_ascii=True), flush=True)
except BaseException as e:
    put_json(PROOF / "failure.json", {"type":type(e).__name__, "message":str(e)})
    raise