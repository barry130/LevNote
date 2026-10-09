/**
 * Mermaid 图表的**纯逻辑**部分：什么算一个图表块。
 *
 * ## 为什么这个文件里没有一行 mermaid 的代码
 *
 * mermaid 是一个 3 MB 上下的包，而且它一被 import 就会去碰 `document`
 * （初始化时要建样式表、要读字体度量）。把它放进任何一个会被主包引用的模块图里，
 * 都会拖累每一个打开轻笺的人 —— 哪怕这篇文档一个图表都没有。
 *
 * 所以这里只放**判定**，渲染在 `core/mermaid-render.ts`（那个模块里唯一的
 * import 是 `import('mermaid')`，而且是**动态**的）。判定与渲染分开之后，
 * 判定的测试不需要加载 mermaid，也就不需要一个能跑 mermaid 的环境
 * （jsdom 没有布局引擎，`getBBox` 返回全 0，mermaid 在里面根本画不出图）。
 *
 * ## 为什么「什么算 mermaid 块」要单独成一个函数
 *
 * 这个判定出现在**三处**，而且三处的运行环境完全不同：
 *
 * - 编辑器的 NodeView（决定要不要画图）；
 * - 编辑器的 decoration 插件（决定光标在不在图表块里）；
 * - 导出侧（决定哪个 `<pre>` 要换成 SVG）。
 *
 * 三处各写一个 `language === 'mermaid'` 的话，上游哪天多认一个别名
 * （GitHub 就同时认 `mmd`），就会出现「编辑器画得出、导出不画」——
 * 而那不会报错，只会让发出去的文档少一张图。
 */

/** 围栏代码块上表示「这是 Mermaid」的语言标识。 */
export const MERMAID_LANGUAGE = 'mermaid';

/**
 * 这个语言是不是 Mermaid。
 *
 * ## 为什么要 trim + 忽略大小写
 *
 * 语言来自围栏后面的**任意文本**（```` ```Mermaid ```` 是完全合法的写法），
 * 而它会被原样存进 `code_block` 的 `language` 属性。只认小写精确匹配的后果是：
 * 用户手一抖写成 `Mermaid`，那一块就**永远只是一段代码** —— 而他不觉得自己写错了
 * （文档里老老实实写着 `Mermaid` 两个字）。
 *
 * 这一路的判据是「上游怎么存，我们就怎么认」：`remark` 的 `code` 节点
 * 不对 `lang` 做任何规范化，所以规范化只能是我们自己的事。
 *
 * ## 为什么**不**顺手认别的别名
 *
 * GitHub 认 `mmd`，VS Code 的一些插件认 `mermaidjs`。认了它们的代价是：
 * 用户写 ```` ```mmd ```` 时，轻笺会把那一块画成图，而**别人打开这份文档时
 * 它只是一段代码**（VS Code 默认就不渲染 `mmd`）。那是我们制造的分歧，
 * 不是上游的分歧。等哪天 mermaid 官方默认认别名了，这里再加。
 */
export function isMermaidLanguage(language: string): boolean {
  return language.trim().toLowerCase() === MERMAID_LANGUAGE;
}

/**
 * 插入图表时**预先填好**的那段源码。
 *
 * 为什么不插一个空块：空块会停在源码态（见 `code-block-view.ts` 的
 * `wantDiagram`），用户看到的是一个空框，不知道该写什么、也不知道写对了
 * 会长什么样。给一段**能立刻画出图**的例子，他点开就看见一张图，
 * 接着改那两行字就行 —— 「哦，这么写」比「请看文档」有用得多。
 *
 * 内容刻意只画两个节点：例子一长，用户就得先删掉一堆他没写过的东西。
 */
export const MERMAID_STARTER = 'graph LR\n  A[开始]-->B[结束]';
