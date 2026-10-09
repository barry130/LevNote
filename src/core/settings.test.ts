import { describe, expect, it } from 'vitest';

import {
  clampEditorFontSize,
  clampEditorLineHeight,
  DEFAULT_SETTINGS,
  loadSettings,
  sanitizeEditorFontFamily,
  saveSettings,
  SIDEBAR_MAX_WIDTH,
  SIDEBAR_MIN_WIDTH,
  SPLIT_RATIO_MAX,
  SPLIT_RATIO_MIN,
  type StorageLike,
} from './settings.js';

function memoryStorage(seed?: string): StorageLike & { value: string | null } {
  const box = {
    value: seed ?? null,
    getItem(): string | null {
      return box.value;
    },
    setItem(_key: string, value: string): void {
      box.value = value;
    },
  };
  return box;
}

describe('编辑器字号 / 字体 / 行高', () => {
  it('默认值：15px / 内置字体 / 1.75 行高 —— 与引入偏好前的观感一致', () => {
    expect(DEFAULT_SETTINGS.editorFontSize).toBe(15);
    expect(DEFAULT_SETTINGS.editorFontFamily).toBe('');
    expect(DEFAULT_SETTINGS.editorLineHeight).toBe(1.75);
  });

  it('字号与行高被夹在允许区间内，非数值回落默认', () => {
    expect(clampEditorFontSize(8)).toBe(12);
    expect(clampEditorFontSize(15.4)).toBe(15);
    expect(clampEditorFontSize(99)).toBe(28);
    expect(clampEditorFontSize('big')).toBe(15);
    expect(clampEditorLineHeight(0.5)).toBe(1.3);
    expect(clampEditorLineHeight(1.756)).toBe(1.76);
    expect(clampEditorLineHeight(9)).toBe(2.4);
    expect(clampEditorLineHeight(Number.NaN)).toBe(1.75);
  });

  it('字体族只留白名单字符 —— 声明语法（分号 / 花括号 / url）进不了 CSS 变量', () => {
    expect(sanitizeEditorFontFamily(' 思源宋体, Source Han Serif ')).toBe('思源宋体, Source Han Serif');
    expect(sanitizeEditorFontFamily('"JetBrains Mono"')).toBe('"JetBrains Mono"');
    expect(sanitizeEditorFontFamily("a; b { c } url(http://x)")).toBe('a b  c  urlhttpx');
    // 超长截断 —— 自定义值没有无限长的道理
    expect(sanitizeEditorFontFamily('x'.repeat(500)).length).toBe(120);
    expect(sanitizeEditorFontFamily(42)).toBe('');
  });

  it('存储里的编辑器偏好被逐字段校验，缺字段不影响其它设置', () => {
    const storage = memoryStorage(
      JSON.stringify({ theme: 'ink', editorFontSize: 99, editorFontFamily: 'x} y', editorLineHeight: '宽' }),
    );
    const loaded = loadSettings(storage);
    expect(loaded.theme).toBe('ink');
    expect(loaded.editorFontSize).toBe(28);
    expect(loaded.editorFontFamily).toBe('x y');
    expect(loaded.editorLineHeight).toBe(1.75);
  });
});

describe('loadSettings', () => {
  it('没有存储时给默认值', () => {
    expect(loadSettings(null)).toEqual(DEFAULT_SETTINGS);
  });

  it('空存储给默认值', () => {
    expect(loadSettings(memoryStorage())).toEqual(DEFAULT_SETTINGS);
  });

  it('逐字段校验：坏字段回落，好字段保留', () => {
    // 逐字段而不是整体信任：新增一个偏好项不该让老用户的其他设置一起失效
    const storage = memoryStorage(JSON.stringify({ theme: 'ink', sidebarWidth: 'wide', sidebarVisible: false }));
    const settings = loadSettings(storage);
    expect(settings.theme).toBe('ink');
    expect(settings.sidebarVisible).toBe(false);
    expect(settings.sidebarWidth).toBe(DEFAULT_SETTINGS.sidebarWidth);
  });

  it('非法的主题名被忽略', () => {
    expect(loadSettings(memoryStorage(JSON.stringify({ theme: 'dark' }))).theme).toBe(DEFAULT_SETTINGS.theme);
  });

  it('侧栏宽度被夹在允许区间内', () => {
    expect(loadSettings(memoryStorage(JSON.stringify({ sidebarWidth: 10 }))).sidebarWidth).toBe(SIDEBAR_MIN_WIDTH);
    expect(loadSettings(memoryStorage(JSON.stringify({ sidebarWidth: 9999 }))).sidebarWidth).toBe(SIDEBAR_MAX_WIDTH);
  });

  it('老数据里缺少的新字段回落到默认值 —— 新增偏好项不该影响老用户', () => {
    // 存储里只有 theme（等于更早版本写下的内容）
    const settings = loadSettings(memoryStorage(JSON.stringify({ theme: 'ink' })));
    expect(settings.rememberPositions).toBe(DEFAULT_SETTINGS.rememberPositions);
    expect(settings.codeLineNumbers).toBe(DEFAULT_SETTINGS.codeLineNumbers);
    expect(settings.sidebarVisible).toBe(DEFAULT_SETTINGS.sidebarVisible);
  });

  it('位置记忆默认开，且能被显式关掉', () => {
    expect(DEFAULT_SETTINGS.rememberPositions).toBe(true);
    const off = loadSettings(memoryStorage(JSON.stringify({ rememberPositions: false })));
    expect(off.rememberPositions).toBe(false);
  });

  it('专注 / 打字机默认关，且能被显式打开；非布尔取值回落默认', () => {
    expect(DEFAULT_SETTINGS.focusMode).toBe(false);
    expect(DEFAULT_SETTINGS.typewriterMode).toBe(false);
    const on = loadSettings(memoryStorage(JSON.stringify({ focusMode: true, typewriterMode: true })));
    expect(on.focusMode).toBe(true);
    expect(on.typewriterMode).toBe(true);
    const bad = loadSettings(memoryStorage(JSON.stringify({ focusMode: 'yes', typewriterMode: 1 })));
    expect(bad.focusMode).toBe(false);
    expect(bad.typewriterMode).toBe(false);
  });

  it('非布尔的位置记忆取值被忽略（回落默认）', () => {
    expect(loadSettings(memoryStorage(JSON.stringify({ rememberPositions: 'yes' }))).rememberPositions).toBe(true);
  });

  /**
   * 分屏比例。
   *
   * 它比侧栏宽度多一层风险：拖动分隔条时会算出一个 `(clientX - left) / width`
   * 的比值，而 jsdom / 隐藏容器里 `width` 是 0 → **`NaN`**。
   * `NaN` 一旦被持久化，之后每次打开都是「比例是 NaN」，
   * 布局塌成一条线，而界面上没有任何东西提示「这是存储里的值坏了」。
   * 所以这里必须钉住「非有限数一律回落」。
   */
  it('分屏比例默认 0.5，且被夹在允许区间内', () => {
    expect(DEFAULT_SETTINGS.splitRatio).toBe(0.5);
    expect(loadSettings(memoryStorage(JSON.stringify({ splitRatio: 0.1 }))).splitRatio).toBe(SPLIT_RATIO_MIN);
    expect(loadSettings(memoryStorage(JSON.stringify({ splitRatio: 9 }))).splitRatio).toBe(SPLIT_RATIO_MAX);
  });

  it('分屏比例是 NaN / 字符串 / 缺失时回落到默认 —— 不能把 NaN 存进去', () => {
    // JSON 里没有 NaN，但 `JSON.stringify(NaN)` 会写出 `null` —— 两种都要挡住
    expect(loadSettings(memoryStorage(JSON.stringify({ splitRatio: NaN }))).splitRatio).toBe(0.5);
    expect(loadSettings(memoryStorage(JSON.stringify({ splitRatio: '0.3' }))).splitRatio).toBe(0.5);
    expect(loadSettings(memoryStorage(JSON.stringify({ splitRatio: null }))).splitRatio).toBe(0.5);
    expect(loadSettings(memoryStorage(JSON.stringify({ theme: 'ink' }))).splitRatio).toBe(0.5);
  });

  it('上下限刻意不是 0 和 1 —— 那样会把另一栏压成零宽且拖不回来', () => {
    expect(SPLIT_RATIO_MIN).toBeGreaterThan(0);
    expect(SPLIT_RATIO_MAX).toBeLessThan(1);
  });

  it('存储内容坏了（不是 JSON）不抛错，回落到默认值', () => {
    expect(loadSettings(memoryStorage('{不是 json'))).toEqual(DEFAULT_SETTINGS);
  });

  it('存储本身抛错时也不炸 —— 隐私模式 / 配额用尽', () => {
    const hostile: StorageLike = {
      getItem() {
        throw new Error('SecurityError');
      },
      setItem() {
        throw new Error('QuotaExceededError');
      },
    };
    expect(loadSettings(hostile)).toEqual(DEFAULT_SETTINGS);
    expect(() => saveSettings(hostile, DEFAULT_SETTINGS)).not.toThrow();
  });
});

describe('saveSettings', () => {
  it('往返一致', () => {
    const storage = memoryStorage();
    const settings = { ...DEFAULT_SETTINGS, theme: 'ink' as const, sidebarWidth: 300, outlineVisible: true };
    saveSettings(storage, settings);
    expect(loadSettings(storage)).toEqual(settings);
  });
});
