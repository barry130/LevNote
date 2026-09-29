/**
 * YAML front matter 探测。
 *
 * 单独成文件而不是塞在 `index.ts` 里，是为了能被单测直接引用而不用把
 * Markdown 内核（Milkdown、highlight.js、CSS）整条依赖链拖进测试进程。
 */

/**
 * 探测 YAML front matter。
 *
 * 为什么必须探测：commonmark 不认识 front matter，会把开头的 `---` 读成
 * **水平分隔线**、把 `title: 标题` 读成**普通段落**。用户什么都没改，
 * 只要编辑过一保存，元数据就变成了 `---` 分隔线加一行正文 —— 而且是不可逆的。
 * 对 Obsidian / Hugo / Jekyll 用户来说这等于毁文件。
 *
 * 判定故意**偏向保守**（宁可误判成 front matter）：
 * 误判的代价是这个文件用纯文本打开 —— 内容安全、有提示、可解释；
 * 漏判的代价是用户的元数据被改坏。两者代价不对等，所以往安全那边偏。
 *
 * 三个条件同时成立才算：
 * 1. 第一行恰好是 `---`（允许行尾的 `\r`，因为 CRLF 文件在读进来时可能还没归一）
 * 2. 前 50 行内出现恰好是 `---` 或 `...` 的收尾行
 * 3. 收尾行之前至少有一行形如 `键: 值`
 */
export function hasFrontMatter(text: string): boolean {
  const lines = text.split('\n');
  if (lines.length < 3) return false;
  if (stripCr(lines[0] ?? '') !== '---') return false;

  const limit = Math.min(lines.length, 51);
  let sawKey = false;
  for (let i = 1; i < limit; i += 1) {
    const line = stripCr(lines[i] ?? '');
    if (line === '---' || line === '...') return sawKey;
    if (/^\s*[A-Za-z_][\w.-]*\s*:/.test(line)) sawKey = true;
  }
  return false;
}

function stripCr(line: string): string {
  return line.endsWith('\r') ? line.slice(0, -1) : line;
}
