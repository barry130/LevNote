# 0041 · 失败要有出口：命令的意外 throw 与拖入多文件

日期：2026-10-10
状态：已采纳
关联：0003（不静默，是本 ADR 的母题）、0012（命令是能力的唯一清单）、
0031（i18n：文案只有一份 key，命令标题是惰性 getter）、
0037（菜单只是命令的另一个入口 —— 入口越多，出口越不能各写一份）、
0040（链接激活：成功的分支安静、失败各自点名 —— 本 ADR 沿用同一取舍）
AI助手手册硬约束 2（绝不静默）

## 背景

0040 收尾时留下一个判断：**「清单全部完成」不等于「这一类问题没有了」。**
链接那一条是「用户看得见、点得着、点下去没反应」的一个实例，
而同一形态的东西在别处还有。这一轮把它们清点了一遍，剩下两类：

**候选 A —— 命令失败被 `console.warn` 吞掉。**
命令有四个入口（菜单、活动栏、全局快捷键、命令面板），四个原先都长这样：

```ts
void Promise.resolve(command.run(context)).catch((thrown: unknown) => {
  console.warn(`命令 ${id} 执行失败`, thrown);
});
```

`console.warn` 是**开发者看的**。用户那一侧的表现就是「点了一下，什么都没发生」，
而且最难自查 —— 应用不崩、界面不变、没有一句话。这正落在硬约束 2 上。

现成的例子就在仓库里：`view.toggleFullscreen` 在浏览器端走 Fullscreen API，
**没有用户手势时它会 reject**（浏览器规范如此）。而它被 `void` 掉了，
于是「F11 按下去没进全屏」这件事没有任何出口。

**候选 B —— 一次拖入多个文件，只打开第一个。**

```ts
const first = event.payload.paths[0];
if (first !== undefined) void workspace.open(first);
```

用户拖进来五个文件，界面只开一个，其余四个连「没开成」这句话都没有 ——
那是**静默丢掉用户的选择**，与候选 A 同一条红线。

顺带清点过但**刻意不动**的三类（它们不是「用户的动作没有反馈」）：

- `src/ui/App.tsx` 文件监听的 `console.warn`（后台自动重试，下次刷新会好）；
- `src/core/commands/registry.ts` 无效快捷键写法（是**用户数据**写错了，
  该由设置页的冲突提示去说，不是运行时弹一句）；
- `src/core/platform/desktop.ts` 消息框自身失败（已无处可去，见决策 1 的第二层）。

## 决策

### 1. 兜底抽成一个纯函数，四个入口共用一份写法

新增 `src/core/commands/runner.ts`：

```ts
export async function reportCommandFailure(command: Command, thrown: unknown, ctx: CommandContext): Promise<void>
export function runCommandSafely(command: Command, ctx: CommandContext): void
```

三个调用点（`App.tsx` 的 `runCommand`、全局快捷键 `onKeyDown`、
`CommandPalette`）原来**各自抄了一遍** `Promise.resolve(...).catch(console.warn)`，
抄三遍的结果就是修一处漏两处 —— 这次的起因正是如此。

抽成纯函数还有第二个理由：**它能被单测钉住**。「失败怎么变成一句话」只在这里
定义一次，三个入口各自只剩一行。

`runCommandSafely` 的形状是刻意这么写的：

```ts
let result: void | Promise<void>;
try {
  result = command.run(ctx);
} catch (thrown) {
  void reportCommandFailure(command, thrown, ctx);
  return;
}
void Promise.resolve(result).catch((thrown: unknown) => {
  void reportCommandFailure(command, thrown, ctx);
});
```

**不能写成 `void Promise.resolve(command.run(ctx)).catch(...)`** —— `run` 是在
**参数求值时**跑的，同步 throw 根本到不了后面的 `.catch`。三个调用点原先全这么写，
所以「同步 throw」那一路一直是漏的（快捷键那一路会直接冒进 React 的事件处理里）。
这一条是单测逼出来的：失败信息是
`Error: 同步炸了` at `src/core/commands/runner.test.ts:95`。

### 2. 提示里必须说出**哪个命令**和**为什么**

```ts
await ctx.ui.notify(
  t('core.cmd.commandFailedBody', { title: command.title, reason: error.message }),
  t('core.cmd.commandFailedTitle'),
);
```

只说「失败了」等于什么都没说 —— 用户要能自己判断「这是我该重试的，还是应用坏了」。
`command.title` 是**惰性 getter**（0031），现读才是对的，所以这里收 `Command`
而不是只收 id（不需要知道注册表在哪）。

`asAppError` 保证 `message` 永远是一句人话（不会是 `[object Object]`），
所以 `reason` 可以直接进文案；`detail` 只有开发者用得上，落进控制台那条。

### 3. 兜底层自己要有兜底，但**不再往上抛**

`notify` 自己也可能失败（桌面是原生对话框，浏览器是 `window.alert`）。
那一次失败**留在本层**：

```ts
catch (notifyError) {
  console.warn(`命令 ${command.id} 执行失败，且失败提示也没能弹出`, thrown, notifyError);
}
```

不再找第三个出口 —— 否则就是「为了报告失败而报告失败」的无限递归，
而调用方的 catch 会再次进来，表现是「点了没反应」外加一条未处理 Promise。
这也是 `console.warn` **不删**的理由：它是最后一层痕迹，但不再是唯一一层。

### 4. 它是**最后**一层，不是唯一一层

已经有七处命令**自己**处理了预期内的失败（导出失败、新建文件重名、
复制 HTML 被拒、示例文档写不进去 …），它们各自 `ctx.ui.notify` 一句说清原因。
这里管的是**它们没覆盖到的那部分**：意外的 throw。

命令已经给过提示时，这里再说一遍确实是两句（外部无法判断「这个 throw
是不是已经被报告过」）。取舍是**宁可多一句，不可没有一句**：重了至多是两条提示，
漏了是用户以为应用坏了。

**想让某条命令不落到这里，正确做法是在命令内部把失败接住并说明，
不是在兜底层加白名单。**

### 5. `toggleFullscreen` 的契约从 `void` 改成 `Promise<void>`

命令的 `run` 现在是 `run: (ctx) => ctx.ui.toggleFullscreen()` ——
**把 Promise 交回去，不 `void` 掉**。自己 `void` 掉就等于把这次失败
变成一条未处理 Promise，用户看到的还是「点了没反应」。

> 判断依据：**一个会失败的异步能力，它的类型不许是 `void`。**
> 类型上抹掉 Promise，等于在编译期就宣布「这个失败无处可去」。

### 6. 拖入多文件：全部打开，**只汇报没打开的那几个**

新增 `src/core/workspace/drop.ts`：

```ts
export interface DropOutcome { readonly opened: number; readonly failed: readonly string[] }
export async function openDroppedPaths(paths: readonly string[], workspace: WorkspaceStore): Promise<DropOutcome>
export function describeDropOutcome(outcome: DropOutcome): { readonly title: string; readonly body: string } | null
```

- **顺序打开，失败只记账、不中断** —— 头一个打不开不该连累后面几个；
- **已打开的标签页算成功**（`WorkspaceStore.open` 命中已有标签就激活并返回 `true`）；
- `describeDropOutcome` 在无失败时返回 `null` —— **成功时一句话都不说**（与 0040
  同一取舍：新标签页自己就能看见，再补一句是噪音）；
- 有失败时分两句：`opened === 0` 说「这 N 个都没能打开：…」，否则「这 N 个没能打开：…」；
  名单用 `basename` 拼 —— 用户拖的是文件，他认的是文件名，不是绝对路径。

**否掉的选项是「并行 `Promise.all` 打开」**：工作区的打开有顺序语义
（`revealInTree`、激活哪一个），并行会让「最后停在哪个标签页」变成竞态。

## 后果

- **两个新模块**：`src/core/commands/runner.ts`（+6 条单测）、
  `src/core/workspace/drop.ts`（+8 条单测）。
- **五个新文案 key**（zh + en）：`core.cmd.commandFailedTitle` /
  `commandFailedBody` / `dropSomeFailed` / `dropAllFailed` / `dropTitle`。
- **契约变了一处**：`CommandContext.UiActions.toggleFullscreen(): Promise<void>`。
- **外壳新增一组 4 条**（`src/ui/App.test.tsx` 的「命令失败与拖放的出口」）：
  F11 失败有一句话；三个全开成功**全程安静**；三个里坏一个说出名字且另外两个照开；
  两个全坏说「都没能打开」并列出两个名字。
- **测试侧的一条新知识**：jsdom **没有 Fullscreen API**（`requestFullscreen` 是
  undefined），所以 F11 在测试里**真的会失败** —— 那正是一个不用 stub 的现成入口。
  反过来说：**「全屏能不能开」这件事单测证明不了**，已点名进 `docs/release-smoke.md`。
- **拖放用例必须走 `@tauri-apps/api/mocks`**，且收尾有两处坑：
  ① `clearMocks()` 只删对象**里面的字段**，`window.__TAURI_INTERNALS__` 这个空壳
  会留下，而 `isDesktop()` 判的是 `'__TAURI_INTERNALS__' in window` —— 不把键整个
  拿掉，后面每个用例都会被当成桌面环境；
  ② **必须先卸载再 `clearMocks()`**：卸载走 effect cleanup，里面要 `unlisten()`，
  而它读 `window.__TAURI_EVENT_PLUGIN_INTERNALS__.unregisterListener`，
  晚了就是一片 `Cannot read properties of undefined (reading 'unregisterListener')`。

## 重审条件

- **出现「一句话说不清」的失败时**：若某类命令的失败需要一段解释或一处操作入口
  （比如「保存失败要不要另存为」），兜底层就不该管它 —— 在命令内部接住并给出
  那次对话。**不要**给 `reportCommandFailure` 加参数去定制文案：它有四个入口，
  定制文案会让「命令怎么失败」重新散回各处。
- **同一条命令被报告两次变成一个真实的抱怨时**：本 ADR 的取舍是「宁可多一句」。
  真到那一步，正确的解法是给命令加一个「我已报告过」的约定
  （比如允许 `run` 抛一种已报告过的错误类型），而不是在兜底层加白名单。
- **拖入的文件数量变得很大时**（几十个以上）：顺序打开会明显变慢，
  那时才需要评估并发与进度 —— 但「全部打开、只汇报失败的」这条不变。
- **`describeDropOutcome` 开始要区分失败原因时**（只读 / 编码 / 不存在）：
  现在是「名字清单」一句话。原因多了就该改成按原因分组，
  而不是往一个名单里塞更多字。
- **若将来 `console.warn` 那条也要求有出口**（比如用户主动开了「诊断模式」）：
  那是另一条链路（日志面板），**不要**顺手把 `reportCommandFailure` 接上去 ——
  它的出口是「此刻弹一句」，不是「记一笔」。
