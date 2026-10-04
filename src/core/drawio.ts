/**
 * `.drawio` 格式层的小助手 —— 只做「读 XML 能算出来的事」，不碰渲染。
 *
 * ## 为什么「缺哪些 stencil 库」要在这里算，而不是等着看请求
 *
 * mxGraph 取不到 stencil 时**不报错、不提示**，把形状画成普通矩形，而
 * 标签照旧画出来（ADR 0005 的 V1 实测结论 4 —— 实测还因此被骗过一次）。
 * 「事后看哪些请求 404 了」有两个问题：多张图并发渲染时请求没法归属到
 * 某一张图；XHR 打点又是一个全局性的猴子补丁。
 *
 * 而需要哪些库**写在 XML 里**：stencil 形状的风格串是
 * `shape=mxgraph.<库名>.<形状名>`（实测样本：`shape=mxgraph.bpmn.task`
 * → 取 `stencils/bpmn.xml`）。在渲染前把风格串扫一遍，与内置清单求差，
 * 就得到**这一张图**缺什么 —— 纯函数、逐图准确、无竞态、可直接单测。
 *
 * 刻意的局限：只认 `mxgraph.` 前缀这一种写法。它是 stencil 引用的主流形态，
 * 也是 V1 样本压过的形态；漏网的 exotic 写法会退回 mxGraph 的静默矩形 ——
 * 与内置查看器出现之前一样，不会更糟。
 */

/** 按库内置的 stencil 清单（`public/drawio/stencils/<名>.xml`）。 */
export const VENDORED_STENCIL_LIBS: readonly string[] = ['bpmn', 'flowchart'];

/**
 * 从一段 `.drawio` XML 里提取它引用的 stencil 库名。
 *
 * 只扫 `style` 属性里的 `shape=mxgraph.<库名>.` —— 一张图里同一个库引用
 * 多次只算一次，返回按出现顺序去重后的列表。
 */
export function stencilLibsOf(xml: string): string[] {
  const libs: string[] = [];
  const pattern = /shape=mxgraph\.([a-z0-9_-]+)\./gi;
  for (let match = pattern.exec(xml); match !== null; match = pattern.exec(xml)) {
    const lib = match[1]!;
    if (!libs.includes(lib)) libs.push(lib);
  }
  return libs;
}

/**
 * 这张图需要、但**没有内置**的库。空数组 = 渲染不会缺形状。
 *
 * 注意它回答的是「库层面缺不缺」，不是「每个形状都齐不齐」 ——
 * 一个库哪怕只内置了一部分，这里也按「有」算：细到形状的清单要跟着
 * 上游版本走，维护成本与收益不成比例（重审条件见 ADR 0027）。
 */
export function missingStencilLibs(xml: string): string[] {
  const needed = stencilLibsOf(xml);
  return needed.filter((lib) => !VENDORED_STENCIL_LIBS.includes(lib));
}
