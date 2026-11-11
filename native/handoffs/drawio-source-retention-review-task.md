# Drawio source ownership: bounded read-only plan review

ACTIVE2026-11-10 for GLM owner in native/AGENTS.md. No children, no product/docs
edits, tests, benchmarks, builds, launch or measurement. Publish only
`drawio-source-retention-plan-review.md`. .20 packaging inputs frozen.

Review `drawio-source-retention-plan.md` and actual Parse/rewrite callsites,
mutable public Source and return-value isolation. Validate its proposed one
private immutable buffer shared by original/initial source, a separate public
mutable Source, and edit-time ownership transfer of fresh next. Identify an
actual blocker or minimal necessary safeguards; don't expand into a new parser.

Root requirement: normalize/check MaxBytes BEFORE any full caller-data clone;
the proposed public Parse preclone must not allocate unbounded rejected input.
Ensure rollback on parse error and input/Source/original/Serialize mutation
isolation, preserved readonly compressed pages/unknown XML. Ownership-only
source slice should not alter warning behavior accepted separately.

Assess benchmark fairness: same deterministic moderate/large fixture and code
before/after; B/op measures cumulative allocation, not retained/live RSS.
runtime.KeepAlive only keeps a result reachable during the call and does not
itself measure retained heap. Any pointer-identity test pins internal ownership
only; behavior tests must establish external isolation. Keep existing core cap
and don't introduce unsafe or flaky memory-usage assertions. A small fixed
iteration benchmark is sufficient; no whole-app resource claim or UI run.

Review is final design input for later implementation after warning acceptance.
No additional broad planning phase is needed if those constraints suffice.
