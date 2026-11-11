# Format routing design checkpoint: schema3 fixed native routes

2026-11-09 · design only · no implementation or execution

This checkpoint refines the recommendation in
`native/handoffs/plugin-next-slice-review.md` against the current sources. It
does not change the frozen `.7` delivery inputs or Astra's ownership.

## Decision

Schema3 is the smallest strict extension of the existing declarative manifest:
keep the schema1 source-highlight fields, add exactly one required
`nativeRoute`, and do not add a capability-bit list. The first closed route set
is only:

| `nativeRoute` | Compiled adapter | Export | Readonly scene |
|---|---|---|---|
| `builtin.markdown` | `markdown` | `levnote.markdown-export` (`html`) | none |
| `builtin.mindmap` | `mindmap` | `levnote.mindmap-svg` (`svg`) | `levnote.mindmap-preview` (`mindmap-scene-v1`) |

`builtin.drawio` is intentionally out of this first slice. A schema3 manifest
that names it is rejected during admission. Future route values require a new
root-approved schema or closed-set change; they are never installed dynamically.

### Manifest examples

```json
{
  "schema": 3,
  "id": "task-notes",
  "name": "Task Notes",
  "version": "1.0.0",
  "apiVersion": 1,
  "extensions": [".taskmd"],
  "nativeRoute": "builtin.markdown",
  "lineComment": "#"
}
```

```json
{
  "schema": 3,
  "id": "thoughts",
  "name": "Thoughts",
  "version": "1.0.0",
  "apiVersion": 1,
  "extensions": [".thoughts"],
  "nativeRoute": "builtin.mindmap"
}
```

## Strict admission

`core/plugins` remains the data-only parser and validator. Extend the schema
DTO with `NativeRoute string`; `modules` remains the sole composition root that
maps that route to a compiled adapter ID. The app resolves that ID through its
existing adapter list and never treats the manifest ID as an adapter ID.

Schema3 admission uses a route-specific allowed-field set:

- Required with existing schema1 semantics: `schema`, `id`, `name`,
  `extensions`; existing defaults for omitted `version` (`1.0.0`) and
  `apiVersion` (`1`) remain unchanged.
- Newly required: non-null `nativeRoute`, exactly one of the two closed values
  above.
- Optional: `lineComment`, `blockStart`, `blockEnd`, `keywords`.
- Prohibited: `outline`, arbitrary paths, commands, URLs, scripts, DLLs,
  executables, unknown fields, wrong casing, duplicate keys, and null values
  for present fields.

The existing manifest bounds and file rules remain: valid UTF-8, 64 KiB
manifest cap, regular files only, symlinks rejected, extension normalization
and bounds, ID/name/version validation, 256 installed manifests, and bounded
install-directory reads. Before decoding schema3, a strict top-level object
reader must reject duplicate keys; Go's ordinary map decode cannot detect them.

The schema1 defaults are defaults only when fields are absent. Explicit empty
or null `version`, `apiVersion: 0`, or a wrong type must fail. A bounded,
ordered top-level token pass should discover every `schema` and `nativeRoute`
key before choosing a validator; duplicate `schema` keys such as `3` followed by
`2` must fail rather than let one value hide the schema3 declaration. The
legacy schema1/2 path keeps its current decoding behavior for existing
manifests. Schema3 strings must preserve invalid surrogate-escape rejection;
they must not be silently replaced with replacement characters. This is a
narrow duplicate-key discovery pass, not a generic arbitrary-JSON framework.

The fixed capabilities are not copied into the manifest. They are derived from
the selected adapter at composition time. Outline is prohibited because the
native adapter's semantic outline is authoritative; schema3 never combines it
with a schema2 source outline.

## Registry and composition seams

The following are the minimal implementation seams:

1. `core/plugins/declarative.go`
   - Add schema3 to strict field admission and validate `nativeRoute`.
   - Preserve schema1/2 parsing and all existing bounds unchanged.
2. `core/plugins/registry.go`
   - Preserve builtin-before-installed order and first-match extension
     semantics from `Registry.Match`.
   - Keep schema3 manifest IDs distinct from adapter IDs; `Syntax(manifestID)`
     exposes the route, while `Match(path)` still returns the manifest.
3. `modules/modules.go`
   - Add one fixed composition mapping from `builtin.markdown` to adapter ID
     `markdown`, and `builtin.mindmap` to adapter ID `mindmap`.
   - Keep `InstalledSourceOutlineFactory` schema2-only.
4. `app/plugins.go`
   - For schema3, resolve the route through `modules`, then `adapterByID`.
     No new bypass mechanism is needed: current `configureTab` already uses
     `Registry.Match` and then `adapterByID`; schema3 only removes the
     `match.Manifest.Builtin` condition for an installed native-route match.
   - Apply schema1-style source highlighting in source/split mode.
   - Set the target adapter's `Outline` and do not create
     `t.OutlineFactory`.
   - Fall back to editable source with an explicit warning if the route or
     compiled capability is unexpectedly missing; never parse as another
     format.

Custom suffixes must not depend on compiled adapters' hard-coded
`Adapter.Matches` functions, but the current application route path does not
call those functions. It uses `Registry.Match`, then `adapterByID`. The actual
schema3 change is that an installed schema3 manifest can select the same
compiled adapter as a builtin match; `Adapter.Matches` remains descriptive for
other consumers and needs no custom-suffix edits.

Capability ownership has one authority. `core/plugins` owns only the closed
route-key constants and strict admission. `modules` maps those keys to freshly
returned adapters and is therefore the sole feature authority. The app consumes
the neutral adapter and must not keep a second executable/feature catalog or
mutable capability bits. Existing `Schema >= 2` badge and notice branches must
gain a schema3-specific branch so a routed format is never labeled schema2.

## Product behavior

For a successful route:

- The tab enters the target adapter's normal preview mode and keeps its
  existing source/split controls.
- The source view uses the manifest's schema1-style highlight fields. A
  `.taskmd` example can therefore highlight `#` comments; a `.thoughts` source
  view can omit those optional fields.
- The target adapter's semantic outline and outline IDs are authoritative.
  Markdown heading IDs and Mindmap node IDs remain adapter-native; they are
  not prefixed or replaced by `task-notes`/`thoughts`.
- Export and readonly scene eligibility come only from the selected adapter.
  A routed Mindmap gets the existing SVG export and readonly scene path; a
  routed Markdown gets HTML export and no scene.
- The plugin panel shows both identities, for example:
  `Task Notes · task-notes · 原生路由 builtin.markdown → Markdown`.
  The rich-view header should say `原生路由 · Markdown`, not claim a new
  engine.
- The plugin panel can add one small discoverable action on an enabled
  native-route row, such as `新建文档`. It resolves the current manifest,
  uses its first normalized extension as the new document suffix, and takes
  the content from the fixed target adapter's `Template`; the manifest has no
  template field or custom template authority. The existing shared `add`,
  Flush, and draft barriers remain unchanged.

Disabling the schema3 manifest only stops that custom suffix from matching; it
does not remove the fixed capability from the composition root. Disabling a
builtin registration may release `.md` or `.mindmap` matching, but it must not
destroy the fixed route catalog. Re-enabling the route manifest resumes
matching after the existing flush/reconfigure lifecycle.

## Lifecycle and path-sensitive rules

Existing lifecycle boundaries remain mandatory:

- Flush all views before install, enable/disable, and refresh
  (`native/app/plugins.go:200-284`).
- A failed Flush leaves the current view, unsynchronized draft, undo history,
  selection, and persisted selection unchanged.
- A successful Flush may write a pending draft before a later installation or
  settings operation fails. The implementation must not promise absolute zero
  buffer mutation across that whole action: persistence failure keeps the
  selected routing/views and saved settings, and retains the now-written edit.
- A failed install or persistence operation must roll back atomically and leave
  the existing registry selection unchanged, as `InstallFile` and
  `rollbackInstall` already require (`native/core/plugins/registry.go:331-406`).
- Reconfigure releases the outline and presentation, increments the outline
  factory generation, cancels the tab's export, and cancels its readonly scene
  (`native/app/plugins.go:42-68,265-280`;
  `native/app/export_worker.go:290-297`; `native/app/app.go:772-796`).
- The same manifest ID after refresh is a new generation. Captured outline,
  scene, and export contexts must fail stale rather than accept an old
  controller.
- Existing `Refresh` is not transactional: it can recover usable items while
  reporting load issues, then reconfigure tabs. Schema3 must preserve that
  behavior rather than invent a stronger refresh guarantee.

Concrete path-based breaks that the implementation must cover:

- `App.NewDocument(kind)` only recognizes fixed kinds `markdown`, `mindmap`,
  and `drawio` (`native/app/app.go:381-391`). Add a route-aware creation seam
  that can create `.taskmd`/`.thoughts` from the target adapter template;
  do not add a manifest template field.
- `App.Save`/`SaveAs` reconfigures after a path change
  (`native/app/app.go:644-704`). Saving a routed document as another custom
  route suffix must remount that route; saving as `.md` must match builtin
  Markdown by the new path. Rematching never converts format: a Mindmap saved
  as `.md` remains its JSON source and Markdown opens that source as Markdown.
  Route-aware tests should use same-format aliases except for one deliberate
  cross-format path test with truthful wording. Readonly Save As remains
  writable through the existing behavior.
- Recovery restores by title/path and then calls `configureTab`
  (`native/app/app.go:393-422`; `native/app/state.go:13-49`). A recovered
  `.taskmd`/`.thoughts` draft must remount by its new path, without treating
  the persisted manifest ID as an adapter ID.
- `formats/code.LanguageForPath` is extension-sensitive and only knows fixed
  suffixes such as `.mindmap`/`.drawio` (`native/formats/code/lexer.go:10-25`).
  Schema3 must keep declarative highlighting and must not depend on adding
  `.taskmd` or `.thoughts` to that table.
- App-local references are already generic path-based and open through the
  registry; no new URL handling is needed. One alias-to-alias local link test
  is required to prove custom-suffix references work through the existing path.
- The native outline seam chooses `t.Adapter.Outline` before
  `t.OutlineFactory` (`native/app/outline.go:47-61`). Schema3 must leave the
  factory nil so there is no incompatible mixed outline.

## Finite test plan

These tests are the implementation gate; no broader redesign is implied:

1. Strict schema3 admission: valid examples, closed-route rejection, unknown
   fields, wrong casing, duplicate keys, nulls, `outline`, oversized/invalid
   extension and ID cases, bounded-read limits, and schema1/2 unchanged
   behavior.
2. Install and rollback: success registers the manifest and syntax; duplicate
   ID/file failures and simulated post-publish load failures remove the file
   and restore the previous selection.
3. Real routing: `.taskmd` mounts Markdown and `.thoughts` mounts Mindmap,
   including semantic outline, source/split, export eligibility, and Mindmap
   readonly scene eligibility.
4. Disabled/fallback behavior: disabling the schema3 manifest stops only its
   suffix; disabling builtin Markdown/Mindmap does not remove the fixed route
   mapping; missing compiled capability falls back to source with a visible
   warning.
5. Round trip: BOM, CRLF, unknown native metadata, save, undo, and readonly
   transitions preserve document bytes and existing format behavior.
6. Flush and generation: a failed flush retains draft/undo/selection; refresh
   with the same manifest ID rejects stale outline/scene/export contexts.
7. Path transitions: Save As to another routed suffix, Save As to `.md`,
   reopen, recovery, and new-file creation all re-match by path and use the
   target adapter template.
8. Worker safety: route reconfiguration cancels stale readonly scene and
   export jobs; no worker is launched by this design-only checkpoint.

## Future implementation split

No implementation is active in this checkpoint. When root accepts the `.7
package/docs stage and explicitly activates the next slice:

1. GLM owns only `core/plugins` schema/loader/tests and the `modules`
   fixed-route composition/tests. It first declares the fixed exported route
   types/API, then publishes the core contract before app edits.
2. Astra owns app wiring, plugin-panel UI, and app tests/docs only after that
   core contract checkpoint. There is no same-file overlap.
3. Core can be tested while app source is being prepared; the app remains a
   pure consumer of the neutral composition result.

## Remaining honest gap

Schema3 is an installable route to shipped native capabilities, not an
independently installed document engine. It does not provide plugin package
identity/version negotiation, a two-way document protocol, plugin-owned
processes, crash isolation, or independently installable editable engines.
Those remain a separate future contract.
