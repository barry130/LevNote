# `.26` metadata adoption attempt 1 failure

The first direct invocation of `repair-adoption.ps1` exited1 before producing
`portable-link-repair-result.json`, repair helper provenance, or the repair
candidate copy. The observed exception was:

```text
Portable authored local link does not resolve inside output: README.md -> release-manifest.json
```

At that point the script had written the approved repaired candidate
`README.md` and removed the archived failed candidate manifest so the existing
manifest writer would produce a 25-entry payload manifest. The subsequent
local-link scan still required every authored target to exist, including
`release-manifest.json`, so the scan ran before the writer and failed.

The partial candidate state was preserved and inspected:

| File | State |
|---|---|
|`README.md`|Present,16,694bytes,`DCA36FED6CB7CBEB97D9A6C7C879C6886563703E4BB65BD5293D9C99AA692138`|
|`release-manifest.json`|Absent; archived failed bytes remain outside the candidate|

No raw stdout/stderr wrapper files were emitted by this direct attempt. This
record describes the observed tool exception and does not represent a captured
wrapper log. The corrected helper resumes only from this exact partial state,
regenerates the manifest before scanning links, and preserves the failed
package and archived candidate bytes unchanged.
