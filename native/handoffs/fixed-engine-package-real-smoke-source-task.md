# Installed fixed-engine delivery smoke — source preparation only

2026-11-09. GLM read_native_handoffs_editable_engine_next_design_task_md owns
ONLY a new core/engineruntime/managed_bundle_test.go and this task's result
handoff. Astra owns runtime/app; do not edit their files or duplicate tests.
No subagents; compact before200K. Read current NewManaged/PackageManager and
existing opt-in host-path test conventions once. Implement finite source now.

Purpose: later delivery should prove a fresh host runtime can import and run
an older complete local release, then explicitly restore its bundled engine.
The controlled runtime test already checks held handles and lifecycle. Don't
repeat protocol errors, subprocess matrices or performance sampling here.

One opt-in test, default SKIP BEFORE files/processes. Exact opt-in env plus an
explicit source-release directory; tester is compiled beside a disposable
copy of a fresh complete host release, hostDir from os.Executable. The previous
accepted .8 is read only. Use a temp stateDir; call the real Manager
ImportFromReleaseDir for MarkdownHTML, verify import-only state, then Activate,
construct NewManaged and Export one small immutable UTF8/CRLF/Unicode Markdown
source. Check installed Source/receipt Digest/version/exe SHA against actual
source-release metadata, independently expected HTML fragments, source bytes
unchanged, joined Close. Then explicit Reset and a fresh managed Runner Export
the same source once; verify bundled Source/no digest and actual new bundle
version/exe hash. Output parity expected, no source edit/publication required.
These are exactly TWO repository engine requests, not an arbitrary executable
launch or app UI test. No UI mutation/test-only generic path resolver. Choose
bounded40s per-request context using production API. Always close via cleanup.

The test must log structured minimal record: scope, actual old/new engine
versions/hashes, receipt digest, payload bytes/hash, import selection, reset
selection, two successful request outcomes and joined cleanup. Existing
SHA256 verification stays authoritative; don't create a generalized validator.
No test writes into source release or bundle. If required metadata/config is
absent, fail explicitly after opt-in instead of silently skipping.

Do NOT opt in, build any host/worker/tester, run tests or launch now. Publish
source checkpoint native/handoffs/fixed-engine-package-real-smoke-source.md and
stop. Root reads source, then authorizes one default-SKIP compile/vet gate once
Astra runtime files are frozen. Actual execution occurs only with the later
accepted candidate after root reviews the delivery plan. .8/.7 immutable.
