# Image references: final affected validation PASS

2026-11-10. Sole authorized final app/view test and vet completed once; no
source corrections were needed. All twelve frozen identities matched before
and after. Source remains frozen for root acceptance and separate delivery.

## Actual runs

Both commands used native/scripts/go.ps1 from the repository root, with the
optional image-reference capture environment variable explicitly unset. The
command process immediately forwards `exit $LASTEXITCODE`; result exits below
are numeric propagated Go exits, unlike the preserved initial focused wrapper
defect. All existing failures and interpretation files remain unchanged.

| Evidence | Exact Go arguments | Result | UTC interval / wall time |
|---|---|---|---|
|06-final-test|test ./modules/markdownview ./app -count=1|exit0; markdownview PASS3.776s, app PASS21.392s; stderr0bytes|08:06:21.4272878–08:06:47.4597672 /26.032s|
|07-final-vet|vet ./modules/markdownview ./app|exit0; stdout/stderr0bytes|08:07:31.3705176–08:07:33.2254442 /1.855s|

Exact command scripts, stdout/stderr and timestamped result JSON:
[UI evidence directory](markdown-image-reference-ui-evidence-20261110).

Inherit the corrected format/worker owner's final runs14/15, without repeating
those packages: formats/markdown test PASS0.741s, cmd/levnote-markdown-export
test PASS0.800s, combined test exit0; corresponding vet exit0. Evidence:
[format final tests](evidence/markdown-image-reference-format-20261110/14-multiline-envelope-final-affected-tests.stdout.txt),
[format final vet exit](evidence/markdown-image-reference-format-20261110/15-multiline-envelope-final-affected-vet.exit.txt).
This is the superseding multiline rejected-envelope correction described in
[format result](markdown-image-reference-format-result.md).

## Frozen source identities

12files/138,719bytes. Pre/post inventories are byte-identical:
[05 before](markdown-image-reference-ui-evidence-20261110/05-final-source-before.json),
[08 after](markdown-image-reference-ui-evidence-20261110/08-final-source-after.json).
Both SHA256:
D95C29FB86AF3F8E32E5871A77228B7545202B0813EB57F0CAAE1DB41FE54FFD.

| File (native/) | Bytes | SHA256 |
|---|---:|---|
|modules/markdownview/links.go|11353|6F3283C9B0C36861A327713996FC10561EF2F0DF37292AE0C903E8480CC6D293|
|modules/markdownview/blocks.go|3247|1C1BBCD0282C97E5608E51D2380C50ED791BF27F6D727AB46127183E9B3E3821|
|modules/markdownview/view.go|19104|A720F9D028755DD3EBDEFC16B212C9A19BFA4283B69C238C38816CADDE9268DE|
|modules/markdownview/links_test.go|24226|6264D1490C4E225DCD61329D3FC0B910843F1621857B0AD6A4E54C8A6566A54F|
|modules/markdownview/image_reference_test.go|16963|9EF851FA5A414E0F0F26512822CC4E63AD62F128C9CC9E271A1C6C44E57000BC|
|app/markdown_image_reference_app_test.go|7929|A5CBA2E870AEBBDFD3CD4F9F0C41E1D60962278BCC542DE8C6973C944ABBB0AC|
|formats/markdown/types.go|3347|0FAC32ECA5CD9E8FA49565610700F57AFAF848E16EC05CCE6647FAE9423E1029|
|formats/markdown/inline.go|13928|AA793A782D35BAAB4F27A62AB000BD4B77E4718C36C68C8721167EFBFDB7F070|
|formats/markdown/export.go|12453|F9918914B182EC19DCAC47BB7533F6A44614D8FF9E84D530C4E012F4E7EA6DB6|
|formats/markdown/image_test.go|11686|CE11258B0A1E71BB2B7EA15EBDEA26D298558ED63A9FF43BD492578E8B6A90F5|
|formats/markdown/markdown_test.go|12783|B31BA61545754D39FC80961C3803DBEA891792C2D4FBB7C3EFA15936B71684C9|
|cmd/levnote-markdown-export/image_reference_test.go|1700|8DDA439D04322CDCAF742D3BE37CC7D8855DD05C066CD59DC1076209D0D3D9CB|

Implementation behavior, focused diagnostics and the single previously viewed
265x480capture remain in [UI result](markdown-image-reference-ui-result.md).
No extra capture, broad all-packages suite, build, package, OS worker, hidden
host or resource run occurred. This validates source and synthetic native/app
paths; it makes no physical OS input/IME, screen-reader, driver, memory/CPU or
product-package claim. .25 package/evidence remain immutable; .26 execution is
not implicitly activated by this successful check.
