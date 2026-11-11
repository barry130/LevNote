# Schema3 application slice — root acceptance

2026-11-09. Root read final source changes, focused workflow oracles and raw
attempt01/capture03/final04 logs. Accepted final04 full app PASS9.819s, actual
child exit0, scoped app/plugins vet exit0 with empty streams,54 unchanged inputs
and aggregate10/10. Root also inspected all three capture03 PNGs using view_image:
760x480 light/dark installed rows with fully visible create actions and actual
.taskmd native Markdown presentation. These are native headless tester images,
not physical Windows UI or system IME validation.

The new route keeps installed identity separate from the shipped adapter and
does not override registration ordering. New-document validation after Flush
checks presentation ownership and actual current suffix matching. SaveAs now
rechecks ownership and Flush after the modal, before any file publication or
view disposal. Actual note composition reaches disk in one undoable edit;
pending tag, replaced/closed/recovered/reconfigured owners refuse publication.
SetEnabled distinguishes post-persistence reload failure from failed settings
writes. Tests show removed/corrupt manifests reconcile the display and real
settings failure preserves it. Same-ID refresh clears old scene/export owners.

Core followup sentinel test PASS0.140s is read from attempt01 actual stdout;
that overall attempt still FAILED its capture assertions. Its result.json had
incorrect exit0 because its wrapper failed to propagate LASTEXITCODE; separate
evidence-notes documents this. Capture02 also FAILED (dark click clipped).
Capture03 fixed only the test wheel point/real clip boundary and PASS1.622s.
No prior failed evidence is relabeled or removed. Final/capture scripts forward
the true child exit code. Earlier accepted schema3 core/modules gate is reused.

Accepted aggregate:
artifacts/acceptance/format-routing-app-20261109-105444-final04/aggregate.json
SHA25677DF322DE9DCE9C762173E642931671934E942839E3AB1FBA4CDAD7CF87D14E4.
This accepts source/application integration, not a new executable. .7 remains
the runnable accepted baseline until .8 is built and checked. No resource
improvement or independently installable executable-engine claim is made.
