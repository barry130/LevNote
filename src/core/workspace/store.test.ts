// @vitest-environment node
/**
 * 工作区模型的测试。
 *
 * 这里重点钉的是「多标签」相对「单文档」新引入的那些语义 ——
 * 单文档时代已经由 `store.test.ts` 钉住的（脏判定、mtime 乐观锁、冲突）
 * 不在这里重复测，只测它们**在多个标签页之间不会互相串台**。
 */

import { describe, expect, it } from 'vitest';

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
} from '../document/types.js';
import { WorkspaceStore } from './store.js';

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
  /** 目录路径 → 子项。空目录也要有键，否则测试分不清「空」和「没加载」。 */
  readonly dirs = new Map<string, DirEntry[]>();
  readonly writes: WriteRequest[] = [];
  directoryToPick: string | null = null;
  /** 设成 true 后所有 listDir 都失败，用来测错误分支 */
  failListDir = false;
  private clock = 1000;

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
    const previous = this.files.get(request.path);
    if (request.expectedMtimeMs !== undefined && previous && previous.mtimeMs !== request.expectedMtimeMs) {
      throw { code: 'CONFLICT', message: '磁盘上已被改写', detail: null };
    }
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

  async statFile(path: string): Promise<FileStat> {
    const file = this.need(path);
    return { path, size: file.text.length, mtimeMs: file.mtimeMs, readonly: file.readonly, isDir: false };
  }

  async listDir(path: string): Promise<DirEntry[]> {
    if (this.failListDir) throw new Error('读不了');
    const entries = this.dirs.get(path);
    if (!entries) throw { code: 'NOT_FOUND', message: `没有这个目录：${path}`, detail: null };
    return entries;
  }

  async sniffEncoding(path: string): Promise<string> {
    return this.need(path).encoding;
  }

  async readBinaryFile(): Promise<ArrayBuffer> {
    return new ArrayBuffer(0);
  }

  /** 这个套件只测标签页语义，不测图片落盘；接口要求存在。 */
  async writeBinaryFile(path: string, bytes: ArrayBuffer): Promise<BinaryWriteOutcome> {
    return { path, size: bytes.byteLength, mtimeMs: (this.clock += 10) };
  }

  async runtimeInfo(): Promise<RuntimeInfo> {
    return { name: 'qingmo', version: 'test', os: 'win32', arch: 'x64', nowMs: this.clock };
  }

  async pickOpenPath(): Promise<string | null> {
    return null;
  }

  async pickSavePath(): Promise<string | null> {
    return null;
  }

  async pickDirectoryPath(): Promise<string | null> {
    return this.directoryToPick;
  }

  async notify(): Promise<void> {
    /* 测试里不需要弹窗 */
  }
}

function dir(name: string, path: string): DirEntry {
  return { name, path, isDir: true, size: 0, mtimeMs: 0 };
}
function file(name: string, path: string): DirEntry {
  return { name, path, isDir: false, size: 0, mtimeMs: 0 };
}

describe('标签页', () => {
  it('打开一个文件会新建标签页并激活', async () => {
    const provider = new FakeProvider();
    provider.put('C:\\n\\a.md', 'A');
    const workspace = new WorkspaceStore(provider);

    expect(await workspace.open('C:\\n\\a.md')).toBe(true);
    const state = workspace.getSnapshot();
    expect(state.tabs).toHaveLength(1);
    expect(state.activeId).toBe(state.tabs[0]?.id);
    expect(state.tabs[0]).toMatchObject({ path: 'C:\\n\\a.md', displayName: 'a.md', dirty: false });
    expect(workspace.activeStore?.getSnapshot().doc?.text).toBe('A');
  });

  it('同一个路径只开一个标签页 —— 重复打开是激活，不是再开一个', async () => {
    // 同一个文件在两个标签页里各改一半，会让 mtime 乐观锁在保存时抛冲突，
    // 而那个冲突对用户来说完全无法理解
    const provider = new FakeProvider();
    provider.put('C:\\n\\a.md', 'A');
    provider.put('C:\\n\\b.md', 'B');
    const workspace = new WorkspaceStore(provider);

    await workspace.open('C:\\n\\a.md');
    await workspace.open('C:\\n\\b.md');
    const first = workspace.getSnapshot().tabs[0]?.id ?? -1;

    await workspace.open('C:\\n\\a.md');
    expect(workspace.getSnapshot().tabs).toHaveLength(2);
    expect(workspace.getSnapshot().activeId).toBe(first);
  });

  it('打开失败不留空标签页，错误交给外壳呈现', async () => {
    const provider = new FakeProvider();
    const workspace = new WorkspaceStore(provider);

    expect(await workspace.open('C:\\n\\missing.md')).toBe(false);
    expect(workspace.getSnapshot().tabs).toEqual([]);
    expect(workspace.getSnapshot().activeId).toBeNull();
    expect(workspace.getSnapshot().openError?.code).toBe('NOT_FOUND');

    workspace.clearOpenError();
    expect(workspace.getSnapshot().openError).toBeNull();
  });

  it('有未保存改动时 closeTab(false) 拒绝关闭，force 才关', async () => {
    const provider = new FakeProvider();
    provider.put('C:\\n\\a.md', 'A');
    const workspace = new WorkspaceStore(provider);
    await workspace.open('C:\\n\\a.md');
    workspace.activeStore?.updateText('A 改了');

    const id = workspace.getSnapshot().tabs[0]?.id ?? -1;
    expect(workspace.closeTab(id, false)).toBe(false);
    expect(workspace.getSnapshot().tabs).toHaveLength(1);

    expect(workspace.closeTab(id, true)).toBe(true);
    expect(workspace.getSnapshot().tabs).toEqual([]);
    expect(workspace.getSnapshot().activeId).toBeNull();
  });

  it('关掉活动标签页后接管右边的，没有右边的就接管左边的', async () => {
    const provider = new FakeProvider();
    for (const name of ['a', 'b', 'c']) provider.put(`C:\\n\\${name}.md`, name.toUpperCase());
    const workspace = new WorkspaceStore(provider);
    await workspace.open('C:\\n\\a.md');
    await workspace.open('C:\\n\\b.md');
    await workspace.open('C:\\n\\c.md');

    const [a, b, c] = workspace.getSnapshot().tabs.map((tab) => tab.id);
    workspace.activate(b!);
    workspace.closeTab(b!, true);
    // 关掉中间那个，应该接管右边那个
    expect(workspace.getSnapshot().activeId).toBe(c);

    workspace.closeTab(c!, true);
    expect(workspace.getSnapshot().activeId).toBe(a);
  });

  it('cycle 在两个标签页之间来回切换', async () => {
    const provider = new FakeProvider();
    provider.put('C:\\n\\a.md', 'A');
    provider.put('C:\\n\\b.md', 'B');
    const workspace = new WorkspaceStore(provider);
    await workspace.open('C:\\n\\a.md');
    await workspace.open('C:\\n\\b.md');

    const [a, b] = workspace.getSnapshot().tabs.map((tab) => tab.id);
    expect(workspace.getSnapshot().activeId).toBe(b);
    workspace.cycle(1);
    expect(workspace.getSnapshot().activeId).toBe(a);
    workspace.cycle(-1);
    expect(workspace.getSnapshot().activeId).toBe(b);
  });

  it('moveTab 重排，越界的目标下标被夹住', async () => {
    const provider = new FakeProvider();
    for (const name of ['a', 'b', 'c']) provider.put(`C:\\n\\${name}.md`, name);
    const workspace = new WorkspaceStore(provider);
    for (const name of ['a', 'b', 'c']) await workspace.open(`C:\\n\\${name}.md`);

    workspace.moveTab(0, 2);
    expect(workspace.getSnapshot().tabs.map((tab) => tab.displayName)).toEqual(['b.md', 'c.md', 'a.md']);

    workspace.moveTab(2, 99);
    expect(workspace.getSnapshot().tabs.map((tab) => tab.displayName)).toEqual(['b.md', 'c.md', 'a.md']);
  });

  it('每个标签页各有一套脏状态，互不串台', async () => {
    const provider = new FakeProvider();
    provider.put('C:\\n\\a.md', 'A');
    provider.put('C:\\n\\b.md', 'B');
    const workspace = new WorkspaceStore(provider);
    await workspace.open('C:\\n\\a.md');
    await workspace.open('C:\\n\\b.md');

    workspace.activeStore?.updateText('B 改了');
    const tabs = workspace.getSnapshot().tabs;
    expect(tabs.find((tab) => tab.displayName === 'b.md')?.dirty).toBe(true);
    expect(tabs.find((tab) => tab.displayName === 'a.md')?.dirty).toBe(false);
  });

  it('保存只写活动标签页，且各自的 mtime 基准不串台', async () => {
    const provider = new FakeProvider();
    provider.put('C:\\n\\a.md', 'A');
    provider.put('C:\\n\\b.md', 'B');
    const workspace = new WorkspaceStore(provider);
    await workspace.open('C:\\n\\a.md');
    await workspace.open('C:\\n\\b.md');

    workspace.activeStore?.updateText('B2');
    expect(await workspace.save()).toBe(true);

    expect(provider.writes).toHaveLength(1);
    expect(provider.writes[0]?.path).toBe('C:\\n\\b.md');
    // A 那边什么都没写
    expect(provider.files.get('C:\\n\\a.md')?.text).toBe('A');
  });

  it('checkExternalChanges 会检查所有标签页，不只当前那个', async () => {
    // 只查当前标签页的话，其他标签页会在用户切过去的那一刻才突然报冲突 ——
    // 而那时他已经认为「这个文件是干净的」了
    const provider = new FakeProvider();
    provider.put('C:\\n\\a.md', 'A');
    provider.put('C:\\n\\b.md', 'B');
    const workspace = new WorkspaceStore(provider);
    await workspace.open('C:\\n\\a.md');
    await workspace.open('C:\\n\\b.md');

    // 外部程序改了 A（当前活动的是 B）
    provider.put('C:\\n\\a.md', 'A 被外部改了');
    await workspace.checkExternalChanges();

    const storeA = workspace.storeOf(workspace.getSnapshot().tabs[0]?.id ?? -1);
    // 本地没有改动 → 静默跟上磁盘，不打扰用户
    expect(storeA?.getSnapshot().doc?.text).toBe('A 被外部改了');
  });

  it('内容没变化时不换快照对象 —— 否则每敲一个字整棵树都会重渲染', async () => {
    const provider = new FakeProvider();
    provider.put('C:\\n\\a.md', 'A');
    const workspace = new WorkspaceStore(provider);
    await workspace.open('C:\\n\\a.md');

    const before = workspace.getSnapshot();
    workspace.activeStore?.updateText('A');
    expect(workspace.getSnapshot()).toBe(before);

    workspace.activeStore?.updateText('A2');
    expect(workspace.getSnapshot()).not.toBe(before);
  });
});

describe('会话恢复', () => {
  function seeded(): FakeProvider {
    const provider = new FakeProvider();
    provider.put('C:\\n\\a.md', 'A 的磁盘内容');
    provider.put('C:\\n\\b.md', 'B 的磁盘内容');
    provider.put('C:\\n\\c.md', 'C 的磁盘内容');
    return provider;
  }

  it('一次把上次的标签页全开出来，只激活 activePath 指定的那一个', async () => {
    const workspace = new WorkspaceStore(seeded());
    const result = await workspace.restoreTabs(
      [
        { path: 'C:\\n\\a.md', draft: null },
        { path: 'C:\\n\\b.md', draft: null },
        { path: 'C:\\n\\c.md', draft: null },
      ],
      'C:\\n\\b.md',
    );

    expect(result).toEqual({ restored: 3, failed: [] });
    expect(workspace.getSnapshot().tabs.map((tab) => tab.path)).toEqual(['C:\\n\\a.md', 'C:\\n\\b.md', 'C:\\n\\c.md']);
    expect(workspace.activeTab?.path).toBe('C:\\n\\b.md');
  });

  it('草稿在标签页被激活之前就写进 store —— 内核一出生拿到的就是草稿，不会先渲染一遍磁盘版本', async () => {
    const workspace = new WorkspaceStore(seeded());
    await workspace.restoreTabs([{ path: 'C:\\n\\a.md', draft: 'A 的草稿' }], 'C:\\n\\a.md');

    const doc = workspace.activeStore?.getSnapshot().doc;
    expect(doc?.text).toBe('A 的草稿');
    // 磁盘那份原样留着：脏判定是「与最近一次保存的比」，保存时才知道要写什么
    expect(doc?.savedText).toBe('A 的磁盘内容');
    expect(workspace.getSnapshot().tabs[0]?.dirty).toBe(true);
  });

  it('恢复出来的草稿按 Ctrl+S 就写回磁盘 —— 与崩溃前是同一个状态', async () => {
    const provider = seeded();
    const workspace = new WorkspaceStore(provider);
    await workspace.restoreTabs([{ path: 'C:\\n\\a.md', draft: 'A 的草稿' }], 'C:\\n\\a.md');

    expect(await workspace.save()).toBe(true);
    expect(provider.files.get('C:\\n\\a.md')?.text).toBe('A 的草稿');
    expect(workspace.getSnapshot().tabs[0]?.dirty).toBe(false);
  });

  it('没有草稿的标签页恢复出来是干净的，状态栏那句话留着（它说的是磁盘读取，是准的）', async () => {
    const workspace = new WorkspaceStore(seeded());
    await workspace.restoreTabs([{ path: 'C:\\n\\a.md', draft: null }], 'C:\\n\\a.md');
    expect(workspace.getSnapshot().tabs[0]?.dirty).toBe(false);
    expect(workspace.activeStore?.getSnapshot().notice).toBe('已打开 a.md');
  });

  it('有草稿时把「已打开 X」清掉 —— 那句话说的是磁盘读取，而屏幕上现在是草稿', async () => {
    const workspace = new WorkspaceStore(seeded());
    await workspace.restoreTabs([{ path: 'C:\\n\\a.md', draft: '草稿' }], 'C:\\n\\a.md');
    expect(workspace.activeStore?.getSnapshot().notice).toBe('');
  });

  it('打不开的路径进 failed，**不写 openError**，其余标签页照常恢复', async () => {
    // 写进 openError 的话，那个字段只有一个槽位 —— 启动时失败好几个，
    // 用户只会看到最后一条，前面几个文件悄无声息地不见了
    const workspace = new WorkspaceStore(seeded());
    const result = await workspace.restoreTabs(
      [
        { path: 'C:\\n\\a.md', draft: '草稿' },
        { path: 'C:\\n\\被删了.md', draft: '草稿' },
        { path: 'C:\\n\\b.md', draft: null },
      ],
      'C:\\n\\b.md',
    );

    expect(result.restored).toBe(2);
    expect(result.failed).toEqual(['C:\\n\\被删了.md']);
    expect(workspace.getSnapshot().openError).toBeNull();
    expect(workspace.getSnapshot().tabs.map((tab) => tab.path)).toEqual(['C:\\n\\a.md', 'C:\\n\\b.md']);
  });

  it('上次激活的那个文件打不开时落到第一个 —— 不留一个「什么都没打开」的空窗口', async () => {
    const workspace = new WorkspaceStore(seeded());
    await workspace.restoreTabs(
      [
        { path: 'C:\\n\\a.md', draft: null },
        { path: 'C:\\n\\b.md', draft: null },
      ],
      'C:\\n\\幽灵.md',
    );
    expect(workspace.activeTab?.path).toBe('C:\\n\\a.md');
  });

  it('activePath 为 null 时激活第一个', async () => {
    const workspace = new WorkspaceStore(seeded());
    await workspace.restoreTabs(
      [
        { path: 'C:\\n\\b.md', draft: null },
        { path: 'C:\\n\\a.md', draft: null },
      ],
      null,
    );
    expect(workspace.activeTab?.path).toBe('C:\\n\\b.md');
  });

  it('整个恢复过程**只通知一次** —— 否则 N 个标签页会换来 N 次外壳重渲染与 N 次内核创建', async () => {
    const workspace = new WorkspaceStore(seeded());
    let notifications = 0;
    const unsubscribe = workspace.subscribe(() => {
      notifications += 1;
    });

    await workspace.restoreTabs(
      [
        { path: 'C:\\n\\a.md', draft: null },
        { path: 'C:\\n\\b.md', draft: null },
        { path: 'C:\\n\\c.md', draft: null },
      ],
      'C:\\n\\a.md',
    );
    unsubscribe();

    expect(notifications).toBe(1);
  });

  it('重复调用是幂等的（StrictMode 会把启动 effect 跑两遍）', async () => {
    const workspace = new WorkspaceStore(seeded());
    const entries = [
      { path: 'C:\\n\\a.md', draft: '草稿' },
      { path: 'C:\\n\\b.md', draft: null },
    ];

    await workspace.restoreTabs(entries, 'C:\\n\\a.md');
    const afterFirst = workspace.getSnapshot().tabs.map((tab) => tab.id);

    const second = await workspace.restoreTabs(entries, 'C:\\n\\a.md');
    expect(second).toEqual({ restored: 2, failed: [] });
    expect(workspace.getSnapshot().tabs).toHaveLength(2);
    expect(workspace.getSnapshot().tabs.map((tab) => tab.id)).toEqual(afterFirst);
    // 第二次没有再写一遍草稿：内容还是第一次那一份
    expect(workspace.activeStore?.getSnapshot().doc?.text).toBe('草稿');
  });

  it('空清单什么都不做，也不炸', async () => {
    const workspace = new WorkspaceStore(seeded());
    expect(await workspace.restoreTabs([], null)).toEqual({ restored: 0, failed: [] });
    expect(workspace.getSnapshot().tabs).toEqual([]);
    expect(workspace.getSnapshot().activeId).toBeNull();
  });

  it('已经有标签页时恢复不会把它们关掉或重排', async () => {
    const workspace = new WorkspaceStore(seeded());
    await workspace.open('C:\\n\\c.md');
    const first = workspace.getSnapshot().tabs[0]?.id;

    await workspace.restoreTabs([{ path: 'C:\\n\\a.md', draft: null }], 'C:\\n\\a.md');
    expect(workspace.getSnapshot().tabs.map((tab) => tab.path)).toEqual(['C:\\n\\c.md', 'C:\\n\\a.md']);
    expect(workspace.getSnapshot().tabs[0]?.id).toBe(first);
  });
});

describe('文件树', () => {
  function seeded(): FakeProvider {
    const provider = new FakeProvider();
    provider.dirs.set('C:\\w', [dir('src', 'C:\\w\\src'), file('a.md', 'C:\\w\\a.md')]);
    provider.dirs.set('C:\\w\\src', [file('b.ts', 'C:\\w\\src\\b.ts')]);
    return provider;
  }

  it('打开文件夹会加载根目录并展开它', async () => {
    const provider = seeded();
    const workspace = new WorkspaceStore(provider);

    expect(await workspace.openFolder('C:\\w')).toBe(true);
    const tree = workspace.getSnapshot().tree;
    expect(tree.root).toBe('C:\\w');
    expect(tree.expanded).toContain('C:\\w');
    expect(tree.dirs['C:\\w']).toHaveLength(2);
  });

  it('子目录懒加载：展开时才去读', async () => {
    const provider = seeded();
    const workspace = new WorkspaceStore(provider);
    await workspace.openFolder('C:\\w');

    // 没展开之前不该去读它
    expect(workspace.getSnapshot().tree.dirs['C:\\w\\src']).toBeUndefined();

    await workspace.toggleDir('C:\\w\\src');
    expect(workspace.getSnapshot().tree.dirs['C:\\w\\src']).toHaveLength(1);
    expect(workspace.getSnapshot().tree.expanded).toContain('C:\\w\\src');
  });

  it('折叠时把子孙目录一并收起，重新展开不会凭空冒出子行', async () => {
    const provider = seeded();
    const workspace = new WorkspaceStore(provider);
    await workspace.openFolder('C:\\w');
    await workspace.toggleDir('C:\\w\\src');
    await workspace.toggleDir('C:\\w');

    const tree = workspace.getSnapshot().tree;
    expect(tree.expanded).not.toContain('C:\\w');
    expect(tree.expanded).not.toContain('C:\\w\\src');
  });

  it('目录读不出来时记下错误，不抛出去', async () => {
    const provider = seeded();
    const workspace = new WorkspaceStore(provider);
    await workspace.openFolder('C:\\w');

    provider.failListDir = true;
    await workspace.toggleDir('C:\\w\\src');
    expect(workspace.getSnapshot().tree.error?.message).toContain('读不出目录');
    expect(workspace.getSnapshot().tree.loading).toEqual([]);
  });

  it('打开文件时自动展开它的祖先链 —— 否则侧栏和当前文档看起来毫无关系', async () => {
    const provider = seeded();
    provider.put('C:\\w\\src\\b.ts', 'let a = 1;');
    const workspace = new WorkspaceStore(provider);
    await workspace.openFolder('C:\\w');

    await workspace.open('C:\\w\\src\\b.ts');
    expect(workspace.getSnapshot().tree.expanded).toContain('C:\\w\\src');
  });

  it('refreshTree 重新读取所有已加载的目录', async () => {
    const provider = seeded();
    const workspace = new WorkspaceStore(provider);
    await workspace.openFolder('C:\\w');

    provider.dirs.set('C:\\w', [file('new.md', 'C:\\w\\new.md')]);
    await workspace.refreshTree();
    expect(workspace.getSnapshot().tree.dirs['C:\\w']?.map((item) => item.name)).toEqual(['new.md']);
  });

  it('没打开文件夹时不猜路径', async () => {
    const provider = seeded();
    const workspace = new WorkspaceStore(provider);
    await workspace.revealInTree('C:\\w\\a.md');
    expect(workspace.getSnapshot().tree.expanded).toEqual([]);
  });
});

describe('文件监听推送（P3 第 14 项：handleFsChanges）', () => {
  /** handleFsChanges 内部是 void Promise，给它一拍落地。 */
  function flush(): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, 0));
  }

  it('命中的打开文档会被点名重查 —— 干净文档静默跟上磁盘', async () => {
    const provider = new FakeProvider();
    provider.put('C:\\n\\a.md', '旧版');
    const workspace = new WorkspaceStore(provider);
    await workspace.open('C:\\n\\a.md');

    // 外部程序改写了文件
    provider.put('C:\\n\\a.md', '外部改动');
    workspace.handleFsChanges(['C:\\n\\a.md', 'C:\\n\\别的.md']);
    await flush();

    const store = workspace.storeOf(workspace.getSnapshot().tabs[0]?.id ?? -1);
    expect(store?.getSnapshot().doc?.text).toBe('外部改动');
  });

  it('不相关的路径不会惊动文档', async () => {
    const provider = new FakeProvider();
    provider.put('C:\\n\\a.md', 'A');
    const workspace = new WorkspaceStore(provider);
    await workspace.open('C:\\n\\a.md');

    workspace.handleFsChanges(['C:\\其他目录\\x.md']);
    await flush();

    expect(workspace.activeStore?.getSnapshot().doc?.text).toBe('A');
    expect(workspace.activeStore?.getSnapshot().status).toBe('clean');
  });

  it('空批次无事发生', async () => {
    const provider = new FakeProvider();
    provider.put('C:\\n\\a.md', 'A');
    const workspace = new WorkspaceStore(provider);
    await workspace.open('C:\\n\\a.md');

    workspace.handleFsChanges([]);
    await flush();

    expect(workspace.activeStore?.getSnapshot().doc?.text).toBe('A');
  });

  it('已加载的目录会被重拉，把外部新建的文件刷进树里', async () => {
    const provider = new FakeProvider();
    provider.dirs.set('C:\\w', [file('a.md', 'C:\\w\\a.md')]);
    const workspace = new WorkspaceStore(provider);
    await workspace.openFolder('C:\\w');
    expect(workspace.getSnapshot().tree.dirs['C:\\w']).toHaveLength(1);

    // 外部新建了一个文件
    provider.dirs.set('C:\\w', [file('a.md', 'C:\\w\\a.md'), file('b.md', 'C:\\w\\b.md')]);
    workspace.handleFsChanges(['C:\\w\\b.md']);
    await flush();

    expect(workspace.getSnapshot().tree.dirs['C:\\w']).toHaveLength(2);
  });

  it('没加载过的目录不拉 —— 界面上没有的层级不值得一次 IO', async () => {
    const provider = new FakeProvider();
    provider.dirs.set('C:\\w', []);
    const workspace = new WorkspaceStore(provider);
    await workspace.openFolder('C:\\w');

    workspace.handleFsChanges(['C:\\w\\深层\\新文件.md']);
    await flush();

    expect(workspace.getSnapshot().tree.dirs['C:\\w\\深层']).toBeUndefined();
    // 父目录不在已加载集合里，根目录也不会被误刷
    expect(workspace.getSnapshot().tree.dirs['C:\\w']).toEqual([]);
  });

  it('变更路径本身是已加载目录时重拉那一层（目录被删 / 改名）', async () => {
    const provider = new FakeProvider();
    provider.dirs.set('C:\\w', [dir('src', 'C:\\w\\src')]);
    provider.dirs.set('C:\\w\\src', [file('b.ts', 'C:\\w\\src\\b.ts')]);
    const workspace = new WorkspaceStore(provider);
    await workspace.openFolder('C:\\w');
    await workspace.toggleDir('C:\\w\\src');
    expect(workspace.getSnapshot().tree.dirs['C:\\w\\src']).toHaveLength(1);

    // src 整个被删了：重拉会失败，树要把失败说出来（硬约束 2），不是静默装没事
    provider.failListDir = true;
    workspace.handleFsChanges(['C:\\w\\src']);
    await flush();

    expect(workspace.getSnapshot().tree.error?.message).toContain('读不出目录');
  });
});

describe('dispose', () => {
  it('dispose 之后子 store 的变化不会再惊动工作区', async () => {
    const provider = new FakeProvider();
    provider.put('C:\\n\\a.md', 'A');
    const workspace = new WorkspaceStore(provider);
    await workspace.open('C:\\n\\a.md');

    // 先留住子 store 的引用，才能验证「订阅确实被断开了」
    const child = workspace.activeStore;
    let notified = 0;
    workspace.subscribe(() => {
      notified += 1;
    });

    workspace.dispose();
    expect(workspace.getSnapshot().tabs).toEqual([]);

    child?.updateText('改了');
    expect(notified).toBe(0);
  });
});
