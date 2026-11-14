# Task15 Windows 保存冲突与部分发布修正 · 语义完成，格式与 ROOT 接受仍待完成

## 任务状态（type:state）
本地 Windows F:/AIproject/LevNote；公开 F1/F2/F3 修正有限实现完成，sole8 源码已停止编辑；无 MCP/box、无提交、无全产品 vet/build/包替换，旧 .29 不变。
最终受影响 suite 各一次：Document67主+46子PASS/0FAIL/0SKIP/1.883s；App379主+391子PASS/0FAIL/28既有opt-in SKIP/27.978s；两者实际exit0，[完整语义收据](<F:/AIproject/LevNote/.localtools/windows-save-evidence/final-semantic-receipt.md>)、[JSON计数/skip全名](<F:/AIproject/LevNote/.localtools/windows-save-evidence/run-summary.txt#L17-L18>)。
exact3 新格式债仍待root处理：[只读-l](<F:/AIproject/LevNote/.localtools/windows-save-evidence/21-semantic-format-status/stdout.raw>)；child gofmt-w随机相邻temp拒绝/exit2、源未改且无绕过，[如实记录](<F:/AIproject/LevNote/.localtools/windows-save-evidence/17-owned-format/denied-call.txt>)。语义通过不等于最终格式冻结或ROOT接受。

## 目标（type:goal）
6次/25ms仅32/33/1175真实 ReplaceFileW 重试保持；每次延迟后复核原digest/inode及buffer/path/version/encoding/BOM/readonly/group/baseline，不adopt external winner，无delete+move/rollback，最终check→OS非CAS仍披露。
真实publication后sync/verify错误以PublicationError保留errors.Is/As、实际target与metadataadoption；dirty/draft/history及旧verifiedbaseline不伪clean，验证失败latch以verifiedsave或显式Revert清除，sync失败旧Undo-clean合同不改。
App false出口渲染target/可能字节/metadata/draft/安全重试指引并记Issues；已接管目标同步name/editor/type/view/watch/cache/recovery，不清conflict、不记成功，原flush/ownership屏障保留。

## 已试方案与失败原因（type:dead-end）
首检后等待→01真实外写被覆盖/03新buffer假clean；普通sync/verify错误→06/08丢目标发布状态、09freshdraft假clean；App早退→11告知不足/13目标接管后host陈旧；原RED与GREEN逐项保留在[证据索引](<F:/AIproject/LevNote/.localtools/windows-save-evidence/evidence-index.md>)。
os.Stat原身份→04 Windows lazy inode重解析path；改opened File.Stat fixture。SaveAs同source prepublish→15既有合同转Save；改独立existing target后16全绿；既有70test原字节不变，[身份核验](<F:/AIproject/LevNote/.localtools/windows-save-evidence/existing-test-immutability.txt>)。
Document seam真实temp→OS发布→注入sync/真实readback；App outcome seam不冒充该链；Windows目录sync为no-op，headlesswatch/Tester非物理GUI/IME/WM_CLOSE/新包接受。readonly属性非Unix权限；single-sessiongoroutine边界未扩成非法并发。

## 进度与下一步（type:progress）
原始命令：`bash /f/AIproject/LevNote/.localtools/gate.sh test ./core/document -json -count=1` 与 `... test ./app -json -count=1`；[19raw](<F:/AIproject/LevNote/.localtools/windows-save-evidence/19-final-document-semantic/stdout.raw>)/[19receipt](<F:/AIproject/LevNote/.localtools/windows-save-evidence/19-final-document-semantic/receipt.sha256>)、[20raw](<F:/AIproject/LevNote/.localtools/windows-save-evidence/20-final-app-semantic/stdout.raw>)/[20receipt](<F:/AIproject/LevNote/.localtools/windows-save-evidence/20-final-app-semantic/receipt.sha256>)。
两suite输入前后同SHA256 `5dac92c8fe21134fcac2976b087550b082c6fed6e012e17abb99f3717be3fe35`；全部18run/144receipt成员sha256-c OK，[原校核输出](<F:/AIproject/LevNote/.localtools/windows-save-evidence/final-audit/receipts-check.raw>)；[sole8逐文件完整SHA256](<F:/AIproject/LevNote/.localtools/windows-save-evidence/final-audit/owned-sources.sha256>)及[可读源码/原raw/receipt完整hash](<F:/AIproject/LevNote/.localtools/windows-save-evidence/final-semantic-receipt.md>)用于rootpin，不以chat GREEN替代证据。
下一步root按官方诊断处理exact3格式写入限制；只改获准3路径且记录新SHA后，以新sourcepin执行全产品test/vet/build与双轴审阅，再统一物理GUI/IME/进程/session/包验收；先执行 `sha256sum -c .localtools/windows-save-evidence/final-audit/owned-sources.sha256` 核当前语义快照，非接受后可发包状态。
