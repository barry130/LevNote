# Matrix wrapper accepted for one bounded run

2026-11-10 root read the frozen run-matrix.ps1 and summarize-matrix.ps1,
including failure/finally cleanup, terminal status, package protection and
nullable statistical fields. Actual file hashes match the final correction
handoff and PowerShell7.6.9 parse evidence. Fixture evidence records128/128 PASS
using actual functions/scripts with mocked process launch. Root did not rerun
tests, fixtures, preflight or any product process.

- run-matrix.ps1:25076B SHA256
  7947980b6bdc00fd5fe0171ca0b681726cbbbd55706f88286516c212433eae0e
- summarize-matrix.ps1:8391B SHA256
  1bd9455d41ba17c49b055dc4a30bbc72eeb387ce351278bf4af3b8178bc29929

Accepted only as preparation, not measurement evidence. One actual fixed12
invocation is now authorized by matrix-fixed12-execution-task.md. Stop at first
failure, retain partial results, no retries/warmups/discards. Source/tester,
accepted packages and prior evidence remain immutable. GLM test/build execution
remains held until this bounded measurement finishes.
