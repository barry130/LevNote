/**
 * 图表的全屏查看器（Mermaid）。
 *
 * ## 为什么它和灯箱不是一个组件
 *
 * 灯箱看的是**一张图片**：有文件路径、有原始尺寸，缩放靠 `<img>` 自己。
 * 图表看的是**一段 SVG**：没有文件、没有原始尺寸，而且它的大小是 mermaid
 * 在渲染时按内容算出来的。两者的「放大」不是一回事 ——
 * 硬塞进同一个组件会让每一处都多一个 `if (是图片)`。
 *
 * ## 为什么底色**跟着主题走**，不像灯箱那样写死深色
 *
 * 灯箱写死深色是因为「图片的观感取决于周围亮度」（见 `Lightbox.tsx`）。
 * 图表不是照片：它是一张**矢量图**，而且 mermaid 已经按主题的明暗给过它
 * 一套配色（见 `core/mermaid-render.ts` 的 `configFor`）。再铺一层深色底，
 * 浅色主题下画出来的浅色图就会浮在一块黑底上 —— 那才是真正的读不清。
 *
 * ## 为什么图的尺寸**不**由外壳重算
 *
 * SVG 是 mermaid 给的，里面的宽高已经是「这张图该有的样子」。
 * 这里只用 CSS 限制它不超过视口（`max-width` / `max-height`），
 * 不去改它的 `width` / `height` 属性 —— 改了会让图里的文字与框的比例失真
 * （SVG 的 `viewBox` 还在，浏览器会把内容和框分别缩放）。
 */

import { useEffect } from 'react';

import type { DiagramView } from '../core/editor/types.js';
import { t } from '../core/i18n/index.js';

export interface DiagramViewerProps {
  diagram: DiagramView;
  onClose: () => void;
}

export function DiagramViewer({ diagram, onClose }: DiagramViewerProps) {
  /** Esc 关闭。捕获阶段 + 吃掉事件 —— 理由与灯箱完全一致。 */
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent): void => {
      if (event.key !== 'Escape') return;
      event.preventDefault();
      event.stopPropagation();
      onClose();
    };
    window.addEventListener('keydown', onKeyDown, true);
    return () => window.removeEventListener('keydown', onKeyDown, true);
  }, [onClose]);

  return (
    <div
      className="diagram-viewer"
      role="dialog"
      aria-modal="true"
      aria-label={t('ui.diagram.title')}
      onClick={onClose}
    >
      <button
        type="button"
        className="diagram-viewer-close"
        aria-label={t('ui.preview.close')}
        onClick={(event) => {
          event.stopPropagation();
          onClose();
        }}
      >
        ×
      </button>

      <div
        className="diagram-viewer-body"
        /*
         * 内容来自 mermaid，而 mermaid 在 `securityLevel: 'strict'` 下已经用
         * DOMPurify 清洗过（见 `core/mermaid-render.ts`）。这里是**第二道**：
         * 全屏查看器把同一段字符串再插一次，若将来清洗那道被人改掉，
         * 这里至少不是唯一的防线 —— 但也不必自己再引一个清洗库。
         */
        dangerouslySetInnerHTML={{ __html: diagram.svg }}
      />

      {/*
        源码要显示：图是从它来的，用户看完大图之后常要做的下一件事
        就是「回去改那两行字」。不给源码，他就得先想起来图在哪一段。
      */}
      <pre className="diagram-viewer-src">{diagram.source}</pre>
    </div>
  );
}
