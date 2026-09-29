/**
 * 文档的**附属资源**（目前只有图片）落在哪、叫什么名字、怎么被引用。
 *
 * ## 为什么单独一个模块，而不是塞进 `core/path.ts`
 *
 * `path.ts` 管的是**纯字符串**：切分、拼接、判扩展名，与「这是什么文件」无关。
 * 这里管的是**一项产品决策**：「图片存在文档旁边的一个目录里，Markdown 用相对路径引用」。
 * 两者的重审条件完全不同 —— 换一个资源目录方案不会动 `path.ts` 一行，
 * 而给 `path.ts` 加一个 `..` 解析会影响所有调用方。混在一起会让两边都变得不敢改。
 *
 * ## 布局
 *
 * ```
 * 笔记/
 * ├── 读书笔记.md
 * └── 读书笔记.assets/          ← 与文档同名 + `.assets`
 *     ├── image-20260929-143022-a3f2.png
 *     └── image-20260929-150311-7c19.jpg
 * ```
 *
 * **为什么是 `<文档名>.assets` 而不是全局的 `assets/`**：
 * 一个目录里可能有十份文档，全局 `assets/` 会让它们共用一个目录，
 * 于是「删掉这份文档」时没人知道哪些图还属于别人。同名目录的归属是明确的 ——
 * 用户可以放心地整目录删掉或搬走。
 *
 * 这也**不是** `assets/<文档名>/`：后者在多一层的同时，目录名与文档名的对应关系
 * 藏在了第二层，用户在文件管理器里按名字排序时看不出来。
 */

import { imageExtensionFor, imageMimeFor } from '../image.js';
import { dirname, joinPath, titleOf } from '../path.js';

/** 资源目录的后缀。改它等于换布局，旧文档里的相对路径会失效 —— 重审条件见 ADR 0018。 */
export const ASSETS_DIR_SUFFIX = '.assets';

/**
 * 文档对应的资源目录**路径**。
 *
 * 文档没有目录信息时返回 `null` —— 与 `dirname` 同一条纪律：**拿不到就不猜**。
 * 一份没有保存过的文档（浏览器预览虚拟盘里的 `demo.md`）没有可写的位置，
 * 调用方应当如实拒绝粘贴并说明原因，而不是写到一个猜出来的路径上。
 */
export function assetsDirFor(documentPath: string): string | null {
  const directory = dirname(documentPath);
  if (directory === null) return null;
  return joinPath(directory, `${titleOf(documentPath)}${ASSETS_DIR_SUFFIX}`);
}

/**
 * 资源目录的**名字**（不是路径），写进 Markdown 相对路径里用。
 */
export function assetsDirNameFor(documentPath: string): string {
  return `${titleOf(documentPath)}${ASSETS_DIR_SUFFIX}`;
}

function pad(value: number, width: number): string {
  return String(value).padStart(width, '0');
}

/** `20260929-143022` —— 本地时间，因为用户是按自己的时间找文件的。 */
export function stampOf(at: Date): string {
  return (
    `${at.getFullYear()}${pad(at.getMonth() + 1, 2)}${pad(at.getDate(), 2)}` +
    `-${pad(at.getHours(), 2)}${pad(at.getMinutes(), 2)}${pad(at.getSeconds(), 2)}`
  );
}

/**
 * 生成一个图片文件名。
 *
 * ## 为什么带时间戳与随机后缀，而不是 `image-1.png`
 *
 * 后者需要**先列目录**才知道 1 有没有被占用，而那是异步 IO ——
 * 于是「生成名字」从纯函数变成一个可能失败的异步动作，
 * 中间还夹着「列完目录到写入之间别人占了这个名字」的竞态。
 *
 * 时间戳 + 4 位十六进制随机（约 6.5 万分之一同秒碰撞）把冲突概率压到
 * 用户一辈子遇不到，而且**没有 IO**。文件名也可读、按名字排序就是时间顺序。
 *
 * `salt` 由调用方注入（`Math.random`），所以这个函数**是纯的、可测的**。
 */
export function imageFileName(options: {
  readonly mimeType: string;
  readonly at: Date;
  readonly salt: string;
}): string | null {
  const extension = imageExtensionFor(options.mimeType);
  if (extension === null) return null;
  return `image-${stampOf(options.at)}-${options.salt}${extension}`;
}

/**
 * 这个 `src` 是不是「外部地址」，不需要也不能在本地解析。
 *
 * 覆盖三类：`http(s)://` / `data:` / `blob:`，以及任何别的 scheme（`ftp:`、`mailto:`…）。
 * 判据是「冒号前是不是一个 scheme」，而不是逐个列举 —— 列举的话，
 * 遇到没列到的 scheme 会被当成相对路径去拼一个本地绝对路径，
 * 结果是「图片显示不出来」外加一条指向莫名其妙路径的报错。
 *
 * 注意**不包含** Windows 盘符：`C:\a.png` 里 `C:` 也是「冒号前有东西」，
 * 但它显然是本地绝对路径。所以先排掉盘符形状再判 scheme。
 */
export function isExternalSrc(src: string): boolean {
  if (src === '') return true;
  if (/^[A-Za-z]:[\\/]/.test(src)) return false; // Windows 绝对路径
  if (src.startsWith('\\\\')) return false; // UNC
  return /^[a-z][a-z0-9+.-]*:/i.test(src);
}

/**
 * 把文档里的图片 `src` 解析成本地**绝对路径**；解析不出来返回 `null`。
 *
 * 返回 `null` 的三种情况，调用方都要如实降级（画占位、说明原因）：
 * 1. 外部地址（`http:` / `data:` …）—— 不该去读本地；
 * 2. 文档自己没有目录信息 —— 相对路径无从谈起；
 * 3. `src` 是空的。
 *
 * **刻意不解析 `..`**：`joinPath` 只做拼接。所以 `../../a.png` 会得到一个
 * 带 `..` 的路径，交给 Rust 侧的文件 API —— 由操作系统做真正的规范化。
 * 在这里自己实现一套 `..` 折叠，等于在一个只做显示与拼接的模块里
 * 塞进文件系统语义，两者对「符号链接」这类情况的理解必然不一致。
 *
 * **但分隔符风格会统一成文档路径的风格**（`./a.assets/x.png` 在 Windows 上
 * 得到 `C:\笔记\a.assets\x.png`）。这不是多余的美化：写入时文件名是裸名、
 * 回读时相对路径带正斜杠，不统一的话同一个文件会有两个字符串，
 * 而按字符串比较的地方（缓存键、去重）会悄悄失配 —— 表现为
 * 「图写进去了但显示不出来」。细节见 `core/path.ts` 的 `joinPath`。
 */
export function resolveImagePath(documentPath: string, src: string): string | null {
  if (src === '' || isExternalSrc(src)) return null;
  if (/^[A-Za-z]:[\\/]/.test(src) || src.startsWith('/') || src.startsWith('\\\\')) return src;
  return joinPath(dirname(documentPath), src);
}

/**
 * 图片在 Markdown 里应该写成什么（相对路径）。
 *
 * 只返回「写进文档的那一串」，不返回绝对路径 —— 绝对路径是本机私有的，
 * 写进文档会让这份文档在别人的机器上、在 Git 仓库里都失去意义。
 */
export function imageMarkdownSrc(assetsDirName: string, fileName: string): string {
  return `./${assetsDirName}/${fileName}`;
}

/**
 * 判断一个 `src` 是不是我们已经认得的图片文件（用于灯箱之类的动作）。
 *
 * 判据是扩展名，且**只在没有 MIME 信息时**才用得上 —— 它是给「用户点了一张图，
 * 我们要决定要不要弹灯箱」这种场景用的，判错只会少弹或多弹一次预览，
 * 不涉及任何写入，所以用便宜的扩展名判断是合适的。
 */
export function looksLikeImage(src: string): boolean {
  return imageMimeFor(src) !== null;
}
