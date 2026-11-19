# Feature parity editor · 2026-11-15

## Scope and status

Current assignment: Alt+Up/Down moves a logical line block; Shift+Alt+Up/Down copies it. Source/test writes are frozen after lead's passing editor-01 run (152 PASS / 1 existing SKIP). No Go test, build or vet was executed by gap-editor. No app, vendor, toolchain, existing test or shared document was modified.

Changed: native/ui/editor/editor.go. Added: native/ui/editor/line_commands.go, native/ui/editor/line_commands_test.go, native/core/text/line_edit.go, native/core/text/line_edit_test.go. This handoff is the only documentation output. Existing Tab/ShiftTab indentation remains unchanged.

## Behavior and integration

- Command IDs: moveLinesUp, moveLinesDown, copyLinesUp, copyLinesDown. HandleInput accepts exact ui.Alt and ui.Alt|ui.Shift with Up/Down; added modifiers do not trigger these edits. Commands share editLines.
- Buffer.PlanLineEdit produces one contiguous replacement plus directed UTF-16 selection. It excludes a selection endpoint at the next logical line's column zero. It operates on whole logical lines, including segmented/soft-wrapped lines, and keeps surrogate pairs valid.
- Moves exchange the selected block with one adjacent line. Internal mixed LF/CRLF separators and the existing joining separator remain literal; EOF without a terminator remains unterminated. Top/bottom boundary operations leave source, selection, history and active composition unchanged.
- Copies use the selected block's trailing separator, or the preceding line separator at EOF, or LF for a one-line document. The new selection follows the copied block. Repeated copies operate on that new block.
- Planning checks the resulting UTF-8 byte budget before materializing copied text. Rejection preserves source/snapshot/version/selection/history and an active or refused IME composition. Read-only commands consume the action without mutation; foreign groups and foreign composition owners are retained.
- Successful edits settle only their own accepted IME composition, then BeginGroup/ReplaceUTF16/SetSelection/EndGroup and notify once. IME settlement and the line edit have separate undo entries. One line command is one undo/redo including selection. Identical-line exchange moves selection without text history or OnChange.
- app source/split already share editor.New and Editor.View; no app wiring change is needed. No new public Editor selection API was added. Core PlanLineEdit is an immediate-use plan, invalidated by source/selection changes; it is not retained across callbacks.

## Added test identities

Core (5 top-level):
- TestPlanLineEditTextSelectionAndHistory (21 explicit examples)
- TestPlanLineEditBoundaryAndIdenticalNoHistory
- TestPlanLineEditCopyCapacityKeepsRedoAndComposition
- TestPlanLineEditCapacityExactAndLongLogicalLine
- TestPlanLineEditInvalidActionIsReadOnly

Editor (9 top-level):
- TestNativeLineCommandsKeyboardSelectionUndoRedo (all four real Tester shortcuts)
- TestNativeLineCommandsBlockEndpointAndLogicalWrapping
- TestNativeLineCommandsRepeatedCopyAndEmptyEOF
- TestNativeLineCommandsReadonlyAndForeignGroup
- TestNativeLineCommandsExactModifiersAndReadonlyKeyboard
- TestNativeLineCommandsNoOpKeepsHistoryAndMarkedInput
- TestNativeLineCommandsCompositionHasSeparateUndo
- TestNativeLineCommandsCapacityAndRefusedCompositionAreAtomic
- TestNativeLineCommandsCapacityPreservesRedo

## Verification and known boundaries

All five Go files passed the fixed .localtools/go1.27.3/bin/gofmt.exe parser. First -w attempt exited 2: temporary file creation for line_edit.go, line_edit_test.go and line_commands_test.go returned Access is denied; editor.go and line_commands.go formatted normally. The three affected files were formatted via gofmt stdout then the file write tool, all formatter calls exit 0. ACLs were untouched. Root's concurrent palette-02 compilation saw the initial escaped-newline draft at line_edit.go:193-194; that literal was fixed before the formatting-stable notification, and the original compile receipt remains with lead.

git diff for editor.go contains only four command cases and the precise modifier/key routing. No old assertions or skips changed. Root's editor-01 completed with exit 0: 153 RUN / 152 PASS / 1 existing SKIP; all 418 captured inputs unchanged. Receipt: .localtools/feature-parity-20261115-editor-01/result.json (actual args: gate.sh test -json -count=1 -timeout=300s ./core/text/ ./ui/editor/; elapsed 4.473s). gap-editor read the 25-line receipt in full. Root owns subsequent broader regression. Physical Windows Alt/IME behavior is not claimed by source inspection or synthetic UI tests. No comments, bracket wrapping, configurable indentation, clipboard line-copy behavior or global command-palette entries were added in this scope.
