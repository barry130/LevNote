/**
 * 示例文档（B1）：帮助菜单「打开示例文档」写入工作区的那份 Markdown。
 *
 * ## 它是什么、不是什么
 *
 * 它是**功能的导览**：四类一等公民里 Markdown 能直接演示的那部分
 * （格式、表格、代码块、数学、Mermaid、draw.io inline 图表）各来一段，
 * 并把最常用的入口（斜杠菜单、选中浮层、检索、导出）写在结尾。
 * 思维导图与 draw.io 独立文件没法在一份 Markdown 里「演示」，只在结尾指路。
 *
 * ## 两条纪律
 *
 * 1. **已存在就不再写**：编排（`help.sampleDocument` 命令）会先 stat 磁盘，
 *    命中就直接打开 —— 用户可能已经改过它，覆盖就是毁内容（硬约束 2）。
 *    想重置示例，删掉文件再点一次。
 * 2. **内容必须过一遍真实的读写链路**：列表符号用 `-`（写回时 `useDashSymbols`
 *    也是 `-`），打开 → 什么都没改 → 保存，不该把这份文档重排一遍；
 *    draw.io 块是 **inline 模式**（XML 就在文档里），不依赖第二个文件，
 *    id 在文档内唯一即可 —— 全文只有这一个 doc-block。
 */

export const SAMPLE_DOCUMENT_NAME = '轻墨示例.md';

export function buildSampleDocument(): string {
  return [
    '# 轻墨 · 示例文档',
    '',
    '欢迎。这份文档演示轻墨里 Markdown 的大部分能力 —— 直接动手改它，',
    '改坏了想重置：删除本文件，再从「帮助」菜单打开一次。',
    '',
    '## 基本格式',
    '',
    '这是**粗体**、*斜体*、~~删除线~~、`行内代码`，还有一个[链接](https://example.com)。',
    '选中任意文字会浮出格式工具条；在段落末尾输入 `/` 可以插入各类块。',
    '',
    '## 列表',
    '',
    '- 无序列表项',
    '  - 嵌套项',
    '- 另一项',
    '',
    '1. 有序列表项',
    '2. 第二项',
    '',
    '- [ ] 任务清单：未完成',
    '- [x] 任务清单：已完成',
    '',
    '> 引用块。引用可以说明出处、写批注，',
    '> 也可以**带格式**。',
    '',
    '## 表格',
    '',
    '| 功能 | 入口 | 备注 |',
    '| :--- | :---: | ---: |',
    '| 查找替换 | Ctrl+F | 当前文档内 |',
    '| 全文检索 | Ctrl+Shift+F | 整个工作区 |',
    '| 命令面板 | Ctrl+P | 所有命令都在里面 |',
    '',
    '## 代码块',
    '',
    '代码块带行号与语法高亮；右上角的语言标签可以点开换语言。',
    '',
    '```ts',
    'function greet(name: string): string {',
    "  return `你好，${name}！`;",
    '}',
    '',
    'console.log(greet("轻墨"));',
    '```',
    '',
    '## 数学公式',
    '',
    '行内的欧拉恒等式 $e^{i\\pi} + 1 = 0$，以及独立成块的积分：',
    '',
    '$$',
    '\\int_{0}^{\\infty} e^{-x^2} \\, dx = \\frac{\\sqrt{\\pi}}{2}',
    '$$',
    '',
    '## Mermaid 图表',
    '',
    '下面这张流程图由 Mermaid 渲染；点图可以进入全屏查看。',
    '',
    '```mermaid',
    'flowchart LR',
    '  A[打开文档] --> B{是 Markdown?}',
    '  B -- 是 --> C[富文本编辑]',
    '  B -- 否 --> D[代码 / 导图 / 兜底]',
    '  C --> E[Ctrl+S 保存]',
    '  D --> E',
    '```',
    '',
    '## draw.io 图表',
    '',
    'draw.io 图表以 doc-block 形式内嵌在 Markdown 里，下面的图就是渲染结果 ——',
    'XML 就在文档里，源码视图（Ctrl+/）可以直接改它。',
    '',
    '```doc-block',
    'id: sample-citizens',
    'kind: diagram',
    'mode: inline',
    'title: 同一个工作区里的四类一等公民',
    '---',
    '<mxfile><diagram name="页1"><mxGraphModel><root>',
    '<mxCell id="0" /><mxCell id="1" parent="0" />',
    '<mxCell id="2" value="同一个工作区" style="ellipse;whiteSpace=wrap;html=1;" vertex="1" parent="1"><mxGeometry x="100" y="100" width="180" height="60" as="geometry" /></mxCell>',
    '<mxCell id="3" value="Markdown" style="rounded=1;whiteSpace=wrap;html=1;" vertex="1" parent="1"><mxGeometry x="40" y="40" width="120" height="60" as="geometry" /></mxCell>',
    '<mxCell id="4" value="代码文本" style="rounded=1;whiteSpace=wrap;html=1;" vertex="1" parent="1"><mxGeometry x="220" y="40" width="120" height="60" as="geometry" /></mxCell>',
    '<mxCell id="5" value="思维导图" style="rounded=1;whiteSpace=wrap;html=1;" vertex="1" parent="1"><mxGeometry x="40" y="160" width="120" height="60" as="geometry" /></mxCell>',
    '<mxCell id="6" value="draw.io 图表" style="rounded=1;whiteSpace=wrap;html=1;" vertex="1" parent="1"><mxGeometry x="220" y="160" width="120" height="60" as="geometry" /></mxCell>',
    '</root></mxGraphModel></diagram></mxfile>',
    '```',
    '',
    '## 继续探索',
    '',
    '- 在空行输入 `/`：斜杠菜单能插入表格、代码块、数学块、图表与分隔线；',
    '- 选中文字：浮出的工具条处理加粗、斜体、链接等行内格式；',
    '- 状态栏的「源码 / 分屏」或 Ctrl+/：同一路径的源码视图；',
    '- 「文件 → 导出为自包含 HTML」：单文件带走全部样式与图片；',
    '- 左侧活动栏的检索图标：跨文件的全文检索。',
    '',
    '思维导图（`.mindmap`）与 draw.io 图表（`.drawio`）也是一等公民：',
    '在文件树里新建对应扩展名的文件即可试。',
    '',
  ].join('\n');
}
