// @vitest-environment jsdom
/**
 * `node-images.ts` 的单元测试（ADR 0054）。
 *
 * 这一层全是纯函数与 DOM 小工具 —— 测得到的是「查表的输入对不对、
 * 占位长什么样、字节能不能变成 data: URL」；**像素解码**（jsdom 的
 * `Image` 不真解码）与「画出来好不好看」属于人工冒烟，ADR 0054 里写明。
 */
import { afterEach, describe, expect, it, vi } from 'vitest';

import { setLocale } from '../../core/i18n/index.js';
import { type MindmapNode } from '../../core/mindmap/types.js';
import {
  bytesToDataUrl,
  clampDisplaySize,
  collectImageNodeIds,
  collectImageUrls,
  decodeImageSize,
  extractImageFile,
  placeholderImage,
  prewarmImageCache,
} from './node-images.js';

const leaf = (over: Partial<MindmapNode>): MindmapNode => ({ id: 'x', topic: 't', children: [], extra: {}, ...over });

afterEach(() => {
  vi.unstubAllGlobals();
  setLocale('zh-CN');
});

describe('collectImageUrls / collectImageNodeIds · 树上的图片引用', () => {
  const root: MindmapNode = leaf({
    id: 'root',
    image: { url: 'assets/a.png', width: 1, height: 1 },
    children: [
      leaf({ id: 'n1', image: { url: 'assets/a.png', width: 1, height: 1 } }), // 与根同一张 → 去重
      leaf({ id: 'n2', image: { url: 'data:image/png;base64,AAAA', width: 1, height: 1 } }), // 内嵌 → 不读盘
      leaf({ id: 'n3', image: { url: '', width: 1, height: 1 } }), // 空串 → 没有引用
      leaf({ id: 'n4', image: { url: 'assets/b.png', width: 1, height: 1 } }),
    ],
  });

  it('只收需要解析的引用：跳过 data: 与空串，重复引用只读一次盘', () => {
    expect(collectImageUrls(root)).toEqual(['assets/a.png', 'assets/b.png']);
  });

  it('带图节点的 id 全量收出（先序）—— data: 与空串的节点也算「有图」', () => {
    expect(collectImageNodeIds(root)).toEqual(['root', 'n1', 'n2', 'n3', 'n4']);
  });
});

describe('placeholderImage · 带路径的占位图', () => {
  it('是内嵌 SVG 的 data: URL，能解码回来，路径如实写在里面', () => {
    const url = placeholderImage('assets/lost.png');
    expect(url.startsWith('data:image/svg+xml;charset=utf-8,')).toBe(true);
    const svg = decodeURIComponent(url.slice('data:image/svg+xml;charset=utf-8,'.length));
    expect(svg).toContain('<svg');
    expect(svg).toContain('assets/lost.png');
  });

  it('路径按 XML 文本转义 —— 路径是任意输入，必须当文本处理', () => {
    const svg = decodeURIComponent(placeholderImage('a&b<c>"d\'e.png').slice('data:image/svg+xml;charset=utf-8,'.length));
    expect(svg).toContain('a&amp;b&lt;c&gt;&quot;d&apos;e.png');
    // 转义过的串里不该再出现裸的特殊字符（结构标签除外）
    expect(svg).not.toContain('<c>');
  });

  it('超长路径截断（SVG 的 text 不换行，超宽会溢出虚线框）', () => {
    const long = 'x'.repeat(100);
    const svg = decodeURIComponent(placeholderImage(long).slice('data:image/svg+xml;charset=utf-8,'.length));
    expect(svg).toContain('x'.repeat(43) + '…');
    expect(svg).not.toContain('x'.repeat(44));
  });
});

describe('prewarmImageCache · 解析不到一律落占位', () => {
  it('解析成功的引用原样进表', async () => {
    const cache = await prewarmImageCache(['a.png'], async () => 'data:image/png;base64,AAAA');
    expect(cache.get('a.png')).toBe('data:image/png;base64,AAAA');
  });

  it('resolve 交回 null（图丢了是常见状态）→ 占位，绝不让 src 空着', async () => {
    const cache = await prewarmImageCache(['a.png'], async () => null);
    expect(cache.get('a.png')?.startsWith('data:image/svg+xml;charset=utf-8,')).toBe(true);
  });

  it('resolve 抛异常 → 占位（预热是后台动作，不许把挂载炸掉）', async () => {
    const cache = await prewarmImageCache(['a.png'], async () => {
      throw new Error('disk on fire');
    });
    expect(cache.get('a.png')?.startsWith('data:image/svg+xml;charset=utf-8,')).toBe(true);
  });

  it('外壳没实现契约（undefined）→ 占位', async () => {
    const cache = await prewarmImageCache(['a.png'], undefined);
    expect(cache.get('a.png')?.startsWith('data:image/svg+xml;charset=utf-8,')).toBe(true);
  });
});

describe('extractImageFile · 剪贴板里的图片', () => {
  it('没有剪贴板数据 → null', () => {
    expect(extractImageFile(null)).toBeNull();
  });

  it('只有文本 → null（这次粘贴跟图片无关）', () => {
    const data = new DataTransfer();
    data.setData('text/plain', 'hello');
    expect(extractImageFile(data)).toBeNull();
  });

  it('非图片文件（text/plain 的 File）→ null', () => {
    const data = new DataTransfer();
    data.items.add(new File(['hi'], 'note.txt', { type: 'text/plain' }));
    expect(extractImageFile(data)).toBeNull();
  });

  it('图片文件 → 取出第一个（png 的魔数当内容，够辨类型就行）', () => {
    const data = new DataTransfer();
    data.items.add(new File([new Uint8Array([0x89, 0x50, 0x4e, 0x47])], 'shot.png', { type: 'image/png' }));
    data.items.add(new File([new Uint8Array([0xff, 0xd8])], 'other.jpg', { type: 'image/jpeg' }));
    const file = extractImageFile(data);
    expect(file?.name).toBe('shot.png');
    expect(file?.type).toBe('image/png');
  });

  it('只放进 files、items 是空壳的填充方式同样取得到（平台差异，见函数注释）', () => {
    // jsdom 恰好就是这样一种填充方式：items.add 后 files 有货、
    // items[i].kind/type/getAsFile 全是空 —— 这条用例钉住「两处都要看」。
    const data = new DataTransfer();
    data.items.add(new File([new Uint8Array([0x89, 0x50, 0x4e, 0x47])], 'shot.png', { type: 'image/png' }));
    const item = data.items[0];
    expect(item?.kind).toBeUndefined();
    expect(item?.getAsFile() ?? null).toBeNull();
    const file = extractImageFile(data);
    expect(file?.name).toBe('shot.png');
  });
});

describe('clampDisplaySize · 只缩不放', () => {
  it('小于上限的尺寸原样返回（128×64 是合法答案，不是「太小」）', () => {
    expect(clampDisplaySize(128, 64)).toEqual({ width: 128, height: 64 });
  });

  it('恰好等于上限也原样返回（边界含在「不用缩」里）', () => {
    expect(clampDisplaySize(320, 200)).toEqual({ width: 320, height: 200 });
  });

  it('超限按最长边压到 320，比例不动、四舍五入、至少 1px', () => {
    expect(clampDisplaySize(640, 480)).toEqual({ width: 320, height: 240 });
    expect(clampDisplaySize(480, 640)).toEqual({ width: 240, height: 320 });
    expect(clampDisplaySize(4000, 3000)).toEqual({ width: 320, height: 240 });
    expect(clampDisplaySize(640, 100)).toEqual({ width: 320, height: 50 });
  });
});

describe('bytesToDataUrl · 本会话立刻可见', () => {
  it('四个 PNG 魔数字节 → data: URL（FileReader 在 jsdom 里是真跑的）', async () => {
    const bytes = new Uint8Array([0x89, 0x50, 0x4e, 0x47]).buffer;
    const url = await bytesToDataUrl(bytes, 'image/png');
    expect(url).toBe('data:image/png;base64,iVBORw==');
  });
});

describe('decodeImageSize · jsdom 不解码，注入替身验协议', () => {
  it('onload 且 naturalWidth/Height > 0 → 尺寸', async () => {
    class FakeImage {
      naturalWidth = 640;
      naturalHeight = 480;
      onload: (() => void) | null = null;
      onerror: (() => void) | null = null;
      src = '';
      constructor() {
        // 赋 src 的动作发生在构造之后，这里用微任务把 onload 补上
        queueMicrotask(() => this.onload?.());
      }
    }
    vi.stubGlobal('Image', FakeImage);
    const size = await decodeImageSize(new Uint8Array([1]).buffer, 'image/png');
    expect(size).toEqual({ width: 640, height: 480 });
  });

  it('onerror → null（坏字节如实交回，不猜尺寸）', async () => {
    class FakeImage {
      onload: (() => void) | null = null;
      onerror: (() => void) | null = null;
      src = '';
      constructor() {
        queueMicrotask(() => this.onerror?.());
      }
    }
    vi.stubGlobal('Image', FakeImage);
    const size = await decodeImageSize(new Uint8Array([1]).buffer, 'image/png');
    expect(size).toBeNull();
  });
});
