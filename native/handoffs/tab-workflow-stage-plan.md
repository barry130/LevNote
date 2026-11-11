# Next stage: keyboard navigation and visible active tabs

2026-11-10 root decision. Start only after .21 binary/evidence acceptance.
Treat the two related app improvements as one small delivery stage so users get
a usable multi-document workflow without a new package for every few lines.

1. Implement conventional Ctrl+Tab / Ctrl+Shift+Tab through existing Select,
   preserving all failed-Flush/composition/export/persistence semantics.
2. Once focused shortcut cases pass, add one-frame active-tab reveal using the
   public ScrollIntoView API. No cross-frame widget pointer or retained Tab.
3. Run focused behavioral/Tester cases during work, then ONE final affected
   app suite/vet for the completed stage. No duplicate final full app suite
   between the two small increments if no independent acceptance requires it.
4. Review the combined small diff, then do a single later package delivery
   (.22 planned, exact date/version assigned at activation). Carry forward
   unchanged worker evidence; only host/UI changed, so no repeat managed worker
   request/benchmarks for this UI stage. A narrow Tester geometry capture may
   show the active tab visible; physical Windows UI/IME remains unverified.

Implementation uses `tab-keyboard-implementation-task.md` and
`tab-active-reveal-next-task.md` as scoped requirements. Root will assign a
single app owner to avoid overlapping edits. .21 docs remain a separate owner.
Keep documentation short: one usage row plus explicit source-vs-package state
until the next binary is accepted. No MRU popup, animation, new dependency or
persisted tab history.
