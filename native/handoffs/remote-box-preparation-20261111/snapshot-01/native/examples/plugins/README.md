# Declarative syntax plugin example

The schema1 `toml.json` sample is a complete data-only plugin accepted by
**插件 → 安装**.
It adds TOML keyword, string, number and comment highlighting. It cannot run
code, access files, or perform network requests.

That schema1 sample uses the bounded JSON manifest contract:

- `schema` and `apiVersion` must be `1`.
- `id` uses lowercase letters, digits and single hyphens.
- `extensions` is a non-empty list such as `[".toml"]`.
- `lineComment`, `blockStart` / `blockEnd`, and `keywords` are optional
  highlighting metadata.
- Unknown fields are rejected so typos cannot silently disable behavior.

The application copies the selected manifest without overwriting an existing
file or plugin ID.

## Schema2 source outlines

Schema2 keeps every schema1 field and adds an optional `outline` array of up to
eight rules. Each rule has a required `prefix`, optional `suffix`, and required
integer `level` from1 through6:

```json
{
  "schema": 2,
  "id": "ini-outline",
  "name": "INI outline",
  "version": "1.0.0",
  "apiVersion": 1,
  "extensions": [".ini"],
  "outline": [
    { "prefix": "[", "suffix": "]", "level": 1 },
    { "prefix": "# ", "suffix": "", "level": 2 }
  ]
}
```

The outline is source-only: it reveals the matched source line and never creates
a rich preview, split view, or export. A missing `outline` or an empty array
means the plugin has no outline capability. Marker bytes must be valid UTF-8;
a literal U+FFFD character is allowed, but malformed UTF-8 and ill-formed
surrogate escapes are rejected.

Rules are literal line rules. Optional leading spaces/tabs are ignored, exact
markers must match, and the first declared matching rule owns the line. Labels
are trimmed only at horizontal ASCII boundaries. Prefix and suffix are exact
byte strings, so a rule can appear inside multiline comments or strings; this is
not a language-aware parser. Put more-specific rules before broader rules.

## Schema3 native formats

The accepted `.20` package supports all three schema3 routes. Install a
complete example through **插件 → 安装**:

- [task-notes.json](task-notes.json) maps `.taskmd` to the shipped Markdown
  editor, native outline and HTML export.
- [thoughts.json](thoughts.json) maps `.thoughts` to the shipped mindmap editor,
  native outline and SVG export. Readonly files use the shipped scene worker;
  their inspector directs you to source or Save As to a writable copy.
- [diagram-notes.json](diagram-notes.json) maps `.diagramnote` to the shipped
  draw.io editor and selected-page SVG engine. It does not add outline or
  read-only scene capabilities. The file itself must contain draw.io XML;
  changing a suffix does not convert Markdown, mindmap or another format into
  draw.io.

Schema3 accepts the exact routes `builtin.markdown`, `builtin.mindmap` and
`builtin.drawio`.
Configuration is local JSON data that selects existing application capabilities;
it cannot install or run an executable. Optional highlighting metadata is still
supported. Schema2 `outline` rules are rejected because a selected native format
owns its capabilities, but that does not mean every route supplies an outline.

The plugin panel shows the chosen format and **新建文档**. New uses the first
normalized extension (leading dot, lowercase and deduplication) and that format's
template. First registered match wins: if another configuration owns that
extension, New refuses instead of trying another extension. The installed
configuration has its own enabled state, independent of any builtin format's
enabled state. Disabling it rematches open documents; unmatched documents stay
available as source, with readonly restrictions preserved.

For the draw.io route, source mode cannot export SVG. Switch to a visible reading,
editing or split view and select the page first; export fails explicitly if the
selected page disappears or its identity changes. The existing native draw.io
subset keeps unsupported XML and warnings visible. A save without edits preserves
the original bytes, and supported source-range edits preserve unrelated and
unknown XML metadata; this is not a promise that every draw.io construct can be
rendered. The route also never creates a mindmap-style outline or scene.

Save As rematches the new path without converting the document's content.
Install, disable, New and Save As honor pending native drafts; failed writes or
unsubmitted tag input remain available for explicit completion, discard or
recovery. Installation never overwrites an existing manifest or ID. Configuration
and session data live in `%APPDATA%\LevNoteNative`; moving a program bundle does
not move that data. The portable bundle README contains all three complete examples
inline, so it does not depend on this source directory.
