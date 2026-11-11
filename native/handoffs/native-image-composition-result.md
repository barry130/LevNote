# Native image composition result

2026-11-10. Ownership was `core/plugins/**`, `modules/modules.go`, and focused
composition tests. No app, session, imageview, editor, raster, document, view,
vendor, package, host, or worker files were changed.

## API checkpoint

The neutral host seam is:

```go
func modules.DecodeImage(ctx context.Context, r io.ReaderAt, size int64)
    (editor.ImageResult, error)
```

It invokes `raster.Decode` once with the caller-owned `ReaderAt` and declared
size. It opens no file, starts no provider goroutine, retains no reader/source
bytes, and performs no duplicate validation or decoding. On success it maps
`raster.KindPNG` to `"PNG"` and `raster.KindJPEG` to `"JPEG"`, preserves final
and source dimensions, EXIF orientation, and the admitted size, and transfers
the same owned tightly packed `*image.RGBA` pixels without a full-size copy.
Admission, decode, rejection, and cancellation errors transfer unchanged.

The host should call `snapshot.ReaderAt(), snapshot.Size()` while retaining its
own immutable snapshot authority. The seam intentionally does not accept a
document snapshot type.

## Registry and adapter composition

The builtin manifest is ID `image`, name `Native Image Reader`, version
`1.0.0`, order 4, and extensions `.png`, `.jpg`, `.jpeg`. Its exact capabilities
are `Read=true`, `Edit=false`, `Source=true`, `Export=false`. It remains
independent of the plaintext/code fallback and has no schema3 native route,
installed codec, arbitrary executable, template, new action, outline, scene, or
export capability.

`modules.Builtins()` now appends `imageview.Adapter()`. The adapter and registry
both use ID `image`. Existing plugin manager enable/disable, persistence,
reload, list, and fallback behavior is reused.

## Verification

Focused checks passed:

- `native/scripts/go.ps1 -GoArguments test,./core/plugins,./modules`
  (actual wrapper form:
  `& .\scripts\go.ps1 -GoArguments @('test','./core/plugins','./modules')`)
  — both packages `ok`, exit 0.
- `native/scripts/go.ps1 -GoArguments vet,./core/plugins,./modules`
  — no output, exit 0.

Evidence is in `native/handoffs/native-image-composition-evidence-20261110`.
Tests cover exact image capabilities/extensions, manager disable/re-enable and
reload persistence, fallback ordering, shipped adapter read-only/no template/
no outline/no export, real tiny PNG and JPEG bytes through `DecodeImage`,
rejection and cancellation transfer, declared-size use, and separate per-call
pixel ownership.

## Scoped identities

SHA256:

- `native/core/plugins/registry.go`
  `091a1a66dd41601ff6f28399a3fca3b5f05f0e445140ddebaad68c04d95280fd`
- `native/core/plugins/plugins_test.go`
  `a03015e17672f1dc0bac6359f9383f3b49ba0f9347c828a7b980dd34cb5ab739`
- `native/modules/modules.go`
  `a40a02288dbe60f387f8d658c319942b5d1339f6044aa54e34d7401e19a7d62e`
- `native/modules/image_composition_test.go`
  `19432dd2289b38176559c008cfdbb58d4b228a6a7f2a4a32d8af2814274c1508`
