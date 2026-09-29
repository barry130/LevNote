# 贡献指南

## 先读这三份

1. [`AGENTS.md`](AGENTS.md) —— 上手手册：架构、约定、容易踩的坑。
2. [`docs/decisions/`](docs/decisions/README.md) —— 为什么是现在这个样子。
   **每条 ADR 都带「重审条件」**，如果你认为某条决策错了，先看它的重审条件是否已经命中。
3. [`docs/open-decisions.md`](docs/open-decisions.md) —— 哪些事还没定、哪些验证还没做。

---

## 开发环境

```bash
npm install
npm run desktop      # 真桌面版（需要 Rust ≥ 1.77）
npm run dev          # 只跑前端（浏览器预览，内存虚拟盘）
```

提交之前**必须**跑：

```bash
npm run check        # = npm run typecheck && npm run lint && npm run test
```

CI 会跑同样三条命令加一次 `vite build`。**任何一条不过都不合**。

### 关于 lint

`npm run lint` 是**阻塞**的：`npm run check` 与 CI 都会因它失败。
它曾经以「非阻塞」引入（首轮有 33 条历史问题），欠账清零后转正 ——
「有规则但不执行」比「没有规则」更糟，因为它让人以为已经守住了。

配置的取向是**只开能抓到真问题的规则**，不开风格规则（缩进、引号、分号交给
`.editorconfig`）。最重要的三条是 `no-floating-promises`
（异步调用没接住 → 保存失败被吞掉，而用户以为存上了）、`no-misused-promises`、
`eqeqeq`。

**要加规则之前先问一句：它抓的是哪一类真 bug？** 答不上来就先别加。
lint 一旦变成噪声源就会开始被忽略，那比没有 lint 更糟。

**回调一律用属性签名。** 这条约定来自 `@typescript-eslint/unbound-method`：

```ts
interface Props { onClose(id: number): void }        // 方法简写 → 解构时报 unbound-method
interface Props { onClose: (id: number) => void }    // 属性签名 → 不会，而且这才是 React 的惯用写法
```

**返回实例上的方法**（`EditorHandle`、`MarkdownEditorInstance`）保持方法简写 ——
它们只会以 `instance.foo()` 的形式调用，不会被拆下来单传。

为什么较真这个：属性签名下参数是**逆变**的（方法签名是**双变**的），
所以回调的参数类型会被真正检查，而不是「长得差不多就行」。

### 关于 TypeScript 的两个大版本

`package.json` 里有两行看起来奇怪的依赖，它们是有意的：

```jsonc
"typescript":         "npm:@typescript/typescript6@^6.0.2",  // tsc 与 lint 都用它
"@typescript/native": "npm:typescript@^7.0.2"                // TS 7 仍在，需要时显式调用
```

原因是 `typescript-eslint@8` 的 peer 范围是 `>=4.8.4 <6.1.0`，
在 TS 7 上**导入时就抛错**（跟踪 issue：typescript-eslint#10940）。
让 `tsc` 跑 TS 6 是为了拿到类型感知的规则（尤其是 `no-floating-promises`，
它是这类应用最容易出真 bug 的一类）。
**重审条件写在 `eslint.config.js` 的头部注释里** —— 等 typescript-eslint 支持 TS ≥ 7.1，
就换回去。

---

## 代码约定

### 依赖方向是单向的

```
modules/（内核）  →  core/editor/types.ts（契约）  ←  ui/（外壳）
```

- 外壳**不许** import 任何具体内核。
- 内核**不许**直接碰文件系统（那是 `DocumentProvider` 的职责）。
- 内核与 UI **不许**直接 import `@tauri-apps/*`（只有 `core/platform/desktop.ts` 可以）。

这条是硬约束。破坏它会让「加一个新内核」变成一件要改遍全仓库的事。

### 注释写「为什么」，不写「是什么」

代码里已有的注释密度是**有意的**，不要以「太啰嗦」为由删掉。判断标准：
**这条注释是否记录了某次踩坑、某个反直觉的取舍、或某个「换个写法就会坏」的约束？**
是 → 留下并写得更好；否 → 可以删。

反面例子（该删）：`// 遍历所有标签页`。
正面例子（该留）：`// 优先接管右边的标签页，没有就接管左边 —— 与主流编辑器的行为一致`。

### 测试

- 纯逻辑放 `core/`，配 `*.test.ts`。**core 里的每个模块都该有测试**。
- 测试的注释同样写「为什么测这条」：如果这条断言曾经拦下过一个真实 bug，写出来。
- 涉及 DOM 的测试在文件顶部加 `// @vitest-environment jsdom`。

### 错误处理

- 需要用户决策的事（冲突、未保存改动）**不抛错**，返回一个状态让 UI 去问。
- 只在「程序错误」时抛错，并让它在开发期就炸出来（例如快捷键写法非法）。
- **绝不静默失败**：任何降级都要在 UI 上说出来（「N 张图片读不到，只保留了原链接」）。

---

## 提交与 PR

### 提交信息

```
<范围>: <做了什么>

<为什么这么做 / 排除了什么方案>
```

范围用目录名：`workspace` / `markdown` / `code` / `export` / `commands` / `ui` / `tauri` / `docs`。

例：

```
export: 图片读不到时保留原链接并打标记，而不是丢掉这张图

静默丢图会让用户以为导出成功了，发出去对方看到的却是破图。
标注出来让用户自己决定，比替他决定好。
```

### PR 要求

- **一个 PR 一件事。** 「顺手修了个 bug」请单独开一个 PR。
- 引入新依赖时说明：许可（必须允许商业分发）、体积影响、维护状态。
  **GPL 系的依赖一律不接受**（见 `docs/decisions/0004-editor-cores.md`）。
- 有长期影响的取舍请新增一条 ADR（`docs/decisions/NNNN-*.md`），
  并更新 `docs/decisions/README.md` 的索引。**ADR 必须写「重审条件」。**

---

## 明确不接受的做法

| 做法 | 原因 |
|---|---|
| 参考 Typora 的代码 | 闭源商业软件。只做**功能对标** |
| 参考 Notepad++ 的代码 | GPLv3。任何代码复用都会把整个项目拖入 GPL |
| 使用 draw.io 的名称与图标作为产品标识 | 商标约定（见 `docs/decisions/0005`） |
| 把 draw.io 的图标集用作 AI 训练素材 | 上游明确禁止 |
| 静默转码 / 静默覆盖 / 静默丢内容 | 见 ADR 0003 与 `docs/format-spec.md` §8 |
