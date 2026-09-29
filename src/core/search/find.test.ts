import { describe, expect, it } from 'vitest';

import { findInText, replaceInText, countMatches, createFindPattern, escapeRegExp } from './find.js';

const base = { query: '', caseSensitive: false, wholeWord: false, regex: false };

describe('字面量查找', () => {
  it('默认不区分大小写', () => {
    const hits = findInText('Foo foo FOO', { ...base, query: 'foo' });
    expect(hits.map((hit) => hit.text)).toEqual(['Foo', 'foo', 'FOO']);
    expect(hits.map((hit) => hit.from)).toEqual([0, 4, 8]);
  });

  it('区分大小写时只命中一致的', () => {
    expect(findInText('Foo foo', { ...base, query: 'foo', caseSensitive: true })).toHaveLength(1);
  });

  it('查询里的正则元字符被当成普通字符', () => {
    // 用户搜 `a.b` 是想找字面的 a.b，不是「a 任意字符 b」
    const hits = findInText('a.b axb', { ...base, query: 'a.b' });
    expect(hits).toHaveLength(1);
    expect(hits[0]?.text).toBe('a.b');
  });

  it('全词匹配不命中词内片段', () => {
    expect(findInText('cat category', { ...base, query: 'cat', wholeWord: true })).toHaveLength(1);
    expect(findInText('cat category', { ...base, query: 'cat' })).toHaveLength(2);
  });
});

describe('正则查找', () => {
  it('正则模式按正则解释', () => {
    const hits = findInText('a1 b2 c3', { ...base, query: '[a-c]\\d', regex: true });
    expect(hits.map((hit) => hit.text)).toEqual(['a1', 'b2', 'c3']);
  });

  it('非法正则不抛错 —— 用户正在输入 `(` 的中间态就是非法的', () => {
    expect(() => findInText('abc', { ...base, query: '(', regex: true })).not.toThrow();
    expect(findInText('abc', { ...base, query: '(', regex: true })).toEqual([]);
    expect(createFindPattern({ ...base, query: '(', regex: true })).toBeNull();
  });

  it('零宽匹配不会死循环', () => {
    // 这是这类实现最经典的死循环来源：lastIndex 不前进
    const hits = findInText('abc', { ...base, query: 'x*', regex: true });
    expect(hits.length).toBeGreaterThan(0);
    expect(hits.every((hit) => hit.from === hit.to)).toBe(true);
  });

  it('正则 + 全词组合', () => {
    const hits = findInText('foobar foo', { ...base, query: 'foo\\w*', regex: true, wholeWord: true });
    expect(hits.map((hit) => hit.text)).toEqual(['foobar', 'foo']);
  });
});

describe('空查询', () => {
  it('空查询没有匹配，也不抛错', () => {
    expect(findInText('abc', base)).toEqual([]);
    expect(countMatches('abc', base)).toBe(0);
  });
});

describe('替换', () => {
  it('字面量替换：`$` 是普通字符', () => {
    // 用户搜 `价格$` 想替换成 `￥`，不该被当成正则替换语法
    const result = replaceInText('价格$ 价格$', { ...base, query: '价格$' }, '￥');
    expect(result.count).toBe(2);
    expect(result.text).toBe('￥ ￥');
  });

  it('正则替换支持 $1 / $&', () => {
    const result = replaceInText('a1 b2', { ...base, query: '([a-z])(\\d)', regex: true }, '$2$1');
    expect(result.text).toBe('1a 2b');
    expect(result.count).toBe(2);
  });

  it('正则替换里的 $$ 表示一个字面美元号', () => {
    // `a` 在匹配之外，只有 `1` 被替换成 `$` + 第一个捕获组（也就是 `1`）
    const result = replaceInText('a1', { ...base, query: '(\\d)', regex: true }, '$$$1');
    expect(result.text).toBe('a$1');
  });

  it('没有匹配时原样返回', () => {
    const result = replaceInText('abc', { ...base, query: 'z' }, 'y');
    expect(result.text).toBe('abc');
    expect(result.count).toBe(0);
  });

  it('全词替换只动完整单词', () => {
    const result = replaceInText('cat category', { ...base, query: 'cat', wholeWord: true }, 'dog');
    expect(result.text).toBe('dog category');
    expect(result.count).toBe(1);
  });
});

describe('escapeRegExp', () => {
  it('转义所有元字符', () => {
    expect(escapeRegExp('a.b*c')).toBe('a\\.b\\*c');
    expect(new RegExp(escapeRegExp('(x)')).test('(x)')).toBe(true);
  });
});
