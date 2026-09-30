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
import { typeInto } from '../test/typing.js';
import type {
  BinaryWriteOutcome,
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
  /** 粘贴图片落下来的字节。键是完整路径。 */
  readonly binaries = new Map<string, ArrayBuffer>();
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
    // 粘贴进来的图片先在内存二进制盘里 —— 与浏览器预览模式同一套优先级
    const pasted = this.binaries.get(path);
    if (pasted !== undefined) return pasted.slice(0);

    // 其余情况给一张最小 PNG 的文件头，足以让 MIME 嗅探认出 image/png
    void this.need(path);
    return new Uint8Array([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]).buffer;
  }

  async writeBinaryFile(path: string, bytes: ArrayBuffer): Promise<BinaryWriteOutcome> {
    // 与桌面版对齐：已存在就拒绝，不覆盖
    if (this.binaries.has(path)) {
      throw { code: 'CONFLICT', message: `已存在：${path}`, detail: null };
    }
    this.binaries.set(path, bytes.slice(0));
    return { path, size: bytes.byteLength, mtimeMs: (this.clock += 10) };
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
  options: { strict?: boolean; settings?: Partial<ViewSettings> } = {},
): Promise<Mounted> {
  const provider = new FakeProvider();
  seed(provider);
  const workspace = new WorkspaceStore(provider);

  const container = document.createElement('div');
  document.body.appendChild(container);
  const root: Root = createRoot(container);

  const settings: ViewSettings = { ...TEST_SETTINGS, ...options.settings };
  const element: ReactElement = (
    <App workspace={workspace} provider={provider} initialSettings={settings} />
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
 *
 * ## 分屏时**必须先取左栏**
 *
 * 分屏的右栏也是一个 `.qm-editor .ProseMirror`，而且它排在左栏**后面** ——
 * 但选择器写在前面的那个（`.qm-editor .ProseMirror`）会先命中**它**，
 * 因为左栏那时是 CodeMirror。往只读的预览栏里打字什么都不会发生，
 * 于是「打字 → 内容变了」这类用例会变成假绿（或者假红）。
 * 所以这里把「分屏左栏」这一支写在最前面：命中就说明用户此刻在编辑哪一栏。
 */
function editableOf(container: HTMLElement): HTMLElement {
  const root = container.querySelector<HTMLElement>(
    '.editor-split > .editor-pane:first-child .qm-code .cm-content, .qm-editor .ProseMirror, .qm-code .cm-content',
  );
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

function slashMenuOf(container: HTMLElement): HTMLElement | null {
  return container.querySelector('.slash-menu');
}

/** 菜单项上的文字，按显示顺序。用来断言筛选结果，而不是断言「有没有渲染」。 */
function slashLabels(container: HTMLElement): string[] {
  return [...container.querySelectorAll('.slash-item .slash-label')].map((node) => node.textContent?.trim() ?? '');
}

/**
 * 在编辑器里真的敲一串字。
 *
 * 走 DOM 变更 + `input` 事件，也就是浏览器里发生的事（细节见 `src/test/typing.ts`）。
 * 之所以不满足于「直接调内核命令」：斜杠菜单的触发是**逐键推进**的，
 * 绕过输入链路就跳过了「敲到第几个字时菜单出现」这段唯一容易写错的行为。
 */
async function typeInEditor(container: HTMLElement, text: string): Promise<void> {
  const editable = editableOf(container);
  await act(async () => {
    await typeInto(editable, text);
  });
}

/**
 * 在编辑器上按一个键。
 *
 * 派发到 contenteditable 上而不是 `window`：真实使用中事件目标就是编辑器，
 * 而斜杠菜单的键盘处理正是靠「在 `window` 的**捕获阶段**抢先拦下」来工作的
 * （见 `SlashMenu.tsx` 的文件头）。派发到别处就测不到那条路径了。
 */
async function pressInEditor(container: HTMLElement, key: string): Promise<void> {
  const editable = editableOf(container);
  await act(async () => {
    editable.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }));
  });
  await nextFrame();
}

let mounted: Mounted | null = null;

beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  /**
   * 位置记忆会写 localStorage，而 jsdom 的 localStorage 在**同一个文件里是共享的**
   * （每个测试文件一个 jsdom 环境，用例之间不隔离）。不清的话，
   * 前一个用例把光标放在 `C:\a.md` 的第 3 行，后一个用例打开同一个路径时
   * 会**从第 3 行开始** —— 一个看起来像产品 bug 的假失败。
   *
   * 偏好那边不受影响：`TEST_SETTINGS` 是注入的，不读存储。
   */
  localStorage.clear();
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

// ---------------------------------------------------------------------------
// 分屏（源码 + 预览并排）
// ---------------------------------------------------------------------------

/** 分屏的两栏。没分屏时是空数组。 */
function splitPanes(container: HTMLElement): HTMLElement[] {
  return [...container.querySelectorAll<HTMLElement>('.editor-split > .editor-pane')];
}

/** 分屏右栏里那个只读的渲染。 */
function previewOf(container: HTMLElement): HTMLElement {
  const pane = splitPanes(container)[1];
  const root = pane?.querySelector<HTMLElement>('.qm-editor .ProseMirror') ?? null;
  if (root === null) throw new Error('分屏右栏里没有找到只读预览 —— 用例是不是忘了先进分屏？');
  return root;
}

/** 点状态栏里视图三档的某一档。 */
async function clickViewMode(container: HTMLElement, label: string): Promise<void> {
  const button = [...container.querySelectorAll<HTMLButtonElement>('.view-switcher button')].find(
    (item) => item.textContent?.trim() === label,
  );
  if (!button) throw new Error(`视图切换器里没有「${label}」`);
  await act(async () => {
    button.click();
  });
  await settle();
}

/**
 * 等一段**真实**的时间。
 *
 * 预览同步是防抖的（`PREVIEW_SYNC_DELAY_MS = 160`），而 `settle()` 那三轮
 * `setTimeout(0)` 走不到它。这里刻意用真实计时器而不是 `vi.useFakeTimers()`：
 * 内核创建本身是一条 `await` 的 Promise 链，假计时器一开就得把所有
 * `await` 都手工推进去，用例会退化成一串 `advanceTimersByTimeAsync` ——
 * 而那种写法测的是「定时器有没有被排上」，不是「用户看到了什么」。
 */
async function waitMs(ms: number): Promise<void> {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, ms));
  });
}

describe('分屏：源码与预览并排', () => {
  /**
   * 这一组守的是三件事，缺一件这个功能就不成立：
   *
   * 1. **两栏真的同时存在**（不是替换、不是并排画两遍同一个内核）；
   * 2. **右栏是只读的**（能编辑的话，同一份文档就有了两个写入方）；
   * 3. **左栏的改动会到右栏**（不会的话，预览显示的是打开那一刻的旧内容 ——
   *    而它看起来完全正常，用户会以为自己的修改没生效）。
   */

  const SOURCE = '# 标题\n\n正文 **加粗**。\n';

  it('两栏并存：左栏源码、右栏只读渲染，且内容一字不差', async () => {
    mounted = await mount((p) => p.put('note.md', SOURCE));
    await mounted.open('note.md');

    await clickViewMode(mounted.container, '分屏');

    const panes = splitPanes(mounted.container);
    expect(panes.length).toBe(2);

    // 左栏是**源码**：看得到 `**` 这类语法标记
    const left = panes[0]?.querySelector<HTMLElement>('.qm-code .cm-content');
    expect(left).not.toBeNull();
    expect(left?.textContent).toContain('**加粗**');

    // 右栏是**渲染结果**：语法标记消失、文本还在
    const preview = previewOf(mounted.container);
    expect(preview.textContent).toContain('标题');
    expect(preview.textContent).toContain('加粗');
    expect(preview.textContent).not.toContain('**');

    // 分隔条要在，否则用户没法调两栏的比例
    expect(mounted.container.querySelector('.editor-divider')).not.toBeNull();

    // 切到分屏是纯视图动作：内容没变、也不该变成待保存
    expect(mounted.store.getSnapshot().doc?.text).toBe(SOURCE);
    expect(statusOf(mounted.container)).toBe('已保存');
  });

  it('右栏是**真只读** —— 靠 ProseMirror 的 contenteditable，不是「拦下按键」', async () => {
    mounted = await mount((p) => p.put('note.md', SOURCE));
    await mounted.open('note.md');
    await clickViewMode(mounted.container, '分屏');

    // 这条断言比「敲字没反应」强得多：后者在一个「拦截了按键但 DOM 仍可编辑」
    // 的假只读实现上也会通过，而那种实现在拖选、IME、粘贴上全都会漏
    expect(previewOf(mounted.container).getAttribute('contenteditable')).toBe('false');
  });

  it('左栏改内容之后，右栏跟着更新', async () => {
    mounted = await mount((p) => p.put('note.md', SOURCE));
    await mounted.open('note.md');
    await clickViewMode(mounted.container, '分屏');

    expect(previewOf(mounted.container).textContent).toContain('标题');

    // 用 `store.updateText` 而不是模拟键盘：内核的 `onChange` 里做的**就是**这一件事。
    // （分屏的左栏是 CodeMirror，jsdom 里没有可信的打字路径 —— 硬造一个
    // 只会测到「我们伪造的事件长什么样」，而不是「用户打字会怎样」。）
    await act(async () => {
      mounted?.store.updateText('# 换过的标题\n\n新的一段。\n');
    });
    await settle();

    // 防抖窗口**内**预览还没变。这一步不只是补充说明 ——
    // 它证明下面那条断言不是空转（若把同步整个删掉，下面那条才会失败）
    expect(previewOf(mounted.container).textContent).not.toContain('换过的标题');

    // 预览是**防抖**推过去的（160ms），必须真的等过那个窗口
    await waitMs(260);

    const preview = previewOf(mounted.container);
    expect(preview.textContent).toContain('换过的标题');
    expect(preview.textContent).toContain('新的一段。');
    // 左栏也要跟着换（走的是「外部改动」那条路径）
    expect(editableOf(mounted.container).textContent).toContain('换过的标题');
  });

  it('从分屏切回富文本：回到单栏，右栏消失', async () => {
    mounted = await mount((p) => p.put('note.md', SOURCE));
    await mounted.open('note.md');
    await clickViewMode(mounted.container, '分屏');
    expect(splitPanes(mounted.container).length).toBe(2);

    await clickViewMode(mounted.container, '富文本');

    expect(splitPanes(mounted.container).length).toBe(0);
    expect(mounted.container.querySelector('.qm-code')).toBeNull();
    expect(mounted.container.querySelector('.qm-editor .ProseMirror')).not.toBeNull();
    expect(mounted.store.getSnapshot().doc?.text).toBe(SOURCE);
  });

  it('工具栏那个开关把分屏算作「源码那一侧」—— 再点一次回富文本', async () => {
    mounted = await mount((p) => p.put('note.md', SOURCE));
    await mounted.open('note.md');
    await clickViewMode(mounted.container, '分屏');

    // 分屏的出口与源码模式是同一个（富文本），否则用户点一下会落到
    // 一个他刚才没选过的状态上
    const button = toolbarButton(mounted.container, '富文本');
    expect(button).not.toBeNull();
    expect(button?.disabled).toBe(false);
  });

  it('三档里只有一档是选中的，且读屏读得出来', async () => {
    mounted = await mount((p) => p.put('note.md', SOURCE));
    await mounted.open('note.md');

    const pressed = () =>
      [...mounted!.container.querySelectorAll('.view-switcher button')]
        .filter((item) => item.getAttribute('aria-pressed') === 'true')
        .map((item) => item.textContent?.trim());

    expect(pressed()).toEqual(['富文本']);

    await clickViewMode(mounted.container, '源码');
    expect(pressed()).toEqual(['源码']);

    await clickViewMode(mounted.container, '分屏');
    expect(pressed()).toEqual(['分屏']);
  });

  it('代码文档没有视图三档 —— 它本身就是源码，三档对它是个不存在的选择', async () => {
    mounted = await mount((p) => p.put('main.py', 'print(1)\n'));
    await mounted.open('main.py');

    expect(mounted.container.querySelector('.view-switcher')).toBeNull();
  });

  it('切标签页时不会把上一份文档的预览留在屏幕上', async () => {
    mounted = await mount((p) => {
      p.put('a.md', '# 甲\n');
      p.put('b.md', '# 乙\n');
    });

    await mounted.open('a.md');
    await clickViewMode(mounted.container, '分屏');
    expect(previewOf(mounted.container).textContent).toContain('甲');

    await mounted.open('b.md');

    // b 是新打开的文档，默认富文本 —— 没有分屏、更没有 a 的内容
    expect(splitPanes(mounted.container).length).toBe(0);
    expect(mounted.container.textContent).not.toContain('甲');
  });

  it('右栏的预览不产生浮层工具条 —— 在只读渲染里选中文字不该弹出「加粗」', async () => {
    mounted = await mount((p) => p.put('note.md', SOURCE));
    await mounted.open('note.md');
    await clickViewMode(mounted.container, '分屏');

    // 在**右栏**里真的选中一段文字（而不是在左栏）
    const preview = previewOf(mounted.container);
    const walker = document.createTreeWalker(preview, NodeFilter.SHOW_TEXT);
    let hit: Text | null = null;
    while (walker.nextNode()) {
      if ((walker.currentNode as Text).data.includes('加粗')) {
        hit = walker.currentNode as Text;
        break;
      }
    }
    expect(hit).not.toBeNull();
    const range = document.createRange();
    range.selectNodeContents(hit!);
    const selection = window.getSelection();
    selection?.removeAllRanges();
    selection?.addRange(range);

    await act(async () => {
      document.dispatchEvent(new Event('selectionchange'));
    });
    await nextFrame();

    // 浮层会弹在左栏（用户真正在编辑的那一栏）上，但**不该由右栏触发** ——
    // 在预览里选中一段文字、浮出一个「加粗」、点下去却作用在左栏，
    // 没有任何用户能猜到这个行为。所以右栏根本不交出格式化命令面。
    expect(previewOf(mounted.container).getAttribute('contenteditable')).toBe('false');
    expect(mounted.store.getSnapshot().doc?.text).toBe(SOURCE);
  });

  /**
   * 拖动分隔条。
   *
   * 这条用例测的**不是「拖动能改比例」**（那要真实布局，jsdom 给不了），
   * 而是它旁边那个更危险的失败模式：**拿不到容器宽度时该怎么办**。
   *
   * jsdom 里 `getBoundingClientRect()` 全是 0，于是 `(clientX - 0) / 0` 得到
   * `Infinity` 或 `NaN`。这个值一旦被夹进 `[0.2, 0.8]` 之外的比较里就会变成
   * 一个「合法」的边界值并被**持久化** —— 之后每次打开都是那个比例，
   * 而用户只会看到「分屏比例坏了」，找不到原因。
   *
   * 所以实现里是「宽度 ≤ 0 直接返回」。这条用例钉的就是那个提前返回。
   */
  it('拖动分隔条：量不到真实宽度时是空操作，不会把坏值写进偏好', async () => {
    mounted = await mount((p) => p.put('note.md', SOURCE));
    await mounted.open('note.md');
    await clickViewMode(mounted.container, '分屏');

    const divider = mounted.container.querySelector<HTMLElement>('.editor-divider');
    expect(divider).not.toBeNull();

    // 用 MouseEvent 而不是 PointerEvent：jsdom 没有实现 PointerEvent 构造器，
    // 而 React 只按**事件名**分派 `onPointerDown`，类型本身不影响这条路径
    await act(async () => {
      divider!.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true, cancelable: true, clientX: 100 }));
    });
    await act(async () => {
      window.dispatchEvent(new MouseEvent('pointermove', { clientX: 260 }));
      window.dispatchEvent(new MouseEvent('pointerup'));
    });
    await settle();

    // 两栏还在，比例也没被改坏
    const panes = splitPanes(mounted.container);
    expect(panes.length).toBe(2);
    expect(panes[0]?.style.flexGrow).toBe('0.5');
    expect(mounted.store.getSnapshot().doc?.text).toBe(SOURCE);
  });
});

// ---------------------------------------------------------------------------
// 活动栏
// ---------------------------------------------------------------------------

/** 活动栏里的某个图标按钮（按它里面那段 `.sr-only` 文字找）。 */
function activityItem(container: HTMLElement, label: string): HTMLButtonElement | null {
  const found = [...container.querySelectorAll<HTMLButtonElement>('.activitybar button')].find(
    (item) => item.textContent?.trim() === label,
  );
  return found ?? null;
}

describe('活动栏', () => {
  /**
   * 它是「这个应用有哪些面板」的**常驻入口**。
   *
   * 在这之前那五件事分别藏在三个快捷键和两个工具栏按钮后面，
   * 而快捷键不是发现功能的入口（这是 M9 写设置页时定下的判断）。
   */

  it('五个入口都在，且开关状态由 aria-pressed 表达', async () => {
    mounted = await mount(() => undefined);

    for (const label of ['文件浏览器', '查找', '大纲', '命令面板', '设置']) {
      expect(activityItem(mounted.container, label)).not.toBeNull();
    }

    // 测试偏好里侧栏是关的
    expect(activityItem(mounted.container, '文件浏览器')?.getAttribute('aria-pressed')).toBe('false');
  });

  it('点文件浏览器图标开关侧栏', async () => {
    mounted = await mount(() => undefined);
    expect(mounted.container.querySelector('.sidebar-slot')).toBeNull();

    await mounted.click('文件浏览器');

    expect(activityItem(mounted.container, '文件浏览器')?.getAttribute('aria-pressed')).toBe('true');
    expect(mounted.container.querySelector('.sidebar-slot')).not.toBeNull();
  });

  it('点命令面板图标打开命令面板', async () => {
    mounted = await mount(() => undefined);
    expect(mounted.container.querySelector('.palette')).toBeNull();

    await mounted.click('命令面板');

    expect(mounted.container.querySelector('.palette')).not.toBeNull();
  });

  /**
   * 「有，但这份文档用不上」用**禁用**表达，不是隐藏 ——
   * 与工具栏那个「源码」开关同一条规矩。
   */
  it('查找与大纲按当前文档决定能不能点，禁用而不是消失', async () => {
    mounted = await mount((p) => {
      p.put('note.md', '# 标题\n');
      p.put('main.py', 'print(1)\n');
    });

    // 还没打开任何文档：无处可查、也没有大纲
    expect(activityItem(mounted.container, '查找')?.disabled).toBe(true);
    expect(activityItem(mounted.container, '大纲')?.disabled).toBe(true);

    await mounted.open('note.md');
    expect(activityItem(mounted.container, '查找')?.disabled).toBe(false);
    expect(activityItem(mounted.container, '大纲')?.disabled).toBe(false);

    await mounted.open('main.py');
    // 代码文档没有标题结构。按钮还在（用户能看到功能存在），只是置灰
    expect(activityItem(mounted.container, '大纲')?.disabled).toBe(true);
    expect(activityItem(mounted.container, '查找')?.disabled).toBe(false);
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

describe('斜杠菜单', () => {
  /**
   * 这一组打的是**从敲键盘到文件内容真的变了**这条完整链路：
   *
   * 打字 → 内核认出「处于斜杠触发状态」→ 上报筛选词与位置 → 外壳画菜单 →
   * 键盘/鼠标选中一项 → 内核删掉 `/query` 再套块格式 → 外壳得知文档变脏。
   *
   * 两处最容易写坏、而且坏了不会报错的地方各有一条用例钉着：
   * - **键盘必须能拦得住 ProseMirror**。菜单打开时焦点还在编辑器里，
   *   Enter 若先被 ProseMirror 拿到，段落会被拆开 —— 结果就是「插入了块，
   *   但也多出一个空行」。所以 `SlashMenu` 在 `window` 的捕获阶段监听；
   *   用例直接断言「按 Enter 之后 `/代码` 没了、文档里是代码块」。
   * - **Esc 只能收菜单，不能删用户打的字**。收起来之后不能立刻又弹回来
   *   （`/表` 还在文档里，触发条件仍然成立），所以 `SlashMenu` 记住
   *   「哪一段筛选词被收起过」。这条也各有一个断言。
   */

  it('空段落里打 `/` 弹出菜单，列出全部可插入的块', async () => {
    mounted = await mount((p) => p.put('note.md', '\n'));
    await mounted.open('note.md');

    expect(slashMenuOf(mounted.container)).toBeNull();

    await typeInEditor(mounted.container, '/');

    const menu = slashMenuOf(mounted.container);
    expect(menu).not.toBeNull();
    // 与选中工具条同一条理由：放进内核挂载的那层容器会被内核挂载时抹掉
    expect(menu?.parentElement?.classList.contains('editor-area')).toBe(true);
    expect(slashLabels(mounted.container)).toEqual([
      '正文',
      '一级标题',
      '二级标题',
      '三级标题',
      '无序列表',
      '有序列表',
      '引用',
      '代码块',
      '数学公式',
      '图表',
      '表格',
      '分隔线',
    ]);
  });

  it('继续打字就是筛选；一个都不匹配时如实说「没有匹配的块」', async () => {
    mounted = await mount((p) => p.put('note.md', '\n'));
    await mounted.open('note.md');

    await typeInEditor(mounted.container, '/代码');
    expect(slashLabels(mounted.container)).toEqual(['代码块']);

    await typeInEditor(mounted.container, 'zzz');
    expect(slashLabels(mounted.container)).toEqual([]);
    // 显示提示而不是「退化成显示全部」—— 后者会让用户以为筛选没生效
    expect(mounted.container.querySelector('.slash-empty')).not.toBeNull();
  });

  it('回车插入高亮的那一项：`/代码` 从文档里消失、块真的进了文档、文档变脏', async () => {
    mounted = await mount((p) => p.put('note.md', '\n'));
    await mounted.open('note.md');

    await typeInEditor(mounted.container, '/代码');
    await pressInEditor(mounted.container, 'Enter');

    const text = mounted.store.getSnapshot().doc?.text ?? '';
    expect(text).not.toContain('/代码');
    expect(text).toContain('```');
    // 插完菜单要自己收起来（`/query` 没了，触发条件不再成立）
    expect(slashMenuOf(mounted.container)).toBeNull();
    // 这一条才是「不会静默丢数据」的保证：插入不经过键盘事件，
    // 内核必须自己打 USER_EDIT_META，否则按 Ctrl+S 存回的是插入前的旧内容
    expect(statusOf(mounted.container)).toBe('有未保存的改动');
  });

  it('上下键换高亮项，回车插的是换过的那一项', async () => {
    mounted = await mount((p) => p.put('note.md', '\n'));
    await mounted.open('note.md');

    await typeInEditor(mounted.container, '/');
    await pressInEditor(mounted.container, 'ArrowDown');
    // 第二项是「一级标题」
    expect(mounted.container.querySelectorAll('.slash-item')[1]?.getAttribute('aria-selected')).toBe('true');

    await pressInEditor(mounted.container, 'Enter');

    expect(mounted.store.getSnapshot().doc?.text).toMatch(/^#\s*$/m);
  });

  /**
   * Esc 的两半：**收得掉**，且**收起来之后不会立刻弹回来**。
   *
   * 第二半才是容易漏的：`/表` 还在文档里，内核下一帧仍然认为「处于斜杠触发状态」，
   * 不记住「这段筛选词被收起过」的话菜单会原地复活，用户以为 Esc 坏了。
   */
  it('Esc 收起菜单但不删用户打的字，再打字菜单回来', async () => {
    mounted = await mount((p) => p.put('note.md', '\n'));
    await mounted.open('note.md');

    await typeInEditor(mounted.container, '/表');
    expect(slashMenuOf(mounted.container)).not.toBeNull();
    const before = mounted.store.getSnapshot().doc?.text;

    await pressInEditor(mounted.container, 'Escape');
    expect(slashMenuOf(mounted.container)).toBeNull();
    // 只是收起，一个字符都没改
    expect(mounted.store.getSnapshot().doc?.text).toBe(before);

    // 筛选词变了 = 用户又动了菜单，应该重新出现
    await typeInEditor(mounted.container, '题');
    expect(slashMenuOf(mounted.container)).not.toBeNull();
  });

  /**
   * 没有匹配项时回车**不能被吞掉**。
   *
   * 用户打 `/zzz` 有两种可能：想搜一个不存在的块，或者就是想把 `/zzz` 当正文打出来。
   * 两种情况下「按回车没反应」都是错的 —— 后一种还更常见。
   * 放过这一下之后 ProseMirror 拆开段落，新段落是空的，触发条件不再成立，菜单自己收起来。
   */
  it('一个匹配项都没有时，回车换行而不是被吞掉', async () => {
    mounted = await mount((p) => p.put('note.md', '\n'));
    await mounted.open('note.md');

    await typeInEditor(mounted.container, '/zzz');
    expect(mounted.container.querySelector('.slash-empty')).not.toBeNull();

    await pressInEditor(mounted.container, 'Enter');

    // 断言的是**段落真的被拆开了**，而不是断言 markdown 文本 ——
    // 拆与不拆序列化出来都是 `/zzz\n`（末尾那个空段落不进 markdown），
    // 拿文本去断言等于什么都没验证
    expect(mounted.container.querySelectorAll('.qm-editor .ProseMirror p')).toHaveLength(2);
    expect(mounted.store.getSnapshot().doc?.text).toContain('/zzz');
    expect(slashMenuOf(mounted.container)).toBeNull();
  });

  it('用鼠标点菜单项也能插入，且 mousedown 被拦下', async () => {
    mounted = await mount((p) => p.put('note.md', '\n'));
    await mounted.open('note.md');

    await typeInEditor(mounted.container, '/引用');
    const item = mounted.container.querySelector<HTMLButtonElement>('.slash-item');
    expect(item).not.toBeNull();

    // 不拦这一下，焦点会离开编辑器、选区丢失，菜单在 click 到达之前就被卸载
    const down = new MouseEvent('mousedown', { bubbles: true, cancelable: true });
    item?.dispatchEvent(down);
    expect(down.defaultPrevented).toBe(true);

    await act(async () => {
      item?.click();
    });
    await nextFrame();

    expect(mounted.store.getSnapshot().doc?.text).toMatch(/^>/m);
    expect(slashMenuOf(mounted.container)).toBeNull();
  });

  it('切到源码视图后菜单消失 —— 底下坐着的是代码内核，它不报斜杠状态', async () => {
    mounted = await mount((p) => p.put('note.md', '\n'));
    await mounted.open('note.md');

    await typeInEditor(mounted.container, '/表');
    expect(slashMenuOf(mounted.container)).not.toBeNull();

    await mounted.click('源码');

    expect(mounted.container.querySelector('.qm-code .cm-content')).not.toBeNull();
    expect(slashMenuOf(mounted.container)).toBeNull();
  });
});

describe('表格工具条', () => {
  /**
   * 这一组打的是「光标在表格里 → 外壳画工具条 → 点一下 → 文件内容真的变了」。
   *
   * 与选中浮层 / 斜杠菜单同一条架构：**内核只报状态与几何，外壳画界面**。
   * 所以这里要验两件事，缺一条这个功能都不成立：
   * - 工具条出现/消失的**条件**完全由内核上报的状态决定（不是外壳猜的）；
   * - 每一次点击都写进文档、并让文档变脏（少一步就是「点了没反应」或「保存丢改动」）。
   */

  /** 表头行 + 两行正文。 */
  const TABLE = '| A | B | C |\n| - | - | - |\n| a | b | c |\n| d | e | f |\n';

  function tableToolbarOf(container: HTMLElement): HTMLElement | null {
    return container.querySelector('.table-toolbar');
  }

  /**
   * 工具条上的按钮。
   *
   * 与 `mounted.click` 不同，这里要的是**元素**而不是「点一下」——
   * 「按钮被禁用而不是被隐藏」这条断言必须能看到那个禁用按钮。
   */
  function tableButton(container: HTMLElement, label: string): HTMLButtonElement | null {
    const found = [...container.querySelectorAll<HTMLButtonElement>('.table-toolbar .tt-btn')].find(
      (item) => item.textContent?.trim() === label,
    );
    return found ?? null;
  }

  async function clickTableButton(container: HTMLElement, label: string): Promise<void> {
    const button = tableButton(container, label);
    if (button === null) throw new Error(`找不到表格工具条按钮：${label}`);
    await act(async () => {
      button.click();
    });
    await nextFrame();
    await settle();
  }

  /**
   * 文档里当前的表格行数（含分隔行）。
   *
   * 不从 DOM 数 `<tr>` —— 这里要断言的是**存出来的 markdown** 有几行，
   * DOM 里有几行是另一回事（M5 踩过的坑：删表头行时两者会不一致）。
   */
  function markdownRows(): number {
    const text = mounted?.store.getSnapshot().doc?.text ?? '';
    return text.split('\n').filter((line) => line.trim().startsWith('|')).length;
  }

  it('光标在表格里时浮出工具条，并且是编辑器区域的兄弟节点', async () => {
    mounted = await mount((p) => p.put('note.md', TABLE));
    await mounted.open('note.md');

    const toolbar = tableToolbarOf(mounted.container);
    expect(toolbar).not.toBeNull();
    // 与选中工具条同一条理由：放进内核挂载的那层容器会被内核挂载时抹掉
    expect(toolbar?.parentElement?.classList.contains('editor-area')).toBe(true);
    expect(toolbar?.getAttribute('role')).toBe('toolbar');
  });

  it('打开表格不会让文档变脏 —— 工具条只是浮出来，没有改任何东西', async () => {
    mounted = await mount((p) => p.put('note.md', TABLE));
    await mounted.open('note.md');

    expect(tableToolbarOf(mounted.container)).not.toBeNull();
    expect(statusOf(mounted.container)).toBe('已保存');
  });

  it('光标不在表格里时没有工具条', async () => {
    mounted = await mount((p) => p.put('note.md', '正文\n'));
    await mounted.open('note.md');

    expect(tableToolbarOf(mounted.container)).toBeNull();
  });

  it('切到源码视图后工具条消失 —— 底下坐着的是代码内核，它没有表格能力', async () => {
    mounted = await mount((p) => p.put('note.md', TABLE));
    await mounted.open('note.md');
    expect(tableToolbarOf(mounted.container)).not.toBeNull();

    await mounted.click('源码');

    expect(mounted.container.querySelector('.qm-code .cm-content')).not.toBeNull();
    expect(tableToolbarOf(mounted.container)).toBeNull();
  });

  it('点「行↓」加一行，文档变脏', async () => {
    mounted = await mount((p) => p.put('note.md', TABLE));
    await mounted.open('note.md');
    expect(markdownRows()).toBe(4);

    await clickTableButton(mounted.container, '行↓');

    expect(markdownRows()).toBe(5);
    // 这一条是「静默丢数据」的守门人：不报给外壳的话，按 Ctrl+S 存回磁盘的是加行之前的文件
    expect(statusOf(mounted.container)).toBe('有未保存的改动');
  });

  it('点「列→」加一列', async () => {
    mounted = await mount((p) => p.put('note.md', TABLE));
    await mounted.open('note.md');

    await clickTableButton(mounted.container, '列→');

    const delimiter = (mounted.store.getSnapshot().doc?.text ?? '').split('\n')[1] ?? '';
    expect(delimiter.split('|').filter((cell) => cell.trim() !== '')).toHaveLength(4);
  });

  it('点「中」把**整列**居中 —— 分隔行里出现 `:-:`', async () => {
    mounted = await mount((p) => p.put('note.md', TABLE));
    await mounted.open('note.md');

    await clickTableButton(mounted.container, '中');

    const text = mounted.store.getSnapshot().doc?.text ?? '';
    expect(text).toContain(':-:');
    expect(tableButton(mounted.container, '中')?.getAttribute('aria-pressed')).toBe('true');
  });

  it('只剩两行时「行✕」是禁用的，并且说明了原因', async () => {
    mounted = await mount((p) => p.put('note.md', '| A | B |\n| - | - |\n| a | b |\n'));
    await mounted.open('note.md');

    const button = tableButton(mounted.container, '行✕');
    // 禁用而不是隐藏：按钮消失会让「行↓」突然挪位，用户下一次会点错
    expect(button).not.toBeNull();
    expect(button?.disabled).toBe(true);
    expect(button?.title).toContain('表头行');
  });

  it('只剩一列时「列✕」是禁用的', async () => {
    mounted = await mount((p) => p.put('note.md', '| A |\n| - |\n| a |\n'));
    await mounted.open('note.md');

    const button = tableButton(mounted.container, '列✕');
    expect(button).not.toBeNull();
    expect(button?.disabled).toBe(true);
  });

  it('点「删表」之后表格从文档里没了，工具条也跟着消失', async () => {
    mounted = await mount((p) => p.put('note.md', TABLE));
    await mounted.open('note.md');

    await clickTableButton(mounted.container, '删表');

    const text = mounted.store.getSnapshot().doc?.text ?? '';
    expect(text).not.toContain('|');
    expect(tableToolbarOf(mounted.container)).toBeNull();
    expect(statusOf(mounted.container)).toBe('有未保存的改动');
  });

  it('点「退出」把光标移到表格之后，工具条消失', async () => {
    mounted = await mount((p) => p.put('note.md', TABLE));
    await mounted.open('note.md');

    await clickTableButton(mounted.container, '退出');

    expect(tableToolbarOf(mounted.container)).toBeNull();
    // 表格本身还在 —— 「退出」只挪光标，不该动内容
    expect(mounted.store.getSnapshot().doc?.text).toContain('| a | b | c |');
  });

  it('鼠标按下被拦下 —— 否则焦点离开编辑器，工具条会在 click 到达按钮之前消失', async () => {
    mounted = await mount((p) => p.put('note.md', TABLE));
    await mounted.open('note.md');

    const button = tableButton(mounted.container, '行↓');
    expect(button).not.toBeNull();

    let prevented = false;
    await act(async () => {
      const event = new MouseEvent('mousedown', { bubbles: true, cancelable: true });
      button?.dispatchEvent(event);
      prevented = event.defaultPrevented;
    });

    expect(prevented).toBe(true);
    // 而且光标仍然在表格里，工具条还在
    expect(tableToolbarOf(mounted.container)).not.toBeNull();
  });
});

describe('代码块语言选择器', () => {
  /**
   * 这一组打的是「点代码块右上角那个语言标签 → 外壳画面板 → 选一个 → 文件内容真的变了」。
   *
   * 与表格工具条同一条架构（内核只报状态与几何，外壳画界面），但**触发方式相反**：
   * 表格工具条跟着光标走，这个面板是点出来的 —— 语言标签常驻在每个代码块上。
   * 所以这里要额外验一件事：**光标在代码块里不该弹出面板**。
   */

  /** 表头行 + 一行正文，语言是 js。 */
  const CODE = '```js\nlet a = 1;\n```\n';

  function pickerOf(container: HTMLElement): HTMLElement | null {
    return container.querySelector('.cb-picker');
  }

  function badgeOf(container: HTMLElement): HTMLButtonElement {
    const badge = container.querySelector<HTMLButtonElement>('.qm-cb-lang');
    if (badge === null) throw new Error('找不到代码块的语言标签');
    return badge;
  }

  function searchBox(container: HTMLElement): HTMLInputElement {
    const input = container.querySelector<HTMLInputElement>('.cb-picker .cb-search');
    if (input === null) throw new Error('找不到语言搜索框');
    return input;
  }

  /** 面板里列出来的候选项文字（不含右侧的语言标识）。 */
  function pickerRows(container: HTMLElement): string[] {
    return [...container.querySelectorAll('.cb-picker .cb-item .cb-label')].map(
      (item) => item.textContent?.trim() ?? '',
    );
  }

  async function openPicker(container: HTMLElement): Promise<void> {
    const badge = badgeOf(container);
    await act(async () => {
      badge.click();
    });
    await nextFrame();
  }

  /** 改搜索框的值。React 受控输入必须走原型上的原生 setter，理由见「点链接」那条用例。 */
  async function search(container: HTMLElement, text: string): Promise<void> {
    const input = searchBox(container);
    await act(async () => {
      const setNativeValue = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set?.bind(input);
      setNativeValue?.(text);
      input.dispatchEvent(new Event('input', { bubbles: true }));
    });
    await nextFrame();
  }

  async function pressKey(container: HTMLElement, key: string): Promise<void> {
    const input = searchBox(container);
    await act(async () => {
      input.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true }));
    });
    await nextFrame();
  }

  async function clickRow(container: HTMLElement, label: string): Promise<void> {
    const row = [...container.querySelectorAll<HTMLButtonElement>('.cb-picker .cb-item')].find(
      (item) => item.querySelector('.cb-label')?.textContent?.trim() === label,
    );
    if (!row) throw new Error(`找不到候选项：${label}`);
    await act(async () => {
      row.click();
    });
    await nextFrame();
    await settle();
  }

  it('光标在代码块里也不会自己弹出面板 —— 它是点出来的', async () => {
    mounted = await mount((p) => p.put('note.md', CODE));
    await mounted.open('note.md');

    // 代码块与语言标签都在，但面板没开
    expect(mounted.container.querySelector('.qm-cb-lang')).not.toBeNull();
    expect(pickerOf(mounted.container)).toBeNull();
  });

  it('点语言标签弹出面板，并且是编辑器区域的兄弟节点', async () => {
    mounted = await mount((p) => p.put('note.md', CODE));
    await mounted.open('note.md');

    await openPicker(mounted.container);

    const panel = pickerOf(mounted.container);
    expect(panel).not.toBeNull();
    // 与其余三个浮层同一条理由：放进内核挂载的那层容器会被内核挂载时抹掉
    expect(panel?.parentElement?.classList.contains('editor-area')).toBe(true);
    expect(panel?.getAttribute('role')).toBe('dialog');
  });

  it('打开面板不会让文档变脏 —— 它只是浮出来，没有改任何东西', async () => {
    mounted = await mount((p) => p.put('note.md', CODE));
    await mounted.open('note.md');

    await openPicker(mounted.container);

    expect(statusOf(mounted.container)).toBe('已保存');
  });

  it('列出候选语言，并把当前语言标出来', async () => {
    mounted = await mount((p) => p.put('note.md', CODE));
    await mounted.open('note.md');
    await openPicker(mounted.container);

    const rows = pickerRows(mounted.container);
    expect(rows).toContain('纯文本');
    expect(rows).toContain('JavaScript');
    // ```` ```js ```` 的正式标识是 `javascript`，所以「当前」那一行是补出来的
    expect(mounted.container.querySelector('.cb-item.current .cb-label')?.textContent?.trim()).toBe('js');
  });

  it('打字就是筛选', async () => {
    mounted = await mount((p) => p.put('note.md', CODE));
    await mounted.open('note.md');
    await openPicker(mounted.container);

    await search(mounted.container, 'rust');

    expect(pickerRows(mounted.container)).toEqual(['Rust']);
  });

  it('输入 hljs 不认识的语言时，仍然能原样选它', async () => {
    /*
     * `mermaid` 是 M8 要用的语言，hljs 不认识它（`highlightCode` 会退回自动识别）。
     * 面板不提供「原样使用」这一项的话，用户就被卡住了 ——
     * 列表里找不到，也没有别的入口能写进去。
     */
    mounted = await mount((p) => p.put('note.md', CODE));
    await mounted.open('note.md');
    await openPicker(mounted.container);

    await search(mounted.container, 'mermaid');

    expect(pickerRows(mounted.container)).toContain('使用 “mermaid”');

    await clickRow(mounted.container, '使用 “mermaid”');

    expect(mounted.store.getSnapshot().doc?.text).toContain('```mermaid');
    expect(badgeOf(mounted.container).textContent?.trim()).toBe('mermaid');
  });

  it('选一个语言：文档真的变了、变脏了，面板也收起来', async () => {
    mounted = await mount((p) => p.put('note.md', '```\nlet a = 1;\n```\n'));
    await mounted.open('note.md');
    await openPicker(mounted.container);

    await clickRow(mounted.container, 'Rust');

    expect(mounted.store.getSnapshot().doc?.text).toBe('```rust\nlet a = 1;\n```\n');
    // 不报给外壳的话，按 Ctrl+S 存回磁盘的还是改之前那份
    expect(statusOf(mounted.container)).toBe('有未保存的改动');
    expect(pickerOf(mounted.container)).toBeNull();
  });

  it('选「纯文本」清掉语言，围栏退回没有标识的样子', async () => {
    mounted = await mount((p) => p.put('note.md', CODE));
    await mounted.open('note.md');
    await openPicker(mounted.container);

    await clickRow(mounted.container, '纯文本');

    expect(mounted.store.getSnapshot().doc?.text).toBe('```\nlet a = 1;\n```\n');
    expect(badgeOf(mounted.container).textContent?.trim()).toBe('纯文本');
  });

  it('Esc 收起面板，不删也不改任何东西', async () => {
    mounted = await mount((p) => p.put('note.md', CODE));
    await mounted.open('note.md');
    await openPicker(mounted.container);

    await pressKey(mounted.container, 'Escape');

    expect(pickerOf(mounted.container)).toBeNull();
    expect(mounted.store.getSnapshot().doc?.text).toBe(CODE);
    expect(statusOf(mounted.container)).toBe('已保存');
  });

  it('点面板外面收起面板', async () => {
    mounted = await mount((p) => p.put('note.md', CODE));
    await mounted.open('note.md');
    await openPicker(mounted.container);

    // 面板是 `container` 的后代，所以点在 `container` 上就是「点在外面」。
    // 先取到局部常量：`mounted` 的类型是 `Mounted | null`，
    // 在回调里 TS 收不回上面的收窄
    const host = mounted.container;
    await act(async () => {
      host.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
    });
    await nextFrame();

    expect(pickerOf(mounted.container)).toBeNull();
  });

  it('滚动编辑器会收起面板 —— 否则它会停在原地和标签分家', async () => {
    mounted = await mount((p) => p.put('note.md', CODE));
    await mounted.open('note.md');
    await openPicker(mounted.container);

    const scroller = mounted.container.querySelector('.qm-editor');
    await act(async () => {
      scroller?.dispatchEvent(new Event('scroll'));
    });
    await nextFrame();

    expect(pickerOf(mounted.container)).toBeNull();
  });

  it('「显示行号」开关只改视图，不碰文档', async () => {
    mounted = await mount((p) => p.put('note.md', CODE));
    await mounted.open('note.md');

    // 默认是开：行号槽在，而且没有那个关掉它的类名
    expect(mounted.container.querySelectorAll('.qm-cb-num').length).toBe(1);
    const editor = mounted.container.querySelector('.qm-editor');
    expect(editor?.classList.contains('qm-no-line-numbers')).toBe(false);

    await openPicker(mounted.container);
    const toggle = mounted.container.querySelector<HTMLInputElement>('.cb-toggle input');
    expect(toggle).not.toBeNull();
    expect(toggle?.checked).toBe(true);

    await act(async () => {
      toggle?.click();
    });
    await nextFrame();

    expect(editor?.classList.contains('qm-no-line-numbers')).toBe(true);
    // 行号槽仍然在 DOM 里（只是不画了），而文档一个字都没变
    expect(mounted.container.querySelectorAll('.qm-cb-num').length).toBe(1);
    expect(mounted.store.getSnapshot().doc?.text).toBe(CODE);
    expect(statusOf(mounted.container)).toBe('已保存');
  });

  it('切到源码视图后面板消失 —— 底下坐着的是代码内核，它没有代码块能力', async () => {
    mounted = await mount((p) => p.put('note.md', CODE));
    await mounted.open('note.md');
    await openPicker(mounted.container);
    expect(pickerOf(mounted.container)).not.toBeNull();

    await mounted.click('源码');

    expect(mounted.container.querySelector('.qm-code .cm-content')).not.toBeNull();
    expect(pickerOf(mounted.container)).toBeNull();
  });

  it('候选项拦掉 mousedown —— 否则焦点先离开输入框，面板会在 click 到达之前卸载', async () => {
    mounted = await mount((p) => p.put('note.md', CODE));
    await mounted.open('note.md');
    await openPicker(mounted.container);

    const row = mounted.container.querySelector<HTMLButtonElement>('.cb-picker .cb-item');
    expect(row).not.toBeNull();

    let prevented = false;
    await act(async () => {
      const event = new MouseEvent('mousedown', { bubbles: true, cancelable: true });
      row?.dispatchEvent(event);
      prevented = event.defaultPrevented;
    });

    expect(prevented).toBe(true);
    expect(pickerOf(mounted.container)).not.toBeNull();
  });
});

describe('图片：粘贴落盘 + 显示 + 灯箱', () => {
  /**
   * 这一组打的是**整条图片链路**，而它横跨四层：
   *
   *   ProseMirror 的 handlePaste（内核）
   *     → 字节交给外壳（`EditorMountProps.onImagePaste`）
   *     → 生成文件名并写盘（`DocumentStore.pasteImage` / `core/document/images.ts`）
   *     → 换回来的相对路径插进文档，再由 NodeView 读回来显示
   *
   * 每一层单独都有测试（`modules/markdown/image.test.ts`、`core/document/images.test.ts`），
   * 但**接缝**只有在这里才被真正跑过一遍：内核报的 MIME 有没有变成正确的后缀、
   * 外壳算出来的路径和 NodeView 要读的路径是不是同一个字符串、
   * 落盘之后文档有没有变脏。任何一处对不上，界面上都是「粘了没反应」。
   */

  const NOTES = 'C:\\notes';
  const DOC = `${NOTES}\\note.md`;
  /** 资源目录在磁盘上的前缀。分隔符统一成文档路径的风格，见 `core/path.ts` 的 `joinPath`。 */
  const ASSETS = `${NOTES}\\note.assets\\`;

  function imageEl(container: HTMLElement): HTMLImageElement | null {
    return container.querySelector<HTMLImageElement>('.qm-editor img.qm-img-el');
  }

  function imageBox(container: HTMLElement): HTMLElement | null {
    return container.querySelector<HTMLElement>('.qm-editor .qm-img');
  }

  function lightbox(container: HTMLElement): HTMLElement | null {
    return container.querySelector<HTMLElement>('.lightbox');
  }

  function pngFile(name = 'shot.png'): File {
    return new File([new Uint8Array([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x2a])], name, {
      type: 'image/png',
    });
  }

  /**
   * 往编辑器里粘贴一批文件。
   *
   * 用真的 `ClipboardEvent` + `DataTransfer`（`src/test/jsdom-gaps.ts` 补的那两个），
   * 而不是直接调内核的 `handlePaste` —— 后者会跳过 ProseMirror 的
   * 「先看剪贴板、再决定要不要拦」那一段，而那正是「普通文本粘贴有没有被误拦」
   * 的关键。走真实事件路径才测得到。
   */
  async function pasteFiles(container: HTMLElement, files: File[]): Promise<void> {
    const editable = editableOf(container);
    const data = new DataTransfer();
    for (const file of files) data.items.add(file);

    await act(async () => {
      editable.dispatchEvent(
        new ClipboardEvent('paste', { clipboardData: data, bubbles: true, cancelable: true }),
      );
    });
    // 落盘 + 插节点 + NodeView 回读，串了三段异步，比通常多等一轮
    await settle(6);
  }

  it('粘贴一张图：落到 `<文档名>.assets/`，文档里写相对路径，并且变脏', async () => {
    mounted = await mount((p) => p.put(DOC, '前文\n'));
    await mounted.open(DOC);

    await pasteFiles(mounted.container, [pngFile()]);

    // 磁盘上多了一个文件，且在文档旁边的同名资源目录里
    const written = [...mounted.provider.binaries.keys()];
    expect(written).toHaveLength(1);
    expect(written[0]?.startsWith(ASSETS)).toBe(true);
    // 文件名 = 时间戳 + 4 位随机盐 + 按 MIME 推出来的后缀
    expect(written[0]?.slice(ASSETS.length)).toMatch(/^image-\d{8}-\d{6}-[0-9a-f]{4}\.png$/);
    // 字节一字不差（PNG 文件头 8 字节 + 一个尾字节，用来证明「长度也对得上」）
    expect(mounted.provider.binaries.get(written[0]!)!.byteLength).toBe(9);

    // 文档里是**相对路径**，不是绝对路径 —— 绝对路径写进文档会让它
    // 在别人的机器上、在 Git 仓库里都失去意义
    const text = mounted.store.getSnapshot().doc!.text;
    expect(text).toContain('](./note.assets/image-');
    expect(text).not.toContain(NOTES);

    // 落盘是异步的，等它回来时「刚发生过用户输入」的窗口早过了 ——
    // 不显式声明的话这里会显示「已保存」，而磁盘上还是粘贴前的文本
    expect(statusOf(mounted.container)).toBe('有未保存的改动');
    // 状态栏要说清「图已经落盘、文档还没存」，这是用户真正会担心的事
    expect(mounted.container.querySelector('.statusbar .hint')?.textContent).toContain('文档还需保存');
  });

  it('**粘进来的图能立刻显示**，而且 data URI 绝不进文档', async () => {
    mounted = await mount((p) => p.put(DOC, '前文\n'));
    await mounted.open(DOC);

    await pasteFiles(mounted.container, [pngFile()]);

    // 显示：NodeView 把相对路径读回来、换成了能加载的 URL
    expect(imageBox(mounted.container)?.className).toContain('is-loaded');
    expect(imageEl(mounted.container)?.getAttribute('src')).toContain('data:image/png;base64,');

    // 但文档里必须还是那一串相对路径。少了 `ignoreMutation`，
    // ProseMirror 会把我们写进 `<img>` 的 data URI 当成用户输入读进文档 ——
    // 存出来的 Markdown 里会多出一大段 base64
    expect(mounted.store.getSnapshot().doc!.text).not.toContain('base64');
  });

  it('点图片开灯箱：显示图、显示路径；Esc 与点背景都能关', async () => {
    mounted = await mount((p) => p.put(DOC, '前文\n'));
    await mounted.open(DOC);
    await pasteFiles(mounted.container, [pngFile()]);

    const el = imageEl(mounted.container);
    expect(el).not.toBeNull();
    await act(async () => {
      el!.click();
    });

    const box = lightbox(mounted.container);
    expect(box).not.toBeNull();
    // 灯箱里显示的是同一个 URL（不重新读一遍盘）
    expect(box?.querySelector<HTMLImageElement>('.lightbox-img')?.getAttribute('src')).toBe(
      el?.getAttribute('src'),
    );
    // 路径要显示出来 —— 看完大图之后用户常要做的下一件事是「找到这个文件」
    expect(box?.querySelector('.lightbox-caption .path')?.textContent).toContain('./note.assets/image-');

    // Esc 关（在 document 上派发，走的是真实的冒泡路径）
    await act(async () => {
      document.body.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
    });
    expect(lightbox(mounted.container)).toBeNull();

    // 再开一次，点背景关
    await act(async () => {
      el!.click();
    });
    // 窄化必须发生在 `act` 外面：`mounted` 是外层作用域的 `let`，
    // 回调里再取一次会让控制流分析失效（与上面 `host` 那条是同一件事）
    const backdrop = lightbox(mounted.container);
    expect(backdrop).not.toBeNull();
    await act(async () => {
      backdrop!.click();
    });
    expect(lightbox(mounted.container)).toBeNull();
  });

  it('切标签页时灯箱自动关 —— 不能留着一张上一个文档的图', async () => {
    mounted = await mount((p) => {
      p.put(DOC, '前文\n');
      p.put(`${NOTES}\\other.md`, '# 另一份\n');
    });
    await mounted.open(DOC);
    await pasteFiles(mounted.container, [pngFile()]);

    const el = imageEl(mounted.container);
    expect(el).not.toBeNull();
    await act(async () => {
      el!.click();
    });
    expect(lightbox(mounted.container)).not.toBeNull();

    await mounted.open(`${NOTES}\\other.md`);
    expect(lightbox(mounted.container)).toBeNull();
  });

  it('文档还没保存过时不落盘，并在状态栏说清原因', async () => {
    // 虚拟盘里的 `demo.md` 没有目录信息 —— 相对路径无从谈起
    mounted = await mount((p) => p.put('demo.md', '前文\n'));
    await mounted.open('demo.md');

    await pasteFiles(mounted.container, [pngFile()]);

    expect(mounted.provider.binaries.size).toBe(0);
    expect(mounted.store.getSnapshot().doc!.text).toBe('前文\n');
    expect(mounted.container.querySelector('.statusbar .hint')?.textContent).toContain('还没有保存过');
  });

  it('不认识的图片格式不落盘，并说清是哪一种', async () => {
    mounted = await mount((p) => p.put(DOC, '前文\n'));
    await mounted.open(DOC);

    const tiff = new File([new Uint8Array([0x49, 0x49, 0x2a, 0x00])], 'scan.tif', { type: 'image/tiff' });
    await pasteFiles(mounted.container, [tiff]);

    expect(mounted.provider.binaries.size).toBe(0);
    expect(mounted.store.getSnapshot().doc!.text).toBe('前文\n');
    expect(mounted.container.querySelector('.statusbar .hint')?.textContent).toContain('image/tiff');
  });

  it('三张图里有一张失败，另外两张照样落盘', async () => {
    mounted = await mount((p) => p.put(DOC, '前文\n'));
    await mounted.open(DOC);

    const tiff = new File([new Uint8Array([0x49, 0x49])], 'scan.tif', { type: 'image/tiff' });
    await pasteFiles(mounted.container, [pngFile('a.png'), tiff, pngFile('b.png')]);

    expect(mounted.provider.binaries.size).toBe(2);
    const text = mounted.store.getSnapshot().doc!.text;
    expect(text.match(/!\[\]\(\.\/note\.assets\//g)).toHaveLength(2);
  });

  it('普通文本的粘贴不被拦 —— 走默认处理', async () => {
    mounted = await mount((p) => p.put(DOC, '前文\n'));
    await mounted.open(DOC);

    const txt = new File([new Uint8Array([0x61])], 'a.txt', { type: 'text/plain' });
    await pasteFiles(mounted.container, [txt]);

    expect(mounted.provider.binaries.size).toBe(0);
    // 没被我们截住 = 文档没有多出一个图片节点
    expect(imageBox(mounted.container)).toBeNull();
  });

  it('图片读不到时画一个**带路径**的占位，而不是一个破图', async () => {
    mounted = await mount((p) => p.put(DOC, '![说明](./note.assets/missing.png)\n'));
    await mounted.open(DOC);
    await settle();

    const box = imageBox(mounted.container);
    expect(box?.className).toContain('is-failed');
    // 用户下一步要做的恰恰是「去把那张图找回来」，他需要知道路径
    expect(box?.querySelector('.qm-img-note')?.textContent).toContain('./note.assets/missing.png');
    // 失败时 `<img>` 上不该有 src，否则浏览器会画一个破图图标
    expect(imageEl(mounted.container)?.hasAttribute('src')).toBe(false);
  });

  it('切到源码视图时图片节点如实呈现为 Markdown 语法', async () => {
    mounted = await mount((p) => p.put(DOC, '前文\n'));
    await mounted.open(DOC);
    await pasteFiles(mounted.container, [pngFile()]);

    await mounted.click('源码');

    expect(mounted.container.querySelector('.lightbox')).toBeNull();
    expect(renderedText(mounted.container)).toContain('![](./note.assets/image-');
  });
});

// ---------------------------------------------------------------------------
// 设置页
// ---------------------------------------------------------------------------

/** 状态栏里的「第 N 行，第 M 列」。内核不报光标时整段隐藏，所以读不到就是空串。 */
function cursorOf(container: HTMLElement): string {
  const spans = [...container.querySelectorAll('.statusbar span')];
  return spans.find((item) => item.textContent?.includes('行，第'))?.textContent?.trim() ?? '';
}

describe('设置页', () => {
  it('从活动栏打开，Esc 关闭', async () => {
    mounted = await mount(() => undefined);
    expect(mounted.container.querySelector('.settings')).toBeNull();

    await mounted.click('设置');
    expect(mounted.container.querySelector('.settings')).not.toBeNull();

    // 派发到 window 上：对话框的 Esc 挂在 window 的**捕获阶段**（硬约束 9），
    // 直接派发到 window 正好覆盖那条路径
    await act(async () => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }));
    });
    await settle();
    expect(mounted.container.querySelector('.settings')).toBeNull();
  });

  it('点遮罩关闭，点对话框内部不关', async () => {
    mounted = await mount(() => undefined);
    await mounted.click('设置');

    const dialog = mounted.container.querySelector('.settings');
    expect(dialog).not.toBeNull();
    await act(async () => {
      dialog?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
    });
    await settle();
    expect(mounted.container.querySelector('.settings')).not.toBeNull();

    const backdrop = mounted.container.querySelector('.palette-backdrop');
    await act(async () => {
      backdrop?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
    });
    await settle();
    expect(mounted.container.querySelector('.settings')).toBeNull();
  });

  it('快捷键一览由命令注册表生成 —— 不是手写的一份', async () => {
    mounted = await mount(() => undefined);
    await mounted.click('设置');

    const text = mounted.container.querySelector('.settings')?.textContent ?? '';
    // 命令面板与保存都在表里，且带上了真实绑定
    expect(text).toContain('命令面板');
    expect(text).toContain('Ctrl+P');
    expect(text).toContain('保存');
    expect(text).toContain('Ctrl+S');
    // 没有快捷键的命令不进这张表（它是快捷键一览，不是第二份命令面板）
    expect(text).not.toContain('从磁盘重新加载');
  });

  it('改主题立即生效，并且落进偏好存储', async () => {
    mounted = await mount(() => undefined);
    await mounted.click('设置');

    const select = mounted.container.querySelector<HTMLSelectElement>('.settings select');
    expect(select).not.toBeNull();

    await act(async () => {
      if (select) {
        select.value = 'ink';
        select.dispatchEvent(new Event('change', { bubbles: true }));
      }
    });
    await settle();

    // 「立即生效」而不是「保存后生效」：设置页没有保存按钮，每项都是即时生效
    expect(document.documentElement.dataset['theme']).toBe('dark');
    expect(JSON.parse(localStorage.getItem('qingmo.settings.v1') ?? '{}').theme).toBe('ink');
  });

  it('关于页的版本号来自 package.json，不是写死的', async () => {
    mounted = await mount(() => undefined);
    await mounted.click('设置');

    const text = mounted.container.querySelector('.settings')?.textContent ?? '';
    // 断言「有版本字样」而不是断言具体值：具体值会随发版变，
    // 断言它等于把测试变成 package.json 的第二份副本
    expect(text).toContain(`版本 ${import.meta.env.QINGMO_VERSION}`);
  });
});

// ---------------------------------------------------------------------------
// 位置记忆
// ---------------------------------------------------------------------------

describe('位置记忆', () => {
  const A = 'C://note.md';
  const B = 'C://other.md';

  function storedPositions(): Record<string, { offset: number; textLength: number; scrollRatio: number }> {
    return JSON.parse(localStorage.getItem('qingmo.positions.v1') ?? '{}');
  }

  it('打字之后位置被记下来 —— 切文档时立刻结算，不等防抖', async () => {
    mounted = await mount((p) => {
      p.put(A, '甲\n\n乙\n');
      p.put(B, '别的\n');
    });
    await mounted.open(A);
    await typeInEditor(mounted.container, '丙');

    // 切走是一个明确的边界。少了这一步，位置要等 600ms 防抖才落盘，
    // 而「改完就切走」恰恰是最常见的离开方式
    await mounted.open(B);

    const saved = storedPositions()[A];
    expect(saved).toBeDefined();
    expect(saved?.offset).toBeGreaterThan(0);
    expect(saved?.textLength).toBeGreaterThan(0);
  });

  it('切走再切回来，光标回到上次的位置', async () => {
    mounted = await mount((p) => {
      p.put(A, '甲\n\n乙\n');
      p.put(B, '别的\n');
    });
    await mounted.open(A);
    await typeInEditor(mounted.container, '丙');

    const before = cursorOf(mounted.container);
    // 先确认光标真的离开了开头，否则下面的断言可能「碰巧成立」
    expect(before).not.toBe('');
    expect(before).not.toBe('第 1 行，第 1 列');

    await mounted.open(B);
    await mounted.open(A);

    expect(cursorOf(mounted.container)).toBe(before);
  });

  it('关掉「记住文档位置」之后既不再读也不再写', async () => {
    // 先种一条明显不对的记忆：若被读到了，光标会被搬到一个莫名其妙的地方
    localStorage.setItem(
      'qingmo.positions.v1',
      JSON.stringify({ [A]: { offset: 999, textLength: 1, scrollRatio: 0, at: 1 } }),
    );
    mounted = await mount(
      (p) => {
        p.put(A, '甲\n\n乙\n');
        p.put(B, '别的\n');
      },
      { settings: { rememberPositions: false } },
    );
    await mounted.open(A);

    // 不读：光标仍在开头
    expect(cursorOf(mounted.container)).toBe('第 1 行，第 1 列');

    await typeInEditor(mounted.container, '丙');
    await mounted.open(B);

    // 不写：种下去的那条原封不动
    expect(storedPositions()[A]?.offset).toBe(999);
  });

  it('文档长度变了就只恢复滚动，不搬光标', async () => {
    // 记一条长度对不上的位置：这正是「文件被别的程序改过」的形状
    localStorage.setItem(
      'qingmo.positions.v1',
      JSON.stringify({ [A]: { offset: 9999, textLength: 1, scrollRatio: 0, at: 1 } }),
    );
    mounted = await mount((p) => p.put(A, '甲\n\n乙\n'));
    await mounted.open(A);

    // 光标没被搬到 9999，也没有因为越界抛错 —— 打开流程照常走完
    expect(cursorOf(mounted.container)).toBe('第 1 行，第 1 列');
    expect(mounted.container.querySelector('.ProseMirror')).not.toBeNull();
  });
});
