# Table columns: final integrated validation passed

2026-11-10. Final affected app/view test and vet both passed on the frozen
implementation. No correction or rerun was needed. The exact ten owned
source/test files plus preserved `formats/markdown/table.go` matched before
and after: **11/11 unchanged**. Source is frozen for root acceptance/delivery.

## Final commands and evidence

All commands ran from the repository root using the required workspace wrapper.

```powershell
& 'native/scripts/go.ps1' -GoArguments @('test','./modules/markdownview','./app','-count=1')
& 'native/scripts/go.ps1' -GoArguments @('vet','./modules/markdownview','./app')
```

| Check | Result | Timing |
|---|---|---|
| Full modules/markdownview test | PASS | 3.287s |
| Full app test | PASS | 20.074s |
| Combined test command | exit 0 | 07:19:30.9447272Z–07:19:54.4733820Z; 23.529s wall |
| Combined vet | exit 0; empty stdout/stderr | 07:20:14.7985677Z–07:20:16.0481769Z; 1.250s wall |

The test stderr is empty. Saved exact commands, outputs and exits:

- [06 test command](markdown-table-columns-ui-evidence-20261110/06-final-test-command.ps1),
  [stdout](markdown-table-columns-ui-evidence-20261110/06-final-test.stdout.log),
  [stderr](markdown-table-columns-ui-evidence-20261110/06-final-test.stderr.log),
  [result](markdown-table-columns-ui-evidence-20261110/06-final-test-result.json).
- [07 vet command](markdown-table-columns-ui-evidence-20261110/07-final-vet-command.ps1),
  [stdout](markdown-table-columns-ui-evidence-20261110/07-final-vet.stdout.log),
  [stderr](markdown-table-columns-ui-evidence-20261110/07-final-vet.stderr.log),
  [result](markdown-table-columns-ui-evidence-20261110/07-final-vet-result.json).

## Exact source preservation

[05 pre-check](markdown-table-columns-ui-evidence-20261110/05-final-source-before.json)
compares the eight UI/app entries with their owner's final inventory and the two
format entries plus table.go with the final format result.
[08 post-check](markdown-table-columns-ui-evidence-20261110/08-final-source-after.json)
records the same eleven identities after test/vet. Both are PASS.

| Path under native/ | SHA256 |
|---|---|
| modules/markdownview/table_actions.go | `1CFE1B0C2C876B39FE262ACF354B63162B36FBBCFBB0E2406E94CBE1BB9AD141` |
| modules/markdownview/table_rows.go | `2B59DDB8B16D607E1E535F20069DB2B09925995E55DEFCC1BDBBAF1AABEAC345` |
| modules/markdownview/table_alignment.go | `418356BD18CAC2FFFEB910036322FA3097BB5D49658CBE4FAE78397707D6C614` |
| modules/markdownview/table_columns.go | `8E39C1CD4D63BD7BF649BF4D20E5D659E43F828CF8242DC1336F8D12FAFF74DF` |
| modules/markdownview/table_columns_test.go | `165FC4221EB49735091FDF4B823A9CE371B56B8052EE13FC75A28E59C3533888` |
| modules/markdownview/table_alignment_test.go | `851A47223DDFB553D9CC8461A0BE218E980BD592EDB5CCAA8CF4397B544DE7A4` |
| app/markdown_columns_app_test.go | `2F414217DF18DEBE1D345EEDA81D608D1FBDE2DAC99739AEFB0392B278EF135A` |
| app/markdown_alignment_app_test.go | `AE0CFF94B3953ACA418410865F1597A784D1FE8DB93748039DBBF78EFFBB76DD` |
| formats/markdown/table_column_edit.go | `8DEB679C85C1351A1119A16DB4B0B41295CDD1D34A1173E304EC84A72D7ECC4F` |
| formats/markdown/table_column_edit_test.go | `6A372B8E406FAC406D33803ABBF6367D0BF4BEC589B09F0AF0D72BAEBA213CCD` |
| formats/markdown/table.go (preserved baseline) | `AF49F0DD750CB490C2E8E78B1ACA80F58B01739C691B347323CF19360B6FE898` |

## Evidence scope

The format owner's full `formats/markdown` test **PASS 0.692s** and vet exit 0
are inherited from [the exact final format freeze](markdown-table-columns-format-result.md).
They were not repeated. The final app/view suites above compile and exercise
that frozen planner through the native view and actual registered app adapter.

[UI focused evidence](markdown-table-columns-ui-result.md) retains both initial
fixture failures and their narrow corrections, with pointer/key/composition,
transaction/selection/ownership and BOM+CRLF structural Save/reopen coverage.
The previously reviewed 300×480 Tester screenshot is the only column-panel
capture; none was regenerated in final validation. Format-first failures remain
in the separate format progress/result evidence.

This establishes source tests and synthetic native UI behavior. It adds no
physical OS IME, packaged executable, host, driver or resource claim. No build,
package, host, capture or unrelated suite ran; accepted .24 package/evidence
remain frozen.
