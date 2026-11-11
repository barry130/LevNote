# Markdown worker protocol — read-only design review

Status: READ-ONLY REVIEW, QUEUED. Sources read: `markdown-worker-contract.md`,
`markdown-worker-runtime-proposal.md`, and current
`formats/markdown/{parser,export,types,outline}.go`. No product edit, test,
build or launch was performed.

The one-frame layout, bounds-before-allocation ordering, strict JSON rules,
no-partial-success policy, source/raw payload choice, bundle verification and
Windows lifecycle direction are materially sound. The following are the items
the controlling contract should resolve before separate activation:

1. **BOM policy is unresolved.** A UTF-8 BOM is valid UTF8, so “UTF8 is
   validated, not repaired” does not reject it. Current `Parse` retains the BOM
   bytes, so raw-BOM source would produce HTML starting with a BOM before
   `<!doctype html>`. The acceptance mentions “BOM-decoded” source, implying the
   host decodes/strips BOM first. State explicitly whether `Source` excludes
   BOM, whether warning ranges are relative to that transformed payload, or
   whether the worker must reject a leading BOM with a non-UTF8 error code.
2. **Lock the exact worker error-code enum in the contract.** The proposal lists
   `invalid-request`, `source-limit`, `invalid-utf8`, `output-limit`,
   `warning-limit`, `render-failed`, and `partial-document`, but the contract
   only says “fixed bounded enum”. Also state which malformed inputs receive a
   valid error frame versus a nonzero exit/no frame.
3. **Define `EngineVersion` exactly.** “Printable UTF8” is ambiguous. Require
   bytewise equality with the manifest value and name the allowed byte set.
4. **Make decoder cancellation explicit.** The current API has no cancel
   callback. Either document that a non-nil `ResponseDecoder.Write` error is the
   terminal signal that stops the stdout copy loop, or add a private cancellation
   seam. Do not defer overflow discovery to `Finish`.
5. **State warning-range ownership precisely.** Warning ranges must be byte
   offsets into the exact submitted source payload; if the host strips BOM, the
   contract must say that `Source.Size` and ranges already exclude that prefix.
6. **Keep UTF8 validation before `Parse`.** The existing parser slices bytes and
   does not itself validate UTF8; the worker should reject invalid source with a
   distinct error rather than relying on parser/export behavior.
7. **Keep the 8MiB protocol cap and `markdown.MaxParseBytes` equality as a
   focused test only**, not as a runtime/import dependency from protocol to the
   format package.

No other material protocol risk was found in the reviewed documents.
