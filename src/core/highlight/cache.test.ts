import { describe, expect, it } from 'vitest';

import { createLruCache } from './cache.js';

describe('createLruCache', () => {
  it('没放过的 key 取出来是 undefined', () => {
    const cache = createLruCache<number>(4);
    expect(cache.get('nope')).toBeUndefined();
    expect(cache.has('nope')).toBe(false);
  });

  it('放进去能取出来，size 跟着变', () => {
    const cache = createLruCache<number>(4);
    cache.set('a', 1);
    expect(cache.get('a')).toBe(1);
    expect(cache.has('a')).toBe(true);
    expect(cache.size).toBe(1);
  });

  it('没超限时不淘汰任何东西', () => {
    const cache = createLruCache<number>(3);
    cache.set('a', 1);
    cache.set('b', 2);
    cache.set('c', 3);

    expect(cache.size).toBe(3);
    expect([cache.get('a'), cache.get('b'), cache.get('c')]).toEqual([1, 2, 3]);
  });

  it('超限时淘汰最久未使用的那条', () => {
    const cache = createLruCache<number>(3);
    cache.set('a', 1);
    cache.set('b', 2);
    cache.set('c', 3);
    cache.set('d', 4);

    expect(cache.has('a')).toBe(false);
    expect(cache.size).toBe(3);
    expect([cache.get('b'), cache.get('c'), cache.get('d')]).toEqual([2, 3, 4]);
  });

  /**
   * 这条是**整个 LRU 存在的理由**，也是它与 FIFO 的唯一区别。
   *
   * 没有它，`highlightCode` 的缓存会退化成 FIFO：打字时「正在编辑的那个块」
   * 源源不断地产生新 key，把文档里其余十几个内容没变的块一个个挤出去，
   * 于是整篇重跑又变回每次都重算 —— 缓存看起来装了、其实没起作用。
   */
  it('命中会把条目刷新为「最近使用」，于是它不会被淘汰', () => {
    const cache = createLruCache<number>(3);
    cache.set('a', 1);
    cache.set('b', 2);
    cache.set('c', 3);

    // 碰一下 a：此时 a 是最新的，b 才是最久未用的
    expect(cache.get('a')).toBe(1);

    cache.set('d', 4);

    expect(cache.has('a')).toBe(true);
    expect(cache.has('b')).toBe(false);
  });

  it('对同一个 key 再写一次也算「最近使用」', () => {
    const cache = createLruCache<number>(3);
    cache.set('a', 1);
    cache.set('b', 2);
    cache.set('c', 3);

    cache.set('a', 10); // 覆盖写：既换值，也刷新位置
    cache.set('d', 4);

    expect(cache.get('a')).toBe(10);
    expect(cache.has('b')).toBe(false);
  });

  it('反复命中的「稳定 key」永远沉不到队头', () => {
    const cache = createLruCache<number>(3);
    cache.set('stable', 0);

    // 模拟打字：不断产生只读一次的临时 key，同时每轮都读一次 stable
    for (let i = 0; i < 50; i += 1) {
      cache.set(`transient-${String(i)}`, i);
      expect(cache.get('stable')).toBe(0);
    }

    expect(cache.has('stable')).toBe(true);
  });

  it('容量为 1 时只留最后一条', () => {
    const cache = createLruCache<number>(1);
    cache.set('a', 1);
    cache.set('b', 2);

    expect(cache.has('a')).toBe(false);
    expect(cache.get('b')).toBe(2);
    expect(cache.size).toBe(1);
  });

  it('容量非法时直接抛错，而不是安静地永不命中', () => {
    expect(() => createLruCache(0)).toThrow(RangeError);
    expect(() => createLruCache(-1)).toThrow(RangeError);
    expect(() => createLruCache(1.5)).toThrow(RangeError);
    expect(() => createLruCache(Number.NaN)).toThrow(RangeError);
  });
});
