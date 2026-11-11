# Image plugin registry and neutral composition seam

ACTIVE2026-11-10 for GLM after session source freeze. Session source remains
frozen for root review; this task has no session edits or dependency on acceptance.
No children. Own core/plugins/** and modules/modules.go plus focused module
composition tests only. No app/imageview/editor/document/session/raster/vendor
edits, no product build/host/worker/resource run. Imageview is DeepSeek ownership;
app integration is Astra ownership. Read current active contracts.

Register builtin image alongside current formats, ID image, extensions
.png/.jpg/.jpeg, Read=true, Edit=false, Source=true (readonly byte inspection),
Export=false. Enabled/disable/persistence/list behaviors reuse existing manager.
Plaintext/byte fallback remains independent. No schema3 image route or arbitrary
installed codec/executable; no image template/new action.

Add imageview.Adapter() to modules.Builtins after the view compiles. Add neutral
composition function DecodeImage(ctx context.Context, r io.ReaderAt, size int64)
(editor.ImageResult,error), mapping raster.Decode fields and truthful PNG/JPEG
labels/size, returning same owned Pixels without copying. Host imports modules,
not a concrete format parser. No provider goroutines, retained reader/source,
filesystem or duplicate validation/decoding in composition.

Root settled the signature above for app integration: the app passes
snapshot.ReaderAt(), snapshot.Size() while retaining its own immutable snapshot
authority. Do not replace this seam with a document.BinarySnapshot parameter.
The builtin registry and adapter both use ID `image`. API availability is the
first implementation checkpoint, before the remaining focused verification.

Test registry enable/disable/reload and exact image capabilities/extension match;
test shipped adapter has no template/export/outline; use real tiny PNG/JPEG data
through neutral seam plus rejection/cancellation transfer. Avoid tests mirroring
literal fields only or changing existing schemas. Preserve order/fallback tests.
Coordinate compile timing with imageview owner through root. One focused suite
then final affected modules/plugins vet after real fixes, exact saved evidence at
native/handoffs/native-image-composition-evidence-20261110. Publish frozen
native-image-composition-result.md and communicate API to root promptly.
