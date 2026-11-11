# .19 delivery harness preparation

Prepared only. This directory does not authorize a package, test executable,
worker, host or aggregate execution until root activates
`drawio-worker-release19-delivery-task.md` and the context is initialized with
`-ExecuteDelivery`.

The activation sequence writes only to fresh candidate/evidence/config roots:

1. `initialize-context.ps1` with the absent F: candidate directory, absent
   evidence directory, accepted final Stage2 file manifest, accepted Stage2
   evidence directory, accepted final Stage2 gate files and exact version
   `native-worker.20261110.19`.
2. `run-stage.ps1 -Name baseline`
3. `run-stage.ps1 -Name package`
4. `run-stage.ps1 -Name managed-smoke`
5. `run-stage.ps1 -Name host-smoke`
6. `run-stage.ps1 -Name aggregate`

`initialize-context.ps1` copies this prepared harness into the fresh evidence
directory, so the five `run-stage.ps1` commands above run from that copied
`run-stage.ps1` path and use the evidence-local `context.json`.

Each stage is launched by a hidden PowerShell process and preserves its raw
stdout, stderr, exit and invocation record. The package stage invokes the
accepted package script exactly once. The managed stage compiles and runs the
reviewed `TestDrawioManagedRealBundleSmoke` once from a separate candidate copy.
The host stage is a separate hidden copied-host launch with one owned
`WM_CLOSE`. The aggregate fails closed unless the actual candidate inventory is
25 files and every package, source, accepted-gate, managed-worker and host
condition passes.
