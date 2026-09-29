/**
 * 公式的纯逻辑：**识别**（打字时）与**校验**（保存前）。
 *
 * ## 这个文件在钉什么
 *
 * 两条，都不是「函数返回了预期的值」这么简单：
 *
 * 1. **识别必须与解析器同源。** `inlineMathAtEnd` 给出的值最终会写进节点，
 *    而重新打开文件时那段值要由 `remark-math` 读回来。两者一旦分叉，
 *    表现是「打字时是公式、保存后变成文字」—— 用户完全无从理解。
 *    所以这里有一节直接拿**真实的解析器**对照。
 * 2. **校验拦住的必须是真的存不住的值。** `texProblem` 不是「保守一点总没错」：
 *    多拦一条，用户就写不出一个本来合法的公式。所以除了它拦的那两条，
 *    其余一律要有「往返正常」的用例作证（见最后一节）。
 *
 * ## 为什么最后一节要引入 `remark-stringify`
 *
 * 「存不住」这个判断的证据只能是**真的存一遍再读回来**。序列化器是
 * `mdast-util-math`，内核那条链路上跑的就是它（`@milkdown/core` 的 remark
 * 处理器 = `remarkParse` + `remarkStringify` + `remarkMath`）。这里用同一组
 * 包跑一遍，而不是把「我以为会坏」写进注释。
 *
 * `remark-stringify` 只出现在测试里，所以它是 devDependency。
 */

import { describe, expect, it } from 'vitest';
import remarkParse from 'remark-parse';
import remarkStringify from 'remark-stringify';
import remarkMath from 'remark-math';
import { unified } from 'unified';

import { inlineMathAtEnd, MATH_BLOCK, MATH_INLINE, TEX_DATA_ATTR, texProblem } from './math.js';

// ---------------------------------------------------------------------------
// 1. DOM 契约
// ---------------------------------------------------------------------------

describe('内核与导出的 DOM 契约', () => {
  /**
   * 这三个字符串是**跨层**的：内核的 `toDOM` 把它们写进 DOM，导出侧
   * （`core/export/inline-html.ts`）靠 `[data-tex]` 找到每一处公式。
   *
   * 它们同时是 ProseMirror 的节点名（`math_inline` / `math_block`）。
   * 改名不会有任何报错 —— 只会让导出文件里的公式变成一坨 `$…$` 源码，
   * 而编辑器里一切正常。所以在这里钉成字面量。
   */
  it('取值是字面量，改名必须是有意识的动作', () => {
    expect(MATH_INLINE).toBe('math_inline');
    expect(MATH_BLOCK).toBe('math_block');
    expect(TEX_DATA_ATTR).toBe('data-tex');
  });
});

// ---------------------------------------------------------------------------
// 2. 识别：光标前是不是刚闭合了一个行内公式
// ---------------------------------------------------------------------------

describe('inlineMathAtEnd', () => {
  it('认出贴着末尾闭合的行内公式，并给出源码长度', () => {
    // `length` 含两边的 `$`，用来算输入规则该替换掉哪一段。
    // 少了它就只能靠正则去猜边界，而边界猜错 = 每转换一次多留一个 `$`
    expect(inlineMathAtEnd('a $x^2$')).toEqual({ tex: 'x^2', length: 5 });
    expect(inlineMathAtEnd('$a$')).toEqual({ tex: 'a', length: 3 });
    expect(inlineMathAtEnd('a$b$')).toEqual({ tex: 'b', length: 3 });
    expect(inlineMathAtEnd('$\\frac{1}{2}$')).toEqual({ tex: '\\frac{1}{2}', length: 13 });
  });

  /**
   * `tex` 与 `length` **不是**一回事，这条用例专门把差别摆出来。
   *
   * micromark 只在内容**首尾同时**是空白时才剥离（`mathTextPadding`），
   * 所以 `$ x $` 的值是 `x`（2 个字符被剥掉），而源码占 5 个字符。
   * 用 `tex.length` 去算替换区间会少删两个字符，用户看到的是 `$ $x$`。
   */
  it('值被剥离了首尾空白，但长度仍是源码长度', () => {
    expect(inlineMathAtEnd('$ x $')).toEqual({ tex: 'x', length: 5 });
  });

  /**
   * 末尾还有别的字符 = 公式**早就闭合了**。
   *
   * 允许「任意位置」的话，在 `$a$ 后面还有字` 里敲任何键都会重新识别一遍，
   * 而那时公式已经转换过了 —— 结果是重复转换。
   */
  it('不贴着末尾就不算', () => {
    expect(inlineMathAtEnd('$x$ ')).toBeNull();
    expect(inlineMathAtEnd('$a$ 后面还有字')).toBeNull();
  });

  it('还没闭合的、或不是公式的一律不认', () => {
    expect(inlineMathAtEnd('')).toBeNull();
    expect(inlineMathAtEnd('$')).toBeNull();
    expect(inlineMathAtEnd('a$')).toBeNull();
    expect(inlineMathAtEnd('$$')).toBeNull();
    expect(inlineMathAtEnd('a $$')).toBeNull();
    expect(inlineMathAtEnd('$a$$')).toBeNull();
    expect(inlineMathAtEnd('$$x$')).toBeNull();
    expect(inlineMathAtEnd('$x$$')).toBeNull();
  });

  /**
   * 转义过的 `$` 不是公式 —— 这是用户**唯一**能表达「我就要两个美元符号」
   * 的写法，必须认。
   */
  it('`\\$` 转义之后不是公式', () => {
    expect(inlineMathAtEnd('\\$x$')).toBeNull();
  });

  /**
   * `$$x$$` 写在一行里，解析器认的是**行内**公式（`$$` 只是更长的围栏），
   * 不是块级。这里如实反映上游规则 —— 顺手把它当块级处理就会与
   * 「重新打开文件」不一致。
   */
  it('`$$x$$` 在同一行里是行内公式', () => {
    expect(inlineMathAtEnd('$$x$$')).toEqual({ tex: 'x', length: 5 });
    expect(inlineMathAtEnd('$$a$$')).toEqual({ tex: 'a', length: 5 });
  });

  /**
   * **已知代价，不修。**
   *
   * `singleDollarTextMath` 保持默认的 `true`（关掉它行内公式就没了），
   * 代价是「两个美元符号夹一段文字」会被当成公式 —— 这里连**尾随空格**
   * 都一起进了值（单边空白不剥）。
   *
   * 不在这里私自加判断（比如「收尾的 `$` 前面是空格就不算公式」）：加了就与
   * 解析器分叉，而解析器才是重新打开文件时说了算的那个。用户可以用 `\$` 转义。
   */
  it('`$5 到 $` 会被当成公式 —— 上游规则，不私自加判断', () => {
    expect(inlineMathAtEnd('价格 $5 到 $')).toEqual({ tex: '5 到 ', length: 6 });
  });

  it('`$ $` 的值是**一个空格**，不是空字符串', () => {
    // 单字符内容首尾是同一个空格，micromark 不剥。这一点很要紧：
    // 空格值的行内公式过不了 `texProblem`，所以输入规则必须自己拦一道
    expect(inlineMathAtEnd('$ $')).toEqual({ tex: ' ', length: 3 });
  });
});

// ---------------------------------------------------------------------------
// 3. 识别与解析器同源
// ---------------------------------------------------------------------------

describe('识别与重新打开文件时的解析同源', () => {
  const processor = unified().use(remarkParse).use(remarkMath);

  /** 解析器认为「这段文本的最后一个节点」是什么。 */
  function lastNode(text: string): { type: string; value: unknown } | null {
    const tree = processor.parse(text) as unknown as {
      children: Array<{ type: string; children?: Array<{ type: string; value?: unknown }> }>;
    };
    const last = tree.children[tree.children.length - 1];
    if (last === undefined || last.type !== 'paragraph') return null;
    const inner = last.children?.[last.children.length - 1];
    return inner === undefined ? null : { type: inner.type, value: inner.value };
  }

  /**
   * `inlineMathAtEnd` 说「是公式」时，解析器必须给出**同名、同值**的节点。
   *
   * 这是「打字时是公式、保存后变成文字」这类 bug 的防线。判定只有一处
   * （都在 `remark-math` 里），所以这条用例通过是**结构性**的，而不是巧合。
   *
   * 反方向（识别说 null 时解析器也必须说 null）**不成立**，见下一条 ——
   * 那正是这个函数比「最后一个子节点是 inlineMath」多出来的那部分判断。
   */
  it.each([
    'a $x^2$',
    '$a$',
    '$ x $',
    'a$b$',
    '价格 $5 到 $',
    '$$x$$',
    '$a\nb$',
    '$\\frac{1}{2}$',
    '$ $',
  ])('%j：识别说是公式时，解析器的最后一个节点同名同值', (text) => {
    const span = inlineMathAtEnd(text);
    expect(span, `识别应该认出 ${text}`).not.toBeNull();

    const node = lastNode(text);
    expect(node?.type, '解析器必须同意').toBe('inlineMath');
    expect(node?.value, '值必须逐字相同').toBe(span?.tex);
  });

  /**
   * **为什么不能只看「最后一个子节点是 inlineMath」。**
   *
   * remark 不会为末尾那个空格留一个文本节点，所以 `$x$ ` 的最后一个子节点
   * **就是** inlineMath —— 只看类型的话它会被误判成「公式贴着末尾」，
   * 于是在 `$a$ 后面还有字` 这种段落里敲任何键都会重复转换一次。
   *
   * 判定必须用 mdast 的**绝对偏移**（`position.end.offset === text.length`）。
   */
  it('`$x$ ` 的最后节点确实是 inlineMath，但它在末尾之前就结束了', () => {
    expect(lastNode('$x$ ')?.type).toBe('inlineMath');
    expect(inlineMathAtEnd('$x$ ')).toBeNull();
  });

  it.each(['$$', 'a $$', '\\$x$', '$a$$', '$x$$'])(
    '%j：解析器也没把它当成行内公式',
    (text) => {
      expect(lastNode(text)?.type).not.toBe('inlineMath');
      expect(inlineMathAtEnd(text)).toBeNull();
    },
  );
});

// ---------------------------------------------------------------------------
// 4. 校验
// ---------------------------------------------------------------------------

describe('texProblem', () => {
  it('空值拦下来，两种形态都拦', () => {
    // 空的行内公式**存不住**：序列化成 `$$`，读回来变成块级公式
    expect(texProblem('', 'inline')).not.toBeNull();
    // 空值的块级公式能存住，但没有任何意义 —— 用户多半是误删了内容
    expect(texProblem('', 'block')).not.toBeNull();
  });

  it('只含空白也算空 —— 用户看不见它，但它在文档里', () => {
    expect(texProblem('   ', 'inline')).not.toBeNull();
    expect(texProblem('\n', 'block')).not.toBeNull();
    expect(texProblem('\t\n ', 'inline')).not.toBeNull();
    // `$ $` 打出来的就是这一个空格值
    expect(texProblem(' ', 'inline')).not.toBeNull();
  });

  it('行内公式里不能有空行 —— 那是真的会丢内容', () => {
    expect(texProblem('a\n\nb', 'inline')).not.toBeNull();
    // 空行里带空格也一样（`\n[ \t]*\n`）
    expect(texProblem('a\n  \nb', 'inline')).not.toBeNull();
    expect(texProblem('a\n\t\nb', 'inline')).not.toBeNull();
  });

  /**
   * 块级公式**没有**空行问题：`mdast-util-math` 遇到值里有空行会换用更长的
   * 围栏（`$$$`），照样读得回来。对块级多拦一条就等于让用户写不出
   * 本来合法的公式。
   */
  it('块级公式允许空行 —— 序列化器会自己换更长的围栏', () => {
    expect(texProblem('a\n\nb', 'block')).toBeNull();
  });

  it('单个换行是合法的（`aligned` 环境靠它排版）', () => {
    expect(texProblem('a\nb', 'inline')).toBeNull();
    expect(texProblem('\\begin{aligned}a\\\\b\\end{aligned}', 'block')).toBeNull();
  });

  /**
   * **除了那两条，一律放行。**
   *
   * 多加一条无谓的限制，代价是用户写不出本来合法的公式 ——
   * 而「拦住一个合法输入」和「放进去一个坏输入」一样糟。
   */
  it.each(['$', 'a$b', '#', '---', '\\frac{1}{2}', '  a', 'a  ', '\\$', '\\text{中文}'])(
    '%j 放行',
    (tex) => {
      expect(texProblem(tex, 'inline')).toBeNull();
    },
  );
});

// ---------------------------------------------------------------------------
// 5. 存不住的两个值 —— 证据
// ---------------------------------------------------------------------------

describe('「存不住」的证据：真的存一遍再读回来', () => {
  const stringify = unified().use(remarkStringify).use(remarkMath);
  const parse = unified().use(remarkParse).use(remarkMath);

  /** 按内核的方式造文档：段落里的 inlineMath。 */
  function inlineDoc(tex: string): unknown {
    return {
      type: 'root',
      children: [{ type: 'paragraph', children: [{ type: 'inlineMath', value: tex }] }],
    };
  }

  /** 按内核的方式造文档：顶层的 math。 */
  function blockDoc(tex: string): unknown {
    return { type: 'root', children: [{ type: 'math', value: tex, meta: null }] };
  }

  /** 把读回来的文档压成一句话，方便断言。 */
  function describeTree(tree: unknown): string {
    const root = tree as { children: Array<Record<string, unknown>> };
    return (
      root.children
        .map((node) => {
          if (node['type'] === 'paragraph') {
            const children = node['children'] as Array<{ type: string; value?: unknown }>;
            const inner = children
              .map((c) => (c.type === 'inlineMath' ? `inlineMath(${JSON.stringify(c.value)})` : c.type))
              .join(',');
            return `p[${inner}]`;
          }
          if (node['type'] === 'math') return `math(${JSON.stringify(node['value'])})`;
          return String(node['type']);
        })
        .join(' + ') || '(空文档)'
    );
  }

  /** 存一遍、读回来，返回文档的形状。 */
  function roundTrip(doc: unknown): string {
    return describeTree(parse.parse(stringify.stringify(doc as never)));
  }

  /**
   * 第一个坏值：**空**。
   *
   * 行内空公式序列化成 `$$`，而那被解析成**块级**公式 —— 一个行内位置
   * 凭空冒出一个独占一段的公式，把整段文字劈成两半。这正是 `texProblem`
   * 拦空值、以及行内输入规则额外过一道 `texProblem` 的理由。
   */
  it('空的行内公式存回来会变成**块级**公式', () => {
    expect(roundTrip(inlineDoc(''))).toBe('math("")');
  });

  /**
   * 第二个坏值：**空行**。
   *
   * `$a` + 空行 + `b$` 重新解析后整段消失：公式没了、文字也散了。
   * 这是「行内公式里不能有空行」这条拦截的全部理由。
   */
  it('含空行的行内公式存回来会**整段消失**', () => {
    const tree = roundTrip(inlineDoc('a\n\nb'));
    expect(tree, '公式没了').not.toContain('inlineMath');
    expect(tree, '文字也被劈成了两段').toBe('p[text] + p[text]');
  });

  it('空行里带空格也一样会坏', () => {
    expect(roundTrip(inlineDoc('a\n  \nb'))).not.toContain('inlineMath');
  });

  /**
   * 反面：**其余的值全都往返正常**。
   *
   * 这一节是「除了那两条一律放行」的证据。序列化器会自己加空格
   * （`$` → `$$ $ $$`）或换更长的围栏（`$$` → `$$$`），不需要我们操心。
   */
  it.each(['x', 'a\nb', '$', '\\$', 'a$b', '#', '---', '\\frac{1}{2}', '  a', 'a  '])(
    '行内 %j 往返后值不变',
    (tex) => {
      expect(roundTrip(inlineDoc(tex))).toBe(`p[inlineMath(${JSON.stringify(tex)})]`);
    },
  );

  /**
   * 块级公式**一个都不坏** —— 包括空行、包括值里自带 `$$`。
   * 所以 `texProblem` 对块级只挡空值。
   */
  it.each(['', 'x', 'a\n\nb', '$$', '$$x$$', '\\frac{1}{2}', 'a\nb'])(
    '块级 %j 往返后值不变',
    (tex) => {
      expect(roundTrip(blockDoc(tex))).toBe(`math(${JSON.stringify(tex)})`);
    },
  );
});
