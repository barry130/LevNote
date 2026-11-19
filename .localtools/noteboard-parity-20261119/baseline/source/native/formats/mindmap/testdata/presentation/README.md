These four SVGs and their source/Bounds/Warnings files were captured from the
original graphview implementation before the pure presentation extraction.

Retained run: `native/artifacts/acceptance/mindmap-pure-20261109-060918-baseline-attempt01`.
The run stores the original implementation and test source, actual command
streams, timing/result, and SHA-256 manifest. The generator is retained only in
that evidence; the regular tests read these fixtures and cannot rewrite them.

| Fixture | SVG bytes | SVG SHA-256 |
| --- | ---: | --- |
| right-rich | 1680 | 60D3122C086D9B1EA702516B924B2F975CE46C9B81F2AB2E4D2B45A3ED6755E4 |
| left-plain | 1677 | 918F60DBC2FDF39BE76FA8FA224714541B67252FC33C63947CD833697DC1A4A5 |
| negative-unsafe | 954 | A516029166E17B357371D77EC9431CC0700CB5B8DCE2FCD0A733E72879AF0998 |
| blank-long | 1731 | 90DF679E350DF6CBABAEE196133BF036745B904C57BC4B7670E15BFF58500B9B |

Negative bounds are intentionally supplied by that fixture. All other bounds
come from the original native layout options. Covered behavior includes left
and right trees, org layout, unknown layout warnings, escaping and multiline
Unicode, duplicate/custom icons and tags, literal/unsafe paints, hidden children,
collapsed nodes, image placeholders and long-label truncation.
