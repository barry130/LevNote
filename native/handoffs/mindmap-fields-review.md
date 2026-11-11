# Mindmap fields independent review (2026-11-08)

Reviewer: GLM (read-only)

Current-source baseline inspected:

- `native/modules/graphview/fields.go` (18:00)
- `native/modules/graphview/draft.go` (18:00)
- `native/modules/graphview/field_link.go` (18:03)
- `native/modules/graphview/adapter.go` (18:03)
- `native/modules/graphview/view_ui.go` (18:01)
- supporting `edit.go`, `input.go`, `sync.go`, and `preview_limit.go`

## Result

No actionable current-code defect found in the requested field draft, native
activation, optional presence, Flush, readonly, Release, or long-note paths.

## Verified behavior

- The private field discriminator identifies topic, note, hyperlink, and the
  drawio label separately; selectors cannot infer an authoritative field from
  a label alone.
- `fieldDraft` captures Buffer pointer, node/page identity, field kind, original
  presence/value, source text, input, and input version. `pendingDraft()` uses
  the input version to avoid repeatedly materializing an unchanged long field.
- Flush parses the current bounded host source, checks node plus exact
  presence/value, then applies only the captured field. Unrelated edits can
  merge; same-field changes, deletion, Buffer replacement, foreign groups,
  readonly, and oversize inputs reject atomically while retaining the draft.
- Field switching calls Flush first and keeps the old draft/field/focus when
  Flush fails. Clean refresh yields to authoritative source.
- `Release()` releases heavy parsed/render caches but retains a pending draft
  and its independent input. A synchronous `Host.OpenReference` callback may
  therefore run against a released view without derefererring stale graph
  caches. Recovery writes a separate plain-text document only, identifies
  node/field/original presence, and appends exact draft text.
- Readonly permits selection, reading, copying, and clean link navigation, but
  rejects source mutation and retains a failed draft for correction/recovery.
- The native pointer/key gesture captures Buffer, node, optional state, and the
  exact draft/input/commit identity. Missing, canceled, outside, stale-node,
  changed-source, changed-input, and replaced-Buffer gestures do not fall back
  to a current target. The action calls `Host.OpenReference` at most once and
  returns immediately.
- The explicit pending-link action verifies the just-committed authoritative
  value against the captured draft. A longer draft/input interaction that
  changes the input version before activation is refused; the
  `commits+1` check covers only the one authorized field transaction from the
  captured action, without accepting unrelated later commits.
- Stored destinations above 4096 bytes are visibly unavailable and are not
  shortened into an `href`. Tooltip/action metadata is bounded at 512 bytes and
  rune-safely truncated.
- Drawio label drafting keeps the plain/source distinction: the candidate is
  checked against rich source text, `UpdateText` patches the XML label, and
  after commit the candidate's reparsed rich `Text` becomes the new source base.
  Plain text remains the editor baseline.
- The malformed doc-block parser correction is outside this review; its focused
  evidence is in `native/handoffs/docblock-safety.md`.

## Scope

No product, test, or documentation file owned by Astra was modified. No peer
package tests were run.
