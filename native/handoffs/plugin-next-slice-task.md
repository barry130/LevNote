# Read-only format plugin roadmap review

The user wants LevNote to replace the need for many separate editors, using
MyGo native UI and project-authored document engines, with format support as
plugins and low resource use. Root coordinates and reviews; all implementation
is delegated. Native Windows workspace is F:/AIproject/LevNote. A finite
Markdown selected-column alignment implementation is already owned by Astra;
do not edit, test, build, launch or interfere with it.

Your task is READ-ONLY planning. Inspect native/AGENTS.md current paragraphs,
native/ARCHITECTURE.md, native/formats/API.md, native/README.md and actual relevant
core/plugins, core/engineipc, core/engineruntime, app/plugin and modules contracts.
Use rg to find real paths rather than guessing names. You may inspect legacy
product requirements as behavioral references, not reuse third-party editor
implementations. Do not run software, tests, builds, benchmarks, network requests
or websites. Do not spawn agents. Write only
native/handoffs/plugin-next-slice-review.md and send compact findings to root.

Explain the actual present boundary between (1) built-in format adapters,
(2) installable local JSON syntax/outline configurations and (3) the fixed local
Markdown HTML worker. Identify the highest-value next finite slice for making
format support pluggable while keeping native UI and existing source/undo/file
safety. It can be a contract addition or a concrete new format only if actual
code justifies it. Do not assume arbitrary executable plugins, a sandbox,
cross-platform support or a low-memory advantage already exists.

Recommend one next slice with one concrete user workflow and acceptance
criteria, proposed minimal files/interfaces, memory/loading ownership, failure
and source-preservation behavior, and compatibility with disabled builtins and
unconditional plaintext fallback. Name two plausible alternatives briefly and
why they are lower priority based on actual current implementation. Distinguish
demonstrated code gaps from hypotheses needing experiments. Include verified
file/line references. Keep the handoff concise (roughly1500words maximum) and
do not create an implementation backlog of unrelated features.

Only the new handoff file may be changed. No package, accepted evidence, source
documentation or running-agent code changes. Send an interim finding once the
current boundary is understood, then finish the handoff. Stay well below200K
context; request compaction before that point if unexpectedly needed.
