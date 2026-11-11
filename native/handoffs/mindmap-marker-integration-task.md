# Marker release — integration takeover

Active sole owner: `/root/astra_marker_view`. Root interrupted stalled
`glm_marker_app` and `glm_formats_finish`; no concurrent ownership. User allows
Astra. Root plans/reviews only. No subdelegation. Product/tests are frozen.

Root accepted focused marker gate after reviewing format API, final graphview
source/test4.477s/vet0, independent SVG resolution, app source/oracles, actual
focused PASS2.817s stdout and all4final760x480app images in:
`artifacts/acceptance/mindmap-markers-20261108-234731-final-run/`.
Independent reviewer document still says split pending, but root independently
verified actual split native clear action, resulting icons/SVG, BOM reopen,
RoleButton recovery bounds and final images. No further focused test required.

FIRST run one full test/vet/release build/isolated hidden startup+WM_CLOSE
sequence using existing previous successful cell harness as reference. Reuse
code by copying into a NEW unique artifact directory, adapt stage/fixture names
and source hash list. Do not overwrite any prior log/result/image or source exe
before preserving it. The current accepted executable is:

- `bin/levnote-native.exe`,13,425,152bytes
- SHA256 `34F3F48C98300B053B0BA0E69BE9D9F89CDA15E17759E6A26AD817A1E9C1D14C`
- mtimeUTC2026-11-08T13:45:35.0697169Z

Prior successful harness/evidence:
`artifacts/acceptance/markdown-cells-20261108-213713/` and
`artifacts/acceptance/startup-smoke-markdown-cells-20261108-213713/`.
Read scripts before reuse. Do not fabricate metadata or reconstruct raw output.
Capture actual streams (including empty), exit codes, times, executable digest
and mtime. Full native Go commands use existing wrapper, Go1.27.3:

```
test ./... -count=1 -timeout 300s
vet ./...
build -trimpath -buildvcs=false -ldflags '-s -w -H=windowsgui' -o bin/levnote-native.exe ./cmd/levnote
```

Optional screenshot env vars must not accidentally overwrite final evidence
during full tests. Isolated smoke uses private config/workspace, hidden startup,
find exact owned PID/window then WM_CLOSE, graceful exit0, fixture+sentinel hash
preservation and no remaining owned child. Start-Process hidden. No broad kill,
delete/move outside verified workspace. No resource sampling or benchmarks.
If any command fails preserve evidence, report exact failure, request root
scope adjustment for product changes; don't retry same artifact directory.

Produce aggregate `result.json`, evidence validation and concise
`handoffs/integration.md` latest section. Root reads actual files and accepts
exact binary before release status changes. Do not claim root already accepted
this future artifact; separate acceptance record follows explicit root decision.

THEN own final docs previously assigned to GLMapp: native README/VALIDATION/API/
ARCHITECTURE as needed and root ROADMAP/CHANGELOG/README native entry. Mention
19glyph palette,64/4096 UI cap,512byte summary, known glyph exclusivity,
unknown order preservation, clear[], candidate/draft safety, SVG bounded labels.
Keep actual limitations: no full parity, node images placeholder, tags not
editable, narrow split source pane squeezed, screenshots not realOSIME/GPU,
no current resource comparison. No unsupported claims from older artifacts.

Correct stale `handoffs/mindmap-markers-app.md` without rewriting history:
- current final focused run234731PASS2.817s, original run234000FAIL2.623s
  preserved; only test bugs (priority4 replaces3, and Find returns text glyph
  instead of28DIP recovery button; now AccessByRole).
- older230124attempt images were overwritten by its author's rerun; copying
  final bytes to231014does NOT preserve old failures. Do not say it did.
- final4images234731are reviewed, clear/recovery full controls visible;
  allcontrols not simultaneously visible, graph labels can be clipped.
- no extra focused vet is required; use upcoming full vet result.

Worker stage remains QUEUED in `handoffs/markdown-worker-contract.md`; after
marker artifact acceptance root will separately activate ownership. Do not
implement it during this integration.
