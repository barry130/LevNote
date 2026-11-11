# Narrow .29 documentation check correction

ACTIVE2026-11-10 for existing docs owner. Root read first saved check:
33/37PASS,4FAIL,exit1. Preserve script/results/streams and observed draft as
attempt1. Do not rerun broad checks or change valid product wording to satisfy
an incorrect literal. No product execution.

The four actual failures are specific verifier issues to resolve:
- VALIDATION heading pattern misses allowed space between `.29` and 当前交付.
- ROADMAP required literal says 保存、切换/关闭, while actual row states
  保存/切换/关闭 with the same behavior. Inspect actual row and exact alternatives.
- Link scanner treats inline-code examples ![label](target) and doc-block src
  syntax as real links. Exclude inline/fenced code before scanning actual links;
  do not whitelist arbitrary missing target paths globally.
- .28 VALIDATION body comparison reports changed body because permitted first
  line 当前入口 became 历史入口. Compare exact section body after only this explicit
  leading-line normalization and show any other difference; no general replace
  across facts or hashes. .27 body already passed and need not rerun.

Write one narrow saved correction script checking exactly these four conditions,
capture actual exit/streams/results, and then finish final report/file identities.
If any true content difference emerges, fix within six-doc scope and disclose
it specifically. Original failed test wasn't deliberate and remains exit1.
Finish report promptly after correction; no extra preparation/report cycle.
