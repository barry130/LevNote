import { describe, expect, it } from 'vitest';

import {
  basename,
  dirname,
  extensionOf,
  isInside,
  isMarkdownPath,
  joinPath,
  MARKDOWN_EXTENSIONS,
  segments,
  separatorOf,
  titleOf,
} from './path.js';

describe('basename / titleOf / extensionOf', () => {
  it('同时认 Windows 与 POSIX 两种分隔符', () => {
    expect(basename('C:\\notes\\a.md')).toBe('a.md');
    expect(basename('/home/u/a.md')).toBe('a.md');
    // 混用也要对：WebView 里拿到的路径有时是这种
    expect(basename('C:\\notes/sub/a.md')).toBe('a.md');
  });

  it('没有分隔符时原样返回', () => {
    expect(basename('a.md')).toBe('a.md');
    expect(basename('')).toBe('');
  });

  it('titleOf 去掉最后一个扩展名', () => {
    expect(titleOf('C:\\notes\\a.md')).toBe('a');
    expect(titleOf('archive.tar.gz')).toBe('archive.tar');
    expect(titleOf('README')).toBe('README');
  });

  it('扩展名统一小写，且隐藏文件不算扩展名', () => {
    expect(extensionOf('A.MD')).toBe('.md');
    // `.gitignore` 不是「扩展名为 gitignore 的无名文件」，它就是一个隐藏文件
    expect(extensionOf('.gitignore')).toBe('');
    expect(extensionOf('noext')).toBe('');
    expect(extensionOf('C:\\a\\b.Tsx')).toBe('.tsx');
  });
});

describe('dirname', () => {
  it('常规路径', () => {
    expect(dirname('C:\\notes\\a.md')).toBe('C:\\notes');
    expect(dirname('/home/u/a.md')).toBe('/home/u');
  });

  it('Windows 盘根：不能只切出 `C:`', () => {
    // `C:` 表示「C 盘的当前目录」，不是根目录。少这一个反斜杠，
    // 拼出来的图片路径就会指向一个不存在的位置
    expect(dirname('C:\\a.md')).toBe('C:\\');
    expect(dirname('/a.md')).toBe('/');
  });

  it('拿不到目录时返回 null —— 不能猜', () => {
    // 光秃秃的文件名（浏览器虚拟盘）没有目录信息，硬拼一定是错的
    expect(dirname('a.md')).toBeNull();
    expect(dirname('')).toBeNull();
  });
});

describe('joinPath', () => {
  it('按基路径的风格拼分隔符，相对部分的分隔符也统一成同一种', () => {
    expect(joinPath('C:\\notes', 'img/a.png')).toBe('C:\\notes\\img\\a.png');
    expect(joinPath('/home/u', 'img/a.png')).toBe('/home/u/img/a.png');
    // 反过来的方向也要统一：POSIX 基路径 + 反斜杠的相对路径
    expect(joinPath('/home/u', 'img\\a.png')).toBe('/home/u/img/a.png');
  });

  it('基路径结尾已有分隔符时不重复', () => {
    expect(joinPath('C:\\', 'a.png')).toBe('C:\\a.png');
    expect(joinPath('/home/u/', 'a.png')).toBe('/home/u/a.png');
  });

  it('相对路径开头的分隔符被吃掉（它本来就是相对的）', () => {
    expect(joinPath('C:\\notes', '\\img\\a.png')).toBe('C:\\notes\\img\\a.png');
  });

  it('`./` 前缀也被吃掉 —— 否则它会一路走到 UI 和错误消息里', () => {
    expect(joinPath('C:\\notes', './a.assets/x.png')).toBe('C:\\notes\\a.assets\\x.png');
    expect(joinPath('/n', './x.png')).toBe('/n/x.png');
  });

  it('但 `../` 必须保留 —— 它是语义，不是多余的前缀', () => {
    expect(joinPath('/n/sub', '../img/x.png')).toBe('/n/sub/../img/x.png');
    expect(joinPath('C:\\n\\sub', '..\\x.png')).toBe('C:\\n\\sub\\..\\x.png');
  });

  it('单独的 `.` 原样保留（它表示当前目录，不是前缀）', () => {
    expect(joinPath('/n', '.')).toBe('/n/.');
  });

  it('重复的分隔符**不折叠** —— 那是写入层的事', () => {
    expect(joinPath('/n', 'a//b.png')).toBe('/n/a//b.png');
    expect(joinPath('C:\\n', 'a\\\\b.png')).toBe('C:\\n\\a\\\\b.png');
  });

  it('基路径为 null 时返回 null，而不是拼一个相对路径出来', () => {
    expect(joinPath(null, 'a.png')).toBeNull();
  });

  /**
   * 这条是整个「图片落盘 → 回读」链路的基石，所以单独钉一次。
   *
   * 写入时的文件名是**裸名**（`image-x.png`，没有分隔符），
   * 回读时来自 Markdown 的相对路径是 `./a.assets/image-x.png`（正斜杠）。
   * 两条路算出来的必须是**同一个字符串** —— 否则任何按字符串比较的地方
   * （缓存键、去重、相等判断）都会悄悄失配，而文件系统那边一切正常，
   * 于是表现为「写进去了但显示不出来」。
   */
  it('写进去的路径与回读时算出来的路径**逐字符相同**', () => {
    const directory = 'C:\\笔记\\读书笔记.assets';

    const written = joinPath(directory, 'image-20260929-143022-a3f2.png');
    const readBack = joinPath('C:\\笔记', './读书笔记.assets/image-20260929-143022-a3f2.png');
    expect(readBack).toBe(written);
  });
});

describe('isInside', () => {
  it('判断路径是否在目录之内', () => {
    expect(isInside('C:\\notes\\a.md', 'C:\\notes')).toBe(true);
    expect(isInside('C:\\notes2\\a.md', 'C:\\notes')).toBe(false);
  });

  it('Windows 下不区分大小写', () => {
    expect(isInside('c:\\Notes\\a.md', 'C:\\notes')).toBe(true);
  });
});

describe('segments / separatorOf', () => {
  it('拆成层级片段', () => {
    expect(segments('C:\\notes\\sub\\a.md')).toEqual(['C:', 'notes', 'sub', 'a.md']);
    expect(segments('/home/u/a.md')).toEqual(['home', 'u', 'a.md']);
  });

  it('分隔符按「出现过反斜杠」判断', () => {
    expect(separatorOf('C:\\a')).toBe('\\');
    expect(separatorOf('/a')).toBe('/');
  });
});

/**
 * 这份名单有四个使用者（Markdown 内核、代码内核的源码视图、外壳的大纲判断、
 * 将来的引用解析），所以「它到底包含什么」值得逐条钉住 ——
 * 尤其是它**不**包含什么，因为「多算一个」的后果是把用户的文件改坏。
 */
describe('Markdown 扩展名名单', () => {
  it('名单里的每一个扩展名都被认出来，且大小写不敏感', () => {
    for (const ext of MARKDOWN_EXTENSIONS) {
      expect(isMarkdownPath(`笔记${ext}`), ext).toBe(true);
    }
    expect(isMarkdownPath('C:\\notes\\A.MDOWN')).toBe(true);
    expect(isMarkdownPath('notes.Markdown')).toBe(true);
  });

  it('不把 .mdx 算进来 —— 它的 JSX 会让富文本内核改坏文件', () => {
    expect(isMarkdownPath('component.mdx')).toBe(false);
  });

  it('不是「所有文本文件」', () => {
    expect(isMarkdownPath('a.txt')).toBe(false);
    expect(isMarkdownPath('a.md.bak')).toBe(false);
    // 后缀相似但不是：`.markdownx` 不该被前缀匹配骗过去
    expect(isMarkdownPath('a.markdownx')).toBe(false);
  });

  it('名为 .md 的隐藏文件不算 Markdown', () => {
    // 与 extensionOf 的约定一致：`.gitignore` 是「没有主名的隐藏文件」，
    // 不是一个扩展名为 `gitignore` 的文件。`.md` 同理。
    expect(isMarkdownPath('.md')).toBe(false);
    expect(extensionOf('.md')).toBe('');
  });
});
