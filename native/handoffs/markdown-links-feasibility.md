# Link feasibility — historical source inspection and later probe

This source inspection preceded activation of the link stage at15:50. The
stage is now active; actual public-input probe and implementation results are
recorded in [markdown-links-view.md](markdown-links-view.md). The findings
below describe the pinned MyGo v0.8.3 source and are retained for provenance.

- `ui/richtext.go:148` supports Text/RichText children in one shaped paragraph.
  `ui/inline.go:281` maps each inline child to glyph selection rectangles,
  including multiple wrapped fragments; hit testing uses those fragments.
  This is more accurate than using one bounding rectangle for a wrapped link.
- `ui/widgets.go:190` native Link calls `host.openURL` for a non-router URL.
  The product should therefore use styled Text with RoleLink and its own host
  callback, not ui.Link with an authored href.
- A merely `.Focusable().Clicked()` child does **not** preserve ordinary
  paragraph selection: `ui/input.go:412` chooses the innermost interactive
  pointer target; `ui/textselection.go:36` skips creation of a selectable editor
  for inline children and gives them textScope0. `pressTextSelection` does not
  climb from that child to its parent. `.Selectable()` on an inline child is
  explicitly a no-op (`ui/textinput.go:330`). Styling alone is insufficient.
- A child `HandleInput` also replaces the parent's handler for that event:
  `ui/handler.go:216` chooses only the innermost handler. Returning false means
  normal native routing, not invoking the parent handler next. The checkbox
  deliberately uses this to block parent double-click editing; links need the
  opposite behavior for ordinary pointer input.
- Public `Element.HitTest` can reject the child's pointer target while retaining
  keyboard focus/accessibility. A promising finite seam is a modifier-sensitive
  hit test: plain pointer falls through to the selectable paragraph/row, while
  Ctrl/Cmd pointer reaches the inline link; Tab+Enter remains on its focusable
  RoleLink. `Context.Modifiers()` reads current runtime modifiers. This needs
  an actual native probe after stage activation: plain drag starting inside the
  link, plain double click into block draft, Ctrl-click, Tab/Enter, wrapped
  fragments and modifier changes while held. Do not claim it works from source
  inspection alone, or broaden vendor code before proving a needed defect.

The existing bounded32KiB rich row rule and virtual list remain the outer
resource guard. A link's numeric source identity must bind buffer/version plus
whole block/list-item and exact link ranges; copied href/label metadata needs a
finite budget. No new shared contract is required merely to style links; Host
OpenReference owns destination policy and filesystem routing. Exact activation,
stale target and metadata-budget semantics still belong to the next approved
contract.
