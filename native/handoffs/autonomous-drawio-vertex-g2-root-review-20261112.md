# draw.io 顶点删除公开键盘闭环 · G2 主负责人有限审阅

日期：2026-11-12。状态：**单一公开用例 Linux 开发证据接受；任务8和应用交付未完成。**

## 审阅范围

本记录接续任务8的公开顶点画布删除纵片。主负责人直接读取 G2 生产删除与输入正文、原公开测试、完整数字命令回执、包装器结果和四行原始输出，并独立比较预期清单与运行前三份全输入清单。没有启动产品命令、重跑 G2、修改格式或测试源码，也不是正式双轴独立审查。

G2 只增加成功 Delete、Undo、Redo 回执后的下一次画布绑定焦点请求。它没有修复或证明全部历史回调所有权边界；旧 G1 的 Undo 失败和最初 UI RED 仍保持原始记录。

## 固定输入身份

| 输入 | 字节 | SHA256 |
|---|---:|---|
| `formats/drawio/delete.go` | 6219 | `c5b39206156b45627832b8627f13aa89abcec1381f2aa56ee1960b63e44da4a2` |
| `formats/drawio/delete_test.go` | 4390 | `1c495674b2d63af43f29bec228c5bbf0aa15ef56763475ed193ac8c693c29ce4` |
| `modules/graphview/drawio_delete.go` | 5754 | `21427291ae54ca78ac596aa2b2fcdb5fe14f2dd9c52fbc2cc717b4db9f7055e9` |
| `modules/graphview/input.go` | 6997 | `f27984381f81f53391fc036ac943c3f639b0b228d9ee370967b32aa25ce99d64` |
| `modules/graphview/drawio_delete_test.go` | 2689 | `8938202e1a2e94d156a42a1f3d00dfa6153628b8fb513d6186bff9c878831541` |

五份输入共26049字节。2689字节测试与已消费 G1 的对应测试直接原字节比较相同；没有为通过改变字面 XML、断言或输入动作。格式化派生回执记录本轮两份生产文件未被 gofmt 改写。

## 原始执行证据

授权 box 证据目录：[/home/user/levnote-dev-reliability-20261112-01/runs/drawio-ui-green-02/evidence](</home/user/levnote-dev-reliability-20261112-01/runs/drawio-ui-green-02/evidence>)。

```text
/home/user/go/bin/go test -v -count=1 -timeout=120s \
  -run '^TestDrawioCanvasDeleteUndoRedoPreservesSource$' ./modules/graphview
```

- 工作目录：`/home/user/levnote-dev-reliability-20261112-01/runs/drawio-ui-green-02/source/native`。
- Go PID/owned PGID：26945；Go退出0，包装器退出0。
- 实际启动：`2026-11-12T10:03:52.843092+00:00`；结束：`2026-11-12T10:04:10.071679+00:00`；回执 elapsed `17.22988707802142` 秒。
- 包装器 timeout180秒，Go timeout120秒；`timed_out=false`，`owned_cleanup=[]`，`owned_group_remaining_after_wait=false`。
- stdout176字节，SHA256 `a1328a64b087145c3232abdb50c21ae593b2219d43741a67a3f0877cfa845eae`；stderr0字节，空串 SHA256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`。
- 完整输出只有原用例 RUN、PASS(0.22s)、PASS、包 PASS(0.232s)，没有额外跳过或未到达的 Redo 断言。

环境实读为 Go1.27.3、Linux/amd64、CGO0、vendor、本地工具链、离线模块与遥测关闭，GOENV/GOWORK关闭；label隔离 HOME/XDG/tmp，lane隔离 GOPATH/cache。没有借用原 guarded 输入的执行环境，也不证明 Windows 行为。

## 全输入守卫复核

- 预期清单：201490字节，SHA256 `2f68252d2beb633be5e01947fc3a858a3ae7c2a8b1dd5afffffbafd757d93f9c`。
- `source-before.json`、`source-after.json`、`snapshot-after.json`均199865字节，完整 SHA256均为 `f42d554d236a82c6564cc2bc689f658d2e653af1650ef13d22733a7e28da1003`。
- 主负责人使用只读元数据脚本逐项比较 roots 与1195个文件的字节数/完整SHA：各清单都与预期相等，三实际清单相互相等，总输入16988573字节。此比较脚本未执行产品。
- 包装器结果为 `COMMAND_PASSED_LIMITED_SCOPE`；独立收取回执标注 `product_executed=false`，表示收取动作本身不执行产品，不否定其索引的上述真实 Go 执行。

## 已证明行为与未证明边界

原公开外部包测试通过真实 Adapter.Create、可见画布 Find/ClickAt、Delete、CmdZ、CmdShiftZ，证明一个可编辑顶点删除保持独立预期 CRLF/XML/未知属性与注释、只提交一个主文档历史项、一步 Undo 精确恢复原字节/选区、Redo 稳定，并按断言触发 Changed1→2→3而无普通失败 Report。

尚未证明：连接线实际点击选择、删除按钮与关联线计数、只读/不支持格式反馈、保留字段草稿与原生 IME、全部重入 Release/Changed/Flush 所有权、两个完整包测试/vet、正式双轴审查、集成回归、真实 Windows 平台及新包交付。不能将本用例 PASS 扩大为任务8或应用已可交付。

## 下一纵片

已独立实读新公开连接线用例：仅在现有测试后追加一个函数，5484字节，SHA256 `1d128ff3e1f13ee647b8a2267f56d1cb3018511d8c61bdd3a25213af8613aa11`；前2689字节与 G2 原用例相同，其余四份生产/格式输入未变。它使用对称320×80图纸中心的真实可见线段点击，要求仅删除连接线、两顶点和注释/未知属性原样保留、Undo/Redo精确闭环。

唯一产品执行 owner `native-reliability`以全新 label 收取该用例实际 RED，然后由 `native-readiness`最小修复顶点优先、复用已绘制线段、屏幕 DIP 有界距离和最后绘制连接线的命中选择。原 G2 已消费，不重复运行；后续每个纵片仍保留其原始失败和真实输入身份。
