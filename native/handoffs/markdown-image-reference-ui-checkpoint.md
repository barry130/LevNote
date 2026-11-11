# Markdown image-reference UI checkpoint

2026-11-10. Assigned implementation in progress; .25 package/evidence untouched.

The fixed format contract is `InlineImage="image"` with original Range, raw
label Text and raw Href. The view now uses the existing native link path for
both kinds, sharing16controls/64spans per row. Images display original syntax,
have a distinct bounded picture-reference accessible name/tooltip, and keep
raw syntax in dense fallback. No I/O, decoding or new host route was added.

Captured activation now includes inline kind and view-release generation with
the existing buffer/version/row/enclosing+syntax ranges and cloned target.
It rechecks kind after Flush/map/reparse, drops canceled/scrolled/retired
gestures, and suppresses image key-repeat/rapid pointer continuation. Existing
plain selection, doubleclick edit, readonly reading and ordinary link route
are retained. A transient pressed-row snapshot settles after release.

Owned files: modules/markdownview/{links.go,blocks.go,view.go,links_test.go,
image_reference_test.go}, app/markdown_image_reference_app_test.go. Existing
links_test.go change only supplies the added captured kind/release fields.
rows.go needs no edit because its capability query delegates to shared count.

First focused run will exercise new image behaviors plus affected ordinary-link
routes and actual registered Markdown-to-PNG/JPEG app reading, duplicate select,
unchanged source/BOM/CRLF/history, refusal and disabled-plugin byte fallback.
One optional265x480 synthetic image will assess raw-reference wrapping. Exact
commands/streams/exits, including any failure, go into
markdown-image-reference-ui-evidence-20261110. No broad suites/vet, package,
host, worker or resource run is active; final combined checks await root review.

Completed: corrected focused view1.445s/app2.151s exit0, original failure and
outer-exit defect preserved; PNG diagnostic established rapid continuation,
then explicit fresh-gesture fixture passed. Six-file identities and scope are
in markdown-image-reference-ui-result.md. Own source frozen; no further checks
until root final combined activation after format corrections.
