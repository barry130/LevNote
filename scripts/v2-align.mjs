/**
 * V2 的**数据半边**：`.mindmap`（`docs/format-spec.md` §7.1 的 v0 骨架）
 * 与 mind-elixir 的 `MindElixirData` 之间，往返一次到底丢什么。
 *
 * ## 为什么这一半可以离线跑，而且必须先跑
 *
 * 「500 节点卡不卡」要真实浏览器（见 `v2-run.mjs`），但「格式对不对得上」
 * 是**纯数据**问题 —— 而且它是**更贵**的那一半：格式定错了，用户的文件会
 * 在换内核 / 升级版本时**静默丢内容**（硬约束 2）。性能不行可以换内核，
 * 格式丢了就找不回来了。
 *
 * ## 为什么要有这个脚本，而不是在文档里手写一张表
 *
 * 手写的表会**随依赖升级而失真**，而且没人知道它什么时候失真的。
 * 这个脚本每次都从**真实包**里读官方示例数据、真的走一遍往返，
 * 所以「丢了哪几个字段、丢了几处」是**读数**，不是回忆。
 *
 * 它还会断言 `docs/format-spec.md` 里的 §7.1 仍然列着脚本认识的那几个字段 ——
 * 规范改了而脚本没改时会**直接报错**，而不是安静地拿旧假设继续算。
 *
 * ## 用法
 *
 * 这个脚本从一个**外部目录**读 mind-elixir，而不是从本仓库的 `node_modules`：
 * 它是**验证工具**，不该依赖「产品依赖已经装好」这件事 ——
 * 任何时候都能拿一份干净的包重新跑一遍对齐验证。
 *
 * > **一处需要纠正的解读**：这里原先写着「按 ADR 0007，`mind-elixir`
 * > **不进本仓库的 `package.json`**」—— 那是把 ADR 0007 的后果 1 读反了。
 * > 后果 1 的原话是「**必须锁死 `mind-elixir@5.15.1` 并在 `package.json` 里
 * > 使用精确版本（不加 `^`）**」。产品代码 `import MindElixir from 'mind-elixir'`
 * > 时必须有依赖声明，否则打包器、类型检查、测试三处都解析不了。
 * > 见 ADR 0007 文末的「落地记录」。
 *
 * ```bash
 * npm i --prefix /tmp/qingmo-v2 mind-elixir@5.15.1
 * node scripts/v2-align.mjs --pkg /tmp/qingmo-v2/node_modules/mind-elixir
 * ```
 *
 * 找不到包时脚本会给出这条命令并退出 —— 不会拿一份猜出来的字段表糊过去。
 */

import { readFileSync } from 'node:fs';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';

// ---------------------------------------------------------------------------
// 参数
// ---------------------------------------------------------------------------

const argv = process.argv.slice(2);
let pkgDir = process.env.MIND_ELIXIR_DIR ?? null;
for (let i = 0; i < argv.length; i += 1) {
  if (argv[i] === '--pkg') {
    pkgDir = argv[i + 1];
    i += 1;
  }
}

if (pkgDir === null) {
  console.error('用法：node scripts/v2-align.mjs --pkg <mind-elixir 包目录>');
  console.error('  npm i --prefix /tmp/qingmo-v2 mind-elixir@5.15.1');
  process.exit(2);
}
pkgDir = resolve(pkgDir);
if (!existsSync(resolve(pkgDir, 'dist/example.js'))) {
  console.error(`这个目录里没有 mind-elixir 的 dist：${pkgDir}`);
  console.error('  npm i --prefix /tmp/qingmo-v2 mind-elixir@5.15.1');
  process.exit(2);
}

// ---------------------------------------------------------------------------
// §7.1 的骨架 —— 唯一的真相来源是那份规范
// ---------------------------------------------------------------------------

/**
 * 脚本认识的 §7.1 字段。**这不是真相来源**，真相来源是 `docs/format-spec.md`；
 * 下面 `assertSpecStillMatches()` 会在两者不一致时让脚本停下。
 */
const SPEC_ROOT_FIELDS = ['format', 'version', 'layout', 'root'];
const SPEC_NODE_FIELDS = ['id', 'topic', 'note', 'collapsed', 'style', 'children'];

function assertSpecStillMatches() {
  const spec = readFileSync(new URL('../docs/format-spec.md', import.meta.url), 'utf8');
  const section = spec.slice(spec.indexOf('### 7.1'), spec.indexOf('### 7.2'));
  if (section === '') throw new Error('docs/format-spec.md 里找不到 §7.1');

  const missing = [...SPEC_ROOT_FIELDS, ...SPEC_NODE_FIELDS].filter((field) => !section.includes(`"${field}"`));
  if (missing.length > 0) {
    throw new Error(
      `§7.1 的骨架变了，而 scripts/v2-align.mjs 还停在旧假设上。\n` +
        `  §7.1 里现在找不到这些字段名：${missing.join(', ')}\n` +
        `  改脚本里的 SPEC_ROOT_FIELDS / SPEC_NODE_FIELDS，再看一遍结论是否还成立。`,
    );
  }
}

// ---------------------------------------------------------------------------
// 样本
// ---------------------------------------------------------------------------

/** 官方示例（`dist/example.js`）—— 真实使用过的数据长什么样。 */
async function loadOfficialSample() {
  const module = await import(pathToFileURL(resolve(pkgDir, 'dist/example.js')).href);
  return module.default;
}

/**
 * 覆盖 `NodeObj` **全部字段**的合成样本。
 *
 * 官方示例碰巧没用上 `branchColor` / `metadata` / `arrow` 的每个选项 ——
 * 只测它会把「骨架装不下的字段」少算几个。这份样本按类型定义逐条造，
 * 缺一个字段就会在这里暴露出来。
 */
function buildFullSample() {
  return {
    nodeData: {
      id: 'root',
      topic: '根主题',
      note: '根节点备注',
      expanded: true,
      style: { fontSize: '20px', color: '#16302a', background: '#ffffff', fontWeight: '700' },
      tags: ['标签一', { text: '带样式的标签', style: { color: '#f00' } }],
      icons: ['🌟'],
      hyperLink: 'https://example.com/',
      image: { url: 'a.png', width: 10, height: 10, fit: 'contain' },
      branchColor: '#2e7d6a',
      metadata: { 自定义: '任意结构' },
      dangerouslySetInnerHTML: '<b>自定义 HTML</b>',
      children: [
        { id: 'n1', topic: '分支一', note: '', expanded: false, direction: 0, children: [] },
        { id: 'n2', topic: '分支二', children: [{ id: 'n2a', topic: '叶子', children: [] }] },
      ],
    },
    arrows: [{ id: 'a1', label: '箭头', from: 'n1', to: 'n2a' }],
    summaries: [{ id: 's1', label: '摘要', parentId: 'root', start: 0, end: 1 }],
    direction: 2,
    theme: { name: 'Latte', type: 'light', palette: ['#dd7878'] },
    compact: true,
    meta: { 作者: '某人' },
  };
}

// ---------------------------------------------------------------------------
// 候选适配层（v0 骨架 ↔ 内核数据）
// ---------------------------------------------------------------------------

/**
 * 内核数据 → §7.1 v0。
 *
 * **只搬骨架里有的字段。** 这就是「对齐损耗」的定义：没搬的字段在写盘那一刻
 * 就不存在了，而用户看不见 —— 他打开时图还在，只是标签、图标、链接、
 * 双向箭头、摘要全没了。
 */
function toV0(data) {
  const layout = LAYOUT_BY_DIRECTION[data.direction ?? 1];
  return {
    format: 'qingmo.mindmap',
    version: 1,
    layout,
    root: toV0Node(data.nodeData),
  };
}

const LAYOUT_BY_DIRECTION = { 0: 'leftLogic', 1: 'logicalStructure', 2: 'bidirectional', 3: 'orgChart' };
const DIRECTION_BY_LAYOUT = Object.fromEntries(Object.entries(LAYOUT_BY_DIRECTION).map(([k, v]) => [v, Number(k)]));

function toV0Node(node) {
  const out = { id: node.id, topic: node.topic, note: node.note ?? '', collapsed: node.expanded === false, style: node.style ?? {} };
  if (Array.isArray(node.children) && node.children.length > 0) out.children = node.children.map(toV0Node);
  else out.children = [];
  return out;
}

/** §7.1 v0 → 内核数据。 */
function fromV0(document) {
  return {
    nodeData: fromV0Node(document.root),
    direction: DIRECTION_BY_LAYOUT[document.layout] ?? 1,
  };
}

function fromV0Node(node) {
  const out = { id: node.id, topic: node.topic };
  if (typeof node.note === 'string' && node.note !== '') out.note = node.note;
  // `expanded` 缺省即展开（内核源码里判的是 `expanded !== false`），
  // 所以 `collapsed: false` **不写**这个字段 —— 写 `true` 也对，但那会
  // 让「从没折过」和「折过又展开」两种状态在文件里长得一样
  if (node.collapsed === true) out.expanded = false;
  if (node.style !== undefined && Object.keys(node.style).length > 0) out.style = node.style;
  out.children = (node.children ?? []).map(fromV0Node);
  return out;
}

// ---------------------------------------------------------------------------
// 量：走了这一趟之后，哪些字段不见了
// ---------------------------------------------------------------------------

/** 收集一份数据里用到的**节点字段**（不含 children，它两边都有）及其出现次数。 */
function countNodeFields(root) {
  const counts = new Map();
  const walk = (node) => {
    for (const key of Object.keys(node)) {
      if (key === 'children') continue;
      counts.set(key, (counts.get(key) ?? 0) + 1);
    }
    for (const child of node.children ?? []) walk(child);
  };
  walk(root);
  return counts;
}

function countTopFields(data) {
  const counts = new Map();
  for (const key of Object.keys(data)) counts.set(key, 1);
  return counts;
}

/** 骨架装不下的字段（按出现次数从多到少）。 */
function lost(counts, representable) {
  return [...counts.entries()].filter(([key]) => !representable.has(key)).sort((a, b) => b[1] - a[1]);
}

const TOP_REPRESENTABLE = new Set(['nodeData']);
const NODE_REPRESENTABLE = new Set(['id', 'topic', 'note', 'expanded', 'style', 'children']);
/**
 * §7.1 里**换了名字**的字段。
 *
 * `expanded` → `collapsed`：名字变了，语义还在。算进「装不下」会把损失夸大，
 * 而夸大的数字会让人去做一件不必做的事（把内核字段全搬进规范）。
 */
const RENAMED = new Set(['expanded']);

// ---------------------------------------------------------------------------
// 跑
// ---------------------------------------------------------------------------

function fmt(counts) {
  return counts.map(([key, n]) => (n > 1 ? `${key}×${n}` : key)).join('、');
}

function report(label, data) {
  const topCounts = countTopFields(data);
  const nodeCounts = countNodeFields(data.nodeData);

  const v0 = toV0(data);
  const back = fromV0(v0);
  const v0Again = toV0(back);

  const lostTop = lost(topCounts, TOP_REPRESENTABLE);
  const lostNode = lost(nodeCounts, NODE_REPRESENTABLE);
  const lostNodeOccurrences = lostNode.reduce((sum, [, n]) => sum + n, 0);

  // 「装不下」与「丢内容」不是一回事，分开报：
  // `expanded` 在 §7.1 里叫 `collapsed`，名字换了但语义在 —— 算进损失会夸大
  const renames = [...nodeCounts.keys()].filter((key) => RENAMED.has(key));

  console.log(`\n### ${label}`);
  console.log(`  节点数：${countNodes(data.nodeData)}`);
  console.log(`  顶层字段：${fmt([...topCounts.entries()])}`);
  console.log(`  节点字段：${fmt([...nodeCounts.entries()])}`);
  console.log('  ── 写盘一次之后 ──');
  console.log(`  顶层装不下：${lostTop.length === 0 ? '（无）' : fmt(lostTop)}`);
  console.log(`  节点装不下：${lostNode.length === 0 ? '（无）' : fmt(lostNode)}`);
  console.log(`  → 一共 ${lostNodeOccurrences} 处字段在写盘那一刻消失`);
  console.log(`  改名但语义还在：${renames.length === 0 ? '（无）' : renames.join('、')}`);
  console.log(`  能表示的节点字段：${[...NODE_REPRESENTABLE].filter((key) => key !== 'children').join('、')}`);

  // 二次往返是否稳定（幂等）—— 不稳定的话「打开→保存→再打开」会一直改文件
  const stable = JSON.stringify(v0) === JSON.stringify(v0Again);
  console.log(`  §7.1 二次往返稳定：${stable ? '是' : '**否**'}`);

  return { lostTop, lostNode, lostNodeOccurrences, stable };
}

function countNodes(root) {
  return 1 + (root.children ?? []).reduce((sum, child) => sum + countNodes(child), 0);
}

/** 语义对齐检查：`collapsed` ↔ `expanded` 的方向，以及缺省值。 */
function reportSemantics() {
  console.log('\n### 语义对齐（两个容易搞反的地方）');

  const collapsed = toV0({ nodeData: { id: 'a', topic: 't', expanded: false, children: [] } });
  const expanded = toV0({ nodeData: { id: 'a', topic: 't', expanded: true, children: [] } });
  const absent = toV0({ nodeData: { id: 'a', topic: 't', children: [] } });
  console.log(`  expanded: false  → collapsed: ${collapsed.root.collapsed}`);
  console.log(`  expanded: true   → collapsed: ${expanded.root.collapsed}`);
  console.log(`  expanded 缺省    → collapsed: ${absent.root.collapsed}`);
  console.log('  （内核源码判的是 `expanded !== false`，所以**缺省 = 展开**。');
  console.log('    规范里写 `collapsed: false` 是对的，但读回来时不能把「缺省」当成折叠。）');

  console.log('');
  console.log('  §7.1 的 layout 只有一个值 logicalStructure，而内核有四个方向：');
  for (const [value, name] of Object.entries(LAYOUT_BY_DIRECTION)) console.log(`    direction ${value} → ${name}`);
  console.log('  规范里 `layout` 目前只固定了 logicalStructure 一个 —— 另外三个方向写不进去。');
}

async function main() {
  assertSpecStillMatches();

  console.log('# V2 · `.mindmap`（§7.1 v0）与 mind-elixir 的对齐损耗');
  console.log(`\nmind-elixir 包：${pkgDir}`);
  console.log(`§7.1 顶层字段：${SPEC_ROOT_FIELDS.join('、')}`);
  console.log(`§7.1 节点字段：${SPEC_NODE_FIELDS.join('、')}`);

  const official = await loadOfficialSample();
  const full = buildFullSample();

  report('样本一：官方示例（dist/example.js）', official);
  report('样本二：覆盖全部字段的合成样本', full);
  reportSemantics();

  console.log('\n### 结论');
  console.log('  §7.1 v0 是**有意的最小公因式**，所以装不下内核的富字段是设计内的。');
  console.log('  问题不在「装不下」，而在**装不下的部分没有任何落脚点** ——');
  console.log('  写盘那一刻它们就没了，而用户打开时图还在（只是标签 / 图标 / 链接 /');
  console.log('  双向箭头 / 摘要 / 自定义样式全不见）。这违反硬约束 2（绝不静默丢内容）。');
  console.log('  解法不是把内核的字段全搬进规范（那是绑定内核），而是给 v1 加一个');
  console.log('  **原样透传的桶**（见 docs/format-spec.md §7.1 的 v1 提案）。');
}

await main();
