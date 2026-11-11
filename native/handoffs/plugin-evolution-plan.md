# Plugin evolution recommendation — 2026-11-08

Scope: design-only review requested by `plugin_evolution_review_task.txt`. No
product, test, or documentation source was changed, and no tests or builds were
run. Outline implementation remains the active deliverable; this note is for a
later finite stage after its acceptance.

## Current boundary

The current native path already has an honest two-level plugin boundary:

- Trusted compiled format adapters (`modules.Builtins()`) implement native
  Markdown, mindmap, draw.io, and source views. The app selects them through
  `editor.Adapter`; it does not import concrete view packages.
- Untrusted installed plugins are bounded JSON data only. A plugin claims
  extensions and supplies lexical metadata (`lineComment`, `blockStart`,
  `blockEnd`, `keywords`). The host uses that data for source highlighting;
  it cannot provide Go functions, scripts, files, processes, or network
  access.

The application always has a source/plain-text fallback. Installing or
disabling a plugin flushes views, reconfigures tabs, and does not silently
rewrite document content. This is a good foundation: the next increment should
extend the untrusted data model and trusted host interpreter, rather than turn
the current manager into a general plugin SDK.

## Options considered

| Option | Assessment |
|---|---|
| Go `plugin` package | Not a Windows-native distribution answer. It does not remove ABI/version/process constraints and would not make format plugins safer or easier to install. |
| Dynamic DLL loading | Unsafe and brittle: untrusted native code would run in-process, with crashes and memory/file access outside the current data-only guarantees. |
| Local process protocol engines | Technically possible later, but requires a stable protocol, process lifecycle, sandboxing, crash recovery, memory budgets, and packaging. Too large for the next finite stage. |
| Trusted compiled modules with factories | Useful architecture work, but it mostly formalizes what `modules.Builtins()` already does. It cannot provide user-installed formats and should not be presented as such. |
| Declarative native source-format schema | Best next increment. It extends the existing bounded JSON model with a small, trusted host interpreter and gives installed plugins a real format capability without code execution or a heavy runtime. |

## Recommended next finite stage: schema 2 source outline formats

Add a conservative, declarative source-outline capability to installed plugins.
This is not a general parsing SDK and does not promise arbitrary rich format
support. It gives a locally installed format control over extension matching,
source highlighting, and a bounded, source-only outline.

### User-visible behavior

A schema 2 plugin could describe a simple source format such as INI, TOML-like
configuration, Org-style headings, or another line-oriented format. When
enabled:

- files with its claimed extensions open in the existing source editor;
- declarative highlighting continues to work;
- the sidebar outline lists bounded heading/section labels from source lines;
- activating an item reveals the truthful UTF-16 source position;
- no preview, graph canvas, second runtime, or document rewrite is created;
- disabling or uninstalling the plugin removes that outline and returns the tab
  to the normal source/code fallback.

`AnchorID` remains empty for these items because a custom source format has no
Markdown link-anchor semantics. The outline list identity remains separate and
version-safe, as in the outline contract.

### Manifest shape

Use a new schema version rather than adding unknown fields to schema 1:

```json
{
  "schema": 2,
  "id": "ini-outline",
  "name": "INI Outline",
  "apiVersion": 1,
  "version": "1.0.0",
  "extensions": [".ini"],
  "lineComment": ";",
  "keywords": ["true", "false"],
  "outline": [
    {"prefix": "[", "level": 1, "suffix": "]"}
  ]
}
```

Keep the field set deliberately tiny:

- fixed literal prefixes/suffixes, not regular expressions;
- at most 8 outline rules;
- each marker at most 16 valid UTF-8 bytes;
- levels 1–6;
- the existing lexical fields remain optional;
- unknown fields continue to be rejected.

Avoid regex or expression fields in this stage. They increase review surface
and make resource behavior harder to reason about without adding much value to
the first useful source-outline case.

### Host interpretation

Add one trusted, generic source-outline source used only for installed schema
2 plugins. It can reuse the existing bounded `code.HighlightDeclarative` result
to identify comment/string ranges and reject outline candidates that fall
inside those tokens. This avoids inventing a second lexical engine.

For each accepted source line:

- match a literal outline rule after optional leading whitespace;
- extract and bound the display label;
- use the outline rule syntax start as the UTF-16 `Offset`;
- produce `Level` from the rule;
- leave `AnchorID` empty;
- use a host-generated stable outline `ID` and existing text range mapping to
  resolve identity through compatible edits.

The source outline is observational. It reads the authoritative buffer, does
not call Flush, does not commit a draft, and does not alter selection, caret,
scroll, or document content. Navigation uses the same fallible Flush and stale
identity checks as the outline contract.

The app can synthesize a source-only `editor.Adapter` for the installed plugin.
It supplies only `Outline`; it supplies no `Create`, rich view, or export. The
tab stays in source mode. This keeps the app independent of concrete view
packages while making the capability visible through the standard outline
contract.

### Low-resource constraints

- Keep the manifest limit at 64 KiB, the installed plugin count at 256, and the
  current no-symlink/no-overwrite installation rules.
- Cache one bounded outline snapshot per active source document keyed by
  buffer identity plus buffer version; never use version alone.
- Reuse the outline item cap and label bound from the accepted outline contract
  (for example, 2,048 items and bounded labels).
- Release the outline cache when the tab is closed, reconfigured, the plugin is
  disabled/refreshed, or the buffer is replaced.
- A 2 MiB source scan should be bounded, version-keyed, and not repeated for
  every visible frame. Visible sidebar rows continue to use the existing native
  virtual list.

### Compatibility and fidelity

- Existing schema 1 syntax plugins keep their exact behavior.
- A schema 2 plugin without an `outline` field behaves like a schema 1 syntax
  plugin.
- Plugin ordering remains: specific builtins, installed plugins, then code
  fallback. An installed plugin must not override Markdown, mindmap, or draw.io
  by extension collision.
- The document source remains the authority; this feature never serializes or
  rewrites it merely to collect or navigate an outline.
- Partial or failed extraction is visible. It must not be replaced by fake
  headings or an empty “unsupported” state when a partial outline is usable.
- The feature must not be described as complete third-party format support; it
  is a bounded source-format capability.

### Substantive acceptance checks

When implemented, focused tests should cover:

1. schema 2 validation, including marker limits, invalid UTF-8, duplicate IDs,
   unknown fields, and install rollback;
2. source outline extraction with Unicode, CRLF, duplicate labels, astral
   characters, surrogate-safe UTF-16 offsets, and blank lines;
3. candidates inside line/block comments or strings being ignored;
4. stale identity after insertion/deletion above and between duplicate labels;
5. failed/conflicting Flush retaining draft, mode, selection, and source content;
6. plugin enable/disable/refresh releasing caches without retaining a view;
7. source-only navigation creating no rich view and never changing the file;
8. installed ordering remaining behind specific builtins and ahead of code;
9. a 2 MiB/2,048-item bounded workload with snapshot reuse and release;
10. visible partial/truncated disclosure and distinguishable collection error;
11. schema 1 compatibility after the schema 2 change.

## Future possibilities, explicitly not this stage

- Incremental, accurate cross-line highlighting with a bounded per-line state
  cache. This would improve source fidelity, but should be a separate design
  because visible-line state reconstruction needs explicit work and memory
  accounting.
- A trusted compiled module factory registry. This could make internal format
  composition more explicit and lazy, but it is not user installation.
- A local process engine for advanced formats. This should wait until a format
  genuinely needs capabilities that declarative data cannot express, and only
  with a separate sandbox, protocol, lifecycle, and resource design.

None of these future ideas should be promised in user documentation until they
have concrete tests and measured resource behavior.
