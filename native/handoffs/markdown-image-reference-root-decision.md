# Next increment decision: explicit Markdown image references

2026-11-10. Accept the useful narrow direction in native-next-reading-review.md
and markdown-image-reference-contract-review.md, with these overriding choices.
Implementation is activated only after .25 package/evidence closure and root
acceptance. Source owners must not change packaging inputs before that point.

Add InlineImage with exact original Range/Text/Href. This enables explicit
opening through existing Host.OpenReference; no inline pixels, new decoder,
filesystem access in format/view, network reads, bitmap cache or new host API.
Supported PNG/JPEG opens through the existing app reader; other readable or
unsupported targets follow the existing app open/byte fallback. A missing or
unreadable file reports failure. Do not claim all rejected targets open no tab:
the existing byte-inspection fallback may deliberately create a readonly tab.

Initial actionable syntax is single-line ![label](destination), unescaped !,
balanced/escaped label brackets supported by the existing bracket helper.
Destination is nonempty,1..4096UTF8bytes, no CR/LF/control/ASCII whitespace,
angle delimiters or unescaped nested parentheses. Escaped destination parens
are not actionable because raw backslashes are not Markdown-unescaped by the
host. Plain Windows backslashes elsewhere remain literal. No trimming, percent
decoding, title extraction or path rewriting. Angle/title/reference-style,
multiline, malformed and oversized references stay visibly literal with bounded
warnings. URLs may be syntactically recognized; the host remains authoritative
and refuses network/UNC/unsafe references on action without network access.

Do not narrow/change ordinary link semantics. In particular an existing outer
InlineLink containing raw image syntax in its label retains its existing outer
link behavior. No independent image action is created inside it. Escaped ! does
not become InlineImage; subsequent ordinary link behavior stays as before.
Code/bold/italic owners keep their existing nonrecursive literal inner syntax;
table cells remain literal. Unsupported image fallback must not turn inner
pieces into a new image action; preserve other ordinary constructs outside it.

Accepted image syntax still is not rendered inline. Emit an accurate bounded
reference-only warning instead of silently claiming full image support; rejected
forms retain unsupported-image-style warnings. HTML explicitly emits escaped
original image syntax with no img element/resource fetching. Native dense-row
and fallback rendering also retain original syntax rather than only alt text.

Use one shared16interactive-control per-row budget for links plus image refs,
the existing64span interactive budget and all existing parser/preview budgets.
Oversized destination yields no InlineImage and a visible raw fallback; do not
promise a disabled image action for a span the parser deliberately did not emit.

Native image control is visibly identified as a picture reference with distinct
accessible name and bounded tooltip. Ctrl/Cmd+click or Tab+Enter follows existing
link behavior; ordinary selection/double-click editing and readonly reading stay
intact. No extra toolbar, decorative bitmap or per-frame polling. The raw syntax
may itself be the visible control label so content is not silently hidden.
Capture inline kind as part of source identity, exact source/enclosing ranges,
buffer/version/row and bounded cloned target. Revalidate kind after Flush and
mapping/reparse. Repeated/held/canceled or stale gestures must not open twice or
open the replacement target. Synchronous host release must be safe.

Ownership: DeepSeek formats/markdown types+inline+export+focused tests only;
Astra modules/markdownview implementation/tests and one new app test only;
GLM product docs for already accepted .25. App routing/image/session/runtime
and vendor are outside scope. Final tests cover affected format/view/app once
after focused corrections and frozen source; no automatic broad suite/build
repeat. Root reviews actual evidence and directs the next delivery separately.
