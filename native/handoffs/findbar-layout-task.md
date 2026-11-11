# Small next step: native Find panel at minimum window size

2026-11-09. Activated after search-navigation-root-acceptance.md for implementer
DeepSeek read_native_handoffs_release10_ui_feasibility_task_md. No child agents.

Root source review: app/state.go findbar has fixed260px fields, previous/next,
checkboxes, close and replacement/workspace actions in unwrapped rows. At760px
window with228px sidebar, main width532px. This may overflow. Confirm using
existing MyGo tester geometry first. If confirmed, make minimal native reflow
in findbar only, reuse existing controls, compact labels/full accessible names.
No new dependency, animation, shortcuts, search semantics/cache modifications,
custom numeric/text engine, or general responsive framework.

Controls must stay within main pane above fixed status row at760x480 in both
themes and default font; also check largest existing font setting geometrically.
Wrapping input/options/action rows is permitted, and fields may flex/shrink.
Keep useful editor area where possible, keyboard Tab access, query/replacement
text and focus across redraws; no new autofocus-each-frame. Read actual toolkit
layout semantics before choosing widths. Stable Context keys before TextInput
construction if needed; don't reset field composition during resize/reflow.

Use meaningful tester interaction/geometry (previous/next, options, replace
current, close), and two760x480 light/dark captures inspected by you. If visual
layout already fits, publish exact geometry and do not change source. No
physical OSIME/helper/resource claims. Preserve all existing captures/evidence;
use new findbar-layout-evidence directory.

Scope app/state.go findbar and new focused layout tests. Existing search feature
accepted separately. Run focused layout+search navigation tests then one final
app suite/vet only if source changed. No worker suites/build/docs until root
accepts actual layout. Publish findbar-layout-result.md, actual failure/check
logs, small changed-file list and screenshots; freeze for root review.

Once accepted, package combined search corrections/layout as .13 under existing
search-navigation-release13-plan.md. Do not preempt that delivery authorization.
