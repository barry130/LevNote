// @vitest-environment jsdom
/**
 * 图片类型与字节转换。
 *
 * 这一组是 M6 的**地基**：落盘用它的表决定后缀、编辑器显示用它把字节变成
 * `<img src>`、导出内联也用同一份。三处共用一份，所以这里的每一条断言
 * 实际上同时钉住了三条链路 —— 它一旦错，表现是「编辑器里正常、导出后打不开」
 * 这种只有断网打开导出文件才看得见的错。
 */

import { describe, expect, it } from 'vitest';

import {
  bytesToBase64,
  bytesToDataUrl,
  imageExtensionFor,
  imageMimeFor,
  isSupportedImageType,
  sniffImageMime,
} from './image.js';

describe('MIME → 扩展名', () => {
  it.each([
    ['image/png', '.png'],
    ['image/jpeg', '.jpg'],
    ['image/gif', '.gif'],
    ['image/webp', '.webp'],
    ['image/avif', '.avif'],
    ['image/bmp', '.bmp'],
    ['image/svg+xml', '.svg'],
  ])('%s → %s', (mime, expected) => {
    expect(imageExtensionFor(mime)).toBe(expected);
  });

  it('大小写与参数都要认 —— 剪贴板会给 `image/SVG+XML;charset=utf-8` 这种', () => {
    expect(imageExtensionFor('IMAGE/PNG')).toBe('.png');
    expect(imageExtensionFor('image/svg+xml;charset=utf-8')).toBe('.svg');
    expect(imageExtensionFor('  image/jpeg  ')).toBe('.jpg');
  });

  it('不认识的类型返回 null —— 调用方必须拒绝，不许猜一个扩展名', () => {
    expect(imageExtensionFor('image/tiff')).toBeNull();
    expect(imageExtensionFor('application/pdf')).toBeNull();
    expect(imageExtensionFor('')).toBeNull();
    expect(isSupportedImageType('image/tiff')).toBe(false);
    expect(isSupportedImageType('image/png')).toBe(true);
  });
});

describe('路径 → MIME', () => {
  it.each([
    ['a.png', 'image/png'],
    ['C:\\n\\a.PNG', 'image/png'],
    ['/n/a.jpg', 'image/jpeg'],
    ['.jpeg', 'image/jpeg'],
    ['jpeg', 'image/jpeg'],
    ['a.jfif', 'image/jpeg'],
    ['a.ico', 'image/x-icon'],
    ['a.svg', 'image/svg+xml'],
  ])('%s → %s', (path, expected) => {
    expect(imageMimeFor(path)).toBe(expected);
  });

  it('不认识时返回 null', () => {
    expect(imageMimeFor('a.md')).toBeNull();
    expect(imageMimeFor('a.tiff')).toBeNull();
    expect(imageMimeFor('noext')).toBeNull();
    expect(imageMimeFor('')).toBeNull();
  });

  it('反向比正向多几项 —— 那些只出现在「读一个已存在的文件」时', () => {
    // 我们落盘只会写自己认得的格式，所以不需要为 `image/x-icon` 准备后缀
    expect(imageMimeFor('a.ico')).toBe('image/x-icon');
    expect(imageExtensionFor('image/x-icon')).toBeNull();
  });
});

describe('从字节判断图片类型', () => {
  const bytes = (...values: number[]) => new Uint8Array(values);

  it('按文件头认出常见格式', () => {
    expect(sniffImageMime(bytes(0x89, 0x50, 0x4e, 0x47, 0x0d), 'x.png')).toBe('image/png');
    expect(sniffImageMime(bytes(0xff, 0xd8, 0xff, 0xe0), 'x.jpg')).toBe('image/jpeg');
    expect(sniffImageMime(bytes(0x47, 0x49, 0x46, 0x38, 0x39), 'x.gif')).toBe('image/gif');
    expect(sniffImageMime(bytes(0x42, 0x4d, 0x00), 'x.bmp')).toBe('image/bmp');
  });

  it('WEBP 要同时看 RIFF 与偏移 8 的 WEBP 标记', () => {
    const webp = new Uint8Array(12);
    webp.set([0x52, 0x49, 0x46, 0x46], 0);
    webp.set([0x57, 0x45, 0x42, 0x50], 8);
    expect(sniffImageMime(webp, 'x.webp')).toBe('image/webp');

    // 只有 RIFF 头、偏移 8 不是 WEBP：是别的 RIFF 容器（如 wav），不该认成图片
    const riff = new Uint8Array(12);
    riff.set([0x52, 0x49, 0x46, 0x46], 0);
    riff.set([0x57, 0x41, 0x56, 0x45], 8);
    expect(sniffImageMime(riff, 'x.wav')).toBe('application/octet-stream');
  });

  it('`<` 开头且含 `<svg` 才算 SVG', () => {
    const svg = new TextEncoder().encode('<svg xmlns="http://www.w3.org/2000/svg"></svg>');
    expect(sniffImageMime(svg, 'x.svg')).toBe('image/svg+xml');

    const html = new TextEncoder().encode('<html><body>hi</body></html>');
    // 是 HTML 不是 SVG —— 按扩展名兜底，没有扩展名就是八位字节流
    expect(sniffImageMime(html, 'x.html')).toBe('application/octet-stream');
  });

  it('**文件头优先于扩展名** —— 扩展名写错的图片很常见', () => {
    // 一个真的 PNG，但文件名骗人说是 .jpg
    expect(sniffImageMime(bytes(0x89, 0x50, 0x4e, 0x47), 'x.jpg')).toBe('image/png');
  });

  it('认不出来时退回扩展名，最后才是不透明的字节流', () => {
    expect(sniffImageMime(bytes(0x01, 0x02), 'x.png')).toBe('image/png');
    expect(sniffImageMime(bytes(0x01, 0x02), 'x.bin')).toBe('application/octet-stream');
  });

  it('字节太少时不越界（`bytes[8]` / `bytes[9]` 是裸下标访问）', () => {
    expect(() => sniffImageMime(bytes(0x52, 0x49), 'x.webp')).not.toThrow();
    expect(() => sniffImageMime(bytes(), 'x.png')).not.toThrow();
  });
});

describe('字节 → base64 / data URI', () => {
  it('小数据转对', () => {
    expect(bytesToBase64(new Uint8Array([0x68, 0x69]).buffer)).toBe('aGk=');
  });

  it('**超过 65535 字节也不炸** —— 这是分块的理由，不是优化', () => {
    // `String.fromCharCode(...bytes)` 的参数个数上限约 65535，
    // 一张 1 MB 的图会直接抛 RangeError。这条用例就是钉住它。
    const big = new Uint8Array(200_000);
    big.fill(0x41);
    expect(() => bytesToBase64(big.buffer)).not.toThrow();
    expect(bytesToBase64(big.buffer).length).toBe(Math.ceil(200_000 / 3) * 4);
  });

  it('data URI 的形状正确', () => {
    expect(bytesToDataUrl(new Uint8Array([0x68, 0x69]).buffer, 'image/png')).toBe(
      'data:image/png;base64,aGk=',
    );
  });

  it('空字节也给出合法的 data URI，而不是抛错', () => {
    expect(bytesToDataUrl(new ArrayBuffer(0), 'image/png')).toBe('data:image/png;base64,');
  });
});
