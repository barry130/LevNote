import { beforeEach, describe, expect, it } from 'vitest';

import { DocumentStore, isDirty } from './store.js';
import type {
  BinaryWriteOutcome,
  BomKind,
  DirEntry,
  DocumentProvider,
  FileStat,
  LineEndingName,
  RuntimeInfo,
  TextFile,
  WriteOutcome,
  WriteRequest,
} from './types.js';

function lineEndingOf(text: string): LineEndingName {
  const crlf = (text.match(/\r\n/g) ?? []).length;
  const loneLf = (text.match(/(?<!\r)\n/g) ?? []).length;
  if (crlf > 0 && loneLf > 0) return 'mixed';
  if (crlf > 0) return 'crlf';
  return 'lf';
}

interface FakeFile {
  text: string;
  encoding: string;
  bom: BomKind;
  mtimeMs: number;
  readonly: boolean;
}

/** 内存文件系统。行为刻意贴近 Rust 侧：mtime 不符就拒绝写入。 */
class FakeProvider implements DocumentProvider {
  readonly kind = 'desktop' as const;
  readonly files = new Map<string, FakeFile>();
  /** 二进制写入落下来的字节。这个套件不测它，但接口要求盘上有地方放。 */
  readonly binaries = new Map<string, ArrayBuffer>();
  writes: WriteRequest[] = [];
  private clock = 1_000;

  put(path: string, text: string, options: Partial<FakeFile> = {}): void {
    this.files.set(path, {
      text,
      encoding: options.encoding ?? 'UTF-8',
      bom: options.bom ?? 'none',
      mtimeMs: options.mtimeMs ?? this.tick(),
      readonly: options.readonly ?? false,
    });
  }

  /** 模拟外部程序改文件：内容变、mtime 也变。 */
  touchExternally(path: string, text: string): void {
    const file = this.files.get(path);
    if (!file) throw new Error(`没有这个文件：${path}`);
    this.files.set(path, { ...file, text, mtimeMs: this.tick() });
  }

  private tick(): number {
    this.clock += 10;
    return this.clock;
  }

  /**
   * 二进制读取。测试里只用来验证「导出内联图片」这条路径不炸，
   * 不真读盘：按路径生成几个可辨认的字节。
   */
  async readBinaryFile(path: string): Promise<ArrayBuffer> {
    const file = this.files.get(path);
    if (!file) throw { code: 'NOT_FOUND', message: `没有这个文件：${path}`, detail: null };
    return new TextEncoder().encode(file.text).buffer;
  }

  /** 这个假盘不测图片落盘，但接口要求它存在。语义与桌面版对齐：已存在就拒绝。 */
  async writeBinaryFile(path: string, bytes: ArrayBuffer): Promise<BinaryWriteOutcome> {
    if (this.binaries.has(path)) throw { code: 'CONFLICT', message: `已存在：${path}`, detail: null };
    this.binaries.set(path, bytes.slice(0));
    return { path, size: bytes.byteLength, mtimeMs: this.tick() };
  }

  async readTextFile(path: string): Promise<TextFile> {
    const file = this.files.get(path);
    if (!file) throw { code: 'NOT_FOUND', message: `没有这个文件：${path}`, detail: null };
    return {
      path,
      text: file.text,
      encoding: file.encoding,
      bom: file.bom,
      lineEnding: lineEndingOf(file.text),
      size: file.text.length,
      mtimeMs: file.mtimeMs,
      readonly: file.readonly,
      lossy: false,
    };
  }

  async writeTextFile(request: WriteRequest): Promise<WriteOutcome> {
    this.writes.push(request);
    const existing = this.files.get(request.path);

    if (request.expectedMtimeMs !== undefined && existing && existing.mtimeMs !== request.expectedMtimeMs) {
      throw {
        code: 'CONFLICT',
        message: '文件已被外部修改，为避免覆盖他人的改动，本次保存已取消',
        detail: null,
      };
    }

    let text = request.text;
    if (request.lineEnding) {
      const unified = text.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
      text = request.lineEnding === 'crlf' ? unified.replace(/\n/g, '\r\n') : unified;
    }

    const mtimeMs = this.tick();
    this.files.set(request.path, {
      text,
      encoding: request.encoding ?? existing?.encoding ?? 'UTF-8',
      bom: request.bom ?? existing?.bom ?? 'none',
      mtimeMs,
      readonly: existing?.readonly ?? false,
    });

    return {
      path: request.path,
      size: text.length,
      mtimeMs,
      encoding: request.encoding ?? 'UTF-8',
      bom: request.bom ?? 'none',
      lineEnding: lineEndingOf(text),
    };
  }

  async statFile(path: string): Promise<FileStat> {
    const file = this.files.get(path);
    if (!file) throw { code: 'NOT_FOUND', message: '没有这个文件', detail: null };
    return { path, size: file.text.length, mtimeMs: file.mtimeMs, readonly: file.readonly, isDir: false };
  }

  async listDir(): Promise<DirEntry[]> {
    return [];
  }

  async sniffEncoding(): Promise<string> {
    return 'UTF-8';
  }

  async runtimeInfo(): Promise<RuntimeInfo> {
    return { name: 'qingmo', version: '0.0.0-test', os: 'test', arch: 'test', nowMs: 0 };
  }

  async pickOpenPath(): Promise<string | null> {
    return null;
  }

  async pickSavePath(suggestedName: string): Promise<string | null> {
    return `/tmp/${suggestedName}`;
  }

  async pickDirectoryPath(): Promise<string | null> {
    return null;
  }

  async notify(): Promise<void> {
    /* 测试里不需要弹窗 */
  }
}

let provider: FakeProvider;
let store: DocumentStore;

beforeEach(() => {
  provider = new FakeProvider();
  store = new DocumentStore(provider);
});

describe('打开与脏判定', () => {
  it('打开后是干净的，改一个字就变脏，改回来又变干净', async () => {
    provider.put('/p/a.md', 'hello');

    await store.open('/p/a.md');
    expect(store.getSnapshot().status).toBe('clean');
    expect(store.getSnapshot().doc?.displayName).toBe('a.md');

    store.updateText('hello!');
    expect(store.getSnapshot().status).toBe('dirty');

    store.updateText('hello');
    expect(store.getSnapshot().status).toBe('clean');
  });

  it('行尾被改过也算脏 —— 它确实会改变磁盘上的字节', async () => {
    provider.put('/p/a.md', 'a\nb\n');
    await store.open('/p/a.md');

    store.setLineEnding('crlf');
    expect(store.getSnapshot().status).toBe('dirty');

    store.setLineEnding('lf');
    expect(store.getSnapshot().status).toBe('clean');
  });

  it('打开失败时保留错误信息且不留下半截文档', async () => {
    const ok = await store.open('/p/missing.md');
    expect(ok).toBe(false);
    expect(store.getSnapshot().doc).toBeNull();
    expect(store.getSnapshot().error?.code).toBe('NOT_FOUND');
  });

  it('有未保存改动时拒绝直接打开新文件', async () => {
    provider.put('/p/a.md', 'x');
    provider.put('/p/b.md', 'y');
    await store.open('/p/a.md');
    store.updateText('changed');

    expect(await store.open('/p/b.md')).toBe(false);
    expect(store.getSnapshot().doc?.path).toBe('/p/a.md');
  });
});

describe('保存', () => {
  it('保存时带上 mtime 作为乐观锁，成功后 mtime 与脏状态一起更新', async () => {
    provider.put('/p/a.md', 'hello');
    await store.open('/p/a.md');
    const before = store.getSnapshot().doc!.mtimeMs;

    store.updateText('hello world');
    expect(await store.save()).toBe(true);

    expect(provider.writes).toHaveLength(1);
    expect(provider.writes[0]!.expectedMtimeMs).toBe(before);

    const after = store.getSnapshot();
    expect(after.status).toBe('clean');
    expect(after.doc!.mtimeMs).not.toBe(before);
    expect(provider.files.get('/p/a.md')!.text).toBe('hello world');
  });

  it('把行尾风格一并交给写入层，而不是让磁盘上出现混合行尾', async () => {
    provider.put('/p/a.md', 'a\r\nb\r\n');
    await store.open('/p/a.md');
    expect(store.getSnapshot().doc!.lineEnding).toBe('crlf');

    store.updateText('a\r\nb\r\nc\r\n');
    await store.save();

    expect(provider.writes[0]!.lineEnding).toBe('crlf');
  });

  it('混排行尾不传目标值，避免一次保存擅自统一掉整篇文件', async () => {
    provider.put('/p/a.md', 'a\r\nb\nc');
    await store.open('/p/a.md');
    expect(store.getSnapshot().doc!.lineEnding).toBe('mixed');

    store.updateText('a\r\nb\nc\n');
    await store.save();

    expect(provider.writes[0]!.lineEnding).toBeUndefined();
  });

  it('只读文件直接报错，不尝试写入', async () => {
    provider.put('/p/a.md', 'x', { readonly: true });
    await store.open('/p/a.md');
    store.updateText('y');

    expect(await store.save()).toBe(false);
    expect(provider.writes).toHaveLength(0);
    expect(store.getSnapshot().error?.code).toBe('PERMISSION_DENIED');
  });
});

describe('冲突', () => {
  it('磁盘被外部改写后保存会失败，并把状态切到 conflict 而不是抛错', async () => {
    provider.put('/p/a.md', 'v1');
    await store.open('/p/a.md');
    store.updateText('我的改动');

    provider.touchExternally('/p/a.md', '别人的改动');

    expect(await store.save()).toBe(false);
    const snapshot = store.getSnapshot();
    expect(snapshot.status).toBe('conflict');
    expect(snapshot.conflict).not.toBeNull();
    // 关键：磁盘内容没有被覆盖
    expect(provider.files.get('/p/a.md')!.text).toBe('别人的改动');
  });

  it('选「用磁盘版本覆盖我的」会丢掉本地改动', async () => {
    provider.put('/p/a.md', 'v1');
    await store.open('/p/a.md');
    store.updateText('我的改动');
    provider.touchExternally('/p/a.md', '别人的改动');
    await store.save();

    expect(await store.resolveConflict('reload')).toBe(true);
    const snapshot = store.getSnapshot();
    expect(snapshot.doc!.text).toBe('别人的改动');
    expect(snapshot.status).toBe('clean');
    expect(snapshot.conflict).toBeNull();
  });

  it('选「用我的覆盖磁盘」会写入且不再带 mtime 锁', async () => {
    provider.put('/p/a.md', 'v1');
    await store.open('/p/a.md');
    store.updateText('我的改动');
    provider.touchExternally('/p/a.md', '别人的改动');
    await store.save();

    expect(await store.resolveConflict('overwrite')).toBe(true);
    const last = provider.writes[provider.writes.length - 1]!;
    expect(last.expectedMtimeMs).toBeUndefined();
    expect(provider.files.get('/p/a.md')!.text).toBe('我的改动');
    expect(store.getSnapshot().status).toBe('clean');
  });

  it('选「另存为副本」会把当前内容写到新路径', async () => {
    provider.put('/p/a.md', 'v1');
    await store.open('/p/a.md');
    store.updateText('我的改动');
    provider.touchExternally('/p/a.md', '别人的改动');
    await store.save();

    expect(await store.resolveConflict('save-copy')).toBe(true);
    const snapshot = store.getSnapshot();
    expect(snapshot.doc!.path).toBe('/tmp/a.md');
    expect(snapshot.conflict).toBeNull();
    // 原文件仍然保留别人的改动
    expect(provider.files.get('/p/a.md')!.text).toBe('别人的改动');
  });
});

describe('外部改动检测（P1 用窗口焦点兜住，P3 换成文件监听）', () => {
  it('本地没改动时静默跟上磁盘', async () => {
    provider.put('/p/a.md', 'v1');
    await store.open('/p/a.md');

    provider.touchExternally('/p/a.md', '新版');
    await store.checkExternalChange();

    expect(store.getSnapshot().doc!.text).toBe('新版');
    expect(store.getSnapshot().status).toBe('clean');
    expect(store.getSnapshot().conflict).toBeNull();
  });

  it('本地也有改动时切到冲突，不擅自选一边', async () => {
    provider.put('/p/a.md', 'v1');
    await store.open('/p/a.md');
    store.updateText('我的改动');

    provider.touchExternally('/p/a.md', '新版');
    await store.checkExternalChange();

    const snapshot = store.getSnapshot();
    expect(snapshot.status).toBe('conflict');
    expect(snapshot.doc!.text).toBe('我的改动');
  });

  it('mtime 没变时什么都不做', async () => {
    provider.put('/p/a.md', 'v1');
    await store.open('/p/a.md');
    const before = store.getSnapshot();

    await store.checkExternalChange();

    const after = store.getSnapshot();
    // 不该重新加载、不该报冲突、不该产生多余状态变化
    expect(after.doc).toBe(before.doc);
    expect(after.status).toBe('clean');
    expect(after.conflict).toBeNull();
    expect(after.error).toBeNull();
  });
});

describe('关闭', () => {
  it('有脏改动时 close(false) 拒绝关闭，交回给 UI 去确认', async () => {
    provider.put('/p/a.md', 'x');
    await store.open('/p/a.md');
    store.updateText('y');

    expect(store.close(false)).toBe(false);
    expect(store.getSnapshot().doc).not.toBeNull();

    expect(store.close(true)).toBe(true);
    expect(store.getSnapshot().doc).toBeNull();
  });
});

describe('视图模式（富文本 / 源码 / 分屏）', () => {
  /**
   * 这一组守的核心不变量：**视图模式是「这次会话怎么看待这份文档」，
   * 不是文档本身的属性**。所以它必须
   * - 不影响 `isDirty`（切一下视图不该让文件变成「待保存」）；
   * - 不被 `reload()` / `saveAs()` 这类只碰文档对象的操作重置。
   * 后一条是最容易写错的：`reload()` 会重建文档对象，视图模式若住在里面就会被悄悄清掉。
   */

  it('新打开的文档默认在富文本视图', async () => {
    provider.put('/p/a.md', '# 标题\n');
    await store.open('/p/a.md');

    expect(store.getSnapshot().viewMode).toBe('rich');
  });

  it('切换视图不改内容，也不算脏', async () => {
    provider.put('/p/a.md', '# 标题\n');
    await store.open('/p/a.md');

    store.setViewMode('source');

    const snapshot = store.getSnapshot();
    expect(snapshot.viewMode).toBe('source');
    expect(snapshot.doc?.text).toBe('# 标题\n');
    expect(snapshot.status).toBe('clean');
  });

  /**
   * 分屏与源码**共用同一份 `doc.text`**，所以「进分屏」不该动内容、
   * 更不该把它变成待保存。这条要是坏了，用户切一下视图就会看到
   * 「有未保存的改动」，然后开始怀疑自己是不是改坏了什么。
   */
  it('进分屏同样不改内容、不算脏，且三档之间来回切是无损的', async () => {
    provider.put('/p/a.md', '# 标题\n\n正文 **加粗**。\n');
    await store.open('/p/a.md');

    store.setViewMode('split');
    expect(store.getSnapshot().doc?.text).toBe('# 标题\n\n正文 **加粗**。\n');
    expect(store.getSnapshot().status).toBe('clean');

    store.setViewMode('source');
    store.setViewMode('rich');
    store.setViewMode('split');

    // 三档共用一份权威副本 —— 来回切是**无损**的，不是三次序列化往返
    expect(store.getSnapshot().viewMode).toBe('split');
    expect(store.getSnapshot().doc?.text).toBe('# 标题\n\n正文 **加粗**。\n');
    expect(store.getSnapshot().status).toBe('clean');
  });

  it('reload 之后仍然停在源码视图 —— 视图模式不属于文档对象', async () => {
    provider.put('/p/a.md', '# 标题\n');
    await store.open('/p/a.md');
    store.setViewMode('source');

    provider.touchExternally('/p/a.md', '# 外部改过的标题\n');
    await store.reload();

    expect(store.getSnapshot().viewMode).toBe('source');
    expect(store.getSnapshot().doc?.text).toBe('# 外部改过的标题\n');
  });

  it('另存为之后也保留视图模式', async () => {
    provider.put('/p/a.md', '# 标题\n');
    await store.open('/p/a.md');
    store.setViewMode('source');

    await store.saveAs();

    expect(store.getSnapshot().viewMode).toBe('source');
    expect(store.getSnapshot().doc?.displayName).toBe('a.md');
  });

  it('换一份文件时回到富文本视图', async () => {
    provider.put('/p/a.md', '甲\n');
    provider.put('/p/b.md', '乙\n');
    await store.open('/p/a.md');
    store.setViewMode('source');

    await store.open('/p/b.md');

    // 「打开一份文档」的默认期望是看到渲染后的样子。
    // 沿用上一份文件的视图模式会让用户以为打开失败（看到的是一堆 Markdown 源码）。
    expect(store.getSnapshot().viewMode).toBe('rich');
  });

  it('没有文档时静默忽略 —— 切视图是纯视图动作，不该在空标签页上报错', () => {
    store.setViewMode('source');

    expect(store.getSnapshot().viewMode).toBe('rich');
    expect(store.getSnapshot().error).toBeNull();
  });

  it('设成同一个值时不通知订阅者（快照引用稳定）', async () => {
    provider.put('/p/a.md', '甲\n');
    await store.open('/p/a.md');

    let notifications = 0;
    const unsubscribe = store.subscribe(() => {
      notifications += 1;
    });
    store.setViewMode('rich');
    expect(notifications).toBe(0);

    store.setViewMode('source');
    expect(notifications).toBe(1);
    unsubscribe();
  });
});

describe('isDirty', () => {
  it('只看内容与行尾，不看编码（编码是写出时才生效的属性）', () => {
    const base = {
      path: '/p/a.md',
      displayName: 'a.md',
      text: 'x',
      savedText: 'x',
      encoding: 'UTF-8',
      bom: 'none' as const,
      lineEnding: 'lf' as const,
      savedLineEnding: 'lf' as const,
      session: 1,
      mtimeMs: 1,
      readonly: false,
      lossy: false,
    };
    expect(isDirty(base)).toBe(false);
    expect(isDirty({ ...base, text: 'y' })).toBe(true);
    expect(isDirty({ ...base, lineEnding: 'crlf' })).toBe(true);
    expect(isDirty({ ...base, encoding: 'GBK' })).toBe(false);
  });
});

describe('resolveDocBlock（P5 第 25 项：doc-block 引用的图表文件）', () => {
  it('src 按当前文档的目录解析，读出引用文件原文', async () => {
    provider.put('/p/a.md', '文档');
    provider.put('/p/figs/x.drawio', '<mxfile><diagram/></mxfile>');
    await store.open('/p/a.md');

    expect(await store.resolveDocBlock('figs/x.drawio')).toBe('<mxfile><diagram/></mxfile>');
  });

  it('src 里的分隔符统一成文档路径的风格（硬约束 12：写入路径与回读路径必须是同一个字符串）', async () => {
    provider.put('/p/a.md', '文档');
    provider.put('/p/figs/x.drawio', '<mxfile/>');
    await store.open('/p/a.md');

    expect(await store.resolveDocBlock('figs\\x.drawio')).toBe('<mxfile/>');
  });

  it('读不到就返回 null，不抛 —— 调用链尽头是 NodeView 的一次渲染', async () => {
    provider.put('/p/a.md', '文档');
    await store.open('/p/a.md');

    await expect(store.resolveDocBlock('missing.drawio')).resolves.toBeNull();
  });

  it('没有打开的文档时返回 null', async () => {
    await expect(store.resolveDocBlock('x.drawio')).resolves.toBeNull();
  });
});
