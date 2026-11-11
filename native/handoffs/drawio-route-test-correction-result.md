# Draw.io route late-result test correction result

Status: FROZEN 2026-11-10 for independent reconciliation. Production code,
template behavior, the original route result/evidence, `.19`, and staged `.20`
artifacts were not modified. No build, package, worker, host, capture, benchmark,
or resource measurement was run.

## Scope and correction

Only `native/app/drawio_route_test.go` changed. The two review gaps in
`TestDrawioRouteSameIDRefreshRejectsLateExport` are now covered:

1. After the same-ID manifest refresh, the test calls `requireDrawioRoute` and
   requires `tab.MatchedPlugin.Name == "图纸笔记 已刷新"`. An adapter pointer
   change alone can no longer pass through a nil or failed rematch.
2. Before export, the controlled exporter sets `ignoreCancellation = true`.
   After refresh cancellation, the test sends the old job a successful result,
   waits for `job.done`, drains the completion callback, and then requires
   `job.ready`, an empty response channel, cleared payload/warnings/error, nil
   `a.exportJob`, released lane reservation, zero export dialogs, and no output
   file. The runner therefore has to consume the late success and finish its
   lane cleanup before the stale publication gates are checked.

The first focused attempt is preserved as red evidence. It exposed that the
initial test patch placed `ignoreCancellation` in the preceding route test; the
target test then canceled before consuming the response and failed with
`responseDrained=false`. The flag was moved to this test only, and the corrected
focused test passed. No production failure was inferred from that harness
mistake.

Final test file: 14,177 bytes, SHA-256
`719AB02BE3316FD09334317CF096F2D167048C1B85A89EC89EB1362B50720C86`.

## Commands and results

Raw command text, combined stdout/stderr, and exact exits are under
`native/handoffs/drawio-route-test-correction-evidence/`.

```powershell
.\native\scripts\go.ps1 -GoArguments @(
  'test','./app','-count=1',
  '-run','^TestDrawioRouteSameIDRefreshRejectsLateExport$'
)
```

The preserved first attempt exited `1`; the corrected attempt is
`focused-2.log`: PASS, app `1.508s`, exit `0`.

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./app','-count=1')
.\native\scripts\go.ps1 -GoArguments @('vet','./app')
```

Final app suite: PASS, app `17.193s`, exit `0`. Final app vet: PASS with no
output, exit `0`.

Evidence identities:

- `focused.log`, 284 bytes,
  `45442FB9484DCD121D928D1FE92850B3B5F7B50CE40CB630012345E3AD056401`;
  `focused.exit.txt`, 3 bytes,
  `F1B2F662800122BED0FF255693DF89C4487FBDCF453D3524A42D4EC20C3D9C04`.
- `focused-2.log`, 38 bytes,
  `13D0A5E2692BDE47E84AE58BFE4FE341DB1BD6387122BCD3398D551143852A2C`;
  `focused-2.exit.txt`, 3 bytes,
  `13BF7B3039C63BF5A50491FA3CFD8EB4E699D1BA1436315AEF9CBE5711530354`.
- `final-app.log`, 39 bytes,
  `6F75B1C975F4367078DAE49468F7A9B4E548944E8ECEDE4F50FC3DDD62D781CE`;
  `final-app.exit.txt`, 3 bytes,
  `13BF7B3039C63BF5A50491FA3CFD8EB4E699D1BA1436315AEF9CBE5711530354`.
- `final-vet.log`, 0 bytes,
  `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`;
  `final-vet.exit.txt`, 3 bytes,
  `13BF7B3039C63BF5A50491FA3CFD8EB4E699D1BA1436315AEF9CBE5711530354`.

## Separate warning plan

The requested read-only handoff is
`native/handoffs/drawio-structural-warning-plan.md`, 5,124 bytes, SHA-256
`1EAB4972780AE5681251E40D2C0E572D829DAF8383FD1657C7C59C9B98C8D1E4`.
It proposes suppression only for the exact ordered empty `id="0"` /
`id="1" parent="0"` direct-cell pair, preserves all malformed/arbitrary
non-vertex warnings and source bytes, and leaves `.19`/`.20` evidence immutable.
No implementation was performed.

A reviewer may append resolution. `.20` delivery remains pending after review
and any separately authorized source decision.
