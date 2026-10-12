/**
 * draw.io 图纸内核（`.drawio` 的首选内核，ADR 0048）。
 *
 * ## 这个内核现在**只能查看** —— 而且这件事写在面板上
 *
 * 随包分发的是 draw.io 的**只读查看器**（`public/drawio/viewer.min.js`）。
 * 「可编辑」需要另一份产物（webapp 本体），本轮拿不到 —— 不是没做，是没有资产：
 *
 * - 沙箱无出网，装不上也下载不了；
 * - 本机没有 draw.io 安装；
 * - **实测过「拿查看器当编辑器用」这条路**（`scripts/v4-run.mjs`）：那份产物里
 *   `EditorUi` 确实在，但它构造时要的 `Editor` 类被一个替身顶掉了 ——
 *   `Editor=function(){return new mxCellEditor(this)}`（`viewer.min.js` 第 556907 字节处）。
 *   替身造出来的是 mxGraph 的**单元格编辑器**，于是 `EditorUi` 第一次访问
 *   `this.editor.graph` 就炸，`App.MODE_EMBED` / `EmbedFile` 也不在产物里。
 *   这不是引导代码写错能救的，**这份产物跑不起 draw.io 的编辑器**。
 *
 * 所以走另一条同样诚实的路：**`.drawio` 成为独立的一等公民，能画、能看、能进
 * 源码视图改 XML**，而「现在不能直接拖图编辑」这句话就摆在面板顶部。
 *
 * ## 为什么不给适配器加 `canHandle`（不像导图那样否决）
 *
 * 导图要否决，是因为「坏掉的 `.mindmap` 硬接下来 = 一个空白的导图画布」，
 * 用户连内容在哪都找不到（理由见 `modules/mindmap/index.ts`）。
 * `.drawio` 没有这个问题：**查看器自己会把「不是合法的 XML」写进元素**，
 * `renderDrawio` 把它读出来当失败理由交回（`DrawioRenderFail.message`）。
 * 于是「这份图纸有问题」由渲染器如实说出来，比我们在适配层再判一遍更准
 * —— 也省掉一个 `KernelRejectReason` 的新 kind（ADR 0043 的重审条件卡在 5 个，
 * 不为了一个能由渲染器说清的理由去顶那张表）。
 *
 * ## `getValue()` 交回「最后交到内核的那份原文」
 *
 * 这个内核**从不调用 `onChange`**，所以文档永远不会变脏，
 * 「打开 → 保存」逐字节一致。而 `setValue()`（重新加载、冲突解决）之后
 * 交回的正是外壳刚塞进来的那份 —— 也就是磁盘上的原文，不会被内核改写。
 *
 * ## `toHtml()` 返回 `null`
 *
 * 渲染器确实会给一份 `<svg>` 快照，但图纸里可以有指向外网的图片形状；
 * 把它塞进「自包含导出」就会让那份 HTML 在别人机器上去联网（ADR 0001）。
 * 与导图同一条处理：返回 `null`，自包含导出退回用 `<pre>` 呈现 XML 并如实说明。
 * 重审条件记在 ADR 0048 里（要做的前提是「导出时把外链图片换成内联或明确标注」）。
 *
 * ## 导出成图片（`exportImage`，ADR 0051）
 *
 * ADR 0048 那条重审条件说的「明确标注」，先在这条路上兑现：外链图片形状
 * 数出来交给 `warnings`，随文件带出去；把图片**换成内联**仍未做 —— 那一步
 * 要真去读外部文件，所以 `toHtml()` 维持 `null`（自包含 HTML 退回 `<pre>`）。
 *
 * 交出去的是「此刻画在画布上的那张 svg」：查看器画完就把整张图留在 canvas 里
 * （多页 / 缩放的交互也在它上面），序列化它就是「把用户正看着的导出去」
 * —— 当前页、当前缩放，所见即所得。
 *
 * 与导图那轮（ADR 0050）的三处不同，都是图纸内核自己的事实决定的：
 * - 底色不用换：mxGraph 把解析好的底色写进 svg 根的**行内 style**（viewer.min.js
 *   实证 `this.svg.style.backgroundColor = <色>.cssText`），序列化天然带着走；
 *   位图那边没有这层待遇，画布先铺同色再画。
 * - PNG 只能自己转：mind-elixir 有现成的 `exportPng()` 可借，查看器产物里
 *   没有任何导出 API（见上面「只能查看」那段）—— Image + canvas 在这里补上。
 * - 序列化用 `XMLSerializer` 而不是 `outerHTML`：mxGraph 的图片形状用的是
 *   `xlink:href`，`outerHTML` 不补命名空间声明 —— 那样的独立文件按 XML 一解析
 *   就报错，在别人机器上根本打不开。
 */

import { useEffect, useRef, useState } from 'react';

import { missingStencilLibs } from '../../core/drawio.js';
import { renderDrawio, type DrawioRender } from '../../core/drawio-render.js';
import type { EditorMountProps } from '../../core/editor/types.js';
import { msg, t, type LocalizedText } from '../../core/i18n/index.js';

import './drawio.css';

/**
 * 占位区要显示的那一条。
 *
 * **存数据，不存句子**（ADR 0047）：`kind: 'failed'` 带的是**渲染器的原话**，
 * 那句话一个字都不许翻 —— 它是 mxGraph 报的错，翻译它等于替系统说话；
 * 另外两种是 key + 参数，画的那一刻才 `t()`。于是切语言时它们跟着变，
 * 而渲染器的原话保持原样（`EACCES`、`Invalid XML` 这些本来就该原样给你看）。
 */
type Blank =
  | { readonly kind: 'empty' }
  | { readonly kind: 'failed'; readonly reason: string }
  | { readonly kind: 'missing-libs'; readonly libs: readonly string[] };

/** 只在渲染的那一刻调 —— 这是本文件里唯一把「数据」变成「一句话」的地方。 */
function blankText(blank: Blank): string {
  switch (blank.kind) {
    case 'empty':
      return t('ui.drawio.emptyHint');
    case 'failed':
      return blank.reason;
    case 'missing-libs':
      // 分隔符也是一个 key：英文里顿号不对，而「把翻译过的片段用硬写的顿号拼起来」
      // 正是 ADR 0047 记下的那个坑。
      return t('ui.drawio.missingLibs', { libs: blank.libs.join(t('ui.drawio.libJoin')) });
  }
}

/** 画布上此刻挂着的那张图 —— 查看器画完就把整张图留在 canvas 里（图区是内容
 * 尺寸的，见 drawio.css），`renderDrawio` 取的也是这第一个 svg。 */
function drawnDiagram(canvas: HTMLElement | null): SVGSVGElement | null {
  const found = canvas?.querySelector('svg') ?? null;
  return found instanceof SVGSVGElement ? found : null;
}

/**
 * 引用**外部文件**的图片形状数（`data:` 的已内嵌进图里，其余的 mxGraph
 * 原样写引用）。与导图那份 `countExternalImages` 同一条理由：外链图片导出去
 * 之后**不再是自包含的**，用户必须知道 —— 数出来交给 `exportImage` 的
 * `warnings`，而不是装作没看见。mxGraph 把地址写在 `xlink:href`（老产物）
 * 或 `href`（新产物）里，两处都看。
 */
function countExternalImages(svg: SVGSVGElement): number {
  let count = 0;
  for (const image of svg.querySelectorAll('image')) {
    const href = image.getAttribute('href') || image.getAttributeNS('http://www.w3.org/1999/xlink', 'href') || '';
    if (href !== '' && !href.startsWith('data:')) count += 1;
  }
  return count;
}

/**
 * 序列化成一份**独立打开也合法**的 SVG —— 用 `XMLSerializer` 而不是
 * `outerHTML`：它会补上 `xmlns`（以及用到的 `xmlns:xlink`）声明，
 * `outerHTML` 不补，而 mxGraph 的图片形状用的正是 `xlink:href` ——
 * 少了那份声明，文件按 XML 一解析就报错，在别人机器上根本打不开。
 */
function serializeSvg(svg: SVGSVGElement): string {
  return new XMLSerializer().serializeToString(svg);
}

/** 位图的尺寸：查看器按内容尺寸写宽高属性 → viewBox → 布局尺寸，一路兜底。 */
function pngSize(svg: SVGSVGElement): { readonly width: number; readonly height: number } {
  const attribute = (name: 'width' | 'height'): number => {
    const parsed = parseFloat(svg.getAttribute(name) ?? '');
    return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
  };
  const box = svg.viewBox.baseVal;
  return {
    width: attribute('width') || box.width || svg.clientWidth,
    height: attribute('height') || box.height || svg.clientHeight,
  };
}

/**
 * 把序列化好的 SVG 画成 PNG —— 查看器没有的导出能力在这里补（见文件头）。
 *
 * 诚实边界与导图那轮同一条：位图编码失败（图加载不出、拿不到 2d 上下文、
 * `toBlob` 交回 null）一律交回 `null`，让外壳如实报「没能画出来」。
 * 外链图片形状画不进来 —— SVG 当位图源用时不取外部资源 —— 正好对应
 * `ui.drawio.exportPngImagesExternal` 那句警告。
 */
async function svgToPng(svg: SVGSVGElement): Promise<Blob | null> {
  const { width, height } = pngSize(svg);
  if (width <= 0 || height <= 0) return null;

  const url = URL.createObjectURL(new Blob([serializeSvg(svg)], { type: 'image/svg+xml' }));
  try {
    const bitmap = await new Promise<HTMLImageElement | null>((resolve) => {
      const image = new Image();
      image.onload = () => resolve(image);
      image.onerror = () => resolve(null);
      image.src = url;
    });
    if (bitmap === null) return null;

    const canvas = document.createElement('canvas');
    canvas.width = Math.round(width);
    canvas.height = Math.round(height);
    const ctx = canvas.getContext('2d');
    if (ctx === null) return null;

    // 底色铺成「用户看到的那种」：SVG 文件靠行内 style 自带底色（见文件头），
    // 位图这边要自己铺 —— 画布此刻是什么色就铺什么色；解析不出（空串）就不铺，
    // 宁可透明也不编一个颜色（与导图 `withResolvedBackground` 同一条纪律）。
    const background = getComputedStyle(svg).backgroundColor;
    if (background.startsWith('rgb') && background !== 'rgba(0, 0, 0, 0)') {
      ctx.fillStyle = background;
      ctx.fillRect(0, 0, canvas.width, canvas.height);
    }
    ctx.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
    return await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/png'));
  } finally {
    URL.revokeObjectURL(url);
  }
}

export default function DrawioEditor({ initialValue, onReady }: EditorMountProps) {
  const canvasRef = useRef<HTMLDivElement | null>(null);

  /**
   * 当前这份图纸的原文。
   *
   * 放 state 是因为外部（`setValue`）换内容时要重画；放 ref 是因为
   * `getValue()` 要在没有重渲染的时刻也答得出来（保存走的就是那条路）。
   */
  const [xml, setXml] = useState(initialValue);
  const xmlRef = useRef(initialValue);

  /**
   * 正在画的那一趟渲染。`exportImage` 要等它结束 —— 查看器是异步画完的
   * （脚本注入、渲染回调、两帧收尾），「打开图纸立刻点导出」的那一刻
   * svg 可能还没挂上画布。
   */
  const renderRef = useRef<Promise<DrawioRender> | null>(null);

  /** 占位区要显示的东西。`null` = 画出来了，不用占位。存的是**数据**不是句子，见 ADR 0047。 */
  const [blank, setBlank] = useState<Blank | null>(null);

  const onReadyRef = useRef(onReady);
  onReadyRef.current = onReady;

  useEffect(() => {
    onReadyRef.current({
      getValue: () => xmlRef.current,
      setValue: (text: string) => {
        xmlRef.current = text;
        setXml(text);
      },
      focus: () => canvasRef.current?.focus(),
      // 见文件头：没有语义 HTML，返回 `null` 是合法结果（外壳会退回 `<pre>` + 如实说明）。
      toHtml: async () => null,

      /**
       * 把「此刻画在画布上的那张图」交出去（ADR 0051 —— 文件头「导出成图片」
       * 那节的三个取舍都落在这几十行里）。
       *
       * 先等正在画的这一趟结束，再取画布上的 svg：空文件、渲染失败、正在重画
       * 的那一刻画布上都没有图 —— `null` 照契约语义如实传（「这一次没能画
       * 出来」），外壳会按失败报告，不许当成成功。
       */
      exportImage: async (format) => {
        await renderRef.current;
        const svg = drawnDiagram(canvasRef.current);
        if (svg === null) return null;

        const warnings: LocalizedText[] = [];
        const external = countExternalImages(svg);
        if (external > 0) {
          warnings.push(
            msg(
              format === 'png' ? 'ui.drawio.exportPngImagesExternal' : 'ui.drawio.exportSvgImagesExternal',
              { n: external },
            ),
          );
        }
        // 缺 stencil 库的形状在屏幕上就是矩形（占位区也说了）—— 可导出的文件
        // 一离开这面板，那句话就没了，如实随文件带出去（ADR 0005 实测结论 4）。
        const missing = missingStencilLibs(xmlRef.current);
        if (missing.length > 0) {
          warnings.push(msg('ui.drawio.exportMissingLibs', { libs: missing.join(t('ui.drawio.libJoin')) }));
        }

        if (format === 'svg') {
          return { blob: new Blob([serializeSvg(svg)], { type: 'image/svg+xml' }), warnings };
        }
        const blob = await svgToPng(svg);
        return blob === null ? null : { blob, warnings };
      },
    });
  }, []);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (canvas === null) return;

    // 重画前先清空 —— `renderDrawio` 只往里**追加**，容器的内容归调用方管。
    canvas.replaceChildren();
    setBlank(null);
    // 上一趟的 Promise 作废：这一趟要么马上挂上新的，要么根本不画（空文件）。
    renderRef.current = null;

    // 空文件不进渲染器：它会回一句「图表内容是空的」，而这里要说的是
    // 「怎么让它有内容」—— 一句能照着做的提示，比一句准确的诊断有用。
    if (xml.trim() === '') {
      setBlank({ kind: 'empty' });
      return;
    }

    let cancelled = false;
    const render = renderDrawio(xml, canvas);
    // `exportImage` 等的就是这个 Promise（见 `renderRef`）。
    renderRef.current = render;
    void render.then((result) => {
      // 卸载之后不要再动 state：内核换文档时这个 Promise 还挂着。
      if (cancelled) return;
      if (!result.ok) {
        setBlank({ kind: 'failed', reason: result.message });
        return;
      }
      // 缺 stencil 库：图画出来了，只是部分形状退化成矩形。
      // 一定要说出来（ADR 0005 的 V1 实测结论 4：mxGraph 取不到 stencil
      // 既不报错也不提示，不说的话就是**静默地画错**）。
      const missing = missingStencilLibs(xml);
      if (missing.length > 0) setBlank({ kind: 'missing-libs', libs: missing });
    });

    return () => {
      cancelled = true;
    };
  }, [xml]);

  return (
    <div className="qm-drawio">
      <p className="qm-drawio-note">{t('ui.drawio.readOnlyNote')}</p>
      {/*
        canvas **一直挂着**，哪怕这一份是空图纸/渲染失败。
        条件挂载会让「从空图纸改成有图纸」这条路径整个失效：
        `xml` 变化时 effect 重跑，而那一刻 `canvasRef.current` 恰好是 null
        （因为正在显示占位），于是什么也不画 —— 一个只在空图纸上才复现的死角。
      */}
      <div
        ref={canvasRef}
        className={`qm-drawio-canvas${blank === null ? '' : ' is-tucked'}`}
        tabIndex={-1}
      />
      {blank !== null && (
        <div
          className={`qm-drawio-blank${blank.kind === 'failed' ? ' is-failed' : ''}`}
          role="status"
        >
          {blankText(blank)}
        </div>
      )}
    </div>
  );
}