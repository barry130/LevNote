# Local dependency patches

## `mygo-v0.8.3-no-startup-cleanup.patch`

LevNote vendors `github.com/egoist/mygo v0.8.3`. The upstream
`Application.Run()` starts `cleanUpdateLeftovers()` on every launch, even when
the application has no updater. That helper expands
`<executable-directory>/.*.old-*` and calls `os.RemoveAll` for every match,
including unrelated user-created files and directories. MyGo v0.8.3 has no
public Application or environment-variable opt-out for this behavior.

The local patch removes that one automatic call from the vendored
`github.com/egoist/mygo/app.go` and leaves the upstream cleanup helper unchanged.
LevNote does not enable or call the MyGo updater.
This is a narrow safety patch to the licensed upstream toolkit, not an
independently authored replacement.

Upstream artifacts retained under `vendor/github.com/egoist/mygo/` include its
`LICENSE` and `README.md`. The same notice requirement applies to every
vendored module under `vendor/`.

## `mygo-v0.8.3-bitmap-dispose.patch`

This local MIT-licensed modification adds main-thread `Bitmap.Dispose() bool`.
Owned bitmaps become terminal transparent tombstones; base/mip CPU pixels,
compressed decoder and source closures are released. Nil/repeated calls succeed;
Canvas bitmap/region wrappers are borrowed and return false without mutation.
Same-size Update preserves image identity; replacing dimensions retires the old
base and mip images. Bitmap values must not be copied.

Each renderer independently observes a retirement epoch. Weak image cache owners
do not retain CPU pixels or decoder closures. The next drawable render purges
retired/dead images and derived canvas, clip-mask, blur and backdrop caches,
before any unchanged-frame return; font/gradient atlases and unrelated live
images stay. CPU nested canvases rebuild using the epoch. Metal first discards
pending main batches while committing/tracking already encoded work. D3D11
unbinds SRVs/targets before release; GL deletes only in the owning context.

Dispose itself schedules no redraw: use the ordinary state update that removes
the view. Idle/minimized renderers retain ownership until rendering or Release;
in-flight commands can retain native resources after cache release. External
pixel aliases and independent captures are not erased. This is not a guarantee
of immediate process-memory/VRAM reduction or an upstream-accepted change.

The patch is derived from pristine `github.com/egoist/mygo@v0.8.3` module sources,
including new tests of actual UI/scene/raster/GPU helpers. It contains no shader
binary change. `mygo-v0.8.3-patch-files.json` records exact SHA256 identities of
both patch files and every affected vendor file, including tests and app.go.
The wrapper pins the manifest itself; partial application and missing/changed
backend source are rejected. It is an accidental-bypass guard, not a signature.

### Refresh

From any directory:

```powershell
powershell -NoProfile -File F:\AIproject\LevNote\native\dependency-patches\refresh.ps1
```

The refresh script:

1. runs standard `go mod vendor` through the existing
   `native/scripts/go.ps1` wrapper;
2. applies `mygo-v0.8.3-no-startup-cleanup.patch`, then
   `mygo-v0.8.3-bitmap-dispose.patch`;
3. runs `go test ./dependency-patches`, checking the pin, startup regression,
   both patch files and all modified vendor file identities.

Use `native/scripts/go.ps1` for project Go commands. The wrapper checks the
actual vendor patch before build/run/install/test, rejects case-sensitive
`-C`/`--C`, `-modfile`, `-overlay` and non-vendor `-mod` forms in arguments and
quoted GOFLAGS, while allowing valid lowercase `test -c` and ordinary flags.
It sets GOWORK=off and GOENV=off only during its Go invocation, then restores
both environment values. This prevents an enclosing workspace or persisted Go
configuration from changing the inspected build context; it is not a security
boundary against hostile source, compiler flags or OS actors. Non-guarded
`mod` operations remain usable so refresh can start with an absent patch.

PowerShell7+ focused checks use new evidence directories and tiny isolated
fixture modules with the pinned toolchain; production vendor is never edited:

```powershell
.\native\scripts\check-build-guard.ps1 -EvidenceDirectory .\native\artifacts\guard-check-local
.\native\scripts\check-portable-package.ps1 -EvidenceDirectory .\native\artifacts\portable-check-local
```

The first emits a real fixture tester through `test -c` and a tiny normal build,
checks useful quoted GOFLAGS, and rejects alternate build sources/unsafe or
missing patch inputs. The second checks destination boundaries and exact
license/attribution/manifest assets without compiling or launching the product.
Do not pass `-mod=mod` for project builds or edit the Go module cache.

Normal `./...` skips vendor tests. Run focused tests explicitly through the wrapper:

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','-count=1','./vendor/github.com/egoist/mygo/ui','./vendor/github.com/egoist/mygo/internal/scene','./vendor/github.com/egoist/mygo/internal/raster','./vendor/github.com/egoist/mygo/internal/gpu','./vendor/github.com/egoist/mygo/internal/gpu/d3d11','./dependency-patches')
```

Linux GL and Darwin Metal test files use fake release calls and compile on their
targets; they require their target OS to execute. Pure/fake tests verify ownership,
transparent stale scenes and call ordering, not actual driver lifetime or VRAM.
Portable attribution includes both patches; already accepted packages are not
rewritten. Refresh mutates vendor, so run it only when no other owner uses it.

### Upgrade

When changing MyGo, rebase both patches against the new pristine source, review
all renderer checkpoints, regenerate exact identities, and retain the startup
and disposal regressions. Do not carry these patches blindly across versions.
