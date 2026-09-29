/**
 * 主题。
 *
 * 只有两个真实主题（浅色「宣纸」/ 深色「松烟」）加一个「跟随系统」。
 * **刻意不做主题市场**：主题是一组 CSS 变量（见 `src/ui/styles.css` 的
 * `[data-theme='…']` 段），加主题就是加一段 CSS，不需要运行时注册机制。
 * 一个「主题 = 一份 CSS」的模型比一个可编程的主题引擎更难被写坏。
 *
 * 解析与落地分开：
 * - `resolveTheme` 是纯函数（偏好 + 系统是否深色 → 实际用哪个），可单测；
 * - `applyTheme` 只做一件事 —— 把结果写到根元素的 `data-theme` 上。
 *   所有颜色都在 CSS 里由 `[data-theme]` 选择器决定，JS 不碰具体色值。
 *   这样「深色下某个控件没换色」永远是一个 CSS 问题，不用去翻 JS。
 */

export type ThemePreference = 'paper' | 'ink' | 'system';
export type ResolvedTheme = 'light' | 'dark';

export const THEME_ORDER: readonly ThemePreference[] = ['paper', 'ink', 'system'];

export const THEME_LABELS: Record<ThemePreference, string> = {
  paper: '宣纸（浅色）',
  ink: '松烟（深色）',
  system: '跟随系统',
};

export const DEFAULT_THEME: ThemePreference = 'paper';

export function isThemePreference(value: unknown): value is ThemePreference {
  return value === 'paper' || value === 'ink' || value === 'system';
}

export function resolveTheme(preference: ThemePreference, systemPrefersDark: boolean): ResolvedTheme {
  if (preference === 'paper') return 'light';
  if (preference === 'ink') return 'dark';
  return systemPrefersDark ? 'dark' : 'light';
}

/** 循环切换：宣纸 → 松烟 → 跟随系统 → 宣纸。 */
export function cycleTheme(preference: ThemePreference): ThemePreference {
  const at = THEME_ORDER.indexOf(preference);
  return THEME_ORDER[(at + 1) % THEME_ORDER.length] ?? DEFAULT_THEME;
}

/**
 * 把主题写到根元素。
 *
 * 同时设置 `color-scheme`：它决定 WebView 给滚动条、原生下拉、以及
 * `<input type="file">` 之类浏览器自绘控件用哪套配色。少了它，深色主题下
 * 滚动条会白得刺眼 —— 而且那是 CSS 改不到的少数几个地方之一。
 */
export function applyTheme(theme: ResolvedTheme, root: HTMLElement): void {
  root.dataset['theme'] = theme;
  root.style.colorScheme = theme;
}

/** 系统是否偏好深色。环境不支持时按浅色处理（宁可亮，不要用户在白天看到黑屏）。 */
export function systemPrefersDark(matchMedia?: ((query: string) => { matches: boolean }) | null): boolean {
  // 两级兜底都要有：`window` 可能不存在（Node 里跑纯逻辑），
  // `window.matchMedia` 也可能不存在（jsdom 就没有实现它）。
  // 少一层可选链，整个应用会在测试环境里直接崩在第一次渲染上。
  const probe = matchMedia ?? (typeof window === 'undefined' ? null : (window.matchMedia?.bind(window) ?? null));
  if (!probe) return false;
  try {
    return probe('(prefers-color-scheme: dark)').matches;
  } catch {
    return false;
  }
}
