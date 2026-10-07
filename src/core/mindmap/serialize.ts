/**
 * 内存里的树 → `.mindmap` 文本（`docs/format-spec.md` §4 / §7.1）。
 *
 * ## 输出是**规范形式**，不是「原样回填」
 *
 * 这一点与 `doc-block` 不同：那边未改动的块可以直接回填 `raw`（原文切片），
 * 因为块在原文里有精确的字符区间。**JSON 做不到这件事** —— 键顺序、缩进、空行、
 * 数字字面量（`1` 与 `1.0`）都在解析时就没了。
 *
 * 所以「打开即保存不产生 diff」这条保证**不在这里**，在调用方：
 * **文档没有被编辑过就不要重写文件**（`parseMindmap` 的 `canonical` 就是给这件事用的）。
 * 序列化器只负责「一旦要写，写出来的东西是确定的」。
 *
 * ## 确定性的两条
 *
 * 1. **键顺序固定**（与 §4 给 `doc-block` 定的同一条理由）：
 *    同一份内容永远产出同样的文本，git diff 才有意义。
 *    顺序是 `format` → `version` → `layout` → 顶层未识别键（按读进来的顺序）→ `root`；
 *    节点是 `id` → `topic` → `note` → `collapsed` → `style` → 未识别键 → `children`。
 * 2. **未识别键保持读进来的相对顺序**：`parseMindmap` 按源文件的键顺序建 `extra`，
 *    而 JS 对象对字符串键保持插入顺序，所以往返不会打乱它们。
 *
 * ## 版本号按传入的值写，不改成当前版本
 *
 * 见 `types.ts` 里 `MINDMAP_VERSION` 的注释：擅自升级会让装了老版本的机器读不了。
 */

import { t } from '../i18n/index.js';
import {
  KNOWN_TOP_KEYS,
  MINDMAP_FORMAT,
  type MindmapDocument,
  type MindmapNode,
  type SerializeMindmapOptions,
} from './types.js';

const DEFAULT_INDENT = 2;

/**
 * 往「键 → 值」的列表里塞一项。
 *
 * **值为 `undefined` 时抛错，不静默丢。** `JSON.stringify` 会把值为 `undefined`
 * 的对象属性**整个略掉** —— 那是这一层最危险的一类静默丢数据：文件写出来了、
 * 保存成功了、界面上什么都没说，而字段没了。
 * 从 `JSON.parse` 出来的数据不可能有 `undefined`（JSON 里没有这个值），
 * 所以走到这里一定是**构造文档的那段代码**写错了 —— 那是 bug，不是用户处境，
 * 就该当场炸掉而不是替它掩盖。
 */
function pushEntry(entries: [string, unknown][], key: string, value: unknown, where: string): void {
  if (value === undefined) {
    throw new Error(
      t('core.mm.serUndefined', { where, key: JSON.stringify(key) }),
    );
  }
  entries.push([key, value]);
}

/**
 * 节点 → 可以直接交给 `JSON.stringify` 的普通对象。
 *
 * ## 关于「`extra` 里能不能有已知键」
 *
 * 能 —— **只要那个已知字段本身没有被写出来**。
 *
 * 这条规矩是测试逼出来的：`parseMindmap` 会把**类型不对**的 `note` / `collapsed` / `style`
 * 移进 `extra`（这样值一个都不丢）。如果这里无条件禁止已知键出现在 `extra` 里，
 * 那么一份 `"note": 42` 的文件就变成**读得进来、写不出去** —— 那比丢掉它更糟。
 *
 * 真正要防的是**重复的 JSON 键**（`JSON.parse` 只保留最后一个，又是一次静默丢数据），
 * 而重复只在「已知字段也写了」的时候才发生。所以判据是「**两边都有**才炸」：
 * `id` / `topic` / `children` 永远会被写出，它们出现在 `extra` 里一定是 bug；
 * `note` / `collapsed` / `style` 则要看对应的槽位有没有被填。
 */
function nodeToJson(node: MindmapNode): Record<string, unknown> {
  const entries: [string, unknown][] = [
    ['id', node.id],
    ['topic', node.topic],
  ];
  /**
   * 已经占掉的键。`extra` 里的键只有在**这里也占了**的时候才会撞成重复键。
   *
   * `children` 一开始就要占位，哪怕它是在最后才写出去的 —— 它**永远**会被写出，
   * 所以 `extra` 里出现 `children` 一定是重复键。（写测试时漏了这一条，
   * 因为循环跑在 push 之前，`taken` 里还没有它。）
   */
  const taken = new Set<string>(['id', 'topic', 'children']);

  if (node.note !== undefined) {
    entries.push(['note', node.note]);
    taken.add('note');
  }
  if (node.collapsed !== undefined) {
    entries.push(['collapsed', node.collapsed]);
    taken.add('collapsed');
  }
  if (node.style !== undefined) {
    entries.push(['style', node.style]);
    taken.add('style');
  }

  for (const [key, value] of Object.entries(node.extra)) {
    if (taken.has(key)) {
      throw new Error(
        t('core.mm.serDuplicateKey', { id: JSON.stringify(node.id), key: JSON.stringify(key) }),
      );
    }
    pushEntry(entries, key, value, t('core.mm.whereNode', { id: JSON.stringify(node.id) }));
  }

  entries.push(['children', node.children.map(nodeToJson)]);
  return Object.fromEntries(entries);
}

export function serializeMindmap(
  document: MindmapDocument,
  options: SerializeMindmapOptions = {},
): string {
  const indent = Math.max(0, Math.floor(options.indent ?? DEFAULT_INDENT));
  const lineEnding = options.lineEnding ?? '\n';

  const entries: [string, unknown][] = [
    ['format', MINDMAP_FORMAT],
    ['version', document.version],
    ['layout', document.layout],
  ];
  for (const [key, value] of Object.entries(document.extra)) {
    if (KNOWN_TOP_KEYS.has(key)) {
      throw new Error(
        t('core.mm.serTopKnownKey', { key: JSON.stringify(key) }),
      );
    }
    pushEntry(entries, key, value, t('core.mm.whereTop'));
  }
  entries.push(['root', nodeToJson(document.root)]);

  // `JSON.stringify` 只产出 `\n`；行尾是文件属性，由调用方指定（§4）。
  // 字符串内部的换行早被转义成 `\n` 两个字符了，所以这里的全局替换碰不到内容。
  const body = JSON.stringify(Object.fromEntries(entries), null, indent).replace(/\n/g, lineEnding);
  return `${body}${lineEnding}`;
}
