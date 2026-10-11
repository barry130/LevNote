/**
 * 思维导图内核语言包的判据（ADR 0049）。
 *
 * 界面效果那条（「菜单真的变中文了」「切语言当场变」）在 `MindmapEditor.test.tsx`
 * 里，因为那要真的建一个内核。这里只管两件内核外面就能验的事：
 *
 * 1. **十二个键一个都不少** —— 少一个，内核就在那一条上退回它自己的英文默认包
 *    （`Object.assign` 只覆盖传进去的键），界面上就成了半中半英，比全英文更难解释。
 * 2. **现查现变** —— 这份包是模块被求值那一刻烤死的，还是每次调用现查的。
 *    切语言能不能当场生效，全押在这一条上。
 */

import { describe, expect, it } from 'vitest';

import { setLocale, t } from '../../core/i18n/index.js';
import { mindmapLangPack } from './lang-pack.js';

/** 内核 `LangPack` 的十二个键（`node_modules/mind-elixir/dist/types/i18n.d.ts`）。 */
const KEYS = [
  'addChild',
  'addParent',
  'addSibling',
  'cancelFocus',
  'clickTips',
  'focus',
  'link',
  'linkBidirectional',
  'moveDown',
  'moveUp',
  'removeNode',
  'summary',
];

describe('思维导图内核的语言包（ADR 0049）', () => {
  it('十二个键一个不少，且没有空串', () => {
    const pack = mindmapLangPack();
    expect(Object.keys(pack).sort()).toEqual([...KEYS].sort());
    for (const value of Object.values(pack)) {
      expect(value).not.toBe('');
    }
  });

  it('每一条都取自本项目的目录（ADR 0031：用户可见的字符串只有一个来源）', () => {
    // 用运行时比对而不是抄一遍字面量：抄的那份会先于目录掉队。
    const values = Object.values(mindmapLangPack());
    for (const key of KEYS) {
      expect(values).toContain(t(`ui.mindmap.${key}` as Parameters<typeof t>[0]));
    }
  });

  it('是函数不是常量 —— 切语言之后现查现变', () => {
    setLocale('zh-CN');
    const zh = mindmapLangPack();
    setLocale('en');
    const en = mindmapLangPack();
    setLocale('zh-CN');

    expect(zh.addChild).toBe('插入子主题');
    expect(en.addChild).toBe('Add child');
    // 整包都不该是同一串 —— 只换一条的话界面上就是半中半英
    expect(Object.values(en)).not.toEqual(Object.values(zh));
  });
});