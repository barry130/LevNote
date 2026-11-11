# Local binary/image prerequisite

ACTIVE 2026-11-10: read-only contract review while .22 packaging is active.
Owner: fresh DeepSeek read_native_handoffs_binary_image_prerequisite_task_md.
Root plans/reviews; no child delegation. Context max256K; compact before200K.
Read native/AGENTS.md and native-image-view-next-plan.md. Do not revive any
historical tasks. Task payload fallback is this exact file.

Work now: inspect document atomic publication, DiskToken and stable reads;
session capture/output/validation; app binary inspector lifecycle; MyGo public
Bitmap and scene/renderer ownership. Publish a concise implementable contract
in binary-image-prerequisite-design.md with exact APIs and file ownership.
No code/product docs/tests/builds/launches yet: .22 scoped source is frozen.

Recommend a narrow first implementation in core/document only: explicit binary
baseline, bounded stable snapshot with immutable byte access, exact nonoverwriting
SaveAs, changed/missing-source behavior and baseline identity/revision. No fake
text Document authority. Reuse existing disk/atomic boundaries. Distinguish
publication committed then verify/durability failure from no write. Avoid a
generic document rewrite. Read current helpers/tests before choosing the API.
Future app integration must retain no inactive compressed bytes and leave text
session JSON byte behavior unchanged. Session/app edits are later ownership.

Also resolve the concrete GPU-lifecycle decision at design level. Inspect actual
Windows UI/render threading and other backends; propose the smallest public
retirement or bounded cache contract that releases inactive bitmap textures
after one normal redraw and then stays idle. Include CPU/source/mip ownership,
same-size/different-size Update, stale scene references, close and device loss.
Do not implement vendor patch/import internal packages/use finalizer-only release,
forced frame loops or process memory claims. If a public solution already exists,
show exact code. Otherwise list exact proposed changes/tests and whether Windows
scope can be honest without silently breaking other backends. Prefer deterministic
bounded texture ownership over merely weakening the low-memory goal.

Keep this review bounded: one concrete recommendation, no alternative framework
or sprawling benchmark harness. Root will activate core implementation after .22
acceptance. Write progress/checkpoint if service delivery fails. No network needed.
