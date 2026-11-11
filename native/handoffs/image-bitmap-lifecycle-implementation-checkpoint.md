# Bitmap lifecycle implementation checkpoint

2026-11-10 Astra, implementation saved before checks. No product build, UI,
driver or resource run; active vendor was edited in place, never regenerated.

Public API: UI-thread `(*ui.Bitmap).Dispose() bool`, nil/repeat true, borrowed
Canvas false. Terminal base/mip ownership, source/remake callbacks and zero size;
same-size Update remains stable, changed-size Update retires old generations.
Internal `scene.Image.Retire/Retired`, global monotonic retirement epoch and CPU
canvas PixEpoch. `gpu.ImageRetirement.Check`, weak-owner SweepRetired and
Builder.ForgetImages are actual production helpers with focused tests.

Vendor files relative to github.com/egoist/mygo:
- ui/{bitmap.go,paint.go,material.go,canvas.go,bitmap_retirement_test.go}
- internal/scene/{scene.go,retirement.go,image_retirement_test.go}
- internal/raster/{damage.go,canvas.go,raster.go,blur.go,retirement_test.go}
- internal/gpu/{gpu.go,retirement.go,retirement_test.go}
- internal/gpu/d3d11/{d3d11.go,retirement_test.go}
- internal/gpu/metal/{metal.go,present.go,retirement_test.go}
- internal/gpu/gl/{gl.go,retirement_test.go}

gpu.go additionally rejects stale shader/blur inputs before callbacks; scene
canvas traversal includes image clip inputs so retained clip canvases rebuild.
D3D cleanup precedes Diff and unbinds slots0–9/RTV; Metal commits/tracks only
encoded work, clears unencoded pending batches, then purges before waking;
GL context mismatch never deletes. Derived clip masks are included because they
can retain copied bitmap alpha. Font/gradient atlases and live base images stay.

Propagation staged: dependency-patches/{refresh.ps1,README.md,verify_test.go,
mygo-v0.8.3-bitmap-dispose.patch,mygo-v0.8.3-patch-files.json}; scripts/{go.ps1,
check-build-guard.ps1,portable-package-lib.ps1,check-portable-package.ps1};
THIRD_PARTY_NOTICES.md. Manifest/guard enforcement is enabled only after all
final source/patch files exist. Existing startup guard remains runnable meanwhile.

Next: format, focused Windows vendor tests/vet; compile affected packages/tests
on Windows/Darwin/Linux/js as available; generate pristine-source patch, validate
application once in isolated scratch, then final guard/provenance/asset fixtures.
All Go through scripts/go.ps1. Evidence root:
artifacts/acceptance/image-bitmap-lifecycle-20261110. Preserve failures and exact
commands/exits; final handoff states runtime limits and frozen file identities.
