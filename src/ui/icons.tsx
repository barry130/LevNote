/**
 * 图标集。
 *
 * ## 为什么自己画，不引图标库
 *
 * 需要的图标不到十个，而一个图标库（codicons / lucide / heroicons）会带来
 * 三件事：一个**新依赖**、一套**和项目无关的命名**、以及一次**按需摇树也摇不干净**
 * 的包体增长。十个 16×16 的 path 手写下来是六十行，代价比接一个库小得多。
 *
 * ## 统一约定
 *
 * - 一律 `16×16` 视图框、`stroke="currentColor"`、`fill="none"`。
 *   **颜色由调用方通过 CSS 的 `color` 决定** —— 于是「活动栏图标的选中态」
 *   是一件 CSS 的事，不需要给每个图标传一个 `color` 参数。
 * - 线宽 1.4：VSCode 的 codicon 在 16px 下大约就是这个观感。
 *   再细在浅色主题下发虚，再粗会糊成一团。
 * - `strokeLinecap/Linejoin` 用 `round`，与 VSCode 的圆角笔画一致。
 * - `aria-hidden`：这些图标**永远**和一个有文字的按钮或 `title` 一起出现，
 *   自己再报一次名字只会让读屏念两遍。
 */

import type { ReactNode } from 'react';

interface IconProps {
  /** 边长，默认 16。活动栏用 22，标签栏用 16。 */
  readonly size?: number;
}

function Svg({ size = 16, children }: IconProps & { children: ReactNode }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.4}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      {children}
    </svg>
  );
}

/** 文件浏览器（侧栏）。 */
export function IconFiles(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M1.9 4.3a1 1 0 0 1 1-1h3.3l1.5 1.6h6.4a1 1 0 0 1 1 1v6.8a1 1 0 0 1-1 1H2.9a1 1 0 0 1-1-1z" />
    </Svg>
  );
}

/** 查找。 */
export function IconSearch(props: IconProps) {
  return (
    <Svg {...props}>
      <circle cx="7.1" cy="7.1" r="4.4" />
      <path d="M10.4 10.4 14 14" />
    </Svg>
  );
}

/** 大纲。 */
export function IconOutline(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M2.2 4.2h1.6M2.2 8h1.6M2.2 11.8h1.6" />
      <path d="M6.4 4.2h7.4M6.4 8h7.4M6.4 11.8h7.4" />
    </Svg>
  );
}

/** 命令面板。一个终端提示符 —— 比「放大镜」更像「输入一条命令」。 */
export function IconCommand(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M2.6 4.4 6.2 8l-3.6 3.6" />
      <path d="M8.4 12h5" />
    </Svg>
  );
}

/** 设置。 */
export function IconSettings(props: IconProps) {
  return (
    <Svg {...props}>
      <circle cx="8" cy="8" r="2.2" />
      <path d="M8 1.7v1.7M8 12.6v1.7M14.3 8h-1.7M3.4 8H1.7M12.5 3.5l-1.2 1.2M4.7 11.3l-1.2 1.2M12.5 12.5l-1.2-1.2M4.7 4.7 3.5 3.5" />
    </Svg>
  );
}

/** 新建文件。 */
export function IconNewFile(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M8 3.6v8.8M3.6 8h8.8" />
    </Svg>
  );
}

/**
 * 分屏（源码 + 预览）。
 *
 * 用「一个框中间一道竖线」而不是两个分开的框 —— 后者在 16px 下会糊成一片，
 * 而前者一眼能看出「左右两栏」。
 */
export function IconSplit(props: IconProps) {
  return (
    <Svg {...props}>
      <rect x="1.8" y="3" width="12.4" height="10" rx="1.2" />
      <path d="M8 3v10" />
    </Svg>
  );
}

/** 预览（只读渲染）。 */
export function IconPreview(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M1.6 8S3.9 4.2 8 4.2 14.4 8 14.4 8 12.1 11.8 8 11.8 1.6 8 1.6 8z" />
      <circle cx="8" cy="8" r="1.8" />
    </Svg>
  );
}

/** 源码。一对尖括号。 */
export function IconSource(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M5.6 4.4 2 8l3.6 3.6" />
      <path d="M10.4 4.4 14 8l-3.6 3.6" />
    </Svg>
  );
}

/** 富文本。一段带格式的正文（首行短、末行短）。 */
export function IconRichText(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M2.4 4.2h11.2M2.4 8h11.2M2.4 11.8h6.6" />
    </Svg>
  );
}
