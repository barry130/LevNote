# Table columns: final integrated validation

ACTIVE2026-11-10. Astra owner, no child delegation. Root has read the final
format/UI sources and handoffs and matched all10source/test identities plus
the restored table.go baseline. UI eight-file freeze is in its result; format
two-file freeze and final test/vet are in markdown-table-columns-format-result.md.
Proceed directly with the final validation below.

One final combined `test ./modules/markdownview ./app -count=1`, then one
`vet ./modules/markdownview ./app`, both through native/scripts/go.ps1. Format
package full test/vet is inherited from the same exact final source, so do not
repeat it just to put three package names on one command. App/view tests import
that frozen format implementation and establish integration. No other suite,
build, package, host, capture or resource run.

Before and after, save only the exact10owned source/test identities and table.go
baseline-preservation identity, compare them with owners' final inventories.
No toolchain/whole-repo inventory. Store command, output, start/end and exit
in the existing UI evidence directory with new numbered files. Preserve prior
failures; never overwrite them. If test failures occur, diagnose actual cause
and report a narrow correction; fixes to the existing owner files and necessary
affected reruns are authorized, new product behavior is not. Route any format
fix to its owner; no overlapping edits.

On pass publish markdown-table-columns-final-validation-result.md with exact
source identities/evidence links and actual package timings, clearly separating
format inheritance, focused synthetic UI/capture and final integration. No
physical OS IME, executable or resource claim. Freeze source for delivery.
