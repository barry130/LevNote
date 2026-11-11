# DSH 只读复核备忘：准备02仍未迁移或执行

日期：2026-11-11。
原 Codex 主会话：`01a1b6b1-3caf-76b1-81e2-183c090b7b86`。
唯一远端：`main / cb-698f697fed7c`。
状态：辅助源码阅读记录，不是原根代理接受、语法检查、远端守卫或产品验收。

## 读取范围与身份

DSH Lead 分段完整读取了 [Linux守卫草稿02](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/linux-guarded-validation-02.py>) 全599行，以及 [操作说明](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/remote-runbook-02.md>) 和 [原根交接接受](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/root-preparation-acceptance-02.md>)。

再次以本机只读文件身份工具核对，三项与既有固定清单一致：

| 输入 | 字节 | SHA256 |
| --- | ---: | --- |
| [源码ZIP02](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/levnote-source-inputs-20261111-02.zip>) | 5227577 | `457503deecbcf0a534cc3240027048454bdf758076e0ee8d3a0a64f7773db7f9` |
| [输入清单02](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/inputs-02.json>) | 244715 | `a51734c0e9b40051ad1ba06ad4b5e0acc4d92ef1a3de5261259fa435b5df2e51` |
| [守卫草稿02](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/linux-guarded-validation-02.py>) | 33772 | `7d53c6b7374a52c9d3e6749e423449d073778ac1c33c49888a121b97d43e17d7` |

这是本机文件身份，不是远端收到的文件身份；此次未重新解压或核对1208条输入。

## 源码可见的边界

- 草稿限定 prepare/test/vet；产品参数固定为原 app test/vet，另需激活字串和被审阅的 guard-ready 哈希。
- prepare 与产品命令分离，包含完整输入、五源码、26补丁、工具链归档与提取物检查；指定隔离环境和离线 vendor。
- 前后身份检查、原始子进程流及退出记录存在于代码路径；仅阅读不能证明它们在远端实际工作。
- 既有平台限制未变：Linux/CGO0、Python ctypes 与字体库前提不能代替 MyGo实际后端、Windows物理IME或Windows.30交付。
- 未解析、导入、语法检查或执行此Python文件；没有任何新的Go、test、vet、构建、启动或测量结果。

## 已撤回的显示层误判

最初工具文本显示使转义序列看似重复，形成了错误的疑虑；不得把该疑虑作为源码缺陷。
原始字节读取确认 newline 字面量的内容为 `22 5c 6e 22`，路径反斜杠字面量为 `22 5c 5c 22`，符合Python源码正常写法。
连续反斜杠搜索只命中路径反斜杠字面量一处；重复转义是工具输出显示层现象，不是已证实的代码错误。
未因此改动草稿或冻结输入，也未建立“静态阻塞缺陷”报告。

## 仍未具备的执行条件

[既有传输结构证据](<F:/AIproject/LevNote/native/handoffs/remote-transfer-20261111/transfer-investigation-result.json>)仍为 transferred=false，上传操作0；没有远端文件长度与SHA256回执。
当前工具表没有可调用的本机文件原字节到指定box的桥接；远端shell可用不等于能够读取本机文件。
活动profile登记 dsh-ops 0.5.2，但版本登记不证明先前本机 EXECUTOR_UNAVAILABLE 已恢复；没有重试本机命令或变更执行器。
已安装PTC相关组件的存在不代表当前可调用、已获启用授权或已证明能转发这些文件。未启用任何组件、变更全局配置、读取凭据、开公开服务或上传源码。

## 下一步

先明确并取得工具能力变更的范围授权；若采用PTC，只考虑现有组件和既有box授权，不安装新包、不读取令牌，不改变项目输入或验收门槛。
工具变更后必须先证实可用的程序化原字节通道；仍无能力时直接报告，不能再循环查询同一工具清单。
真实上传及远端身份核对完成后，才按原根流程处理固定Go归档和独立prepare；实际证据必须交回同一Codex根会话，app test/vet及.30仍不自动激活。
