# Markdown table column source accepted

Accepted2026-11-10 after root review of the format planner, view transaction,
app persistence route, actual Tester capture, final outputs and source inventory.
Root performed read-only review; children implemented and executed verification.
This accepts source for .25 delivery; the current .24 executable does not yet
contain this feature.

The native cell panel adds before/after column insertion and current-column
deletion. Tables keep at least one and at most eight columns. All physical rows
are planned from the authoritative source; surviving raw cell bytes, padding,
delimiter spelling, indentation, line endings and neighboring content are
preserved. Necessary outer pipes are the only extra retained-cell boundaries.
Ambiguous trailing escapes, stale/ragged/unsupported targets or budget changes
refuse with no patch. The parser has no net change.

Ascending original-byte edits are prevalidated, converted to originalUTF16
coordinates and applied descending in one owned undo group. Selection follows
the same mapping. A changed cell flushes first as a separate undo step; failures
retain a recoverable draft or explicitly disclose that only the cell was saved.
Readonly/minimum/maximum disable actions; composition, keyboard navigation,
pointer continuity, callbacks, rollback and release behavior retain protections.

Evidence:
- [Format result](markdown-table-columns-format-result.md): full format test
  PASS0.692s and vet exit0; final focused matrix PASS0.531s. Historical compile
  and oracle failures are preserved as excerpts. Full raw format transcripts
  were not initially saved. The metadata-only
  [available session transcript](markdown-table-columns-format-evidence-transcript.md)
  preserves26recorded command/output/exit events, including failures and the
  earlier truncated display. It is145,345B, SHA256
  `48FD3DE7B0CBBCF98899870F29E5002F3B9373B1268DCFC2E598ECDAD54D9DA3`.
  These are unified session-recorded outputs, not separate raw stream files.
  No test reran solely for log formatting.
- [UI result](markdown-table-columns-ui-result.md): meaningful pointer/key and
  transaction checks; real app BOM+CRLF Save/reopen of the changed three-column
  table, with original disk unchanged before Save. The300x480Tester image and
 265/300/760DIP geometry support the narrow layout claim.
- [Final integration](markdown-table-columns-final-validation-result.md):
  markdownview PASS3.287s,app PASS20.074s,vet exit0; empty stderr. No rerun.
  Final result4,451B SHA256
  `490FE257939094761BEDDEB9AB0BA7FF4385ABBCD959D0D75C40FBC80E9EB50F`.
- [Final11-file inventory](markdown-table-columns-ui-evidence-20261110/08-final-source-after.json):
  all before/after entries match;2,382B SHA256
  `2432812FA53197C42B2431BD965810AAB7C864C3A5FF24DFCD9F5318C82D2377`.

The incidental parser formatting was restored byte-for-byte to its accepted
baseline, SHA256`AF49F0DD750CB490C2E8E78B1ACA80F58B01739C691B347323CF19360B6FE898`.
No physical Windows input/IME, packaged executable or resource claim is added.
The11source identities stay frozen through the separately assigned .25 delivery.
