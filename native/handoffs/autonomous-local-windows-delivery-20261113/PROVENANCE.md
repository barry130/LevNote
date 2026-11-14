# Helper provenance and preparation freeze

Task-16, owner windows-delivery, root lead. All new files are authored under
this sole fresh handoff directory; no existing source/helper/evidence was edited.
Reading and literal drafting/review only: no parser/compiler/command/Go/launch,
network/MCP/Git or delegation. A SHA below is from the native file reader's
content revision identity, not an executed Get-FileHash/parse/test command.
Root independently verifies these bytes before later checks and activation.

## Final runtime helper identities (frozen preparation)

| File | Bytes | Lines | SHA256 / native reader revision |
| --- | ---: | ---: | --- |
| [runner](<run-host-smoke.ps1>) | 22673 | 315 | 01D897C5F854426A6874B70E0828DA8065DE77BC6BDC563B5270EF9C2F51FCC6 |
| [library](<delivery-lib.ps1>) | 20338 | 317 | D59E0FDC4E2257E04A3F470E502342EA1B98DEA83A9A4C9AC73D97B2919DD9CF |
| [owned launcher](<owned-host.cs>) | 20445 | 325 | 68806C747B79445A8EB3DB4781C35E07AEC69AD50A56A867E2DBC3AF634D50FA |

These are drafts with complete bounded native readback plus targeted changed
ranges reread, not demonstrated PowerShell parse, C# compilation or runtime
success. No further child runtime edits after this freeze without a new task.
The parent parser/compiler review must use these exact identities in a fresh
receipt; any later edit needs new pins/review, never rewriting old outcomes.

## Non-executable contract/document identities

| File | Bytes | Lines | SHA256 / native reader revision |
| --- | ---: | ---: | --- |
| [README](<README.md>) | 3641 | 61 | F9045A2D09D09567DD01F68096ABF542D2738995F446BC4F8C438353EBAB5C5C |
| [complete input contract](<INPUTS.md>) | 8611 | 127 | 43F0FDC44148863DE5EC07202919D84DAC77FB567EB6E285B152FFFEB9E0BCE9 |
| [input shape](<input-contract.template.json>) | 1809 | 32 | 115408FE334E3D7E0394B2BB9FAAA006939D65668A9FB15344CB7C05E0CA82B4 |
| [shipping-source shape](<source-inventory.template.json>) | 659 | 32 | 1C51F3D25A5B17A760AFB6739228E50195301993D742864EF6780D399541ED18 |

No actual future source/package/runtime/fixture freeze was captured: current
product work is concurrent and unaccepted by this preparation. Root creates
separate real inputs, rather than adopting historical/template pins as authority.
This provenance file intentionally does not self-hash.

## Immutable historical design references (not executable dependencies)

- [Closed .30 host helper](<F:/AIproject/LevNote/native/artifacts/acceptance/open-document-picker-release30-20261110/run-host-smoke.ps1>):
  33197B/717lines;3EF211726C83C460589773FEB6C44797D2AEAEEA66CAF594BD258BADB97B5DAF.
  Retained design: complete copied bundle, ordinary default C: config, exact local
  plugin, ordered text/PNG,5000ms alive settling, preclose strict session capture,
  newer close publication, exit0/Job accounting/protected input inventories.
- [Closed .30 library](<F:/AIproject/LevNote/native/artifacts/acceptance/open-document-picker-release30-20261110/delivery-lib.ps1>):
  30481B/680lines;1BB850C3E43527F4E3426B21C4EA2132E9707676EE1F9C8B78E4EEAE45DE82C4.
  Exact inventory/PE/session principles were reauthored, not dot-sourced. Old
  context/activation/version/run-root decisions do not activate this candidate.
- [Closed .30 preparation result](<F:/AIproject/LevNote/native/handoffs/open-document-picker-release30-preparation-result.md>):
  read only for historical boundaries. No consumed namespace reopened/mutated.

Deliberate corrections: old Start-Process-before-Job race is replaced with
CreateProcessW CREATE_SUSPENDED + explicit inheritable stdio handle list,
non-inheritable kill-on-close Job assignment/identity checks before ResumeThread.
Old preclose-failure/cleanup normal-close paths are not copied: zero WM_CLOSE
on failed preclose gate, and one total owned close attempt only after success.
No title/name-based killing, repaint or keyboard/mouse/IME simulation.

## Current source/data references (read only; no execution)

- [Packaging result definition](<F:/AIproject/LevNote/native/scripts/package-markdown-worker.ps1#L90-L160>):
  9323B/161lines;E888150C909962EE8F65F4EA536F5BAA9AF1B1D69AA91E484F53BF72FD98A0F3.
  Fixed host/four worker paths, engine IDs/formats, GUI PE requirements and
  successful package-result shape. Runner validates bytes; never invokes builds.
- [Portable manifest definition](<F:/AIproject/LevNote/native/scripts/portable-package-lib.ps1>):
  7295B/113lines;809FB09683D290628A48E2AA7CBF3B94F8F09E35AB678D0C2190B4D5E3B51D89.
  Complete file/self-exclusion hashes and `%APPDATA%/LevNoteNative` config.
- [Host entry](<F:/AIproject/LevNote/native/cmd/levnote/main.go#L13-L35>):
  882B/40lines;D558420F897CF94095D3B66F2454CDF4C51DC1316B567C781F380B0DCFDA15E6.
  Real title, os.UserConfigDir/LevNoteNative, Restore then ordered OpenMany.
- [Session wire](<F:/AIproject/LevNote/native/core/session/session.go#L19-L67>):
  7027B/237lines;BD3593BCBA8E22FC96B1A27B8BD772C9E183D413723D9346D0AE4173AFCE9FE3.
  Version1, bounded session, clean text fields/preferences. Its separate
  DefaultPath LevNote/native-session.json is NOT the app host session path.
- [Binary wire](<F:/AIproject/LevNote/native/core/session/binary.go#L28-L38>):
  2815B/95lines;6AF880BE6931D0284C8B971C452ED9D0C632A1DA257EEF7867CF202ACD9FBB77.
  Binary path-only record; rejects draft/encoding/BOM/originalPath/text payload.
- [No terminal newline writer](<F:/AIproject/LevNote/native/core/session/output.go#L14-L16>):
  2969B/121lines;672355D4087018592AA0E9EA0D15A4E4C06D2198B36571EF973914EA7BBDBB4B.
- [Selection shape](<F:/AIproject/LevNote/native/core/text/buffer.go#L37-L38>):
  exported Anchor/Head range; nested session key check follows current wire.
- [Reviewed plugin example](<F:/AIproject/LevNote/native/examples/plugins/diagram-notes.json>):
  179B/9lines;317FB43C43863A092FF7AA346C1A436A26C75A8BE41DC5E74C7F184AAE529A07.
  Data-only schema3/id diagram-notes/api1/.diagramnote/builtin.drawio. Root must
  pin current reviewed copy later; this historical read is not activation.
- [Current native coordination](<F:/AIproject/LevNote/native/AGENTS.md#L27-L43>)
  supplies local Windows-only/disjoint fresh scope/historical immutability;
  [physical limits](<F:/AIproject/LevNote/native/AGENTS.md#L595-L598>) prohibit
  homemade UI input automation and physical IME/resource claims from hidden hosts.

## Acceptance remaining with root

Source freeze/current Windows product gates/expanded shipping-source manifest,
helper parse/C# compile/runtime readiness, package build/result/manifest review,
real fixture review+pins, final root receipt+contract and one actual bounded
Windows run all remain pending. TerminateJobObject+host join≤10s only joins the
host; failed subsequent Job active0 accounting remains FAIL/unverified even if
CloseHandle later triggers asynchronous Job kill. Handle disposal never turns
that failure into empty Job or product acceptance. See [limits](<README.md>).
