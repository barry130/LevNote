# Managed runtime — actual source checkpoint

2026-11-09. Implementation active; first source is complete and formatted.
Focused tests follow without a further gate. No app/worker/wire/catalog/build.

`NewManaged(manager *PackageManager) (*Runner,error)` rejects nil/uninitialized
managers, otherwise binds immutable manager roots without filesystem work.
Corrupt state is retained for repair; Export refuses its captured actual error.
NewBundled remains available and keeps its established paths.

Export now reads one `selectedPackage` under Runner admission lock via the
manager's small selection lock. No file I/O/hashing occurs there. The copied
engine/source/digest/version/private roots are passed into `run`; later manager
Activate/Reset cannot change that request's target. A narrow private
`afterAdmission` test hook makes the selection-before-verification ordering
deterministic without holding the admission mutex in a test callback.

`managed.go:verifySelectedBundle` verifies the selected fixed package and retains
the entire `heldPackage` owner inside `verifiedBundle`. `verifiedBundle.close`
closes that owner after the existing process/pipe/job defers join. No verified
file is closed and reopened by an arbitrary public path. Bundled verification
uses the original function; selected corruption/missing bytes do not fall back.

EngineInfo adds `Source PackageSource` and `PackageDigest string`; ID, opaque
Version and executable SHA256 remain distinct. Bundled reports `bundled` with
empty digest; installed values come from verified bytes plus captured digest.
Start refusal now says “无法启动已验证的引擎程序”, covering either source.

Changed production: `types.go`, `runner.go`, `run.go`, `bundle.go`, new
`managed.go`. Package-store source is unchanged. Existing Close/admission/
cancellation/timeout/cleanup poison behavior is retained. Planned focused tests:
constructor/corrupt/refusal, admitted selection changed before verify, all fixed
IDs' metadata without launch, exact full guard ownership, selected corruption
without fallback, one repository test-helper launch/Wait, injected close failure
retained by idempotent Close. Existing bundle-focused checks cover the small
shared verifiedBundle extension; no resource/descendant matrix.
