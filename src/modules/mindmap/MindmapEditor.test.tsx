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
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { EditorHandle, EditorMountProps } from '../../core/editor/types.js';
import { setLocale } from '../../core/i18n/index.js';
import { DEFAULT_FIND_OPTIONS } from '../../core/search/find.js';
import { MINDMAP_FORMAT, MINDMAP_VERSION } from '../../core/mindmap/types.js';
import { extractMindmapOutline } from '../../core/outline/mindmap-outline.js';
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
 * 备注也搜得到（ADR 0058）。`collectHits` 每个节点先搜 `topic` 后搜 `note`，
 * `EditorSearch` 契约（索引制）零改动 —— 命中表是内核私有的，外壳只见计数。
 * 这族用例要么整体成立要么整体不成立（换掉的是同一张命中表），不拆。
 */
describe('查找也搜备注（ADR 0058）', () => {
  /** 根主题带一条含「分支」的备注，子分支 n2 的 topic 也叫「分支」—— 一处 topic 一处 note。 */
  const noteSearchFile = (): string =>
    `${JSON.stringify(
      {
        format: MINDMAP_FORMAT,
        version: MINDMAP_VERSION,
        layout: 'logicalStructure',
        root: {
          id: 'n1',
          topic: '根主题',
          note: '根主题的备注；分支也写在这里',
          children: [{ id: 'n2', topic: '分支' }],
        },
      },
      null,
      2,
    )}\n`;

  /** 整段备注就是查询词本身 —— 替换成空串应当把 `note` 键整条摘掉（清空即移除）。 */
  const wholeNoteFile = (): string =>
    `${JSON.stringify(
      {
        format: MINDMAP_FORMAT,
        version: MINDMAP_VERSION,
        layout: 'logicalStructure',
        root: {
          id: 'n1',
          topic: '根主题',
          note: '分支',
          children: [{ id: 'n2', topic: '分支' }],
        },
      },
      null,
      2,
    )}\n`;

  const settleKernel = async (): Promise<void> => {
    await act(async () => {
      await new Promise((resolve) => {
        setTimeout(resolve, 0);
      });
    });
  };

  const lastChange = (changes: readonly string[]): string => changes[changes.length - 1] ?? '';

  it('备注里的词计入总数，topic 与 note 各算一处', async () => {
    const { handle } = await mountReady({ initialValue: noteSearchFile() });
    const count = handle.search?.find({ ...DEFAULT_FIND_OPTIONS, query: '分支' }) ?? 0;
    expect(count).toBe(2);

    const noteOnly = handle.search?.find({ ...DEFAULT_FIND_OPTIONS, query: '备注' }) ?? 0;
    expect(noteOnly).toBe(1);
  });

  it('同一节点的命中 topic 在前 note 在后；定位到 note 时弹层要开着、回到 topic 时关掉', async () => {
    const { handle } = await mountReady({ initialValue: noteSearchFile() });
    const count = handle.search?.find({ ...DEFAULT_FIND_OPTIONS, query: '根主题' }) ?? 0;
    expect(count).toBe(2);

    // 第一处是 n1 自己的 topic —— 弹层不该赖在屏上
    act(() => {
      handle.search?.reveal(0);
    });
    expect(container.querySelector('.mm-note-popover')).toBeNull();

    // 第二处是 n1 的备注 —— 打开弹层把整条备注端出来（jsdom 里 rect 全 0，弹层照样渲染）
    act(() => {
      handle.search?.reveal(1);
    });
    const popover = container.querySelector<HTMLElement>('.mm-note-popover');
    expect(popover).not.toBeNull();
    expect(popover?.querySelector<HTMLTextAreaElement>('.mm-note-input')?.value).toBe('根主题的备注；分支也写在这里');
  });

  it('替换命中改的是备注那一段，弹层跟着关上', async () => {
    const { handle, changes } = await mountReady({ initialValue: noteSearchFile() });
    handle.search?.find({ ...DEFAULT_FIND_OPTIONS, query: '分支' });

    // 命中表是 [n1 note, n2 topic]，第 0 处是备注
    const remaining = handle.search?.replace(0, '换好了') ?? -1;
    expect(container.querySelector('.mm-note-popover')).toBeNull();

    await settleKernel();
    expect(lastChange(changes)).toContain('"note"');
    expect(lastChange(changes)).toContain('换好了');
    // n2 的 topic 还有一处「分支」，重找的计数就是剩下的那处
    expect(remaining).toBe(1);
  });

  it('把整条备注替换成空串 = 清空即移除，note 键不再出现在文件里', async () => {
    const { handle, changes } = await mountReady({ initialValue: wholeNoteFile() });
    handle.search?.find({ ...DEFAULT_FIND_OPTIONS, query: '分支' });

    handle.search?.replace(0, '');
    await settleKernel();
    expect(lastChange(changes)).not.toContain('"note"');
    // n2 的 topic 还是完整的「分支」—— 被摘掉的只有备注
    expect(lastChange(changes)).toContain('"topic": "分支"');
  });

  it('replaceAll 一把换掉 topic 与 note 两处，报数按命中数不按节点数', async () => {
    const { handle, changes } = await mountReady({ initialValue: noteSearchFile() });
    handle.search?.find({ ...DEFAULT_FIND_OPTIONS, query: '分支' });

    const done = handle.search?.replaceAll('改字') ?? -1;
    expect(done).toBe(2);
    expect(container.querySelector('.mm-note-popover')).toBeNull();

    // 备注那段是 reshapeNode → operation，当场就发；topic 那段是 setNodeTopic
    // → linkDiv 兜底，要等 300ms 的去抖过去才到（ADR 0056 同款路径）。
    await act(async () => {
      await new Promise((r) => {
        setTimeout(r, 350);
      });
    });
    expect(lastChange(changes)).toContain('改字也写在这里');
    expect(lastChange(changes)).toContain('"topic": "改字"');
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

/**
 * 大纲跳转（ADR 0052）。这里钉的接缝：
 *
 * - handle 上真的有 `goToOutlineItem`（缺了它，导图文档的大纲条目全灰）；
 * - 跳转是**选中**不是编辑：没动过文件的跳转之后 `changes` 一条都没有，
 *   `getValue()` 逐字节不变；
 * - 折叠的祖先会被展开，而**展开是文档变更**：`collapsed` 是文件字段（§7.1），
 *   所以那条 onChange 与标脏是诚实的，保存会把展开状态写进文件
 *   （例外：**根**—— 内核对根的折叠不做视觉隐藏，也没有根的展开器，
 *   跳转照常成功，但不假装展开过）；
 * - 面板（`extractMindmapOutline`）与内核吃的 id **同源** —— 缺 id 的文件
 *   两边都由同一个解析器补 `auto-` 前缀，跳转才能指对地方。
 *
 * jsdom 没有布局引擎，所以「图真的滚到那个主题跟前」断不了；这里断的是
 * 数据层的每一步（返回值、onChange、getValue），视口效果靠
 * `pnpm run tauri dev` 的人工冒烟。
 */
describe('goToOutlineItem（大纲跳转，ADR 0052）', () => {
  /** 根折叠着，孙辈（甲的孙）藏在里面 —— 跳它必须先把根展开。 */
  const COLLAPSED = `${JSON.stringify(
    {
      format: MINDMAP_FORMAT,
      version: MINDMAP_VERSION,
      layout: 'logicalStructure',
      root: {
        id: 'r',
        topic: '根',
        collapsed: true,
        children: [
          { id: 'jia', topic: '甲', children: [{ id: 'a1', topic: '甲的孙', children: [] }] },
          { id: 'yi', topic: '乙', children: [] },
        ],
      },
    },
    null,
    2,
  )}\n`;

  /** 两层都折叠：a 折着 a1，a1 折着 a11 —— 跳 a11 要展开两段。 */
  const DOUBLE = `${JSON.stringify(
    {
      format: MINDMAP_FORMAT,
      version: MINDMAP_VERSION,
      layout: 'logicalStructure',
      root: {
        id: 'root',
        topic: '根',
        children: [
          {
            id: 'a',
            topic: '一层',
            collapsed: true,
            children: [{ id: 'a1', topic: '二层', collapsed: true, children: [{ id: 'a11', topic: '三层', children: [] }] }],
          },
        ],
      },
    },
    null,
    2,
  )}\n`;

  it('★ handle 上真的有 goToOutlineItem —— 外壳的大纲条目靠它才点得动', async () => {
    const { handle } = await mountReady();
    expect(typeof handle.goToOutlineItem).toBe('function');
  });

  it('按 id 跳到可见的主题：true，且选中不是编辑 —— changes 空、getValue 逐字节不变', async () => {
    const { handle, changes } = await mountReady();

    const jumped = handle.goToOutlineItem?.({ level: 2, text: '分支甲', id: 'n2' }) ?? false;

    expect(jumped).toBe(true);
    expect(changes).toEqual([]);
    expect(handle.getValue()).toBe(CANONICAL);
  });

  it('没带 id 时按 (层级, 文字) 找第一个 —— 老契约不破', async () => {
    const { handle } = await mountReady();
    expect(handle.goToOutlineItem?.({ level: 2, text: '分支乙' }) ?? false).toBe(true);
  });

  it('找不到的主题交 false —— 如实说没跳过去，而不是抛错或假装成功', async () => {
    const { handle } = await mountReady();
    expect(handle.goToOutlineItem?.({ level: 2, text: '分支甲', id: 'no-such-node' }) ?? false).toBe(false);
  });

  it('根折叠着的文件照样跳：内核本就不藏根的子分支，也不假装展开它', async () => {
    const { handle, changes } = await mountReady({ initialValue: COLLAPSED });

    const jumped = handle.goToOutlineItem?.({ level: 3, text: '甲的孙', id: 'a1' }) ?? false;

    // 内核对根的 collapsed 不做视觉隐藏（子分支永远渲染），`expandNode`
    // 对根还会抛错（根没有 +/- 气泡）—— 所以这里如实的做法是：
    // 跳过去（true），但不假装展开过（一次 onChange 都没有，文件一个字节不动）
    expect(jumped).toBe(true);
    expect(changes).toEqual([]);
    expect(handle.getValue()).toBe(COLLAPSED);
  });

  it('中层折叠跳最深层：每展开一层是一次文档变更 —— changes 恰好两条', async () => {
    const { handle, changes } = await mountReady({ initialValue: DOUBLE });

    const jumped = handle.goToOutlineItem?.({ level: 4, text: '三层', id: 'a11' }) ?? false;

    expect(jumped).toBe(true);
    // 两次展开 = 两条 onChange：collapsed 是文件字段，保存会把展开状态写进文件
    expect(changes).toHaveLength(2);
    expect(handle.getValue()).not.toContain('"collapsed"');
  });

  it('缺 id 的文件：面板算出的 auto-id 与内核吃到的是同一套（同源解析器）—— 跳得中', async () => {
    const noId = `${JSON.stringify(
      {
        format: MINDMAP_FORMAT,
        version: MINDMAP_VERSION,
        layout: 'logicalStructure',
        root: { topic: '根', children: [{ topic: '分支', children: [] }] },
      },
      null,
      2,
    )}\n`;
    const { handle } = await mountReady({ initialValue: noId });

    // 面板走的就是这条提取器 —— 拿它的条目跳，等于复放真实点击的那条数据链
    const items = extractMindmapOutline(noId);
    expect(items.map((item) => item.id)).toEqual(['auto-0', 'auto-1']);

    const leaf = items[1];
    expect(leaf).toBeDefined();
    expect(handle.goToOutlineItem?.({ level: leaf?.level ?? 0, text: leaf?.text ?? '', id: leaf?.id }) ?? false).toBe(true);
  });

  it('只读时跳折叠子树：不展开（那是编辑），交 false —— 而不是弄脏文件', async () => {
    const { handle, changes } = await mountReady({ initialValue: DOUBLE, readOnly: true });

    expect(handle.goToOutlineItem?.({ level: 4, text: '三层', id: 'a11' }) ?? false).toBe(false);
    expect(changes).toEqual([]);
    expect(handle.getValue()).toBe(DOUBLE);
  });

  it('只读时跳可见主题照跳 —— 选中不是编辑，文件不动一个字节', async () => {
    // 根折叠不影响可见性（内核不藏根的子分支），只读也照跳、照样一个字节不动
    const { handle, changes } = await mountReady({ initialValue: COLLAPSED, readOnly: true });

    expect(handle.goToOutlineItem?.({ level: 3, text: '甲的孙', id: 'a1' }) ?? false).toBe(true);
    expect(changes).toEqual([]);
    expect(handle.getValue()).toBe(COLLAPSED);
  });
});

describe('标记调色盘（ADR 0053）', () => {
  /**
   * 右键根主题，把菜单点开。内核 `g`（MindElixir.js:1198-1203）有三个坑：
   * ① 只认 `f.button === 2`——jsdom 的 `MouseEvent` 默认 button=0，必须显式给；
   * ② 菜单是 `setTimeout(…, 200)` 之后才弹的，右键完要真等一下；
   * ③ 内核把节点 id 加了 `me` 前缀（:397 `t.dataset.nodeid = "me" + e.id`），
   *    所以夹具根 id 是 `n1`、DOM 里要找 `[data-nodeid="men1"]`。
   * 等到之后内核已先 `selectNode`（:1202），所以 extend 项的 handler 里
   * `currentNode` 就是右键到的那个节点 —— 这条链路断了调色盘会打开但落不了盘。
   */
  const rightClickRoot = async (): Promise<void> => {
    const topic = container.querySelector<HTMLElement>('[data-nodeid="men1"]');
    expect(topic).not.toBeNull();
    await act(async () => {
      topic?.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, button: 2 }));
      await new Promise((resolve) => {
        setTimeout(resolve, 250);
      });
    });
  };

  /** 菜单里那条「标记…」（内核对 extend 项不藏菜单，是我们自己藏，见实现注释）。 */
  const markerEntry = (): HTMLLIElement | null => {
    const items = [...container.querySelectorAll<HTMLLIElement>('.context-menu li')];
    return items.find((li) => li.textContent === '标记…') ?? null;
  };

  const openPalette = async (): Promise<HTMLElement> => {
    await rightClickRoot();
    const entry = markerEntry();
    expect(entry).not.toBeNull();
    await act(async () => {
      entry?.click();
    });
    const palette = container.querySelector<HTMLElement>('.mm-marker-palette');
    expect(palette).not.toBeNull();
    return palette as HTMLElement;
  };

  /** 调色盘里某个字形的按钮。 */
  const glyphButton = (palette: HTMLElement, glyph: string): HTMLButtonElement => {
    const button = [...palette.querySelectorAll<HTMLButtonElement>('button')].find(
      (entry) => entry.textContent === glyph,
    );
    expect(button).not.toBeNull();
    return button as HTMLButtonElement;
  };

  const iconsOf = (text: string): readonly string[] => {
    const root = (JSON.parse(text) as { root: { icons?: readonly string[] } }).root;
    return root.icons ?? [];
  };

  it('右键菜单末尾有「标记…」，点开是调色盘 —— 三组全在（词表住内核模块）', async () => {
    await mountReady();
    const palette = await openPalette();

    expect(palette.querySelectorAll('.mm-marker-group')).toHaveLength(3);
    // 悬停文案走 i18n（组头是中文的 —— 语言切换靠内核重建，与 ADR 0049 同一条命门）
    expect(palette.textContent).toContain('优先级');
    expect(palette.textContent).toContain('符号');
    expect(palette.textContent).toContain('清除这个主题上的全部标记');
  });

  it('标一个优先级：落盘走 reshapeNode → operation → onChange，文件里多出 icons', async () => {
    const { handle, changes } = await mountReady();
    const palette = await openPalette();

    act(() => {
      glyphButton(palette, '1️⃣').click();
    });

    expect(changes).toHaveLength(1);
    expect(iconsOf(changes[0] ?? '')).toEqual(['1️⃣']);
    // 挂上去的字形在调色盘里亮着 —— 再点一次就是摘掉
    expect(glyphButton(palette, '1️⃣').className).toContain('active');
    // 内核真把它画在主题上（icons 渲染由内核负责，这里只证载荷在）
    expect(container.querySelector('[data-nodeid="men1"]')?.textContent).toContain('1️⃣');
    expect(handle.getValue()).toContain('"icons"');
  });

  it('再点一次是摘掉：全清过写成 `"icons": []` —— 不是悄悄少一句话', async () => {
    const { changes } = await mountReady();
    const palette = await openPalette();

    act(() => {
      glyphButton(palette, '⭐').click();
    });
    act(() => {
      glyphButton(palette, '⭐').click();
    });

    const last = changes[changes.length - 1] ?? '';
    expect(iconsOf(last)).toEqual([]);
    expect(last).toContain('"icons": []');
  });

  it('互斥组：标了优先级 1 再标 2 —— 文件里只有一个，后标的排在后面', async () => {
    const { changes } = await mountReady();
    const palette = await openPalette();

    act(() => {
      glyphButton(palette, '1️⃣').click();
    });
    act(() => {
      glyphButton(palette, '2️⃣').click();
    });

    expect(iconsOf(changes[changes.length - 1] ?? '')).toEqual(['2️⃣']);
  });

  it('清除按钮：两个组的标记一起摘，文件里留下 `"icons": []`', async () => {
    const { changes } = await mountReady();
    const palette = await openPalette();

    act(() => {
      glyphButton(palette, '1️⃣').click();
    });
    act(() => {
      glyphButton(palette, '⭐').click();
    });
    expect(iconsOf(changes[changes.length - 1] ?? '')).toEqual(['1️⃣', '⭐']);

    act(() => {
      (palette.querySelector<HTMLButtonElement>('.mm-marker-clear') as HTMLButtonElement).click();
    });

    const cleared = changes[changes.length - 1] ?? '';
    expect(iconsOf(cleared)).toEqual([]);
    expect(cleared).toContain('"icons": []');
  });

  it('点调色盘外面就关 —— 但点字形不关（一次标多个是正常用法）', async () => {
    await mountReady();
    const palette = await openPalette();

    act(() => {
      glyphButton(palette, '1️⃣').click();
    });
    // 点了字形，调色盘还开着
    expect(container.querySelector('.mm-marker-palette')).not.toBeNull();

    act(() => {
      document.body.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
    });
    expect(container.querySelector('.mm-marker-palette')).toBeNull();
  });

  it('只读时右键没有菜单，更没有调色盘 —— 标记是编辑，不悄悄放行', async () => {
    await mountReady({ readOnly: true });

    expect(container.querySelector('.context-menu')).toBeNull();
    await rightClickRoot();
    expect(markerEntry()).toBeNull();
    expect(container.querySelector('.mm-marker-palette')).toBeNull();
  });
});

describe('节点插图（ADR 0054）', () => {
  /**
   * 显示链路的事实（侦察期定的形状）：
   * - 文件里的 `image.url` 是**相对路径**（外壳的粘贴约定），内核要的是能画的
   *   data: URL —— 中间这层转换由内核挂载前的「预热」完成，查不到就画占位图。
   * - jsdom 不解码图片，所以这里只证**接缝**：`resolveImageUrl` 收到什么、
   *   `<img>` 的 src 最终是什么、文件有没有被动过。像素观感靠人工冒烟。
   */

  /** 根主题带一张图（引用 assets/ 下的一张 png）的规范文件。 */
  const imageFile = (url: string): string =>
    `${JSON.stringify(
      {
        format: MINDMAP_FORMAT,
        version: MINDMAP_VERSION,
        layout: 'logicalStructure',
        root: {
          id: 'n1',
          topic: '带图的根',
          image: { url, width: 320, height: 240 },
          children: [{ id: 'n2', topic: '分支', children: [] }],
        },
      },
      null,
      2,
    )}\n`;

  /** 根主题载荷里内核画出来的那张 `<img>`（有图才存在）。 */
  const rootImg = (): HTMLImageElement | null =>
    container.querySelector<HTMLImageElement>('[data-nodeid="men1"] img');

  /** 右键菜单里那条「图片…」。 */
  const imageEntry = (): HTMLLIElement | null => {
    const items = [...container.querySelectorAll<HTMLLIElement>('.context-menu li')];
    return items.find((li) => li.textContent === '图片…') ?? null;
  };

  const openImagePopover = async (): Promise<HTMLElement> => {
    const topic = container.querySelector<HTMLElement>('[data-nodeid="men1"]');
    expect(topic).not.toBeNull();
    await act(async () => {
      topic?.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, button: 2 }));
      await new Promise((resolve) => {
        setTimeout(resolve, 250);
      });
    });
    const entry = imageEntry();
    expect(entry).not.toBeNull();
    await act(async () => {
      entry?.click();
    });
    const popover = container.querySelector<HTMLElement>('.mm-image-popover');
    expect(popover).not.toBeNull();
    return popover as HTMLElement;
  };

  /** jsdom 不解码图片，尺寸解码必须换成可控的假货（真图路径靠人工冒烟）。 */
  const stubImage = (width: number, height: number): void => {
    class FakeImage {
      naturalWidth = width;
      naturalHeight = height;
      onload: (() => void) | null = null;
      onerror: (() => void) | null = null;
      src = '';
      constructor() {
        queueMicrotask(() => {
          this.onload?.();
        });
      }
    }
    vi.stubGlobal('Image', FakeImage);
  };

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('相对路径先经外壳解析成 data: URL 再上屏 —— 内核要的是能直接画的 src', async () => {
    const asked: string[] = [];
    const { handle, changes } = await mountReady({
      initialValue: imageFile('assets/a.png'),
      resolveImageUrl: async (src) => {
        asked.push(src);
        return 'data:image/png;base64,AAAA';
      },
    });

    expect(asked).toEqual(['assets/a.png']);
    expect(rootImg()?.getAttribute('src')).toBe('data:image/png;base64,AAAA');
    // 预热只影响显示，文件一个字节不动
    expect(changes).toEqual([]);
    expect(handle.getValue()).toBe(imageFile('assets/a.png'));
  });

  it('图丢了走占位：路径如实写在占位图里，照样不碰文件', async () => {
    const { changes } = await mountReady({
      initialValue: imageFile('assets/lost.png'),
      resolveImageUrl: async () => null,
    });

    const src = rootImg()?.getAttribute('src') ?? '';
    expect(src.startsWith('data:image/svg+xml')).toBe(true);
    expect(decodeURIComponent(src)).toContain('assets/lost.png');
    expect(changes).toEqual([]);
  });

  it('data: 引用不劳驾外壳 —— 本来就能画，原样透传', async () => {
    const asked: string[] = [];
    await mountReady({
      initialValue: imageFile('data:image/png;base64,AAAA'),
      resolveImageUrl: async (src) => {
        asked.push(src);
        return 'SHOULD_NOT_BE_USED';
      },
    });

    expect(asked).toEqual([]);
    expect(rootImg()?.getAttribute('src')).toBe('data:image/png;base64,AAAA');
  });

  it('选中主题后粘贴图片：字节交给外壳落盘，回来的是文件 src + 内联 data: 上屏', async () => {
    stubImage(640, 480);
    const pasted: { bytes: ArrayBuffer; mimeType: string }[] = [];
    const { handle, changes, errors } = await mountReady({
      onImagePaste: async (request) => {
        pasted.push({ ...request });
        return 'assets/pasted.png';
      },
    });

    const topic = container.querySelector<HTMLElement>('[data-nodeid="men1"]');
    expect(topic).not.toBeNull();
    await act(async () => {
      topic?.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, button: 2 }));
      await new Promise((resolve) => {
        setTimeout(resolve, 250);
      });
    });

    const data = new DataTransfer();
    data.items.add(new File([new Uint8Array([0x89, 0x50, 0x4e, 0x47])], 'shot.png', { type: 'image/png' }));
    await act(async () => {
      container.querySelector('.map-container')?.dispatchEvent(
        new ClipboardEvent('paste', { clipboardData: data, bubbles: true, cancelable: true }),
      );
      await new Promise((resolve) => {
        setTimeout(resolve, 30);
      });
    });

    // 外壳收到的是原始字节和 MIME —— 存哪、叫什么都归外壳管
    expect(pasted).toHaveLength(1);
    expect(pasted[0]?.bytes.byteLength).toBe(4);
    expect(pasted[0]?.mimeType).toBe('image/png');

    // 挂图是编辑：文件里多出 image，尺寸按最长边 320 缩好
    const last = changes[changes.length - 1] ?? '';
    expect(last).toContain('"image"');
    expect(last).toContain('assets/pasted.png');
    expect(last).toContain('"width": 320');
    expect(last).toContain('"height": 240');
    expect(errors).toEqual([]);
    // 上屏的 src 是粘贴字节的内联 data:（粘贴路径不经过外壳解析）
    expect(rootImg()?.getAttribute('src')).toBe('data:image/png;base64,iVBORw==');
    expect(handle.getValue()).toContain('"image"');
  });

  it('没选中主题就粘贴：如实说「先点一个主题」，字节不交出去', async () => {
    const pasted: unknown[] = [];
    const { changes, errors } = await mountReady({
      onImagePaste: async (request) => {
        pasted.push(request);
        return 'assets/pasted.png';
      },
    });

    const data = new DataTransfer();
    data.items.add(new File([new Uint8Array([0x89, 0x50, 0x4e, 0x47])], 'shot.png', { type: 'image/png' }));
    await act(async () => {
      container.querySelector('.map-container')?.dispatchEvent(
        new ClipboardEvent('paste', { clipboardData: data, bubbles: true, cancelable: true }),
      );
      await new Promise((resolve) => {
        setTimeout(resolve, 30);
      });
    });

    expect(pasted).toHaveLength(0);
    expect(changes).toEqual([]);
    expect(errors.map((error) => error.message)).toContain('剪贴板里有图片，但没有选中的主题 —— 先点一个主题，再粘贴。');
  });

  it('外壳存盘失败（交回 null）：内核什么都不做也不报错 —— 两条消息比零条少', async () => {
    const { changes, errors } = await mountReady({
      onImagePaste: async () => null,
    });

    const topic = container.querySelector<HTMLElement>('[data-nodeid="men1"]');
    await act(async () => {
      topic?.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, button: 2 }));
      await new Promise((resolve) => {
        setTimeout(resolve, 250);
      });
    });

    const data = new DataTransfer();
    data.items.add(new File([new Uint8Array([0x89, 0x50, 0x4e, 0x47])], 'shot.png', { type: 'image/png' }));
    await act(async () => {
      container.querySelector('.map-container')?.dispatchEvent(
        new ClipboardEvent('paste', { clipboardData: data, bubbles: true, cancelable: true }),
      );
      await new Promise((resolve) => {
        setTimeout(resolve, 30);
      });
    });

    expect(changes).toEqual([]);
    expect(errors).toEqual([]);
    expect(rootImg()).toBeNull();
  });

  it('只读时粘贴直接不接招 —— 图片是编辑，不悄悄放行', async () => {
    const pasted: unknown[] = [];
    const { changes, errors } = await mountReady({
      readOnly: true,
      onImagePaste: async (request) => {
        pasted.push(request);
        return 'assets/pasted.png';
      },
    });

    const data = new DataTransfer();
    data.items.add(new File([new Uint8Array([0x89, 0x50, 0x4e, 0x47])], 'shot.png', { type: 'image/png' }));
    await act(async () => {
      container.querySelector('.map-container')?.dispatchEvent(
        new ClipboardEvent('paste', { clipboardData: data, bubbles: true, cancelable: true }),
      );
      await new Promise((resolve) => {
        setTimeout(resolve, 30);
      });
    });

    expect(pasted).toHaveLength(0);
    expect(changes).toEqual([]);
    expect(errors).toEqual([]);
  });

  it('右键「图片…」：有图弹层显示引用与尺寸，移除后文件里不再有 image', async () => {
    const { handle, changes } = await mountReady({
      initialValue: imageFile('assets/a.png'),
      resolveImageUrl: async () => 'data:image/png;base64,AAAA',
    });

    const popover = await openImagePopover();
    const meta = popover.querySelector<HTMLElement>('.mm-image-meta');
    expect(meta?.textContent).toContain('assets/a.png');
    expect(meta?.textContent).toContain('320 × 240');

    act(() => {
      popover.querySelector<HTMLButtonElement>('.mm-image-remove')?.click();
    });

    const last = changes[changes.length - 1] ?? '';
    expect(last).not.toContain('"image"');
    expect(handle.getValue()).not.toContain('"image"');
    // 移除完弹层自己收掉 —— 「全清过」没有可说的信息了
    expect(container.querySelector('.mm-image-popover')).toBeNull();
  });

  it('右键「图片…」：没图时不给空弹层，给「怎么粘」的提示', async () => {
    await mountReady();

    const popover = await openImagePopover();
    expect(popover.querySelector('.mm-image-hint')).not.toBeNull();
    expect(popover.textContent).toContain('Ctrl+V');
    // 没图就没有移除按钮 —— 不是禁用，是压根没有
    expect(popover.querySelector('.mm-image-remove')).toBeNull();
  });
});

describe('链接与标签（ADR 0055）', () => {
  /**
   * 与标记/插图不同的两个事实（写用例前先说清楚，不然断言会选错层）：
   * - `hyperLink` / `tags` 的**渲染是内核原生**的（Se，MindElixir.js:364-378），
   *   这里只证「落盘 + 交给外壳」的接缝，不证画得对不对。
   * - 内核对 hyperLink **零 UI**（菜单里的 link 项是节点连线，不是它），所以
   *   编辑弹层、🔗 的点击激活全是我们的代码 —— 断言可以直接落在 DOM 上。
   * - 🔗 是显式挂上的链接 chip，单击即激活（XMind 同款）；markdown 正文里
   *   要 Ctrl+点击 是因为正文链接是嵌在文字里的。差异写进了 ADR。
   */

  /** 根主题带可选 hyperLink / tags 的规范文件。 */
  const linkTagFile = (root: Record<string, unknown>): string =>
    `${JSON.stringify(
      {
        format: MINDMAP_FORMAT,
        version: MINDMAP_VERSION,
        layout: 'logicalStructure',
        root: { id: 'n1', topic: '根主题', children: [], ...root },
      },
      null,
      2,
    )}\n`;

  /** 内核在根主题载荷里画出来的那个 🔗（有链接才存在）。 */
  const rootLink = (): HTMLAnchorElement | null =>
    container.querySelector<HTMLAnchorElement>('[data-nodeid="men1"] a.hyper-link');

  const tagsOf = (text: string): readonly unknown[] => {
    const root = (JSON.parse(text) as { root: { tags?: readonly unknown[] } }).root;
    return root.tags ?? [];
  };

  /** 右键菜单里那条「链接…」/「标签…」。 */
  const menuEntry = (title: string): HTMLLIElement | null => {
    const items = [...container.querySelectorAll<HTMLLIElement>('.context-menu li')];
    return items.find((li) => li.textContent === title) ?? null;
  };

  const openPopover = async (title: string, className: string): Promise<HTMLElement> => {
    const topic = container.querySelector<HTMLElement>('[data-nodeid="men1"]');
    expect(topic).not.toBeNull();
    await act(async () => {
      topic?.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, button: 2 }));
      await new Promise((resolve) => {
        setTimeout(resolve, 250);
      });
    });
    const entry = menuEntry(title);
    expect(entry).not.toBeNull();
    await act(async () => {
      entry?.click();
    });
    const popover = container.querySelector<HTMLElement>(className);
    expect(popover).not.toBeNull();
    return popover as HTMLElement;
  };

  /** React 受控输入要借原生 setter 才认输入 —— 直接赋 value 会被 React 吞掉。 */
  const typeInto = (input: HTMLInputElement, value: string): void => {
    // 先 bind 再用（unbound-method）：与 App.test.tsx 里 native setter 的写法一致
    const setNativeValue = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set?.bind(input);
    setNativeValue?.(value);
    input.dispatchEvent(new Event('input', { bubbles: true }));
  };

  it('右键「链接…」：输入框预填当前链接，改成新链接落盘、弹层自己关', async () => {
    const { handle, changes } = await mountReady({ initialValue: linkTagFile({ hyperLink: 'https://example.com/a' }) });

    const popover = await openPopover('链接…', '.mm-link-popover');
    const input = popover.querySelector<HTMLInputElement>('.mm-link-input');
    expect(input?.value).toBe('https://example.com/a');

    await act(async () => {
      typeInto(input as HTMLInputElement, 'https://example.com/b');
    });
    act(() => {
      popover.querySelector<HTMLButtonElement>('.mm-link-save')?.click();
    });

    const last = changes[changes.length - 1] ?? '';
    expect(last).toContain('"hyperLink": "https://example.com/b"');
    expect(handle.getValue()).toContain('"hyperLink": "https://example.com/b"');
    // 保存成功弹层自己关 —— 半途关掉才是异常
    expect(container.querySelector('.mm-link-popover')).toBeNull();
  });

  it('没有链接时：给提示与空输入，移除按钮压根没有；填上才能保存', async () => {
    const { changes } = await mountReady();

    const popover = await openPopover('链接…', '.mm-link-popover');
    expect(popover.querySelector('.mm-link-hint')).not.toBeNull();
    // 没链接就没有移除按钮 —— 不是禁用，是压根没有
    expect(popover.querySelector('.mm-link-remove')).toBeNull();

    const save = popover.querySelector<HTMLButtonElement>('.mm-link-save');
    expect(save?.disabled).toBe(true);

    await act(async () => {
      typeInto(popover.querySelector<HTMLInputElement>('.mm-link-input') as HTMLInputElement, 'notes/别的导图.mindmap');
    });
    expect(save?.disabled).toBe(false);
    act(() => {
      save?.click();
    });

    const last = changes[changes.length - 1] ?? '';
    expect(last).toContain('"hyperLink": "notes/别的导图.mindmap"');
  });

  it('移除链接：文件里不再有 hyperLink —— undefined 被 stringify 干净丢掉', async () => {
    const { handle, changes } = await mountReady({ initialValue: linkTagFile({ hyperLink: 'https://example.com/a' }) });

    const popover = await openPopover('链接…', '.mm-link-popover');
    act(() => {
      popover.querySelector<HTMLButtonElement>('.mm-link-remove')?.click();
    });

    const last = changes[changes.length - 1] ?? '';
    expect(last).not.toContain('"hyperLink"');
    expect(handle.getValue()).not.toContain('"hyperLink"');
    expect(container.querySelector('.mm-link-popover')).toBeNull();
  });

  it('点 🔗 激活链接：交给外壳分类打开，默认行为拦下 —— WebView 不能整页飞走', async () => {
    const activated: { href: string; text: string }[] = [];
    const { changes } = await mountReady({
      initialValue: linkTagFile({ hyperLink: 'https://example.com/a' }),
      onLinkClick: (link) => {
        activated.push({ ...link });
      },
    });

    const link = rootLink();
    expect(link).not.toBeNull();
    const event = new MouseEvent('click', { bubbles: true, cancelable: true });
    await act(async () => {
      link?.dispatchEvent(event);
    });

    // href 从内核 nodeObj 取，不从 DOM 读 —— DOM 会说谎（marker 教训）
    expect(activated).toEqual([{ href: 'https://example.com/a', text: '🔗' }]);
    // 整页导航必须拦在内核挂载层 —— 内核自己不拦（grep 全库只有渲染与导出两处）
    expect(event.defaultPrevented).toBe(true);
    // 激活不是编辑：文件一个字节不动
    expect(changes).toEqual([]);
  });

  it('只读也能点 🔗 —— 查链接不是改文件；但右键没有编辑菜单', async () => {
    const activated: { href: string; text: string }[] = [];
    await mountReady({
      readOnly: true,
      initialValue: linkTagFile({ hyperLink: 'notes/别的导图.mindmap' }),
      onLinkClick: (link) => {
        activated.push({ ...link });
      },
    });

    await act(async () => {
      rootLink()?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    });
    expect(activated).toEqual([{ href: 'notes/别的导图.mindmap', text: '🔗' }]);

    const topic = container.querySelector<HTMLElement>('[data-nodeid="men1"]');
    await act(async () => {
      topic?.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, button: 2 }));
      await new Promise((resolve) => {
        setTimeout(resolve, 250);
      });
    });
    expect(container.querySelector('.context-menu')).toBeNull();
    expect(menuEntry('链接…')).toBeNull();
    expect(menuEntry('标签…')).toBeNull();
  });

  it('标签弹层：回车与按钮都能挂，挂上后输入框清空、弹层不关 —— 挂多个是正常用法', async () => {
    const { handle, changes } = await mountReady();

    const popover = await openPopover('标签…', '.mm-tag-popover');
    const input = popover.querySelector<HTMLInputElement>('.mm-tag-input');

    await act(async () => {
      typeInto(input as HTMLInputElement, '重要');
    });
    await act(async () => {
      input?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
    });
    await act(async () => {
      typeInto(input as HTMLInputElement, '风险');
    });
    act(() => {
      popover.querySelector<HTMLButtonElement>('.mm-tag-add')?.click();
    });

    const last = changes[changes.length - 1] ?? '';
    expect(tagsOf(last)).toEqual(['重要', '风险']);
    expect(handle.getValue()).toContain('"tags"');
    // 输入框清空、弹层还开着 —— 连挂是正常用法
    expect(input?.value).toBe('');
    expect(container.querySelector('.mm-tag-popover')).not.toBeNull();
    // chip 挂上来了（文字在独立的 span 里，✕ 按钮不算文字）
    const chips = [...popover.querySelectorAll<HTMLElement>('.mm-tag-text')];
    expect(chips.map((chip) => chip.textContent)).toEqual(['重要', '风险']);
  });

  it('重复标签不落第二份：现值原样返回，文件不动', async () => {
    const { changes } = await mountReady({ initialValue: linkTagFile({ tags: ['重要'] }) });

    const popover = await openPopover('标签…', '.mm-tag-popover');
    const input = popover.querySelector<HTMLInputElement>('.mm-tag-input');
    await act(async () => {
      typeInto(input as HTMLInputElement, '重要');
    });
    act(() => {
      popover.querySelector<HTMLButtonElement>('.mm-tag-add')?.click();
    });

    // no-op：内核没收到 reshapeNode，onChange 一条都没多
    expect(changes).toEqual([]);
    const chips = [...popover.querySelectorAll<HTMLElement>('.mm-tag-text')];
    expect(chips).toHaveLength(1);
  });

  it('✕ 按索引摘标签；最后一个摘掉留下 `"tags": []` —— 全清过不是没有', async () => {
    const { handle, changes } = await mountReady({ initialValue: linkTagFile({ tags: ['重要', '风险'] }) });

    const popover = await openPopover('标签…', '.mm-tag-popover');
    const first = popover.querySelectorAll<HTMLButtonElement>('.mm-tag-remove')[0];
    act(() => {
      first?.click();
    });
    let last = changes[changes.length - 1] ?? '';
    expect(tagsOf(last)).toEqual(['风险']);

    const next = popover.querySelectorAll<HTMLButtonElement>('.mm-tag-remove')[0];
    act(() => {
      next?.click();
    });
    last = changes[changes.length - 1] ?? '';
    expect(last).toContain('"tags": []');
    expect(handle.getValue()).toContain('"tags": []');
    // 挂过又全摘掉 ≠ 从没挂过 —— 空列表提示与空输入并存
    expect(popover.querySelector('.mm-tag-empty')).not.toBeNull();
  });
});

describe('备注（ADR 0056）', () => {
  /**
   * 与标记/插图/链接/标签不同的一个事实（写用例前先说清楚）：
   * `note` 内核**零渲染**（Se 只画 text / icons / image / hyperLink / tags
   * 五样，MindElixir.js:353-378 里一个字都不画它）—— 徽章与右键
   * 「备注…」是我们的全部。徽章挂重画链（`linkDiv`）重同步，所以
   * undo / redo 把它带走又找回来的那条链也在这里钉死。
   */

  /** 根主题带可选 `note` 的规范文件；子分支 n2 恒不带备注（对照用）。 */
  const noteFile = (root: Record<string, unknown>): string =>
    `${JSON.stringify(
      {
        format: MINDMAP_FORMAT,
        version: MINDMAP_VERSION,
        layout: 'logicalStructure',
        root: {
          id: 'n1',
          topic: '根主题',
          children: [{ id: 'n2', topic: '分支' }],
          ...root,
        },
      },
      null,
      2,
    )}\n`;

  /** 根主题上的那枚 📝（有备注才存在）。 */
  const rootBadge = (): HTMLButtonElement | null =>
    container.querySelector<HTMLButtonElement>('[data-nodeid="men1"] .mm-note-badge');

  /** 右键菜单里那条「备注…」。 */
  const noteMenuEntry = (): HTMLLIElement | null => {
    const items = [...container.querySelectorAll<HTMLLIElement>('.context-menu li')];
    return items.find((li) => li.textContent === '备注…') ?? null;
  };

  /** 右键根主题 → 点「备注…」→ 拿到弹层（从零创建的唯一入口）。 */
  const openNoteViaMenu = async (): Promise<HTMLElement> => {
    const topic = container.querySelector<HTMLElement>('[data-nodeid="men1"]');
    expect(topic).not.toBeNull();
    await act(async () => {
      topic?.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, button: 2 }));
      await new Promise((resolve) => {
        setTimeout(resolve, 250);
      });
    });
    const entry = noteMenuEntry();
    expect(entry).not.toBeNull();
    await act(async () => {
      entry?.click();
    });
    const popover = container.querySelector<HTMLElement>('.mm-note-popover');
    expect(popover).not.toBeNull();
    return popover as HTMLElement;
  };

  /** React 受控 textarea 同样要借原生 setter（与输入框那条同一条命门）。 */
  const typeIntoArea = (area: HTMLTextAreaElement, value: string): void => {
    const setNativeValue = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')?.set?.bind(area);
    setNativeValue?.(value);
    area.dispatchEvent(new Event('input', { bubbles: true }));
  };

  /** 等内核把 reshapeNode 的异步重建跑完（徽章在 linkDiv 之后才回来）。 */
  const settleKernel = async (): Promise<void> => {
    await act(async () => {
      await new Promise((resolve) => {
        setTimeout(resolve, 0);
      });
    });
  };

  it('★ 右键菜单末尾有「备注…」—— 没有徽章可点的节点从这里创建第一条', async () => {
    const { handle, changes } = await mountReady({ initialValue: noteFile({}) });

    const popover = await openNoteViaMenu();
    // 还没有备注：空弹层 —— 没有移除按钮（不是禁用），保存按不下去
    expect(popover.querySelector('.mm-note-remove')).toBeNull();
    expect(popover.querySelector<HTMLButtonElement>('.mm-note-save')?.disabled).toBe(true);

    const area = popover.querySelector<HTMLTextAreaElement>('.mm-note-input');
    await act(async () => {
      typeIntoArea(area as HTMLTextAreaElement, '第一条备注');
    });
    act(() => {
      popover.querySelector<HTMLButtonElement>('.mm-note-save')?.click();
    });
    await settleKernel();

    const last = changes[changes.length - 1] ?? '';
    expect(last).toContain('"note": "第一条备注"');
    expect(handle.getValue()).toContain('"note": "第一条备注"');
    // 保存成功弹层自己关；徽章跟着 linkDiv 的重建长出来 —— 备注从看不见变成看得见
    expect(container.querySelector('.mm-note-popover')).toBeNull();
    expect(rootBadge()).not.toBeNull();
  });

  it('有备注的节点挂 📝 徽章，没有的节点不挂 —— 空串也算没有（弹层落盘已折掉它）', async () => {
    await mountReady({ initialValue: noteFile({ note: '根的备注' }) });

    const badge = rootBadge();
    expect(badge).not.toBeNull();
    expect(badge?.textContent).toBe('📝');
    // title 走 i18n（ADR 0049 同一条命门：语言变了内核重建、徽章重同步）
    expect(badge?.title).toBe('备注');
    // 对照：n2 没有备注，不挂徽章
    expect(container.querySelector('[data-nodeid="men2"] .mm-note-badge')).toBeNull();
  });

  it('点徽章直接开弹层：预填当前备注原文，Ctrl+Enter 落盘、弹层自关', async () => {
    const { handle, changes } = await mountReady({ initialValue: noteFile({ note: '第一版' }) });

    await act(async () => {
      rootBadge()?.click();
    });
    const popover = container.querySelector<HTMLElement>('.mm-note-popover');
    expect(popover).not.toBeNull();
    const area = popover?.querySelector<HTMLTextAreaElement>('.mm-note-input');
    expect(area?.value).toBe('第一版');

    await act(async () => {
      typeIntoArea(area as HTMLTextAreaElement, '第二版\n第二行');
    });
    await act(async () => {
      // 备注里的普通 Enter 是换行 —— 保存是 Ctrl+Enter，占位文案里写着
      area?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', ctrlKey: true, bubbles: true }));
    });

    const last = changes[changes.length - 1] ?? '';
    expect(last).toContain('"note": "第二版\\n第二行"');
    expect(handle.getValue()).toContain('"note": "第二版\\n第二行"');
    expect(container.querySelector('.mm-note-popover')).toBeNull();
  });

  it('移除备注：note 键干净消失（undefined 被 stringify 丢掉），徽章随重画摘掉', async () => {
    const { handle, changes } = await mountReady({ initialValue: noteFile({ note: '要被移除的' }) });

    await act(async () => {
      rootBadge()?.click();
    });
    const popover = container.querySelector<HTMLElement>('.mm-note-popover');
    // 有备注才有移除按钮 —— 与链接弹层同一条判据
    expect(popover?.querySelector('.mm-note-remove')).not.toBeNull();
    act(() => {
      popover?.querySelector<HTMLButtonElement>('.mm-note-remove')?.click();
    });
    await settleKernel();

    const last = changes[changes.length - 1] ?? '';
    expect(last).not.toContain('"note"');
    expect(handle.getValue()).not.toContain('"note"');
    expect(rootBadge()).toBeNull();
  });

  it('undo 把备注找回来：徽章跟着 linkDiv 重建重现，脏标记迟到但会到（ADR 0056 的两件套）', async () => {
    const { handle, changes } = await mountReady({ initialValue: noteFile({ note: '会被撤销的' }) });

    // 先移除（走徽章入口，证明同一条落盘通道）
    await act(async () => {
      rootBadge()?.click();
    });
    act(() => {
      container.querySelector<HTMLButtonElement>('.mm-note-remove')?.click();
    });
    await settleKernel();
    expect(handle.getValue()).not.toContain('"note"');

    // 内核的 undo（Kn 插件监听 .map-container 的 keydown：Ctrl+Z）
    // —— 它走 refresh(prev)，**不 fire operation**：徽章靠 linkDiv 重挂，
    // 脏标记靠 linkDiv → scheduleVerify 迟到。两条都在这里钉死。
    const map = container.querySelector<HTMLElement>('.map-container');
    expect(map).not.toBeNull();
    await act(async () => {
      map?.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', ctrlKey: true, bubbles: true }));
      await new Promise((resolve) => {
        setTimeout(resolve, 350);
      });
    });

    expect(rootBadge()).not.toBeNull();
    expect(handle.getValue()).toContain('"note": "会被撤销的"');
    // scheduleVerify（300ms）追上之后 onChange 也到了 —— 迟到不是缺席
    expect(changes[changes.length - 1] ?? '').toContain('"note": "会被撤销的"');
  });

  it('清空 textarea 后保存按不下去 —— 要清空走「移除」，两个出口不混', async () => {
    await mountReady({ initialValue: noteFile({ note: '占位的旧备注' }) });

    await act(async () => {
      rootBadge()?.click();
    });
    const popover = container.querySelector<HTMLElement>('.mm-note-popover');
    const area = popover?.querySelector<HTMLTextAreaElement>('.mm-note-input');
    expect(popover?.querySelector<HTMLButtonElement>('.mm-note-save')?.disabled).toBe(false);

    await act(async () => {
      typeIntoArea(area as HTMLTextAreaElement, '   ');
    });
    // 全是空白 = 没有可保存的内容；移除按钮还在，出路没堵死
    expect(popover?.querySelector<HTMLButtonElement>('.mm-note-save')?.disabled).toBe(true);
    expect(popover?.querySelector('.mm-note-remove')).not.toBeNull();
  });

  it('只读：徽章照显、点开是只读弹层 —— 读备注是读，不是编辑', async () => {
    await mountReady({ initialValue: noteFile({ note: '只读备注' }), readOnly: true });

    const badge = rootBadge();
    expect(badge).not.toBeNull();
    await act(async () => {
      badge?.click();
    });
    const popover = container.querySelector<HTMLElement>('.mm-note-popover');
    expect(popover).not.toBeNull();
    const area = popover?.querySelector<HTMLTextAreaElement>('.mm-note-input');
    expect(area?.value).toBe('只读备注');
    // 只读弹层：整套按钮不渲染（不是禁用）、textarea 只读
    expect(area?.readOnly).toBe(true);
    expect(popover?.querySelector('.mm-note-save')).toBeNull();
    expect(popover?.querySelector('.mm-note-remove')).toBeNull();
  });
});

describe('主题样式（ADR 0059）', () => {
  /**
   * 与前面几族不同的一条事实（先说清楚再写用例）：`style` 是格式层
   * **早已存在的正字段**（`KNOWN_NODE_KEYS` 里就有它，适配层双向直通、
   * 空对象不写），内核也照原样渲染（`Se` 把每个键塞给 CSSOM）。
   * 本轮补的是「改得了」：右键「样式…」弹层，词表住内核模块
   * （`node-style.ts`），落盘走 `reshapeNode` → `operation` → 既有 emit
   * 主通道 —— 标脏 / 保存 / undo 都不用单独伺候。
   *
   * 两条只有读过内核源码才知道的事，都在下面钉死：
   * ① `reshapeNode` 是**同步**的（`Xt`，`MindElixir.js:507-515` 里没有 await），
   *    `operation` 监听器同步跑完 `emit()` —— 补丁交出去的那一刻 `onChange`
   *    已经跑完了，所以摘键必须在补丁里就干净（值置 `undefined`，靠 JSON
   *    丢掉它），而不是交个空串等 300ms 兜底通道来补；
   * ② 数字字号 CSSOM 不认（无单位的非零长度按无效值忽略），编辑器里那一半
   *    靠 `linkDiv` 兜底补 `px`，而导出侧要的**正是数字**。
   */

  /** 根主题带可选 `style` 的规范文件；子分支 n2 恒不带（对照用）。 */
  const styleFile = (root: Record<string, unknown>): string =>
    `${JSON.stringify(
      {
        format: MINDMAP_FORMAT,
        version: MINDMAP_VERSION,
        layout: 'logicalStructure',
        root: {
          id: 'n1',
          topic: '根主题',
          children: [{ id: 'n2', topic: '分支' }],
          ...root,
        },
      },
      null,
      2,
    )}\n`;

  /** 根主题的 `me-tpc` —— 内核把 `style` 写成的内联样式就在它身上。 */
  const rootTpc = (): HTMLElement | null => container.querySelector<HTMLElement>('[data-nodeid="men1"]');

  /** 右键菜单里那条「样式…」。 */
  const styleMenuEntry = (): HTMLLIElement | null => {
    const items = [...container.querySelectorAll<HTMLLIElement>('.context-menu li')];
    return items.find((li) => li.textContent === '样式…') ?? null;
  };

  /** 右键根主题 → 点「样式…」→ 拿到弹层。 */
  const openStyleViaMenu = async (): Promise<HTMLElement> => {
    const topic = rootTpc();
    expect(topic).not.toBeNull();
    await act(async () => {
      topic?.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, button: 2 }));
      await new Promise((resolve) => {
        setTimeout(resolve, 250);
      });
    });
    const entry = styleMenuEntry();
    expect(entry).not.toBeNull();
    await act(async () => {
      entry?.click();
    });
    const popover = container.querySelector<HTMLElement>('.mm-style-popover');
    expect(popover).not.toBeNull();
    return popover as HTMLElement;
  };

  /** 文字色色块（按色值取 —— `title` 就是色值）。 */
  const textSwatch = (popover: HTMLElement, hex: string): HTMLButtonElement | null =>
    popover.querySelector<HTMLButtonElement>(`.mm-style-swatch-text[title="${hex}"]`);

  /** 底色色块。 */
  const fillSwatch = (popover: HTMLElement, hex: string): HTMLButtonElement | null =>
    popover.querySelector<HTMLButtonElement>(`.mm-style-swatch-fill[title="${hex}"]`);

  /** 字号档位按钮（加粗那枚也带 `.mm-style-chip`，按类名排掉）。 */
  const sizeChip = (popover: HTMLElement, size: number): HTMLButtonElement | null =>
    [...popover.querySelectorAll<HTMLButtonElement>('.mm-style-chip:not(.mm-style-bold)')].find(
      (button) => button.textContent === String(size),
    ) ?? null;

  const isActive = (button: HTMLButtonElement | null): boolean => button?.classList.contains('active') === true;

  /** 等内核把重画跑完（`reshapeNode` 同步，但重画链上的同步回调排在 act 之后）。 */
  const settleKernel = async (): Promise<void> => {
    await act(async () => {
      await new Promise((resolve) => {
        setTimeout(resolve, 0);
      });
    });
  };

  const TEXT_HEX = '#2f5f8f';
  const FILL_HEX = '#e3ecf3';

  it('★ 右键菜单末尾有「样式…」，点开是弹层 —— 三行词表全在（词表住内核模块）', async () => {
    await mountReady({ initialValue: styleFile({}) });

    const popover = await openStyleViaMenu();
    // 8 个文字色 + 8 个底色 + 4 档字号 + 加粗 + 清除
    expect(popover.querySelectorAll('.mm-style-swatch-text')).toHaveLength(8);
    expect(popover.querySelectorAll('.mm-style-swatch-fill')).toHaveLength(8);
    expect(popover.querySelectorAll('.mm-style-chip:not(.mm-style-bold)')).toHaveLength(4);
    expect(popover.querySelector('.mm-style-bold')).not.toBeNull();
    expect(popover.querySelector('.mm-style-clear')).not.toBeNull();
    // 一个样式都没有：一块都不亮（默认外观不是「某一档」）
    expect(popover.querySelectorAll('.mm-style-swatch.active')).toHaveLength(0);
    expect(popover.querySelectorAll('.mm-style-chip.active')).toHaveLength(0);
  });

  it('点一个文字色：落盘走 reshapeNode → operation → onChange，弹层不关、色块亮起', async () => {
    const { handle, changes } = await mountReady({ initialValue: styleFile({}) });

    const popover = await openStyleViaMenu();
    act(() => {
      textSwatch(popover, TEXT_HEX)?.click();
    });
    await settleKernel();

    const last = changes[changes.length - 1] ?? '';
    expect(last).toContain(`"color": "${TEXT_HEX}"`);
    expect(handle.getValue()).toContain(`"color": "${TEXT_HEX}"`);
    // 点色块不关弹层：一次配好几样是正常用法（标记调色盘同款）
    expect(container.querySelector('.mm-style-popover')).not.toBeNull();
    expect(isActive(textSwatch(popover, TEXT_HEX))).toBe(true);
    expect(isActive(textSwatch(popover, '#333333'))).toBe(false);
  });

  it('再点同一个色块 = 摘掉：最后一个键也摘掉时，文件里连 "style" 都不留', async () => {
    const { handle } = await mountReady({ initialValue: styleFile({ style: { color: TEXT_HEX } }) });

    const popover = await openStyleViaMenu();
    // 文件里已经有这个颜色：打开就亮着（快照来自内核的 nodeObj）
    expect(isActive(textSwatch(popover, TEXT_HEX))).toBe(true);

    act(() => {
      textSwatch(popover, TEXT_HEX)?.click();
    });
    await settleKernel();

    // 整份空了 → 整个字段撤掉（不是留下 `"style": {}`）
    expect(handle.getValue()).not.toContain('"style"');
    expect(container.querySelector('.mm-style-popover')).not.toBeNull();
    expect(isActive(textSwatch(popover, TEXT_HEX))).toBe(false);
  });

  it('摘一个键不碰另一个：色 + 字号都在时摘掉颜色，字号原样留着', async () => {
    const { handle } = await mountReady({ initialValue: styleFile({}) });

    const popover = await openStyleViaMenu();
    act(() => {
      textSwatch(popover, TEXT_HEX)?.click();
    });
    await settleKernel();
    act(() => {
      sizeChip(popover, 20)?.click();
    });
    await settleKernel();

    let text = handle.getValue();
    expect(text).toContain(`"color": "${TEXT_HEX}"`);
    expect(text).toContain('"fontSize": 20');

    act(() => {
      textSwatch(popover, TEXT_HEX)?.click();
    });
    await settleKernel();

    text = handle.getValue();
    expect(text).not.toContain('"color"');
    expect(text).toContain('"fontSize": 20');
    // 亮着的档位跟着走：字号那一档还亮着，颜色那一行灭了
    expect(isActive(sizeChip(popover, 20))).toBe(true);
    expect(isActive(textSwatch(popover, TEXT_HEX))).toBe(false);
  });

  it('底色走的是 background 键，与文字色互不干扰 —— 两行各查各的', async () => {
    const { handle } = await mountReady({ initialValue: styleFile({}) });

    const popover = await openStyleViaMenu();
    act(() => {
      fillSwatch(popover, FILL_HEX)?.click();
    });
    await settleKernel();

    expect(handle.getValue()).toContain(`"background": "${FILL_HEX}"`);
    expect(isActive(fillSwatch(popover, FILL_HEX))).toBe(true);
    expect(isActive(textSwatch(popover, FILL_HEX))).toBe(false);
  });

  it('字号写进文件的是**数字**，而屏幕上补出了 px —— 导出侧要数字、CSSOM 要单位', async () => {
    const { handle, changes } = await mountReady({ initialValue: styleFile({}) });

    const popover = await openStyleViaMenu();
    act(() => {
      sizeChip(popover, 20)?.click();
    });
    await settleKernel();

    const last = changes[changes.length - 1] ?? '';
    expect(last).toContain('"fontSize": 20');
    // 不能写成 `"20px"`：内核导出按 `${v}px` 拼，字符串会被拼成 `20pxpx` 那样的非法值
    expect(last).not.toContain('px');
    expect(handle.getValue()).not.toContain('px');
    // 编辑器这一半：`Se` 把数字交给 CSSOM 时被当无效值忽略，靠 linkDiv 兜底补 px
    expect(rootTpc()?.style.fontSize).toBe('20px');
  });

  it('加粗就是 fontWeight: bold；再点一次摘掉', async () => {
    const { handle } = await mountReady({ initialValue: styleFile({}) });

    const popover = await openStyleViaMenu();
    const bold = popover.querySelector<HTMLButtonElement>('.mm-style-bold');
    act(() => {
      bold?.click();
    });
    await settleKernel();
    expect(handle.getValue()).toContain('"fontWeight": "bold"');
    expect(isActive(bold)).toBe(true);

    act(() => {
      bold?.click();
    });
    await settleKernel();
    expect(handle.getValue()).not.toContain('"style"');
  });

  it('词表外的键照原样保留（§5）：单键编辑不碰它，只有「清除」才整份撤掉', async () => {
    const { handle } = await mountReady({ initialValue: styleFile({ style: { borderColor: '#ff0000' } }) });

    const popover = await openStyleViaMenu();
    act(() => {
      textSwatch(popover, TEXT_HEX)?.click();
    });
    await settleKernel();

    let text = handle.getValue();
    expect(text).toContain('"borderColor": "#ff0000"');
    expect(text).toContain(`"color": "${TEXT_HEX}"`);

    // 「清除」的语义是「回到默认外观」—— 用户显式要的，词表外的键也一并撤掉
    act(() => {
      popover.querySelector<HTMLButtonElement>('.mm-style-clear')?.click();
    });
    await settleKernel();

    text = handle.getValue();
    expect(text).not.toContain('"style"');
    expect(text).not.toContain('borderColor');
  });

  it('点弹层外面就关 —— 但点色块不关（一次配好几样是正常用法）', async () => {
    await mountReady({ initialValue: styleFile({}) });
    const popover = await openStyleViaMenu();

    act(() => {
      textSwatch(popover, TEXT_HEX)?.click();
    });
    expect(container.querySelector('.mm-style-popover')).not.toBeNull();

    act(() => {
      document.body.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
    });
    expect(container.querySelector('.mm-style-popover')).toBeNull();
  });

  it('只读时右键没有菜单，更没有弹层 —— 改样式是编辑，不悄悄放行', async () => {
    await mountReady({ initialValue: styleFile({ style: { color: TEXT_HEX } }), readOnly: true });

    expect(container.querySelector('.context-menu')).toBeNull();
    await act(async () => {
      rootTpc()?.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, button: 2 }));
      await new Promise((resolve) => {
        setTimeout(resolve, 250);
      });
    });
    expect(styleMenuEntry()).toBeNull();
    expect(container.querySelector('.mm-style-popover')).toBeNull();
  });

  it('undo 把样式找回来：内核走 refresh(prev) 不发 operation，补丁靠 linkDiv 重放', async () => {
    const { handle } = await mountReady({ initialValue: styleFile({ style: { fontSize: 20 } }) });
    // 挂载时补丁就该在：文件里的数字字号在屏幕上是有单位的
    expect(rootTpc()?.style.fontSize).toBe('20px');

    const popover = await openStyleViaMenu();
    act(() => {
      sizeChip(popover, 20)?.click();
    });
    await settleKernel();
    expect(handle.getValue()).not.toContain('"style"');
    expect(rootTpc()?.style.fontSize).toBe('');

    // 内核的 undo（Kn 插件监听 .map-container 的 keydown：Ctrl+Z）走 refresh(prev)
    // —— 它不发 operation，重建会把内联样式整份抹掉，靠 linkDiv 上的补丁重放。
    const map = container.querySelector<HTMLElement>('.map-container');
    expect(map).not.toBeNull();
    await act(async () => {
      map?.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', ctrlKey: true, bubbles: true }));
      await new Promise((resolve) => {
        setTimeout(resolve, 350);
      });
    });

    expect(handle.getValue()).toContain('"fontSize": 20');
    expect(rootTpc()?.style.fontSize).toBe('20px');
  });
});
