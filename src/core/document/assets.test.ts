// @vitest-environment node
/**
 * 附属资源的路径与命名规则。
 *
 * 这一组测试的价值在于**把「图片存在哪」变成一个可断言的答案** ——
 * 它决定了用户磁盘上会长出什么目录、Markdown 里会写进什么字符串。
 * 一旦这里算错，后果是「图片存到了一个 Markdown 引用不到的地方」，
 * 而那种错在界面上表现为「图片显示不出来」，查起来会绕到内核和渲染层去。
 */

import { describe, expect, it } from 'vitest';

import {
  assetsDirFor,
  assetsDirNameFor,
  imageFileName,
  imageMarkdownSrc,
  isExternalSrc,
  looksLikeImage,
  resolveImagePath,
  stampOf,
} from './assets.js';

describe('资源目录', () => {
  it('与文档同名 + .assets，放在文档旁边', () => {
    expect(assetsDirFor('C:\\笔记\\读书笔记.md')).toBe('C:\\笔记\\读书笔记.assets');
    expect(assetsDirFor('/home/u/notes/a.md')).toBe('/home/u/notes/a.assets');
  });

  it('文档在盘根时不会把 `C:` 当成目录', () => {
    expect(assetsDirFor('C:\\a.md')).toBe('C:\\a.assets');
  });

  it('文档没有目录信息时返回 null —— 不猜', () => {
    expect(assetsDirFor('demo.md')).toBeNull();
    expect(assetsDirNameFor('demo.md')).toBe('demo.assets');
  });

  it('多后缀名只去掉最后一个', () => {
    expect(assetsDirNameFor('a.b.md')).toBe('a.b.assets');
  });

  it('没有扩展名的文件也用整名', () => {
    expect(assetsDirNameFor('C:\\x\\README')).toBe('README.assets');
  });
});

describe('时间戳', () => {
  it('补零到固定宽度', () => {
    expect(stampOf(new Date(2026, 8, 29, 14, 30, 22))).toBe('20260929-143022');
    expect(stampOf(new Date(2026, 0, 1, 0, 0, 0))).toBe('20260101-000000');
    expect(stampOf(new Date(2026, 11, 31, 23, 59, 59))).toBe('20261231-235959');
  });
});

describe('图片文件名', () => {
  const at = new Date(2026, 8, 29, 14, 30, 22);

  it('时间戳 + 随机后缀 + 扩展名', () => {
    expect(imageFileName({ mimeType: 'image/png', at, salt: 'a3f2' })).toBe(
      'image-20260929-143022-a3f2.png',
    );
    expect(imageFileName({ mimeType: 'image/jpeg', at, salt: '0000' })).toBe(
      'image-20260929-143022-0000.jpg',
    );
  });

  it('不认识的类型返回 null —— 宁可不落盘，也不写一个扩展名撒谎的文件', () => {
    expect(imageFileName({ mimeType: 'image/tiff', at, salt: 'aaaa' })).toBeNull();
  });

  it('同一毫秒用不同 salt 不会撞名', () => {
    const a = imageFileName({ mimeType: 'image/png', at, salt: '1111' });
    const b = imageFileName({ mimeType: 'image/png', at, salt: '2222' });
    expect(a).not.toBe(b);
  });
});

describe('哪些 src 是外部的', () => {
  it.each([
    'http://a.com/x.png',
    'https://a.com/x.png',
    'data:image/png;base64,AAAA',
    'blob:http://localhost/abc',
    'ftp://a.com/x.png',
    'mailto:a@b.com',
    '',
  ])('%s 是外部（或空），不去本地找', (src) => {
    expect(isExternalSrc(src)).toBe(true);
  });

  it.each([
    './x.assets/a.png',
    'x.assets/a.png',
    '../images/a.png',
    'C:\\notes\\a.png',
    'c:/notes/a.png',
    '\\\\server\\share\\a.png',
    '/home/u/a.png',
  ])('%s 是本地路径', (src) => {
    expect(isExternalSrc(src)).toBe(false);
  });
});

describe('解析图片的本地路径', () => {
  it('相对路径拼到文档目录上', () => {
    /*
     * 分隔符被统一成**文档路径的风格**（Windows 反斜杠、POSIX 正斜杠）。
     *
     * 这不是「改写用户写在文档里的路径」—— 文档里那一串（`./a.assets/x.png`）
     * 一个字符都没动。变的只是**算出来的本地绝对路径**，
     * 而它必须和写入时算出来的那一个逐字符相同：
     * 写的时候文件名是裸名（`joinPath(目录, 'image-x.png')`），
     * 回读的时候相对路径带正斜杠 —— 不统一的话，同一个文件会有两个字符串，
     * 任何按字符串比较的地方（缓存键、去重、相等判断）都会悄悄失配。
     */
    expect(resolveImagePath('C:\\笔记\\a.md', './a.assets/x.png')).toBe('C:\\笔记\\a.assets\\x.png');
    expect(resolveImagePath('/n/a.md', 'a.assets/x.png')).toBe('/n/a.assets/x.png');
    expect(resolveImagePath('/n/sub/a.md', '../img/x.png')).toBe('/n/sub/../img/x.png');
  });

  it('绝对路径原样返回', () => {
    expect(resolveImagePath('C:\\n\\a.md', 'D:\\pic\\x.png')).toBe('D:\\pic\\x.png');
    expect(resolveImagePath('/n/a.md', '/abs/x.png')).toBe('/abs/x.png');
  });

  it('外部地址与空 src 返回 null —— 调用方要如实降级', () => {
    expect(resolveImagePath('C:\\n\\a.md', 'https://a.com/x.png')).toBeNull();
    expect(resolveImagePath('C:\\n\\a.md', 'data:image/png;base64,AA')).toBeNull();
    expect(resolveImagePath('C:\\n\\a.md', '')).toBeNull();
  });

  it('文档没有目录信息时返回 null（相对路径无从谈起）', () => {
    expect(resolveImagePath('a.md', './x.png')).toBeNull();
  });

  it('`..` 交给文件系统去折叠，这里不自己实现', () => {
    // 断言的是「没有崩、也没有被静默吃掉」，而不是「折叠成了什么」
    const resolved = resolveImagePath('C:\\n\\a.md', '../../x.png');
    expect(resolved).toContain('..');
  });
});

describe('写进 Markdown 的 src', () => {
  it('以 ./ 开头 —— 明确表示「相对这份文档」', () => {
    expect(imageMarkdownSrc('a.assets', 'image-1.png')).toBe('./a.assets/image-1.png');
  });

  it('不含任何绝对路径（文档要能发给别人、能进 Git）', () => {
    const src = imageMarkdownSrc('a.assets', 'x.png');
    expect(src).not.toMatch(/^[A-Za-z]:/);
    expect(src.startsWith('/')).toBe(false);
  });
});

describe('看起来像图片吗', () => {
  it.each(['./a.assets/x.png', 'x.jpg', 'X.WEBP', '/abs/a.svg'])('%s 是', (src) => {
    expect(looksLikeImage(src)).toBe(true);
  });

  it.each(['a.md', 'x.tiff', 'noext', ''])('%s 不是', (src) => {
    expect(looksLikeImage(src)).toBe(false);
  });
});
