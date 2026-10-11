/**
 * 适配器注册表。
 *
 * 只登记「专门化」的内核。纯文本兜底**不注册**，而是作为常量存在 ——
 * 这样它不可能被某个内核的错误配置给遮蔽掉，也就不会出现「打开任何文件都没反应」。
 *
 * 各产品模块在自己的目录里注册（`src/modules/index.ts` 汇总），
 * 外壳不 import 任何内核 —— 依赖方向必须是 模块 → 契约 ← 外壳。
 */

import type { EditorAdapter } from './types.js';

/**
 * 注册顺序即优先级：先注册的先匹配。
 *
 * ⚠️ 这条契约曾经与实现不符：早先的版本用 `unshift` 插入，于是**最后**注册的
 * 反而排在最前，与注释说的正好相反。在只有一个内核时看不出来，
 * 一旦接上第二个内核（代码内核认得 `.md`，作为 Markdown 的备胎），
 * 备胎就会抢在正主前面把所有 `.md` 都吃掉。
 * 现在改成「原地替换 + 追加」，让实现与契约一致。
 */
const adapters: EditorAdapter[] = [];

export function registerAdapter(adapter: EditorAdapter): void {
  const existing = adapters.findIndex((item) => item.id === adapter.id);
  if (existing >= 0) {
    // 重复注册保持**原来的位置**：否则一次热更新就会把优先级顺序洗牌
    adapters[existing] = adapter;
    return;
  }
  adapters.push(adapter);
}

export function listAdapters(): readonly EditorAdapter[] {
  return adapters;
}

/**
 * 有没有**专门化**内核认得这个路径（只看 `matches`，不问 `canHandle`）。
 *
 * ## 为什么外壳需要问这个
 *
 * 文档里的链接指向一个文件时（ADR 0040），外壳要决定「在轻笺里打开」还是
 * 「交给系统的关联程序」。判据就是这个文件是不是**一等公民**：
 * 一份 `.md` / `.mindmap` / `.drawio` 交给系统默认程序，用户会看到另一个
 * 编辑器里的源码；而一张 `.png` 在轻笺里打开，会看到一堆字节被当成文本。
 *
 * ## 为什么它在注册表里，而不是让外壳去查扩展名名单
 *
 * 那份名单长在各个内核上（`modules/code/index.ts` 的 `CODE_EXTENSIONS` 等），
 * 外壳不 import 任何具体内核（硬约束 1）。「有没有内核认得它」本来就只有
 * 注册表回答得了 —— 问注册表而不是抄一份名单，正是为了不让
 * 「哪几类文件是一等公民」有第二个真相。
 *
 * 刻意**不问 `canHandle`**：那一层需要文件内容，而这里在决定
 * 「要不要先把它读进来」，还没到那一步。
 */
export function hasSpecializedAdapter(path: string): boolean {
  return adapters.some((adapter) => adapter.matches(path));
}

/**
 * 按路径挑内核。
 *
 * 找不到专门化内核时返回 `fallbackAdapter`（纯文本）——
 * **任何文件都必须能被打开**，哪怕只是以纯文本形式。
 * 打不开文件是比「用错内核」严重得多的问题。
 *
 * `text` 是可选的内容。给出内容时，适配器还有一次 `canHandle` 的机会否决自己
 * （见 `EditorAdapter.canHandle` 的说明）。否决后继续找下一个候选，
 * 而不是直接跳到兜底 —— 将来若有第二个能处理 `.md` 的内核，它应该拿到机会。
 *
 * `canHandle` 抛错按「否决」处理，且**不让异常冒出去**：一个内核的判定逻辑写挂了，
 * 不该让用户打不开文件。这里宁可退到纯文本，也不能让打开动作失败。
 */
export function resolveAdapter(path: string, fallback: EditorAdapter, text?: string): EditorAdapter {
  for (const adapter of adapters) {
    if (!adapter.matches(path)) continue;
    if (text !== undefined && adapter.canHandle) {
      let accepted: boolean;
      try {
        accepted = adapter.canHandle(path, text);
      } catch (thrown) {
        // 判定失败只影响「要不要用这个内核」，不影响「文件能不能打开」
        console.warn(`内核 ${adapter.id} 的 canHandle 抛错，跳过它`, thrown);
        continue;
      }
      if (!accepted) continue;
    }
    return adapter;
  }
  return fallback;
}
