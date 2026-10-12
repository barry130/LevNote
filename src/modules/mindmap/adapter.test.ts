import type { MindElixirData } from 'mind-elixir';
import { describe, expect, it } from 'vitest';

import { parseMindmap } from '../../core/mindmap/parse.js';
import { serializeMindmap } from '../../core/mindmap/serialize.js';
import { MINDMAP_DEFAULT_LAYOUT, MINDMAP_FORMAT, MINDMAP_LAYOUTS, MINDMAP_VERSION, type MindmapDocument } from '../../core/mindmap/types.js';
import { checkKernelSafety, describeMindmapVeto, directionOf, fromKernelData, toKernelData } from './adapter.js';

// ---------------------------------------------------------------------------
// 夹具
// ---------------------------------------------------------------------------

const doc = (over: Partial<MindmapDocument> = {}): MindmapDocument => ({
  root: { id: 'n1', topic: '根', children: [], extra: {} },
  layout: MINDMAP_DEFAULT_LAYOUT,
  extra: {},
  version: MINDMAP_VERSION,
  ...over,
});

/** 内核侧最小的数据。`direction` 默认给「双向」（2），也就是我们的默认布局。 */
const kernel = (nodeData: MindElixirData['nodeData'], over: Partial<MindElixirData> = {}): MindElixirData => ({
  nodeData,
  direction: 2,
  ...over,
});

const file = (body: unknown): string => `${JSON.stringify(body, null, 2)}\n`;

// ---------------------------------------------------------------------------
// 布局 ↔ 方向
// ---------------------------------------------------------------------------

describe('布局与内核方向的映射', () => {
  /**
   * 这条测试盯的是一个**没有任何地方会报错**的错误：
   * `LAYOUT_BY_DIRECTION` 用下标查表，顺序错了只会让「向右」显示成「向左」。
   * 类型系统管不了数组顺序，所以只能靠断言钉住。
   */
  it('MINDMAP_LAYOUTS 的下标就是内核的方向编号', () => {
    MINDMAP_LAYOUTS.forEach((layout, index) => {
      expect(directionOf(layout)).toBe(index);
    });
  });

  it('默认布局落在「双向」（2），而不是内核自己的默认值「向右」（1）', () => {
    expect(directionOf(MINDMAP_DEFAULT_LAYOUT)).toBe(2);
    expect(MINDMAP_LAYOUTS[2]).toBe(MINDMAP_DEFAULT_LAYOUT);
  });

  it('认不出的布局按默认方向渲染', () => {
    expect(directionOf('treeLeft')).toBe(2);
  });

  it('每个方向都能回到自己那个布局（映射是互逆的）', () => {
    for (const layout of MINDMAP_LAYOUTS) {
      const back = fromKernelData(doc({ layout }), kernel({ id: 'n1', topic: '根', children: [] }, { direction: directionOf(layout) }));
      expect(back.layout).toBe(layout);
    }
  });
});

// ---------------------------------------------------------------------------
// 节点字段
// ---------------------------------------------------------------------------

describe('节点字段的映射', () => {
  it('id / topic / children 原样往返', () => {
    const source = doc({
      root: {
        id: 'n1',
        topic: '根',
        children: [{ id: 'n2', topic: '子', children: [], extra: {} }],
        extra: {},
      },
    });
    const back = fromKernelData(source, toKernelData(source));
    expect(back.root).toEqual(source.root);
  });

  it('collapsed 映射到 expanded，且只在一个方向写值', () => {
    const collapsed = doc({ root: { id: 'n1', topic: '根', children: [], collapsed: true, extra: {} } });
    expect(toKernelData(collapsed).nodeData.expanded).toBe(false);

    // 展开是内核的默认值 —— 写 `expanded: true` 出来只是噪声，
    // 而且会给「打开 → 保存」平白加上一次 diff
    const expanded = doc({ root: { id: 'n1', topic: '根', children: [], collapsed: false, extra: {} } });
    expect(toKernelData(expanded).nodeData.expanded).toBeUndefined();
    expect(fromKernelData(expanded, kernel({ id: 'n1', topic: '根', children: [], expanded: true })).root.collapsed).toBeUndefined();
  });

  it('expanded === false 读成 collapsed: true', () => {
    const back = fromKernelData(doc(), kernel({ id: 'n1', topic: '根', children: [], expanded: false }));
    expect(back.root.collapsed).toBe(true);
  });

  it('空 note / 空 style 不写出来', () => {
    const source = doc({ root: { id: 'n1', topic: '根', children: [], note: '', style: {}, extra: {} } });
    const data = toKernelData(source);
    expect(data.nodeData.note).toBeUndefined();
    expect(data.nodeData.style).toBeUndefined();
  });

  it('内核认识的字段走 extra 往返，一个都不丢', () => {
    // `icons` 曾经也在这一排里 —— ADR 0053 升格为正字段，搬进下面那条
    const data = kernel({
      id: 'n1',
      topic: '根',
      children: [],
      hyperLink: 'https://example.com',
      tags: ['重要'],
      branchColor: '#ff0000',
      metadata: { owner: 'me' },
    });
    const back = fromKernelData(doc(), data);

    expect(back.root.extra).toEqual({
      hyperLink: 'https://example.com',
      tags: ['重要'],
      branchColor: '#ff0000',
      metadata: { owner: 'me' },
    });

    // 交回内核时原样展开 —— 这一步漏了，「存得住但看不见」
    const round = toKernelData(back);
    expect(round.nodeData.hyperLink).toBe('https://example.com');
    expect(round.nodeData.tags).toEqual(['重要']);
    expect(round.nodeData.metadata).toEqual({ owner: 'me' });
  });

  it('icons 是正字段（ADR 0053）：文件 ↔ 内核同名直通，不进 extra、空数组照带', () => {
    const withIcons = doc({ root: { id: 'n1', topic: '根', children: [], icons: ['1️⃣'], extra: {} } });
    expect(toKernelData(withIcons).nodeData.icons).toEqual(['1️⃣']);

    const back = fromKernelData(doc(), kernel({ id: 'n1', topic: '根', children: [], icons: ['1️⃣', '⭐'] }));
    expect(back.root.icons).toEqual(['1️⃣', '⭐']);
    // 关键回归点：升格之前它掉在 extra 里；掉在 extra 里会与正字段重复写键
    expect(back.root.extra).toEqual({});

    // 「全清过」的空数组从内核拿回来还是空数组 —— serialize 会写成 `"icons": []`
    const cleared = fromKernelData(doc(), kernel({ id: 'n1', topic: '根', children: [], icons: [] }));
    expect(cleared.root.icons).toEqual([]);
  });

  it('内核自己加的 parent 与改名的 expanded 都不进 extra', () => {
    const back = fromKernelData(
      doc(),
      kernel({ id: 'n1', topic: '根', children: [], parent: { id: 'p', topic: '父' }, expanded: false }),
    );
    expect(back.root.extra).toEqual({});
    expect(back.root.collapsed).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// 顶层字段
// ---------------------------------------------------------------------------

describe('顶层字段的映射', () => {
  /**
   * 这是 `fromKernelData` **必须**拿到上一份信封的唯一原因：
   * 内核的 `getData()` 返回的是一个写死的字段列表，
   * 我们展开到它上面的自定义顶层键会原样消失。
   */
  it('内核带不回来的顶层键从上一份信封继承', () => {
    const previous = doc({ extra: { customTop: { a: 1 }, theme: { name: '用户的主题' }, compact: true } });
    const back = fromKernelData(previous, kernel({ id: 'n1', topic: '根', children: [] }));
    expect(back.extra.customTop).toEqual({ a: 1 });
    expect(back.extra.theme).toEqual({ name: '用户的主题' });
    expect(back.extra.compact).toBe(true);
  });

  it('theme 与 compact 保留，但**不交回内核**', () => {
    const previous = doc({ extra: { theme: { name: '用户的主题' }, compact: true } });
    const data = toKernelData(previous);
    expect(data.theme).toBeUndefined();
    expect(data.compact).toBeUndefined();
    // 保留的那一份没有被内核的主题覆盖掉
    expect(fromKernelData(previous, kernel({ id: 'n1', topic: '根', children: [] }, { theme: { name: '轻笺', palette: [] } })).extra.theme).toEqual({
      name: '用户的主题',
    });
  });

  it('arrows / summaries / meta 进 extra，并且交得回内核', () => {
    const arrows: NonNullable<MindElixirData['arrows']> = [{ id: 'a1', label: '依赖', from: 'n1', to: 'n2' }];
    const summaries: NonNullable<MindElixirData['summaries']> = [{ id: 's1', label: '这一段', parent: 'n1', start: 0, end: 1 }];
    const back = fromKernelData(doc(), kernel({ id: 'n1', topic: '根', children: [] }, { arrows, summaries, meta: { x: 1 } }));

    expect(back.extra.arrows).toEqual(arrows);
    expect(back.extra.summaries).toEqual(summaries);
    expect(back.extra.meta).toEqual({ x: 1 });

    const round = toKernelData(back);
    expect(round.arrows).toEqual(arrows);
    expect(round.summaries).toEqual(summaries);
    expect(round.meta).toEqual({ x: 1 });
  });

  it('用户删光箭头之后，extra 里那一项要跟着消失', () => {
    const previous = doc({ extra: { arrows: [{ id: 'a1', from: 'n1', to: 'n2' }] } });
    const back = fromKernelData(previous, kernel({ id: 'n1', topic: '根', children: [] }, { arrows: [] }));
    expect(back.extra.arrows).toBeUndefined();
  });

  it('按源文件的版本写回，不擅自升级', () => {
    const back = fromKernelData(doc({ version: 1 }), kernel({ id: 'n1', topic: '根', children: [] }));
    expect(back.version).toBe(1);
  });

  it('认不出的 layout 在方向没被动过时原样保留', () => {
    const back = fromKernelData(doc({ layout: 'treeLeft' }), kernel({ id: 'n1', topic: '根', children: [] }));
    expect(back.layout).toBe('treeLeft');
  });

  it('认不出的 layout 在方向被改过之后让位给映射值', () => {
    const back = fromKernelData(doc({ layout: 'treeLeft' }), kernel({ id: 'n1', topic: '根', children: [] }, { direction: 3 }));
    expect(back.layout).toBe('orgChart');
  });
});

// ---------------------------------------------------------------------------
// 端到端：一份真实文件走完整圈
// ---------------------------------------------------------------------------

describe('整份文件的往返', () => {
  it('解析 → 交内核 → 拿回来 → 序列化，内容一个不少', () => {
    const text = file({
      format: MINDMAP_FORMAT,
      version: MINDMAP_VERSION,
      layout: 'orgChart',
      // 更高版本写的顶层键：解析时进 extra，序列化时照原样写回（§5）
      futureTop: { note: '更高版本的东西' },
      root: {
        id: 'n1',
        topic: '根',
        children: [
          { id: 'n2', topic: '分支一', collapsed: true, children: [] },
          // 认不出的节点键同样要留住
          { id: 'n3', topic: '分支二', futureNode: 42, children: [] },
        ],
      },
    });

    const parsed = parseMindmap(text);
    expect(parsed.document).not.toBeNull();
    const first = parsed.document;
    if (first === null) return;

    const back = fromKernelData(first, toKernelData(first));
    const again = serializeMindmap(back, { lineEnding: parsed.lineEnding });

    expect(JSON.parse(again)).toEqual(JSON.parse(text));
    // 方向也回来了
    expect(back.layout).toBe('orgChart');
  });

  it('二次往返幂等（第一次规范化之后不再变化）', () => {
    const text = file({
      format: MINDMAP_FORMAT,
      version: MINDMAP_VERSION,
      root: { id: 'n1', topic: '根', children: [{ id: 'n2', topic: '子', expanded: true, children: [] }] },
    });

    const first = parseMindmap(text).document;
    expect(first).not.toBeNull();
    if (first === null) return;

    const once = fromKernelData(first, toKernelData(first));
    const twice = fromKernelData(once, toKernelData(once));
    expect(serializeMindmap(twice)).toBe(serializeMindmap(once));
  });
});

// ---------------------------------------------------------------------------
// 「这份文件能不能安全地交给内核」
// ---------------------------------------------------------------------------

describe('交给内核前的安全检查', () => {
  it('普通文件通过', () => {
    expect(checkKernelSafety(doc()).ok).toBe(true);
  });

  it('id 里有双引号会坏掉内核的属性选择器', () => {
    const result = checkKernelSafety(doc({ root: { id: 'a"b', topic: '根', children: [], extra: {} } }));
    expect(result.ok).toBe(false);
  });

  it('id 里有反斜杠同样不行', () => {
    const result = checkKernelSafety(doc({ root: { id: 'a\\b', topic: '根', children: [], extra: {} } }));
    expect(result.ok).toBe(false);
  });

  it('id 里有空格 / 井号 / 点是安全的（它们在引号里）', () => {
    for (const id of ['a b', 'a#b', 'a.b', 'a/b', 'a:b']) {
      expect(checkKernelSafety(doc({ root: { id, topic: '根', children: [], extra: {} } })).ok).toBe(true);
    }
  });

  it('id 重复要拦下来 —— querySelector 只返回第一个，拖拽会作用到别的节点上', () => {
    const result = checkKernelSafety(
      doc({
        root: {
          id: 'n1',
          topic: '根',
          children: [
            { id: 'same', topic: '甲', children: [], extra: {} },
            { id: 'same', topic: '乙', children: [], extra: {} },
          ],
          extra: {},
        },
      }),
    );
    expect(result.ok).toBe(false);
  });

  it('id 有问题的那个 id 会被点名带出来 —— 横幅上要能看出改哪一个', () => {
    // ADR 0043：理由必须是**数据**而不是翻译好的一句话，外壳要拿这个 id 去插值。
    // 只断言 `.ok === false` 的话，「横幅上写的是不是这个 id」这条链路上没有任何测试。
    const unsafe = checkKernelSafety(doc({ root: { id: 'a"b', topic: '根', children: [], extra: {} } }));
    expect(unsafe).toEqual({ ok: false, reason: { kind: 'mindmap-unsafe-id', id: 'a"b' } });

    const duplicated = checkKernelSafety(
      doc({
        root: {
          id: 'n1',
          topic: '根',
          children: [
            { id: 'same', topic: '甲', children: [], extra: {} },
            { id: 'same', topic: '乙', children: [], extra: {} },
          ],
          extra: {},
        },
      }),
    );
    expect(duplicated).toEqual({ ok: false, reason: { kind: 'mindmap-duplicate-id', id: 'same' } });
  });

  it('深层节点里的问题也能查出来', () => {
    const result = checkKernelSafety(
      doc({
        root: {
          id: 'n1',
          topic: '根',
          children: [
            {
              id: 'n2',
              topic: '子',
              children: [{ id: 'n3', topic: '孙', children: [], extra: {} }],
              extra: {},
            },
          ],
          extra: {},
        },
      }),
    );
    expect(result.ok).toBe(true);

    const bad = checkKernelSafety(
      doc({
        root: {
          id: 'n1',
          topic: '根',
          children: [{ id: 'n2', topic: '子', children: [{ id: 'bad"', topic: '孙', children: [], extra: {} }], extra: {} }],
          extra: {},
        },
      }),
    );
    expect(bad.ok).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// 否决的**理由**（ADR 0043）
// ---------------------------------------------------------------------------

describe('describeMindmapVeto —— 判定与理由同源', () => {
  const body = (over: Record<string, unknown> = {}): string =>
    JSON.stringify({ format: 'levnote.mindmap', version: 1, root: { id: 'n1', topic: '根', children: [] }, ...over });

  const node = (id: string, topic: string): Record<string, unknown> => ({ id, topic, children: [] });

  it('能画的文件没有理由', () => {
    expect(describeMindmapVeto(body())).toBeNull();
    // 空文件不是坏文件，是「还没开始写」
    expect(describeMindmapVeto('')).toBeNull();
    expect(describeMindmapVeto('\n  \n')).toBeNull();
  });

  it('读不出来的文件给出 mindmap-invalid', () => {
    expect(describeMindmapVeto('{ 这不是 JSON')).toEqual({ kind: 'mindmap-invalid' });
    expect(describeMindmapVeto(JSON.stringify({ format: '别的格式', version: 1, root: {} }))).toEqual({
      kind: 'mindmap-invalid',
    });
  });

  it('id 带引号 / 反斜杠给出 mindmap-unsafe-id，并带上那个 id', () => {
    const text = body({ root: { id: 'a\\b', topic: '根', children: [] } });
    expect(describeMindmapVeto(text)).toEqual({ kind: 'mindmap-unsafe-id', id: 'a\\b' });
  });

  it('id 重复给出 mindmap-duplicate-id', () => {
    const text = body({ root: { id: 'n1', topic: '根', children: [node('same', '甲'), node('same', '乙')] } });
    expect(describeMindmapVeto(text)).toEqual({ kind: 'mindmap-duplicate-id', id: 'same' });
  });

  /**
   * 这条是 ADR 0043 里「为什么不并入 canHandle」那条取舍的守卫。
   *
   * `canHandle` 与 `canHandleReason` 是契约上两个方法，外壳分别调用它们。
   * 只要它们各自实现一遍，就会漂移到「一个说不行、另一个说不出为什么」——
   * 表现正是横幅上写着与文件无关的话。两个方法都读同一个 `describeMindmapVeto`，
   * 这条断言就是那件事的钉子（放在 `wiring.test.ts` 里更合适，
   * 但源头在这里，就近钉住）。
   */
  it('能画的文件没有理由，且被判定的三种都各自对得上 kind', () => {
    // 三种否决各来一份，确认「为什么」与「哪一个」都没串味
    expect(describeMindmapVeto('{ 这不是 JSON')?.kind).toBe('mindmap-invalid');
    expect(describeMindmapVeto(body({ root: { id: 'a"b', topic: '根', children: [] } }))?.kind).toBe(
      'mindmap-unsafe-id',
    );
    expect(
      describeMindmapVeto(body({ root: { id: 'n1', topic: '根', children: [node('same', '甲'), node('same', '乙')] } }))
        ?.kind,
    ).toBe('mindmap-duplicate-id');
  });
});
