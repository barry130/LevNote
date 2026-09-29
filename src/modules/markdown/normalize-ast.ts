/**
 * 解析前的 mdast 规整。
 *
 * ## 为什么需要这一步
 *
 * remark 解析 `![图](a.png)`（没有标题的图片）时，mdast 节点上写的是
 * `title: null`，而不是 `title: undefined`。
 *
 * Milkdown 的 image 节点把 `title` 声明成 `{ default: "", validate: "string" }`，
 * 而 `default` **只在值为 `undefined` 时才生效**。`null` 会直接进校验，
 * 于是抛 `RangeError: Expected value of type string for attribute title on type image, got null`。
 *
 * 后果不是「报个错」而是**整张图片从文档里消失**：
 * 用户打开一篇带插图的笔记，随手改一个字保存，所有无标题的图片就没了。
 * 这是不可逆的数据丢失，属于必须堵死的红线。
 *
 * ## 为什么修在 mdast 而不是改 schema
 *
 * 换成自定义 image 节点意味着要重写它的 parse / toMarkdown / 命令，
 * 而 preset-commonmark 内部还引用着它自己那份 schema，两边会打架。
 * 在 mdast 层把 `null` 收敛成 `''`，是在**数据进入内核之前**把形状摆正，
 * 内核一行都不用改，而且链接（同样传 `title: null`）一类节点一并受益。
 *
 * ## 策略
 *
 * 只动「声明为字符串、实际是 null」的字段，不做任何语义改写：
 * 缺标题的图片仍然是缺标题，只是从「null 字符串」变成「空字符串」，
 * 序列化回 markdown 时两者都写作 `![](a.png)`，字节级一致。
 */

/** mdast 里声明为字符串、但 remark 可能填 `null` 的字段。 */
const STRING_FIELDS = ['title', 'alt', 'url', 'lang', 'identifier', 'label', 'referenceType', 'align', 'meta'] as const;

interface AstNode {
  type?: string;
  children?: AstNode[];
  [key: string]: unknown;
}

function normalizeNode(node: AstNode): void {
  for (const field of STRING_FIELDS) {
    if (node[field] === null) node[field] = '';
  }
  // align 在表格里是字符串数组（'left' | 'right' | 'center' | null），单独处理：
  // 数组里的 null 会让表格单元格的 alignment 属性校验失败
  const align = node['align'];
  if (Array.isArray(align)) {
    node['align'] = align.map((value) => (value === null ? '' : value));
  }
  if (Array.isArray(node.children)) {
    for (const child of node.children) normalizeNode(child);
  }
}

/**
 * unified 的 transformer。签名故意写得宽松：
 * 我们只关心树的形状，不需要 unified 的完整类型，也就没有必要为此引入 `unified` 依赖。
 */
export function normalizeMarkdownAst() {
  return (tree: unknown): void => {
    if (tree && typeof tree === 'object') normalizeNode(tree as AstNode);
  };
}
