# Stage2 plan preflight (read-only)

ACTIVE 2026-11-10 for GLM plan owner. DeepSeek owns stage1 implementation,
another DeepSeek owns .18 docs. Do not edit/test/build product files, don't
spawn children. Read `drawio-worker-stage2-task.md` and actual app/graphview
lifecycles, especially retained views, lazy creation, preparation callbacks,
reparse, page change, native-dialog reentrancy and current tests.

Write at most two pages `drawio-worker-stage2-preflight.md` identifying definite
missing guards, relevant existing test fixtures and a minimal implementation
map. Do not repeat the full plan. Concrete facts prevail over assumptions.
Root already read: app/engine_lane.go owns runner call/task.Source clearing;
preview_worker.go creates lane and preview inputs; graphview.Flush with no draft
can leave stale cache; releasePresentation retains View; view toolbar precedes
lazy Create. Design to these facts. No global framework/new background worker
or allocation claims. Stage1 is active; don't inspect half-edited lower-layer
files to declare defects until its final frozen result.

Outcome should make later stage2 execution straightforward and avoid extra
serial review loops. Publish then stop pending assignment. You may be selected
as independent reviewer for stage1 after its completed source handoff.
