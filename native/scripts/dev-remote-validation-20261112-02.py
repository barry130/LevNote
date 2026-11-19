#!/usr/bin/env python3
"""Task-5 ordinary Linux development lane. Never a Windows acceptance runner.

Each invocation uses a new label, snapshots explicitly pinned native inputs,
records actual commands/raw streams/exits, and leaves all evidence in place.
This helper never synchronizes sources, fetches, commits, pushes, or edits the
ordinary clone. Lead coordinates --expected-manifest and any source overlays.
"""

import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import sys
import time
import traceback

LANE = Path("/home/user/levnote-dev-reliability-20261112-01")
CLONE = Path("/home/user/LevNote")
GO_ROOT = Path("/home/user/go")
DEV_ENV = Path("/home/user/levnote-dev-env.sh")
INPUT_ROOTS = ("go.mod", "go.sum", "app", "cmd", "core", "formats", "modules", "ui", "vendor", "dependency-patches", "examples")
ENV_KEYS = ("HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "PATH", "GOROOT", "GOPATH", "GOCACHE", "GOTMPDIR", "TMPDIR", "GOENV", "GOWORK", "GOVCS", "GOFLAGS", "GOTOOLCHAIN", "GOPROXY", "GOSUMDB", "GONOPROXY", "GONOSUMDB", "GOPRIVATE", "CGO_ENABLED", "GOTELEMETRY", "GOOS", "GOARCH", "GOEXPERIMENT")
AUDITED_DEV_ENV = "\n".join(("export PATH=/home/user/go/bin:$PATH", "export GOPATH=/home/user/gopath", "export GOCACHE=/home/user/gocache", "export GOTOOLCHAIN=local", "export GOFLAGS=-mod=vendor", "export CGO_ENABLED=0", ""))


def utc():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def file_identity(path):
    data = Path(path).read_bytes()
    return {"bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()}


def write_json(path, value):
    with Path(path).open("x", encoding="utf-8") as f:
        json.dump(value, f, ensure_ascii=False, indent=2)
        f.write("\n")


def safe_path(path):
    path = Path(path)
    if not path.is_absolute() or path.resolve() != path:
        raise ValueError("refuse missing, relative, or symlinked path: " + str(path))
    return path


def inventory(root):
    """Only runtime/build/test inputs; no historical handoff or release assets."""
    files = {}
    for name in INPUT_ROOTS:
        base = root / name
        if not base.exists():
            raise ValueError("required native input root is missing: " + name)
        candidates = [base] if base.is_file() else sorted(base.rglob("*"))
        for path in candidates:
            if path.is_symlink():
                raise ValueError("source input symlink is not supported: " + str(path))
            if path.is_file():
                files[path.relative_to(root).as_posix()] = file_identity(path)
    return {"roots": list(INPUT_ROOTS), "files": dict(sorted(files.items()))}


def make_environment(run_root):
    env_root = run_root / "env"
    env_root.mkdir()
    for name in ("home", "config", "cache", "data", "state", "tmp"):
        (env_root / name).mkdir()
    for name in ("gopath", "gocache"):
        (LANE / name).mkdir(exist_ok=True)
    telemetry = env_root / "config" / "go" / "telemetry"
    telemetry.mkdir(parents=True)
    # GOTELEMETRY is an observation, not sufficient to disable collection. The
    # audited toolchain reads this owned XDG mode file before any Go invocation.
    (telemetry / "mode").write_text("off\n", encoding="ascii")
    env = dict(os.environ)
    env.update({"HOME": str(env_root / "home"), "XDG_CONFIG_HOME": str(env_root / "config"),
                "XDG_CACHE_HOME": str(env_root / "cache"), "XDG_DATA_HOME": str(env_root / "data"),
                "XDG_STATE_HOME": str(env_root / "state"), "TMPDIR": str(env_root / "tmp"),
                "GOTMPDIR": str(env_root / "tmp"), "PATH": str(GO_ROOT / "bin") + ":/usr/local/bin:/usr/bin:/bin",
                "GOROOT": str(GO_ROOT), "GOPATH": str(LANE / "gopath"), "GOCACHE": str(LANE / "gocache"),
                "GOENV": "off", "GOWORK": "off", "GOVCS": "*:off", "GOTOOLCHAIN": "local", "GOFLAGS": "-mod=vendor", "GOPROXY": "off",
                "GOSUMDB": "off", "GONOPROXY": "", "GONOSUMDB": "", "GOPRIVATE": "", "CGO_ENABLED": "0",
                "GOTELEMETRY": "off", "GOOS": "linux", "GOARCH": "amd64", "GOEXPERIMENT": ""})
    return env


def run_command(evidence, label, argv, cwd, env, timeout):
    rec = {"argv": argv, "cwd": str(cwd), "environment": {key: env.get(key) for key in ENV_KEYS},
           "environment_capture": "allowlisted development keys; no credentials or proxy inspection",
           "started_at": utc(), "timeout_s": timeout, "timed_out": False, "owned_cleanup": []}
    start = time.monotonic()
    rec["status"] = "prepared"
    # Persist intent before spawn; a connector interruption must not erase the
    # distinction between a prepared command and an actual started process.
    write_json(evidence / (label + ".command.json"), rec)
    stdout_path, stderr_path = evidence / (label + ".stdout.bin"), evidence / (label + ".stderr.bin")
    with stdout_path.open("xb") as stdout, stderr_path.open("xb") as stderr:
        proc = subprocess.Popen(argv, cwd=cwd, env=env, stdin=subprocess.DEVNULL,
                                stdout=stdout, stderr=stderr, start_new_session=True)
        # start_new_session establishes this child as leader of its own group.
        rec.update({"pid": proc.pid, "owned_pgid": proc.pid, "launched_at": utc(), "status": "started"})
        write_json(evidence / (label + ".started.json"), rec)
        try:
            proc.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            rec["timed_out"] = True
            try:
                os.killpg(proc.pid, signal.SIGTERM)
                rec["owned_cleanup"].append({"pgid": proc.pid, "signal": "SIGTERM"})
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(proc.pid, signal.SIGKILL)
                rec["owned_cleanup"].append({"pgid": proc.pid, "signal": "SIGKILL"})
                proc.wait()
            except ProcessLookupError:
                proc.wait()
        rec["exit_code"] = proc.returncode
    # Only this freshly-created process group is inspected/cleaned; never a
    # system-wide process list, name match, shared cache, or historical caller.
    try:
        os.killpg(proc.pid, 0)
        rec["owned_group_remaining_after_wait"] = True
        os.killpg(proc.pid, signal.SIGKILL)
        rec["owned_cleanup"].append({"pgid": proc.pid, "signal": "SIGKILL", "reason": "owned descendants after command exit"})
    except ProcessLookupError:
        rec["owned_group_remaining_after_wait"] = False
    except OSError as exc:
        rec["owned_cleanup_error"] = str(exc)
    rec.update({"status": "completed", "ended_at": utc(), "elapsed_s": time.monotonic() - start,
                "stdout": {"path": str(stdout_path), **file_identity(stdout_path)},
                "stderr": {"path": str(stderr_path), **file_identity(stderr_path)}})
    write_json(evidence / (label + ".json"), rec)
    return rec, stdout_path.read_bytes()


def checked_command(evidence, label, argv, cwd, env, timeout):
    rec, data = run_command(evidence, label, argv, cwd, env, timeout)
    if rec["exit_code"] or rec["timed_out"] or rec["owned_cleanup"] or rec.get("owned_cleanup_error"):
        raise RuntimeError(label + " failed; raw command evidence is retained")
    return rec, data


def validate_go_arguments(arguments):
    # This lane runs pinned packages, not caller-supplied tools or workspaces.
    # Match split flags and flag=value forms, including test-binary wrappers.
    forbidden = ("-C", "-mod", "-modfile", "-overlay", "-tags", "-toolexec", "-exec", "-test.exec",
                 "-vettool", "-args", "-o", "-outputdir", "-test.outputdir", "-pkgdir",
                 "-gcflags", "-asmflags", "-ldflags", "-coverprofile", "-memprofile", "-cpuprofile",
                 "-trace", "-blockprofile", "-mutexprofile")
    for value in arguments:
        if value == "--" or value.startswith("/") or value in forbidden or any(value.startswith(flag + "=") for flag in forbidden):
            raise ValueError("alternate command/module/platform/tool/output inputs are not permitted: " + value)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--label", required=True)
    parser.add_argument("--expected-head", required=True)
    parser.add_argument("--expected-manifest")
    parser.add_argument("--source", default=str(CLONE / "native"))
    parser.add_argument("--mode", choices=("inspect", "test", "vet", "format"), default="inspect")
    parser.add_argument("--timeout", type=int, default=480)
    parser.add_argument("arguments", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    if not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,79}", args.label):
        parser.error("label must be a fresh safe lowercase name")
    if not re.fullmatch(r"[0-9a-f]{40}", args.expected_head) or not 1 <= args.timeout <= 1800:
        parser.error("exact HEAD and a 1..1800-second command timeout are required")
    safe_path(LANE)
    safe_path(CLONE)
    safe_path(GO_ROOT)
    source = safe_path(args.source)
    if source != CLONE / "native" and not source.is_relative_to(LANE / "inputs"):
        parser.error("source must be the ordinary native clone or a lead-coordinated input under this lane/inputs")
    run_root = LANE / "runs" / args.label
    run_root.mkdir(parents=True, exist_ok=False)
    evidence = run_root / "evidence"
    evidence.mkdir()
    summary = {"owner": "native-reliability/task-5", "started_at": utc(), "argv": sys.argv,
               "wrapper_cwd": str(Path.cwd()), "python": sys.executable,
               "runner_input": {"path": str(Path(__file__).resolve()), **file_identity(__file__)},
               "mode": args.mode, "source_root": str(source), "expected_head": args.expected_head,
               "command_timeout_s": args.timeout, "not_windows_or_whole_application_acceptance": True}
    code = 2
    write_json(evidence / "invocation.json", summary)
    with (evidence / "wrapper.stdout.txt").open("x", encoding="utf-8") as wrapper:
        def emit(text):
            wrapper.write(text + "\n")
            wrapper.flush()
            print(text, flush=True)

        try:
            if DEV_ENV.read_text() != AUDITED_DEV_ENV:
                raise ValueError("ordinary dev env differs from the audited six exports")
            summary["ordinary_env_input"] = {"path": str(DEV_ENV), **file_identity(DEV_ENV)}
            summary["ordinary_env_use"] = "audited toolchain/vendor/CGO/local policy reused; HOME/cache/tmp isolated; no old guard sourcing"
            summary["go_version_file"] = file_identity(GO_ROOT / "VERSION")
            env = make_environment(run_root)
            _, head = checked_command(evidence, "01-clone-head", ["/usr/bin/git", "rev-parse", "HEAD"], CLONE, env, 15)
            if head.decode().strip() != args.expected_head:
                raise ValueError("ordinary clone HEAD differs; lead must coordinate source synchronization")
            checked_command(evidence, "02-clone-status", ["/usr/bin/git", "status", "--porcelain=v1", "--untracked-files=all"], CLONE, env, 20)
            before = inventory(source)
            write_json(evidence / "source-before.json", before)
            summary["source_before_manifest"] = file_identity(evidence / "source-before.json")
            if args.expected_manifest:
                expected_path = safe_path(args.expected_manifest)
                if not expected_path.is_relative_to(LANE):
                    raise ValueError("expected input manifest must be in the owned lane")
                expected = json.loads(expected_path.read_bytes())
                summary["expected_input"] = {"path": str(expected_path), **file_identity(expected_path)}
                if expected.get("head") != args.expected_head or any(expected.get(key) != before[key] for key in ("roots", "files")):
                    raise ValueError("current source bytes do not match the lead-coordinated expected input")
            elif args.mode != "inspect":
                raise ValueError("product command refused without an explicit synchronized-input manifest")
            _, version = checked_command(evidence, "03-go-version", [str(GO_ROOT / "bin" / "go"), "version"], run_root, env, 20)
            if version.decode().strip() != "go version go1.27.3 linux/amd64":
                raise ValueError("required ordinary Go1.27.3 Linux amd64 is unavailable")
            _, effective_data = checked_command(evidence, "04-go-env", [str(GO_ROOT / "bin" / "go"), "env", "-json", "GOVERSION", "GOOS", "GOARCH", "GOROOT", "GOENV", "GOWORK", "GOVCS", "GOTOOLCHAIN", "GOFLAGS", "GOPROXY", "GOSUMDB", "CGO_ENABLED", "GOTELEMETRY", "GOTELEMETRYDIR"], run_root, env, 20)
            effective = json.loads(effective_data)
            # cfg.EnvFile returns an empty filename for the GOENV=off policy.
            required = {"GOVERSION": "go1.27.3", "GOOS": "linux", "GOARCH": "amd64", "GOROOT": str(GO_ROOT),
                        "GOENV": "", "GOWORK": "off", "GOVCS": "*:off", "GOTOOLCHAIN": "local", "GOFLAGS": "-mod=vendor", "GOPROXY": "off",
                        "GOSUMDB": "off", "CGO_ENABLED": "0", "GOTELEMETRY": "off",
                        "GOTELEMETRYDIR": str(run_root / "env" / "config" / "go" / "telemetry")}
            if any(effective.get(key) != value for key, value in required.items()):
                raise ValueError("effective Go/offline/vendor/telemetry environment mismatch")
            summary["effective_go_environment"] = effective
            arguments = args.arguments[1:] if args.arguments[:1] == ["--"] else args.arguments
            if args.mode != "inspect":
                if not arguments:
                    raise ValueError("explicit focused command arguments are required")
                snapshot = run_root / "source" / "native"
                snapshot.mkdir(parents=True)
                for name in before["files"]:
                    target = snapshot / name
                    target.parent.mkdir(parents=True, exist_ok=True)
                    with target.open("xb") as f:
                        f.write((source / name).read_bytes())
                if inventory(snapshot) != before:
                    raise ValueError("snapshot bytes changed during copy; no product command executed")
                if args.mode == "format":
                    if any(not name.endswith(".go") or name not in before["files"] for name in arguments):
                        raise ValueError("format accepts only exact pinned Go input files")
                    command = [str(GO_ROOT / "bin" / "gofmt"), "-w", *arguments]
                else:
                    validate_go_arguments(arguments)
                    command = [str(GO_ROOT / "bin" / "go"), args.mode, *arguments]
                rec, data = run_command(evidence, "05-product-command", command, snapshot, env, args.timeout)
                summary["product_command"] = rec
                emit(data.decode("utf-8", "replace"))
                emit(Path(rec["stderr"]["path"]).read_bytes().decode("utf-8", "replace"))
                write_json(evidence / "snapshot-after.json", inventory(snapshot))
                if args.mode != "format" and inventory(snapshot) != before:
                    raise ValueError("product command modified pinned snapshot inputs")
                summary["formatted_copy"] = str(snapshot) if args.mode == "format" else None
                code = 124 if rec["timed_out"] else 2 if rec["exit_code"] else 0
                if rec["owned_cleanup"] or rec.get("owned_cleanup_error"):
                    code = 3
                summary["outcome"] = "COMMAND_PASSED_LIMITED_SCOPE" if code == 0 else "COMMAND_FAILED"
            else:
                code = 0
                summary["outcome"] = "ENVIRONMENT_AND_SOURCE_INSPECTED_NOT_PRODUCT_ACCEPTANCE"
            after = inventory(source)
            write_json(evidence / "source-after.json", after)
            if after != before:
                raise ValueError("source changed during execution; result not applicable to current diff")
            _, head_after = checked_command(evidence, "06-clone-head-after", ["/usr/bin/git", "rev-parse", "HEAD"], CLONE, env, 15)
            if head_after != head:
                raise ValueError("clone HEAD changed during this invocation")
        except BaseException:
            text = traceback.format_exc()
            with (evidence / "wrapper.stderr.txt").open("x", encoding="utf-8") as f:
                f.write(text)
            emit(text)
            summary["outcome"] = "RUNNER_OR_INPUT_CHECK_FAILED"
            code = 2
        summary.update({"exit_code": code, "ended_at": utc(), "evidence": str(evidence)})
        write_json(evidence / "result.json", summary)
        emit("evidence=" + str(evidence) + " wrapper_exit=" + str(code) + " outcome=" + summary["outcome"])
    return code


if __name__ == "__main__":
    os.umask(0o077)
    raise SystemExit(main())
