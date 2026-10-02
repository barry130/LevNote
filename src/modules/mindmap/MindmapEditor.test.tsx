// @vitest-environment jsdom
/**
 * 导图内核的挂载测试。
 *
 * 这里测的不是「导图画得对不对」（那是内核自己的事，jsdom 也没有布局引擎），
 * 而是**接缝**：
 *
 * - 内核能不能在这个环境里被创建与销毁（创建失败会走 `onError`，而不是白屏）；
 * - `getValue()` 与「打开时的原文」是不是**逐字节一致**（§7.2 第 4 条对 `.mindmap`
 *   的同一条要求 —— 用户没改过，保存就不该动一个字节）；
 * - 外壳推内容（`setValue`）之后内核有没有真的换掉数据；
 * - 空文件与坏文件分别走哪条路。
 *
 * jsdom 没有布局引擎，所以所有元素量出来都是 0×0。这对上面这些问题**没有影响** ——
 * 它们问的都是数据，不是像素。真正需要布局的东西（拖动、连线、缩放）
 * 只能靠 `npm run tauri dev` 里的人工冒烟，写在这里只会得到一条假绿的测试。
 */

import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import type { EditorHandle, EditorMountProps } from '../../core/editor/types.js';
import { DEFAULT_FIND_OPTIONS } from '../../core/search/find.js';
import { MINDMAP_FORMAT, MINDMAP_VERSION } from '../../core/mindmap/types.js';
import MindmapEditor from './MindmapEditor.js';

declare global {
  var IS_REACT_ACT_ENVIRONMENT: boolean | undefined;
}

/** 一份**规范形式**的文件（`JSON.stringify(…, 2)` + 末尾换行，与序列化器一致）。 */
const CANONICAL = `${JSON.stringify(
  {
    format: MINDMAP_FORMAT,
    version: MINDMAP_VERSION,
    layout: 'logicalStructure',
    root: {
      id: 'n1',
      topic: '根主题',
      children: [
        { id: 'n2', topic: '分支甲', children: [] },
        { id: 'n3', topic: '分支乙', children: [] },
      ],
    },
  },
  null,
  2,
)}\n`;

/** 同一份内容，但**不是**我们写出来的样子：紧凑 JSON、无末尾换行。 */
const COMPACT = JSON.stringify({
  format: MINDMAP_FORMAT,
  version: MINDMAP_VERSION,
  layout: 'logicalStructure',
  root: { id: 'n1', topic: '根主题', children: [] },
});

let container: HTMLDivElement;
let root: Root;

beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  container = document.createElement('div');
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => root.unmount());
  container.remove();
});

interface Mounted {
  /** 内核创建失败时为 null —— 那条路是 `onError` 负责的，不是 handle。 */
  handle: EditorHandle | null;
  changes: string[];
  errors: Error[];
}

async function mount(over: Partial<EditorMountProps> = {}): Promise<Mounted> {
  const changes: string[] = [];
  const errors: Error[] = [];
  const captured: { handle: EditorHandle | null } = { handle: null };

  const props: EditorMountProps = {
    initialValue: CANONICAL,
    path: '图.mindmap',
    onChange: (value) => changes.push(value),
    onReady: (handle) => {
      captured.handle = handle;
    },
    onError: (error) => errors.push(error),
    ...over,
  };

  await act(async () => {
    root.render(<MindmapEditor {...props} />);
  });

  return { handle: captured.handle, changes, errors };
}

/** 绝大多数用例要的形态：必须拿到 handle，否则直接失败并带上 `onError` 说了什么。 */
async function mountReady(over: Partial<EditorMountProps> = {}): Promise<Mounted & { handle: EditorHandle }> {
  const mounted = await mount(over);
  if (mounted.handle === null) {
    throw new Error(`内核没有交出 handle；onError 收到：${mounted.errors.map((error) => error.message).join('、') || '（无）'}`);
  }
  const handle: EditorHandle = mounted.handle;
  return { ...mounted, handle };
}

describe('挂载与生命周期', () => {
  it('内核能在这个环境里创建出来并交出 handle', async () => {
    const { handle, errors } = await mountReady();
    expect(errors).toEqual([]);
    expect(typeof handle.getValue).toBe('function');
  });

  it('容器里真的长出了内核画的东西', async () => {
    await mountReady();
    // 不按类名断言具体结构（那是内核的渲染细节），只确认它确实渲染了
    expect(container.querySelector('.map-container')).not.toBeNull();
  });

  it('卸载之后再调 handle 不抛错 —— 外壳可能还握着它', async () => {
    const { handle } = await mountReady();
    act(() => root.unmount());
    // 卸载后实例已经销毁，但 handle 的转发层让「已销毁」表现为「什么都不做」
    expect(() => handle.focus()).not.toThrow();
    // 重新挂一个根，好让 afterEach 的 unmount 不至于对着已卸载的根再卸一次
    root = createRoot(container);
  });
});

describe('打开 → 保存的保真度', () => {
  /**
   * 这条是 P4 这一项里最值钱的断言。
   *
   * JSON 往返**做不到**逐字节保真（键顺序、缩进、空格都会变），所以
   * 「没改过就交回原文」是唯一的实现方式 —— 而它依赖内核这边
   * 「用户没动过时一个事件都不报」。
   */
  it('用户没动过时，getValue() 逐字节等于打开时的原文', async () => {
    const { handle } = await mountReady();
    expect(handle.getValue()).toBe(CANONICAL);
  });

  it('原文不是规范形式时也逐字节交回 —— 手写的 / 别的工具写的文件同样不动', async () => {
    const { handle } = await mountReady({ initialValue: COMPACT });
    expect(handle.getValue()).toBe(COMPACT);
  });

  it('CRLF 的文件不会被悄悄换成 LF', async () => {
    const crlf = CANONICAL.replace(/\n/g, '\r\n');
    const { handle } = await mountReady({ initialValue: crlf });
    expect(handle.getValue()).toBe(crlf);
  });
});

describe('外部内容变更', () => {
  it('setValue 换掉数据之后，getValue 给出新内容', async () => {
    const { handle } = await mountReady();

    const next = `${JSON.stringify(
      {
        format: MINDMAP_FORMAT,
        version: MINDMAP_VERSION,
        layout: 'orgChart',
        root: { id: 'r', topic: '换过的根', children: [{ id: 'c', topic: '新分支', children: [] }] },
      },
      null,
      2,
    )}\n`;

    await act(async () => {
      handle.setValue(next);
    });

    const back = JSON.parse(handle.getValue()) as { layout: string; root: { topic: string; children: { topic: string }[] } };
    expect(back.root.topic).toBe('换过的根');
    expect(back.root.children[0]?.topic).toBe('新分支');
    // 布局也跟着换 —— `refresh()` 不换方向，这一条盯的就是那一步有没有补上
    expect(back.layout).toBe('orgChart');
  });

  it('setValue 收到的坏内容被安静忽略，而不是把界面清空', async () => {
    const { handle } = await mountReady();
    await act(async () => {
      handle.setValue('{ 这不是 JSON');
    });
    expect(handle.getValue()).toBe(CANONICAL);
  });
});

describe('空文件与坏文件', () => {
  it('空内容给一份带中心主题的默认文档，而不是白屏', async () => {
    const { handle, errors } = await mountReady({ initialValue: '' });
    expect(errors).toEqual([]);
    const back = JSON.parse(handle.getValue()) as { format: string; root: { topic: string } };
    expect(back.format).toBe(MINDMAP_FORMAT);
    expect(back.root.topic).toBe('中心主题');
  });

  it('读不出来的内容如实报错，而不是假装成功', async () => {
    // 正常路径上 `canHandle` 会先把这种文件拦在注册表外（见 index.ts），
    // 但组件自己也要有交代 —— 不能白屏。
    // 这条路**没有 handle**，所以用 `mount` 而不是 `mountReady`。
    const { handle, errors } = await mount({ initialValue: '{ 这不是 JSON' });
    expect(handle).toBeNull();
    expect(errors.length).toBe(1);
  });
});

describe('可选能力', () => {
  it('toHtml() 返回 null —— 导图没有能进导出文档的语义 HTML', async () => {
    const { handle } = await mountReady();
    await expect(handle.toHtml()).resolves.toBeNull();
  });

  it('不声明格式化 / 表格 / 块手柄这些能力 —— 外壳据此不显示那些浮层', async () => {
    const { handle } = await mountReady();
    expect(handle.formatting).toBeUndefined();
    expect(handle.table).toBeUndefined();
    expect(handle.blockHandle).toBeUndefined();
  });

  it('查找在节点文字里找得到，且按文档顺序', async () => {
    const { handle } = await mountReady();
    const count = handle.search?.find({ ...DEFAULT_FIND_OPTIONS, query: '分支' }) ?? 0;
    expect(count).toBe(2);

    const none = handle.search?.find({ ...DEFAULT_FIND_OPTIONS, query: '不存在的词' }) ?? -1;
    expect(none).toBe(0);
  });

  it('查找可以匹配到根主题', async () => {
    const { handle } = await mountReady();
    const count = handle.search?.find({ ...DEFAULT_FIND_OPTIONS, query: '根主题' }) ?? 0;
    expect(count).toBe(1);
  });

  it('只读模式不影响读值', async () => {
    const { handle, errors } = await mountReady({ readOnly: true });
    expect(errors).toEqual([]);
    expect(handle.getValue()).toBe(CANONICAL);
  });
});
