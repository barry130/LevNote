# 0039 · 专注模式与打字机模式：两个视图偏好，一个状态对象

日期：2026-10-09
状态：已采纳（补记：功能随改名那次提交 `2224a48` 一起进来，本 ADR 是补上的决策记录）
关联：0021（位置记忆：滚动容器归内核所有）、0012（命令注册表）、
0014（源码视图复用代码内核）、0034（面板常驻：切回即离开时的样子）、
`core/editor/position.ts` 的 `typewriterScrollTop`、
`src/core/editor/types.ts` 的 `EditorMountProps.focusMode` / `typewriterMode`

## 背景

`ROADMAP.md` 早期把「大纲面板 / 专注模式 / 打字机模式 / PDF 导出」四项一起推到
P2 之后（见 [`0008-scope-and-order.md`](./0008-scope-and-order.md) 的推迟表）。
前三项后来陆续补齐，但这两个视图偏好是**随改名那次提交一起进来的** ——
`git log -S focusMode` 显示 `2224a48~1` 里一个匹配都没有。也就是说功能有、
取舍没记，欠了 `AGENTS.md` §五-2 要求的那条 ADR。这里补上。

它确实是有取舍的，不是「一个 `opacity` 加一个 `scrollTop`」：
两个开关要不要合并状态、切换时能不能重建内核、装饰走增量还是现算、
滚动算术放在哪一份、浓淡归谁的样式表 —— 五处都有第二个选项，都被否掉了。

## 决策

### 1. 两个开关，Markdown 侧**同一个插件状态对象**

`composePlugin`（`src/modules/markdown/plugin.ts:729`）一个插件管两件事，
状态是一个 `ComposeMode { focus, typewriter }`（`:690-693`），
key 是 `composeKey`（`:695`）。

**否掉的选项是拆成两个插件。** 理由：两个开关由同一处外壳偏好推来、
用同一个 meta 事务切换，拆开只会得到**两次 dispatch 与两份要同步的真相**。

代码内核侧只实现了打字机，于是状态就是一个普通闭包对象
`const typewriter = { on: options.typewriter ?? false }`（`src/modules/code/setup.ts:318`），
由 `setTypewriter`（`:469`）写、`updateListener`（`:349`）读。
**刻意不过 Compartment**：它不改编辑行为，只改一个布尔，
`Compartment.reconfigure` 是为「要换一整套 extension」准备的。

### 2. 切换偏好走原位事务，**不重建内核**

`setComposeMode`（`src/modules/markdown/setup.ts:1914-1924`）派发一条只带 meta 的事务：

```ts
view.dispatch(
  view.state.tr
    // 只改插件状态：文档不变（不会变脏、不上报），也不该是一个撤销步骤
    .setMeta(composeKey, mode)
    .setMeta('addToHistory', false),
);
```

`addToHistory: false` 不是洁癖：否则用户按一次 Ctrl+Z 撤销的是「我刚才开了专注模式」。

**重建内核是最顺手的写法，也是错的** —— 会丢光标、滚动位置与撤销栈
（与 ADR 0034 「切回即离开时的样子」同一条纪律）。代价是契约变了：
`EditorMountProps.focusMode` / `typewriterMode`（`src/core/editor/types.ts:946-963`）
在注释里明确写了「实现它的内核必须支持挂载后的实时开关」。
外壳侧靠 `MarkdownEditor.tsx:91-95` 那个 effect 兑现：内核还没建好时安静跳过，
创建时读的 `composeRef` 已经是当前值。

### 3. 专注：装饰**每次现算**，不走 `DecorationSet.map` 增量

`decorations` 里 `doc.forEach` 遍历顶层子节点，
「活跃」= 光标所在块（空选区）或与选区相交的块（非空选区），
其余挂 `DIMMED_CLASS = 'qm-dimmed'`（`:740-757`）。

**否掉的选项是增量簿记。** 顶层块的生老病死本来就少，
长文档也就几百个孩子，`map` 的簿记成本反而更贵 ——
这是「量一下再选」的结论，不是偏好。

### 4. 打字机：**只写 `scrollTop`，绝不 dispatch**

`view.update` 里选区一动（或开关刚打开）就把光标行居中（`plugin.ts:759-769`）。

**dispatch 是这里唯一的死循环**：dispatch → `onPositionChange` → 位置记忆写盘
→ 恢复滚动 → 又是一次滚动事件。所以代码内核那侧同样只在
`if (!typewriter.on) return; if (!update.selectionSet && !update.docChanged) return;`
之后调 `centerCaret`（`setup.ts:349-353`）。

`setTypewriter` 打开时**立刻居中一次**（`:471-472`），而不是等下一次光标移动 ——
否则用户按下开关的第一眼看到的是没反应的界面。

### 5. 滚动算术只有一份，在 `core/editor/position.ts`

`typewriterScrollTop(caretViewportTop, scrollerViewportTop, currentScrollTop, clientHeight): number | null`
（`position.ts:64-78`）是四个普通数的纯函数，返回 `null` 表示**不用动**。
三种 null：几何不是有限数、`clientHeight <= 0`（还没布局：jsdom 或首帧）、
`|delta| < 1`（已经在正中）。

**差一行的算术必须只有一份** —— 与 `applyScrollRatio` / `readScrollRatio` 同处一个文件，
两个内核共用。内核自己取几何：Markdown 拿 `.qm-editor`（`rootCtx` 那个元素，
与位置记忆读的是同一个容器，硬约束 13 的滚动条款），代码内核拿 `.cm-scroller`。

`< 1px` 那个阈值不是美观问题：**每敲一个字都重设一次 `scrollTop`
会跟浏览器的滚动合成器打架，也会吵醒位置记忆**。

### 6. 浓淡归内核的样式表（硬约束 13）

```css
/* src/modules/markdown/markdown.css:631-634 */
.qm-editor .qm-dimmed { opacity: 0.35; transition: opacity 0.2s ease; }
```

浓淡是**内核文档内容**的视觉，归内核所有；外壳只声明 flex / min-height。
`0.35` 是实测出来的：再淡就分不清「被调淡」和「被隐藏」，
再浓就起不到聚焦作用。过渡只过渡 `opacity`，**布局属性一律不过渡**。

### 7. 生效范围不对称，且写在设置页的提示里

**专注只在 Markdown 生效**（纯文本没有「段落」可言，导图没有线性光标）；
**打字机在 Markdown 与代码 / 文本生效**。两条都同时进中英文目录
（`src/core/i18n/messages.ts:284,286` / `:965,967`）。

范围不对称本身不是缺陷，**不对称却不说是** —— 用户在导图里点了专注模式，
看到「什么都没发生」会以为是坏了。所以提示句里直接写清适用范围，
这比在 ADR 里写一遍有用。

### 8. 刻意不给默认快捷键

同组的 `view.toggleWordWrap` / `view.toggleCodeLineNumbers` 也都没有
（对照 `view.toggleTheme` 的 `Mod+Shift+T`，`definitions.ts:626`）。
入口是视图菜单（`MenuBar.tsx:141-142`）、设置页（`SettingsDialog.tsx:324-333`）
与命令面板（`keywords` 带「专注 / 打字机」）。
菜单项**不按内核禁用** —— 与「行号」同一先例（注释在 `definitions.ts:648-651`）：
偏好是全局的，当前文档用不上时切了也没害处。

## 测试边界

`src/modules/markdown/compose.test.ts` 只钉**插件自己的契约**：
默认关 → 没有装饰；光标在第二段 → 第一、三段变淡（`to = [5, 15]`）；
移动光标后亮块跟着走；非空选区跨两块时两块都亮；关掉后装饰清空；
初始状态可以带开。

**打字机那半边在 jsdom 里天然是空操作**（没有布局，高度恒 0，
`typewriterScrollTop` 返回 null），能测的是状态进得去、且**开着不改文档**。
滚动算术本身在 `position.test.ts:81-117` 有自己的单测。

这不是测试没写够，是**只有人能做的**那一类里的一项 ——
`docs/release-smoke.md` 的名单应该点名它。

## 后果

- 偏好层两个字段（`settings.ts:130,138`，默认 `false`，`asBoolean` 强转 `:276-277`）
  与其它设置项同一条数据流；命令只翻偏好，不碰内核。
- `view.toggleFocusMode` / `view.toggleTypewriterMode` 的勾选态由
  `App.tsx:1195-1198` 从设置快照读（**只画状态，不产生语义**，同菜单三态那条纪律）。
- 新增内核时：**要么支持挂载后的实时开关，要么在契约注释里写明不支持** ——
  别让外壳那个 effect 静默空转。
- 专注模式的判定只看**顶层**块。嵌套结构（列表、引用、代码块）里，
  只要顶层块亮着，里面就全亮 —— 这是刻意的：按顶层走才能让「写到哪里亮到哪里」
  与位置记忆的光标口径一致。

## 重审条件

- **专注要不要推广到代码 / 纯文本**：CM6 有行装饰，纯文本两者都没有。
  真有需求时这是一条**新决策**（按行还是按段落、代码块内怎么办），
  不是本 ADR 的延伸。
- **长文档性能**：现算是 O(顶层块数)。若它在输入延迟剖面里显形
  （ADR 0004 的 V3 判据：1 万字 p50 ≤ 50ms），改走增量 `DecorationSet.map` +
  块索引。触发条件是**实测**，不是猜。
- **打字机 vs 位置记忆**：两者都写滚动，**目前位置恢复赢**
  （恢复发生在挂载那一刻，在 typewriter 的 updateListener 之后）。
  若实测发现切回标签时居中被打断，把这条反过来写进本 ADR。
- **jsdom 测不了的那一半**：若 CI 接上带布局的浏览器测试（Playwright 一类），
  打字机的滚动行为就该有真测试 —— 现在只有算术被钉住。
- **给这两个命令加默认快捷键**：记下是哪个键、为什么，
  以及与内核 keymap 是否冲突（撤销/重做那条前例见 ADR 0037）。