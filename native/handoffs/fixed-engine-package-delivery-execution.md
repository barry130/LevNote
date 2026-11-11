# Local engine package delivery — GLM execution plan

2026-11-09. Prepared after app source/test/image completion. Do not execute
until root accepts `fixed-engine-package-app-result.md`; `.8` and `.7` remain
immutable.

1. Verify final result source hashes and inspect the four tester PNGs.
2. Package once with the existing `scripts/package-markdown-worker.ps1` wrapper
   into fresh absent `.9` names, preserving raw streams/exit and manifests.
3. Compile/run the already approved default-SKIP managed bundle tester in one
   isolated copy of `.9`, using immutable `.8` release inputs and two production
   requests: imported `.8` MarkdownHTML, then reset to `.9` bundled.
4. Run one isolated hidden startup and `WM_CLOSE` of exact `.9`, using an
   ordinary bundled startup because app tests already prove persisted selection
   loading. Verify normal exit and no forced cleanup.
5. Build one compact combined validator record: exact file count/bytes, PE and
   manifest counts, protected `.8` inventory, old/new release preservation, and
   package hashes. Reuse the four app tester images as the source-matched UI
   evidence; no unrelated host/export/resource replay.
6. Publish exact `.9` paths, digests, bytes, raw evidence links and limitations.
   Only after root acceptance should README/ROADMAP/CHANGELOG/native docs be
   updated from that actual accepted result.
