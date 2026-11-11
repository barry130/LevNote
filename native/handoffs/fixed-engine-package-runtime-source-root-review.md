# Managed runtime source checkpoint — root review

2026-11-09. Root read current runner.go/run.go/managed.go/types.go/bundle.go,
the held-package close helper and actual source checkpoint. No blocking source
finding. Selection is captured under admission before any I/O; private roots
remain unchanged and the full heldPackage owner closes after process cleanup.
The version-mismatch close path already returns typed FailureCleanup through
packageGuard.close; preserve that. EngineInfo.Source/PackageDigest is the final
additive API. Accepted original run/Close semantics remain recognizable.

Finish the finite runtime checks already in progress. If they pass, publish the
actual result, then CONTINUE DIRECTLY into fixed-engine-package-app-activation.md
without another idle permission checkpoint. Root reviews the raw evidence in
parallel. If a focused check identifies a production issue, fix it and report
before considering the runtime source frozen. No broad resource/process replay.

App owns additive neutral SceneEngineInfo.Source/PackageDigest propagation if
needed so installed preview metadata is truthful, plus existing UI status copy
that currently says every external engine is bundled. No worker protocol change.
