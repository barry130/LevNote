/**
 * 图片类型：MIME ↔ 扩展名，以及「从字节判断这是什么图」。
 *
 * ## 为什么这份知识必须只有一处
 *
 * 它服务三个地方，而三处对它的要求**恰好互相制约**：
 *
 * 1. **落盘**（`core/document/assets.ts`）：粘贴时按 MIME 决定文件后缀；
 * 2. **显示**（外壳把字节变成 `<img src>`）：需要一个 MIME 才能构造 data URI；
 * 3. **导出**（`core/export/inline-html.ts`）：把图片内联进单文件 HTML 时同样需要 MIME。
 *
 * 抄成三份的下场是**静默不一致**：编辑器里显示正常（用的是 A 表），
 * 导出的文件里那张图却打不开（用的是 B 表），而两张表看上去都没写错。
 * 这类 bug 只有「导出一份带图文档、断网打开」才暴露得出来。
 *
 * ## 两个方向的表为什么都手写
 *
 * 从 MIME 推扩展名不能靠「斜杠后那一段」（`image/svg+xml` 会得到 `svg+xml`），
 * 从扩展名推 MIME 也不能靠「拼 `image/`」（`jpg` 要拼成 `jpeg`）。
 * 两张表各自显式列出，代价是加一种格式要改两行 —— 换来的是不会有推导错误。
 */

/**
 * 认得的图片类型：MIME → 扩展名（**带点**，因为落盘时直接用）。
 *
 * **刻意只有这几项**：不在表里的一律拒绝，而不是猜一个 `.png`。
 * 猜错的后果是「文件叫 a.png，内容其实是 TIFF」—— 本地看起来正常
 * （我们自己按 MIME 显示），发给别人就打不开，而那时已经无从追查。
 * 剪贴板与浏览器的复制几乎只会给出这几个值。
 */
const EXTENSION_BY_MIME: Readonly<Record<string, string>> = {
  'image/png': '.png',
  'image/jpeg': '.jpg',
  'image/gif': '.gif',
  'image/webp': '.webp',
  'image/avif': '.avif',
  'image/bmp': '.bmp',
  'image/svg+xml': '.svg',
};

/**
 * 扩展名 → MIME。比上面那张表**多几项**，因为它们只出现在「读一个已存在的文件」时：
 * `jpe` / `jfif` 是 `image/jpeg` 的历史别名，`ico` 是图标。
 *
 * 但**反向不会多出 `image/x-icon` 这类冷门 MIME** —— 落盘时我们只会写自己认得的格式，
 * 没必要为它们准备后缀。
 */
const MIME_BY_EXTENSION: Readonly<Record<string, string>> = {
  png: 'image/png',
  jpg: 'image/jpeg',
  jpeg: 'image/jpeg',
  jpe: 'image/jpeg',
  jfif: 'image/jpeg',
  gif: 'image/gif',
  webp: 'image/webp',
  bmp: 'image/bmp',
  ico: 'image/x-icon',
  svg: 'image/svg+xml',
  avif: 'image/avif',
};

/** 认得的图片 MIME 的扩展名（带点）；不认识时返回 `null` —— 调用方必须如实拒绝，不要猜。 */
export function imageExtensionFor(mimeType: string): string | null {
  // 剪贴板偶尔带参数：`image/svg+xml;charset=utf-8`
  const bare = mimeType.split(';')[0]?.trim().toLowerCase() ?? '';
  return EXTENSION_BY_MIME[bare] ?? null;
}

/** 这个 MIME 是不是我们认识的图片。 */
export function isSupportedImageType(mimeType: string): boolean {
  return imageExtensionFor(mimeType) !== null;
}

/**
 * 从路径或扩展名推 MIME；不认识时返回 `null`。
 *
 * 三种输入都认：`C:\a\x.png`、`x.png`、`.png`、`png` —— 调用方手上可能是其中任何一种，
 * 要求它们各自先切一遍只会让这个函数在两个地方被重写。
 */
export function imageMimeFor(pathOrExtension: string): string | null {
  const name = pathOrExtension.split(/[\\/]/).pop() ?? '';
  const dot = name.lastIndexOf('.');
  const bare = (dot >= 0 ? name.slice(dot + 1) : name).toLowerCase();
  return MIME_BY_EXTENSION[bare] ?? null;
}

/**
 * 判断文件头，**再看**扩展名。返回值一定非空（认不出来给 `application/octet-stream`）。
 *
 * 顺序不能反：`.png` 里装的其实是一张 JPEG、扩展名写错的图片很常见，
 * 而 MIME 报错的后果是浏览器直接不渲染 —— 用户看到的是一个空白方框，
 * 完全不知道为什么。**文件头是事实，扩展名只是猜测。**
 *
 * 最后那个兜底值而不是 `null`，是因为调用方（导出、data URI 构造）都必须有一个 MIME
 * 才能继续；返回 `null` 只会让每个调用方各自编一个兜底。
 */
export function sniffImageMime(bytes: Uint8Array, path: string): string {
  const at = (...signature: number[]) => signature.every((byte, index) => bytes[index] === byte);

  if (at(0x89, 0x50, 0x4e, 0x47)) return 'image/png';
  if (at(0xff, 0xd8, 0xff)) return 'image/jpeg';
  if (at(0x47, 0x49, 0x46, 0x38)) return 'image/gif';
  if (at(0x52, 0x49, 0x46, 0x46) && bytes[8] === 0x57 && bytes[9] === 0x45) return 'image/webp';
  if (at(0x42, 0x4d)) return 'image/bmp';

  if (bytes[0] === 0x3c) {
    // `<` 开头：可能是 SVG，也可能是 HTML。
    // 按 `image/svg+xml` 处理更安全 —— 认成 HTML 的话浏览器会拒绝渲染这张图，
    // 而 SVG 里混着 XML 声明（`<?xml …?>`）和注释也很常见，那些都还是 SVG。
    const head = new TextDecoder('utf-8', { fatal: false })
      .decode(bytes.subarray(0, 512))
      .toLowerCase();
    if (head.includes('<svg')) return 'image/svg+xml';
  }

  return imageMimeFor(path) ?? 'application/octet-stream';
}

/**
 * 把字节转成 base64。
 *
 * 分块是为了避开 `String.fromCharCode(...bytes)` 的参数个数上限
 * （约 65535，一个 1 MB 的图片会直接抛 RangeError）。
 */
export function bytesToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  const chunk = 0x8000;
  let binary = '';
  for (let offset = 0; offset < bytes.length; offset += chunk) {
    binary += String.fromCharCode(...(bytes.subarray(offset, offset + chunk) as unknown as number[]));
  }
  return btoa(binary);
}

/**
 * 字节 → `data:` URI。
 *
 * 编辑器显示本地图片与导出内联图片**走的是同一个函数**：两边都是
 * 「读字节 → 拼 data URI」，所以不会出现「编辑器里能看、导出后打不开」。
 *
 * 刻意不缓存、也不返回 `Promise`：它是一个纯转换，缓存的职责在调用方
 * （外壳按绝对路径缓存，导出是一次性的）。
 */
export function bytesToDataUrl(buffer: ArrayBuffer, mime: string): string {
  return `data:${mime};base64,${bytesToBase64(buffer)}`;
}
