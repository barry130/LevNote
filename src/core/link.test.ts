import { describe, expect, it } from 'vitest';

import { classifyLink, decodeAnchor, splitFragment } from './link.js';

describe('splitFragment', () => {
  it('没有 # 时整串都是路径，锚点为 null', () => {
    expect(splitFragment('notes/a.md')).toEqual({ path: 'notes/a.md', anchor: null });
  });

  it('按第一个 # 切开', () => {
    expect(splitFragment('notes/a.md#sec')).toEqual({ path: 'notes/a.md', anchor: 'sec' });
  });

  it('锚点里的第二个 # 属于锚点文字', () => {
    expect(splitFragment('#a#b')).toEqual({ path: '', anchor: 'a#b' });
  });

  it('以 # 结尾时锚点是空串（不是 null）', () => {
    expect(splitFragment('a.md#')).toEqual({ path: 'a.md', anchor: '' });
  });
});

describe('decodeAnchor', () => {
  it('解码百分号转义 —— 与浏览器匹配 id 的方式对齐', () => {
    expect(decodeAnchor('%E6%A0%87%E9%A2%98')).toBe('标题');
  });

  it('没有 % 时原样返回', () => {
    expect(decodeAnchor('hello-world')).toBe('hello-world');
  });

  it('畸形的 % 序列不抛错，原样返回', () => {
    // 手写地址里 `%` 很常见（「#100%-覆盖率」），那是用户的原文，不是错误
    expect(decodeAnchor('100%-覆盖率')).toBe('100%-覆盖率');
  });
});

describe('classifyLink · 空地址', () => {
  it.each(['', '   ', '#', '  #  '])('「%s」是空地址', (href) => {
    expect(classifyLink(href)).toEqual({ kind: 'empty' });
  });
});

describe('classifyLink · 本文档锚点', () => {
  it('纯锚点', () => {
    expect(classifyLink('#标题')).toEqual({ kind: 'anchor', anchor: '标题' });
  });

  it('锚点会被解码', () => {
    expect(classifyLink('#%E6%A0%87%E9%A2%98')).toEqual({ kind: 'anchor', anchor: '标题' });
  });

  it('首尾空白被 trim', () => {
    expect(classifyLink('  #sec  ')).toEqual({ kind: 'anchor', anchor: 'sec' });
  });
});

describe('classifyLink · 外部地址', () => {
  it.each([
    'https://example.com/a?b=1#c',
    'http://example.com',
    'mailto:someone@example.com',
  ])('「%s」是外部地址，且 href 原样保留（含查询串与锚点）', (href) => {
    expect(classifyLink(href)).toEqual({ kind: 'external', href });
  });

  it('协议大小写不敏感', () => {
    expect(classifyLink('HTTPS://example.com')).toEqual({
      kind: 'external',
      href: 'HTTPS://example.com',
    });
  });
});

describe('classifyLink · 不支持的协议', () => {
  it.each(['javascript:alert(1)', 'data:text/html,<b>x</b>', 'ftp://host/a', 'tel:+8613800000000'])(
    '「%s」不在主动交出的白名单里',
    (href) => {
      expect(classifyLink(href)).toEqual({ kind: 'unsupported', href });
    },
  );

  it('混进控制字符的 javascript: 也认得出协议', () => {
    // 浏览器解析 URL 时会忽略制表符，`java\tscript:` 仍然是 javascript:
    expect(classifyLink('java\tscript:alert(1)').kind).toBe('unsupported');
  });
});

describe('classifyLink · 文件', () => {
  it('相对路径', () => {
    expect(classifyLink('notes/a.md')).toEqual({
      kind: 'file',
      path: 'notes/a.md',
      absolute: false,
      anchor: null,
    });
  });

  it('带锚点的相对路径', () => {
    expect(classifyLink('./a.md#sec')).toEqual({
      kind: 'file',
      path: './a.md',
      absolute: false,
      anchor: 'sec',
    });
  });

  it('盘符绝对路径 —— 不能被当成协议', () => {
    // 这是本项目最容易写错的一处：`C:` 与 `https:` 在正则上完全同形
    expect(classifyLink('C:\\notes\\a.md')).toEqual({
      kind: 'file',
      path: 'C:\\notes\\a.md',
      absolute: true,
      anchor: null,
    });
    expect(classifyLink('c:/notes/a.md')).toEqual({
      kind: 'file',
      path: 'c:/notes/a.md',
      absolute: true,
      anchor: null,
    });
  });

  it('UNC 路径是绝对路径', () => {
    expect(classifyLink('\\\\server\\share\\a.md')).toEqual({
      kind: 'file',
      path: '\\\\server\\share\\a.md',
      absolute: true,
      anchor: null,
    });
  });

  it('POSIX 根路径是绝对路径', () => {
    expect(classifyLink('/home/u/a.md')).toEqual({
      kind: 'file',
      path: '/home/u/a.md',
      absolute: true,
      anchor: null,
    });
  });

  it('以 # 结尾的相对路径：锚点是空串，收敛成 null', () => {
    expect(classifyLink('a.md#')).toEqual({
      kind: 'file',
      path: 'a.md',
      absolute: false,
      anchor: null,
    });
  });

  it('裸文件名（没有目录）也是相对路径 —— 能不能解析由外壳回答', () => {
    expect(classifyLink('a.md')).toEqual({
      kind: 'file',
      path: 'a.md',
      absolute: false,
      anchor: null,
    });
  });
});
