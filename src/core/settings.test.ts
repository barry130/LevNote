import { describe, expect, it } from 'vitest';

import { DEFAULT_SETTINGS, loadSettings, saveSettings, SIDEBAR_MAX_WIDTH, SIDEBAR_MIN_WIDTH, type StorageLike } from './settings.js';

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

  it('非布尔的位置记忆取值被忽略（回落默认）', () => {
    expect(loadSettings(memoryStorage(JSON.stringify({ rememberPositions: 'yes' }))).rememberPositions).toBe(true);
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
