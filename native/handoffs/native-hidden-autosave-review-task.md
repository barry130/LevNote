# Hidden/minimized autosave read-only review

ACTIVE2026-11-10 for GLM. No child delegation. Do not edit product/runtime/tests/
docs or execute tests/builds/hosts. .23 delivery source is frozen. Write only
native-hidden-autosave-review.md with a concise supported finding and the
smallest next implementation if warranted. No broad architecture proposal.

The narrow native-image-hidden-readiness-review.md established that hidden
MyGo Surface may not redraw after its first frame while automatic session
persistence is initiated from App.View. Normal close persists synchronously.
Now distinguish the synthetic hidden launch limitation from a real user case:
editing, then minimizing/losing visible presentation before the900ms debounce;
opening files while minimized; and restoring a real window afterward.

Trace actual app/state.go, app/view.go and MyGo public lifecycle/timer/update
contracts. Determine whether pending drafts can remain unsaved indefinitely
until re-show/close. Distinguish an established bug from unknown Windows paint
behavior; don't infer runtime facts from source alone. If confirmed, propose
the smallest application-owned one-shot persistence scheduling mechanism using
public Window.Update/lifetime-safe callbacks, no polling, no vendor change and
no new allocation/cache architecture. It must preserve debounce, modal/restore/
shutdown barriers, failure suppression and draft Flush semantics. Identify
existing meaningful test seams instead of requiring physical-input automation.

This review does not authorize code changes or broaden .23 delivery. A short
source-grounded finding (or no actionable finding) is sufficient. Stop after
publishing the handoff and notify root.
