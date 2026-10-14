# 0058 · 导图查找也搜备注：命中带字段，写回分通道

- 状态：已采纳
- 日期：2026-10-14
- 关联：[0056](./0056-mindmap-node-notes.md)（取舍 ⑤「搜索不伺候」留的边界 —— 本轮兑现的就是它的重审条件 3）、[0057](./0057-markdown-kernel-ui-follows-the-language.md)（重审条件 2 登记「note 搜索是另一轮」—— 就是本轮）
- 承接：`EditorSearch` 契约（`core/editor/types.ts`）、`collectHits`（`modules/mindmap/MindmapEditor.tsx`）

## 背景

0056 让备注看得见、改得了，但查找栏搜不到备注：`collectHits` 只扫 `topic`。
旧注释的理由（「内核不渲染 note」）在 0056 落地那一刻已经过期 —— 真正缺的
是另一半：**`replace` 的写回通道是 topic 专用的**。`TopicHit` 的 `from/to`
是段内偏移，topic 命中走 `setNodeTopic`；把 note 命中塞进同一张表而不分
通道，替换就会按备注的偏移去改主题文字 —— 搜得到却换不对，是半成品。
本轮补的就是路由这一半：搜得到也换得了，契约一个字不改。

## 判据

1. **`EditorSearch` 是索引制的**（`find` / `reveal` / `replace` / `replaceAll` /
   `clear`，全按命中下标说话）：命中表是内核私有的，外壳只见计数。
   所以 `TopicHit` 加一个 `field` 字段是内核内部事 —— 契约零改动，外壳零改动。
2. **两条写回通道都已存在**：topic 走 `setNodeTopic`（同步），note 走
   `reshapeNode`（异步，经 `operation` 主通道）—— 备注弹层落盘走的正是后者
   （0056 取舍 ②）。本轮没有发明新通道，只是按 `field` 分诊。
3. **信封侧同步 `patch` 补内核异步的空档**：`setNodeTopic` / `reshapeNode`
   的 DOM 更新是同步的、返回的 Promise 是异步的；`refind` 紧接着读信封，
   只改内核会让它读到旧文本（`patchTopic` 注释的既有理由，note 侧同款
   `patchNote`，`undefined` = 清空即移除沿 0056 纪律）。
4. **弹层草稿只在挂载时播种**（`useState(state.note)` 初值）：鼠标流有关闭自愈
   （外点 mousedown 关弹层），键盘流（查找栏 `reveal` 连跳）没有 —— 不换挂载点，
   跳到下一个节点时端出来的是上一个节点的陈旧草稿。

## 取舍

1. **`collectHits` 同一节点内 topic 先、note 后**：两段偏移各以自己那段文字为
   基准，塞进同一张表；导航顺序仍是文档顺序（先序遍历）。空串 note 不扫
   （`typeof === 'string' && !== ''`）—— 空串画不出来，也搜不出有意义的命中。
2. **`reveal` 到 note 命中就把弹层亮出来、到 topic 命中就收掉**：命中落在弹层
   里，不亮等于没定位到；旧弹层赖在屏上就是陈旧内容。只读下亮的是只读弹层
   （读不是编辑，0056 取舍 ⑥ 同款）。
3. **`replace` 按 `field` 分诊**：note 路径走 `patchNote` + `reshapeNode`，替换
   到空串 = 清空即移除（`undefined` 干净丢键，0056 纪律）；写回后**必须关弹层**
   —— 它的草稿是挂载时播种的，note 被改后留着就是陈旧输入框。topic 路径一行不动。
4. **`replaceAll` 按「字段 + 节点」分段键控**（`${field}:${id}`），组内倒序：
   topic 与 note 是两段独立文本，混作一段会互相踩偏移；倒序守的还是那条旧纪律
   （同段多命中，先换前面的会让后面的下标失效）。返回值按**命中数**报 ——
   一条 topic 和一条 note 是两处替换，旧版「数节点」在分家后说不通。
5. **`NotePopover` 挂 `key={nodeId}`**：判据 4 —— React 靠 key 换挂载点，
   键盘流切节点时草稿重新播种。鼠标流不受影响（外点照样关）。
6. **大纲与导出不伺候**：大纲是导航不是阅读器（0056 取舍 ④ 照旧，只取 topic）；
   导出画的是文件内容、徽章与弹层是编辑器部件（0056 取舍 ⑤ 照旧）。

## 后果

- **测到的**：`MindmapEditor.test.tsx` 新增 `查找也搜备注` describe 5 条 ——
  计数（topic 与 note 各一处、纯 note 词一处）、顺序（同节点 topic 在前）与
  reveal 开/关弹层、单条替换改备注段 + 关弹层 + 重找计数、整条备注换成空串
  = `note` 键干净消失且 topic 不受牵连、`replaceAll` 报命中数（2）且两段都落地。
  全量 **87 文件 / 1,847 测试全绿**（+5）。
- **没测到的**：真 WebView 里从查找栏一路键盘跳 note 命中的观感（弹层位置、
  焦点归属）；`reveal` 在折叠节点上的行为沿 0052 旧例（`findEle` 看不见抛错
  那条是 `goToOutlineItem` 的，`focusHit` 走 `selectNode` + `scrollIntoView`
  同一条内核 API，折叠链路无新增语义）。
- **已知边界**：正则 `$1` 展开在 `replace` 单条路径沿代码内核旧例（捕获组逐条
  拿不到，整段替换才准）；`clear` 不动（只 empties 表，不碰弹层 —— 关弹层是
  写回路径的纪律，不是清空的）。
- **工程教训**：注释里写「为什么不搜」的理由过期了要改写 —— 0056 的旧注释
  「只搜 topic 因为内核不渲染 note」在本轮动手前已经是错话，错注释比没注释
  更能把后人带偏；`key` 是 React 里最便宜的「换状态」手段，弹层草稿这类
  「挂载播种」状态优先考虑 key 而不是补同步 effect。

## 重审条件

1. **换思维导图内核**：新内核若原生支持备注搜索（自带 UI 与通道），`field`
   分诊与弹层联动退役；届时「`replaceAll` 报数口径」跟着新契约重定。
2. **大纲要搜备注正文**：今天大纲只取 `topic`（取舍 6）；若导航要进备注，
   那是 `extractMindmapOutline` 的增补，另立 ADR，别顺手改这张命中表。
3. **备注要富文本**：0056 重审条件 4 照旧 —— 结构化富文本是格式层 breaking
   change，段内偏移的含义先变，这套分诊跟着重审。
4. **外壳要「只搜主题」开关**：今天无过滤（全搜）；若加开关，过滤点在
   `collectHits` 的调用侧（`find` 入口），不进命中表结构。
