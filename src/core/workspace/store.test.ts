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
