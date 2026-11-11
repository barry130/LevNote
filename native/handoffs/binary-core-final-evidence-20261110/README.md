# Final evidence

All commands ran from the repository root through `native/scripts/go.ps1`.
Each `.txt` file is the actual captured stream; each `.exit` file is the actual
PowerShell `$LASTEXITCODE`.

- `focused-binary.txt` / `.exit`: `go test -run ^TestBinary ./core/document
  -count=1 -v` — 17/17 PASS, exit 0.
- `document-suite.txt` / `.exit`: `go test ./core/document -count=1` — pass,
  exit 0.
- `document-vet.txt` / `.exit`: `go vet ./core/document` — no output, exit 0.

The earlier implementation result did not capture command streams into files and
is not presented as stream-level proof. Its concise tool outputs remain in the
conversation, including two real failed focused runs during correction; they are
historical, not final evidence.

## 10:40 review corrections

The original evidence above is preserved unchanged. After the appended source
review, the three requested assertions were corrected and new numbered evidence
was captured without overwriting the originals:

- `focused-binary-2.txt` / `.exit`: `go test -run ^TestBinary ./core/document
  -count=1 -v` — 17/17 PASS, exit 0.
- `document-suite-2.txt` / `.exit`: `go test ./core/document -count=1` — pass,
  exit 0.
- `document-vet-2.txt` / `.exit`: `go vet ./core/document` — no output, exit 0.

## Rename identity fixture correction

The suffix-2 rename assertion was a false positive on Windows: two path-based
`os.Stat` values were first passed to `os.SameFile` only after the replacement,
and Go loaded their file IDs lazily from the new path. The fixture was corrected
to capture the baseline from the eager handle-based `file.disk.info`, capture the
replacement from an open file handle, and assert identical bytes/size/timestamp
with different identities. Existing suffix-1/2 evidence remains unchanged.

- `rename-identity-3.txt` / `.exit`: `go test -run
  ^TestBinaryReplaceByRenameChangesIdentity$ ./core/document -count=1 -v` —
  pass, exit 0.
- `document-suite-3.txt` / `.exit`: `go test ./core/document -count=1` — pass,
  exit 0.
- `document-vet-3.txt` / `.exit`: `go vet ./core/document` — no output, exit 0.
