# Next reader increment: keyboard image navigation

ACTIVE read-only planning 2026-11-10 for Astra after frozen app acceptance.
No child delegation, source/docs edits, tests/builds/captures/launches. Root is
finishing .23 delivery; do not reopen its inputs. Write only
native-image-keyboard-next-plan.md, concise enough for direct implementation.

The native image reader has pointer zoom/pan and reachable toolbar controls,
but imageview/input.go has no direct key handling. Plan a small useful keyboard
increment matching familiar desktop-reader interactions while preserving the
app's text/navigation shortcuts. No claimed live competitor research. Inspect
actual MyGo public input/focus APIs, imageview viewport/finite/clamp helpers and
app shortcuts. Identify exact focus rule and keys for +/- zoom, 100%, fit, and
bounded arrow pan; avoid duplicate/global shortcuts or consuming commands in
dialogs, text fields, inactive image/byte modes. Reuse existing geometry; no new
timer, animation, image cache, framework or dependency. Composition/text state
must be unaffected. Explain pointer-vs-keyboard zoom anchor and focus-return
behavior using actual public controls.

Propose exact small file ownership, optional visible key hints only if useful,
and a few substantive Tester events proving focus containment, zoom limits,
center/pan clamp and no action after Release/byte switch. No mirror unit tests,
broad screenshots/performance benchmark or package stage in this plan. Root
will choose/activate after .23 acceptance. If an actual public API constraint
makes keys unreliable, report it with source evidence instead of inventing an
introspection/vendor workaround. Current .23 inputs stay frozen throughout.
