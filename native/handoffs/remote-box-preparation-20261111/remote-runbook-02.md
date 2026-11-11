# Remote handoff / guard draft 02

Status: preparation only; wrapper has NOT been executed or syntax-checked.
Original conversation: 01a1b6b1-3caf-76b1-81e2-183c090b7b86.
The only later remote target is main / cb-698f697fed7c, Ubuntu24.04 Linux amd64.

## Files to transfer through an approved private byte channel

- levnote-source-inputs-20261111-02.zip
- inputs-02.json
- linux-guarded-validation-02.py

Use transfer-files-02.json and this stage result for exact size/hash checks.
Do not transfer snapshot01 or historical evidence as execution inputs. A local
expanded byte copy is retained in snapshot-02 for inspection. No upload channel
has been exercised or demonstrated here. DSH shell/text output is not evidence
that binary upload/download is safe. Without an authorized private file channel,
hand the ZIP to the user; do not publish it or open an upload service.

The remote inbox convention is a NEW `/home/user/levnote-inbox-20261111-01`;
if it exists, choose a new suffix and preserve it. DSH must bind every operation
to the exact box ID using its own authorized connector. The wrapper records the
caller-provided box ID but cannot independently identify a remote service box
from Linux OS metadata. It never accesses token stores or Xauthority.

## Toolchain, future migration guard, then review

The same inbox will also need the official go1.27.3.linux-amd64.tar.gz downloaded
on the remote only after authorization. DSH observed its official record/HEAD,
not archive-body bytes or an installed toolchain. Expected bytes: 70,588,678;
SHA256: 65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40.
No toolchain is included in the source ZIP and none was downloaded here.

After safe transfer and a separate migration-guard activation, the intended
non-product command is:

```text
python3 -I -B /home/user/levnote-inbox-20261111-01/linux-guarded-validation-02.py prepare --box-id cb-698f697fed7c --inbox /home/user/levnote-inbox-20261111-01 --run-root /home/user/levnote-remote-20261111-01
```

This command is documented, not executed. `prepare` requires fresh run storage,
pins ZIP/JSON bytes, safely extracts source files, validates all 1,208 inputs,
checks the original five picker files and every one of the 26 patch entries,
startup marker/forbidden cleanup call, module pins and MyGo v0.8.3. It verifies
the official Go archive and compares every extracted toolchain file against it,
checks VERSION plus the go executable ELF64 x86_64 header, without invoking Go.
Paths, duplicate/extra members, links and alternative roots are rejected.

Its source, toolchain, transport, state and evidence are separate siblings.
HOME/XDG/cache/temp/telemetry/GOPATH are run-owned; input-tree hashes are checked
again afterward. The guard runs metadata/prerequisite tools (ldconfig, fc-match
and ctypes), records raw separate subprocess streams and returns a guard-ready
receipt only when all guards pass. The external DSH caller must also capture
outer command UTC, raw stdout/stderr and exit: early argument/platform/path
rejections can precede internal evidence creation. Preparation success means
neither app tests nor Windows delivery have passed.

Root must inspect the real remote result and complete backend/platform concerns
before activating products. Shared library/font presence and ctypes results
remain separate from observed MyGo text backend behavior. Missing library,
font, byte identity or Linux compatibility is a stop-and-report condition.

## Product gate remains inactive

The draft's only Go command arrays are `test -mod=vendor ./app -count=1` and
`vet -mod=vendor ./app`. No passthrough arguments, overlay/modfile/-C/workspace,
ambient GOFLAGS or replacement executable can select other inputs. Both receive
full source/26-patch/toolchain pre/post guards and a clean offline environment:
GOTOOLCHAIN=local, GOWORK=off, GOENV=off, GOPROXY=off, GOSUMDB=off, GOVCS=*:off,
CGO_ENABLED=0, explicit vendor and isolated telemetry. All ambient LEVNOTE_*
opt-ins are excluded. DISPLAY=:0 and C.UTF-8 are explicit; credential and
session environment is not imported.

A future test/vet invocation requires the exact SHA256 of the reviewed remote
guard-ready.json and `--activate-product-stage picker-final-app-linux-20261111`.
These are deliberate workflow interlocks, not a security sandbox or automatic
permission grant. Each phase has a create-only evidence directory; no implicit
retry or overwrite. Vet is held if the test failed, until root reviews the real
failure. The wrapper records ProductAttempted separately from actual Spawned /
ProductExecuted, raw binary stdout/stderr, UTC argv/cwd/environment, real child
return code/signal and cleanup errors. Wrapper exit 2 is a failed guard/command;
the original Go exit is separately retained in go-command/exit.json.

Successful source hashes cannot turn a failing Linux test into a pass. No source
rewrite, test skip, broadening suite, screenshots, workers, resources, packaging
or product host run is authorized by this runbook.

## Windows limitation

The existing go.ps1 invokes Windows tools/go/bin/go.exe and bootstrap.mjs fetches
a Windows ZIP. The frozen .30 helper chain requires GUI PE, five GUI builds,
C: configuration, WM_CLOSE and Windows process/Job cleanup. Linux tests,
cross-compilation or Wine do not replace that acceptance. .29 remains the latest
accepted runnable, .30 remains deferred, and no historical Windows evidence is
relabeled as remote/Linux evidence.

`inputs-02.json` raw SHA256 identifies the exact JSON file. The separate full
input identity hashes UTF-8 records sorted by ordinal Path, each formed as:
`Path + TAB + decimal Length + TAB + lowercase Sha256 + LF`.
Both identities are stated separately in the preparation result.