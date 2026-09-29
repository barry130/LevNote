/**
 * KaTeX 的**动态加载 + 渲染 + 错误翻译**。
 *
 * ## 这个文件在钉什么
 *
 * 三件编辑器与导出**共用**的事，任何一件错了都会以「两边不一致」的形式出现：
 *
 * 1. **渲染真的发生了。** 断言里必须有 MathML —— 只看有没有 `katex` 这个类名，
 *    一个「把源码原样吐回来」的实现也能骗过去。
 * 2. **失败要给出人话。** KaTeX 的 `message` 里混着「错误定位标记」（组合下划线
 *    `U+0332`），直接显示是一团乱码。翻译成人话这件事只有这里能钉。
 * 3. **安全边界。** `trust: false` 之下，公式里的 `\href` / `\url` /
 *    `\includegraphics` / `\htmlStyle` 绝不能产出真链接或真样式 —— 公式的内容
 *    来自文档，而文档可能来自别人。这条不是「理论上应该」，而是实测过
 *    KaTeX 把它们渲染成红色的命令名。
 *
 * ## 为什么不用 mock
 *
 * 这里刻意**不**替换 KaTeX。要验的正是「我们选的这几个选项在真实的 KaTeX 上
 * 是什么效果」，mock 掉之后就只剩「我调了我自己写的桩」。代价是这个文件会
 * 真的去 `import('katex')`（约 272 KB，一次性）。
 */

import { describe, expect, it } from 'vitest';

import { renderTex } from './katex.js';

describe('renderTex：成功', () => {
  it('产出含 MathML 的完整片段 —— 渲染真的发生了', async () => {
    const result = await renderTex('a^2+b^2=c^2', false);

    expect(result.ok).toBe(true);
    if (!result.ok) return;

    // MathML 是给读屏软件用的，也是「这不是源码回显」的证据
    expect(result.html).toContain('<math');
    expect(result.html).toContain('class="katex"');
    // 源码**不该**原样出现在可见文本里（它只会出现在 MathML 的 annotation 里）
    expect(result.html).toContain('<annotation encoding="application/x-tex">a^2+b^2=c^2</annotation>');
  });

  it('`displayMode` 决定是块级还是行内 —— 这个开关由节点类型给，不由公式内容决定', async () => {
    const inline = await renderTex('\\frac{1}{2}', false);
    const block = await renderTex('\\frac{1}{2}', true);

    expect(inline.ok && inline.html).not.toContain('katex-display');
    expect(block.ok && block.html).toContain('katex-display');
  });

  it('空串不抛错（编辑器自己会先短路，但导出侧不该因此崩）', async () => {
    const result = await renderTex('', false);
    expect(result.ok).toBe(true);
  });

  it('并发调用都能成功 —— 加载的是同一个 Promise，不会取两遍 chunk', async () => {
    const results = await Promise.all([
      renderTex('a', false),
      renderTex('b', false),
      renderTex('c', true),
    ]);

    expect(results.every((r) => r.ok)).toBe(true);
  });

  it('中文与 `\\text` 能用（`strict: false` 不拦它）', async () => {
    const result = await renderTex('\\text{中文}', false);
    expect(result.ok).toBe(true);
  });
});

describe('renderTex：失败', () => {
  it('不存在的命令：给出**原因**与**位置**，不带定位标记', async () => {
    const result = await renderTex('\\foo{', false);

    expect(result.ok).toBe(false);
    if (result.ok) return;

    // 这是用户看到的原文：KaTeX 的 `rawMessage` 加上第几个字符起。
    // 文案里带上位置，用户才知道该改哪儿 —— `\foo` 在只有一个字符的公式里
    // 看起来完全正常，光说「有错」等于没说
    expect(result.message).toBe('Undefined control sequence: \\foo（第 1 个字符起）');

    /**
     * **不能带组合下划线。**
     *
     * KaTeX 原始的 `message` 是
     * `KaTeX parse error: Undefined control sequence: \̲f̲o̲o̲{`，
     * 那个 `̲`（U+0332）是「错误定位标记」，用在与原文并排显示的场合。
     * 单独显示时它会把后面的字符全叠在一起 —— 用户看到的是一团乱码。
     * 所以必须走 `rawMessage` 这条路。
     */
    expect(result.message).not.toContain('\u0332');
    expect(result.message).not.toContain('KaTeX parse error');
  });

  it('另一种错法（括号没闭合）也给原因与位置', async () => {
    const result = await renderTex('x^{', false);

    expect(result.ok).toBe(false);
    if (result.ok) return;
    expect(result.message).toContain("Expected '}', got 'EOF'");
    expect(result.message).toContain('第 4 个字符起');
  });

  it('**失败之后还能成功** —— 一次失败不该被记成「永远失败」', async () => {
    const bad = await renderTex('\\nope{', false);
    const good = await renderTex('x', false);

    expect(bad.ok).toBe(false);
    // 缓存的是 Promise 而不是「成功/失败」这个结果，所以这里必然成功。
    // 这条用例防的是「把失败也缓存起来」这种改动 —— 用户把公式改对了却
    // 依然看到红字，只能重启应用
    expect(good.ok).toBe(true);
  });
});

describe('renderTex：安全边界（trust: false）', () => {
  /**
   * 公式的内容来自文档，文档可能来自别人。`trust` 打开就等于允许一份 Markdown
   * 在渲染时构造出链接、或者去取外部资源。
   *
   * 实测：KaTeX 把这些命令渲染成**红色的命令名**（`<mstyle mathcolor="#cc0000">`
   * 里包一个 `<mtext>\href</mtext>`），既不抛错、也不产出链接。所以断言要看
   * **输出里有没有链接/属性**，而不是「有没有抛错」。
   */
  it.each([
    ['\\href', '\\href{https://evil.example/x}{点我}'],
    ['\\url', '\\url{https://evil.example/x}'],
    ['\\includegraphics', '\\includegraphics{https://evil.example/a.png}'],
  ])('%s 不产出链接或外部资源引用', async (_label, tex) => {
    const result = await renderTex(tex, false);

    expect(result.ok, '这些命令不该抛错，只是不生效').toBe(true);
    if (!result.ok) return;

    expect(result.html).not.toContain('<a ');
    expect(result.html).not.toContain('href=');
    expect(result.html).not.toContain('src=');
    // 渲染成「不受信任的命令」：红色的命令名
    expect(result.html).toContain('mathcolor="#cc0000"');
  });

  it('`\\htmlStyle` 的用户样式进不了输出', async () => {
    const result = await renderTex('\\htmlStyle{color:red}{x}', false);

    expect(result.ok).toBe(true);
    if (!result.ok) return;

    /**
     * KaTeX 会把**源码原文**嵌进 MathML 的 `<annotation encoding="application/x-tex">`
     * 里（给读屏软件与「复制回 LaTeX」用）。那是一段惰性元数据、会被 HTML 转义，
     * 所以 `color:red` 出现在那里是正常的。
     *
     * 要去掉它之后再看：**其余部分**里不该有用户给的样式。输出里确实有
     * `style=` —— 那是 KaTeX 自己的版式定位（`height` / `vertical-align`），
     * 以及它给「不受信任的命令」上的红字。
     */
    const withoutAnnotation = result.html.replace(
      /<annotation[^>]*>[\s\S]*?<\/annotation>/g,
      '',
    );
    expect(withoutAnnotation).not.toContain('color:red');
    expect(withoutAnnotation).toContain('mathcolor="#cc0000"');
  });

  /**
   * 导出的 HTML 会把 KaTeX 的输出原样写进文件。所以「公式源码里的尖括号有没有
   * 被转义」是一条**真的**安全问题：一份别人给的 Markdown 里的公式，不该能撕开
   * 导出的页面。
   *
   * 这条同时也是「annotation 是惰性的」这个说法的证据 —— 源码以文本形式出现，
   * 以 `&lt;` 的形式转义。
   */
  it('源码里的尖括号被转义，撕不开导出的页面', async () => {
    const result = await renderTex('\\text{<script>alert(1)</script>}', false);

    expect(result.ok).toBe(true);
    if (!result.ok) return;

    expect(result.html).not.toContain('<script');
    expect(result.html).toContain('&lt;script&gt;');
  });
});
