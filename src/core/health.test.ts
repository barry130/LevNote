import { describe, expect, it } from 'vitest';

import {
  EMPTY_HEALTH,
  activeHealth,
  activeHealthCount,
  clearHealth,
  recordIssue,
  resolveIssue,
  syncFamily,
  type HealthNote,
} from './health.js';

function note(id: string, overrides: Partial<HealthNote> = {}): HealthNote {
  return {
    id,
    area: 'storage',
    severity: 'warning',
    lines: [{ key: 'ui.health.empty', params: {} }],
    ...overrides,
  };
}

describe('recordIssue', () => {
  it('第一次记下：一次、时刻落在同一处、还没恢复', () => {
    const log = recordIssue(EMPTY_HEALTH, note('a'), 100);

    expect(log).toHaveLength(1);
    expect(log[0]).toMatchObject({ id: 'a', count: 1, firstSeenAt: 100, lastSeenAt: 100, resolvedAt: null });
  });

  it('同一个 id 再来一次是累加，不是新增一条', () => {
    const first = recordIssue(EMPTY_HEALTH, note('a'), 100);
    const second = recordIssue(first, note('a'), 200);

    expect(second).toHaveLength(1);
    expect(second[0]).toMatchObject({ count: 2, firstSeenAt: 100, lastSeenAt: 200 });
  });

  it('又发生了，刷新文案：丢的可能是另一批文档', () => {
    const first = recordIssue(
      EMPTY_HEALTH,
      note('positions', { lines: [{ key: 'ui.app.readLostPositionNames', params: { names: 'a.md' } }] }),
      100,
    );
    const second = recordIssue(
      first,
      note('positions', { lines: [{ key: 'ui.app.readLostPositionNames', params: { names: 'b.md' } }] }),
      200,
    );

    expect(second[0]?.lines).toEqual([{ key: 'ui.app.readLostPositionNames', params: { names: 'b.md' } }]);
  });

  it('已恢复的那条又发生时，恢复标记要清回去', () => {
    const first = recordIssue(EMPTY_HEALTH, note('a'), 100);
    const healed = resolveIssue(first, 'a', 150);
    const again = recordIssue(healed, note('a'), 200);

    expect(again[0]).toMatchObject({ count: 2, resolvedAt: null });
  });

  it('新的 id 排在最前面 —— 用户先看最近发生的', () => {
    const log = recordIssue(recordIssue(EMPTY_HEALTH, note('a'), 100), note('b'), 200);

    expect(log.map((record) => record.id)).toEqual(['b', 'a']);
  });

  it('超过上限只留最近的若干条（丢弃只发生在这里，不散落在别处）', () => {
    let log = EMPTY_HEALTH;
    for (let i = 0; i < 70; i += 1) log = recordIssue(log, note(`id-${i}`), i);

    expect(log).toHaveLength(60);
    expect(log[0]?.id).toBe('id-69');
    expect(log[59]?.id).toBe('id-10');
  });
});

describe('resolveIssue', () => {
  it('打一个时刻，并且保留这条记录', () => {
    const healed = resolveIssue(recordIssue(EMPTY_HEALTH, note('a'), 100), 'a', 150);

    expect(healed).toHaveLength(1);
    expect(healed[0]?.resolvedAt).toBe(150);
    expect(healed[0]?.count).toBe(1);
  });

  it('重复标记恢复不改时刻（没有新信息就别覆盖旧信息）', () => {
    const healed = resolveIssue(recordIssue(EMPTY_HEALTH, note('a'), 100), 'a', 150);

    expect(resolveIssue(healed, 'a', 999)[0]?.resolvedAt).toBe(150);
  });

  it('没有的问题不会凭空多出一条「已恢复」', () => {
    const log = recordIssue(EMPTY_HEALTH, note('a'), 100);

    expect(resolveIssue(log, 'nope', 150)).toBe(log);
  });
});

describe('activeHealth / clearHealth', () => {
  it('只有还没恢复的才算「现在还有问题」', () => {
    let log = recordIssue(EMPTY_HEALTH, note('a'), 100);
    log = recordIssue(log, note('b'), 110);
    log = resolveIssue(log, 'a', 150);

    expect(activeHealth(log).map((record) => record.id)).toEqual(['b']);
    expect(activeHealthCount(log)).toBe(1);
  });

  it('清空：本来就没有就一个引用也不换', () => {
    expect(clearHealth(EMPTY_HEALTH)).toBe(EMPTY_HEALTH);
    expect(clearHealth(recordIssue(EMPTY_HEALTH, note('a'), 100))).toBe(EMPTY_HEALTH);
  });

  it('空记录本上问什么都是零', () => {
    expect(activeHealth(EMPTY_HEALTH)).toEqual([]);
    expect(activeHealthCount(EMPTY_HEALTH)).toBe(0);
  });
});

describe('syncFamily', () => {
  const IDS = ['watch.reason', 'watch.paths'] as const;

  it('有的记上：此刻的两条问题各占一条记录', () => {
    const log = syncFamily(EMPTY_HEALTH, [note('watch.reason'), note('watch.paths')], IDS, 100);

    expect(log.map((record) => record.id)).toEqual(['watch.paths', 'watch.reason']);
    expect(activeHealthCount(log)).toBe(2);
  });

  it('同样的问题再报一次，什么都不改 —— 重复的读不是又出了一次事', () => {
    const first = syncFamily(EMPTY_HEALTH, [note('watch.reason')], IDS, 100);

    expect(syncFamily(first, [note('watch.reason')], IDS, 200)).toBe(first);
  });

  it('内容变了才算又发生一次（文案不同 = 另一件事）', () => {
    const first = syncFamily(EMPTY_HEALTH, [note('watch.reason')], IDS, 100);
    const changed = note('watch.reason', { lines: [{ key: 'ui.app.watchLostBody', params: { reason: 'x' } }] });
    const second = syncFamily(first, [changed], IDS, 200);

    expect(second[0]).toMatchObject({ count: 2, lastSeenAt: 200 });
    expect(second[0]?.lines).toEqual([{ key: 'ui.app.watchLostBody', params: { reason: 'x' } }]);
  });

  it('同一句话里参数的键序不影响判重（数据不是结构）', () => {
    const first = syncFamily(
      EMPTY_HEALTH,
      [note('a', { lines: [{ key: 'ui.app.watchLostBody', params: { reason: 'x', n: 1 } }] })],
      ['a'],
      100,
    );
    const shuffled = syncFamily(
      first,
      [note('a', { lines: [{ key: 'ui.app.watchLostBody', params: { n: 1, reason: 'x' } }] })],
      ['a'],
      200,
    );

    expect(shuffled).toBe(first);
  });

  it('文案行数变了就是另一件事', () => {
    const first = syncFamily(EMPTY_HEALTH, [note('a', { lines: [{ key: 'ui.health.empty', params: {} }] })], ['a'], 100);
    const longer = syncFamily(
      first,
      [note('a', { lines: [{ key: 'ui.health.empty', params: {} }, { key: 'ui.health.intro', params: {} }] })],
      ['a'],
      200,
    );

    expect(longer[0]?.count).toBe(2);
  });

  it('消失的那条标成已恢复：横幅消失的那一刻，清单里要看得见', () => {
    const full = syncFamily(EMPTY_HEALTH, [note('watch.reason'), note('watch.paths')], IDS, 100);
    const healed = syncFamily(full, [note('watch.reason')], IDS, 300);

    expect(healed.find((record) => record.id === 'watch.paths')?.resolvedAt).toBe(300);
    expect(healed.find((record) => record.id === 'watch.reason')?.resolvedAt).toBeNull();
  });

  it('全好了之后又坏起来：旧的恢复标记要清回去，而且算第二次', () => {
    const full = syncFamily(EMPTY_HEALTH, [note('watch.reason')], IDS, 100);
    const healed = syncFamily(full, [], IDS, 200);
    const broken = syncFamily(healed, [note('watch.reason')], IDS, 300);

    expect(broken[0]).toMatchObject({ count: 2, resolvedAt: null, lastSeenAt: 300 });
  });

  it('整族都没问题时，家族之外的记录一条也不许碰', () => {
    const withDrafts = recordIssue(EMPTY_HEALTH, note('session.drafts'), 100);
    const healed = syncFamily(withDrafts, [], IDS, 200);

    expect(healed.find((record) => record.id === 'session.drafts')?.resolvedAt).toBeNull();
  });

  it('一帧里什么都没变就一个引用也不换', () => {
    const log = recordIssue(EMPTY_HEALTH, note('other'), 100);

    expect(syncFamily(log, [], IDS, 999)).toBe(log);
  });
});