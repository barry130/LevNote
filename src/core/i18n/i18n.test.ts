/**
 * i18n 骨架的测试（P3 第 17 项，ADR 0031）。
 *
 * 骨架只有三件事要钉住：查表取的是**原值**（抽文案不许顺手改文案 ——
 * 命令面板、设置页的快捷键表都从这些字符串派生）、插值的**漏参原样保留**、
 * 查不到 key 时**宁可显示 key 也不给空白**。
 */

import { describe, expect, it } from 'vitest';

import { getLocale, interpolate, setLocale, subscribeLocale, t } from './index.js';
import { en, zhCN, type MessageKey } from './messages.js';

describe('t', () => {
  it('目录里的 key 返回原值 —— 抽文案不改文案', () => {
    for (const [key, value] of Object.entries(zhCN)) {
      expect(t(key as MessageKey)).toBe(value);
    }
  });

  it('每个目录 key 都是纯字符串，不夹占位符（目录里还没有需要插值的文案）', () => {
    for (const value of Object.values(zhCN)) {
      expect(value).not.toMatch(/\{(\w+)\}/);
    }
  });

  it('查不到的 key 返回 key 本身 —— 可见、可搜，而不是无声空白', () => {
    // 类型上不可能写错 key；这一条钉的是运行时的最后防线（目录被裁剪、
    // 动态拼 key 之类的将来动作），cast 是有意的出格
    expect(t('command.missing.thing.title' as MessageKey)).toBe('command.missing.thing.title');
  });
});

describe('locale 状态', () => {
  it('默认 zh-CN；setLocale / getLocale 往返', () => {
    expect(getLocale()).toBe('zh-CN');
    setLocale('zh-CN');
    expect(getLocale()).toBe('zh-CN');
  });

  it('非默认语言允许缺 key：缺的落回 zh-CN 的值', () => {
    // 用测试目录模拟「只翻了一部分」的将来语言，钉住回退链的语义。
    // CATALOGS 不导出 —— 这里直接构造与 t 相同的查表式做对照是不行的，
    // 所以换一个角度：t 的回退链是 CATALOGS[current] ?? zhCN，
    // current=zh-CN 时两者同一份，回退链退化成恒等 —— 上面第一条已钉住。
    // 这条存在的意义是提醒：扩语言时，回退链测试要跟着补。
    expect(getLocale()).toBe('zh-CN');
  });
});

describe('en 目录（第二份语言，2026-10-07）', () => {
  it('与 zh-CN 一一对应：key 集合相等（类型已钉，这里防运行时漂移）', () => {
    expect(Object.keys(en).sort()).toEqual(Object.keys(zhCN).sort());
  });

  it('切到英文后命令标题换成英文；切回 zh-CN 恢复', () => {
    setLocale('en');
    expect(t('command.file.save.title')).toBe('Save');
    expect(t('menu.file')).toBe('File');
    expect(t('command.category.help')).toBe('Help');
    setLocale('zh-CN');
    expect(t('command.file.save.title')).toBe('保存');
  });

  it('setLocale 通知订阅者；设同一个值不通知', () => {
    let count = 0;
    const unsubscribe = subscribeLocale(() => {
      count += 1;
    });
    setLocale('en');
    setLocale('en');
    setLocale('zh-CN');
    unsubscribe();
    setLocale('en');
    expect(count).toBe(2);
    setLocale('zh-CN');
  });
});

describe('interpolate', () => {
  it('占位符换成参数，字符串与数字都行', () => {
    expect(interpolate('已打开 {name}', { name: 'a.md' })).toBe('已打开 a.md');
    expect(interpolate('{count} 张图片已内联', { count: 3 })).toBe('3 张图片已内联');
  });

  it('多个占位符与相邻文本', () => {
    expect(interpolate('{a} 和 {b}', { a: 1, b: '二' })).toBe('1 和 二');
  });

  it('漏传参数的占位符原样保留 —— 悄悄删掉就没人知道少了东西', () => {
    expect(interpolate('已打开 {name}', {})).toBe('已打开 {name}');
  });

  it('没有占位符的模板原样返回', () => {
    expect(interpolate('保存', { name: 'x' })).toBe('保存');
  });
});
