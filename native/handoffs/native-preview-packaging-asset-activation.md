# Packaging source review and asset-only activation

2026-11-09. Root read the corrected fixed executable guard, per-engine output
metadata and independent 13-path asset oracle. These source changes are accepted.
The portable README still has one inaccurate paragraph despite the checkpoint's
claim: it does not actually mention Save As, and says changing views is needed
for export. Correct this exact remaining wording before the fixture run.

Use these actual user-visible semantics in the readonly paragraph (equivalent
clear Chinese is fine):

只读场景支持缩放、平移、节点选择与大纲导航；节点属性、标签和图片引用
请在源码中查看。需要修改时先“另存为”可写副本，随后可使用可编辑导图
的属性面板；切换源码本身不会解除只读。SVG导出仍可直接使用应用的
“导出”按钮，不必切换视图。下方标签编辑说明仅适用于可编辑思维导图。

Also prefix the actual tag section with 可编辑思维导图. Capture the final file
hashes truthfully; do not retain old hashes as final. No other source changes.

Then execute the existing scripts/check-portable-package.ps1 once, asset/path
fixture only, in a fresh evidence location. This activation permits that run
and its bounded evidence capture without another source-only round. Retain real
arguments, stdout, stderr, exit, fixture result and before/after input hashes
for the three owned files, portable-package-lib.ps1 and actual copied assets.
The helper's fixture bytes must remain unexecuted. No Go, build, production
worker, host UI/resource launch or edits to prior bundles/evidence. Report the
exact artifact location and actual outcome, then relinquish packaging sources
for the later Astra delivery. Root will read source and raw results together.
