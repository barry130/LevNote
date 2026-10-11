/**
 * 新建 `.mindmap` 时落盘的那份**模板**（ADR 0048）。
 *
 * ## 为什么需要这一层
 *
 * `.mindmap` 的内核（`modules/mindmap/`）一直是一等公民：单开一个 `.mindmap`，
 * 导图画布就会接管。但**能打开不等于能新建** ——
 * 此前唯一的「新建」是 `file.new`，它落的是一个**空文本文件**，
 * 侧栏的「新建文件」同理（`WorkspaceStore.createEntry` → `fileManager.createFile`）。
 *
 * 结果是：**这个类型在产品里没有入口。** 用户得先自己去记事本造一个
 * 合法的 `.mindmap`（还得知道 `format` / `version` / `layout` / `root` 这几个键），
 * 才可能打开一次导图画布。能力在，路径不通。
 *
 * ## 为什么模板是**生成**的而不是**抄一段字面量**
 *
 * 一段抄来的 JSON 字面量会跟格式漂移：改了 `MINDMAP_VERSION`、改了默认布局、
 * 改了 `KNOWN_NODE_KEYS`，模板不会跟着动，于是新文件一落盘就是「旧版格式」，
 * 还得靠解析器的告警去兜。**生成的模板永远等于当前格式**——
 * 这也是 `types.ts` 顶上那条纪律（§7.1 结构自持）在新建路径上的延续。
 *
 * 顺带满足 `ParseMindmapResult.canonical` 那条约定：
 * 模板产出的文本经解析器往返后 `canonical === true`，
 * 所以用户「打开 → 一个字没改 → 保存」不产生 diff。
 */

import {
  MINDMAP_DEFAULT_LAYOUT,
  MINDMAP_VERSION,
  type MindmapDocument,
  type MindmapNode,
  type SerializeMindmapOptions,
} from './types.js';
import { serializeMindmap } from './serialize.js';

/**
 * 中心节点的 id。
 *
 * 固定值而不是随机 / 时间戳：`.mindmap` 的 `id` 会出现在文件里，
 * 随机 id 让「同一个新文件」每次生成都不一样，也让测试没法断言。
 * 而根节点本来只有一个，id 唯一是天然成立的。
 *
 * 合法性按 `modules/mindmap/adapter.ts` 的 `UNSAFE_ID_CHARS = /["\\]/` 判定 ——
 * 只禁引号与反斜杠，`root` 两个字母天然安全。**那一层在 `modules/`，
 * 这一层在 `core/`，所以这里只能照着它的规则写，不能 import 它**（硬约束 1）。
 */
const ROOT_ID = 'root';

/** 一张空导图：只有中心节点，没有子节点。 */
export function emptyMindmapDocument(rootTopic: string): MindmapDocument {
  const root: MindmapNode = { id: ROOT_ID, topic: rootTopic, children: [], extra: {} };
  return { version: MINDMAP_VERSION, layout: MINDMAP_DEFAULT_LAYOUT, root, extra: {} };
}

/**
 * 空导图的**落盘文本**。
 *
 * `options` 原样交给序列化器（默认缩进 2、`\n` 行尾，与 §7.1 的示例一致）。
 */
export function emptyMindmapText(
  rootTopic: string,
  options: SerializeMindmapOptions = {},
): string {
  return serializeMindmap(emptyMindmapDocument(rootTopic), options);
}