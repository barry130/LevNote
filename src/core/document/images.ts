/**
 * 图片的**读取与落盘** —— 两个文档级动作。
 *
 * ## 为什么单独一个模块，而不是塞进 `DocumentStore`
 *
 * `DocumentStore` 管的是「一份文档的状态机」：文本、格式、脏标记、冲突。
 * 而这两个动作是**无状态的文件 IO**，唯一的输入是「provider + 文档路径 + 若干参数」。
 * 放进 store 会让它的每个方法都要先面对一堆与状态无关的分支，
 * 而单独放出来之后，测试可以直接喂一个假 provider，不必先搭一个 store。
 *
 * 这与 `core/export/export-document.ts` 是同一个形状 —— 那边的导出动作
 * 也是「拿 provider 干活、把结果（含失败原因）交回去」，由外壳决定怎么呈现。
 *
 * ## 这两个动作各自的不变量
 *
 * 1. **读**：解析不出来就返回 `null`，绝不抛。一张图丢了是常见状态
 *    （图被删了、文档被拷到别的机器），内核会画一个带路径的占位 ——
 *    那个占位就是「不静默」的体现，不需要再弹一次错误横幅。
 * 2. **写**：**绝不覆盖已存在的文件**。撞名时换一个盐重试，
 *    重试到底就如实失败。理由见 `writePastedImage`。
 */

import { bytesToDataUrl, imageExtensionFor, sniffImageMime } from '../image.js';
import { t } from '../i18n/index.js';
import { joinPath } from '../path.js';
import { assetsDirFor, assetsDirNameFor, imageFileName, imageMarkdownSrc, resolveImagePath } from './assets.js';
import { asAppError, type DocumentProvider } from './types.js';

/**
 * 把文档里的图片 `src` 解析成能直接放进 `<img>` 的 URL（`data:` URI）。
 *
 * ## 为什么是 `data:` 而不是 `asset:` 协议
 *
 * Tauri 提供了 `convertFileSrc()`，它能让 WebView 直接按路径流式加载本地文件，
 * 不必把字节读进 JS。代价是要打开 `app.security.assetProtocol` 并配置一份
 * **路径白名单**（scope）—— 那是一块新增的安全面，而白名单写宽了就等于
 * 允许页面读取任意本地文件。`data:` 走的是我们已有的 `read_binary_file`
 * 命令，权限收口在同一个地方，没有新增面。
 *
 * 代价是内存：base64 比原始字节大 1/3，而且会作为属性留在 DOM 上。
 * 上游 `read_binary_file` 已经卡了 24MB 的上限（与导出内联共用），
 * 所以最坏情况是有界的。**重审条件**：当一份文档里有几十张大图、
 * 打开明显变慢时，再评估切到 `asset:` 协议。
 *
 * ## 为什么用「文件头 + 扩展名」而不是信扩展名
 *
 * `.png` 里装着一张 JPEG 是常见的（改过名、某些截图工具）。MIME 报错时
 * 浏览器会直接不渲染 —— 用户看到空白方框，完全不知道为什么。详见 `core/image.ts`。
 */
export async function readImageAsUrl(
  provider: DocumentProvider,
  documentPath: string,
  src: string,
): Promise<string | null> {
  const absolute = resolveImagePath(documentPath, src);
  // 外部地址（http / data / blob）、空 src、文档没有目录 —— 三种都拿不到本地字节
  if (absolute === null) return null;

  try {
    const bytes = await provider.readBinaryFile(absolute);
    return bytesToDataUrl(bytes, sniffImageMime(new Uint8Array(bytes), absolute));
  } catch {
    // 读不到（文件不在、超过内联上限、权限不足）就是「这张图没有」。
    // 这里**不**把错误抛出去：调用链的尽头是 NodeView 的一次渲染，
    // 一次未捕获的拒绝会变成一个没人处理的 Promise，而界面毫无变化
    return null;
  }
}

/** 落盘成功。`src` 是**写进文档的那一串**（相对路径）。 */
export interface PasteImageOk {
  readonly ok: true;
  readonly src: string;
  /** 实际落盘的绝对路径。出错时能告诉用户文件在哪。 */
  readonly path: string;
  readonly size: number;
}

/**
 * 落盘失败。`reason` 是**给用户看的一句话**，不是错误码 ——
 * 调用方应当原样呈现它（状态栏 / 横幅），不要吞掉。
 */
export interface PasteImageFail {
  readonly ok: false;
  readonly reason: string;
}

export type PasteImageResult = PasteImageOk | PasteImageFail;

export interface PasteImageOptions {
  readonly bytes: ArrayBuffer;
  readonly mimeType: string;
  /** 落盘时间。**注入**是为了让生成的文件名可预测、测试可断言。 */
  readonly at: Date;
  /**
   * 取一个新的随机盐（4 位十六进制）。**注入**是为了让测试能造出「撞名」，
   * 从而真正走到重试分支 —— 用 `Math.random` 的话那条分支永远测不到。
   */
  readonly salt: () => string;
}

/**
 * 撞名时的重试次数。
 *
 * 3 次足够：第一次撞名的概率已经约 6.5 万分之一（同秒 + 同盐），
 * 三次全撞上说明**不是随机碰撞**（比如目标目录里真有一个同名文件），
 * 那时再重试也只是浪费时间 —— 如实失败更诚实。
 */
const WRITE_ATTEMPTS = 3;

/** 4 位十六进制的随机盐。`Math.floor` 而不是切片：`Math.random().toString(16)` 在小数部分为 0 时会返回 `"0"`。 */
export function randomSalt(): string {
  return Math.floor(Math.random() * 0x1_0000)
    .toString(16)
    .padStart(4, '0');
}

/**
 * 把一张粘贴进来的图片写到文档旁边的资源目录，返回**写进文档的相对路径**。
 *
 * ## 为什么是「已存在就换名字」而不是「覆盖」
 *
 * 名字由「时间戳 + 随机盐」生成，撞名只可能是两件事：同一秒里粘了两次
 * 且盐也撞了（概率极低），或者目标目录里**已经有一个我们不认识的文件**
 * 叫这个名字。两种情况下覆盖都是错的 —— 前者会丢掉刚粘的那张，
 * 后者会毁掉用户的文件。所以换一个盐重试，把「不覆盖」这条硬约束
 * 落到具体的动作上，而不是只写在注释里。
 *
 * ## 为什么先落盘再返回路径
 *
 * 调用方（内核的 `handlePaste`）拿到路径才插节点。反过来先插节点再落盘的话，
 * 落盘失败时文档里会留下一个指向不存在文件的图片节点 —— 用户保存之后
 * 得到一份「打开就有破图」的文档。**先写文件，再改文档。**
 */
export async function writePastedImage(
  provider: DocumentProvider,
  documentPath: string,
  options: PasteImageOptions,
): Promise<PasteImageResult> {
  // 不认识的格式一律拒绝，不猜一个后缀。猜错的后果是「文件叫 a.png，
  // 内容其实是 TIFF」—— 本地看起来正常（我们自己按 MIME 显示），
  // 发给别人就打不开，而那时已经无从追查
  if (imageExtensionFor(options.mimeType) === null) {
    const shown = options.mimeType.trim() === '' ? t('core.images.unknownType') : options.mimeType;
    return { ok: false, reason: t('core.images.unknownTypeDetail', { shown }) };
  }

  const directory = assetsDirFor(documentPath);
  if (directory === null) {
    // 没有目录信息 = 这份文档还没保存过（浏览器预览虚拟盘、新建未保存的文档）
    return {
      ok: false,
      reason: t('core.images.noLocation'),
    };
  }

  const directoryName = assetsDirNameFor(documentPath);
  let lastReason = '';

  for (let attempt = 0; attempt < WRITE_ATTEMPTS; attempt += 1) {
    const fileName = imageFileName({ mimeType: options.mimeType, at: options.at, salt: options.salt() });
    // 上面已经确认过 MIME 认得，所以这里拿不到扩展名是不可能的；
    // 真拿到了就如实失败，而不是编一个后缀
    if (fileName === null) return { ok: false, reason: t('core.images.nameFailed') };

    const target = joinPath(directory, fileName);
    if (target === null) return { ok: false, reason: t('core.images.targetFailed') };

    try {
      const outcome = await provider.writeBinaryFile(target, options.bytes);

      // 落盘字节数与内存里不一致 —— 说明写入被截断了。不检查的话，
      // 用户会得到一张显示成破图（或更糟：只显示上半截）的图片，
      // 而没有任何线索指向「写入出了问题」。这里如实说出来，并给出路径，
      // 让用户能自己去删掉那个坏文件
      if (outcome.size !== options.bytes.byteLength) {
        return {
          ok: false,
          reason: t('core.images.partialWrite', { size: outcome.size, total: options.bytes.byteLength, path: outcome.path }),
        };
      }

      return { ok: true, src: imageMarkdownSrc(directoryName, fileName), path: outcome.path, size: outcome.size };
    } catch (thrown) {
      const error = asAppError(thrown);
      lastReason = error.message;

      // 只有「已存在」才值得换名字重试。权限不足、磁盘满了之类的错误
      // 换多少次名字都一样，继续循环只会拖慢失败的反馈
      if (error.code !== 'CONFLICT') break;
    }
  }

  return { ok: false, reason: t('core.images.writeFailed', { reason: lastReason }) };
}
