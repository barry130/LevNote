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
import { setLocale } from '../../core/i18n/index.js';
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
  // 顺序有讲究：先卸内核再改语言。反过来的话 `setLocale` 会先把内核重建一遍、
  // 紧接着又被卸载，白白建一个出来（还让 act 的时序变得难读）。
  act(() => root.unmount());
  container.remove();
  setLocale('zh-CN');
});

interface Mounted {
  /** 内核创建失败时为 null —— 那条路是 `onError` 负责的，不是 handle。 */
  handle: EditorHandle | null;
  changes: string[];
  errors: Error[];
  /**
   * 此刻内核交出的那个 handle。
   *
   * 不能只靠 `mounted.handle`：切语言会重建内核并**再交一次** `onReady`，
   * 于是挂载时拿到的那个句柄已经指向一个销毁了的实例。
   */
  currentHandle: () => EditorHandle | null;
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

  return { handle: captured.handle, changes, errors, currentHandle: () => captured.handle };
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

/**
 * 内核**自己**画的那些字：右键菜单十一条 + 新节点的占位文字。
 *
 * 单独一个 describe 的理由：这一族要么整体成立要么整体不成立（内核在安装那一刻
 * 一次建好十一个 `<li>`，少一条就是「半中半英」），拆开测没有意义。
 */
describe('内核自己画的界面（ADR 0049）', () => {
  /** 右键菜单的文字。菜单是 `hidden` 的，但文字照样在 DOM 里，测的就是它。 */
  const menuText = (): string => container.querySelector('.context-menu')?.textContent ?? '';

  /** 用户改过的一份文件（规范形式，所以「改过的根主题」能原样断言）。 */
  const EDITED = `${JSON.stringify(
    {
      format: MINDMAP_FORMAT,
      version: MINDMAP_VERSION,
      layout: 'logicalStructure',
      root: { id: 'n1', topic: '改过的根主题', children: [] },
    },
    null,
    2,
  )}\n`;

  it('右键菜单是中文的 —— 不是内核硬编码的那份英文', async () => {
    await mountReady();
    const menu = menuText();
    expect(menu).toContain('插入子主题');
    expect(menu).toContain('删除主题');
    expect(menu).toContain('专注模式');
    // 连「连接」那两条都在（传的是对象，`focus` / `link` 两组都要显式打开）
    expect(menu).toContain('双向连接');
    expect(menu).not.toContain('Add child');
  });

  it('切语言之后菜单当场变英文 —— 不必关掉重开文件', async () => {
    await mountReady();
    const before = menuText();
    await act(async () => {
      setLocale('en');
    });
    const after = menuText();
    expect(after).not.toBe(before);
    expect(after).toContain('Add child');
    expect(after).toContain('Remove node');
    expect(after).not.toContain('插入子主题');
  });

  it('切语言不丢内容：用户改过的图重建后还在，保存写出去的也是那一份', async () => {
    const { handle, currentHandle, errors } = await mountReady();
    await act(async () => {
      handle.setValue(EDITED);
    });
    await act(async () => {
      setLocale('en');
    });

    const next = currentHandle();
    expect(errors).toEqual([]);
    // 重建交的是**新**句柄：挂载时拿到的那个已经指向一个销毁了的实例
    expect(next).not.toBeNull();
    expect(next).not.toBe(handle);

    const saved = next?.getValue() ?? '';
    expect(saved).toContain('改过的根主题');
    // 种子取的是 `latestRef` 而不是 `initialValue` —— 换了的话这里会回到打开时的样子
    expect(saved).not.toContain('分支甲');
    expect(container.querySelector('.map-container')?.textContent).toContain('改过的根主题');
  });

  it('切语言不把没改过的文件变成改过的 —— 逐字节一致那条不能因为重建破掉', async () => {
    const { currentHandle, changes } = await mountReady();
    await act(async () => {
      setLocale('en');
    });
    // 重建本身不是一次编辑：外壳不该因此收到任何 onChange
    expect(changes).toEqual([]);
    expect(currentHandle()?.getValue()).toBe(CANONICAL);
  });

  it('保存过一次再切语言，文件仍然逐字节一致 —— 种子不能被规范形式污染', async () => {
    // COMPACT 不是我们写出来的样子：它的规范形式与它自己不是同一串字节
    const { handle, currentHandle } = await mountReady({ initialValue: COMPACT });
    expect(handle.getValue()).toBe(COMPACT);

    await act(async () => {
      setLocale('en');
    });
    // `getValue()` 记进 `latestRef` 的必须是**交出去的那个值**。记成 `lastText`
    // （规范形式）的话，下一次保存就会把用户的文件重排版，而界面上看不出任何异常。
    expect(currentHandle()?.getValue()).toBe(COMPACT);
  });

  it('只读时根本没有右键菜单 —— 换成对象传参之后这条不能松', async () => {
    await mountReady({ readOnly: true });
    expect(menuText()).toBe('');
  });
});

/**
 * 图片导出（ADR 0050）：这里只钉**接缝**——
 *
 * - 内核真的把 `exportImage` 挂上了（缺了它，外壳的导出命令整条禁用）；
 * - SVG 路径真的交得出一份 `image/svg+xml` 的 Blob（内核 exportSvg 是同步的，
 *   jsdom 里也跑得完）；
 * - 这份没有外链图片的文档，`warnings` 是空的（谎报警告与漏报一样是错）。
 *
 * **PNG 不在这里测，也不可能在**：内核转 PNG 用 `new Image()` 等加载事件，
 * 而 jsdom 的图片永远不加载 —— 那条路在测试环境里不会 resolve，只会挂起。
 * 像素对不对（底板底色、文字、连线）更没有布局引擎可言，
 * 靠 `pnpm run tauri dev` 里的人工冒烟，写在这里只会得到一条假绿的测试。
 */
describe('exportImage（导图内核 → 一张图）', () => {
  it('★ handle 上真的有 exportImage —— 外壳的命令靠它才亮得起来', async () => {
    const { handle } = await mountReady();
    expect(typeof handle.exportImage).toBe('function');
  });

  it('★ SVG：交得出一份真的 SVG Blob，且没有外链图片时不多说警告', async () => {
    const { handle } = await mountReady();

    const image = await handle.exportImage?.('svg');

    expect(image).not.toBeNull();
    expect(image?.blob.type).toBe('image/svg+xml');
    // 真画了东西，不是空壳（内核画布尺寸 + 节点序列化，都在这个尺寸里）
    expect((image?.blob.size ?? 0)).toBeGreaterThan(0);
    expect(image?.warnings).toEqual([]);
  });

  it('导一次之后还能再导 —— 临时换掉的主题真被还原了，而不是留在那儿', async () => {
    const { handle } = await mountReady();

    const first = await handle.exportImage?.('svg');
    const second = await handle.exportImage?.('svg');

    // 第二次还能画且尺寸一致：还原没还原好（主题对象被换坏）的话，
    // 第二次要么画不出、要么内容明显不同
    expect(second).not.toBeNull();
    expect(second?.blob.type).toBe('image/svg+xml');
    expect(second?.blob.size).toBe(first?.blob.size);
  });
});
