# Root decisions for fixed engine package design

2026-11-09. Read proposed package design and current bundle/runner/lane source.
Direction accepted for a later source stage, NOT yet implementation. GLM may
correct only fixed-engine-package-next-design.md once. Astra .8 unchanged.

1. Call it a local fixed-engine package, not authenticated first-party. Fixed
   compatibility metadata and local release inventory cannot prove who built
   the exe. Local import should disclose that subsequent preview/export runs
   this code, and ask for an explicit product-UI choice there. No network,
   signature infrastructure or generalized plugin marketplace in this slice.
2. API needs an exact digest selection: `Import...` returns a verified package
   receipt but does NOT change active selection; `Activate(engine,digest)` and
   `Reset(engine)` separately persist. Retain old packages without deleting or
   overwriting them. If activation fails, old selection remains live; imported
   inactive bytes may remain and must be described accurately. Do not promise
   atomic rollback of two different filesystem objects. Package list API must
   expose all installed digests if the UI offers switching/rollback.
3. Two-file copy is insufficient for a reusable/distributable package: include
   original license/attribution assets and copied release inventory from the
   selected complete release. Existing13asset copy list plus portable notice
   and README define concrete source truth; use a fixed allowlist, not arbitrary
   release paths. Verify selected executable/manifest and required notices
   against a bounded strict release inventory; do not hash irrelevant other
   engines. Preserve the source inventory unchanged as provenance, separate
   from a new installed-package receipt (inventory is not authentication).
4. Define digest bytes precisely (domain+length-prefixed ordered relative path,
   size and raw file SHA256 or equivalent canonical binary scheme) covering all
   copied files, not string concatenation. 64hex path segment; never version
   strings as paths. Manifest<=16KiB, exe<=64MiB; explicitly bound release and
   total notice bytes/count and selection-file read BEFORE allocation. Every
   fixed path including destination ancestors needs normal local-disk checking.
5. Import uses streaming32KiB copy from held validated handles to a new private
   staging directory, syncs files, verifies digests, then exclusively renames to
   its content-addressed final dir. Import cancellation/failure leaves no active
   selection; clean only exact created staging paths. Existing destination may
   be reused only after complete verification; never overwrite a differing one.
   Sharing-violation/cleanup errors are visible. Reuse Windows handle primitives
   through a narrow helper, not an exported arbitrary executable resolver.
6. Runner captures a selection snapshot per request and verifies its exact
   resolved directory with current lock/hash logic. Do not hand a caller a
   generic public path override. Selected invalid/missing means failure, never
   fallback; explicit Reset is separate. Installed version remains opaque.
7. Switching drains the app-owned lane asynchronously and invalidates jobs;
   no new work while a switch is pending. Old selection remains until persistence
   succeeds, then recreate runner/lane; on write failure recreate from old valid
   selection. Preserve cleanup poison and do not revive a failed cleanup. Drafts
   remain in their views; switching package source need not reconfigure document
   adapters or discard view state. Explain any readonly-scene refresh needed.

Use these as concrete corrections and publish the revised short handoff. No
more design iterations are expected unless an actual source constraint appears.
Root will stage core import/store independently, then runtime/app only after .8.
