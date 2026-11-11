# Local dependency patches

## `mygo-v0.8.3-no-startup-cleanup.patch`

LevNote vendors `github.com/egoist/mygo v0.8.3`. The upstream
`Application.Run()` starts `cleanUpdateLeftovers()` on every launch, even when
the application has no updater. That helper expands
`<executable-directory>/.*.old-*` and calls `os.RemoveAll` for every match,
including unrelated user-created files and directories. MyGo v0.8.3 has no
public Application or environment-variable opt-out for this behavior.

The local patch removes that one automatic call from the vendored
`github.com/egoist/mygo/app.go` and leaves the upstream cleanup helper and all
other MyGo code unchanged. LevNote does not enable or call the MyGo updater.
This is a narrow safety patch to the licensed upstream toolkit, not an
independently authored replacement.

Upstream artifacts retained under `vendor/github.com/egoist/mygo/` include its
`LICENSE` and `README.md`. The same notice requirement applies to every
vendored module under `vendor/`.

### Refresh

From any directory:

```powershell
powershell -NoProfile -File F:\AIproject\LevNote\native\dependency-patches\refresh.ps1
```

The refresh script:

1. runs standard `go mod vendor` through the existing
   `native/scripts/go.ps1` wrapper;
2. applies `mygo-v0.8.3-no-startup-cleanup.patch`;
3. runs `go test ./dependency-patches`, which fails with an explicit missing
   patch diagnostic if the unsafe call or local marker is absent.

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

### Upgrade

When changing the MyGo version, replace this patch with one based on the new
upstream source, retaining a source-level marker and the startup regression.
Do not carry this patch blindly across a version change.
