# Document growth admission — implementation frozen

2026-11-10 Astra. Completed assigned implementation and focused checks. Source
ownership is released for root review; no further changes or execution pending
from this owner without activation. No full package suite/vet, product package,
host/worker/resource/benchmark run or screenshot. Accepted .27 remains intact.

## Behavior

- `core/text` owns `DefaultMaxBytes=128MiB`; document's int64 disk-read constant
  aliases it. `New` retains UTF8 panic compatibility. `NewWithLimit` requires a
  positive immutable base; its fixed effective ceiling is max(base,seed bytes).
  Oversized decoded seeds remain whole, can shrink and regrow to that ceiling.
  Restore keeps buffer identity/base, clears history and re-seeds the ceiling.
- ReplaceUTF16 validates UTF8/range/surrogate then checks prospective bytes by
  subtraction before copying removed source or allocating edit/history pieces.
  Rejection leaves version/text/selection/history/group unchanged; no-op behavior
  remains compatible. `errors.Is(ErrSizeLimit)` and typed limit carry actionable
  Chinese guidance. Undo/Redo/CancelGroup replay accepted states unchanged.
- ReplaceAll preflights its actual reverse application order before BeginGroup;
  any oversized intermediate is refused even when final output would fit.
  Literal strings are shared. Regex expands once per match, retains at most
  existing MaxMatches outputs, and accepted planned output bytes fit the ceiling.
  Indent checks all positive growth by division/subtraction before unmark/group;
  outdent remains available and selection direction/CRLF/history are retained.
- Size-refused first/ongoing/final IME input keeps a small error and buffer owner,
  never the rejected string. First refusal opens no group; ongoing refusal keeps
  prior accepted mark/group. Flush/Save/Select/CloseTab/CanClose refuse until valid
  marked/final retry or explicit Esc. Focus loss/UnmarkText/ReleaseView do not
  settle it. Navigation, selection/edit commands and delayed cut cannot mutate
  unresolved preedit. A focused element Escape shortcut outranks window Escape;
  after resolution ordinary Escape returns to the host. Cancellation targets the
  original buffer if the public buffer pointer changed. Normal paste refusal is
  nonsticky. Injected Paste no longer first reads the native clipboard as well.
- Existing Markdown/graph writeback retains independently edited full drafts on
  size refusal. Actual view/app checks demonstrate stopped save/switch/close,
  unchanged original disk/buffer and successful shortened retry or RecoverDraft
  into a separate savable document. No format or app runtime edits were required.

## Focused evidence

All commands ran via scripts/go.ps1 with `exit $LASTEXITCODE` immediately after
the wrapper; child-process exit and both streams are preserved under
`document-growth-budget-evidence-20261110/`. Every fixture is small. Nineteen new
top-level tests cover UTF16/GBK/BOM/CRLF decode/save/revert, byte/surrogate/overflow
boundaries, snapshots/redo/foreign groups, ReplaceAll/indent and real draft gates.

1. `01-focused.ps1`, `-run ^TestGrowth`, seven affected packages: actual exit1.
   text0.825s,document0.939s,search0.889s,editor1.643s,markdownview1.748s,
   graphview2.036s passed. App failed only first/ongoing explicit Esc cancel:
   window shortcut swallowed source raw input. Original stdout/stderr/result kept.
2. Editor-only focused-element Escape correction plus real global/local routing
   assertion. `02-focused-correction.ps1`, editor/app `^TestGrowth`: exit0;
   editor0.925s,app2.526s. Six other package results from01 were not rerun.
3. `03-existing-seams.ps1`: exact selected23existing tests, no growth repetition
   or full suite. Search1.044s,editor2.327s,markdownview1.685s,graphview2.327s,
   app4.058s all pass, exit0. Selection covers successful composition, CRLF indent,
   undo/clipboard/regex, rich Flush and utility-field Escape routing.

No failed stream was replaced or relabeled PASS. All three stderr files are
empty. Initial identities and final change accounting show only editor.go and
its new test changed after01; the other nine source files are identical.

## Exact frozen identities

Manifest: `document-growth-budget-evidence-20261110/source-frozen.json`, SHA256
`B0A212FF61660DDADD9C5A040289FA6057E52B36E3A030B91BA32196A7C8D275`.
Paths below are native-relative; four runtime files plus seven new test files.

| Path | Bytes | SHA256 |
| --- | ---: | --- |
| core/text/buffer.go | 18700 | 0AF9281A6051B8715854665043CBCC3EA9DFBB89BA5D028BA719B312B244FB21 |
| core/document/document.go | 12537 | 6859017748C8884DD9CC84B185A405C0A78089AA0D84F191024489CB24C11219 |
| core/search/find.go | 4677 | D3D7EDD2D2A2CD47C1D362A225387566D3F8709A24D92A479CCD57207645A4F5 |
| ui/editor/editor.go | 23437 | 09BEA640C8F1953E615B43088A7F58FA25AA8342B59694CC4023CBD36A885EBD |
| core/text/growth_test.go | 5660 | D7284C8285BD48061BB31224C6DF1596C1580F413164192F80883D2A3A518F38 |
| core/document/growth_test.go | 3318 | 0E8E8803FA22EEE748897F59FCEE148F16E4D920E98991A5CA0640BCDE847E45 |
| core/search/growth_test.go | 2966 | 6C0DEBA2F982F0A4A10D5F7288C13F9BE4030CAEF6BDA44EDAD0DF9AA5D64D3D |
| ui/editor/growth_test.go | 9109 | 21FB5BCC2464AFEA0D5E5FC76236B5FF4CB4AF110370E0F55E756CCD00F775A8 |
| modules/markdownview/growth_test.go | 2892 | 1F45BCE17706E8CEE44C26727C920B2AE08F68A1605B627954B80A436F9B0990 |
| modules/graphview/growth_test.go | 1903 | 0F14DE124EB3108FC07F5913A29A9512EAE892DD9DDC7BC6FBC53BD4AB594119 |
| app/document_growth_test.go | 7026 | 613446886000151828C5DB505A24CD9B3CD9A2617B080E67B82D55C705F7B47D |

## Limits and next gate

This bounds logical content growth, not RSS, transient clipboard/codec/individual
regex expansion allocations, source copies, history/group accumulation or all
documents to128MiB. No active-group retention policy was added. Existing format
multi-step rollback may advance revision; no blanket unchanged-revision claim.
Tests use MyGo public headless input/app/view seams, not physical Windows IME.
The pending refusal retains accepted preedit and actionable guidance, not a copy
of oversized rejected input. After focus leaves, refocus the editor to retry or
press Esc. Direct external Buffer mutation is outside editor ownership; host
lifecycle must continue using Flush and existing document identity checks.

Root may activate one final combined affected test/vet after review. No .28
execution is authorized by this result itself.
