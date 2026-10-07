/**
 * 灯箱：点开文档里的一张图，铺满整个视口看它。
 *
 * ## 为什么底色**恒定是深的**，不跟主题走
 *
 * 这个项目的其余部分都严格跟着 `data-theme` 走，灯箱是**唯一**的例外，
 * 而理由是具体的：图片的观感取决于它周围的亮度。同一个浅色截图，
 * 放在白底上显得发灰、放在深底上显得通透 —— 用浅色底会让用户
 * 对「这张图到底亮不亮」判断错。所有正经的看图工具都是深底，正是这个原因。
 *
 * 所以这里不引用 `--surface` 之类的变量，而是写死一组深色值。
 * 它不随主题变化，也就不存在「深色主题下漏改」的问题。
 *
 * ## 关闭方式给三种，而不是一种
 *
 * 点任意处、按 Esc、点右上角的 × —— 三条路都通向 `onClose`。
 * 只留「点背景」的话，用户点在图上是关不掉的（图占了大部分面积），
 * 会以为卡住了。三种里没有一种是「隐藏功能」，都不需要解释。
 */

import { useEffect } from 'react';

import type { ImageInfo } from '../core/editor/types.js';
import { t } from '../core/i18n/index.js';

export interface LightboxProps {
  image: ImageInfo;
  onClose: () => void;
}

export function Lightbox({ image, onClose }: LightboxProps) {
  /**
   * Esc 关闭。
   *
   * 用 `capture: true` 在 window 上监听，而不是挂在自己身上靠焦点：
   * 灯箱没有把焦点抢过来（抢焦点会打断用户正在编辑器里的输入法状态），
   * 所以 `keydown` 只会在 document 上冒泡。捕获阶段拿下来之后
   * `stopPropagation` 把它吃掉 —— 否则同一个 Esc 还会被编辑器、
   * 查找栏、斜杠菜单各处理一次。
   */
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

  // 有 title 用 title、有 alt 用 alt、都没有就退回路径 —— 总有一样能说清这是哪张图
  const caption = image.title !== '' ? image.title : image.alt;

  return (
    <div className="lightbox" role="dialog" aria-modal="true" aria-label={t('ui.preview.title')} onClick={onClose}>
      <button
        type="button"
        className="lightbox-close"
        aria-label={t('ui.preview.close')}
        onClick={(event) => {
          event.stopPropagation();
          onClose();
        }}
      >
        ×
      </button>

      <img className="lightbox-img" src={image.url} alt={image.alt} />

      <p className="lightbox-caption">
        {caption !== '' && <span className="title">{caption}</span>}
        {/* 路径一定要显示：用户看完大图之后常要做的下一件事是「找到这个文件」 */}
        <span className="path">{image.src}</span>
      </p>
    </div>
  );
}
