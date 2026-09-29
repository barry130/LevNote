import { describe, expect, it } from 'vitest';

import { cycleTheme, resolveTheme, systemPrefersDark, THEME_ORDER, isThemePreference } from './theme.js';

describe('resolveTheme', () => {
  it('显式主题不看系统设置', () => {
    expect(resolveTheme('paper', true)).toBe('light');
    expect(resolveTheme('ink', false)).toBe('dark');
  });

  it('跟随系统', () => {
    expect(resolveTheme('system', true)).toBe('dark');
    expect(resolveTheme('system', false)).toBe('light');
  });
});

describe('cycleTheme', () => {
  it('循环一圈回到原点', () => {
    let current = THEME_ORDER[0] ?? 'paper';
    const seen = [current];
    for (let i = 0; i < THEME_ORDER.length; i += 1) {
      current = cycleTheme(current);
      seen.push(current);
    }
    expect(seen[0]).toBe(seen[seen.length - 1]);
  });
});

describe('isThemePreference', () => {
  it('只认三种合法值', () => {
    expect(isThemePreference('paper')).toBe(true);
    expect(isThemePreference('ink')).toBe(true);
    expect(isThemePreference('system')).toBe(true);
    expect(isThemePreference('dark')).toBe(false);
    expect(isThemePreference(null)).toBe(false);
  });
});

describe('systemPrefersDark', () => {
  it('环境不支持 matchMedia 时按浅色处理', () => {
    // jsdom 没有 matchMedia —— 这里崩了会让整个应用渲染不出来
    expect(systemPrefersDark(null)).toBe(false);
    expect(systemPrefersDark(undefined)).toBe(false);
  });

  it('正常读取', () => {
    expect(systemPrefersDark(() => ({ matches: true }))).toBe(true);
  });

  it('matchMedia 抛错时也不炸', () => {
    expect(
      systemPrefersDark(() => {
        throw new Error('boom');
      }),
    ).toBe(false);
  });
});
