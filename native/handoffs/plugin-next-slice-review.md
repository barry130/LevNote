# Plugin next-slice review

## 已核实边界

- `modules.Builtins()` 只组合随包 Markdown、导图、图纸适配器，富视图、编辑和导出实现均在宿主内（`native/modules/modules.go:14-20`）。
- schema1/2 JSON 只提供源码匹配、高亮和逐行字面大纲；`install` 不执行代码。schema1/2 之外的 schema、未知字段会被拒绝（`native/core/plugins/declarative.go:114-215,321-355`）。
- 当前只有内置 manifest 且其 ID 能命中编译适配器时才挂载富视图；安装 manifest 即使声明相同后缀，也不会绑定该适配器（`native/app/plugins.go:103-125`）。schema2 大纲已由 `InstalledSourceOutlineFactory` 接到源码编辑器，明确不创建富视图（`native/modules/modules.go:22-45`、`native/modules/sourceview/outline.go:1-45`）。
- 启停、安装、刷新前会 Flush 所有视图，随后重配标签；安装项停用后回到下一条匹配或可编辑源码兜底（`native/app/plugins.go:200-288`）。
- 固定 Markdown HTML worker 是可选的 `ExportEngine`，路径、ID、协议和完整性校验均固定；它不是可安装的文档内核（`native/ARCHITECTURE.md:44-79`、`native/formats/API.md:21-80`）。

## 推荐下一有限切片：schema3 固定能力绑定

增加数据型 schema3，让本地格式 manifest 把自己绑定到仓库白名单内的 native engine/renderer capability；manifest ID 与 engine/renderer/export ID 必须分开。首版只允许 `builtin.markdown`、`builtin.mindmap`、`builtin.drawio` 三个固定值，以及各自真实支持的能力位；禁止路径、命令、DLL、脚本和任意 URL。

可见流程：安装声明 `.taskmd -> builtin.markdown`、`preview/source/split/edit/export-html` 的 schema3 清单；打开 `notes.taskmd` 后直接进入 Markdown 阅读/编辑，可切源码、分屏、定位大纲并用现有 worker 导出 HTML；插件面板显示“配置 my-markdown · 引擎 builtin.markdown”，避免把别名误称为新内核。

验收至少覆盖：

1. schema1/2 行为完全不变；schema3 未知 capability、缺失 engine、虚假能力位和未知字段在安装时失败，不落半成品。
2. builtin Markdown manifest 可停用以释放 `.md` 匹配，但 `builtin.markdown` 能力仍可供 schema3 清单绑定；无匹配时仍走无条件源码兜底。
3. 启停/刷新前 Flush；失败保留当前 View、草稿和选区。停用 schema3 后 Flush，再重配到下一匹配或源码，不沿用旧 adapter/factory。
4. 同一 Buffer、撤销/保存、只读、BOM/CRLF 和导出拒绝语义沿用现有 native 实现；schema3 只增加路由，不改变文档字节。
5. 实现量应限于 schema/model、白名单能力表、`configureTab` 的“manifest ID → engine ID”解析及插件面板文案；不加载新进程，不复制解析器。

## 必须保留的两层差距

该切片只是**随包能力的可安装配置路由**。真正的格式插件仍需可独立安装、启停的文档内核，而当前缺：统一 engine 包身份/版本与能力协商；解析、渲染/输入意图、编辑事务、序列化及诊断的双向文档协议；通用进程生命周期、崩溃隔离、取消与资源预算；安装/卸载时的缓存与进程所有权。现有 IPC 只做固定 Markdown 源快照到 HTML 的单向导出，不能冒充上述内核协议。

因此 schema3 是用户可见、低风险的桥梁：先证明“installable format route + trusted engine resolution”而不延后用户流程。更直接的多引擎 IPC 切片在没有上述契约前会同时引入协议、生命周期和安全设计，暂不宜作为下一刀；仅继续扩 schema2 语法又不会产生 preview/split/export，价值更低。
