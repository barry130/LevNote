# Growth composition refusal must survive host Flush

2026-11-10 root reviewed document-growth-draft-review.md and actual editor/app
code. Concrete gap accepted for this increment: a rejected update during IME
composition currently reports an error, but Flush unconditionally UnmarkText
and returns nil, sealing an earlier partial preedit and authorizing Save/
Select/Close. This violates the assigned refused-composition host-action gate.

Astra remains sole implementation owner. Add the smallest explicit pending
composition-refusal state. Keep only bounded error/owner metadata, never copy
the rejected potentially huge string. On accepted marked update or accepted
final replacement, clear it. Explicit Escape/native cancellation through a
proven public route may clear/cancel it. Do not assume toolkit UnmarkText means
explicit cancel: it also arrives on focus loss and is called by our own Flush,
ReleaseView, navigation, delete/copy/undo paths.

Flush must return the stable size refusal while unresolved and must not close
the owned group. First rejected preedit has no group but still needs retry or
explicit cancel before lifecycle success. Existing accepted preedit remains
intact. Keep foreign groups owned by caller. Prevent pointer/navigation or
other incidental UnmarkText callers from silently clearing the refusal or
corrupting its mark/selection; choose a small consistent guard rather than
retaining rejected text or modifying toolkit vendor. Explicit cancel must be
reachable with first-refusal(no active mark) as well as ongoing composition.
Normal oversize paste refusal is not an IME pending draft; do not block save
forever on every ordinary failed edit. Surface clear retry/Escape guidance.

Add actual tiny-limit source App/Tester first+ongoing composition tests:
refused update -> Flush/Save/Select/Close refuse with same tab/view/buffer,
disk/selection/version/history/group/accepted mark; incidental focus/geometry
release cannot erase gate; valid retry or explicit cancel permits progress.
Test legitimate composition success and cancellation, not synthetic races.
Follow existing host identity contracts if buffer pointer is replaced.
Report precise UI behavior and caller limitations. No physical OS IME claim.

Markdown/graph host-writeback refusal paths already preserve draft on static
review. Still add the assigned real-seam regression. Table structural rollback
currently may advance revision on later failure; no need broaden that existing
documented rule unless an actual draft-loss trigger is found. Do not claim all
format commands have unchanged revision merely because size is checked per edit.
