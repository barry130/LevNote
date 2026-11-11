# Independent app-list lifetime review

PLANNED2026-11-10, activates after `.30` closure and separate implementation
assignment. Owner DeepSeek reviewer, no children, source edits, tests, builds,
launches or private toolkit introspection.

Read native-list-handle-lifetime-next-review.md and implementation-task.md.
Review only concrete app TreeList/SearchList/OutlineList lifecycle changes.
Track public Handle identity, CancelFocus, clearing old callbacks/items, safe
ListState reset before build/late-layout, origin/generation/searchID/controller
revalidation, preserved intentional focus and existing scroll/selection UX.
Tree same-root refresh should remain unchanged. Search collapse should preserve
results where it did before. Distinguish source bookkeeping retention from
measured heap/RSS and public Tester behavior from physical OS input.

Read draft as available and report precise reachable concerns once. Later root
will request exact frozen-source review with actual focused logs/identities;
do not repeatedly poll or widen into vendor/modules. Publish
native-list-handle-lifetime-independent-review.md and release review ownership.
