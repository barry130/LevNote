# .23 delivery runtime corrections

ACTIVE2026-11-10 for current delivery owner. Preserve all prior attempts.

1. Initialization attempt1 failed before context/package/host work with a
   positional argument error for the second acceptance file. PowerShell -File
   process invocation does not carry a typed string array as one argument.
   Put the quoted acceptance array in a small invocation .ps1 in the delivery
   evidence and call initialize-context with a hashtable/splat inside that same
   PowerShell process. Capture a distinct initialization-attempt2 command/output/
   exit record. Do not modify accepted preparation or overwrite attempt1.
   This pre-product correction is authorized; continue normal stages on success.
   Ensure the recorded command is the actual process/script invocation, not an
   illustrative in-process command that was never executed.

2. Attempt2 reached the real helper but rejected the pinned patch manifest's
   count. Root read the actual4463B manifest and verified accepted SHA256
   9113120BD2EBCD9A86842153239CDDFF84D1F1BD3F46F5D4F0F152EB4560E16A;
   it contains26 entries. The helper's literal25 is wrong, as is the corresponding
   aggregate condition. Correct only the copied delivery-lib/aggregate checks
   to26, preserving original copied versions separately and the preparation
   unchanged. Do not alter product/manifest files or .22's actual25-file count.
   Record the two-file correction and one scoped parse, then initialize with a
   fresh numbered invocation. No product build/host has run yet; no repeat of
   product operations is involved.
