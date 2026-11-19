# Outline post-callback candidate · read-only review04 · 2026-11-12

## Decision and scope

**No contract-conforming failing public App.View/Tester case was established. The candidate is closed as unproved, not a fixed defect or a passing regression.** No revision04 product source, test, source freeze or product execution was created. Revision01/02/03, their assertions and the 25-case selector remain unchanged. This document is a planning/review artifact, not a substitute for sole-owner RED evidence.

## Snapshot and Resolve: the proposed mutation is outside the contract

[The neutral OutlineSource contract](<F:/AIproject/LevNote/native/ui/editor/contract.go#L60-L69>) requires inspection without mounting a view, flushing a draft, changing selection or constructing graph layout. A stub whose OutlineSnapshot or ResolveOutlineItem directly calls App.Select/Open/NewDocument to switch the active tab does not automatically qualify as a conforming production callback. Keeping two Buffers at version0 and returning captured literal IDs avoids a released-buffer dereference, but does not repair that contractual violation.

[ensureOutlineSnapshot](<F:/AIproject/LevNote/native/app/outline.go#L184-L253>) indeed performs cache publication after the synchronous snapshot return and uses Resolve as a selection hint. The absence of a post-call guard alone is not proof that an allowed call path can replace its owner. No such path was found in the reviewed shipped providers.

## Factory and CurrentOutlineProvider: concrete reviewed paths

- [Markdown factory and refresh](<F:/AIproject/LevNote/native/modules/markdownview/outline.go#L30-L84>) retains only Host.Buffer and performs bounded parsing; it does not retain or call Host.OpenReference, RecoverDraft, ShowSource or Changed.
- [Mindmap factory and refresh](<F:/AIproject/LevNote/native/modules/graphview/outline.go#L22-L81>) retains Host.Buffer and parses descriptors without view/layout/metadata mutations.
- [InstalledSourceOutlineFactory](<F:/AIproject/LevNote/native/modules/modules.go#L78-L101>) forwards copied declarative configuration to [NewOutline](<F:/AIproject/LevNote/native/modules/sourceview/outline.go#L36-L69>); NewOutline retains Buffer/configuration and returns invalid configuration as a fallible snapshot error. It does not invoke lifecycle-capable Host callbacks.
- [Markdown CurrentOutlineItem](<F:/AIproject/LevNote/native/modules/markdownview/outline.go#L166-L185>) refreshes descriptors, reads caret state and resolves an ID. [Mindmap CurrentOutlineItem](<F:/AIproject/LevNote/native/modules/graphview/outline.go#L209-L215>) reads the selected ID and resolves it. [Scene CurrentOutlineItem](<F:/AIproject/LevNote/native/modules/graphview/scene_preview.go#L190-L196>) checks loading/stale/error flags and its ID index. None of these query bodies switches tabs, reloads Buffers or invokes Host.OpenReference/RecoverDraft/ShowSource.

The bare Adapter.Outline factory and CurrentOutlineProvider declarations do not repeat the complete inspection-only comment. This review therefore does **not** assert that every future custom implementation is prohibited from every side effect, nor that all possible post-callback ownership hazards are disproved. It establishes that no legal lifecycle-changing callback was located in the concrete implementations reviewed, so a fabricated lifecycle-changing stub is not submitted as genuine product RED.

The actual revision03 [editorHost](</home/user/levnote-dev-experience-20261112-01/revision-03-source/native/app/reference.go#L161-L190>) does expose OpenReference/RecoverDraft and owner-checked ShowSource. [openReference](</home/user/levnote-dev-experience-20261112-01/revision-03-source/native/app/reference.go#L202-L222>) routes a local reference through App.Open. Existing-tab selection eventually uses public [Select and selectChecked](<F:/AIproject/LevNote/native/app/app.go#L384-L430>), which flushes/releases the old presentation before switching Active. Those are genuine action paths, but their existence on Host does not prove that a read-only outline query or the shipped constructors invoke them.

## Existing consumption guards and limits of this conclusion

[outlineSidebar](<F:/AIproject/LevNote/native/app/outline.go#L123-L172>) captures the current tab, Buffer/version and controller generation after ensuring the snapshot. Its callback guard includes [outlineNavigationContextMatches](<F:/AIproject/LevNote/native/app/outline.go#L422-L437>), which separately verifies OutlineTabID, OutlineBuffer and factory generation. A different-tab factory return is not accepted for row label/submission merely because both Buffer versions equal0. [releaseOutline](<F:/AIproject/LevNote/native/app/outline.go#L36-L55>) detaches and clears the cache before releasing its controller; [releaseTabPresentation](<F:/AIproject/LevNote/native/app/app.go#L1108-L1124>) invokes that reset during the normal tab transition. The next source lookup rejects mismatched owner metadata.

These are source-level observations, not executed safety tests. In particular, same-tab Buffer replacement from an otherwise legitimate future constructor would require separate proof; it is not declared safe by this review. Empty/error focus already uses CancelFocus and is not a new defect.

## Input identity and delivery

The F working tree's eight scoped inputs now match [manifest03](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/freeze-03/source-inputs.json>) exactly. [The actual mirror receipt](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/working-mirror-receipt-03.json>) is 7624B, SHA256 `136735d88a33052945aeacc5e2ebf5ecd81ea669501a220b3fa9c3627e95cecd`. The exclusive [revision03 report](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112.md>) is 14089B, SHA256 `ee31ec3a5cfa5be3b83422c6848830fb407a109d7d73ba50f6188b97027c123b`; its complete88lines were read back. No old report or freeze was overwritten without its allowed old-identity check.

Contract4655B/`cc7d5510f81abbc10704cf0ef32f4ff9fc6f96de66937d1af19ded312b4fca8d`, modules3275B/`a40a02288dbe60f387f8d658c319942b5d1339f6044aa54e34d7401e19a7d62e`, Markdown outline5623B/`2212b10063a8290bfcec755719c7647b2b93d90ebfb4d3152516fea545c16d58`, Mindmap outline6195B/`25efef1e60aa097808b204dfc4e1e09c5318e647767c9c4e550f6a17423413b4`, Scene preview7997B/`e3420fc1200b6454014674bbd135284329b7eaedc146c3efa17779c3ff8b597b`, and sourceview outline6567B/`6473b5a2ba26b0a6279cc8836a8dfd27da8cc5daee1d2e971905bc6f87676a32` matched between the actual revision03 box inputs and the locally read files. The narrow box metadata call printed its full inventory but ended in an outer transport timeout; it is not represented as an exit0 verification run. Local metadata comparison later completed normally.

The F reference.go belongs to the separately mirrored task5 source and differs from revision03. Its editorHost/openReference bodies were therefore read directly from revision03 on the box rather than assumed identical. Neither reference version was edited here.

The sole native-reliability lane retains the unchanged priority Tree GREEN02 and 25-case regression request. Its real outcomes remain pending; this read-only closure neither changes their inputs nor claims Windows/IME/PE/WM_CLOSE/Job or measured-memory acceptance.
