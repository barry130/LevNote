/**
 * 造 V1 用的 `.drawio` 样本。
 *
 * 每个样本压的是**一种真实的变体**，而不是「随便一个文件」——
 * §7.2 的硬要求里写着「压缩变体要支持」「多页必须保留」，而这两条只有在
 * 变体被真的造出来之后才谈得上验证。
 *
 * ## 变体表是从查看器源码读出来的，不是想出来的
 *
 * 判据在 `viewer.min.js` 的 `Editor.parseDiagramNode`：
 *
 * ```js
 * Editor.parseDiagramNode = function (node, d, l) {
 *   Editor.validateDiagramNode(node);
 *   var text = mxUtils.trim(mxUtils.getTextContent(node)), model = null;
 *   if (text.length > 0) {                      // ← ① 有文本 → 一律当压缩数据
 *     d = Graph.decompress(text, null, d);
 *     if (d != null && d.length > 0) model = mxUtils.parseXml(d).documentElement;
 *   } else {                                    // ← ② 没文本 → 取第一个子元素当模型
 *     var kids = mxUtils.getChildNodes(node);
 *     if (kids.length > 0) { model = kids[0]; ... }
 *   }
 *   return model;
 * };
 * ```
 *
 * 压缩那一路的编码顺序是**从源码读出来的**（`Graph.compress` / `Graph.decompress`），不是猜的：
 * `decompress` = `base64 → inflateRaw → decodeURIComponent`，所以压缩方向就是它的逆序。
 * 注意 `decompress` 的第二个参数决定 `inflate` 还是 `inflateRaw`，
 * 而 `parseDiagramNode` 传的是 `null` → **必须是 `deflateRaw`**（zlib 头的那种读不了）。
 *
 * ## 一条被证伪的变体（留着当证据，别再发明一遍）
 *
 * 变体表里原本有一行「未压缩 · 转义文本」——`<diagram>` 里放 `&lt;mxGraphModel …&gt;`。
 * **实测读不了**，报 `Failed to execute 'atob' on 'Window': The string to be decoded is not correctly encoded`。
 *
 * 根因就是上面那段源码：`<diagram>` 只要**有文本内容**，就**无条件**走 `Graph.decompress`，
 * 而 `Graph.decompress` 第一句是 `b = Graph.stringToArrayBuffer(atob(b))` —— **没有保护**，
 * 也没有「这段文本是不是已经是裸 XML」的分支。于是转义文本进 `atob` 必抛。
 *
 * 反向也证过：`viewer.min.js` 里**没有任何路径会把转义文本写进 `<diagram>`**
 * （全文搜 `"<diagram` 只有一处，是 Firefox 拖拽的一个 hack），`Graph.compress` 默认就是
 * `deflateRaw`。**所以这不是「查看器不支持某版本」，而是这种形式根本不是 `.drawio` 的合法变体。**
 * （转义形式真实存在的地方是别处：HTML 的 `data-mxgraph` 属性、SVG 的 `content` 属性、
 * PNG 的 `zTXt` 块 —— 那些走的是另外的抽取函数，不经过 `parseDiagramNode`。）
 *
 * 它现在留在样本表里，是作为**负样本**：期望是「明确报错」，而不是「画出一张空白画布」。
 * 这条对 P5 直接有用 —— 读不了的文件必须说出来（硬约束 2「绝不静默」）。
 */
import { writeFileSync } from 'node:fs';
import { deflateRawSync } from 'node:zlib';

/** 一页的图模型。`items` 里每个元素的 `label` 会被断言用来判「画对了没有」。 */
function graphModel(items) {
  const cells = [
    '<mxCell id="0" />',
    '<mxCell id="1" parent="0" />',
  ];
  for (const item of items) {
    if (item.edge === true) {
      cells.push(
        `<mxCell id="${item.id}" style="edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;" edge="1" parent="1" source="${item.from}" target="${item.to}">` +
          '<mxGeometry relative="1" as="geometry" /></mxCell>',
      );
      continue;
    }
    const style =
      item.shape === undefined
        ? 'rounded=0;whiteSpace=wrap;html=1;'
        : `shape=${item.shape};whiteSpace=wrap;html=1;`;
    cells.push(
      `<mxCell id="${item.id}" value="${item.label}" style="${style}" vertex="1" parent="1">` +
        `<mxGeometry x="${item.x}" y="${item.y}" width="${item.width}" height="${item.height}" as="geometry" /></mxCell>`,
    );
  }
  return (
    '<mxGraphModel dx="800" dy="600" grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" ' +
    'fold="1" page="1" pageScale="1" pageWidth="850" pageHeight="1100" math="0" shadow="0">' +
    `<root>${cells.join('')}</root></mxGraphModel>`
  );
}

/** 基础页：矩形 + 矩形 + 一条连线。只用 mxGraph 内置形状，**不需要任何 stencil**。 */
const basicPage = (tag) =>
  graphModel([
    { id: 'n1', label: `${tag}-A`, x: 120, y: 80, width: 160, height: 60 },
    { id: 'n2', label: `${tag}-B`, x: 120, y: 220, width: 160, height: 60 },
    { id: 'e1', edge: true, from: 'n1', to: 'n2' },
  ]);

/** 需要 stencil 的页：BPMN 与 flowchart 的形状**不在** mxGraph 核心里，得去 STENCIL_PATH / SHAPES_PATH 取。 */
const stencilPage = () =>
  graphModel([
    { id: 'n1', label: 'BPMN-TASK', x: 120, y: 80, width: 160, height: 80, shape: 'mxgraph.bpmn.task' },
    { id: 'n2', label: 'FLOW-PROCESS', x: 120, y: 220, width: 160, height: 80, shape: 'mxgraph.flowchart.process' },
    { id: 'e1', edge: true, from: 'n1', to: 'n2' },
  ]);

/**
 * **对照页**：与 `stencilPage` 的几何、标签位置、连线**完全相同**，只是顶点用的是默认矩形样式。
 *
 * 存在的理由只有一条：判「stencil 到底有没有起作用」**只能靠对照**。
 * mxGraph 取不到 stencil 时会安静回落成普通矩形，而标签照样画出来 ——
 * 单看 `stencil-required` 这一个样本，「画对了」与「回落了」都表现为「画出了两张图」。
 * 把两者的渲染结构一比，回落就等于「与对照一模一样」。
 */
const plainPage = () =>
  graphModel([
    { id: 'n1', label: 'PLAIN-TASK', x: 120, y: 80, width: 160, height: 80 },
    { id: 'n2', label: 'PLAIN-PROC', x: 120, y: 220, width: 160, height: 80 },
    { id: 'e1', edge: true, from: 'n1', to: 'n2' },
  ]);

const escapeXml = (text) =>
  text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

const escapeText = (text) => escapeXml(text).replace(/'/g, '&#39;').replace(/\n/g, '&#xa;');

/** 压缩一路：`base64(deflateRaw(encodeURIComponent(xml)))`。必须是 **Raw**，见文件头注释。 */
const compress = (xml) => deflateRawSync(Buffer.from(encodeURIComponent(xml), 'utf8')).toString('base64');

const HEADER =
  '<mxfile host="app.diagrams.net" modified="2026-10-02T00:00:00.000Z" agent="levnote-v1-samples" version="24.7.17" type="device">';

/**
 * 拼一份 `.drawio`。
 *
 * `pages[i].mode`：
 *   - `'child'`   未压缩，`<mxGraphModel>` 作为子元素（drawio 关掉压缩后的写法）
 *   - `'deflate'` 压缩（drawio 的默认保存格式）
 *   - `'empty'`   `<diagram>` 里什么都没有 —— 真实存在（新建页但没画东西）
 *   - `'text'`    **非法变体**，只给负样本用，见文件头
 */
function buildFile(pages) {
  const body = pages
    .map((page) => {
      const open = `<diagram id="${page.id}" name="${page.name}">`;
      if (page.mode === 'deflate') return `${open}${compress(page.xml)}</diagram>`;
      if (page.mode === 'text') return `${open}${escapeText(page.xml)}</diagram>`;
      if (page.mode === 'empty') return `${open}</diagram>`;
      return `${open}${page.xml}</diagram>`;
    })
    .join('');
  return `${HEADER}${body}</mxfile>`;
}

/** 返回 `Map<文件名, 内容>`。写盘交给调用方（测试与脚本要的落点不同）。 */
export function buildSamples() {
  const threePages = (mode) => [
    { id: 'page-1', name: '第一页', xml: basicPage('P1'), mode },
    { id: 'page-2', name: '第二页', xml: basicPage('P2'), mode },
    { id: 'page-3', name: '第三页', xml: basicPage('P3'), mode },
  ];

  return new Map([
    // ① 未压缩 · 子元素：drawio 关掉压缩后最常见的写法
    ['child-uncompressed.drawio', buildFile(threePages('child'))],
    // ② 压缩：drawio 的默认保存格式
    ['compressed.drawio', buildFile(threePages('deflate'))],
    // ③ 混合：同一份文件里两种变体并存 —— 「按原变体压回去」这条要求的现实场景
    [
      'mixed.drawio',
      buildFile([
        { id: 'page-1', name: '第一页', xml: basicPage('P1'), mode: 'child' },
        { id: 'page-2', name: '第二页', xml: basicPage('P2'), mode: 'deflate' },
      ]),
    ],
    // ④ 空页：`validateDiagramNode` 会现造一个空模型顶上，所以它不该让整份文件读不了。
    //    对我们的意义是「**空页也必须被读进来并保留**」—— 丢了它 = 静默丢页（硬约束 2）。
    [
      'empty-page.drawio',
      buildFile([
        { id: 'page-1', name: '第一页', xml: basicPage('P1'), mode: 'child' },
        { id: 'page-2', name: '第二页', xml: null, mode: 'empty' },
      ]),
    ],
    // ⑤ 没有 `<mxfile>` 外壳，根直接是 `<mxGraphModel>`。
    //    这一行是**待验证**的：`extractGraphModel` 的末句显式接受 `mxGraphModel` 当根，
    //    但 `GraphViewer` 的页集合是 `getElementsByTagName('diagram')` → 会是 0 页。
    //    两个读数都可能对，跑一遍看它到底走哪条路。
    ['bare-model.drawio', basicPage('BM')],
    // ⑥ 需要 stencil：用来回答「离线能不能画」这个问题的对照样本
    [
      'stencil-required.drawio',
      buildFile([{ id: 'page-1', name: '第一页', xml: stencilPage(), mode: 'child' }]),
    ],
    // ⑥b 纯矩形对照：与 ⑥ 几何相同、不用 stencil。**它自己不是被测对象**，
    //     它的指纹是判 ⑥ 的基准（见 EXPECTED 里 ⑥ 的 `对照` 字段）。
    [
      'plain-rectangles.drawio',
      buildFile([{ id: 'page-1', name: '第一页', xml: plainPage(), mode: 'child' }]),
    ],
    // ⑦ **负样本**：证伪过的转义文本变体，见文件头。期望是「明确报错」而不是「空白画布」。
    [
      'escaped-text.drawio',
      buildFile([{ id: 'page-1', name: '第一页', xml: basicPage('P1'), mode: 'text' }]),
    ],
  ]);
}

/**
 * 每个样本「画对了」应该看到什么 —— 断言写在数据里，不散在测试代码里。
 *
 * 字段：
 *   - `pages`     期望页数（`null` = 不检查，用于 ⑤ 那种「根本没有 `<diagram>`」的形态）
 *   - `pageNames` 期望页名数组（`null` = 不检查）
 *   - `labels`    **按页号**列出该页必须出现的标签。**判据是这张表，不是「有没有 svg」** ——
 *                 「页数对」不等于「每页都画对了」，所以要逐页切过去读标签。
 *                 页号不必从 0 连续（⑤ 只有第 0 页）。
 *   - `throws`    负样本：期望渲染**失败**，且错误消息匹配这个正则
 *   - `对照`      另一个样本的名字。判据是「与那个样本比，结构必须变了」——
 *                 专治「标签对而形状是错的」这类假绿（见 ⑥ 的注释）
 */
export const EXPECTED = new Map([
  [
    'child-uncompressed.drawio',
    {
      pages: 3,
      pageNames: ['第一页', '第二页', '第三页'],
      labels: { 0: ['P1-A', 'P1-B'], 1: ['P2-A', 'P2-B'], 2: ['P3-A', 'P3-B'] },
    },
  ],
  [
    'compressed.drawio',
    {
      pages: 3,
      pageNames: ['第一页', '第二页', '第三页'],
      labels: { 0: ['P1-A', 'P1-B'], 1: ['P2-A', 'P2-B'], 2: ['P3-A', 'P3-B'] },
    },
  ],
  [
    'mixed.drawio',
    {
      pages: 2,
      pageNames: ['第一页', '第二页'],
      labels: { 0: ['P1-A', 'P1-B'], 1: ['P2-A', 'P2-B'] },
    },
  ],
  [
    'empty-page.drawio',
    {
      pages: 2,
      pageNames: ['第一页', '第二页'],
      // 第二页故意**没有**标签可查 —— 它必须存在、必须能切过去，但不该画出任何东西。
      // 「存在」这条由 pages / pageNames 两个读数担着。
      labels: { 0: ['P1-A', 'P1-B'] },
    },
  ],
  [
    'bare-model.drawio',
    {
      // 没有 `<mxfile>` → 查看器的页集合是 0。第 0 页的标签仍要能画出来。
      pages: null,
      pageNames: null,
      labels: { 0: ['BM-A', 'BM-B'] },
    },
  ],
  [
    'stencil-required.drawio',
    {
      pages: 1,
      pageNames: ['第一页'],
      // **这一行的真正判据不是标签**：mxGraph 取不到 stencil 时会安静回落成普通矩形，
      // 标签照样画出来 —— 「标签都在」会把「静默降级」判成通过。
      labels: { 0: ['BPMN-TASK', 'FLOW-PROCESS'] },
      // 判据是「与几何完全相同的纯矩形对照比，结构必须**变了**」，方向是
      // **矩形变少、路径变多**（stencil 的轮廓是用 `<path>` 画的，回落则是一个 `<rect>`）。
      //
      // 为什么不用「指纹不等于某个实测值」：那样写过一版，然后在**预加载
      // `stencils.min.js` 那一轮**被自己的判据骗了 —— 那一轮形状**仍然是回落的**
      // （rect 2 / path 3，与纯矩形一模一样），只因为 svg 总长度是 2827 而不是 2809
      // （差 18 个字符）就被判成通过。**「不等于」这种判据对「差一点点」毫无分辨力。**
      // 换成比结构，就没有一个会飘的量参与判定。
      对照: 'plain-rectangles.drawio',
    },
  ],
  [
    'plain-rectangles.drawio',
    {
      pages: 1,
      pageNames: ['第一页'],
      // 它不是被测对象，是 ⑥ 的基准 —— 只要它自己能画对（标签在）就够。
      labels: { 0: ['PLAIN-TASK', 'PLAIN-PROC'] },
    },
  ],
  [
    'escaped-text.drawio',
    {
      pages: null,
      pageNames: null,
      labels: {},
      throws: /not correctly encoded|atob/i,
    },
  ],
]);

// CLI：写盘到 --out 指定的目录
if (process.argv[1] !== undefined && process.argv[1].endsWith('v1-samples.mjs')) {
  const flagIndex = process.argv.indexOf('--out');
  const dir = flagIndex === -1 ? '.v1-out/samples' : process.argv[flagIndex + 1];
  const { mkdirSync } = await import('node:fs');
  const { resolve, dirname } = await import('node:path');
  mkdirSync(resolve(dir), { recursive: true });
  for (const [name, content] of buildSamples()) {
    const target = resolve(dir, name);
    mkdirSync(dirname(target), { recursive: true });
    writeFileSync(target, content, 'utf8');
    console.log(`${name}  ${content.length} 字符`);
  }
}
