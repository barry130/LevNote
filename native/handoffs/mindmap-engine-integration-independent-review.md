# Independent second-engine integration review

Status: COMPLETE 2026-11-09. Read-only source review; no product, test,
evidence, build, process, network or UI execution was performed.

## Conclusion

No actionable integration defect found in the frozen runtime/app sources or the
published Mindmap command source.

Reviewed runtime/app fingerprint evidence:
`native/artifacts/acceptance/mindmap-engine-20261109-063755-owned-final-attempt03/source-after.json`
(inventory SHA256
`DAD757EC9BEF9AB04844D4D7B40CDFC0E35AF29C0073E86613AEA65E12847464`),
with key seams matching `mindmap-engine-runtime.md`: `run.go`
`2F7901206AD42D6A7F361931655E645361A9E27C3CAEAACF300BE4C7B4E0F960`,
`bundle.go`
`9ABCC9EF3D7CF4DC2649171BA543726E6C8682CAC47E9D3E7F67888C80EFDDAE`,
`app/export_worker.go`
`DFDE688656A15B3671780872E9D69CEBC1FB84C42B9607405ED31BF25A3E457E`,
and `modules/graphview/adapter.go`
`06BCAF60C3ADAB8DB30C5943DFDB75FCCBA2FBA0F032A3BE8D0A861A99364915`.

The reviewed Mindmap command is
`native/cmd/levnote-mindmap-svg/main.go`, SHA256
`7207D0BBD20F218ADEE8215ADC56F4D9919C8583FE30FA2777A4FF057F3B687E`,
4929 bytes, matching the published worker checkpoint. Its tests are not yet
present in the working tree, so command test coverage remains outside this
conclusion until DeepSeek publishes it.

## Runtime findings

`Runner.Export` admits only a catalog `Lookup` result and then the platform
`Match` before source-size, prefix encoding, bundle verification, source
writing or process start. Markdown remains 8 MiB and Mindmap is 16 MiB. The
request uses the descriptor's exact format, and the response decoder is bound
to the same expected ID and manifest version.

`verifyBundle` re-resolves the expected descriptor and derives both directory
and executable from it. `parseManifest(data,id)` compares schema, engine,
protocol, format, platform, filename and positive executable size against that
descriptor; manifest paths are never trusted. A coherent Markdown manifest in
the Mindmap directory is therefore rejected, and a changed manifest filename
cannot redirect the request. Runtime Version, Size and SHA remain runtime
manifest facts rather than compiled hashes.

The runner's single admission slot is shared across both IDs. Busy admission
does not read the second source. Failure paths clear `Result.Payload`, and the
existing lock, Close, cancellation, timeout, Job, pipe, stderr and cleanup
lifecycles are reused. The mechanical `HTML` to `Payload` rename has one field
and no compatibility alias; historical JSON field names remain test-only.

## App findings

`exportDescriptor` uses catalog `Lookup` and requires the advertised format to
equal the descriptor format before `Match` checks platform. A wrong/unknown
pair is visibly unavailable and cannot silently select another format. Source
preflight uses the operation's cap and names SVG/Mindmap correctly. Status
text derives from the validated descriptor, while status entries remain keyed
by ID, so Mindmap does not overwrite Markdown's historical status.

The app captures tab, document, buffer, editor, adapter, name and capability
before synchronous `Flush`, then rechecks them after it. It captures the
version after a legitimate field Flush. `validExportIdentity` rechecks all
owners and buffer version, and `recheckExport` repeats Flush plus the identity
and in-group checks before publication. The immutable `text.Snapshot` copies
only piece descriptors; live edits cannot alter an in-flight source.

Worker failure, missing/corrupt result, stale source/adapter/capability and
invalidation paths do not call `View.Export`. Source mode retains `View == nil`.
Only `MindmapAdapter` advertises the fixed SVG engine. Disabling Mindmap falls
back to source without creating a new worker job. Existing modal depth,
deferred delivery, cancel/close, exclusive `WriteNewFile` and payload clearing
paths remain intact.

## Mindmap command findings

`serve` uses `ReadRequestFor(MindmapSVG)`, so foreign requests and oversized
source are refused before source allocation. Source bytes are retained as
received; a document BOM is not stripped. The command uses the fixed 16 MiB
descriptor, and `ParseOptions{MaxBytes:16MiB}` leaves depth and node limits at
the parser defaults 256 and 100,000 because zero values are normalized to those
defaults.

Layout uses `NativeLayoutOptions` and rejects severity-error layout issues.
Warnings are converted in parse, layout, presentation order, all no-location
issues use `[0,len(source)]`, and the 257th warning or an over-long warning is
refused as `warning-limit` rather than truncated. Errors use bounded non-source
messages. `ErrSVGOutputLimit` maps to `output-limit`, and all other export
failures return no partial bytes. `WriteResponse` validates the status, warning
and payload bounds before the command returns success.

## Scope and limits

This is a finite source review only. It does not repeat or revalidate the
owner's test executions, inspect every historical lifecycle oracle line-by-line,
run the real worker, validate a packaged bundle, exercise a physical UI, or
claim resource/OSIME acceptance. The command's future test file and actual
worker gate evidence remain pending from its owner. No stdlib behavior was
contradicted in the reviewed paths.
