/**
 * 极简 LRU 缓存。
 *
 * ## 为什么不直接用 `Map` 当 FIFO
 *
 * 这个缓存的调用方是 `highlightCode`，它的工作负载有一个很具体的形状：
 * **少数几个 key 反复命中，同时有一批 key 源源不断地产生且再也不会被读**。
 *
 * 具体到编辑器里 —— 打字时「正在编辑的那个代码块」每敲一个字就产生一个新 key
 * （它本来就该重新高亮），而文档里其余十几个代码块的内容一动不动。
 *
 * FIFO 在这种情况下会退化：新 key 把稳定 key 一个个挤出队头，
 * 打到第 N 个字时缓存里全是没人会再看的旧版本，稳定块反而 miss。
 * 命中后把条目挪到队尾，稳定块就永远沉不到队头 —— 被淘汰的只会是旧版本。
 *
 * ## 实现
 *
 * `Map` 的迭代顺序就是插入顺序，所以「队头 = 最久未使用」是天然成立的：
 * 命中时 `delete` + `set` 把条目挪到队尾，超限时删第一个 key。
 * 全部操作 O(1)，没有链表、没有堆。
 *
 * ## 与「按使用频率淘汰」的区别
 *
 * LFU 在这个负载下未必更好，而它要维护计数与堆。这里的访问模式是
 * 「稳定块持续命中、临时 key 只读一次」，LRU 已经能精确地把后者挑出来。
 */

/**
 * ## 一条接口限制
 *
 * `get` 用 `undefined` 表示「没有这条」，所以 **`V` 不应包含 `undefined`** ——
 * 否则「没命中」与「命中的值就是 `undefined`」无法区分。
 * 需要存可空值的话，把 `undefined` 包一层（如 `{ value?: T }`）再存。
 */
export interface LruCache<V> {
  /** 取一个值。命中时该条目会被标记为「最近使用」。 */
  get(key: string): V | undefined;
  /** 放一个值。超限时淘汰最久未使用的那条。 */
  set(key: string, value: V): void;
  has(key: string): boolean;
  /** 当前条目数。主要给测试与调试用。 */
  readonly size: number;
}

/**
 * 建一个容量为 `limit` 的 LRU 缓存。
 *
 * `limit` 必须 ≥ 1 —— 容量为 0 的缓存意味着「每次 `set` 都立刻淘汰」，
 * 那不是一个有用的东西，把它当成调用方的错误直接拒绝掉，
 * 好过让它在运行期表现为「缓存永远不命中」。
 */
export function createLruCache<V>(limit: number): LruCache<V> {
  if (!Number.isInteger(limit) || limit < 1) {
    throw new RangeError(`LRU 缓存容量必须是 ≥1 的整数，收到 ${String(limit)}`);
  }

  const entries = new Map<string, V>();

  return {
    get size(): number {
      return entries.size;
    },

    get(key: string): V | undefined {
      const hit = entries.get(key);
      if (hit === undefined) return undefined;
      // 挪到队尾 = 标记为最近使用。先删再插，因为 Map 的插入顺序不可变更。
      entries.delete(key);
      entries.set(key, hit);
      return hit;
    },

    set(key: string, value: V): void {
      // 同一个 key 再写一次也算「最近使用」，所以先删掉旧位置
      entries.delete(key);
      if (entries.size >= limit) {
        // 迭代顺序的第一个就是最久未使用的那条
        const oldest = entries.keys().next().value;
        if (oldest !== undefined) entries.delete(oldest);
      }
      entries.set(key, value);
    },

    has(key: string): boolean {
      return entries.has(key);
    },
  };
}
