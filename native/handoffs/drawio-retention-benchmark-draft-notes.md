# Retention fixture draft correction before baseline

2026-11-10 root read current source_retention_benchmark_test.go. It currently
puts compressed base64 INSIDE `<diagram><mxGraphModel><root>...` which is text
inside an uncompressed root, not a compressed page. Put base64 directly inside
`<diagram id="compressed" ...>PAYLOAD</diagram>` with no mxGraphModel/root
wrapper. Verify benchmark fixture with one setup parse outside timed loops:
len(Pages)==8, last Page.Compressed && Page.ReadOnly and expected node. This
ensures claimed fixture semantics don't drift. If baseline already ran, retain
that attempt, correct benchmark before production edit and use a fresh valid
baseline; do not compare different fixture/code pre/post. No timing cost from
fixture generation/setup validation in measured subbenchmarks.
