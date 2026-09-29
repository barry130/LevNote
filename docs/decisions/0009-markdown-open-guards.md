# ADR 0009：Markdown 打开路径的两道防线

- 状态：已采纳
- 日期：2026-09-28
- 影响的决策：0004（Markdown 内核选型）、0008（范围与顺序）
- 关联代码：`src/modules/markdown/frontmatter.ts`、`src/modules/markdown/normalize-ast.ts`、`src/core/editor/registry.ts`

## 背景

P2 接入 Milkdown 的过程中，用真实内核跑语料时发现了**两个会静默毁文件**的问题。
两个都不是「显示得不好看」，而是「用户的文件被改了而他不知道」，属于本项目明令禁止的那一类。

### 问题一：无标题的图片会被整张丢弃

`![图](a.png)` 这种没有标题的图片，remark 给 mdast 节点填的是 `title: null`（不是 `undefined`）。
Milkdown 的 image 节点把 `title` 声明成 `{ default: "", validate: "string" }`，
而 `default` **只在值为 `undefined` 时才生效**，于是 `null` 直接进校验并抛：

```
RangeError: Expected value of type string for attribute title on type image, got null
```

后果是整张图片从文档里消失。用户的体感是：打开一篇带插图的笔记，随手改一个字保存，插图全没了。

实测**链接（link mark）的 `title` 同样传 `null`，但没有出问题**，所以这不是「remark 的锅」，
是 image 节点自己的 runner 少了一次 `?? ''`。

### 问题二：YAML front matter 会被拆成「分隔线 + 正文」

commonmark 不认识 front matter。开头的 `---` 会被读成**水平分隔线**，
`title: 我的笔记` 会被读成**普通段落**。用户没动过元数据，
只要编辑过一次再保存，元数据就变成了分隔线加正文 —— 而且不可逆。

对 Obsidian / Hugo / Jekyll 用户，这等于毁文件。这类文件在 `.md` 里占比很高。

## 决策

### 一、在 mdast 层规整 null，而不是改 schema

新增 `src/modules/markdown/normalize-ast.ts`，通过 `remarkPluginsCtx` 注册一个
transformer，在**数据进入内核之前**把「声明为字符串、实际是 `null`」的字段收敛成 `''`。

为什么不在 schema 层修：自定义 image 节点意味着要重写它的 `parseMarkdown` / `toMarkdown` /
全部命令，而 `preset-commonmark` 内部还引用着它自己那份 schema，两边会打架。
mdast 层是数据形状问题的正确归属地，而且链接一类节点一并受益。

只做形状规整，不做语义改写：缺标题的图片仍然是缺标题，序列化回 markdown 时
`![](a.png)` 的字节保持不变。

### 二、front matter 文件降级到纯文本，并说明原因

新增 `EditorAdapter.canHandle(path, text)` 这个可选的**二次确认**钩子：
`matches()` 只看扩展名（发生在读到内容之前），`canHandle()` 在拿到内容之后再判一次。
返回 `false` 就交给下一个候选，最终落到纯文本兜底。

Markdown 内核的 `canHandle` 在识别出 front matter 时返回 `false`，
于是这类文件**以纯文本打开**，内容一个字节都不会动，界面上给出一条说明为什么。

判定故意偏向保守（宁可误判）：三个条件同时成立才算 front matter ——
第一行恰好是 `---`、前 50 行内出现恰好是 `---` 或 `...` 的收尾行、
收尾行之前至少有一行形如 `键: 值`。

代价不对等是这么定阈值的理由：

| | 代价 |
|---|---|
| 误判（把普通文档当 front matter） | 这个文件用纯文本打开。内容安全、有提示、可解释，用户可接受 |
| 漏判（把 front matter 当普通文档） | 元数据被不可逆改写 |

### 三、`canHandle` 抛错按「否决」处理，不让异常冒出去

`resolveAdapter` 捕获 `canHandle` 的异常、打一条 warn、继续找下一个候选。

理由：「任何文件都必须能被打开」是比「用对内核」更硬的要求。
一个内核的判定逻辑写挂了，不该让用户打不开文件。

## 影响

- front matter 文件在 P2 里没有富文本视图。这是**有意的降级**，不是缺陷。
  保留 front matter 的富文本编辑是 P3 的候选任务（把它当作不透明前缀，
  编辑区只呈现正文，保存时原样拼回）—— 在那之前，纯文本是唯一不毁文件的选择。
- `.mdx` 也被排除在 Markdown 内核之外（JSX 会让 Milkdown 把 `import` / `<Component/>`
  当段落文本）。同样落到纯文本。
- 这两个问题的共同教训是：**「能不能解析」和「解析完还在不在」是两件事**。
  由此新增了 `markdown-core.test.ts` 这份广谱语料测试 ——
  未在语料里的构件就是未验证的，这一点写在测试文件头部。

## 未采纳的替代方案

- **在 schema 层给 image 加 `?? ''`**：需要覆盖 preset 的内部 schema，脆且影响面大。
- **打开时做「往返保真度自检」并弹警告**：实测序列化器的规范化是真实存在的
  （setext → ATX、行尾空白收敛、`~~~` → ```），正常文档也会触发，噪音太大。
  保真度改为用测试保证（`markdown-core.test.ts` 的「序列化器的规范化行为」一组），
  而不是运行时告警。
