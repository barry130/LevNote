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
