# App closed-tab backing retention result

2026-11-09. Scope stayed in `native/app/app.go` (`removeTab` only) and the new
`native/app/tab_retention_test.go`. No package, build, product documentation,
launch or UI run was performed. Accepted `.10` and older package directories
were not written.

## Confirmed defect

Before the correction, `removeTab` shortened `a.Tabs` with:

```go
a.Tabs = append(a.Tabs[:index], a.Tabs[index+1:]...)
```

The shortened slice no longer exposed the removed tab through indexed access,
but its pointer remained in the still-live backing array. Last-tab removal left
the closed tab directly in the vacated slot. Closing several tabs left the last
shifted tab duplicated in the vacated tail.

The new test uses `newTestApp`, ordinary documents/files and public `CloseTab`.
It covers:

1. the last tab after 128 retained edits;
2. middle removal with remaining order, content and selected tab checked;
3. closing all tabs, with every slot beyond the final length checked;
4. `closeCancel` after a dirty recovery snapshot was persisted, with the full
   live pointer array, content, selected tab and exact session bytes checked.

The retention assertions inspect the complete live capacity
(`a.Tabs[:cap(a.Tabs)]`), not only `len(a.Tabs)`. They use no GC/finalizer timing,
RSS threshold or heap profile.

## Red evidence

All Go commands used the required wrapper. The final regression draft was run
against the old `removeTab`; it exited `1` and the three removal cases failed:

```text
> .\native\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestCloseTabBackingRetention$','-count=1','-v')
=== RUN   TestCloseTabBackingRetention
=== RUN   TestCloseTabBackingRetention/last_tab_with_history
    tab_retention_test.go:31: closed tab 0x1a5db8b60000 remains in live backing array at index 0 (len=0 cap=1)
=== RUN   TestCloseTabBackingRetention/middle_tab_preserves_remaining_order_and_selection
    tab_retention_test.go:63: stale tab 0x1a5db88e26e0 remains beyond slice length at backing index 2 (len=2 cap=4)
=== RUN   TestCloseTabBackingRetention/closing_all_tabs_clears_every_vacated_slot
    tab_retention_test.go:88: closed tab 0x1a5db88e29a0 remains in live backing array at index 0 (len=0 cap=4)
=== RUN   TestCloseTabBackingRetention/cancel_preserves_tabs_content_selection_and_recovery
--- FAIL: TestCloseTabBackingRetention (0.06s)
    --- FAIL: TestCloseTabBackingRetention/last_tab_with_history (0.02s)
    --- FAIL: TestCloseTabBackingRetention/middle_tab_preserves_remaining_order_and_selection (0.01s)
    --- FAIL: TestCloseTabBackingRetention/closing_all_tabs_clears_every_vacated_slot (0.02s)
    --- PASS: TestCloseTabBackingRetention/cancel_preserves_tabs_content_selection_and_recovery (0.02s)
FAIL
FAIL	levnote.local/native/app	1.677s
FAIL
exit 1
```

An initial regression draft had its cancel subtest compare the selected tab to
the wrong pointer while the active index was `1`. That assertion was corrected
before the product change. It is not counted as product-red evidence; the
corrected red run above is the evidence.

## Minimal correction

`removeTab` now shifts the surviving pointers, clears the obsolete tail pointer,
then shortens the slice:

```go
copy(a.Tabs[index:], a.Tabs[index+1:])
a.Tabs[len(a.Tabs)-1] = nil
a.Tabs = a.Tabs[:len(a.Tabs)-1]
```

Export cancellation, presentation disposal, watch cleanup, disk-check
invalidation, active selection, focus and recovery persistence are unchanged.

## Green evidence

The focused regression exited `0`:

```text
> .\native\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestCloseTabBackingRetention$','-count=1','-v')
=== RUN   TestCloseTabBackingRetention
=== RUN   TestCloseTabBackingRetention/last_tab_with_history
=== RUN   TestCloseTabBackingRetention/middle_tab_preserves_remaining_order_and_selection
=== RUN   TestCloseTabBackingRetention/closing_all_tabs_clears_every_vacated_slot
=== RUN   TestCloseTabBackingRetention/cancel_preserves_tabs_content_selection_and_recovery
--- PASS: TestCloseTabBackingRetention (0.07s)
    --- PASS: TestCloseTabBackingRetention/last_tab_with_history (0.02s)
    --- PASS: TestCloseTabBackingRetention/middle_tab_preserves_remaining_order_and_selection (0.01s)
    --- PASS: TestCloseTabBackingRetention/closing_all_tabs_clears_every_vacated_slot (0.02s)
    --- PASS: TestCloseTabBackingRetention/cancel_preserves_tabs_content_selection_and_recovery (0.02s)
PASS
ok  	levnote.local/native/app	1.793s
exit 0
```

The existing focused close, recovery and tab lifecycle tests also exited `0`:

```text
> .\native\scripts\go.ps1 -GoArguments @('test','./app','-run','^(TestFlushFailureRetainsViewForLifecycleActions|TestWindowCloseCancellationDoesNotRemoveTabs|TestWindowCloseSavesReopenRecordsAndRestoresWorkspace|TestWindowCloseCancelKeepsTabsAndRecoveryStable|TestExplicitlyDiscardedTabIsRemovedFromRecovery|TestClosePersistsOnlyTheRemainingTabs|TestCloseDoesNotDiscardTabWhenRemainingDraftIsOversized|TestCloseRejectsEscapeHeavyRecoveryBeforeChangingWorkflow|TestCloseStopsBeforeCapturingAggregateOversizedRecovery|TestCloseDoesNotRemoveTabsWhenRecoveryWriteFails|TestEnginePackageCloseRefusalLeavesPackageWorkAlive)$','-count=1','-v')
--- PASS: TestFlushFailureRetainsViewForLifecycleActions (0.01s)
--- PASS: TestWindowCloseCancellationDoesNotRemoveTabs (0.01s)
--- PASS: TestWindowCloseSavesReopenRecordsAndRestoresWorkspace (0.04s)
--- PASS: TestWindowCloseCancelKeepsTabsAndRecoveryStable (0.02s)
--- PASS: TestExplicitlyDiscardedTabIsRemovedFromRecovery (0.02s)
--- PASS: TestClosePersistsOnlyTheRemainingTabs (0.01s)
--- PASS: TestCloseDoesNotDiscardTabWhenRemainingDraftIsOversized (0.03s)
--- PASS: TestCloseRejectsEscapeHeavyRecoveryBeforeChangingWorkflow (0.04s)
--- PASS: TestCloseStopsBeforeCapturingAggregateOversizedRecovery (0.09s)
--- PASS: TestCloseDoesNotRemoveTabsWhenRecoveryWriteFails (0.01s)
--- PASS: TestEnginePackageCloseRefusalLeavesPackageWorkAlive (0.08s)
PASS
ok  	levnote.local/native/app	1.813s
exit 0
```

Scoped vet exited `0` with no output:

```text
> .\native\scripts\go.ps1 -GoArguments @('vet','./app')
exit 0
```

## Final files and release state

```text
6387CDF9938D3546C26CD8D387CA4B334A011A2A47456CBCB300E0AA5890527A  native/app/app.go
B0CA0D1E70A2C58F3F81EA943081550F6B0D3DF368A25B702E0562BCE241C316  native/app/tab_retention_test.go
```

The accepted `.10` directory still has 23 files / 23,622,214 bytes. Its host
hash remains:

```text
4EB096FAEA7CA79E71F0C9FF30E659478C0E2AA135803DDBF7C0AEE47A0AEF6C  native/bin/levnote-native-worker-20261109.10/levnote-native.exe
```

This evidence proves that the live app tab backing no longer owns removed
`*Tab` pointers after the tested removals. The test intentionally keeps each
closed pointer in its own local variable while examining the backing array. It
does not claim immediate process-memory reclamation and does not measure RSS or
working set. No UI resource comparison exists because the Computer Use helper
was unavailable.

The app correction is ready for the core/text owner's single combined dependent
fullapp check. No fullapp or package build was run here.
