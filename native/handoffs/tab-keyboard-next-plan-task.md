# Multi-document keyboard navigation: next usability slice

ACTIVE READ-ONLY2026-11-10 for GLM as native/AGENTS.md. No children, compact
before200K; no code/docs edits, tests/build/launch/measurement/network. Publish
short `tab-keyboard-next-plan.md` only. Source-retention independent review may
interrupt this plan when ready. No changes entering .21 package.

User wants one simple low-overhead app for many files and continued improvements
inspired by mature editors. Plan the smallest useful missing workflow: Ctrl+Tab
next and Ctrl+Shift+Tab previous in visual tab order with wrap, no MRU switcher,
popup, animation, new dependency or persistent history. Root read app.shortcuts
and tabs/Select: only click switching exists, active input/drafts require Flush.

Read actual shortcut APIs/event order, Select lifecycle/flush/recovery reentry,
source/rich IME state and Find/GoToLine/modal rules. Identify appropriate guards
and whether simply calling Select is sufficient under synchronous recovery or
tab-array replacement. Zero/one tabs should no-op without unwanted flush/focus;
failed Flush must preserve active tab and draft; successful switch should use
existing cancellation/release/persist path. Input methods must not be disturbed
by stale focus/callbacks. Need not redesign generic Select unless concrete gap.

Explain active offscreen tab visibility and supported MyGo public scroll API:
if existing tabs don't auto-reveal, choose a small public-API path or explicitly
split that UX issue into later work, no widget-tree introspection. Do not infer
tested OS shortcut behavior from Tester. Identify substantive app/Tester cases
and concise shortcut help update. No web/competitor-copy research needed for
these conventional editor bindings. Keep plan implementation-ready and bounded.
