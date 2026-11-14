# 文档保存指纹 · 主负责人受控字节观测边界

日期：2026-11-12。状态：任务11的具体测试/实现边界决定，**不是已执行 RED/GREEN 或最终验收**。

## 已确定的公开业务边界

任务11的主要回归通过 Document.Open、真实 Buffer 编辑、Save、SaveAs、CheckExternalChange，以及独立字面磁盘字节、Dirty、error、Undo和重试观察。禁止注入 Document.disk、添加公开测试 getter、改变公开 API 以迎合测试，或依赖时间/超大文件/RSS证明读边界。

公开正常文件在未有界和有界实现中都可能最终返回 Conflict，因此它们不能单独证明“实际最多读取 expected encoded bytes + 1”。这项严格字节工作边界需要在真实 OS 读取接口观察；业务正确性仍由上述公开路径独立核验。

## 唯一批准的受控读取观测

允许在既有授权 box 内提取 **unexported、生产实际调用** 的 `fingerprintOpened`；最小文件接口只有 `io.Reader` 与 `Stat`。生产 wrapper 使用真实 `os.File`，路径当前 Stat 使用真实 OS；预期长度为实际编码字节数，可选预期文件信息来自真实 OS 与公开 Open所得身份。不得创建 fake FileInfo、公开 getter 或全局测试替换开关。

首 RED 阶段只机械提取当前 `io.Copy` 读取、现有稳定 Stat 检查和必要参数接线；不提前加入 LimitReader、新提前 size 拒绝或其它 GREEN 修复。仅一个计数 Reader包装真实打开的文件，在合法 before Stat之后、首次 Read之前确定性增长同一个真实文件；它统计实际交付字节，并在独立字面 N+1 阈值之后提供 sentinel error。旧读取越界必须形成实际失败，而不是编译失败或断言耦合的假 RED。

随后最小 GREEN 采用 `io.LimitReader(expected+1)`与精确 count，并对真实增长返回 Conflict、不得成功返回 identity。没有 sleeps、wall-clock PASS、RSS推断或 bulk imagined tests。报告明确区分这项内部 OS byte-work 观察与公开业务回归，不能宣称原子文件 CAS 或所有 TOCTOU已消除。

## 必须保持的产品不变量

- Save和CheckExternalChange使用公开打开时的实际编码文件长度与真实 SameFile身份；长度或身份改变可尽早拒绝，不比较 baseline mtime。
- before/after/current真实 Stat、长度/count、稳定读取期间 mtime和 SameFile检查必须完整；短读、增长、IO错误可见，失败不推进 baseline/saved/Dirty/history。
- 另增公开 Open **之后**同文件同内容仅时间戳改变的阳性：CheckExternalChange不误报、Save兼容。它不能用旧测试在 Open之前设置时间戳代替。
- 保存发布后以实际 UTF8 snapshot字节+BOM或其它编码字节的长度及 digest验证；UTF16坐标、source size cap不可代替编码长度。保存后别的写者改动必须诚实报失败，不能提前建立错误 baseline或伪装撤回已发布文件。
- 原 `didSave(wrong digest)`测试的baseline、Dirty、history、selection不变断言保留；如内部签名调整，全部失败不变量仍必须验证。
- 保留 atomic writer、create-only SaveAs、权限、temp close/sync、编码/BOM/CRLF与composition/Undo语义；无公共API扩大或其它模块修改。

## 执行与记录

实施和全部 Go/gofmt/test/vet/build/run仅由 `native-reliability`在既有授权 box 新区串行进行。每次纵片使用精确冻结输入、新 label与真实 starts/exit/raw/三清单，已消费的原 guard、task5和 G2命令不重跑。此决定只释放具体观测边界，不声称测试通过、完整应用性能、平台或包接受；主负责人本机仅规划、实读、原字节镜像与 Git交付。
