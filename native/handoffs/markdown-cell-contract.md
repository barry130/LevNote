# Markdown single-cell editing — approved second slice

Status: ACTIVE after root accepted the20:49 first table slice. This contract
supersedes design proposals where they differ. Root plans/reviews only; child
agents implement, execute tests and build. The accepted first-slice binary is
13,384,192B, SHA256
7632B1C50D9536B25A12CF9FA7E288B62F903A3B4ABA9896EF8849356864BBCD,
UTC2026-11-08T12:49:19.7290537Z. Its source/layout/export remain the baseline.

## Scope and owners

Give valid table header/body cells a direct literal-source edit path using the
ONE existing independent editor. Preserve the complete-table source button,
responsive grid/stacked preview and all accepted task/link/reference behavior.
No row/column toolbar, formulas, active links, inline decoding, new dependency,
generic input framework, or per-cell editor instances.

- deepseek_reference_view exclusively owns modules/markdownview/** product and
  focused tests/images. Publish action/field semantics promptly for app tests.
- deepseek_plugins_app finishes accepted first-slice artifact docs, then owns
  app/** actual registered-adapter acceptance and final product documentation.
  It remains sole final integrator after root's focused review. No shell
  product edits without a demonstrated reachable defect.
- glm_formats_finish is READ-ONLY independent reviewer of current source and
  test oracles. No product/test/doc edits or executions; own handoff only.

formats/markdown/**, core/**, ui/editor/**, vendor/** and scripts/** remain
frozen. No child delegation or peer restart. Compact GLM/DeepSeek before200K.

## Native actions

Use existing cell boxes/text as focusable native RoleCell/RoleColumnHeader
targets, with stable numeric row/column keys and bounded accessible labels
containing row/column identity. Explicitly apply Focusable; role alone is not
proof of keyboard reachability. Tab plus Enter and ordinary double-click are
required. No new per-cell buttons or popovers. Space retains scrolling.

Single pointer selection/drag remains available. A cell double-click must be
consumed by that cell and not also open the row's whole-table editor. Existing
whole-table button and non-cell/fallback double-click routes remain available.
Readonly cells never create drafts. Native pointer/keyboard paths must be
tested, not only direct invocation of a validation helper.

Capture Buffer/version, numeric row identity, row/column index, complete table
UTF16 span, exact TextRange UTF16 span and bounded source identity at the
actual gesture start. No per-frame clone of the32KiB table. Keyboard captures
the actual focused cell; virtualized rows and duplicate identical tables may
not redirect the action. Avoid retaining full source on a mere frame/focus.
Flush the existing draft before opening the captured target, then rebase and
revalidate exact source and parsed cell identity. An unrelated insertion before
the table may rebase; any edit inside the guarded table, replacement, reorder,
expired journal, group, Partial/oversize or readonly must refuse visibly.

If flushing a previous cell changes this same table, the conservative captured
target may be refused; tell the user to select again without claiming that the
prior draft was not saved. Failed prior Flush keeps its draft and blocks open.

## Staged draft and commit

The cell draft initially contains exactly the chosen TextRange, excluding its
existing ASCII padding. Keep existing padding, separators, delimiter, outer
pipes and line endings untouched. Empty ranges use the parser's right-edge
anchor. Reuse openDraft/Flush/recovery with an explicit private staged-cell
mode; no exported editor/format/session API is needed.

Unlike normal and whole-table drafts, cell OnChange remains LOCAL. Do not call
host.Changed, mutate the host Buffer or hold a host transaction while typing.
Pending IME text remains local too. The existing editor's native composition
must complete before validation. Local undo may operate before commit; after
commit the host receives exactly one ReplaceUTF16 and one Changed callback.

Commit on host Flush (save/export/switch/close/navigation), 完成编辑,
Ctrl+Enter, or Escape. Escape attempts commit, consistent with current drafts.
Expose 放弃草稿 whenever a staged cell is open, including before an error; this
explicit action clears only local staged text. On failure keep the same draft,
selection/text and recovery controls; host bytes/version/history stay unchanged.
The host must not clear, navigate away or dispose the view on a failed Flush.
If Release is invoked directly as a cache-lifecycle action, the existing
ReleaseView may free geometry/focus and the preview may drop parsed caches,
provided the exact staged draft, selection/history, cell identity and recovery
path remain intact and return on the next Draw. Keeping an obsolete parsed
document is not required. Verify this direct-Release case explicitly. Recovery
creates a copy through the existing host callback and clears only on success.

On successful commit, including no-op, close the cell draft and invalidate the
preview. Closing avoids retaining a now-different trimmed field after inserting
leading/trailing padding. No-op does not call Replace/Changed or add history.
Normal/whole-table draft immediate synchronization remains unchanged.

The cell panel uses the current independent editor with a compact bounded
height and a visible row/column label. Show that content is staged until save
or finish. Uncommitted cell drafts remain in this running view, not automatic
session snapshots; disclose this in the help/hint and docs. No new persistence
mechanism or crash-recovery claim in this slice. Preserve the existing large
draft segmentation/recovery route instead of silently truncating invalid input.

## Validation before the one host write

1. Check Buffer identity/current state and exact table+cell mapping. Current
   full Markdown parse must be non-Partial at activation and commit. Reuse a
   verified cached parse when current; no parse-per-frame or parse-per-key.
2. Check draft.Buffer.LenBytes <=4096 BEFORE String materialization for a
   changed cell. This is an explicit raw draft byte cap, including newly typed
   whitespace, and not a claim about the parser's trimmed-cell cap. Reject
   CR/LF, unescaped structural pipes and invalid UTF8. Keep input in the draft.
3. Check prospective table bytes <=32KiB and host.LenBytes+delta <=MaxParseBytes
   before constructing/parsing candidates. Compare byte lengths without an
   unbounded host/prefix copy. pendingDraft may use unequal byte lengths to
   avoid materializing an oversized local draft during repeated rendering.
4. Reconstruct ONLY the current bounded whole-table candidate by replacing the
   exact TextRange. No other bytes may change. Parse it with markdown.Parse;
   no second table grammar in the view. Require exactly one valid BlockTable
   consuming the complete candidate, with no Problem/Partial, and the same
   header/body row count, per-row cell counts and alignment values.
5. Verify delimiter and all non-target cells still describe their original
   source bytes at ranges translated by the replacement byte delta. Later
   ranges must shift; an absolute-range-equality comparison would wrongly
   reject legitimate length changes. Only the target's content/padding
   classification may change; never reinterpret another cell or separator.
   Original padding outside TextRange remains unchanged by construction.
6. Only then replace the exact UTF16 field once, clear staged state and report
   one Changed. Candidate shape failures keep the draft with concrete guidance
   to use whole-table source editing for structural changes.

The candidate oracle is authoritative for boundary interpretation. `|a` ->
`|` is valid one-empty-cell under the accepted grammar and MUST NOT be rejected
by an invented blanket empty-cell rule. A no-outer `a | b` row changed to
`a | ` loses a cell and must be refused. Odd trailing backslashes that absorb
a separator, heading/list reclassification, and outer-pipe reinterpretation
must be caught without normalization. Escaped pipes remain literal.

## Focused acceptance

View owner: actual double-click and Tab/Enter on header/body/empty/second
duplicate table and narrow stacked cells; ordinary selection preserved; row
handler cannot double-open; readonly and transition. Capture/Flush order,
insertion-before rebase, same-table conflicts, stale/replaced/group/expired/
Partial/oversize refusal. Several local key/Compose changes leave host intact;
one finish writes exactly once; failed finish retains draft and selection;
no-op/discard do nothing; local correction after invalid input succeeds.

Exact-byte fixtures include Unicode/astral, CRLF and EOF, optional/mixed outer
pipes, padded and all-padding empty cells, compact one-column valid empties,
no-outer edge loss, odd/even backslashes, backticks, newline/pipe/block-boundary
refusal. Check4096/4097 raw draft bytes,32KiB/+1 candidate and document-size
boundary. Include valid shorter/longer edits so delta handling is exercised.
Release/cache/virtualization and no-idle-frame regression remain bounded.
Capture narrow light/dark normal staged and invalid/recovery states; controls
must be visible in the integrated760x480 split app layout.

App owner: actual registered adapter with BOM+CRLF, staged typing vs host bytes,
one-undo/redo commit and Save/reopen; save/export/source/tab/close barriers;
invalid staged input blocks each action and retains disk/source/selection;
explicit discard/recover and no-op; readonly; whole-table edit/export regression;
representative760x480 light/dark cell draft and visible refusal controls.
Do not mirror view helper internals or count direct helper tests as native UI.

Reviewer reports concrete current findings or no findings to
handoffs/markdown-cell-independent-review.md. Root reviews focused source,
actual results and images before the sole full test/vet/build/hidden-startup
integration. No resource sampling or Windows IME/GPU acceptance claim.
