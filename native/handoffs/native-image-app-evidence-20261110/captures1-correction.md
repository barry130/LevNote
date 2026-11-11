# Preserved first capture failure

captures1.output.txt is a real failed Go test. The real view's tiny decode may
already complete by the second Tester frame; requiring a nonnil pending job was
an incorrect test assertion. No image files were produced by this failed run.
The corrected fixture waits if the job is pending, then checks actual metadata
and visible controls whether completion happened on the first or later frame.

The initial runner's generated nested PowerShell command did not explicitly
propagate LASTEXITCODE after calling go.ps1, so captures1.result.json recorded the
outer shell's zero rather than the failed Go process outcome. Keep both artifacts
unchanged as failed evidence. The runner now emits `exit $LASTEXITCODE`; all next
commands use that explicit propagation. Earlier focused outputs both show PASS;
they are not rerun merely to replace metadata. Final affected suite/vet will
record propagated exits with the corrected runner.
