# Portable README: narrow wording reconciliation

ACTIVE for DeepSeek portable-doc owner2026-11-10. Preserve original prepared
result/link evidence. Final Stage2 tests still running; no packaging yet.

Root inspected selected actual implemented UI strings: reset button is
`重置全部引擎`, access label `将所有引擎恢复为随包选择并保留安装包`, action label
`重置全部引擎选择`. README invented `重置全部四个选择`; replace two quoted button
mentions with actual `重置全部引擎`. Source instruction should say the visible
`阅读 / 编辑` or `分屏` modes rather than inventing a `预览` button. No need quote
all tooltips. Worker source refusal behavior remains pending final Stage2 tests.

Two factual edits: import copies the fixed required executable, manifest and
notice assets, not an arbitrary '完整目录'. Restore precise prior wording
`复制所选引擎的可执行程序、清单和必需的许可证/归属文件`. Warning spans are whole
source for drawio (including page-specific issues), as source verifies; say
whole-source range with scope in message, rather than a new implied different
range for drawio only. Keep concise user guidance and VERSION exact once.

Only portable README and new `drawio-worker-portable-wording-result.md`. No
tests/builds/launches or duplicate link scan (no links changed). Record final hash
and no further behavior claim. Freeze. Root will confirm with final Stage2 result.
