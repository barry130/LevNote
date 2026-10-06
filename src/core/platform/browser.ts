/**
 * 浏览器端文档来源：内存虚拟盘 + 真实的本机文件读写。
 *
 * 定位：**开发期兜底**，不是产品形态（本项目只出桌面版）。
 * 有了它，`npm run dev` 不启动 Tauri 就能调界面，不必为了改一个按钮去编译 Rust。
 * 因此它刻意不做原子写入、不做 mtime 语义，只保证接口形状一致。
 *
 * 打开：走 `<input type="file">`，支持 UTF-8 / UTF-16 / GBK 的粗略探测；
 * 保存：走 Blob 下载。两者的行为差异在 UI 上会明说，避免误以为已在写磁盘。
 */

import {
  AppError,
  type BinaryWriteOutcome,
  type BomKind,
  type DirEntry,
  type DocumentProvider,
  type FileStat,
  type RuntimeInfo,
  type TextFile,
  type WriteRequest,
} from '../document/types.js';

interface VirtualFile {
  text: string;
  encoding: string;
  bom: BomKind;
  mtimeMs: number;
  /** 是否来自用户真实选择的本地文件（决定保存时是否触发下载） */
  fromDisk: boolean;
}

const SEED: Array<[string, string]> = [
  [
    'demo.md',
    [
      '# 轻墨 · 试笔',
      '',
      '这是一个**内存虚拟文件**（浏览器预览模式）。桌面版会直接读写磁盘。',
      '',
      '```doc-block',
      'id: arch',
      'kind: diagram',
      'mode: reference',
      'src: diagrams/arch.drawio',
      'title: 整体架构',
      '---',
      '```',
      '',
      '正文继续。',
      '',
    ].join('\n'),
  ],
];

function decodeBytes(bytes: Uint8Array): { text: string; encoding: string; bom: BomKind; lossy: boolean } {
  if (bytes.length >= 3 && bytes[0] === 0xef && bytes[1] === 0xbb && bytes[2] === 0xbf) {
    return { text: new TextDecoder('utf-8').decode(bytes.subarray(3)), encoding: 'UTF-8', bom: 'utf-8', lossy: false };
  }
  if (bytes.length >= 2 && bytes[0] === 0xff && bytes[1] === 0xfe) {
    return { text: new TextDecoder('utf-16le').decode(bytes.subarray(2)), encoding: 'UTF-16LE', bom: 'utf-16le', lossy: false };
  }
  if (bytes.length >= 2 && bytes[0] === 0xfe && bytes[1] === 0xff) {
    return { text: new TextDecoder('utf-16be').decode(bytes.subarray(2)), encoding: 'UTF-16BE', bom: 'utf-16be', lossy: false };
  }

  // 严格 UTF-8：能解码就说 UTF-8，否则按中文场景依次退到 GBK / windows-1252
  try {
    return { text: new TextDecoder('utf-8', { fatal: true }).decode(bytes), encoding: 'UTF-8', bom: 'none', lossy: false };
  } catch {
    /* 落到下面的猜测 */
  }
  for (const label of ['gbk', 'big5']) {
    try {
      const text = new TextDecoder(label, { fatal: true }).decode(bytes);
      return { text, encoding: label.toUpperCase(), bom: 'none', lossy: false };
    } catch {
      /* 继续尝试 */
    }
  }
  return { text: new TextDecoder('windows-1252').decode(bytes), encoding: 'windows-1252', bom: 'none', lossy: true };
}

function detectLineEnding(text: string): TextFile['lineEnding'] {
  const crlf = (text.match(/\r\n/g) ?? []).length;
  const loneLf = (text.match(/(?<!\r)\n/g) ?? []).length;
  const loneCr = (text.match(/\r(?!\n)/g) ?? []).length;
  if (crlf > 0 && (loneLf > 0 || loneCr > 0)) return 'mixed';
  if (crlf > 0) return 'crlf';
  if (loneCr > 0) return 'cr';
  return 'lf';
}

function encodeText(text: string, encoding: string, bom: BomKind): BlobPart[] {
  const normalized = encoding.toUpperCase();
  const prefix: number[] =
    bom === 'utf-8' ? [0xef, 0xbb, 0xbf] : bom === 'utf-16le' ? [0xff, 0xfe] : bom === 'utf-16be' ? [0xfe, 0xff] : [];

  const parts: BlobPart[] = [];
  if (prefix.length > 0) parts.push(new Uint8Array(prefix));

  if (normalized.includes('UTF-16')) {
    const body = new Uint8Array(text.length * 2);
    const view = new DataView(body.buffer);
    const little = normalized.includes('LE');
    for (let i = 0; i < text.length; i += 1) view.setUint16(i * 2, text.charCodeAt(i), little);
    parts.push(body);
    return parts;
  }

  parts.push(new TextEncoder().encode(text));
  return parts;
}

export function createBrowserProvider(): DocumentProvider {
  const files = new Map<string, VirtualFile>();
  /**
   * 二进制虚拟盘。**只活在内存里，刷新即失** —— 与文本虚拟盘一样是开发期道具。
   *
   * 为什么不留空、直接抛 NOT_SUPPORTED：粘贴图片是**看得见**的功能。
   * 预览模式下拒收字节的话，`npm run dev` 里就完全走不到「粘一张图 →
   * 占位消失 → 图出现」这条链路，而那正是最需要眼睛确认的一段。
   * 存进内存后 `readBinaryFile` 能原样读回来，整条显示链路就能跑通。
   *
   * 代价必须说清楚：**这些字节不会被写进任何文件**，刷新页面就没了。
   * 所以它替代不了桌面版，只是让界面开发不用为了看一眼效果去编译 Rust。
   */
  const binaries = new Map<string, ArrayBuffer>();
  const now = () => Date.now();

  for (const [name, text] of SEED) {
    files.set(name, { text, encoding: 'UTF-8', bom: 'none', mtimeMs: now(), fromDisk: false });
  }

  const ensure = (path: string): VirtualFile => {
    const found = files.get(path);
    if (!found) {
      throw new AppError('NOT_FOUND', `（浏览器预览）虚拟盘里没有 ${path}`);
    }
    return found;
  };

  const download = (name: string, parts: BlobPart[]) => {
    const url = URL.createObjectURL(new Blob(parts, { type: 'application/octet-stream' }));
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = name;
    anchor.click();
    URL.revokeObjectURL(url);
  };

  return {
    kind: 'browser',

    async readTextFile(path) {
      const file = ensure(path);
      return {
        path,
        text: file.text,
        encoding: file.encoding,
        bom: file.bom,
        lineEnding: detectLineEnding(file.text),
        size: new Blob([file.text]).size,
        mtimeMs: file.mtimeMs,
        readonly: false,
        lossy: false,
      };
    },

    async writeTextFile(request: WriteRequest) {
      const previous = files.get(request.path);
      if (request.expectedMtimeMs !== undefined && previous && previous.mtimeMs !== request.expectedMtimeMs) {
        throw new AppError(
          'CONFLICT',
          '（浏览器预览）文件已被外部修改，本次保存已取消',
          `expected ${request.expectedMtimeMs}, actual ${previous.mtimeMs}`,
        );
      }

      const encoding = request.encoding ?? previous?.encoding ?? 'UTF-8';
      const bom = request.bom ?? previous?.bom ?? ('none');
      let text = request.text;
      if (request.lineEnding) {
        const unified = text.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
        text = request.lineEnding === 'crlf' ? unified.replace(/\n/g, '\r\n') : request.lineEnding === 'cr' ? unified.replace(/\n/g, '\r') : unified;
      }

      const mtimeMs = now();
      files.set(request.path, { text, encoding, bom, mtimeMs, fromDisk: previous?.fromDisk ?? false });

      if (previous?.fromDisk) {
        // 来自用户真实选择的本机文件：明确用下载表示「保存」，而不是假装写回了磁盘
        download(request.path.split(/[\\/]/).pop() ?? 'untitled.txt', encodeText(text, encoding, bom));
      }

      return {
        path: request.path,
        size: new Blob([text]).size,
        mtimeMs,
        encoding,
        bom,
        lineEnding: detectLineEnding(text),
      };
    },

    async statFile(path): Promise<FileStat> {
      const file = ensure(path);
      return { path, size: new Blob([file.text]).size, mtimeMs: file.mtimeMs, readonly: false, isDir: false };
    },

    async listDir(): Promise<DirEntry[]> {
      return [...files.entries()]
        .map(([name, file]) => ({ name, path: name, isDir: false, size: new Blob([file.text]).size, mtimeMs: file.mtimeMs }))
        .sort((a, b) => a.name.localeCompare(b.name));
    },

    async sniffEncoding(path) {
      return ensure(path).encoding;
    },

    // 先看内存里的二进制盘（粘贴进来的图片就在那儿）。
    // 拿不到才抛错 —— 导出层据此退化为「保留原链接并标注」，
    // 好过生成一张 0 字节的破图。
    async readBinaryFile(path) {
      const hit = binaries.get(path);
      if (hit !== undefined) return hit.slice(0);

      throw new AppError(
        'NOT_SUPPORTED',
        `（浏览器预览）无法读取二进制文件 ${path}`,
        '浏览器预览没有磁盘，只能读到本次会话里粘贴进来的图片；真实文件请在桌面版里试',
      );
    },

    /**
     * 收下字节，放进内存二进制盘。语义与桌面版对齐：**已存在就拒绝**。
     *
     * 对齐这一条不是形式主义 —— 如果预览模式允许覆盖、桌面版拒绝，
     * 那么「撞名」这条分支在开发期永远走不到，等到用户那儿才第一次触发。
     * 返回的形状也与 Rust 的 `BinaryWriteOutcome` 一致。
     */
    async writeBinaryFile(path, bytes): Promise<BinaryWriteOutcome> {
      if (binaries.has(path)) {
        throw new AppError(
          'CONFLICT',
          `（浏览器预览）${path} 已存在，为避免覆盖已有内容，本次写入已取消`,
        );
      }
      if (bytes.byteLength === 0) throw new AppError('INVALID_PATH', '拒绝写入空文件');

      binaries.set(path, bytes.slice(0));
      return { path, size: bytes.byteLength, mtimeMs: now() };
    },

    async runtimeInfo(): Promise<RuntimeInfo> {
      return { name: 'qingmo', version: '0.1.0-preview', os: navigator.platform || 'browser', arch: 'web', nowMs: now() };
    },

    async pickOpenPath() {
      return new Promise<string | null>((resolve) => {
        const input = document.createElement('input');
        input.type = 'file';
        input.accept = '.md,.markdown,.txt,.json,.js,.ts,.xml,.drawio,.mindmap,.yml,.yaml,text/*';
        input.onchange = async () => {
          const picked = input.files?.[0];
          if (!picked) {
            resolve(null);
            return;
          }
          const bytes = new Uint8Array(await picked.arrayBuffer());
          const decoded = decodeBytes(bytes);
          files.set(picked.name, {
            text: decoded.text,
            encoding: decoded.encoding,
            bom: decoded.bom,
            mtimeMs: now(),
            fromDisk: true,
          });
          resolve(picked.name);
        };
        // cancel 事件在部分 TS lib 里还没进 EventMap，用属性赋值兜住
        (input as HTMLInputElement & { oncancel?: () => void }).oncancel = () => resolve(null);
        input.click();
      });
    },

    async pickSavePath(suggestedName) {
      return window.prompt('（浏览器预览）保存为：', suggestedName);
    },

    // 浏览器预览没有真实目录树（虚拟盘是扁平的），如实返回 null 并说明，
    // 而不是造一个假的目录出来 —— 侧栏会显示「没有打开文件夹」，这是对的
    async pickDirectoryPath() {
      window.alert('（浏览器预览）不支持打开文件夹。请用桌面版，或直接打开单个文件。');
      return null;
    },

    async notify(text, title = '轻墨') {
      console.info(`[${title}] ${text}`);
      window.alert(text);
    },
  };
}

/**
 * 浏览器预览的全屏切换：Fullscreen API。
 *
 * 与桌面端（Tauri 的窗口 API）不同源是**有意的** —— 浏览器里根本没有
 * 「窗口」可全屏，铺满视口就是这里能做到的全部。用户手势要求由浏览器
 * 自己执行（菜单点击就是手势），拒绝时如实上抛，由调用方说明。
 */
export async function toggleFullscreenBrowser(): Promise<void> {
  if (document.fullscreenElement) {
    await document.exitFullscreen();
    return;
  }
  await document.documentElement.requestFullscreen();
}
