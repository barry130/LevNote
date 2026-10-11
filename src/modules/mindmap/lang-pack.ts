/**
 * 导图内核（mind-elixir）自己那套界面的语言包。
 *
 * ## 为什么不用内核自带的 `cn` / `en`
 *
 * mind-elixir 在 `mind-elixir/i18n` 这个**公开入口**上确实导出了十七种语言
 * （`:53-56` 的 `exports`），直接引一份最省事。我们没引，两个理由：
 *
 * 1. **文案要跟本项目其余界面一个口径。** 内核自带的 `cn` 写的是「插入子节点」，
 *    而本项目从头到尾管 `topic` 叫「主题」（`core/mindmap/template.ts` 的
 *    `emptyMindmapDocument('中心主题')`、解析器的错误提示都是这个词）。
 *    同一个概念在一屏里出现两种叫法，比措辞不够地道更伤。
 * 2. **一份目录就要有一个真相来源。** 用户可见的字符串按本项目的规矩只在
 *    `core/i18n/messages.ts` 里（ADR 0031）；从第二个地方取词，等于给
 *    「切语言有没有真的生效」这件事开了一个不受测试覆盖的口子。
 *
 * ## 为什么是函数而不是一个模块级常量
 *
 * 模块级 `const PACK = { addChild: t('…') }` 会在**模块被求值那一刻**烤死语言，
 * 那正是 ADR 0031 在命令面踩过的坑（`BUILT_IN_COMMANDS` 的 `title`）。
 * 这里现查表，于是「什么时候调」就是「语言什么时候生效」——
 * `MindmapEditor` 在语言变化时重建内核，靠的正是这一条。
 *
 * 判据仍然是 ADR 0047 那一句：**这句译文会不会跨过下一次 `setLocale`？**
 * 在内核里不会 —— 因为语言一变内核就被重建了，所以这里用 `t()` 而不是
 * `msg()` + `textOf()`。那个「会」正是重建的理由，不是重建的副作用。
 */

import type { LangPack } from 'mind-elixir/i18n';

import { t } from '../../core/i18n/index.js';

/**
 * 现查一份内核语言包。
 *
 * 十二个键一个都不能少：少一个，内核就在那一条上退回它自己的英文默认包
 * （`Object.assign` 只覆盖传进去的键），于是界面上**半中半英**——
 * 比全英文更难解释。少的那一条在这里由类型拦住。
 */
export function mindmapLangPack(): LangPack {
  return {
    addChild: t('ui.mindmap.addChild'),
    addParent: t('ui.mindmap.addParent'),
    addSibling: t('ui.mindmap.addSibling'),
    removeNode: t('ui.mindmap.removeNode'),
    focus: t('ui.mindmap.focus'),
    cancelFocus: t('ui.mindmap.cancelFocus'),
    moveUp: t('ui.mindmap.moveUp'),
    moveDown: t('ui.mindmap.moveDown'),
    link: t('ui.mindmap.link'),
    linkBidirectional: t('ui.mindmap.linkBidirectional'),
    clickTips: t('ui.mindmap.clickTips'),
    summary: t('ui.mindmap.summary'),
  };
}