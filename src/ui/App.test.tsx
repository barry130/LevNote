// @vitest-environment jsdom
/**
 * 外壳接线测试 —— 第一次真正执行 `App` + `DocumentStore` + 编辑器内核这条完整链路。
 *
 * 前面那些测试各自把一块钉住了（存储语义、注册表、导出、内核保真度），
 * 但它们都不覆盖**它们之间的接缝**，而接缝正是这类应用最容易出问题的地方：
 *
 * - 内核是异步创建的，外壳的同步 effect 可能先跑，操作到一个还没生成、
 *   或者已经销毁的实例上；
 * - 「内核报的变更」与「外壳推的变更」如果不区分，就会成环（无限重渲染）或者
 *   把外部变更吞掉；
 * - `main.tsx` 用了 `StrictMode`，开发模式下 effect 会被双调用 ——
 *   异步创建的内核在「创建完 → 发现已卸载 → 销毁 → 再创建一次」这条路径上极易出问题。
 *
 * 所以这里不是「再测一遍单测测过的东西」，而是专门打这些接缝。
 */

import { StrictMode } from 'react';
import type { ReactElement } from 'react';
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import type { DocumentStore } from '../core/document/store.js';
import { DEFAULT_SETTINGS, type ViewSettings } from '../core/settings.js';
import { WorkspaceStore } from '../core/workspace/store.js';
import type {
  BomKind,
  DirEntry,
  DocumentProvider,
  FileStat,
  RuntimeInfo,
  TextFile,
  WriteOutcome,
  WriteRequest,
} from '../core/document/types.js';
import { App } from './App.js';

declare global {
  // React 要求测试环境显式声明这一点，否则 act() 会警告并跳过批处理
  var IS_REACT_ACT_ENVIRONMENT: boolean | undefined;
}

// ---------------------------------------------------------------------------
// 假文档来源
// ---------------------------------------------------------------------------

interface FakeFile {
  text: string;
  encoding: string;
  bom: BomKind;
  mtimeMs: number;
  readonly: boolean;
}

class FakeProvider implements DocumentProvider {
  readonly kind = 'desktop' as const;
  readonly files = new Map<string, FakeFile>();
  readonly writes: WriteRequest[] = [];
  readonly notices: string[] = [];
  /** 导出时用的目标路径；null 模拟用户点了取消 */
  savePathTarget: string | null = 'C:\\out\\结果.html';
  private clock = 1_000;

  put(path: string, text: string, options: Partial<FakeFile> = {}): void {
    this.files.set(path, {
      text,
      encoding: options.encoding ?? 'UTF-8',
      bom: options.bom ?? 'none',
      mtimeMs: options.mtimeMs ?? (this.clock += 10),
      readonly: options.readonly ?? false,
    });
  }

  private need(path: string): FakeFile {
    const file = this.files.get(path);
    if (!file) throw { code: 'NOT_FOUND', message: `没有这个文件：${path}`, detail: null };
    return file;
  }

  async readTextFile(path: string): Promise<TextFile> {
    const file = this.need(path);
    return {
      path,
      text: file.text,
      encoding: file.encoding,
      bom: file.bom,
      lineEnding: 'lf',
      size: file.text.length,
      mtimeMs: file.mtimeMs,
      readonly: file.readonly,
      lossy: false,
    };
  }

  async writeTextFile(request: WriteRequest): Promise<WriteOutcome> {
    this.writes.push(request);
    const mtimeMs = (this.clock += 10);
    this.files.set(request.path, {
      text: request.text,
      encoding: request.encoding ?? 'UTF-8',
      bom: request.bom ?? 'none',
      mtimeMs,
      readonly: false,
    });
    return { path: request.path, size: request.text.length, mtimeMs, encoding: 'UTF-8', bom: 'none', lineEnding: 'lf' };
  }

  async readBinaryFile(path: string): Promise<ArrayBuffer> {
    // 一张最小 PNG 的文件头，足以让 MIME 嗅探认出 image/png
    void this.need(path);
    return new Uint8Array([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]).buffer;
  }

  async statFile(path: string): Promise<FileStat> {
    const file = this.need(path);
    return { path, size: file.text.length, mtimeMs: file.mtimeMs, readonly: file.readonly, isDir: false };
  }

  async listDir(): Promise<DirEntry[]> {
    return [...this.files.keys()].map((name) => ({ name, path: name, isDir: false, size: 0, mtimeMs: 0 }));
  }

  async sniffEncoding(path: string): Promise<string> {
    return this.need(path).encoding;
  }

  async runtimeInfo(): Promise<RuntimeInfo> {
    return { name: 'qingmo', version: 'test', os: 'win32', arch: 'x64', nowMs: this.clock };
  }

  async pickOpenPath(): Promise<string | null> {
    return null;
  }

  async pickSavePath(): Promise<string | null> {
    return this.savePathTarget;
  }

  async pickDirectoryPath(): Promise<string | null> {
    return null;
  }

  async notify(text: string): Promise<void> {
    this.notices.push(text);
  }
}

// ---------------------------------------------------------------------------
// 渲染脚手架
// ---------------------------------------------------------------------------

/**
 * 让异步的内核创建、以及它引发的级联重渲染都跑完。
 *
 * 次数不是随便写的：内核创建是 `await` 一次 Promise 链，之后 `onReady`
 * 会推一次状态更新。三轮足够覆盖，写多了只是白等。
 */
async function settle(rounds = 3): Promise<void> {
  for (let i = 0; i < rounds; i += 1) {
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });
  }
}

/**
 * 测试用的固定偏好。
 *
 * 刻意**不走 localStorage**：偏好是会持久化的，让测试去读写真实存储
 * 会让用例之间互相影响（前一个用例把侧栏关掉，后一个就找不到它了）。
 * 注入固定值之后，每个用例拿到的初始视图完全一样。
 */
const TEST_SETTINGS: ViewSettings = { ...DEFAULT_SETTINGS, sidebarVisible: false };

interface Mounted {
  container: HTMLElement;
  provider: FakeProvider;
  workspace: WorkspaceStore;
  /** 打开一个文件。走工作区，因此会新开一个标签页。 */
  open(path: string): Promise<boolean>;
  /** 当前活动标签页的文档 store。没有活动标签页时抛错（说明用例写错了）。 */
  readonly store: DocumentStore;
  unmount(): Promise<void>;
  /** 按按钮上的文字点一下 */
  click(text: string): Promise<void>;
}

async function mount(
  seed: (provider: FakeProvider) => void,
  options: { strict?: boolean } = {},
): Promise<Mounted> {
  const provider = new FakeProvider();
  seed(provider);
  const workspace = new WorkspaceStore(provider);

  const container = document.createElement('div');
  document.body.appendChild(container);
  const root: Root = createRoot(container);

  const element: ReactElement = (
    <App workspace={workspace} provider={provider} initialSettings={TEST_SETTINGS} />
  );
  await act(async () => {
    root.render(options.strict === false ? element : <StrictMode>{element}</StrictMode>);
  });
  await settle();

  return {
    container,
    provider,
    workspace,
    open: async (path: string) => {
      let ok = false;
      await act(async () => {
        ok = await workspace.open(path);
      });
      await settle();
      return ok;
    },
    get store() {
      const store = workspace.activeStore;
      if (store === null) throw new Error('当前没有活动标签页 —— 用例是不是忘了先 open？');
      return store;
    },
    click: async (text: string) => {
      const button = [...container.querySelectorAll('button')].find((item) => item.textContent?.trim() === text);
      if (!button) throw new Error(`找不到按钮：${text}`);
      await act(async () => {
        button.click();
      });
      await settle();
    },
    unmount: async () => {
      await act(async () => {
        root.unmount();
      });
      container.remove();
    },
  };
}

function statusOf(container: HTMLElement): string {
  return container.querySelector('.status-pill')?.textContent?.trim() ?? '';
}

function kernelOf(container: HTMLElement): string {
  return container.querySelector('.statusbar .kernel')?.textContent?.trim() ?? '';
}

function statisticsOf(container: HTMLElement): string {
  return container.querySelector('.statusbar .statistics')?.textContent?.trim() ?? '';
}

/**
 * 工具栏按钮。
 *
 * 与 `mounted.click` 不同，这里要的是**元素**而不是「点一下」——
 * 「开关被禁用而不是被隐藏」这条断言必须能看到那个禁用按钮。
 */
function toolbarButton(container: HTMLElement, text: string): HTMLButtonElement | null {
  const found = [...container.querySelectorAll<HTMLButtonElement>('.toolbar button')].find(
    (item) => item.textContent?.trim() === text,
  );
  return found ?? null;
}

/** 内核实际渲染出的内容：富文本看 ProseMirror，源码看 CodeMirror。 */
function renderedText(container: HTMLElement): string {
  const rich = container.querySelector('.qm-editor .ProseMirror');
  if (rich) return rich.textContent ?? '';
  return container.querySelector('.qm-code .cm-content')?.textContent ?? '';
}

/**
 * 等一帧。
 *
 * 选区上报是用 `requestAnimationFrame` 合并的（拖动选择时每个 mousemove 都会
 * 触发一次，逐次去量会强制布局、把拖选拖成幻灯片）。所以这里必须真的等一帧，
 * 而不是 `settle()` 那三轮 `setTimeout(0)` —— 那三轮根本走不到 rAF。
 */
async function nextFrame(): Promise<void> {
  await act(async () => {
    await new Promise((resolve) => {
      requestAnimationFrame(() => resolve(undefined));
    });
  });
}

/**
 * 在编辑器里选中一段文字。
 *
 * 走**真实的 DOM 选区**再派发 `selectionchange`，而不是去够 ProseMirror 的内部对象。
 * 这正是浏览器里发生的事，所以顺带验证了「ProseMirror 认这个选区」这一步 ——
 * 直接构造 `TextSelection` 会跳过它，而跳过的恰恰是最容易坏的一环。
 *
 * 同时认两种内核的 contenteditable：`.ProseMirror`（富文本）与 `.cm-content`（代码）。
 * 测试「代码内核没有浮层」时需要在 CodeMirror 里真的选中一段文字，
 * 否则那条用例会退化成「没选区所以没浮层」，什么都没验证到。
 */
function editableOf(container: HTMLElement): HTMLElement {
  const root = container.querySelector<HTMLElement>('.qm-editor .ProseMirror, .qm-code .cm-content');
  if (root === null) throw new Error('没有找到可编辑区 —— 用例是不是忘了先 open？');
  return root;
}

async function selectText(container: HTMLElement, needle: string): Promise<void> {
  const editable = editableOf(container);
  editable.focus();

  const walker = document.createTreeWalker(editable, NodeFilter.SHOW_TEXT);
  let hit: { node: Text; offset: number } | null = null;
  while (walker.nextNode()) {
    const node = walker.currentNode as Text;
    const offset = node.data.indexOf(needle);
    if (offset >= 0) {
      hit = { node, offset };
      break;
    }
  }
  if (hit === null) throw new Error(`编辑器里找不到「${needle}」`);

  const range = document.createRange();
  range.setStart(hit.node, hit.offset);
  range.setEnd(hit.node, hit.offset + needle.length);
  const selection = window.getSelection();
  if (selection === null) throw new Error('测试环境没有 Selection');
  selection.removeAllRanges();
  selection.addRange(range);

  await act(async () => {
    document.dispatchEvent(new Event('selectionchange'));
  });
  await nextFrame();
}

/**
 * 把选区收起来 —— 模拟「用户点了一下，光标塌缩」。
 *
 * ## 两个都必须做对，少一个这条用例就变成假绿
 *
 * 1. **不能只 `removeAllRanges()`**。那样 ProseMirror 会看到「焦点不在自己身上」，
 *    直接跳过这次 `selectionchange`，连事务都不派发 —— 内核根本不知道选区没了。
 *    浏览器里用户点掉选中时，给的是「编辑器里一个塌缩的光标」，这里必须复现那一步。
 *
 * 2. **塌缩点不能落在文档最开头**。ProseMirror 的 `DOMObserver.flush()` 里有一条：
 *    若「刚获得焦点 200ms 内」+「选区塌缩」+「塌缩到了 `Selection.near(doc.resolve(0))`」，
 *    它认定这是**浏览器自作主张把选区重置到了文档开头**，于是把 state 里的选区
 *    **写回 DOM**、丢弃这次塌缩。这是真实浏览器里存在的现象，PM 处理得对；
 *    但我们的测试如果正好塌缩到那里，就会看到「选了、取消了、浮层还在」，
 *    然后去怀疑一个根本没坏的实现。所以这里塌缩到词尾。
 */
async function clearSelection(container: HTMLElement, after: string): Promise<void> {
  const editable = editableOf(container);

  const walker = document.createTreeWalker(editable, NodeFilter.SHOW_TEXT);
  let hit: { node: Text; offset: number } | null = null;
  while (walker.nextNode()) {
    const node = walker.currentNode as Text;
    const offset = node.data.indexOf(after);
    if (offset >= 0) {
      hit = { node, offset };
      break;
    }
  }
  if (hit === null) throw new Error(`可编辑区里找不到「${after}」`);

  const range = document.createRange();
  range.setStart(hit.node, hit.offset + after.length);
  range.collapse(true);
  const selection = window.getSelection();
  selection?.removeAllRanges();
  selection?.addRange(range);

  await act(async () => {
    document.dispatchEvent(new Event('selectionchange'));
  });
  await nextFrame();
}

function toolbarOf(container: HTMLElement): HTMLElement | null {
  return container.querySelector('.selection-toolbar');
}

/**
 * 浮层上的按钮。
 *
 * 按 `aria-label` 找而不是按文字 —— 按钮上写的是 `B` / `I` / `•` 这类符号，
 * 文字会随样式调整，而 `aria-label` 是给读屏用的、不该跟着改。
 */
function formatButton(container: HTMLElement, label: string): HTMLButtonElement | null {
  return container.querySelector<HTMLButtonElement>(`.selection-toolbar button[aria-label="${label}"]`);
}

/** 点浮层上的某个格式按钮，并等一帧让内核把新状态报回来。 */
async function clickFormat(container: HTMLElement, label: string): Promise<void> {
  const button = formatButton(container, label);
  if (button === null) throw new Error(`浮层上没有「${label}」`);
  await act(async () => {
    button.click();
  });
  await nextFrame();
}

let mounted: Mounted | null = null;

beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
});

afterEach(async () => {
  await mounted?.unmount();
  mounted = null;
  document.body.innerHTML = '';
});

// ---------------------------------------------------------------------------
// 用例
// ---------------------------------------------------------------------------

describe('空态', () => {
  it('没有打开文档时给出空态，并且没有内核', async () => {
    mounted = await mount(() => undefined);

    expect(mounted.container.textContent).toContain('还没有打开文档');
    expect(kernelOf(mounted.container)).toBe('');
    expect(mounted.container.querySelector('.ProseMirror')).toBeNull();
  });
});

describe('内核选择', () => {
  it('.md 走 Markdown 内核，并真的把编辑器挂起来了', async () => {
    mounted = await mount((p) => p.put('note.md', '# 标题\n\n正文\n'));
    await mounted.open('note.md');

    expect(kernelOf(mounted.container)).toBe('Markdown');
    // 关键：这不只是「组件挂上了」，而是 Milkdown 真的建出了 ProseMirror 视图
    const prose = mounted.container.querySelector('.qm-editor .ProseMirror');
    expect(prose).not.toBeNull();
    expect(prose?.textContent).toContain('标题');
    // 共用版式类名必须挂上，否则编辑器里看到的和导出的不一样
    expect(prose?.classList.contains('qm-doc')).toBe(true);
  });

  it('带 YAML front matter 的 .md 从富文本降级到代码内核，并说明原因', async () => {
    mounted = await mount((p) => p.put('note.md', '---\ntitle: 笔记\ntags: [a]\n---\n\n# 正文\n'));
    await mounted.open('note.md');

    // 降级不是「掉到最底」：Markdown 富文本内核退位后，注册表继续往下找，
    // 代码内核接住了它 —— 用户拿到的是有行号、有语法着色的源码，
    // 而不是一个白底 textarea。这正是「四类内核」这个架构的价值。
    expect(kernelOf(mounted.container)).toBe('代码 / 文本');
    expect(mounted.container.textContent).toContain('YAML front matter');
    // 降级后内容仍然完整可见，元数据一个字都没被动
    const content = mounted.container.querySelector('.cm-content');
    expect(content?.textContent).toContain('title: 笔记');
    expect(content?.textContent).toContain('# 正文');
  });

  it('不认识的扩展名落到纯文本兜底 —— 任何文件都必须打得开', async () => {
    mounted = await mount((p) => p.put('blob.weird', '随便什么内容\n'));
    await mounted.open('blob.weird');

    expect(kernelOf(mounted.container)).toBe('纯文本');
    expect(mounted.container.querySelector('textarea')?.value).toBe('随便什么内容\n');
  });
});

describe('脏标记 —— 打开不动就不该脏', () => {
  /**
   * 这是整个 P2 最容易出错、后果也最实在的一条：
   * 打开文件没动过却显示「有未保存的改动」，会把用户引着去点保存，
   * 一保存就按内核的规范化把文件里他没碰过的排版改掉。
   *
   * 「以代码块结尾」这类文档尤其危险：`@milkdown/plugin-trailing` 会补一个空段落，
   * 那是一次真实的文档变更 —— 如果上报规则只看 `docChanged`，这里必然误判。
   */
  const cases: Array<[string, string]> = [
    ['普通段落结尾', '# 标题\n\n正文\n'],
    ['代码块结尾', '```js\nlet a = 1;\n```\n'],
    ['表格结尾', '| a | b |\n| - | - |\n| 1 | 2 |\n'],
    ['列表结尾', '- 一\n- 二\n'],
    ['分隔线结尾', '正文\n\n---\n'],
    ['空文档', ''],
  ];

  for (const [label, source] of cases) {
    it(`${label}：打开后状态是「已保存」`, async () => {
      mounted = await mount((p) => p.put('note.md', source));
      await mounted.open('note.md');

      expect(statusOf(mounted.container)).toBe('已保存');
      expect(mounted.store.getSnapshot().status).toBe('clean');
      // 标题栏上那个未保存的小圆点也不能出现
      expect(mounted.container.querySelector('.tabbar .dot')).toBeNull();
    });
  }

  it('内容原样留在内存里，没有被内核的规范化改写', async () => {
    const source = '标题\n====\n\n```\n裸代码\n```\n';
    mounted = await mount((p) => p.put('note.md', source));
    await mounted.open('note.md');

    // 规范化确实会把这篇文章改写成 ATX 标题，但那只在**用户真的改了之后**才落盘。
    // 此刻 store 里必须是磁盘上的原文。
    expect(mounted.store.getSnapshot().doc?.text).toBe(source);
  });
});

describe('StrictMode 下的生命周期', () => {
  it('双调用 effect 之后仍然只有一个编辑器实例，且状态正常', async () => {
    mounted = await mount((p) => p.put('note.md', '# 标题\n'));
    await mounted.open('note.md');

    const editors = mounted.container.querySelectorAll('.qm-editor');
    expect(editors).toHaveLength(1);
    expect(mounted.container.querySelectorAll('.ProseMirror')).toHaveLength(1);
    expect(statusOf(mounted.container)).toBe('已保存');
  });

  it('关闭（卸载）时不抛错', async () => {
    mounted = await mount((p) => p.put('note.md', '# 标题\n'));
    await mounted.open('note.md');

    await expect(mounted.unmount()).resolves.toBeUndefined();
    mounted = null;
  });
});

describe('自包含 HTML 导出', () => {
  it('从 Markdown 内核导出，产出完整 HTML 并落盘', async () => {
    mounted = await mount((p) => {
      // 用真实的绝对路径：桌面端的路径都来自系统对话框，永远是绝对的。
      // 相对路径的图片要靠 dirname 解析，这条顺带验证 Windows 的反斜杠拼接。
      p.put('C:\\notes\\note.md', '# 标题\n\n段落 **加粗**。\n\n```js\nconst a = 1;\n```\n\n![图](img/a.png)\n');
      p.put('C:\\notes\\img\\a.png', 'PNG 占位');
    });
    await mounted.open('C:\\notes\\note.md');

    await mounted.click('导出 HTML');

    expect(mounted.provider.writes).toHaveLength(1);
    const written = mounted.provider.writes[0]!;
    expect(written.path).toBe('C:\\out\\结果.html');

    const html = written.text;
    expect(html.startsWith('<!DOCTYPE html>')).toBe(true);
    expect(html).toContain('<article class="qm-doc">');
    expect(html).toContain('<h1');
    expect(html).toContain('<strong>加粗</strong>');
    // 代码高亮在导出侧单独走了一遍（装饰不参与序列化，见 inline-html.ts）
    expect(html).toContain('hljs-keyword');
    // 图片被内联成 data URI，而不是留一个相对路径
    expect(html).toContain('data:image/png;base64,');
    expect(html).not.toContain('src="img/a.png"');
    // 版式 CSS 被内联，而不是 <link>
    expect(html).toContain('--qm-ink');
    expect(html).not.toContain('<link');

    // 导出文件固定 UTF-8 无 BOM + LF，免得在别人机器上乱码
    expect(written.encoding).toBe('UTF-8');
    expect(written.bom).toBe('none');
    expect(written.lineEnding).toBe('lf');

    // 如实通报做了什么，而不是只说「导出成功」
    expect(mounted.provider.notices.join(' ')).toContain('1 张图片已内联');
    expect(mounted.provider.notices.join(' ')).toContain('1 个代码块已着色');
  });

  it('没有目录信息的路径（浏览器虚拟盘那种）不猜路径，如实标注未内联', async () => {
    mounted = await mount((p) => {
      p.put('note.md', '![图](img/a.png)\n');
      p.put('img/a.png', 'PNG 占位');
    });
    await mounted.open('note.md');

    await mounted.click('导出 HTML');

    const html = mounted.provider.writes[0]!.text;
    // 不能凭空把 img/a.png 当成可读路径去读 —— 那样只会读错文件或读不到
    expect(html).toContain('src="img/a.png"');
    expect(html).toContain('data-qm-missing');
    expect(mounted.provider.notices.join(' ')).toContain('读不到');
  });

  it('纯文本内核也说得出自己不支持富文本，退化为 <pre> 而不是假装成功', async () => {
    mounted = await mount((p) => p.put('note.weird', 'if (a < b) {\n  返回值\n}\n'));
    await mounted.open('note.weird');

    await mounted.click('导出 HTML');

    const html = mounted.provider.writes[0]!.text;
    expect(html).toContain('qm-plain');
    // 原文里的尖括号必须转义，否则导出文件被撕坏
    expect(html).toContain('if (a &lt; b) {');
    expect(mounted.provider.notices.join(' ')).toContain('按纯文本呈现');
  });

  it('用户取消保存对话框时什么都不写，也不弹假成功', async () => {
    mounted = await mount((p) => {
      p.put('note.md', '# 标题\n');
      p.savePathTarget = null;
    });
    await mounted.open('note.md');

    await mounted.click('导出 HTML');

    expect(mounted.provider.writes).toHaveLength(0);
    expect(mounted.provider.notices).toHaveLength(0);
  });

  it('图片读不到时如实说明，并把原链接留在 HTML 里', async () => {
    const provider = new FakeProvider();
    provider.put('note.md', '![丢失的图](missing.png)\n');
    // 让二进制读取失败，模拟文件不存在
    const original = provider.readBinaryFile.bind(provider);
    provider.readBinaryFile = async (path: string) => {
      if (path.includes('missing')) throw new Error('ENOENT');
      return original(path);
    };

    const workspace = new WorkspaceStore(provider);
    const container = document.createElement('div');
    document.body.appendChild(container);
    const root = createRoot(container);
    await act(async () => {
      root.render(<App workspace={workspace} provider={provider} initialSettings={TEST_SETTINGS} />);
    });
    await settle();
    await act(async () => {
      await workspace.open('note.md');
    });
    await settle();

    const button = [...container.querySelectorAll('button')].find((item) => item.textContent?.trim() === '导出 HTML')!;
    await act(async () => {
      button.click();
    });
    await settle();

    const html = provider.writes[0]!.text;
    expect(html).toContain('missing.png');
    expect(html).toContain('data-qm-missing');
    expect(provider.notices.join(' ')).toContain('读不到');

    await act(async () => {
      root.unmount();
    });
    container.remove();
  });
});

describe('源码模式切换', () => {
  /**
   * 这一组测的是**两个内核之间交棒**，而不是某一个内核。
   *
   * 切换视图的实现在外壳看来只有一句话：换掉 `EditorComponent`。
   * 真正容易出事的地方全在接缝上 —— 换过去之后新内核有没有拿到当前内容、
   * 旧内核被销毁后外壳会不会还往上写、这一下算不算「用户改了内容」。
   * 所以断言全部围绕这三件事。
   *
   * 注：用例里用 `store.updateText()` 代替「在编辑器里敲字」。这不是偷懒 ——
   * 内核的 `onChange` 回调里做的**就是**这一件事，两者是同一条路径。
   * 真键盘输入那条路径由「脏标记」那一组覆盖。
   */

  const SOURCE = '# 标题\n\n正文 **加粗**。\n';

  it('Markdown 文档默认在富文本视图，工具栏给出「源码」开关', async () => {
    mounted = await mount((p) => p.put('note.md', SOURCE));
    await mounted.open('note.md');

    expect(mounted.container.querySelector('.qm-editor .ProseMirror')).not.toBeNull();
    const button = toolbarButton(mounted.container, '源码');
    expect(button).not.toBeNull();
    expect(button?.disabled).toBe(false);
  });

  it('切到源码视图：换成代码内核、内容一字不差、且不算脏', async () => {
    mounted = await mount((p) => p.put('note.md', SOURCE));
    await mounted.open('note.md');

    await mounted.click('源码');

    // 富文本视图整个消失，换成 CodeMirror —— 是**替换**不是叠加
    expect(mounted.container.querySelector('.ProseMirror')).toBeNull();
    expect(mounted.container.querySelector('.qm-code .cm-content')).not.toBeNull();
    // 源码视图看到的必须是**原始 Markdown**（连 `**` 这类语法标记一起），
    // 而不是被富文本内核规范化之后的写法
    expect(renderedText(mounted.container)).toContain('**加粗**');
    // 状态栏要说清「这是谁的源码」，否则一份 .md 显示「代码 / 文本」很困惑
    expect(kernelOf(mounted.container)).toBe('代码 / 文本 · Markdown 源码');
    // 切视图是纯视图动作：内容没变，也不该变成待保存
    expect(mounted.store.getSnapshot().doc?.text).toBe(SOURCE);
    expect(statusOf(mounted.container)).toBe('已保存');
    // 开关要翻面，否则用户不知道再点一次会切回去
    expect(toolbarButton(mounted.container, '富文本')).not.toBeNull();
    expect(toolbarButton(mounted.container, '源码')).toBeNull();
  });

  it('切回富文本视图，内容仍然一字不差', async () => {
    mounted = await mount((p) => p.put('note.md', SOURCE));
    await mounted.open('note.md');

    await mounted.click('源码');
    await mounted.click('富文本');

    expect(mounted.container.querySelector('.qm-code')).toBeNull();
    const prose = mounted.container.querySelector('.qm-editor .ProseMirror');
    expect(prose).not.toBeNull();
    expect(prose?.textContent).toContain('标题');
    expect(prose?.textContent).toContain('加粗');
    expect(mounted.store.getSnapshot().doc?.text).toBe(SOURCE);
    expect(statusOf(mounted.container)).toBe('已保存');
  });

  it('切换后外壳写的是**新**内核，不是那个已经销毁的旧实例', async () => {
    mounted = await mount((p) => p.put('note.md', SOURCE));
    await mounted.open('note.md');
    await mounted.click('源码');

    // 走「外部改动」那条路径把内容推给当前内核。
    // 若句柄标识没把视图模式算进去，这一步会被推到一个已销毁的富文本内核上，
    // 而源码视图里什么都看不到。
    await act(async () => {
      mounted?.store.updateText('# 换过的标题\n');
    });
    await settle();

    expect(renderedText(mounted.container)).toContain('换过的标题');
    // 推过去之后基准要同步，否则下一次外部改动会被当成「已经推过了」而漏掉
    expect(statusOf(mounted.container)).toBe('有未保存的改动');
  });

  it('在源码视图里改的内容会带到富文本视图（两种视图共用一份权威副本）', async () => {
    mounted = await mount((p) => p.put('note.md', SOURCE));
    await mounted.open('note.md');

    await mounted.click('源码');
    await act(async () => {
      mounted?.store.updateText('## 换过的标题\n\n新的一段。\n');
    });
    await settle();
    await mounted.click('富文本');

    const prose = mounted.container.querySelector('.qm-editor .ProseMirror');
    expect(prose?.textContent).toContain('换过的标题');
    expect(prose?.textContent).toContain('新的一段。');
    expect(statusOf(mounted.container)).toBe('有未保存的改动');
  });

  it('代码文档没有源码视图 —— 开关禁用而不是隐藏', async () => {
    mounted = await mount((p) => p.put('main.py', 'print(1)\n'));
    await mounted.open('main.py');

    const button = toolbarButton(mounted.container, '源码');
    // 隐藏会让用户以为功能不存在；禁用则明确告诉他「有，但这份文档用不上」
    expect(button).not.toBeNull();
    expect(button?.disabled).toBe(true);
    expect(kernelOf(mounted.container)).toBe('代码 / 文本');
  });

  it('front matter 的 .md 已经就是源码形态，所以没有源码视图可切', async () => {
    mounted = await mount((p) => p.put('note.md', '---\ntitle: 笔记\n---\n\n# 正文\n'));
    await mounted.open('note.md');

    expect(toolbarButton(mounted.container, '源码')?.disabled).toBe(true);
  });

  it('切标签页时视图模式跟着各自的文档走，不会串台', async () => {
    mounted = await mount((p) => {
      p.put('a.md', '# 甲\n');
      p.put('b.md', '# 乙\n');
    });

    await mounted.open('a.md');
    await mounted.click('源码');

    await mounted.open('b.md');
    // b 是新打开的文档，默认富文本 —— 不该继承 a 的源码模式
    expect(mounted.container.querySelector('.ProseMirror')).not.toBeNull();
    expect(toolbarButton(mounted.container, '源码')?.disabled).toBe(false);

    const first = mounted.workspace.getSnapshot().tabs[0];
    expect(first).toBeDefined();
    await act(async () => {
      mounted?.workspace.activate(first!.id);
    });
    await settle();

    // 切回 a：它仍然停在源码视图
    expect(mounted.container.querySelector('.qm-code .cm-content')).not.toBeNull();
    expect(toolbarButton(mounted.container, '富文本')).not.toBeNull();
  });
});

describe('状态栏统计', () => {
  /**
   * 这里守的是**接线**，不是算法 —— 算法在 `core/statistics.test.ts` 里按口径逐条钉过。
   * 分开测是因为两者的失败原因完全不同：算法错是「数错了」，
   * 接线错是「压根没显示」或者「显示的是别的文档的数」。
   */

  it('打开文档后显示行数 / 字数 / 字符数 / 阅读时长', async () => {
    mounted = await mount((p) => p.put('note.md', '# 标题\n\n正文\n'));
    await mounted.open('note.md');

    expect(statisticsOf(mounted.container)).toBe('4 行 · 4 字 · 5 字符 · 约 1 分钟');
  });

  it('空文档不显示「约 0 分钟」', async () => {
    mounted = await mount((p) => p.put('empty.md', ''));
    await mounted.open('empty.md');

    expect(statisticsOf(mounted.container)).toBe('1 行 · 0 字 · 0 字符');
  });

  it('统计跟着活动文档走 —— 它不是全局缓存的', async () => {
    mounted = await mount((p) => {
      p.put('a.md', '短');
      p.put('b.md', '短\n长\n');
    });

    await mounted.open('a.md');
    expect(statisticsOf(mounted.container)).toBe('1 行 · 1 字 · 1 字符 · 约 1 分钟');

    await mounted.open('b.md');
    expect(statisticsOf(mounted.container)).toBe('3 行 · 2 字 · 2 字符 · 约 1 分钟');

    // 切回第一个标签，统计必须跟着回到第一份文档
    const first = mounted.workspace.getSnapshot().tabs[0];
    expect(first).toBeDefined();
    await act(async () => {
      mounted?.workspace.activate(first!.id);
    });
    await settle();
    expect(statisticsOf(mounted.container)).toBe('1 行 · 1 字 · 1 字符 · 约 1 分钟');
  });
});

describe('选中浮层工具条', () => {
  /**
   * 这一组打的是**三件事的接缝**，每一件单独测都测不出问题：
   *
   * 1. 内核能不能把「选了什么格式、选区在屏幕哪儿」报上来；
   * 2. 外壳能不能据此画出一个位置正确、状态正确的浮层；
   * 3. 点浮层上的按钮，能不能真的改到文档、并且**让外壳知道文档改了**。
   *
   * 第 3 条里最后半句是重点：点按钮不产生任何键盘事件，如果格式化不打
   * `USER_EDIT_META`，文档不会变脏，用户按 Ctrl+S 写回磁盘的是格式化**之前**的内容。
   * 这是「静默丢数据」，所以这里专门断言状态栏变成了「有未保存的改动」。
   */

  it('没有选中文字时没有浮层', async () => {
    mounted = await mount((p) => p.put('note.md', '# 标题\n\n正文 abc 结尾\n'));
    await mounted.open('note.md');

    // 刚打开时没有选区
    expect(toolbarOf(mounted.container)).toBeNull();

    // 选中再取消，浮层要跟着收起来
    await selectText(mounted.container, 'abc');
    expect(toolbarOf(mounted.container)).not.toBeNull();
    await clearSelection(mounted.container, 'abc');
    expect(toolbarOf(mounted.container)).toBeNull();
  });

  it('选中文字后浮层出现，并列出全部格式按钮', async () => {
    mounted = await mount((p) => p.put('note.md', '正文 abc 结尾\n'));
    await mounted.open('note.md');
    await selectText(mounted.container, 'abc');

    const toolbar = toolbarOf(mounted.container);
    expect(toolbar).not.toBeNull();
    // 浮层是编辑器区域的**兄弟**节点，不是子节点 ——
    // 放进内核挂载的那层容器会被内核在挂载时抹掉
    expect(toolbar?.parentElement?.classList.contains('editor-area')).toBe(true);

    for (const label of ['加粗（Ctrl+B）', '斜体（Ctrl+I）', '删除线', '行内代码', '链接', '一级标题', '代码块']) {
      expect(formatButton(mounted.container, label), label).not.toBeNull();
    }
  });

  it('浮层按钮拦掉 mousedown —— 不拦的话焦点离开编辑器，选区丢失，按钮点不动', async () => {
    mounted = await mount((p) => p.put('note.md', '正文 abc 结尾\n'));
    await mounted.open('note.md');
    await selectText(mounted.container, 'abc');

    const button = formatButton(mounted.container, '加粗（Ctrl+B）');
    expect(button).not.toBeNull();
    const event = new MouseEvent('mousedown', { bubbles: true, cancelable: true });
    button?.dispatchEvent(event);

    expect(event.defaultPrevented).toBe(true);
  });

  it('点「加粗」：文档真的变了、变脏了，按钮也亮起来了', async () => {
    mounted = await mount((p) => p.put('note.md', '正文 abc 结尾\n'));
    await mounted.open('note.md');

    // 未加粗时按钮不该是生效态
    await selectText(mounted.container, 'abc');
    expect(formatButton(mounted.container, '加粗（Ctrl+B）')?.getAttribute('aria-pressed')).toBe('false');

    await clickFormat(mounted.container, '加粗（Ctrl+B）');

    expect(mounted.store.getSnapshot().doc?.text).toContain('**abc**');
    // 这一条才是「不会静默丢数据」的保证
    expect(statusOf(mounted.container)).toBe('有未保存的改动');
    // 点完之后按钮要变成生效态。文档变了但选区没动 ——
    // 内核如果只在选区变化时上报，这里会停在旧的高亮上
    expect(formatButton(mounted.container, '加粗（Ctrl+B）')?.getAttribute('aria-pressed')).toBe('true');
  });

  it('选区里本来就有加粗时，按钮一出现就是生效态', async () => {
    mounted = await mount((p) => p.put('note.md', '正文 **abc** 结尾\n'));
    await mounted.open('note.md');
    await selectText(mounted.container, 'abc');

    expect(formatButton(mounted.container, '加粗（Ctrl+B）')?.getAttribute('aria-pressed')).toBe('true');
    expect(formatButton(mounted.container, '斜体（Ctrl+I）')?.getAttribute('aria-pressed')).toBe('false');
  });

  it('点「二级标题」把整个块变成标题', async () => {
    mounted = await mount((p) => p.put('note.md', '正文 abc 结尾\n'));
    await mounted.open('note.md');
    await selectText(mounted.container, 'abc');

    await clickFormat(mounted.container, '二级标题');

    expect(mounted.store.getSnapshot().doc?.text).toContain('## 正文 abc 结尾');
    expect(formatButton(mounted.container, '二级标题')?.getAttribute('aria-pressed')).toBe('true');
  });

  it('点「链接」弹出地址输入，回车把链接写进文档', async () => {
    mounted = await mount((p) => p.put('note.md', '正文 abc 结尾\n'));
    await mounted.open('note.md');
    await selectText(mounted.container, 'abc');

    await clickFormat(mounted.container, '链接');

    const input = mounted.container.querySelector<HTMLInputElement>('.st-link-input');
    expect(input).not.toBeNull();

    await act(async () => {
      if (input === null) return;
      // React 受控输入：直接改 `input.value` 不会触发 onChange ——
      // React 在元素上挂了一个 value tracker，看到值没变就把事件吞掉。
      // 必须走原型上的原生 setter 绕过它。
      // `.bind(input)` 是顺手做的：`.set` 是方法，拆下来单传会丢 `this`
      const setNativeValue = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set?.bind(input);
      setNativeValue?.('https://example.com');
      input.dispatchEvent(new Event('input', { bubbles: true }));
      input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
    });
    await nextFrame();

    expect(mounted.store.getSnapshot().doc?.text).toContain('[abc](https://example.com)');
  });

  /**
   * 代码 / 纯文本内核没有格式化能力。
   *
   * 这里要的是「**不出现**」而不是「出现但禁用」—— 与工具栏上那个「源码」按钮相反。
   * 区别在于：源码视图是「有，但这份文档用不上」，格式化是「这个内核根本没有」。
   * 一个纯文本编辑器弹出加粗按钮，用户点下去只会什么都没发生。
   */
  it('代码内核的文档不显示浮层', async () => {
    mounted = await mount((p) => p.put('main.py', 'print("abc")\n'));
    await mounted.open('main.py');

    expect(kernelOf(mounted.container)).toBe('代码 / 文本');
    await selectText(mounted.container, 'abc');
    expect(toolbarOf(mounted.container)).toBeNull();
  });

  it('切到源码模式后浮层消失，切回富文本又能用', async () => {
    mounted = await mount((p) => p.put('note.md', '正文 abc 结尾\n'));
    await mounted.open('note.md');
    await selectText(mounted.container, 'abc');
    expect(toolbarOf(mounted.container)).not.toBeNull();

    await mounted.click('源码');
    // 源码视图底下坐着代码内核，它没有格式化能力
    expect(toolbarOf(mounted.container)).toBeNull();

    await mounted.click('富文本');
    await selectText(mounted.container, 'abc');
    expect(toolbarOf(mounted.container)).not.toBeNull();
  });
});
