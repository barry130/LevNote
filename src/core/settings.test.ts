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
    const read = loadSettings(storage);
    expect(read.value.theme).toBe('ink');
    expect(read.value.editorFontSize).toBe(28);
    expect(read.value.editorFontFamily).toBe('x y');
    expect(read.value.editorLineHeight).toBe(1.75);
    // 夹取与消毒都算「意图已部分兑现」，不报；只有「宽」这种压根读不成数的才算丢
    expect(read.outcome).toEqual({ ok: false, reason: 'corrupt', lostKeys: ['editorLineHeight'], lostCount: 1 });
  });
});

describe('loadSettings', () => {
  it('没有存储时给默认值 —— 但 outcome 说得出「这是没有存储，不是存坏了」', () => {
    const read = loadSettings(null);
    expect(read.value).toEqual(DEFAULT_SETTINGS);
    expect(read.outcome).toEqual({ ok: false, reason: 'no-storage', lostKeys: [], lostCount: 0 });
  });

  it('空存储给默认值，且**什么都不报** —— 第一次打开就是默认值，没人丢了东西', () => {
    const read = loadSettings(memoryStorage());
    expect(read.value).toEqual(DEFAULT_SETTINGS);
    // 首次启动是常态，不是故障。报它 = 每次新装都先看一条红字。
    expect(read.outcome).toEqual({ ok: true });
  });

  it('逐字段校验：坏字段点名回落，好字段保留', () => {
    // 逐字段而不是整体信任：新增一个偏好项不该让老用户的其他设置一起失效
    const storage = memoryStorage(JSON.stringify({ theme: 'ink', sidebarWidth: 'wide', sidebarVisible: false }));
    const read = loadSettings(storage);
    expect(read.value.theme).toBe('ink');
    expect(read.value.sidebarVisible).toBe(false);
    expect(read.value.sidebarWidth).toBe(DEFAULT_SETTINGS.sidebarWidth);
    // 点名到字段这一级：用户能去存储里找「sidebarWidth」把它改回来
    expect(read.outcome).toEqual({ ok: false, reason: 'corrupt', lostKeys: ['sidebarWidth'], lostCount: 1 });
  });

  it('非法的主题名被忽略，并被点名', () => {
    const read = loadSettings(memoryStorage(JSON.stringify({ theme: 'dark' })));
    expect(read.value.theme).toBe(DEFAULT_SETTINGS.theme);
    expect(read.outcome).toEqual({ ok: false, reason: 'corrupt', lostKeys: ['theme'], lostCount: 1 });
  });

  it('键位映射里逐命令坏掉的，只点名那一条命令，其余照旧生效', () => {
    const read = loadSettings(memoryStorage(JSON.stringify({ keybindings: { toggleSidebar: 'mod+b', bold: 7 } })));
    expect(read.value.keybindings.toggleSidebar).toBe('mod+b');
    expect(read.outcome).toEqual({ ok: false, reason: 'corrupt', lostKeys: ['keybindings.bold'], lostCount: 1 });
  });

  it('侧栏宽度被夹在允许区间内 —— 夹取不是丢失，不报', () => {
    // 存的是 10，不可用；但意图（窄一点）兑现了一部分，说出来只是噪音
    expect(loadSettings(memoryStorage(JSON.stringify({ sidebarWidth: 10 }))).outcome).toEqual({ ok: true });
    expect(loadSettings(memoryStorage(JSON.stringify({ sidebarWidth: 9999 }))).outcome).toEqual({ ok: true });
    expect(loadSettings(memoryStorage(JSON.stringify({ sidebarWidth: 10 }))).value.sidebarWidth).toBe(SIDEBAR_MIN_WIDTH);
    expect(loadSettings(memoryStorage(JSON.stringify({ sidebarWidth: 9999 }))).value.sidebarWidth).toBe(SIDEBAR_MAX_WIDTH);
  });

  it('老数据里缺少的新字段回落到默认值 —— 且不报，因为没人因此丢掉过任何东西', () => {
    // 存储里只有 theme（等于更早版本写下的内容）
    const read = loadSettings(memoryStorage(JSON.stringify({ theme: 'ink' })));
    expect(read.value.rememberPositions).toBe(DEFAULT_SETTINGS.rememberPositions);
    expect(read.value.codeLineNumbers).toBe(DEFAULT_SETTINGS.codeLineNumbers);
    expect(read.value.sidebarVisible).toBe(DEFAULT_SETTINGS.sidebarVisible);
    // 版本升级新增的字段若也算「丢失」，每升一版就给全体用户推一条告警
    expect(read.outcome).toEqual({ ok: true });
  });

  it('位置记忆默认开，且能被显式关掉', () => {
    expect(DEFAULT_SETTINGS.rememberPositions).toBe(true);
    const off = loadSettings(memoryStorage(JSON.stringify({ rememberPositions: false })));
    expect(off.value.rememberPositions).toBe(false);
    expect(off.outcome).toEqual({ ok: true });
  });

  it('专注 / 打字机默认关，且能被显式打开；非布尔取值点名回落默认', () => {
    expect(DEFAULT_SETTINGS.focusMode).toBe(false);
    expect(DEFAULT_SETTINGS.typewriterMode).toBe(false);
    const on = loadSettings(memoryStorage(JSON.stringify({ focusMode: true, typewriterMode: true })));
    expect(on.value.focusMode).toBe(true);
    expect(on.value.typewriterMode).toBe(true);
    const bad = loadSettings(memoryStorage(JSON.stringify({ focusMode: 'yes', typewriterMode: 1 })));
    expect(bad.value.focusMode).toBe(false);
    expect(bad.value.typewriterMode).toBe(false);
    expect(bad.outcome).toEqual({
      ok: false,
      reason: 'corrupt',
      lostKeys: ['focusMode', 'typewriterMode'],
      lostCount: 2,
    });
  });

  it('非布尔的位置记忆取值被点名并回落默认', () => {
    const read = loadSettings(memoryStorage(JSON.stringify({ rememberPositions: 'yes' })));
    expect(read.value.rememberPositions).toBe(true);
    expect(read.outcome).toEqual({ ok: false, reason: 'corrupt', lostKeys: ['rememberPositions'], lostCount: 1 });
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
    expect(loadSettings(memoryStorage(JSON.stringify({ splitRatio: 0.1 }))).value.splitRatio).toBe(SPLIT_RATIO_MIN);
    expect(loadSettings(memoryStorage(JSON.stringify({ splitRatio: 9 }))).value.splitRatio).toBe(SPLIT_RATIO_MAX);
  });

  it('分屏比例是 NaN / 字符串 / null 时点名回落 —— 不能把 NaN 存进去', () => {
    // JSON 里没有 NaN，但 `JSON.stringify(NaN)` 会写出 `null` —— 两种都要挡住
    for (const raw of [NaN, '0.3', null]) {
      const read = loadSettings(memoryStorage(JSON.stringify({ splitRatio: raw })));
      expect(read.value.splitRatio).toBe(0.5);
      expect(read.outcome).toEqual({ ok: false, reason: 'corrupt', lostKeys: ['splitRatio'], lostCount: 1 });
    }
    const missing = loadSettings(memoryStorage(JSON.stringify({ theme: 'ink' })));
    expect(missing.value.splitRatio).toBe(0.5);
    expect(missing.outcome).toEqual({ ok: true });
  });

  it('上下限刻意不是 0 和 1 —— 那样会把另一栏压成零宽且拖不回来', () => {
    expect(SPLIT_RATIO_MIN).toBeGreaterThan(0);
    expect(SPLIT_RATIO_MAX).toBeLessThan(1);
  });

  it('存储内容坏了（不是 JSON）不抛错，回落到默认值，并如实说「整份读不出来」', () => {
    const read = loadSettings(memoryStorage('{不是 json'));
    expect(read.value).toEqual(DEFAULT_SETTINGS);
    // 整份报废时点不出名，但**必须报**。这里 lostCount 写死 1，不是 0：
    // 「丢了 0 项」在用户那里读作「什么都没丢」，那等于把一次丢失又藏了回去。
    expect(read.outcome).toEqual({ ok: false, reason: 'corrupt', lostKeys: [], lostCount: 1 });
  });

  it('存的不是对象（数组 / 数字 / null）同样按整份报废算', () => {
    for (const raw of ['[1,2]', '42', 'null']) {
      expect(loadSettings(memoryStorage(raw)).outcome).toEqual({
        ok: false,
        reason: 'corrupt',
        lostKeys: [],
        lostCount: 1,
      });
    }
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
    const read = loadSettings(hostile);
    expect(read.value).toEqual(DEFAULT_SETTINGS);
    // 读不出与读坏了是两回事：前者是环境不给读，后者是内容坏了
    expect(read.outcome).toEqual({ ok: false, reason: 'read-failed', lostKeys: [], lostCount: 1 });
    expect(() => saveSettings(hostile, DEFAULT_SETTINGS)).not.toThrow();
  });
});

describe('saveSettings', () => {
  it('往返一致', () => {
    const storage = memoryStorage();
    const settings = { ...DEFAULT_SETTINGS, theme: 'ink' as const, sidebarWidth: 300, outlineVisible: true };
    saveSettings(storage, settings);
    const read = loadSettings(storage);
    expect(read.value).toEqual(settings);
    // 自己写进去的东西自己也读得回来 —— 若这里报了丢失，告警系统本身就是坏的
    expect(read.outcome).toEqual({ ok: true });
  });

  it('写成功时返回 ok —— 成功永远安静，调用方不该说话', () => {
    expect(saveSettings(memoryStorage(), DEFAULT_SETTINGS)).toEqual({ ok: true });
  });

  it('写被拒时返回 write-failed 而不是静默吞掉（ADR 0042）', () => {
    // 之前这一条只断言「不抛」，返回值被丢掉 —— 那正是外壳无从开口的原因
    const hostile: StorageLike = {
      getItem: () => null,
      setItem() {
        throw new Error('QuotaExceededError');
      },
    };
    expect(saveSettings(hostile, DEFAULT_SETTINGS)).toEqual({ ok: false, reason: 'write-failed' });
  });

  it('没有存储时返回 no-storage —— 这是预期内的降级，不该说话', () => {
    // 与 write-failed 必须是两个值：浏览器预览 / 隐私模式本来就不打算持久化，
    // 为它弹一句「没能记住」是纯粹的噪音（每改一次设置弹一次）。
    expect(saveSettings(null, DEFAULT_SETTINGS)).toEqual({ ok: false, reason: 'no-storage' });
  });
});
