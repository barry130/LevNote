# Next product slice: lightweight local image viewing, read-only design

ACTIVE2026-11-10 for Astra after accepted tab workflow. No child delegation.
Root plans/reviews only. This assignment is source inspection and a concise
implementation plan; no source/docs changes, builds, tests or app launches.
Do not mutate the accepted tab stage or interfere with .22 packaging.

The user's main goal is fewer separate file-reading apps with native MyGo UI,
local format plugins and bounded resource use. Inspect whether an initial
PNG/JPEG read-only image format can fit the existing document/plugin/byte
inspection architecture. Prefer a useful bounded slice over adding more small
shortcut releases. Do not implement before root decides the contract.

Read the actual document open/save/readonly/byte fallback model, neutral view
contract, plugin catalog/routes, MyGo public image drawing and lifecycle APIs.
Use repository/dependency source only; no browser/research/download needed.
Describe concrete integration points and incompatible assumptions. Preserve
byte-identical source and SaveAs semantics, no accidental text conversion or
rewriting binary content, unknown/unsupported fallback remains reachable.

Propose only local PNG/JPEG initial viewing with fit/original-size/zoom/pan if
supported by public primitives; no image editing, animation, metadata editing,
SVG/HTML/WebView/network, thumbnail scanning or speculative background work.
Set compressed byte, decoded pixel and retained surface/cache limits before
decode; discuss limitations of cancellation/peak memory and deterministic
release on inactive tab. Do not claim stdlib codecs are project-authored or
copyright-free; distinguish licensed codec primitives from app-owned behavior.

Deliver native-image-view-next-plan.md with current findings, a minimal set of
owned files/contracts, 2-3 implementable increments, substantive validation,
honest unsupported boundaries and recommended next step. If binary lifecycle
requires too much structural change, say so and propose a smaller prerequisite.
Keep it short (about150lines maximum). Freeze after reporting; no implementation.
