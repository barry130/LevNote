#!/usr/bin/env python3
"""DRAFT, not executed or syntax-checked. Run only on the explicitly authorized box.

Prepare is an independent migration/prerequisite guard, not Go/product execution.
Test/vet require a separately reviewed guard receipt and explicit activation.
This is accidental-bypass protection, not an OS sandbox or physical UI proof.
"""
import argparse
import ctypes
import datetime as dt
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import platform
import re
import shutil
import signal
import stat
import subprocess
import sys
import tarfile
import traceback
import zipfile

BOX = "cb-698f697fed7c"
THREAD = "01a1b6b1-3caf-76b1-81e2-183c090b7b86"
GO_ARCHIVE = "go1.27.3.linux-amd64.tar.gz"
GO_BYTES = 70588678
GO_SHA = "65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40"
ZIP_NAME = "levnote-source-inputs-20261111-01.zip"
ZIP_BYTES = 5332912
ZIP_SHA = "c7454f4f727f7514258d8c41761ba6c9da755744434e35cea5d70fe9ebd3b0d9"
MANIFEST_NAME = "inputs-01.json"
MANIFEST_BYTES = 245791
MANIFEST_SHA = "6d624b8f19a907b5a0f47378572a84fc18fb773db7b642988b0991c2f2c492ec"
INPUT_COUNT = 1212
INPUT_BYTES = 17166862
INPUT_ID = "4cb67e243ff141cc34b0134223b0bc80162f184c375305d5050338cfa4d289cc"
FROZEN = "native/handoffs/open-document-picker-key-correction-evidence-20261110/source-frozen.json"
PATCH = "native/dependency-patches/mygo-v0.8.3-patch-files.json"
PINS = {
    FROZEN: "ab90451d3003e713e6a5260087fa36b0e32dc2b3eda23dd517f8b0502e9b59f6",
    PATCH: "9113120bd2ebcd9a86842153239cddff84d1f1bd3f46f5d4f0f152eb4560e16a",
    "native/go.mod": "376860538beff9e47dafbc110afd5e2c92513c206dfc6947a3061e0a0b0308c3",
    "native/go.sum": "975e1f5b6d7e6e138dddF44c173df5ed1e7aa70395eaad482ed955dfd417c3b7".lower(),
    "native/vendor/modules.txt": "d60e238d007653d4ead22238c261087765c5d14ed4d3492a76d173e4dec17b42",
    "native/scripts/go.ps1": "a3efcd7dc4079641f1767341446168e6778ab07e1c4dd09a968c768d1cc1b178",
}
PRODUCT_ARGV = {
    "test": ["test", "-mod=vendor", "./app", "-count=1"],
    "vet": ["vet", "-mod=vendor", "./app"],
}
ACTIVATION = "picker-final-app-linux-20261111"


def utc():
    return dt.datetime.now(dt.timezone.utc).isoformat(timespec="microseconds")


def require(ok, message):
    if not ok:
        raise RuntimeError(message)


def write_json(path, value):
    with path.open("x", encoding="utf-8", newline="\n") as stream:
        json.dump(value, stream, ensure_ascii=False, indent=2)
        stream.write("\n")


def strict_json(path):
    def object_pairs(pairs):
        result = {}
        for key, value in pairs:
            require(key not in result, "duplicate JSON key: " + key)
            result[key] = value
        return result
    return json.loads(path.read_bytes().decode("utf-8-sig"), object_pairs_hook=object_pairs)


def safe_relative(value):
    require(isinstance(value, str) and value and not any(ord(c) < 32 for c in value), "bad relative path")
    path = PurePosixPath(value)
    require(not path.is_absolute() and "\\" not in value and ":" not in value, "unsafe relative path: " + value)
    require(all(part not in ("", ".", "..") for part in value.split("/")), "traversal/noncanonical path: " + value)
    require(str(path) == value, "noncanonical relative path: " + value)
    return value


def plain_ancestors(path):
    require(path.is_absolute(), "absolute path required")
    for part in [*reversed(path.parents), path]:
        if part.exists() or part.is_symlink():
            require(not part.is_symlink(), "symlink path: " + str(part))


def owned_path(value, prefix):
    path = Path(value)
    require(path.is_absolute() and path.parent == Path("/home/user"), "owned root must be a direct /home/user child")
    require(re.fullmatch(re.escape(prefix) + r"[A-Za-z0-9_-]+", path.name) is not None, "unexpected owned directory name")
    plain_ancestors(path)
    return path


def file_identity(path):
    plain_ancestors(path)
    before = path.lstat()
    require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1, "nonregular/hardlinked input: " + str(path))
    digest = hashlib.sha256()
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(fd, "rb") as stream:
        opened = os.fstat(stream.fileno())
        require((before.st_dev, before.st_ino) == (opened.st_dev, opened.st_ino), "input replaced before read")
        while chunk := stream.read(1024 * 1024):
            digest.update(chunk)
        after = os.fstat(stream.fileno())
    now = path.lstat()
    require((opened.st_dev, opened.st_ino, opened.st_size, opened.st_mtime_ns, opened.st_ctime_ns) ==
            (after.st_dev, after.st_ino, after.st_size, after.st_mtime_ns, after.st_ctime_ns) ==
            (now.st_dev, now.st_ino, now.st_size, now.st_mtime_ns, now.st_ctime_ns), "input changed during read")
    return {"Length": after.st_size, "Sha256": digest.hexdigest()}


def pinned_file(path, length, sha):
    actual = file_identity(path)
    require(actual == {"Length": length, "Sha256": sha}, "pinned file mismatch: " + str(path))
    return actual


def row_identity(rows):
    data = "".join(f"{r['Path']}\t{r['Length']}\t{r['Sha256'].lower()}\n" for r in rows)
    return hashlib.sha256(data.encode("utf-8")).hexdigest()


def ordinal(value):
    return value.encode("utf-16-be")


def load_manifest(path):
    pinned_file(path, MANIFEST_BYTES, MANIFEST_SHA)
    raw = strict_json(path)
    require(isinstance(raw, list) and len(raw) == INPUT_COUNT, "full input manifest count")
    rows, seen = [], set()
    for row in raw:
        require(isinstance(row, dict) and set(row) == {"Path", "Length", "Sha256"}, "invalid input row")
        name = safe_relative(row["Path"])
        require(name.casefold() not in seen, "case-colliding/duplicate input")
        seen.add(name.casefold())
        require(type(row["Length"]) is int and row["Length"] >= 0, "invalid input length")
        require(re.fullmatch(r"[0-9A-Fa-f]{64}", row["Sha256"]) is not None, "invalid input hash")
        require("remote-box-" not in name and "/tools/" not in name and "/build-cache/" not in name and "/bin/" not in name, "excluded preparation/cache input")
        rows.append({"Path": name, "Length": row["Length"], "Sha256": row["Sha256"].lower()})
    require(rows == sorted(rows, key=lambda r: ordinal(r["Path"])), "manifest order differs")
    require(sum(r["Length"] for r in rows) == INPUT_BYTES and row_identity(rows) == INPUT_ID, "aggregate input identity differs")
    return rows


def tree_rows(root):
    plain_ancestors(root)
    require(root.is_dir(), "missing tree: " + str(root))
    rows, directories = [], []
    pending = [root]
    while pending:
        directory = pending.pop()
        for item in os.scandir(directory):
            path = Path(item.path)
            require(not item.is_symlink(), "symlink in input tree: " + str(path))
            name = path.relative_to(root).as_posix()
            safe_relative(name)
            if item.is_dir(follow_symlinks=False):
                directories.append(name)
                pending.append(path)
            else:
                rows.append({"Path": name, **file_identity(path)})
    return sorted(rows, key=lambda r: ordinal(r["Path"])), set(directories)


def expected_directories(rows):
    result = set()
    for row in rows:
        for parent in PurePosixPath(row["Path"]).parents:
            if str(parent) != ".":
                result.add(str(parent))
    return result


def source_guard(root, expected, output):
    actual, dirs = tree_rows(root)
    write_json(output, actual)
    require(actual == expected and dirs == expected_directories(expected), "full source membership/bytes changed; actual manifest retained")
    by_path = {row["Path"]: row for row in actual}
    for name, digest in PINS.items():
        require(by_path.get(name, {}).get("Sha256") == digest, "fixed authority mismatch: " + name)
    require(by_path[FROZEN]["Length"] == 748, "frozen source manifest size")
    frozen = strict_json(root / FROZEN)
    require(isinstance(frozen, list) and len(frozen) == 5, "frozen source scope")
    for row in frozen:
        name = "native/" + safe_relative(row["Path"])
        require(by_path.get(name) == {"Path": name, "Length": row["Length"], "Sha256": row["Sha256"].lower()}, "picker input mismatch: " + name)
    patches = strict_json(root / PATCH)
    require(patches.get("schema") == 1 and patches.get("module") == "github.com/egoist/mygo v0.8.3" and len(patches.get("files", [])) == 26, "patch schema/module/count")
    seen = set()
    for row in patches["files"]:
        name = "native/" + safe_relative(row["path"])
        require(name not in seen and by_path.get(name, {}).get("Sha256") == row["sha256"], "patch entry mismatch: " + name)
        seen.add(name)
    app = (root / "native/vendor/github.com/egoist/mygo/app.go").read_text(encoding="utf-8")
    require("// LEVNOTE PATCH: this app does not enable MyGo updates" in app, "startup patch marker missing")
    require(re.search(r"\bgo\s+cleanUpdateLeftovers\s*\(", app) is None, "startup cleanup restored")
    require("# github.com/egoist/mygo v0.8.3" in (root / "native/vendor/modules.txt").read_text(encoding="utf-8"), "MyGo version mismatch")
    return {"Files": len(actual), "Bytes": sum(r["Length"] for r in actual), "Identity": row_identity(actual), "PickerFiles": 5, "PatchFiles": 26}


def extract_source(archive, root, expected):
    pinned_file(archive, ZIP_BYTES, ZIP_SHA)
    root.mkdir(mode=0o700)
    by_path = {r["Path"]: r for r in expected}
    seen = set()
    with zipfile.ZipFile(archive, "r") as zipped:
        for entry in zipped.infolist():
            name = safe_relative(entry.filename)
            require(name in by_path and name not in seen and not entry.is_dir(), "unexpected/duplicate archive member")
            require(not (entry.flag_bits & 1), "encrypted source archive")
            kind = stat.S_IFMT(entry.external_attr >> 16)
            require(kind in (0, stat.S_IFREG), "nonregular ZIP member")
            require(entry.file_size == by_path[name]["Length"], "ZIP member length differs")
            target = root / name
            target.parent.mkdir(parents=True, exist_ok=True)
            with zipped.open(entry) as inp, target.open("xb") as out:
                shutil.copyfileobj(inp, out, 1024 * 1024)
            os.chmod(target, 0o644)
            require(file_identity(target) == {k: by_path[name][k] for k in ("Length", "Sha256")}, "extracted source bytes differ")
            seen.add(name)
    require(seen == set(by_path), "source ZIP is incomplete")


def toolchain_guard(archive, root, output, extract=False):
    pinned_file(archive, GO_BYTES, GO_SHA)
    expected, names, dirs = [], set(), set()
    if extract:
        root.mkdir(mode=0o700)
    with tarfile.open(archive, "r:gz") as tar:
        for member in tar:
            name = safe_relative(member.name.rstrip("/"))
            require(name == "go" or name.startswith("go/"), "toolchain archive prefix")
            require(name not in names, "duplicate toolchain member")
            names.add(name)
            require(member.isdir() or member.isfile(), "linked/special toolchain entry refused")
            target = root / name
            if member.isdir():
                dirs.add(name)
                if extract:
                    target.mkdir(mode=0o755, parents=True, exist_ok=True)
                continue
            require(member.size >= 0, "invalid toolchain length")
            digest = hashlib.sha256()
            reader = tar.extractfile(member)
            require(reader is not None, "missing toolchain member data")
            destination = None
            try:
                if extract:
                    target.parent.mkdir(mode=0o755, parents=True, exist_ok=True)
                    destination = target.open("xb")
                while chunk := reader.read(1024 * 1024):
                    digest.update(chunk)
                    if destination is not None:
                        destination.write(chunk)
            finally:
                reader.close()
                if destination is not None:
                    destination.close()
            expected.append({"Path": name, "Length": member.size, "Sha256": digest.hexdigest()})
            if extract:
                os.chmod(target, 0o755 if member.mode & 0o111 else 0o644)
            require(bool(target.stat().st_mode & 0o111) == bool(member.mode & 0o111), "toolchain executable bits differ")
    expected.sort(key=lambda r: ordinal(r["Path"]))
    actual, actual_dirs = tree_rows(root)
    write_json(output, actual)
    require(actual == expected and actual_dirs == dirs | expected_directories(expected), "extracted full toolchain differs from verified archive")
    version = (root / "go/VERSION").read_text(encoding="utf-8").splitlines()[0]
    require(version == "go1.27.3", "toolchain VERSION differs")
    go = root / "go/bin/go"
    with go.open("rb") as inp:
        elf = inp.read(20)
    require(elf[:6] == b"\x7fELF\x02\x01" and int.from_bytes(elf[18:20], "little") == 62, "Go executable is not ELF64 little-endian x86_64")
    return {"Files": len(actual), "Bytes": sum(r["Length"] for r in actual), "Identity": row_identity(actual), "GoBinary": file_identity(go), "Archive": GO_SHA, "VersionFile": version, "Executed": False}


def clean_environment(run_root, phase):
    state = run_root / ("state-" + phase)
    state.mkdir(mode=0o700)
    children = ["home", "config", "cache", "data", "runtime", "tmp", "go-tmp", "gopath", "gocache", "telemetry"]
    for name in children:
        (state / name).mkdir(mode=0o700)
    (state / "telemetry/mode").write_bytes(b"off\n")
    go_root = run_root / "toolchain/go"
    return {
        "PATH": str(go_root / "bin") + ":/usr/bin:/bin", "HOME": str(state / "home"),
        "XDG_CONFIG_HOME": str(state / "config"), "XDG_CACHE_HOME": str(state / "cache"),
        "XDG_DATA_HOME": str(state / "data"), "XDG_RUNTIME_DIR": str(state / "runtime"),
        "TMPDIR": str(state / "tmp"), "TMP": str(state / "tmp"), "TEMP": str(state / "tmp"),
        "GOTMPDIR": str(state / "go-tmp"), "GOROOT": str(go_root), "GOPATH": str(state / "gopath"),
        "GOCACHE": str(state / "gocache"), "GOMODCACHE": str(state / "gopath/pkg/mod"),
        "GOTOOLCHAIN": "local", "GOWORK": "off", "GOENV": "off", "GOFLAGS": "",
        "GOOS": "linux", "GOARCH": "amd64", "GOAMD64": "v1", "CGO_ENABLED": "0",
        "GOPROXY": "off", "GOSUMDB": "off", "GOVCS": "*:off",
        "GONOPROXY": "", "GONOSUMDB": "", "GOPRIVATE": "", "GOTELEMETRY": "off",
        "TEST_TELEMETRY_DIR": str(state / "telemetry"),
        "LANG": "C.UTF-8", "LC_ALL": "C.UTF-8", "TZ": "UTC", "DISPLAY": ":0",
    }


def captured(argv, cwd, env, directory, timeout):
    directory.mkdir(mode=0o700)
    start = {"StartedUtc": utc(), "Argv": argv, "Cwd": str(cwd), "Environment": env, "TimeoutSeconds": timeout}
    write_json(directory / "command.json", start)
    process, error, timeout_hit = None, None, False
    with (directory / "stdout.bin").open("xb") as stdout, (directory / "stderr.bin").open("xb") as stderr:
        try:
            process = subprocess.Popen(argv, cwd=cwd, env=env, stdin=subprocess.DEVNULL, stdout=stdout, stderr=stderr, start_new_session=True, shell=False)
            process.wait(timeout=timeout)
        except BaseException as exc:
            error = type(exc).__name__ + ": " + str(exc)
            timeout_hit = isinstance(exc, subprocess.TimeoutExpired)
            if process is not None and process.poll() is None:
                os.killpg(process.pid, signal.SIGTERM)
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    os.killpg(process.pid, signal.SIGKILL)
                    process.wait(timeout=5)
    result = {**start, "FinishedUtc": utc(), "PID": None if process is None else process.pid,
              "ReturnCode": None if process is None else process.returncode, "TimedOut": timeout_hit, "Error": error}
    result["Signal"] = -process.returncode if process is not None and process.returncode is not None and process.returncode < 0 else None
    result["Stdout"] = file_identity(directory / "stdout.bin")
    result["Stderr"] = file_identity(directory / "stderr.bin")
    write_json(directory / "exit.json", result)
    return result


def platform_guard():
    require(sys.flags.isolated and sys.dont_write_bytecode, "invoke Python with -I -B")
    require(sys.version_info >= (3, 10), "Python >=3.10 required")
    require(platform.system() == "Linux" and platform.machine() == "x86_64", "remote Linux amd64 only")
    require(os.geteuid() != 0, "do not run app tests as root")
    release = Path("/etc/os-release").read_text(encoding="utf-8")
    require(re.search(r'^ID=ubuntu$', release, re.M) is not None and re.search(r'^VERSION_ID="24\.04"$', release, re.M) is not None, "Ubuntu24.04 required")
    require(not any(os.environ.get(key) for key in ("LD_PRELOAD", "LD_LIBRARY_PATH", "LD_AUDIT")), "ambient loader override refused")
    require(stat.S_ISSOCK(Path("/tmp/.X11-unix/X0").stat().st_mode), "expected display socket absent")
    return {"OSRelease": release, "System": platform.system(), "Machine": platform.machine(), "Python": sys.version,
            "UID": os.geteuid(), "Display": ":0", "XSocketMetadataOnly": True,
            "BoxID": BOX, "BoxBinding": "caller must independently bind this shell invocation to main/cb-698f697fed7c; not inferred from OS",
            "CredentialFilesRead": False}


def prerequisites(source, env, output):
    output.mkdir(mode=0o700)
    prior = dict(os.environ)
    try:
        os.environ.clear()
        os.environ.update(env)
        libraries = {"gobject": "libgobject-2.0.so.0", "pango": "libpango-1.0.so.0", "glib": "libglib-2.0.so.0", "pangocairo": "libpangocairo-1.0.so.0", "cairo": "libcairo.so.2"}
        loaded = {key: ctypes.CDLL(name, mode=os.RTLD_NOW | os.RTLD_GLOBAL) for key, name in libraries.items()}
        code = (source / "native/vendor/github.com/egoist/mygo/internal/text/pango_linux.go").read_text(encoding="utf-8")
        require(code.startswith("//go:build linux && (amd64 || arm64)"), "Pango platform constraint differs")
        mandatory = code.split("func loadPango() error {", 1)[1].split("// Optional:", 1)[0]
        bindings = re.findall(r'bind\((gobject|pango|glib|pangocairo|cairo),\s*&l\.\w+,\s*"([^"]+)"\)', mandatory)
        require(len(bindings) >= 60, "mandatory Pango symbol extraction unexpectedly small")
        for library, symbol in bindings:
            getattr(loaded[library], symbol)
        create_map = loaded["pangocairo"].pango_cairo_font_map_new
        create_map.argtypes, create_map.restype = [], ctypes.c_void_p
        create_context = loaded["pango"].pango_font_map_create_context
        create_context.argtypes, create_context.restype = [ctypes.c_void_p], ctypes.c_void_p
        unref = loaded["gobject"].g_object_unref
        unref.argtypes, unref.restype = [ctypes.c_void_p], None
        font_map = create_map()
        require(bool(font_map), "Pango font map creation failed; stub-equivalent prerequisite failure")
        context = None
        try:
            context = create_context(font_map)
            require(bool(context), "Pango context creation failed")
        finally:
            if context:
                unref(context)
            unref(font_map)
        matches = []
        for index, pattern in enumerate(("sans-serif", "monospace", "sans-serif:lang=zh-cn"), 1):
            command = captured(["/usr/bin/fc-match", "--format", "%{family}\t%{file}\n", pattern], output, env, output / f"font-{index}", 30)
            require(command["ReturnCode"] == 0 and command["Error"] is None, "fontconfig match failed")
            raw = (output / f"font-{index}/stdout.bin").read_bytes().decode("utf-8", errors="strict").strip()
            require("\t" in raw, "fontconfig returned no font file")
            family, filename = raw.rsplit("\t", 1)
            actual = Path(filename).resolve(strict=True)
            require(str(actual).startswith("/usr/share/fonts/"), "unexpected font source outside fixed system font tree")
            # System font files may have distribution hard links; source/toolchain
            # files remain strictly single-link. Hash system bytes directly.
            matches.append({"Pattern": pattern, "Family": family, "Path": str(actual), "Length": actual.stat().st_size,
                            "Sha256": hashlib.sha256(actual.read_bytes()).hexdigest()})
        result = {"Libraries": libraries, "MandatorySymbols": bindings, "FontMapAndContextCreated": True,
                  "FontMatches": matches, "ProbeProcess": "Python ctypes, separate from Go/MyGo",
                  "MyGoBackendObserved": False, "PhysicalUIOrIMEObserved": False,
                  "Limit": "Prerequisite and pinned platform-source evidence only. A successful nonverbose go test can suppress logs; no backend proof is inferred from log absence."}
        write_json(output / "prerequisites.json", result)
        return result
    finally:
        os.environ.clear()
        os.environ.update(prior)


def main():
    parser = argparse.ArgumentParser(allow_abbrev=False)
    parser.add_argument("phase", choices=("prepare", "test", "vet"))
    parser.add_argument("--box-id", required=True, choices=(BOX,))
    parser.add_argument("--run-root", required=True)
    parser.add_argument("--inbox")
    parser.add_argument("--guard-receipt-sha256")
    parser.add_argument("--activate-product-stage", choices=(ACTIVATION,))
    args = parser.parse_args()
    facts = platform_guard()
    run_root = owned_path(args.run_root, "levnote-remote-20261111-")
    wrapper = Path(__file__).resolve(strict=True)
    wrapper_id = file_identity(wrapper)
    if args.phase == "prepare":
        require(args.inbox and args.guard_receipt_sha256 is None and args.activate_product_stage is None, "prepare accepts inbox only, no product activation")
        inbox = owned_path(args.inbox, "levnote-inbox-20261111-")
        require(inbox.is_dir(), "missing transferred inbox")
        require(not run_root.exists(), "fresh run root required; do not overwrite failed attempts")
        run_root.mkdir(mode=0o700)
        (run_root / "transport").mkdir(mode=0o700)
        (run_root / "evidence").mkdir(mode=0o700)
    else:
        require(args.inbox is None and args.activate_product_stage == ACTIVATION, "product stage requires explicit separate activation, no alternate inputs")
        require(args.guard_receipt_sha256 and re.fullmatch(r"[0-9a-fA-F]{64}", args.guard_receipt_sha256), "reviewed guard receipt SHA256 required")
        receipt_path = run_root / "guard-ready.json"
        receipt_id = file_identity(receipt_path)
        require(receipt_id["Sha256"] == args.guard_receipt_sha256.lower(), "guard receipt differs")
        receipt = strict_json(receipt_path)
        require(receipt["Status"] == "GUARD_READY_NOT_PRODUCT_ACCEPTANCE" and receipt["Wrapper"] == wrapper_id and receipt["RunRoot"] == str(run_root), "guard context differs")
        if args.phase == "vet":
            test_result = strict_json(run_root / "evidence/test-01/result.json")
            require(test_result["Status"] == "COMMAND_PASSED_WITH_GUARDS" and test_result["ProductExecuted"] is True, "failed/missing test holds vet for root review")
    evidence = run_root / "evidence" / (args.phase + "-01")
    evidence.mkdir(mode=0o700)
    write_json(evidence / "invocation.json", {"StartedUtc": utc(), "Argv": sys.argv, "Thread": THREAD, "Platform": facts, "Wrapper": wrapper_id,
               "DroppedAmbientVariableNames": sorted(key for key in os.environ if key.startswith(("GO", "LEVNOTE_", "PYTHON", "XDG_"))),
               "NoAmbientVariableValuesCopied": True})
    status = {"Phase": args.phase, "StartedUtc": utc(), "Status": "FAILED", "ProductExecuted": False, "Platform": facts}
    source, toolchain, transport = run_root / "source", run_root / "toolchain", run_root / "transport"
    expected = None
    exit_code = 2
    try:
        if args.phase == "prepare":
            for name, length, sha in ((MANIFEST_NAME, MANIFEST_BYTES, MANIFEST_SHA), (ZIP_NAME, ZIP_BYTES, ZIP_SHA), (GO_ARCHIVE, GO_BYTES, GO_SHA)):
                pinned_file(inbox / name, length, sha)
                with (inbox / name).open("rb") as src, (transport / name).open("xb") as dst:
                    shutil.copyfileobj(src, dst, 1024 * 1024)
                pinned_file(transport / name, length, sha)
        expected = load_manifest(transport / MANIFEST_NAME)
        pinned_file(transport / ZIP_NAME, ZIP_BYTES, ZIP_SHA)
        if args.phase == "prepare":
            extract_source(transport / ZIP_NAME, source, expected)
        status["InputsBefore"] = source_guard(source, expected, evidence / "inputs-before.json")
        status["ToolchainBefore"] = toolchain_guard(transport / GO_ARCHIVE, toolchain, evidence / "toolchain-before.json", extract=args.phase == "prepare")
        env = clean_environment(run_root, args.phase)
        write_json(evidence / "environment.json", env)
        status["PrerequisitesBefore"] = prerequisites(source, env, evidence / "prerequisites-before")
        if args.phase != "prepare":
            go = str(toolchain / "go/bin/go")
            status["ProductExecuted"] = True
            result = captured([go, *PRODUCT_ARGV[args.phase]], source / "native", env, evidence / "go-command", 900)
            status["Command"] = result
            # Positive fallback logs invalidate evidence. Their absence is not
            # claimed to identify the backend because go test is nonverbose.
            combined = (evidence / "go-command/stdout.bin").read_bytes() + (evidence / "go-command/stderr.bin").read_bytes()
            require(b"text will not show" not in combined, "observed MyGo text fallback; UI evidence invalid")
            require(result["ReturnCode"] == 0 and result["Error"] is None, "Go command failed; original return code/streams retained")
        status["CandidateSuccess"] = True
    except BaseException as exc:
        status["Error"] = type(exc).__name__ + ": " + str(exc)
        with (evidence / "exception.txt").open("x", encoding="utf-8") as out:
            traceback.print_exc(file=out)
    finally:
        post_errors = []
        if expected is not None and source.exists():
            try:
                status["InputsAfter"] = source_guard(source, expected, evidence / "inputs-after.json")
            except BaseException as exc:
                post_errors.append("source: " + str(exc))
        if toolchain.exists():
            try:
                status["ToolchainAfter"] = toolchain_guard(transport / GO_ARCHIVE, toolchain, evidence / "toolchain-after.json")
            except BaseException as exc:
                post_errors.append("toolchain: " + str(exc))
        status["PostGuardErrors"] = post_errors
        if status.get("CandidateSuccess") and not post_errors and "Error" not in status:
            status["Status"] = "GUARD_READY_NOT_PRODUCT_ACCEPTANCE" if args.phase == "prepare" else "COMMAND_PASSED_WITH_GUARDS"
            exit_code = 0
        status["FinishedUtc"] = utc()
        status["WrapperExitCode"] = exit_code
        status["RunRoot"] = str(run_root)
        status["Wrapper"] = wrapper_id
        status["NoPhysicalUIIMEOrWindowsDeliveryClaim"] = True
        write_json(evidence / "result.json", status)
    if args.phase == "prepare" and exit_code == 0:
        write_json(run_root / "guard-ready.json", {"Status": status["Status"], "FinishedUtc": status["FinishedUtc"], "RunRoot": str(run_root),
                   "Wrapper": wrapper_id, "InputIdentity": INPUT_ID, "ManifestSHA256": MANIFEST_SHA, "SourceArchiveSHA256": ZIP_SHA,
                   "ToolchainArchiveSHA256": GO_SHA, "Evidence": "evidence/prepare-01/result.json", "ProductExecuted": False})
    print(json.dumps({"Phase": args.phase, "Status": status["Status"], "Evidence": str(evidence / "result.json"), "ExitCode": exit_code}))
    return exit_code


if __name__ == "__main__":
    try:
        sys.exit(main())
    except Exception as exc:
        # The remote caller must also capture outer wrapper streams/exits;
        # argument/path/platform rejection can precede creation of evidence.
        print(type(exc).__name__ + ": " + str(exc), file=sys.stderr)
        sys.exit(2)