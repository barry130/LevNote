# Markdown inline-link activation design

Status: **read-only candidate analysis; not active and not approved**.

This note records a possible stage after the active Markdown task-checkbox work.
No product files were changed for it. The current app owns real-adapter
acceptance, while `modules/markdownview/**` and `formats/markdown/**` remain
with Astra and GLM respectively.

## Current observable gap

- `formats/markdown` already parses `[label](target)` into
  `InlineSpan{Kind: InlineLink, Range, Text, Href}`. The range and href are
  source-model values.
- `modules/markdownview/blocks.go` currently styles an `InlineLink` as accented
  underlined text inside a single `ui.RichText` paragraph. It has no pointer
  hit target, focus target, accessibility node, or activation callback.
- `editor.Host.OpenReference(string)` already enters the app's local-reference
  route. `app/reference.go` distinguishes `#anchor`, local filesystem targets,
  and rejected URLs/paths. Anchor activation resolves through the Markdown
  outline provider; local files use the ordered tab-open path.
- Therefore the data and host policy exist, but preview/split links cannot be
  activated. Source mode correctly shows the literal Markdown syntax and does
  not need a second embedded-link control.

## MyGo shape to use

MyGo `ui.Span` is presentation-only and has no callback. A plain
`ui.RichText(c, spans...)` cannot make selected spans clickable or keyboard
focused.

`ui.RichText(c).Children(...)` accepts inline text elements and preserves their
interaction and accessibility. For a link segment, a child shaped as follows is
the narrow native primitive:

```go
ui.Text(c, boundedLabel).
    TextColor(c.Theme().Accent).
    Underline().
    Cursor(ui.CursorPointer).
    Focusable().
    Role(ui.RoleLink).
    Label(accessibleLabel).
    Tooltip(detail).
    OnClick(func() { activate(capturedHref) })
```

MyGo routes pointer activation through `OnClick`; a focused `RoleLink` accepts
Enter and deliberately does not consume Space. The inline child remains part of
the paragraph's layout and wraps with neighboring text. Its accessibility node
uses the link's actual fragment bounds.

Do **not** use `ui.Link(c, label, url)`: it calls the host's browser URL opener.
The desktop policy is local-only, and `Host.OpenReference` must remain the sole
activation boundary.

The current view builds one flat span slice. The candidate implementation would
split each parsed inline run into ordinary inline text children and modeled
link children, preserving bold/italic/code styling. It must keep text order,
absolute source ranges, and paragraph wrapping intact. This is a
`modules/markdownview/**` change and requires Astra ownership and approval.

## Activation safety

At render time, retain only numeric source identity plus the full parsed `Href`
for the link fragment. Do not identify a target by label, visible row index, or
a newly rendered DOM-like structure. A separate display-only bounded label may
not be reused as the activation target.

On pointer click or Enter:

1. Capture the buffer pointer, buffer version, exact inline source range, and
   full href when the control was built. `editor.Host` has no tab ID or
   controller field, so the view must not pretend to capture one.
2. Refuse if the app is no longer showing the source tab. The existing
   `OpenReference` callback is already an app-owned closure carrying the source
   tab, so the app route can perform that liveness check without expanding the
   host contract.
3. Complete pending composition and any block draft with the existing fallible
   `Flush`. A failed Flush keeps the draft, source, caret, and current view
   alive and reports the failure.
4. If unrelated edits above the link change the source version during Flush,
   map the captured inline range and its enclosing block/list range through the
   edit journal, then reparse the bounded current source and require the exact
   mapped inline-link identity and full href. Restore, expired history, overlap
   with the link/target, an enclosing-context change, or no exact current match
   must report a stale control. Do not use an unconditional version mismatch
   where a bounded mapped/reparse check is possible.
5. Call `Host.OpenReference(capturedHref)` once.

Because the host is an `editor` interface, the view does not import
`app/reference.go`. The app route owns classification and reports:

- `#anchor` uses the current Markdown anchor resolver after Flush;
- a local file/path is handed to the one ordered open queue;
- an unsaved document's relative path is refused with the existing actionable
  reason;
- `http`, `https`, protocol-relative URLs, `mailto`, `data`, `javascript`,
  UNC/remote file targets, and unsupported schemes are visibly refused;
- a closed/stale source tab is visibly refused rather than acting on a
  different active tab.

Pointer modifier policy remains undecided: primary click and Cmd/Ctrl+click are
both candidates. Keyboard Enter is the accessibility path either way. Existing
selection/caret behavior stays separate:

- preview link activation must not focus the hidden source editor;
- split anchor navigation intentionally follows the existing outline behavior
  and reveals both source and mounted preview positions;
- ordinary local-file opening uses the ordinary lifecycle barrier and may
  switch tabs;
- anchor resolution must use `ResolveOutlineAnchor`, not match a generated
  heading suffix with app-side string logic.

## Bounded presentation

- Keep the visible link label and accessible label bounded with the same
  `MaxOutlineLabelBytes` strategy or a shared smaller helper. Use `…` when
  clipped, and expose the full bounded detail through a tooltip where useful.
- The full authored source remains available in source mode. Do not retain
  another unbounded label copy solely for tooltip text.
- Never clip an href into a different destination. Use the full authoritative
  parsed target for activation. An excessive target must be visibly refused
  under an owner-agreed byte bound rather than shortened.
- External/unsupported links never launch the operating system browser or a
  shell. Reporting the refusal through `Host.OpenReference` makes the behavior
  discoverable instead of rendering a deceptive active link.
- Empty, malformed, or unsafe hrefs remain represented exactly as parsed; the
  host route supplies the visible rejection. No parser rewrite is needed.

## Minimal end-to-end acceptance when activated

Use one compact real registered Markdown adapter fixture containing:

- a relative local-file link,
- a percent-encoded same-document anchor,
- an `https` link that must be refused without a browser launch,
- bold/Unicode text and a wrapping label around links,
- a pending composition in another block before activation.

Assertions:

1. Preview exposes actual `RoleLink` accessibility nodes with positive inline
   bounds and Enter activation; source mode renders no checkbox-like embedded
   link control and leaves the literal Markdown unchanged.
2. Pointer click and focus-plus-Enter on the local link both complete pending
   Flush first, open/select the local target in order, and never use a browser
   URL callback.
3. The anchor link resolves through the Markdown outline source and follows the
   existing preview/split navigation policy, including split source-plus-preview
   reveal.
4. A rejecting URL leaves tabs, source bytes, dirty state, undo history, draft,
   and caret unchanged, while placing a clear Chinese refusal in app status.
5. Failed Flush retains the current tab/view/draft and reports the failure
   before classification or navigation.
6. An unrelated insertion above the link maps and reparses successfully; a
   replaced link or enclosing context rejects the stale captured control.
7. Save/reopen changes no authored bytes.

This remains a candidate. No action should begin until the task-checkbox stage
is reviewed, current owners agree the inline-child renderer change and target
bound, and root activates a link-stage task. Any host liveness contract would
need a separate minimal proposal; none is assumed here.
