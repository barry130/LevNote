# Image toolbar at minimum application width

ACTIVE 2026-11-10 for the DeepSeek imageview owner, interrupting only its .23
preparation work. App owner remains Astra. Root viewed integrated captures and
found the imageview 查看字节 button clips beyond the 532-DIP content pane at the
760-DIP minimum app width (228-DIP sidebar). The earlier standalone 760-wide view
captures did not exercise this. App-level 字节视图 works but duplicate clipped
controls are not an acceptable final layout.

Own modules/imageview UI/layout and focused geometry/Tester regression only,
plus separately named handoff/evidence files. No app/editor/core/modules.go/vendor
changes. Preserve previous view hashes/results/captures and append correction
history. Use public MyGo layout/input APIs. Keep all image controls usable at
532 DIP and at 300 DIP (minimum pane), with bounded wrapping or a second compact
metadata/action row. No added animation, hidden toolbars or heavy abstraction.
Also ensure long metadata cannot push the byte action beyond the clipped pane.

Use one substantive Tester regression with actual control bounds at the narrow
widths and click byte action callback. Avoid regenerated screenshots in this
scope: app owner will create one separately named replacement image capture after
the fix. Run focused imageview test once after the change and communicate freeze
to root/app owner. Their one final integrated modules/... + app etc suite/vet
then covers this exact source; no redundant standalone editor/full suite.

Report native-image-view-narrow-toolbar-result.md with exact files/identities,
geometry and focused command/output/exit. Then resume .23 preparation only after
root acknowledges the view fix. Product build/host execution remains inactive.
