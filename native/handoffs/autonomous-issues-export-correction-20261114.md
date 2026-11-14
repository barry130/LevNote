# Task-19 · Issues Copy All · 2026-11-14

## 任务状态 · type:state
- 原生 Issues 导出修正已完成并冻结：[生产源码](<F:/AIproject/LevNote/native/app/issues_export.go>)、[新增测试](<F:/AIproject/LevNote/native/app/issues_export_budget_test.go>)；原有断言与接线未改。
- 应用身份使用真实标题“轻笺 · 原生版”；每次导出重新采样真实 UTC RFC3339 时刻，不虚构历史发生/恢复时间。
- 本地 Windows 限定实现；无 MCP、联网、子委派、全套测试、vet/build、权限变更或物理剪贴板验收。

## 目标与结果 · type:goal
- 总副本 ≤65,536 UTF-8 字节；三族各20 KiB，标签512 B、警告/问题单行2 KiB、搜索字段640 B；Unicode前缀截短明确标记。
- 当前保留数、历史未保留数、本次预算省略数、单条截短数分别披露；不改原文、保留窗口或历史计数，不扫描巨型省略尾部。
- 600条短记录全保留；三族200条满尺寸记录为61,063 B，三族巨型记录60,987 B；完整边界与限定范围见[证据附录](<F:/AIproject/LevNote/.localtools/issues-copy-20261114/evidence-appendix.md>)。

## 已试方案与失败原因 · type:dead-end
- 首次caller的`noclobber`重定向失败→exit1且未跑产品测试；原始失败保留，未冒充RED。
- 原源码缺身份/时刻→真实RED02 exit1；最小header修正→GREEN03 exit0；header-only巨型字段1,000,352–6,000,299 B→真实RED04 exit1/7子失败，预算修正→GREEN05 exit0。
- 原[即时精确比较](<F:/AIproject/LevNote/native/app/issues_export_test.go#L97-L113>)仍依赖两次真实导出落在同一秒；最终本次通过，跨秒风险未缓存时钟、未改断言掩盖。

## 进度与下一步 · type:progress
- [最终一次focused原始JSON](<F:/AIproject/LevNote/.localtools/issues-copy-20261114/07-final-focused.jsonl>)：25顶级+37子PASS，0FAIL/0SKIP，exit0/2.824s；全部配对源SHA相等，原测试逐字节不变。
- 精确两个文件只读`gofmt -l` exit0/输出0 B；[验证摘要](<F:/AIproject/LevNote/.localtools/issues-copy-20261114/09-verified-summary.json>) exit0；[有限证据receipt](<F:/AIproject/LevNote/.localtools/issues-copy-20261114/evidence-receipt.json>)钉住完整SHA、字节、数字exit。
- root下一步先核对receipt内冻结源SHA，再运行既定[本地gate](<F:/AIproject/LevNote/.localtools/gate.sh>)的全量集成门禁；物理Windows剪贴板/GUI/IME、资源与新包验收仍独立且未宣称GO。
