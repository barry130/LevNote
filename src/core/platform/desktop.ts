/**
 * 桌面端平台实现。
 *
 * 这个文件是**唯一**允许 import `@tauri-apps/*` 的地方（硬约束 1）。
 * 它现在装两条互相独立的链路：
 *
 * - `createDesktopProvider()` —— 文档读写（`DocumentProvider`）；
 * - `createDesktopSearcher()` —— 工作区检索（`WorkspaceSearcher`）。
 *
 * 两者**刻意不合成一个接口**：浏览器预览有内存虚拟盘（文档那条能用），
 * 但它没有目录遍历（检索那条只能用不了）。合成一个的话，
 * 「能不能检索」会变成一个需要逐个方法探测的问题。
 *
 * 为什么不用 `@tauri-apps/plugin-fs`：
 * - 我们需要「编码探测 + 原子写入 + mtime 冲突检测」，插件给的是通用读写，
 *   这些语义要额外绕一圈才能拼出来；
 * - 自带命令的权限由我们自己的 capability 收口，比开放一整个 fs scope 更紧。
 */

import { Channel, invoke } from '@tauri-apps/api/core';
import { confirm, message, open, save } from '@tauri-apps/plugin-dialog';

import { asAppError, type BinaryWriteOutcome, type DirEntry, type DocumentProvider, type FileStat, type RuntimeInfo, type TextFile, type WriteOutcome, type WriteRequest } from '../document/types.js';
import type {
  WorkspaceBatch,
  WorkspaceSearchRequest,
  WorkspaceSearcher,
  WorkspaceSummary,
} from '../search/workspace.js';

const TEXT_EXTENSIONS = ['md', 'markdown', 'txt', 'json', 'js', 'ts', 'tsx', 'jsx', 'css', 'html', 'xml', 'drawio', 'mindmap', 'yml', 'yaml', 'toml', 'rs', 'py', 'go', 'java', 'c', 'h', 'cpp', 'sh', 'sql', 'log', 'csv'];

/**
 * 二进制写入时携带目标路径的请求头。
 *
 * ## 为什么路径走请求头，而不是放进请求体
 *
 * 这个命令的请求体**整段都是文件字节**（`content-type: application/octet-stream`
 * → Rust 侧 `InvokeBody::Raw`）。把路径也塞进请求体就要么自己设计一个
 * 「长度前缀 + 路径 + 字节」的容器格式，要么退回去用 JSON 传 base64 ——
 * 后者会让 4MB 的图膨胀成 5.5MB 的字符串，还要在两边各做一次编解码。
 * 请求头是唯一还能放「元数据」的地方，而且 Tauri 会原样转发自定义头。
 *
 * ## 为什么值要 percent-encode
 *
 * HTTP 头是字节串，惯例上只放 ASCII。路径里出现中文（`C:\笔记\…`）或
 * 换行这类字符时，直接塞进去会被 WebView 的 fetch 拒绝或截断。
 * `encodeURIComponent` 是 JS 侧唯一**逐字节可逆**且不需要新依赖的编码，
 * Rust 侧的 `percent_decode` 与它一一对应。
 */
const PATH_HEADER = 'x-qm-path';

export function createDesktopProvider(): DocumentProvider {
  return {
    kind: 'desktop',

    async readTextFile(path, forcedEncoding) {
      return invoke<TextFile>('read_text_file', { path, forcedEncoding: forcedEncoding ?? null });
    },

    async writeTextFile(request: WriteRequest) {
      return invoke<WriteOutcome>('write_text_file', { req: request });
    },

    async statFile(path) {
      return invoke<FileStat>('stat_file', { path });
    },

    async listDir(path) {
      return invoke<DirEntry[]>('list_dir', { path });
    },

    async sniffEncoding(path) {
      return invoke<string>('sniff_encoding', { path });
    },

    // Rust 侧用 `ipc::Response` 直接回原始字节，这里拿到的是 ArrayBuffer
    async readBinaryFile(path) {
      return invoke<ArrayBuffer>('read_binary_file', { path });
    },

    /**
     * 写二进制文件：请求体是原始字节，路径走 `x-qm-path` 头。
     *
     * `invoke` 的第三个参数就是转发给 `fetch` 的选项，其中的 `headers`
     * 会被 Tauri 注入的 IPC 层并进那次 `fetch` 的请求头里 —— 所以自定义头
     * 是真的会到达 Rust 侧的 `request.headers()`，不是被丢掉。
     *
     * **不要**改成把 `bytes` 包进对象（`{ bytes }`）：那样 IPC 会走 JSON 分支，
     * Rust 侧拿到的是 `InvokeBody::Json`，命令会明确报错拒绝 —— 这是有意的，
     * 见 `write_binary_file` 里那条 `let ... else`。
     */
    async writeBinaryFile(path, bytes) {
      return invoke<BinaryWriteOutcome>('write_binary_file', bytes, {
        headers: { [PATH_HEADER]: encodeURIComponent(path) },
      });
    },

    async runtimeInfo() {
      return invoke<RuntimeInfo>('runtime_info');
    },

    async pickOpenPath() {
      const picked = await open({
        multiple: false,
        directory: false,
        title: '打开文件',
        filters: [
          { name: '轻墨支持的文档', extensions: [...TEXT_EXTENSIONS, 'drawio'] },
          { name: 'Markdown', extensions: ['md', 'markdown'] },
          { name: '思维导图', extensions: ['mindmap'] },
          { name: 'draw.io 图表', extensions: ['drawio', 'xml'] },
          { name: '全部文件', extensions: ['*'] },
        ],
      });
      // 插件在用户取消时返回 null；多选模式下是数组，这里已限定单选
      if (picked === null) return null;
      return typeof picked === 'string' ? picked : (picked[0] ?? null);
    },

    async pickSavePath(suggestedName) {
      const picked = await save({ title: '另存为', defaultPath: suggestedName });
      return picked ?? null;
    },

    async pickDirectoryPath() {
      const picked = await open({ multiple: false, directory: true, title: '打开文件夹' });
      if (picked === null) return null;
      return typeof picked === 'string' ? picked : (picked[0] ?? null);
    },

    async notify(text, title = '轻墨') {
      try {
        await message(text, { title, kind: 'info' });
      } catch (error) {
        // 提示框失败不该影响主流程
        console.warn('弹出提示框失败', asAppError(error));
      }
    },
  };
}

/** 暴露给 UI 的「确认」对话框，语义比原生 confirm 更明确。 */
export async function askConfirm(text: string, title = '轻墨'): Promise<boolean> {
  return confirm(text, { title, kind: 'warning' });
}

// --------------------------------------------------------------------- 检索

/**
 * 工作区检索的桌面端实现。
 *
 * ## 通道只管批次，结局是返回值
 *
 * `onEvent` 上推的是 `SearchBatch`，而**这一次检索怎么结束的**是 `invoke` 的返回值。
 * 两条路的分工写死在 Rust 侧的 `search/mod.rs`：结局只有一个来源，
 * 前端不必去猜「promise 与通道事件谁先到」。
 *
 * ## 为什么不用 `listen` / 事件系统
 *
 * `Channel` 是**这一次调用专用**的：它随请求一起传下去，检索结束就自动清理，
 * 不需要在全局事件总线上起一个名字 —— 而全局事件名会带来「上一次检索的监听
 * 还没摘干净」这类竞态。另外 `Channel` 自带消息序号（Tauri 的 JS 实现里
 * 按 index 排队），批次不会乱序到达。
 */
export function createDesktopSearcher(): WorkspaceSearcher {
  return {
    available: true,

    async run(request: WorkspaceSearchRequest, onBatch: (batch: WorkspaceBatch) => void) {
      const channel = new Channel<WorkspaceBatch>();
      // 必须在 `invoke` 之前挂上：通道不重放「挂之前」的消息（Tauri 的 JS 实现如此）
      channel.onmessage = (batch) => {
        onBatch(batch);
      };
      return invoke<WorkspaceSummary>('search_in_workspace', { request, onEvent: channel });
    },

    async cancel(searchId: number) {
      // 返回值是「这个 id 现在有没有在跑的检索」。`false` 多半意味着
      // 「取消得比注册还早」—— 那一次取消已经被 Rust 侧记下，会在检索开始时生效。
      await invoke<boolean>('cancel_search', { searchId });
    },
  };
}
