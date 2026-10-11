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
 */

import { useEffect, useRef, useState } from 'react';

import { missingStencilLibs } from '../../core/drawio.js';
import { renderDrawio } from '../../core/drawio-render.js';
import type { EditorMountProps } from '../../core/editor/types.js';
import { t } from '../../core/i18n/index.js';

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
    });
  }, []);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (canvas === null) return;

    // 重画前先清空 —— `renderDrawio` 只往里**追加**，容器的内容归调用方管。
    canvas.replaceChildren();
    setBlank(null);

    // 空文件不进渲染器：它会回一句「图表内容是空的」，而这里要说的是
    // 「怎么让它有内容」—— 一句能照着做的提示，比一句准确的诊断有用。
    if (xml.trim() === '') {
      setBlank({ kind: 'empty' });
      return;
    }

    let cancelled = false;
    void renderDrawio(xml, canvas).then((result) => {
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