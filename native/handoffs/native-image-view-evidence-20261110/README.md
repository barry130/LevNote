# Native image view verification evidence

2026-11-10. Scope is `modules/imageview` and the additive `ui/editor` image
contract only. No product package, host, worker, resource, physical UI or
driver run was performed.

## Recorded runs

| Step | Command | Exit | Result |
|---|---|---:|---|
| 01 | focused `modules/imageview` test | 1 | Test source/reference compile errors; see stderr |
| 02 | focused `modules/imageview` test | 1 | Two accessibility assertions failed; see stdout |
| 03 | focused `modules/imageview` test | 0 | PASS after making loading/error text a UI control |
| 04 | focused `modules/imageview` test | 0 | PASS after drag-origin and finite-overflow corrections |
| 05 | final affected `modules/imageview` plus `ui/editor` test | 0 | Both packages PASS |
| 06 | vet for `modules/imageview` and `ui/editor` | 0 | Empty stdout and stderr |
| 07 | focused review regressions | 0 | Tall sub-floor fit, center-preserving resize, float32 position guards and labels PASS |
| 08 | corrected synthetic captures | 0 | Added a separately named light/dark pair; originals retained |
| 09 | final affected `modules/imageview` plus `ui/editor` test after review fixes | 0 | Both packages PASS |
| 10 | final vet for `modules/imageview` and `ui/editor` | 0 | Empty stdout and stderr |

Each `NN-*-command.txt`, `NN-*-stdout.txt`, `NN-*-stderr.txt` and
`NN-*-exit.txt` holds the saved invocation, streams and exit code. The empty
`06-vet-stdout.txt` is the actual successful output. Runs 03 and 04 set
`LEVNOTE_IMAGEVIEW_CAPTURE_DIR` to this directory so the optional Tester capture
test regenerated both original images from the source at that time. Run 08
also set `LEVNOTE_IMAGEVIEW_CAPTURE_SUFFIX=review-fix`, preserving those
originals and adding two corrected captures from the reviewed source.

## Synthetic captures

Both are 760x480 public MyGo Tester captures of a decoded 600x360 transparent
palette PNG. They show the toolbar, 100% current scale, metadata and neutral
alpha checkerboard in light and dark themes.

- `imageview-alpha-toolbar-light-760x480.png`
  SHA256 `f42a9161b09c7c709698fc680a7d5dece825ff4999bab23352ec941477844ad9`
- `imageview-alpha-toolbar-dark-760x480.png`
  SHA256 `43718399e2becd68c0bbf1e296cb8c3eea60a27c98671e469aa88a34a6e3e8bf`

These are synthetic software-renderer captures. They make no physical OS,
font, driver, GPU, IME or resource-use claim.

The review corrected the binary inspection label from `查看只读源码` to
`查看字节`. The separately named final captures from run 08 are:

- `imageview-alpha-toolbar-light-760x480-review-fix.png`
  SHA256 `c00aa0141252f585691226d0ac4092ce67c2425b5175fbd0536ba78e633ba92f`
- `imageview-alpha-toolbar-dark-760x480-review-fix.png`
  SHA256 `37ea85ba9bdd51a20cc43b23d11e5813ae56133a476cf71aadb92d1079d6298e`

The original captures and their identities above remain in place.
