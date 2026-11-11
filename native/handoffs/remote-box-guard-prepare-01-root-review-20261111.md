# Root review: remote guard prepare 01 accepted, product stage authorized (2026-11-11)

Reviewer: the DSH lead session, acting as main responsible party under the
user's 2026-11-11 ruling that transfers Codex-root responsibilities to this
session. The original Codex root thread 01a1b6b1-3caf-76b1-81e2-183c090b7b86
remains the historical baseline; .29 and all closed evidence stay immutable.

## What was reviewed (real remote result, on box)

- Caller capture: /home/user/levnote-remote-caller-evidence-20261111-01/
  (UTC 2026-11-11T09:02:23Z before, 09:02:47Z after, wrapper exit 0,
  stdout one JSON line, stderr empty).
- Guard receipt: /home/user/levnote-remote-20261111-01/guard-ready.json,
  SHA256 da70b0421c48165f9c3db880c75cac31c59f7901f9f55de583f05b0a135e0853,
  Status GUARD_READY_NOT_PRODUCT_ACCEPTANCE, ProductExecuted false.
- evidence/prepare-01/: result.json, invocation.json, environment.json,
  inputs-before/after.json, toolchain-before/after.json,
  prerequisites-before/ (ldconfig, font probes, library identities).

## Review checklist and findings

- Identities: ManifestSHA256 a51734c0..., SourceArchiveSHA256 457503de...,
  ToolchainArchiveSHA256 65efbc80..., Wrapper 33,772 B 7d53c6b7... all match
  the pinned preparation-02 identities in transfer-files-02.json and the
  root preparation acceptance. Input identity
  96e0a103b8538f9810333082ed05217e47b1360e963a7dbc8dbef0585654e12d and
  toolchain identity 9806db80da121ddc6891941ef94ce2da8e15855f15af3ee0e1e33eb51249f7b9
  are identical before and after. InputsBefore = InputsAfter (1,208 files,
  17,045,848 bytes, 5 picker files, 26 patch entries); ToolchainBefore =
  ToolchainAfter (15,657 files, 244,139,206 bytes, go binary 17,164,222
  bytes, VERSION go1.27.3, Executed false).
- Delivery fidelity: the git route delivered the pinned inputs
  byte-identically after commit
  12b420b1e953f48b4d8cf2bc96eb59f5df0d7a8f restored `-text` storage for
  frozen evidence (the earlier `* text=auto eol=lf` normalization had
  rewritten the pinned inputs-02.json into a 238,674-byte LF blob in
  transit; disclosed and fixed before the run, the guard never accepted
  mismatched bytes).
- Invocation and isolation: argv matches the runbook; CredentialFilesRead
  false; ambient variable values not copied (names only, four names
  recorded); environment.json shows a fully run-root-owned environment
  (PATH restricted to the pinned toolchain plus /usr/bin:/bin, HOME/XDG/
  TMP/GOPATH/GOCACHE inside the run root, GOPROXY=off, GOSUMDB=off,
  GOVCS off, GOTOOLCHAIN=local, CGO_ENABLED=0, GOTELEMETRY=off, TZ=UTC).
  This is the documented CGO0 offline Linux validation configuration; it
  does not represent Windows or default CGO.
- Prerequisites (observations, not a hermetic closure): ldconfig, gobject,
  pango, glib, pangocairo, cairo, fontconfig, harfbuzz and pangoft2 all
  resolved under /usr/lib/x86_64-linux-gnu/ with recorded SHA256
  identities; fc-match probes passed; Pango font map and context creation
  succeeded through ctypes; MandatorySymbols present;
  MyGoBackendObserved false; no physical UI/IME claim is made or inferred.
- PostGuardErrors empty; NoPhysicalUIIMEOrWindowsDeliveryClaim true.

## Conclusion

ACCEPTED as GUARD_READY_NOT_PRODUCT_ACCEPTANCE. The backend/platform
concerns from the runbook are satisfied to the extent observable: platform
facts recorded, isolation verified, prerequisites observed, with the
explicit caveat that library availability and a ctypes font map do not
prove the MyGo text backend in a nonverbose run; the guard's
MyGo-text-fallback check remains the enforcement point on the product
stage.

## Authorization (bounded)

The product stage is authorized exactly once per phase under the
runbook-02 interlock, on the same run root, remote-only:

- Single test: python3 -I -B /home/user/levnote-inbox-20261111-01/linux-guarded-validation-02.py test --box-id <authorized main box id, recorded in the phase evidence invocation.json> --run-root /home/user/levnote-remote-20261111-01 --guard-receipt-sha256 da70b0421c48165f9c3db880c75cac31c59f7901f9f55de583f05b0a135e0853 --activate-product-stage picker-final-app-linux-20261111
- If and only if the test passes (COMMAND_PASSED_WITH_GUARDS with
  ProductExecuted true), a single vet with the same interlock.
- No retry on failure: a failed run root is preserved as evidence; any
  further attempt requires a fresh run root and a new explicit review.

Holds that remain after this authorization: Linux results do not prove
Windows physical IME or Windows delivery; .30 stays inactive until
test/vet results and its own activation criteria; the .29 package and all
closed evidence are immutable; no source edits, assertion changes or
added skips are authorized (C:-path assertions, temp-path uppercase
reopen and the four Windows-tagged tests keep their Linux limitations
as-is, whatever the outcome).
