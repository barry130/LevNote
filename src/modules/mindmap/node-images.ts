/**
 * 导图节点插图的三件小事（ADR 0054）：
 *
 * 1. **预热**：挂载内核之前把文件里引用的图全部解析成 `data:` URL
 *    （`resolveImageUrl` 契约），存进一张小表；`imageProxy` 同步查表。
 * 2. **占位**：解析不到（文件丢了、或干脆是个外链 —— ADR 0001 不联网
 *    去 fetch）就画一张**带路径的**内嵌 SVG。图丢了是常见状态（别人发你
 *    一个 .mindmap，没把 assets 目录一起发），占位就是不静默的体现。
 * 3. **粘贴**：剪贴板里取出图片文件（字节交给外壳的 `onImagePaste` 契约
 *    落盘，换回一个引用），解码一次尺寸，压到显示上限。
 *
 * 本文件**全是纯函数与 DOM 小工具**，不碰内核、不碰 React ——
 * jsdom 测得到多少是多少（像素解码除外，见 ADR 0054）。
 */

import { t } from '../../core/i18n/index.js';
import type { MindmapNode } from '../../core/mindmap/types.js';

/**
 * 粘贴图片的显示尺寸上限（px）：**最长边**压到这个值，**只缩不放** ——
 * 4000×3000 的照片直接进画布会把一屏的主题全挤没。原图字节照原样落盘，
 * 这里压的只是「画出来多大」，与文件内容无关。
 */
export const IMAGE_DISPLAY_MAX = 320;

/** 占位框的视口尺寸。内核会按 `image.width/height` 再拉伸它，这只是内部分辨率。 */
const PLACEHOLDER_WIDTH = 160;
const PLACEHOLDER_HEIGHT = 96;
/** 占位框里路径最多显示这么长（SVG `<text>` 不会自动换行，超宽会溢出虚线框）。 */
const PLACEHOLDER_PATH_MAX = 44;

/** XML 转义。占位 SVG 里要嵌文件路径 —— 路径是任意的，必须当文本处理。 */
function escapeXml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;');
}

/**
 * 先序走一遍树，收出**需要解析**的图片引用：非空、且不是 `data:` 内嵌图
 * （内嵌图自包含，谁也不必再解析）。去重 —— 十个节点引用同一张图只读一次盘。
 */
export function collectImageUrls(root: MindmapNode): string[] {
  const seen = new Set<string>();
  const urls: string[] = [];
  const walk = (node: MindmapNode): void => {
    const url = node.image?.url;
    if (url !== undefined && url !== '' && !url.startsWith('data:') && !seen.has(url)) {
      seen.add(url);
      urls.push(url);
    }
    for (const child of node.children) walk(child);
  };
  walk(root);
  return urls;
}

/**
 * 一张**带路径的**占位图（`data:image/svg+xml`）：虚线框 + 路径名。
 * 图丢了 / 外链读不了时用它顶上 —— 节点上空空的等于假装没有这回事。
 *
 * 说明文字走 `t()`：占位图在预热时烘焙（语言切换 = 整个内核重建 = 重温，
 * ADR 0049），所以烘焙时的语言就是画出来的语言，不会出现半张图旧语言。
 */
export function placeholderImage(src: string): string {
  const trimmed = src.length > PLACEHOLDER_PATH_MAX ? `${src.slice(0, PLACEHOLDER_PATH_MAX - 1)}…` : src;
  const svg =
    `<svg xmlns="http://www.w3.org/2000/svg" width="${PLACEHOLDER_WIDTH}" height="${PLACEHOLDER_HEIGHT}" viewBox="0 0 ${PLACEHOLDER_WIDTH} ${PLACEHOLDER_HEIGHT}">` +
    `<rect x="1" y="1" width="${PLACEHOLDER_WIDTH - 2}" height="${PLACEHOLDER_HEIGHT - 2}" rx="8" fill="#f4f4f4" stroke="#9a9a9a" stroke-width="2" stroke-dasharray="6 4"/>` +
    `<text x="${PLACEHOLDER_WIDTH / 2}" y="38" font-family="sans-serif" font-size="22" fill="#9a9a9a" text-anchor="middle">🖼</text>` +
    `<text x="${PLACEHOLDER_WIDTH / 2}" y="60" font-family="sans-serif" font-size="11" fill="#707070" text-anchor="middle">${escapeXml(trimmed)}</text>` +
    `<text x="${PLACEHOLDER_WIDTH / 2}" y="76" font-family="sans-serif" font-size="10" fill="#9a9a9a" text-anchor="middle">${escapeXml(t('ui.mindmap.imagePlaceholderNote'))}</text>` +
    `</svg>`;
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`;
}

/**
 * 先序收出**带图节点**的 id。`setValue` 换了新数据之后用它把已经画出来的
 * 占位换成真图（见 `MindmapEditor` 的 `setValue`）；折叠着的节点不在其中也
 * 无妨 —— 展开那一刻内核会重渲染它们，自然查到已经填好的表。
 */
export function collectImageNodeIds(root: MindmapNode): string[] {
  const ids: string[] = [];
  const walk = (node: MindmapNode): void => {
    if (node.image !== undefined) ids.push(node.id);
    for (const child of node.children) walk(child);
  };
  walk(root);
  return ids;
}

/**
 * 预热：把每个引用都变成「能直接进 `src` 的 URL」。解析不到（返回 null、
 * 抛异常、或外壳根本没实现契约）一律落占位 —— 显示层**永不等到一个空的 `src`**。
 */
export async function prewarmImageCache(
  urls: readonly string[],
  resolve: ((src: string) => Promise<string | null>) | undefined,
): Promise<Map<string, string>> {
  const cache = new Map<string, string>();
  await Promise.all(
    urls.map(async (url) => {
      let resolved: string | null = null;
      if (resolve !== undefined) {
        try {
          resolved = await resolve(url);
        } catch {
          resolved = null;
        }
      }
      cache.set(url, resolved ?? placeholderImage(url));
    }),
  );
  return cache;
}

/**
 * 从剪贴板里取出**第一个**图片文件。不是图片的条目（文本、普通文件）跳过；
 * 取不出就返回 null —— 由调用方决定「这次粘贴跟图片无关」，别的事都不做。
 *
 * **两处都要看**（markdown 内核 `imageFilesOf` 的同一条教训）：各平台 WebView
 * 的填充方式不一致，有的把图片放进 `files`，有的只放进 `items` 且 `files`
 * 是空的 —— 只看一处就是「只在别人的机器上坏」。
 */
export function extractImageFile(clipboardData: DataTransfer | null): File | null {
  if (clipboardData === null) return null;

  for (const file of Array.from(clipboardData.files)) {
    if (file.type.startsWith('image/')) return file;
  }
  const items = clipboardData.items;
  for (let index = 0; index < items.length; index += 1) {
    const item = items[index];
    if (item === undefined) continue;
    if (item.kind === 'file' && item.type.startsWith('image/')) {
      const file = item.getAsFile();
      if (file !== null) return file;
    }
  }
  return null;
}

/** 解码后的原图尺寸。 */
export interface DecodedImageSize {
  width: number;
  height: number;
}

/**
 * 解码一次拿原图尺寸（粘贴的图必须先量出来才谈得上显示框多大）。
 * 量不出（坏字节、不认识的格式）返回 null —— 由调用方如实报告，不猜尺寸。
 *
 * 注：jsdom 的 `Image` 不会真解码（ onload / onerror 都不触发），这条
 * 在单元测试里永远悬着 —— 所以测试**必须注入替身**（`vi.mock`），真实
 * 解码在 `pnpm run tauri dev` 人工冒烟里验。
 */
export function decodeImageSize(bytes: ArrayBuffer, mimeType: string): Promise<DecodedImageSize | null> {
  return new Promise((resolve) => {
    const objectUrl = URL.createObjectURL(new Blob([bytes], { type: mimeType }));
    const image = new Image();
    image.onload = () => {
      URL.revokeObjectURL(objectUrl);
      resolve(
        image.naturalWidth > 0 && image.naturalHeight > 0
          ? { width: image.naturalWidth, height: image.naturalHeight }
          : null,
      );
    };
    image.onerror = () => {
      URL.revokeObjectURL(objectUrl);
      resolve(null);
    };
    image.src = objectUrl;
  });
}

/** 把已经缩好的显示尺寸取整到像素，并保证**只缩不放**。 */
export function clampDisplaySize(width: number, height: number): DecodedImageSize {
  const longest = Math.max(width, height);
  if (longest <= IMAGE_DISPLAY_MAX) return { width, height };
  const ratio = IMAGE_DISPLAY_MAX / longest;
  return {
    width: Math.max(1, Math.round(width * ratio)),
    height: Math.max(1, Math.round(height * ratio)),
  };
}

/**
 * 字节 → `data:` URL。粘贴成功后先塞进本会话的预热表 —— 这张图**本次编辑**
 * 就能看到（写盘已由外壳完成，这里只是让画布立刻反映出来）。
 */
export function bytesToDataUrl(bytes: ArrayBuffer, mimeType: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      if (typeof reader.result === 'string') resolve(reader.result);
      else reject(new Error('readAsDataURL did not produce a string'));
    };
    reader.onerror = () => {
      reject(reader.error ?? new Error('readAsDataURL failed'));
    };
    reader.readAsDataURL(new Blob([bytes], { type: mimeType }));
  });
}
