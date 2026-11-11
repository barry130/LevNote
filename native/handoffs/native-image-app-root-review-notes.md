# Image app integration review notes

2026-11-10 root read-only WIP review; Astra is sole implementation owner.
These are concrete paths to resolve before final integration verification,
not an instruction to repeat passed prerequisite suites. Preserve relevant
failures and use controlled delayed-task/dispatcher tests.

1. Window close during pending copy: current CanClose persists/removes all tabs
   without resolving binaryJob. shutdownBinary later joins and calls report for
   a committed destination, but OnClosed has already removed the user-visible
   window. That does not visibly report a committed/stale-copy outcome. Keep the
   close decision pending/refuse the first close with a clear message while
   explicit binary work resolves, or use an equally small lifecycle that shows
   the actual outcome before accepting close. Do not wait for codec/IO in the UI
   close callback or silently hide committed results. Test before-commit cancel
   and committed result delivery with a queued/closed initiating tab. A direct
   Shutdown test still must join and clear safely.

2. openBinaryRequest after successful task, updateWait(deliverBinary) timeout:
   the current path calls task.abandon() and returns, but a.binaryJob and lane
   reservation remain. Since abandon clears ready, later deliverBinary cannot
   completeBinaryExplicit/release it. Avoid permanently reserving the lane or
   opening the abandoned tab later. Add a controlled dispatcher timeout test
   then prove subsequent binary work remains usable and snapshots are released.

3. Headless/modal ChooseOpen: ChooseOpen defers delivery and calls OpenMany;
   headless openRequests runs synchronously. openBinaryRequest waits j.done
   after deliverBinary returns early for modalDepth, so the caller cannot unwind
   the modal scope. A direct test should call ChooseOpen with the existing fake
   dialog and PNG path; resolve without weakening the real modal delivery guard.

4. Memory accounting needs an explicit transactional exception: an accepted old
   image/snapshot can coexist with a newly read explicit open/reload result until
   safe UI adoption. Reload failure must preserve old data. Either narrow retained
   references or document and test this bounded overlap; don't claim one full
   source throughout when there can be two. Review queued explicit requests that
   capture source bytes while a different active image exists. The GLM reviewer
   is independently tracing actual worst live ownership before root settles the
   final policy; no unrequested general cache/framework redesign.

5. Source snapshot acquisition is currently before checking image suffix mismatch
   at decode completion. Make the concrete mismatch warning accessible through
   preview and byte fallback; successful byte copy must still be exact source,
   never re-encoded. Include plugin disable/re-enable and save-to-nonimage-suffix
   cases in the same route tests where useful.

6. setMode currently calls releasePresentation -> deactivateBinary for a
   same-active-tab image/bytes switch. This drops a verified held snapshot even
   though no other tab is being activated, so source deletion followed by
   clicking 查看字节 loses the ability to inspect/copy the accepted bytes. Keep
   one active-tab snapshot through image/byte mode and plugin presentation
   changes while disposing the bitmap and invalidating decode publication;
   real tab deactivation/close must still release it. Use an explicit narrow
   presentation-vs-deactivation distinction, not two retained source copies.
   Test source-deleted held image -> bytes -> copy and disable-to-byte fallback.

7. applyEditorPreferences currently gives binary inspector the global editable
   line-number/wrap settings. Keep readonly byte presentation capability-aware
   and preserve the source-editor settings for actual text documents. This is a
   small UI consistency audit, not a new preference subsystem.
