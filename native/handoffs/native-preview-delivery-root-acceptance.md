# Root acceptance — native readonly preview .7

2026-11-09. Root read the actual source correction, original two export PASS
records, followup preview PASS, command records and combined validator, then
visually inspected all three native tester PNGs. The finite delivery stage is
accepted. Current runnable delivery is `bin/levnote-native-worker-20261109.7/`,
version `native-worker.20261109.7`; previous bundles and evidence stay intact.

## Actual result

- One host plus three fixed workers were built. The unchanged candidate has
  23 files, 23,430,283 bytes, 13 exact source/license assets, three engine
  manifests, 22 release entries and 19 valid local documentation links.
- Host: 13,811,200 bytes, SHA256
  `24BA9D5B11366BDD90B02467889B7272F5A0DD2456D83938BC9A20150CAC706C`.
- Markdown HTML worker: 3,300,352 bytes; Mindmap SVG worker: 3,122,688 bytes;
  readonly Mindmap scene worker: 3,136,512 bytes. Actual PE headers and matching
  manifest/platform/identity/version were checked by the retained validator.
- Original SVG HostPath PASS 0.18s and HTML HostPath PASS 0.14s are reused.
  The original preview reached scene/file/ownership checks and failed only the
  idle-status-label lookup after an expected refusal had changed the message.
  Root reviewed the helper-only correction: it now finds the actual message,
  validates its status-row geometry, and retains all canvas/control checks.
- Fresh preview-only HostPath PASS 0.20s; app vet exit 0. No product rebuild,
  export replay, full-suite repeat or original evidence modification occurred.
  There were two tester compilations, two invocations, six real worker calls:
  four export calls, the original preview call, and one followup preview call.
  Five requests belong to final passing test oracles; the earlier call remains
  recorded as part of the failed test attempt.
- Combined validation PASS 14/14, exit 0, empty stderr. Original 1,081 input
  membership retained, with exactly one authorized test-file change; followup
  source unchanged. Complete 15,657 toolchain files, 984 protected prior files,
  original failed attempt/copy/tester and current candidate are unchanged.

Evidence: `artifacts/acceptance/native-preview-delivery-20261109-082251-package/`
and `artifacts/acceptance/native-preview-delivery-20261109-083326-status-followup/`.
Followup aggregate SHA256
`3DBF1F0221D08D0621F8330A1CA6AEDAB2A6AA203EB7B8C45348B5DB24323EC6`;
validator SHA256
`DD58DA93E60CAC1209F0A554A58C958382172210F281C345918BEA18D8231F2D`.

## Visual review and limits

Root viewed the actual 760x480 light scene, dark scene and source captures.
Canvas and source/fit controls remain above the status row; outline, readonly
limitation, warnings and refusal text are visible. Saved document colors are
preserved, including the low-contrast dark edge from the fixture's explicit
translucent color. This is not a guarantee that arbitrary document colors meet
theme contrast targets. No image or image badge is rendered by this scene;
full attributes/references stay accessible in source. SVG's prior image badge
behavior is separate.

These are native headless tester images and actual worker-path tests, not
physical desktop/OS IME, per-PID observation, app resource measurements or a
Tauri comparison. The editable Mindmap and semantic-outline parser remain in
the host. Fixed bundled workers are not an arbitrary executable plugin sandbox.
The earlier transport stage's missing pre-run fingerprints remain disclosed.

## Next ownership

Astra finishes a concise delivery handoff and updates only source launch,
capability and validation docs under `native-preview-docs-activation.md`.
The accepted package/evidence remain immutable. Schema3 implementation stays
queued until the short documentation gate is read, then core/modules and app
will receive separate ownership as already planned. Root performs no product
edits or test/build execution.
