# Autosave and image keyboard final validation

2026-11-10. The authorized corrected combined test and vet passed. Both source
sets are frozen. The original combined command's imageview failure and withheld
vet are preserved below; root assigned the narrow explicit100% correction to
the view owner, reviewed its stronger public regression, then activated exactly
one corrected combined run. This runner made no product source change.

## Frozen inputs

All11 autosave entries and all4 keyboard entries matched before and after the
run. Inventories under native-autosave-evidence-20261110:

- 03-source-before.json:15/15 exact byte counts and SHA256 matches.
- 06-source-after.json:15/15 exact byte counts and SHA256 matches.
- source-freeze.json:11 autosave identities; keyboard expectations came from
  native-image-keyboard-result.md.

The preserved capture-identities.ps1 performed both comparisons without source
mutation. Both invocations exited0.

## Actual command and outcome

`native/scripts/go.ps1 -GoArguments @('test','./app','./modules/imageview','-count=1')`

Started2026-11-10T06:11:29.3195230Z; completed06:11:52.8310935Z;
wall23.512s, exit1. Full exact command/output/metadata are04-final-test.command.ps1,
04-final-test.output.txt and04-final-test.result.json in the evidence directory.

```text
ok   levnote.local/native/app 21.808s
--- FAIL: TestOneRequestAndPublicToolbarInput (0.01s)
    imageview_test.go:360: 100% action = imageview.viewport{zoom:1, panX:180, panY:101, fitMode:true}
FAIL levnote.local/native/modules/imageview 16.323s
```

App's full package passed. The imageview failure is after its public Tester
click on the100% toolbar control. Its fitMode remained true. No cause is claimed
until the view owner/root inspects this interaction with the keyboard/focus work.
The first run stopped before `vet ./app ./modules/imageview`, following
the task's stop-on-blocker instruction. Earlier keyboard failures remain in that
owner's separate command manifest and were not overwritten.

## Corrected final validation

Root traced the failure to zoomToCenter's equal-geometry return preserving fit
mode for a small image already fitted at100%. The view owner corrected that
explicit mode transition and added public button/key1 resize coverage; clamped
relative zoom and pan remain unchanged. The strengthened400x200 image test
resizes760x480 to300x240, distinguishes cropped manual100% from fit, then uses0.

07-corrected-source-before.json and10-corrected-source-after.json both match
all15current source entries. Relative to the first run, only these two changed:

- native/modules/imageview/ui.go6456B SHA256
  BEF3A43EA6A028791A623728E9899CF481A1BB46BC12D91D61AFD67FB95AA368.
- native/modules/imageview/imageview_test.go42807B SHA256
  824645F0115D8E4DF7851696E78B33AAC941523C9C2D5929B0EB3B1019094231.

`native/scripts/go.ps1 -GoArguments @('test','./app','./modules/imageview','-count=1')`
ran once after correction,2026-11-10T06:22:59.0240489Z to06:23:26.7965535Z:
exit0, wall27.773s, app PASS23.863s, imageview PASS15.447s. Preserved as
08-corrected-final-test.command.ps1/.output.txt/.result.json.

`native/scripts/go.ps1 -GoArguments @('vet','./app','./modules/imageview')`
ran once,2026-11-10T06:23:58.5354158Z to06:24:00.0679195Z:
exit0, wall1.533s, empty output. Preserved as
09-final-vet.command.ps1/.output.txt/.result.json.

No product execution, build, capture, benchmark or resource run. Public Tester,
fake-time and queued-UI checks do not establish physical Windows IME or
driver/resource behavior. No new release artifact was created.
