import { describe, expect, it } from 'vitest';

import {
  EMPTY_SESSION,
  MAX_DRAFT_CHARS,
  MAX_SESSION_TABS,
  clearSession,
  fitSession,
  loadSession,
  sameSession,
  saveSession,
  type SessionState,
  type SessionTab,
} from './session.js';
import type { StorageLike } from './settings.js';

function memoryStorage(initial: Record<string, string> = {}): StorageLike & { data: Record<string, string> } {
  const data: Record<string, string> = { ...initial };
  return {
    data,
    getItem: (key) => data[key] ?? null,
    setItem: (key, value) => {
      data[key] = value;
    },
  };
}

const blocked: StorageLike = {
  getItem: () => {
    throw new Error('blocked');
  },
  setItem: () => {
    throw new Error('blocked');
  },
};

const KEY = 'levnote.session.v1';

function tab(path: string, draft: string | null = null, at = 0): SessionTab {
  return { path, draft, at };
}

function session(tabs: SessionTab[], extra: Partial<SessionState> = {}): SessionState {
  return { tabs, activePath: null, root: null, droppedDrafts: 0, ...extra };
}

describe('loadSession · 读不出来时的回落', () => {
  it('没有存储、没有键、空串都回落到空会话', () => {
    expect(loadSession(null)).toEqual(EMPTY_SESSION);
    expect(loadSession(memoryStorage())).toEqual(EMPTY_SESSION);
    expect(loadSession(memoryStorage({ [KEY]: '' }))).toEqual(EMPTY_SESSION);
  });

  it('不是 JSON、不是对象、tabs 不是数组都回落到空会话', () => {
    expect(loadSession(memoryStorage({ [KEY]: '{ 坏的' }))).toEqual(EMPTY_SESSION);
    expect(loadSession(memoryStorage({ [KEY]: '"字符串"' }))).toEqual(EMPTY_SESSION);
    expect(loadSession(memoryStorage({ [KEY]: '{"tabs":42}' }))).toEqual(EMPTY_SESSION);
  });

  it('存储直接抛错（隐私模式 / 企业策略）也不炸', () => {
    expect(loadSession(blocked)).toEqual(EMPTY_SESSION);
  });

  it('坏内容**不清空** —— 下一次写自然会覆盖，清空反而多一次失败机会', () => {
    const storage = memoryStorage({ [KEY]: '{ 坏的' });
    loadSession(storage);
    expect(storage.data[KEY]).toBe('{ 坏的');
  });
});

describe('loadSession · 逐条校验', () => {
  it('正常的往返读回来一样', () => {
    const storage = memoryStorage();
    const state = session([tab('a.md', '草稿', 100)], { activePath: 'a.md', root: 'C:/ws' });
    saveSession(storage, state);

    const back = loadSession(storage);
    expect(back.tabs).toEqual([tab('a.md', '草稿', 100)]);
    expect(back.activePath).toBe('a.md');
    expect(back.root).toBe('C:/ws');
  });

  it('路径不合法的**整条丢掉**，但其余条目照常活着', () => {
    const storage = memoryStorage({
      [KEY]: JSON.stringify({
        tabs: [{ path: '' }, { path: 42 }, null, { path: 'ok.md', draft: '留着', at: 1 }],
      }),
    });
    const back = loadSession(storage);
    expect(back.tabs).toEqual([tab('ok.md', '留着', 1)]);
    // 这种是存储被手改坏的异常情形，路径都没了，告诉用户也做不了任何事
    expect(back.droppedDrafts).toBe(0);
  });

  it('草稿不是字符串时**只丢草稿、保住路径**，并计入 droppedDrafts', () => {
    const storage = memoryStorage({
      [KEY]: JSON.stringify({ tabs: [{ path: 'a.md', draft: 42, at: 1 }, { path: 'b.md', draft: '好' }] }),
    });
    const back = loadSession(storage);
    expect(back.tabs).toEqual([tab('a.md', null, 1), tab('b.md', '好', 0)]);
    expect(back.droppedDrafts).toBe(1);
  });

  it('draft 是 null 是合法值（表示没有未保存改动），不算坏', () => {
    const storage = memoryStorage({ [KEY]: JSON.stringify({ tabs: [{ path: 'a.md', draft: null }] }) });
    const back = loadSession(storage);
    expect(back.tabs).toEqual([tab('a.md', null, 0)]);
    expect(back.droppedDrafts).toBe(0);
  });

  it('同一个路径出现两次只留第一个 —— 两处各改一半会让保存时的乐观锁冲突变得无法理解', () => {
    const storage = memoryStorage({
      [KEY]: JSON.stringify({ tabs: [{ path: 'a.md', draft: '先' }, { path: 'a.md', draft: '后' }] }),
    });
    expect(loadSession(storage).tabs).toEqual([tab('a.md', '先', 0)]);
  });

  it('activePath 不在 tabs 里时忽略它（指向一个不存在的标签页没有意义）', () => {
    const storage = memoryStorage({
      [KEY]: JSON.stringify({ tabs: [{ path: 'a.md' }], activePath: 'ghost.md' }),
    });
    expect(loadSession(storage).activePath).toBeNull();
  });

  it('root 是空串时当作没打开文件夹', () => {
    const storage = memoryStorage({ [KEY]: JSON.stringify({ tabs: [], root: '' }) });
    expect(loadSession(storage).root).toBeNull();
  });

  it('存储里记着「上次丢过 N 条」时累加进来 —— 用户还没被告知过', () => {
    const storage = memoryStorage({
      [KEY]: JSON.stringify({ tabs: [{ path: 'a.md', draft: 42 }], droppedDrafts: 3 }),
    });
    // 1 条草稿不合法 + 存储里记着的 3 条
    expect(loadSession(storage).droppedDrafts).toBe(4);
  });
});

describe('fitSession · 容量裁剪', () => {
  it('装得下就原样返回，一条都不丢', () => {
    const state = session([tab('a.md', '草稿', 1)]);
    const { state: fitted, droppedDrafts } = fitSession(state);
    expect(fitted.tabs).toEqual([tab('a.md', '草稿', 1)]);
    expect(droppedDrafts).toBe(0);
  });

  it('不改入参（纯函数）—— 外壳拿旧对象做比较，就地改会让「有没有变」看不出来', () => {
    const state = session([tab('a.md', '草稿', 1)]);
    const snapshot = JSON.stringify(state);
    fitSession(state);
    expect(JSON.stringify(state)).toBe(snapshot);
  });

  it('标签页数超上限时丢**最旧**的（按最后编辑时间），并统计丢掉了几份草稿', () => {
    const tabs = Array.from({ length: MAX_SESSION_TABS + 2 }, (_unused, index) =>
      // at 越大越新；index 0 与 1 最旧
      tab(`f${index}.md`, `草稿${index}`, index),
    );
    const { state: fitted, droppedDrafts } = fitSession(session(tabs));

    expect(fitted.tabs).toHaveLength(MAX_SESSION_TABS);
    expect(fitted.tabs.some((item) => item.path === 'f0.md')).toBe(false);
    expect(fitted.tabs.some((item) => item.path === 'f1.md')).toBe(false);
    expect(droppedDrafts).toBe(2);
  });

  it('草稿总量超上限时**只把草稿置 null、路径留着** —— 还能点开那个文件', () => {
    const big = 'x'.repeat(MAX_DRAFT_CHARS);
    const tabs = [tab('old.md', big, 1), tab('new.md', big, 2)];
    const { state: fitted, droppedDrafts } = fitSession(session(tabs));

    // 丢最旧的：old.md 的草稿没了，但路径还在；new.md 完全保留
    expect(fitted.tabs).toEqual([tab('old.md', null, 1), tab('new.md', big, 2)]);
    expect(droppedDrafts).toBe(1);
  });

  it('总量**恰好等于**上限时不裁', () => {
    const half = 'x'.repeat(MAX_DRAFT_CHARS / 2);
    const { droppedDrafts } = fitSession(session([tab('a.md', half, 1), tab('b.md', half, 2)]));
    expect(droppedDrafts).toBe(0);
  });

  it('没有草稿的条目不会让总量虚高，也不会被当成「可裁的」', () => {
    const big = 'x'.repeat(MAX_DRAFT_CHARS + 1);
    const tabs = [tab('empty.md', null, 1), tab('big.md', big, 2)];
    const { state: fitted, droppedDrafts } = fitSession(session(tabs));

    expect(fitted.tabs).toEqual([tab('empty.md', null, 1), tab('big.md', null, 2)]);
    // 只丢了 big.md 那一份；empty.md 本来就没有草稿
    expect(droppedDrafts).toBe(1);
  });
});

describe('saveSession · 写盘与失败', () => {
  it('写成功时 ok 为 true，内容能读回来', () => {
    const storage = memoryStorage();
    const outcome = saveSession(storage, session([tab('a.md', '草稿', 1)]));
    expect(outcome.ok).toBe(true);
    expect(loadSession(storage).tabs).toEqual([tab('a.md', '草稿', 1)]);
  });

  it('内部先裁剪，所以调用方给什么都装得下', () => {
    const storage = memoryStorage();
    const big = 'x'.repeat(MAX_DRAFT_CHARS + 1);
    const outcome = saveSession(storage, session([tab('a.md', big, 1)]));

    expect(outcome.ok).toBe(true);
    expect(outcome.state.tabs[0]?.draft).toBeNull();
    expect(outcome.state.droppedDrafts).toBe(1);
    expect(loadSession(storage).droppedDrafts).toBe(1);
  });

  it('返回的是**实际写下去的那一份** —— 否则下一轮比较会认为又有变化，反复重写', () => {
    const storage = memoryStorage();
    const outcome = saveSession(storage, session([tab('a.md', 'x')]));
    expect(loadSession(storage)).toEqual(outcome.state);
  });

  it('droppedDrafts 是**累加**不是覆盖 —— 否则「先丢 2 条、再丢 3 条」最后只提示 3 条', () => {
    const storage = memoryStorage();
    const first = saveSession(storage, session([], { droppedDrafts: 2 }));
    expect(first.state.droppedDrafts).toBe(2);

    const big = 'x'.repeat(MAX_DRAFT_CHARS + 1);
    const second = saveSession(storage, session([tab('a.md', big, 1)], { droppedDrafts: first.state.droppedDrafts }));
    expect(second.state.droppedDrafts).toBe(3);
  });

  it('存储抛错时 ok 为 false 且**不抛** —— 会话记不住不该让应用起不来', () => {
    const outcome = saveSession(blocked, session([tab('a.md', '草稿')]));
    expect(outcome.ok).toBe(false);
    // 但内容仍在返回值里，外壳还能拿它做提示
    expect(outcome.state.tabs).toHaveLength(1);
  });

  it('没有存储时 ok 为 false（浏览器预览之外的环境）', () => {
    expect(saveSession(null, EMPTY_SESSION).ok).toBe(false);
  });
});

describe('sameSession · 值不值得重写一遍', () => {
  it('同一个对象、以及内容相同的两份都算相同', () => {
    const a = session([tab('a.md', '草稿', 1)], { activePath: 'a.md', root: 'C:/ws' });
    expect(sameSession(a, a)).toBe(true);
    expect(sameSession(a, session([tab('a.md', '草稿', 1)], { activePath: 'a.md', root: 'C:/ws' }))).toBe(true);
  });

  it('草稿变了就不相同（这是唯一真正需要重写的情形）', () => {
    const a = session([tab('a.md', '旧', 1)]);
    expect(sameSession(a, session([tab('a.md', '新', 1)]))).toBe(false);
  });

  it('草稿从有到无（保存了）、从无到有（第一次改）都不相同', () => {
    expect(sameSession(session([tab('a.md', '草稿', 1)]), session([tab('a.md', null, 1)]))).toBe(false);
    expect(sameSession(session([tab('a.md', null, 1)]), session([tab('a.md', '草稿', 1)]))).toBe(false);
  });

  it('activePath / root 变了就不相同', () => {
    const base = session([tab('a.md'), tab('b.md')], { activePath: 'a.md', root: 'C:/ws' });
    expect(sameSession(base, { ...base, activePath: 'b.md' })).toBe(false);
    expect(sameSession(base, { ...base, root: null })).toBe(false);
  });

  it('droppedDrafts 参与比较 —— 读出来之后那一次写盘正好把旧账目清掉', () => {
    const base = session([tab('a.md')]);
    expect(sameSession(base, { ...base, droppedDrafts: 2 })).toBe(false);
  });

  it('标签页数量与**顺序**都参与比较 —— 顺序不比的话拖出来的排列下次启动会回到旧样子', () => {
    const a = session([tab('a.md', null, 1), tab('b.md', null, 2)]);
    expect(sameSession(a, session([tab('a.md', null, 1)]))).toBe(false);
    expect(sameSession(a, session([tab('b.md', null, 2), tab('a.md', null, 1)]))).toBe(false);
  });

  it('`at` 变了也算不相同 —— 容量裁剪的顺序靠它', () => {
    const a = session([tab('a.md', '草稿', 1)]);
    expect(sameSession(a, session([tab('a.md', '草稿', 2)]))).toBe(false);
  });
});

describe('clearSession', () => {
  it('清空之后读回来是空会话', () => {
    const storage = memoryStorage();
    saveSession(storage, session([tab('a.md', '草稿', 1)], { activePath: 'a.md' }));
    clearSession(storage);
    expect(loadSession(storage)).toEqual(EMPTY_SESSION);
  });

  it('存储抛错时不炸', () => {
    expect(() => clearSession(blocked)).not.toThrow();
    expect(() => clearSession(null)).not.toThrow();
  });
});
