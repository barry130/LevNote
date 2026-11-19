# draw.io 多选批次 Spec 审查 · 2026-11-15

基线：[已交付 .3 的 429 项清单](<F:/AIproject/LevNote/.localtools/drawio-multiselect-20261115-baseline/inputs.json>)；比较当前工作树及 parser 后续修正，非 dirty HEAD。只读源码审查，本代理未执行 Go。

1. **P2 · 实现语义错误，已交 format owner 修复。** [规格:54](<F:/AIproject/LevNote/native/handoffs/feature-parity-drawio-next-20261115.md#L54>)：“普通不参与几何的扩展 metadata 可原样保留”。[positions.go:241](<F:/AIproject/LevNote/native/formats/drawio/positions.go#L241>) 将任意元素的 `parent` 记成子顶点，346–347 判目标非叶；合法选中节点内 `<v:meta parent="a"/>` 亦整组禁用。应只识别实际 cell 层级，保留扩展元数据。

2. **P2 · 框选几何错误，已报根。** [规格:110](<F:/AIproject/LevNote/native/handoffs/feature-parity-drawio-next-20261115.md#L110>)：“完全包含的轴对齐包围盒”“再用 screenToWorld”。[drawio_gesture.go:147](<F:/AIproject/LevNote/native/modules/graphview/drawio_gesture.go#L147>) 用 [显示矩形:151](<F:/AIproject/LevNote/native/modules/graphview/render_cache.go#L151>) 判断包含，后者将尺寸钳至最小 8 屏幕像素。40×40 节点在 zoom=.15 时实际为 6×6；框覆盖真实矩形却未覆盖 8×8 时漏选。应将框映射到世界坐标，与原始几何比较，补小尺寸/缩放边界例。

3. **部分验收仍待闭合。** [规格:137](<F:/AIproject/LevNote/native/handoffs/feature-parity-drawio-next-20261115.md#L137>) 要求真实 Tester 生命周期与截图，[规格:146](<F:/AIproject/LevNote/native/handoffs/feature-parity-drawio-next-20261115.md#L146>) 要求 App Save/reopen。根报告 view01 354PASS/31FAIL/3旧SKIP、app01 42PASS/19FAIL/1旧SKIP；Release 导航、切页缓存、旧 dragging 状态、Shift fixture、Array points 已进入修复，不重复记成未处理发现。新 App 两个草稿恢复失败待本代理诊断；四图最终回读待根验收。

**范围审查：** 未见复制、分组、批量删除或 Host 扩展等越界功能。六对齐公式、最小轴补丁、共同吸附位移及 App 真文件/BOM/历史测试路径可追到实现；执行结论以根冻结树复测为准。
