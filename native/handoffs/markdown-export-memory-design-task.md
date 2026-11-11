# Markdown export allocation follow-up — design only

Status: PAUSED. The GLM reviewer did not deliver an ACK, checkpoint or design
after repeated requests and was interrupted. It has no current ownership. Root
will finish the finite plan from inspected source; packaging remains active.
Do not claim an independent review or restart the reviewer automatically.

Owner to be dispatched by root: one crouter/GLM-5.3 design reviewer. Read-only
code/evidence inspection; write only this stage's design handoff
`native/handoffs/markdown-export-memory-design.md`. No implementation, tests,
builds, launches, network, external benchmarking or child agents. Compact before
200K context. Root coordinates; Astra's active packaging work is separate.

The next finite stage should improve the concrete temporary-allocation issue in
`formats/markdown/export.go` and `cmd/levnote-markdown-export/main.go`: HTML is
fully built as a string using whole-fragment html.EscapeString/concatenation,
then converted into []byte; the64MiB protocol limit is checked only afterward.
The source cap is8MiB. The host decoder already allocates a single bounded result
slice, and protocol v1 requires a known payload length before bytes are emitted.

Propose the smallest maintainable improvement that preserves byte-for-byte HTML,
warnings/order/ranges and partial-document refusals. Compare a bounded writer
with exact-size two-pass rendering briefly; choose one. Keep existing public
callers compatible, no unsafe string/slice aliases, no temporary files/network,
no parser rewrite, no protocol change, no new library and no peak-RSS promise.
Do not say a post-render limit is a hard live-memory guarantee. Inspect all
escaping/large-fragment paths, including heading anchors, tables and fallback.

Handoff should contain actual file/line findings, proposed API/error ownership,
precise size-bound semantics, meaningful focused regression and allocation
measurement plan, and remaining limitations. The max-source escaping test is
already present; do not propose huge redundant suites. Real process/resource
comparison and arbitrary executable plugins remain future work. Design only:
root activates implementation after packaging acceptance.
