# Active tab visibility after keyboard navigation

ACTIVE as second increment after focused keyboard cases pass in the combined
tab-workflow-stage-plan.md. Same DeepSeek app owner, no concurrent writer. No
children; GLM/DeepSeek compact before200K. Root plans/reviews only.

Implement the next small usability increment: when the active document changes
to an offscreen tab, reveal its existing tab button once with MyGo's public
Element.ScrollIntoView. Default zero options provide minimal instantaneous
scrolling. No animation, MRU state, introspection, custom coordinate scrolling
or dependency. Retain no widget handles across frames. Ordinary redraws must
not continuously pull the user's manually scrolled tab bar back.

Root read public docs in vendor/.../ui/api_gen.go:7252 and ui/scroll.go:94.
ScrollIntoView is queued after layout, must be called in the frame a target is
chosen, and repeated calls every frame prevent the user scrolling it away.
Current app.tabs creates a local button, so the handle is already available
for a one-frame request. Use a stable active-tab identity (not only list index)
and handle zero tabs / close / restore / open changes. No document retention
through the visibility marker: prefer an ID, never a retained *Tab pointer.

Coordinate with keyboard owner; do not overlap app.go/view.go writes. Scope
app tab bar and focused synthetic Tester cases. Verify far-end to first/last
wrap, app add/select, close neighbor, same-frame click ordering, failed Flush,
and manual scroll after reveal. Use supported public tester observation only.
One minimum-window capture can document actual tab bar geometry if needed;
Tester capture is not a physical OS input claim. Avoid broad suite repetition;
one final affected app test/vet after corrections. All Go through wrapper and
retain commands/logs/exits. No package/build/host launch.

Publish tab-active-reveal-result.md, final small source/test identities and
evidence. Update only the small README usage row after docs ownership release;
don't change the current package launch path before later delivery acceptance.
