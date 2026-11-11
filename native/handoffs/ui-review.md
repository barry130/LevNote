# Native UI acceptance plan

Root uses frontend-design guidance for hierarchy/typography/visual critique;
the user's explicit simple low-overhead native UI requirement takes priority
over stylistic novelty. The product and audience are known: local multi-format
reading/editing in one lightweight workspace. No confirmation is needed.

## Intended visual system

- Light base: background #FFFFFF, panel #F5F6F7, text #202124, muted #626A73,
  border #DADDE1, action/focus #33766C. Validate actual toolkit colors/contrast
  before replacing working theme values just for cosmetic uniformity.
- Dark base follows toolkit DarkTheme; source syntax needs a brighter palette
  than light source syntax. Warnings and selection remain readable in both.
- System Microsoft YaHei UI for Chinese UI and body; system monospace for
  source. Avoid bundled decorative fonts and additional font loading cost.
- UI/body 13–16 px, source user preference 10–30 px; small metadata must remain
  legible at normal scale. Reading width stays comfortable; source can scroll.
- Persistent layout: compact top commands, workspace/tabs, one main document,
  quiet status line. Optional settings/plugins/search/outline take space only
  while used. Graph properties collapse or reflow at narrow width.

```
commands | new open save             find plugins settings
---------+----------------------------------------------
files    | tabs
         | format / source / split / export
         | document                             optional
         |                                      inspector
---------+----------------------------------------------
status / warnings / cursor / encoding / save state
```

## Review against brief

The existing native shell already fits this restrained structure. Preserve it
and fix concrete clipping/contrast/navigation issues. No landing-page hero,
card grid, animated backgrounds, gratuitous shadows or extra ornaments.
Warnings should state user consequence and next action; implementation details
such as UTF16 are only useful for an explicitly bounded source segment.

## Screenshot review gates

- 1180x780 and supported minimum 760x480; at minimum width graph toolbar,
  inspector, source/preview mode controls, find/encoding rows cannot overlap.
- Every screenshot must be generated from current source, not stale binary.
- Source light/dark, Markdown preview/split, mindmap, drawio, plugin manager;
  inspect text flow, caret, selected state, native CJK and emoji fallback.
- Invalid/unsupported formats and readonly documents remain actionable.
- Show full warning details via an accessible action when a summary hides
  additional issues; repeated identical warnings should not dominate layout.
- Accessibility labels and keyboard focus remain meaningful. Keyboard-only
  save/open/find/tab navigation and Escape dismissal should be testable.

Root will append concrete results after receiving integration screenshots.

## Reviewed component evidence

- editor-wrap-ime.png: Chinese, emoji and combining text are legible; caret,
  composition underline, line numbers and wraps align; no obvious overlap.
- graph-draft-conflict.png (960x640): document canvas and pending field draft
  both remain visible; recovery controls fit the inspector. Full app narrow
  composition and dark theme are still pending.

## Root review of 11:20 integrated screenshots

- source-dark.png: active tab now has legible light text and native source
  palette is clear. Minimum-window settings and Markdown split render without
  overlapping helper text. Plugin panel keeps the document tab visible and
  uses its own scrolling area. Encoding fallback correctly says unknown
  original encoding and explains the explicit reopening action.
- drawio-760x480.png: page title is squeezed into vertical single characters;
  zoom-in is clipped on the right. mindmap-760x480.png also clips zoom-in.
  Astra graph owner should reflow toolbar groups and preserve a single-line
  page title. Document area at this size is only 532px; do not shrink all text.
- Encoding chooser label is cramped in one row; DeepSeek may place its short
  label above buttons. This is a usability refinement after correctness.

## Root review of 11:29 graph and Markdown component fixes

- drawio-narrow.png and mindmap-narrow.png at 532px document width now wrap
  controls into meaningful rows; plus/minus/fit controls remain visible.
  Long drawio page name is a single ellipsized line. Root accepted this
  correction, pending refreshed whole-window screenshots.
- Markdown conflict at 265px keeps the pending Chinese/emoji input and
  conflict explanation in view. Completion/recover/discard controls reflow
  vertically without clipping and the pane remains scrollable.
