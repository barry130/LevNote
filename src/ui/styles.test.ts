/**
 * 样式表的**所有权**测试。
 *
 * ## 为什么样式也需要测试
 *
 * 这个项目的测试几乎都在钉「行为」，而 CSS 一直靠人眼。代价是：**一个从初始提交
 * 就存在的 bug 躺了九个里程碑才被撞上** —— `styles.css` 给 `.qm-editor` 声明了
 * `overflow: hidden`，而内核的 `markdown.css` 声明的是 `overflow: auto`。
 * 同一个属性两个来源，谁赢只看打包顺序；外壳那份在后，于是**长文档被裁掉且滚不动**。
 *
 * 这条链路 jsdom 永远照不出来（没有布局引擎，也没有滚动），而人眼只在
 * 「刚好打开一份够长的文档」时才会看见。所以这里改钉一件**结构事实**：
 *
 * > **内核根节点的滚动由内核自己声明，外壳不许插手。**
 *
 * 它不依赖布局引擎，也不需要浏览器，但正好覆盖那类「外壳悄悄推翻内核」的错误 ——
 * 和硬约束 11（外壳不许按类名认内核画的东西）是同一个病的两种表现：
 * 外壳对内核算计的**内容**动手。
 */

import postcss from 'postcss';
import { describe, expect, it } from 'vitest';

import markdownCss from '../modules/markdown/markdown.css?raw';
import stylesCss from './styles.css?raw';

/**
 * 某个选择器上声明过的属性名（按出现顺序）。
 *
 * 选择器列表按逗号拆开、**逐条精确比对**：`.qm-editor .qm-cb > pre` 不算命中
 * `.qm-editor` —— 那是另一个盒子，滚动归它自己。不这么做的话，
 * `.qm-editor .qm-cb > pre { overflow-x: auto }`（代码块的横向滚动，正当）会被误报。
 */
function propertiesOn(css: string, selector: string): string[] {
  const names: string[] = [];
  postcss.parse(css).walkRules((rule) => {
    if (!rule.selectors.some((item) => item.trim() === selector)) return;
    rule.walkDecls((decl) => {
      names.push(decl.prop);
    });
  });
  return names;
}

describe('滚动容器的所有权', () => {
  /**
   * 正向的那一半。没有它，下面两条会退化成「两边都没声明」——
   * 那比冲突更糟：滚动条会彻底消失。
   */
  it('Markdown 内核自己声明了 .qm-editor 的滚动', () => {
    expect(propertiesOn(markdownCss, '.qm-editor')).toContain('overflow');
  });

  /**
   * 这是被修掉的那个 bug 本身。
   *
   * 外壳**确实**要碰 `.qm-editor`，但只碰「怎么被压进 flex 容器」——
   * `flex` / `min-height` 是布局要求，少了长文档会把外壳撑出第二条滚动条。
   * 而 `overflow` 是内核的选择（Markdown 滚 `.qm-editor`，CodeMirror 滚
   * 内层的 `.cm-scroller`），外壳无从判断，也就不该表态。
   */
  it('外壳样式表不碰 .qm-editor 的 overflow，但保留压扁它的布局属性', () => {
    const props = propertiesOn(stylesCss, '.qm-editor');

    expect(props.filter((prop) => prop.startsWith('overflow'))).toEqual([]);
    expect(props).toContain('flex');
    expect(props).toContain('min-height');
  });

  /**
   * 同一次修复里的第二处。`height: 100%` 与内核要的 `min-height: 100%`
   * **语义不同**，不是同义写法：
   *
   * - `min-height: 100%` → 至少一屏，内容多了就长高，由父级滚出去
   * - `height: 100%`     → 钉死在容器高度，内核那句 `padding-bottom: 40vh`
   *                        （让最后一行能滚到屏幕中间）会被算进固定高度里而失效
   *
   * 所以这里禁的是 `height`，`min-height` 照旧可以写。
   */
  it('外壳样式表不把 .ProseMirror 的高度钉死', () => {
    const props = propertiesOn(stylesCss, '.qm-editor .ProseMirror');

    expect(props.filter((prop) => prop === 'height')).toEqual([]);
  });

  /**
   * CodeMirror 那边**反过来**：它的滚动在 `.cm-scroller` 上，
   * 所以外层 `.qm-code` 必须 `overflow: hidden`，而且 `.cm-editor`
   * 必须有一个显式高度，否则 CodeMirror 算不出滚动区。
   *
   * 钉这一条是为了防止「照着上面两条一起改掉」——那会把代码编辑器弄坏。
   */
  it('CodeMirror 相反：外层拦住滚动、内层给足高度', () => {
    expect(propertiesOn(stylesCss, '.qm-code')).toContain('overflow');
    expect(propertiesOn(stylesCss, '.qm-code .cm-editor')).toContain('height');
  });
});
