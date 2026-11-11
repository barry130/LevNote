# Reopen a closed saved file: bounded next plan

2026-11-10, read-only. Recommendation: implement after .26 closure/activation,
with a small acknowledged-open prerequisite in app. No source/test/build/launch
or package evidence changed; this plan is the only task output. No web research.

## Product contract and stored authority

Ctrl+Shift+T reopens the most recently successfully closed **saved local file**
from its current disk bytes; it does not undo closing a draft. Unsaved documents
and recovered drafts without their own saved path are excluded. Never infer a
path from OriginalPath: restored drafts are deliberately document.New with a
separate original-path hint (core/session/session.go:197).

Use at most16 UI-owned entries `{id uint64, path string}` plus one in-flight
request ID. Clone the authoritative path; deduplicate by the existing canonical
case-insensitive open identity and move the newest close to the top. Keep a new
entry ID when a path is reclosed. Clear vacated slice slots/evictions and on
shutdown. No Tab/Document/Buffer/View, source, undo, selection, image bytes,
plugin/view choice, filesystem handles or persistent session fields. Cap each
path at128KiB UTF-8, never truncate; a noncandidate/overlimit path is omitted
with an explicit close notice. Reject empty/relative/UNC/network paths before
canonicalization; use existing local-reference rules (app/reference.go:31),
not an empty-path filepath.Abs fallback (app/app.go:625).

**Choose path-only metadata for the first slice.** Reuse current Open decoding:
UTF8/BOM UTF16 are detected; unknown legacy data reaches the readonly inspector
and requires explicit GBK/GB18030 selection again. This is a visible limitation,
not a new encoding guess. Current loadForOpen already reports the inspector
fallback (app/app.go:717); reopenWithEncoding performs the explicit choice
(app/app.go:600), and core decodeWith enforces round-trip bytes
(core/document/encoding.go:23). Do not silently force a remembered code page on
a replaced file. One-step preservation of an old explicit encoding would be a
separate refinement requiring verified baseline/choice policy; it is not
necessary for a safe useful first slice. Do not retain BOM or readonly flags:
current stable read supplies both (core/document/document.go:121).

## Record only a completed individual close

CloseTab calls prepareClose (which can SaveAs), persists the remaining tabs,
then removes the target (app/app.go:932,1084). Capture candidate metadata after
prepareClose succeeds so a newly saved/SaveAs path is current; check the same
target still occupies the intended removal slot after any dialog reentry.
Append history only after commitClose returns true and that exact tab was
removed. Invalid index, cancel, failed Flush/save/persistence or aborted cleanup
adds nothing. removeTab owns presentation/export/watch cleanup and backing-tail
nil clearing (app/app.go:1019); do not add history there or bypass it.

An explicit discard of a saved document may add its saved path, but reopening
reads disk only: no unsaved content is retained or restored. Discard of an
unsaved/recovered draft adds nothing. Whole-window CanClose iterates prepareClose,
persists its discard decisions, joins export cleanup, then calls removeTab
(app/app.go:1047); that route and Shutdown must never populate this history.
Byte inspectors with a real saved path and image tabs are valid path candidates;
use t.path(), which handles bounded images and oversized byte-only tabs
(app/binary.go:21). Do not depend on text Doc being present for those tabs.

## Small prerequisite: an actual open/select receipt

Do not pop by observing openRequests' done channel or current tab later.
Its done means queue completion, and existing is set even when Select refuses
(app/app.go:649). openBinaryRequest true is also not success: busy/read failures
can return true; completeBinaryExplicit can reject current Flush/Select while
j.err is assigned only out.err (app/binary_actions.go:39,166,209).

Extend the private openRequest with optional numeric request/entry identity,
and carry it into the specific binary open job. Report one terminal UI outcome
`opened/selected` only at actual successful Select or adopted new tab; failure
is separate. No live tab crosses the worker boundary. Existing public Open,
OpenMany, OpenAt ordering and signatures can stay unchanged. Text append uses
addDocument's Flush barrier (app/app.go:428); recheck duplicate at UI adoption
for this request, discard an unused loaded document, and Select the existing
tab only if its current Flush succeeds. Binary adoption already rechecks
duplicates and Flush (app/binary_actions.go:209); attach the receipt there.

Reserve one reopen attempt; held/repeated keys or a second command while busy
start no extra request. The success callback removes **that exact entry ID**,
never the current top or whichever path was opened most recently. If a later
close replaces/evicts the old entry, its new ID survives. Clear only the matching
in-flight ID on terminal completion. Missing/unreadable file, failed UI dispatch,
shutdown, canceled/adoption-refused binary job or Flush failure retain the entry.
Preserve existing timeout/abandon cleanup; make a timed-out attempt terminal and
release its UI reservation on next delivery/command, without resurrecting its
late result. The binary lane stays single and bounded; no second queue/decoder.

This receipt is the exact prerequisite preventing a safe shortcut-only patch.
Keep it private to app/open/binary lifecycle; no generic command framework,
session schema, format contract or document-core change is needed.

## Native command, focus and outcomes

Keep normal “打开” and add a compact adjacent “▾” native menu button labelled
“最近关闭的文件”. Menu first item: “重新打开已关闭文件  Ctrl+Shift+T”; below it
show up to16 bounded basenames with full-path tooltip. Each item selects its
own ID; this gives an explicit route past a missing newest file while retaining
it for retry. No automatic skip/pop on failure. Empty menu/command says
“没有可重新打开的已关闭文件”. Tooltip explains “重新读取磁盘文件；不恢复已放弃的修改”.
MyGo public PopoverMenuButton/Menu.Item supports keyboard and accessible names
(vendor/github.com/egoist/mygo/ui/popovermenu.go:55, menu.go:389); no widget-tree
inspection. Check760x480toolbar/menu bounds; existing toolbar is46DIP and has
no file menu (app/view.go:42), so avoid another full-width toolbar label.

Wire ui.Cmd|ui.Shift + KeyT in existing shortcuts (app/view.go:533), consume
ShortcutRepeated, and make empty/busy checks before Flush or focus changes.
Use one action for shortcut/menu. During modalDepth>0, restoring or shutdown,
refuse immediately with no queued shortcut. Guard visible Find query/replacement
and GoToLine preedit as selectAdjacentTab does (app/app.go:402): tell the user
“输入法正在组合文本；请先完成组合后再重新打开文件”, preserve focus/preedit, allow
explicit retry after commit/cancel. Disable opening the menu during that guard
so its focus transfer cannot preempt it. Keep existing public focus-leave
reconciliation (app/search_navigation.go:95; source_navigation.go:291).

Source/rich composition still uses normal flushTab (app/app.go:1134), preserving
draft conflicts/recovery. Recheck a reopen's guard at final adoption because
composition/modal state can change during IO. No text-result adoption inside
a modal: fail the attempt and keep the entry; binary results already defer
under modalDepth (app/binary.go:246), then revalidate before adoption. A missing
or refused file reports actual error plus “记录仍保留，可从最近关闭选择其他文件”.
Success says “已重新打开磁盘文件 …” / “已切换到已打开文件 …”; keep any readonly,
unknown-encoding or disabled-plugin warning. Replaced files use current bytes,
current type/plugin configuration and normal readonly/fallback checks, never
the old content. A successfully adopted byte fallback counts as opened.

## Focused acceptance to authorize with implementation

- Clean saved text, last-tab/zero-tab, image, byte inspector;16-entry eviction,
  case-insensitive dedup and cleared tails; no retained document/bitmap fields.
- SaveAs during close records new path; saved discard reopens original disk
  bytes with empty undo; unsaved/recovered drafts never return. Cancel, Flush,
  save/persist/cleanup refusal and whole-window quit do not add history.
- Synthetic shortcut and native menu share exact IDs; held key opens once;
  existing tab selection avoids duplicate and respects failed current Flush.
- Pending image/read then newer close/same-path reclose/eviction/ordinary Open,
  busy/timeout/cancel/shutdown/late completion: only the acknowledged ID is
  consumed. Cover binary duplicate Select refusal as in binary_test.go:737.
- Missing top retained; menu opens older candidate; recreated path retries.
  Changed bytes/readonly/plugin-disabled/oversize targets follow current routes.
- UTF8 BOM/CRLF, UTF16, unknown GBK/GB18030 inspector then explicit choice:
  opening never writes original bytes, guesses legacy encoding or restores edits.
- Visible Find/GoToLine preedit blocks without focus loss; source/rich Flush
  succeeds or preserves conflict draft; modal start/during-IO adoption guarded;
 760x480menu/toolbar and tab reveal stay usable. Focused tests first, one affected
  final app test/vet after freeze; no product/resource claim from synthetic input.
