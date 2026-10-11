import { describe, expect, it } from 'vitest';

import {
  EMPTY_POSITIONS,
  forgetPosition,
  loadPositions,
  MAX_REMEMBERED_POSITIONS,
  rememberPosition,
  savePositions,
} from './positions.js';
import type { StorageLike } from './settings.js';

function memoryStorage(seed?: string): StorageLike & { value: string | null } {
  const box = {
    value: seed ?? null,
    getItem(): string | null {
      return box.value;
    },
    setItem(_key: string, value: string): void {
      box.value = value;
    },
  };
  return box;
}

/** 存储抛错的实现：隐私模式 / 配额用尽 / 企业策略禁用。 */
const hostileStorage: StorageLike = {
  getItem() {
    throw new Error('SecurityError');
  },
  setItem() {
    throw new Error('QuotaExceededError');
  },
};

const SAMPLE = { offset: 120, textLength: 4000, scrollRatio: 0.25 };

describe('loadPositions', () => {
  it('没有存储 / 空存储都给空表', () => {
    expect(loadPositions(null)).toEqual(EMPTY_POSITIONS);
    expect(loadPositions(memoryStorage())).toEqual(EMPTY_POSITIONS);
    expect(loadPositions(memoryStorage(''))).toEqual(EMPTY_POSITIONS);
  });

  it('存储抛错时不炸 —— 位置记不住不该让应用起不来', () => {
    expect(loadPositions(hostileStorage)).toEqual(EMPTY_POSITIONS);
  });

  it('内容不是 JSON 时回落到空表', () => {
    expect(loadPositions(memoryStorage('{不是 json'))).toEqual(EMPTY_POSITIONS);
  });

  it('顶层不是对象时回落到空表', () => {
    expect(loadPositions(memoryStorage('[1,2,3]'))).toEqual(EMPTY_POSITIONS);
    expect(loadPositions(memoryStorage('"abc"'))).toEqual(EMPTY_POSITIONS);
    expect(loadPositions(memoryStorage('null'))).toEqual(EMPTY_POSITIONS);
  });

  it('坏条目整条丢掉，好条目保留', () => {
    // 与 settings 的逐字段回落刻意相反：拿一个 offset:0 配上正确的 textLength
    // 去恢复，等于把光标悄悄移到文档开头，用户会以为文件被改了
    const storage = memoryStorage(
      JSON.stringify({
        'F:/a.md': SAMPLE,
        'F:/b.md': { offset: 'x', textLength: 10, scrollRatio: 0 },
        'F:/c.md': { offset: 1, textLength: 10 },
        'F:/d.md': { offset: 1, textLength: 10, scrollRatio: 0, at: 5 },
      }),
    );
    expect(Object.keys(loadPositions(storage)).sort()).toEqual(['F:/a.md', 'F:/d.md']);
  });

  it('负数偏移 / 负数长度算坏条目', () => {
    const storage = memoryStorage(
      JSON.stringify({ 'F:/a.md': { offset: -1, textLength: 10, scrollRatio: 0 } }),
    );
    expect(loadPositions(storage)).toEqual(EMPTY_POSITIONS);
  });

  it('比例越界时被夹住而不是丢掉整条', () => {
    // 比例只影响滚动，夹一下仍然可用；丢掉整条就白丢了光标位置
    const storage = memoryStorage(JSON.stringify({ 'F:/a.md': { offset: 1, textLength: 10, scrollRatio: 9 } }));
    expect(loadPositions(storage)['F:/a.md']?.scrollRatio).toBe(1);
  });

  it('缺少 at 的旧记录当作 0（最旧），不丢', () => {
    const storage = memoryStorage(JSON.stringify({ 'F:/a.md': { offset: 1, textLength: 10, scrollRatio: 0 } }));
    expect(loadPositions(storage)['F:/a.md']?.at).toBe(0);
  });

  it('空路径的条目被忽略', () => {
    const storage = memoryStorage(JSON.stringify({ '': SAMPLE }));
    expect(loadPositions(storage)).toEqual(EMPTY_POSITIONS);
  });
});

describe('savePositions', () => {
  it('往返一致', () => {
    const storage = memoryStorage();
    const positions = rememberPosition(EMPTY_POSITIONS, 'F:/a.md', SAMPLE, 1000);
    savePositions(storage, positions);
    expect(loadPositions(storage)).toEqual(positions);
  });

  it('存储抛错时不抛出来', () => {
    expect(() => savePositions(hostileStorage, EMPTY_POSITIONS)).not.toThrow();
  });

  it('写被拒时返回 write-failed —— 位置是每 600ms 写一次的，静默就等于一直失败没人知道（ADR 0042）', () => {
    expect(savePositions(hostileStorage, EMPTY_POSITIONS)).toEqual({ ok: false, reason: 'write-failed' });
  });

  it('没有存储时返回 no-storage：那是预期内的降级，不该说话', () => {
    expect(savePositions(null, EMPTY_POSITIONS)).toEqual({ ok: false, reason: 'no-storage' });
  });

  it('写成功时返回 ok', () => {
    expect(savePositions(memoryStorage(), EMPTY_POSITIONS)).toEqual({ ok: true });
  });

  it('写进的是自己的键，不碰设置那个键', () => {
    // 两者生命周期不同：位置每次光标停顿都写，设置只在用户改的时候写
    const keys: string[] = [];
    const spy: StorageLike = {
      getItem: () => null,
      setItem(key) {
        keys.push(key);
      },
    };
    savePositions(spy, EMPTY_POSITIONS);
    expect(keys).toEqual(['levnote.positions.v1']);
  });
});

describe('rememberPosition', () => {
  it('新增一条', () => {
    const next = rememberPosition(EMPTY_POSITIONS, 'F:/a.md', SAMPLE, 1);
    expect(next['F:/a.md']).toEqual({ ...SAMPLE, at: 1 });
  });

  it('不改原对象（返回新表）', () => {
    const before = rememberPosition(EMPTY_POSITIONS, 'F:/a.md', SAMPLE, 1);
    const after = rememberPosition(before, 'F:/b.md', SAMPLE, 2);
    expect(Object.keys(before)).toEqual(['F:/a.md']);
    expect(Object.keys(after).sort()).toEqual(['F:/a.md', 'F:/b.md']);
  });

  it('同一路径再记一次是覆盖', () => {
    const first = rememberPosition(EMPTY_POSITIONS, 'F:/a.md', SAMPLE, 1);
    const second = rememberPosition(first, 'F:/a.md', { ...SAMPLE, offset: 999 }, 2);
    expect(Object.keys(second)).toEqual(['F:/a.md']);
    expect(second['F:/a.md']?.offset).toBe(999);
    expect(second['F:/a.md']?.at).toBe(2);
  });

  it('空路径被忽略', () => {
    expect(rememberPosition(EMPTY_POSITIONS, '', SAMPLE, 1)).toBe(EMPTY_POSITIONS);
  });

  it('比例写进去之前先夹住', () => {
    const next = rememberPosition(EMPTY_POSITIONS, 'F:/a.md', { ...SAMPLE, scrollRatio: 3 }, 1);
    expect(next['F:/a.md']?.scrollRatio).toBe(1);
  });

  it('超过上限时丢掉最旧的那些', () => {
    let map = EMPTY_POSITIONS;
    for (let i = 0; i < MAX_REMEMBERED_POSITIONS + 20; i += 1) {
      map = rememberPosition(map, `F:/doc-${i}.md`, SAMPLE, i);
    }
    const keys = Object.keys(map);
    expect(keys).toHaveLength(MAX_REMEMBERED_POSITIONS);
    // 最旧的 20 条（at = 0..19）被淘汰，最新的一条还在
    expect(map['F:/doc-0.md']).toBeUndefined();
    expect(map[`F:/doc-${MAX_REMEMBERED_POSITIONS + 19}.md`]).toBeDefined();
  });

  it('一次能丢掉超量的多条（旧版本留下的超大表也能收敛）', () => {
    const oversized: Record<string, unknown> = {};
    for (let i = 0; i < MAX_REMEMBERED_POSITIONS * 3; i += 1) {
      oversized[`F:/old-${i}.md`] = { ...SAMPLE, at: i };
    }
    const map = rememberPosition(oversized as never, 'F:/new.md', SAMPLE, 999_999);
    expect(Object.keys(map)).toHaveLength(MAX_REMEMBERED_POSITIONS);
    expect(map['F:/new.md']).toBeDefined();
  });
});

describe('forgetPosition', () => {
  it('删掉一条', () => {
    const before = rememberPosition(EMPTY_POSITIONS, 'F:/a.md', SAMPLE, 1);
    expect(forgetPosition(before, 'F:/a.md')).toEqual(EMPTY_POSITIONS);
  });

  it('删不存在的路径时返回原对象（调用方据此跳过写盘）', () => {
    const before = rememberPosition(EMPTY_POSITIONS, 'F:/a.md', SAMPLE, 1);
    expect(forgetPosition(before, 'F:/b.md')).toBe(before);
  });
});
