Spec：5 项 P2。

固定 HEAD：`f2c2ceb82acecb40fb0502c047e41ba12a12be47`。指定五文件 diff、八个未跟踪文件已完整审阅。仅静态推导，未执行产品/测试；13 项范围源码及 7 项依赖 SHA256 首尾 20/20 一致。

1. **连续克隆丢失选择恢复。** [合同:52](<F:/AIproject/LevNote/native/handoffs/drawio-clone-design-20261116.md#L52>)：“恢复旧多选；redo 后同理恢复新多选”。[实现:129](<F:/AIproject/LevNote/native/modules/graphview/drawio_clone.go#L129>)按 epoch 匹配，[153](<F:/AIproject/LevNote/native/modules/graphview/drawio_clone.go#L153>)仅更新当前记录。Ctrl+D 两次→Undo 两次→Redo：Undo 第二次克隆产生新 epoch，第一条记录仍持旧 afterEpoch，随后选择恢复落空。

2. **合法光标夹取使成功收尾中断。** [合同:51](<F:/AIproject/LevNote/native/handoffs/drawio-clone-design-20261116.md#L51>)：“SetSelection(clamped original caret)”。首副本以 `<mxCell value="🙂"` 开始，旧 caret 位于 rootClose_UTF16+16（原尾标签内）；[提交:49](<F:/AIproject/LevNote/native/modules/graphview/edit.go#L49>)夹至 +15，但[实现:101](<F:/AIproject/LevNote/native/modules/graphview/drawio_clone.go#L101>)仍要求原 caret，已写源却跳过新选择、恢复记录和成功提示。

3. **复制范围内 port 漏检。** [合同:16](<F:/AIproject/LevNote/native/handoffs/drawio-clone-design-20261116.md#L16>)要求 port 整体退出，“不论其当前值”。选 a,b，内部边带 `sourcePort="c"` 或 `style="targetPort=;"` 仍通过；[守卫:215](<F:/AIproject/LevNote/native/formats/drawio/clone_cells.go#L215>)、[226](<F:/AIproject/LevNote/native/formats/drawio/clone_cells.go#L226>)只拦引用选中 ID 的值。

4. **内部边子树资格未校验。** [合同:19](<F:/AIproject/LevNote/native/handoffs/drawio-clone-design-20261116.md#L19>)：“允许子树限定为唯一合法 geometry”。a→b 内部边加 `<future/>` 或省略 geometry 仍被复制；[实现:153](<F:/AIproject/LevNote/native/formats/drawio/clone_cells.go#L153>)至181仅对 vertex 校验，[解析:728](<F:/AIproject/LevNote/native/formats/drawio/parse.go#L728>)只记录未知子树 issue。

5. **全页 ID 避重漏扫。** [合同:18](<F:/AIproject/LevNote/native/handoffs/drawio-clone-design-20261116.md#L18>)：“全部本页 XML”。diagram 内、model 外放 `<meta id="levnote-1"/>`；[解析:355](<F:/AIproject/LevNote/native/formats/drawio/parse.go#L355>)保留它，[扫描:112](<F:/AIproject/LevNote/native/formats/drawio/clone_cells.go#L112>)至120却仅收 model 内 ID，首副本仍分配 levnote-1，形成重名。

批删增量未发现新增 Spec 偏差。
