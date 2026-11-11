# Volume-root correction accepted for .10 delivery

2026-11-09. Root read diagnosis.json and its actual mkdir C:\\ access-denied
cause, original and final packageGuard.lockDir, new Windows regression source,
red-old-behavior/stdout.log, focused-green-final command/stdout and final affected
package/managed stdout/command and scoped-vet command under
artifacts/acceptance/engine-package-volume-root-fix-20261109-190635.

Accept the narrow open-first correction: validate/hold root and existing
ancestors; only genuine absence below a held parent permits Mkdir and reopen.
Reparse/type/identity checks, access failures and handle ownership are retained.
Old behavior failed both root-create and cross-volume manager cases; final
focused cases passed0.746s, affected package/managed set passed6.787s, vet exit0
(recorded wrapper wall0.591s). Default opt-in production smoke skipped here as
intended. Hardlink case passed; two symlink cases skipped because this Windows
account lacks symlink privilege. Do not count those as exercised protection.

Accepted runtime file SHA256:
BC7A4254A62F4A2E21610EC35EA64CB73951FCB6367EE12A468DA24A3447CA05.
New package_guard_windows_test.go SHA256:
3B7A12F49B2B06BBA2FC9A2F39D1645FB95BF2DF0E1B372DFCA668D9BDD9EBF7.
The root-create test is currently C:-specific, cross-volume test requires the
observed F:/C: setup; actual host ran both. No portability result is inferred.

No change to accepted app/render source or images. Delivery may treat precisely
these two files as the reviewed delta from earlier app source hashes and must
keep all final build inputs stable. Continue already authorized .10 delivery;
.8 is accepted runnable until exact new delivery passes. .9 stays failed history.
