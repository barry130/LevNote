# Task26 · display-ambiguity correction, not a prepared CI defect

Date: 2026-11-14. Reviewer: standards-review. Root requested a finite literal audit and this note; Task20's original nine files remain unchanged.

## Result — suspected doubled-backslash defect NOT confirmed

- In [Within:16](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/ci-lib.ps1#L15-L17>), the single-quoted separator's exact numeric codepoints are `[92]`: **one U+005C**, not two. A normalized non-drive-root descendant is not inherently rejected by this prefix expression.
- [RelativeFile:48](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/ci-lib.ps1#L47-L52>) also contains `[92]`; it detects a single backslash. [Inventory:87](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/ci-lib.ps1#L80-L93>) replaces `[92]` with `[47]`, so ordinary Windows separators are converted to `/`.
- The control-range literal at [LocalPath:10](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/ci-lib.ps1#L9-L14>) and [RelativeFile:49](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/ci-lib.ps1#L47-L52>) has codepoints `[91,92,120,48,48,45,92,120,49,70,93]`: one backslash per .NET hexadecimal escape. The drive-pattern separator class has `[91,92,92,47,93]`, correctly using regex escaping for a literal backslash. No ordinary-path control-range rejection was demonstrated.
- The [runner:46](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/run-native-ci.ps1#L43-L46>) default-TEMP prefix is `[67,58,92]` (C: plus one separator), not doubled.

## Independent evidence and exact identities

Python read-only extraction reported these numeric source values, exit 0. A single isolated `powershell.exe -NoProfile -NonInteractive -EncodedCommand` evaluated only copied string/regex literals and in-memory expressions: version `5.1.26100.9168`, separatorLength `1`, within `true`, neighboring-root aliasRejected `true`, contains `true`, replace `a/b`, normalRegex `false`, nullRegex `true`; exit 0. Stderr contained first-use module-preparation CLIXML progress, not an error. No library/runner dot-source, path helper call, file creation expression, owned host, Go, product, Git or network invocation occurred.

- [Library](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/ci-lib.ps1>): 17,759 bytes / 247 lines; SHA256 `e129e24d51da401487284e5207db6b9689392013d61b6b421c72b2a711bdf927` before and after.
- [Runner](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/run-native-ci.ps1>): 17,499 bytes / 184 lines; SHA256 `fedebb8246f70307f6b0adcef0f58f28d88f17aafeafcfd0b2fa84c0162f7b35` before and after.

## Boundary and next step

The apparent double separators came from escaped tool-output presentation, not frozen source bytes. This note corrects that review hypothesis; it adds **no product or prepared-runner defect finding** and requests no source patch. Root-reported PS7 parse-only zero errors and this PS5.1 literal check are not PS7 helper execution or CI runtime PASS. Draft remains inactive and **runtime NOT VALIDATED**. Future activation still needs actual reviewed runtime/input pins, path/guard tests (including drive roots), and separately authorized end-to-end execution; this finite audit establishes none of those gates.
