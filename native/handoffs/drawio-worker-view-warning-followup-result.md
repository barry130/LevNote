# Canceled-gesture warning follow-up: result

FROZEN 2026-11-10 for root and independent reconciliation. This repair is
limited to graphview synchronization warning order and one regression test.
The frozen view-contract and reentry-correction results/evidence remain
unchanged. No contract, app integration, adapter advertisement, build, launch,
product documentation, `.18`, or Stage 2 work is included.

## Regression and correction

The prior reentry correction appended the canceled-gesture warning immediately
after its owner guard, but a successful mindmap/draw.io reparse then assigned
the freshly parsed warning list and removed that visible warning. The issue
report still reached `Host.Report`, while the graph warning panel lost the
supported cancellation message.

`ensureSynced` now retains the cancellation text in a local warning value only
after the existing post-`Report` ownership check succeeds. It publishes that
value after the terminal parsed or error warning list is established. This
preserves the warning across normal stale-source reparses, preview-limit
failures, and parse failures. The immediate ownership guard is unchanged:
nil/replacement buffers, release, source/version changes, and retargeted pages
still return before the outer synchronization can write stale state.

`TestDrawioExportRequestKeepsCanceledGestureWarningAfterReparse` exercises a
normal stale-source reparse with an active drag. It requires a successful page
request, verifies the gesture was canceled, and confirms the graph warning list
still contains the cancellation text. The existing reentry test continues to
cover non-resurrection after callback-owned owner changes.

## Changed files

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/modules/graphview/sync.go` | 7,016 | `F09DB5207C96724CC2DEB936D989C733543AD538E7BB5FC0E9C6549CBE1F1A10` |
| `native/modules/graphview/export_request_test.go` | 10,671 | `D992C5B856358333A3BEA6C2691564B68C601D2A37B3BAA11B91FCA60E3E95B7` |

## Final validation

Graphview test, through the required wrapper:

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./modules/graphview','-count=1')
```

- Exit: `0`
- stdout: `ok  \tlevnote.local/native/modules/graphview\t12.354s`
- stdout evidence:
  `native/handoffs/drawio-worker-view-warning-followup-evidence/final-graphview-test.stdout.log`,
  53 bytes, SHA-256
  `90606F0C1C55FF399C46748E6BCDD1A5BBA200DD0DB5C331CFC1D0AB8E7223B0`
- stderr evidence:
  `native/handoffs/drawio-worker-view-warning-followup-evidence/final-graphview-test.stderr.log`,
  0 bytes, SHA-256
  `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`
- exit evidence:
  `native/handoffs/drawio-worker-view-warning-followup-evidence/final-graphview-test.exit.txt`,
  3 bytes, SHA-256
  `13BF7B3039C63BF5A50491FA3CFD8EB4E699D1BA1436315AEF9CBE5711530354`

Graphview vet:

```powershell
.\native\scripts\go.ps1 -GoArguments @('vet','./modules/graphview')
```

- Exit: `0`
- stdout and stderr: empty
- stdout/stderr SHA-256:
  `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`
- exit evidence:
  `native/handoffs/drawio-worker-view-warning-followup-evidence/final-graphview-vet.exit.txt`,
  3 bytes, SHA-256
  `13BF7B3039C63BF5A50491FA3CFD8EB4E699D1BA1436315AEF9CBE5711530354`

## Preserved prior artifacts

The original view-contract result remains 4,978 bytes, SHA-256
`75EB1B45B6EF59E7A7785697E7B3BA09A6FCCB2B07CAFC2E45647B39546879AF`;
its six evidence files match the identities recorded in that result. The
reentry-correction result remains 4,488 bytes, SHA-256
`DD33860EDA14B265128EA51FAC8DC332099D96E2B527D2D6309D577BCD982B33`;
its six evidence files also retain their frozen identities.

This result proves the method-level warning-panel state ordering only. It does
not add the fourth engine to the application, execute a managed worker, build a
package, launch the UI, or establish physical input/resource behavior.
