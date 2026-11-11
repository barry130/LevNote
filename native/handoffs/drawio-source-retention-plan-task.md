# Next resource increment: drawio source retention assessment

ACTIVE READ-ONLY2026-11-10. DeepSeek owner as listed in native/AGENTS.md.
No children; compact before200K. No product/docs edits, tests, builds, launches,
measurements or broad inventory. Publish only
`drawio-source-retention-plan.md`; use source references and clearly distinguish
inference from measured allocation/process results. Current .20 inputs frozen.

User's original objective is lower memory while opening/editing several formats
in one native app. Assess a concrete bounded follow-up: Parse currently clones
input into Document.original and private source then clones source into public
Source; applyPatch/reparse/Serialize can add more copies. Read actual callers,
mutation semantics, tests and lifecycle releases. Identify which retained copy
can safely be shared/eliminated without changing input ownership, exported
mutable Source API semantics, OriginalBytes/Serialize isolation, no-edit bytes,
unknown XML metadata, compressed-page readonly behavior or edit rollback.

Recommend one small justified optimization only if supported by caller/alias
analysis. Do not remove a public mutable API or permit external input mutation
to alter internal state. If safe improvement requires larger API redesign, say
so and offer a narrower release/retention option instead. Define substantive
aliasing/rollback/source-preservation tests and a fair output/retention or
allocation benchmark for before/after. Do not promise process memory savings
without measurement. Existing engine/host resource-workload limitations and
Computer Use failure remain unchanged; no helper retries/OS input automation.

This plan is a later stage after the warning increment. No implementation is
authorized until explicitly assigned, and no package input may change now.
