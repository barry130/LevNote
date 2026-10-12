/**
 * 内置命令清单。
 *
 * 这里是**能力的唯一清单**：工具栏按钮、命令面板、快捷键都从这一份派生。
 * 因此不存在「加了按钮忘了加快捷键」或「面板里搜不到这个功能」的可能。
 *
 * 命名约定：`域.动作`。域只用于面板分组与将来的快捷键自定义界面。
 */

import type { CommandRegistry } from './registry.js';
import type { Command, CommandContext } from './types.js';
import type { BlockFormat, InlineFormat } from '../editor/types.js';
import { ENCODINGS, encodingSlug, encodingLabel } from '../encodings.js';
import { describeRoundTrip, formatRoundTripReport } from '../fidelity.js';
import { t, type MessageKey } from '../i18n/index.js';
import { extractToReference, flattenReference } from '../document/block-ops.js';
import { asAppError } from '../document/types.js';
import { isDesktop, printHtml } from '../platform/index.js';
import { basename, dirname, joinPath } from '../path.js';
import { buildSampleDocument, SAMPLE_DOCUMENT_NAME } from '../sample-document.js';
import { newFileContent } from '../new-file-content.js';

/**
 * 命令文案的**惰性求值**（2026-10-09，#TASK-I18N-CMD-LIVE-01）。
 *
 * ## 为什么不能是 `title: t('...')`
 *
 * 这个模块产出的是**模块级常量**：`title: t(...)` 在模块加载那一刻求值并烤进
 * 对象，而那一刻语言还停在默认的 zh-CN。运行中切语言时，重注册拿到的是
 * **同一批已经烤死的对象** —— 「重注册」于是成了一次无效动作，菜单下拉项、
 * 命令面板条目、快捷键表全都停在旧语言。
 *
 * 所以内置命令在这里**只声明 key**，`title` / `category` 是**读的那一刻才查表**
 * 的 getter。与 `ViewSwitcher` / `SlashMenu` / `encodings.ts` 已立的纪律同源
 * （ADR 0031 复评记录二：静态清单存 key、渲染时才 `t()`）—— 命令面当初漏了这刀。
 *
 * ## 为什么是 getter，而不是把 `Command.title` 改成函数
 *
 * `Command` 是外壳**按契约消费**的形状：菜单、命令面板、快捷键表三处都读
 * `command.title` 当字符串。改成函数等于让每个调用方各写一次 `title()`，
 * 漏一处就是屏幕上的一行 `function () { ... }`。getter 让「消费方零改动」
 * 与「求值时机正确」同时成立，`rankCommands`（模糊搜索）之类的纯函数一行不用动。
 *
 * ## 编译期就拦住「又烤一次」
 *
 * `BuiltInCommandSpec` **不接受**字面量 `title` / `category` —— 写了就是编译错误。
 * 那是这个缺陷的形状本身，不该靠人记得别写。数据类命令（最近打开的文件）
 * 走 `titleFrom`，与 key 同一条惰性路径，只是标题的来源不是目录。
 */
type TitleInput =
  | { readonly titleKey: MessageKey; readonly titleFrom?: undefined }
  | { readonly titleFrom: () => string; readonly titleKey?: undefined };

type BuiltInCommandSpec = Omit<Command, 'title' | 'category'> &
  TitleInput & {
    /** 分组 key。与 `titleKey` 同一理由 —— 快捷键表按它分组显示。 */
    readonly categoryKey: MessageKey;
  };

/**
 * 把一条「只声明 key」的规格变成命令对象。
 *
 * 命令对象上**不留 `titleKey` / `categoryKey`**（解构时摘掉）：它们只是构造
 * 输入，不该出现在 `Object.keys(command)` 里。id / 快捷键 / keywords / enabled /
 * run 全都原样搬过去 —— 因此它们的行为**一字未变**，变的只是文案的求值时机。
 */
function builtInCommand(spec: BuiltInCommandSpec): Command {
  const { titleKey, titleFrom, categoryKey, ...body } = spec;
  return {
    ...body,
    get title() {
      if (titleKey !== undefined) return t(titleKey);
      return titleFrom === undefined ? body.id : titleFrom();
    },
    get category() {
      return t(categoryKey);
    },
  };
}

function activeDocument(ctx: CommandContext) {
  return ctx.workspace.activeStore?.getSnapshot().doc ?? null;
}

function hasDocument(ctx: CommandContext): boolean {
  return activeDocument(ctx) !== null;
}

/**
 * 图片导出是**可选能力**（ADR 0050）：内核实现了 `exportImage` 命令才可用，
 * 没实现就整条命令禁用 —— 与 `diagnostics.roundTrip` 同一个形状。
 * 绝不按内核 id 分支（AGENTS.md 红线）：会画图的内核接上契约就自动亮起来。
 */
function hasImageExport(ctx: CommandContext): boolean {
  const doc = activeDocument(ctx);
  return doc !== null && typeof ctx.ui.editor()?.exportImage === 'function';
}

/**
 * 「在磁盘上起一份内容，然后打开它」——`file.new` / `file.newMindmap` /
 * 示例文档共用的**一条编排**（ADR 0048）。
 *
 * ## 为什么不各写各的
 *
 * 这条编排只有四步，但它有**三条会出事的语义**：已存在绝不覆盖（硬约束 2）、
 * 写失败必须说出理由、失败后不能继续 open。抄三份的下场是
 * 「给 `file.new` 补上了失败原因，示例文档那条忘了」——
 * 而示例文档恰恰是唯一一条**允许已存在就直接打开**的，差异被抄开之后就再也对不上了。
 *
 * 所以差异只留**数据**：目标已存在时说什么（`null` = 不说，直接打开），
 * 写失败时说什么（收一个 `reason`，系统抛的原话）。
 *
 * `stat` 的 try/catch 是**故意**的：抛 = 文件不存在，那正是「新建」想要的情形；
 * 反过来「先看不存在、再写」在别的程序抢先写入时会覆盖掉别人的文件。
 * 真正的原子性在 `create: true` 与 Rust 侧的 `create_new(true)` 上。
 */
async function writeNewFileThenOpen(
  ctx: CommandContext,
  spec: {
    readonly path: string;
    readonly text: string;
    /** 通知的标题。 */
    readonly title: string;
    /** 已存在时要说的那句。`null` = 不说，直接打开（示例文档的语义）。 */
    readonly existsNotice: string | null;
    /** 写失败时要说的那句。`reason` 是系统抛出来的原话，不翻译。 */
    readonly failedNotice: (reason: string) => string;
  },
): Promise<void> {
  let exists = true;
  try {
    await ctx.provider.statFile(spec.path);
  } catch {
    exists = false;
  }
  if (exists && spec.existsNotice !== null) {
    await ctx.ui.notify(spec.existsNotice, spec.title);
    return;
  }
  if (!exists) {
    try {
      await ctx.provider.writeTextFile({ path: spec.path, text: spec.text, create: true });
    } catch (thrown) {
      await ctx.ui.notify(spec.failedNotice(asAppError(thrown).message), spec.title);
      return;
    }
  }
  await ctx.workspace.open(spec.path);
}

interface ExportInput {
  readonly path: string;
  readonly title: string;
  readonly bodyHtml: string | null;
  readonly fallbackText: string;
}

/**
 * 两种导出（自包含 HTML / 打印 PDF）共用的输入组装。
 *
 * 组装只做一件事：问内核要语义 HTML（要不到就给原文）。渲染在导出模块里，
 * 出口在各自的动作里 —— 两边共用同一份 `buildSelfContainedHtml` 结果，
 * 「导出的 HTML」和「打印出来的 PDF」永远不会长得不一样。
 */
async function composeExportInput(ctx: CommandContext): Promise<ExportInput | null> {
  const doc = activeDocument(ctx);
  if (!doc) return null;

  const handle = ctx.ui.editor();

  // toHtml() 返回 null 是**合法结果**（纯文本内核就是如此），不是错误。
  // 抛错也按 null 处理：导出成纯文本总比导不出来强。
  let bodyHtml: string | null;
  try {
    bodyHtml = handle ? await handle.toHtml() : null;
  } catch {
    bodyHtml = null;
  }

  return {
    path: doc.path,
    title: doc.displayName.replace(/\.[^.]+$/, ''),
    bodyHtml,
    fallbackText: doc.text,
  };
}

async function runExport(ctx: CommandContext): Promise<void> {
  const input = await composeExportInput(ctx);
  if (input === null) return;

  /**
   * 导出模块**用在时才加载**（P3 第 18 项）：它拖着 hljs 与整套 remark
   * （为了公式与代码块的 HTML 化），而导出是一个低频动作 —— 让主包为了
   * 它多解析三百 KB 不划算。动态 import 之余没有任何行为变化。
   */
  const { exportHtml } = await import('../export/export-document.js');
  const result = await exportHtml({ ...input, provider: ctx.provider });

  if (result.status === 'saved') await ctx.ui.notify(result.message, t('core.cmd.exportDone'));
  else if (result.status === 'failed') await ctx.ui.notify(result.message, t('core.cmd.exportFailed'));
}

async function runPrintExport(ctx: CommandContext): Promise<void> {
  const input = await composeExportInput(ctx);
  if (input === null) return;

  // 与 HTML 导出共用同一个按需加载理由：渲染模块不进主包（P3 第 18 项）
  const { printDocument } = await import('../export/print-document.js');
  const result = await printDocument({ ...input, provider: ctx.provider, printHtml });

  if (result.status === 'opened') await ctx.ui.notify(result.message, t('core.cmd.printTitle'));
  else await ctx.ui.notify(result.message, t('core.cmd.exportFailed'));
}

/**
 * 图片导出（ADR 0050）：请内核把自己画成一张图，外壳只管问存到哪、写盘。
 *
 * 与 `runExport` 是**并列**的两条导出路：那边要语义 HTML（文本管线），这边要
 * 二进制 —— 画法长在内核里，这里从头到尾不出现任何具体内核的名字。
 * 导图、图纸（ADR 0050 / 0051）先后接上的就是这同一条路，一行没改；
 * diagram 接上时还是它。PNG / SVG 共用这一个工厂，差的只有 `format`。
 */
function runExportImage(format: 'png' | 'svg') {
  return async (ctx: CommandContext): Promise<void> => {
    const doc = activeDocument(ctx);
    if (doc === null) return;
    const handle = ctx.ui.editor();
    // bind 住 handle：契约方法是内核闭包外的普通方法，别的内核实现可能用 this
    const draw = handle === null ? undefined : handle.exportImage?.bind(handle);
    if (draw === undefined) return;

    // 与导出家族同一个按需加载理由（P3 第 18 项）
    const { exportImage } = await import('../export/export-image.js');
    const result = await exportImage({
      title: doc.displayName.replace(/\.[^.]+$/, ''),
      format,
      draw: () => draw(format),
      provider: ctx.provider,
    });

    if (result.status === 'saved') await ctx.ui.notify(result.message, t('core.cmd.exportDone'));
    else if (result.status === 'failed') await ctx.ui.notify(result.message, t('core.cmd.exportFailed'));
  };
}

/**
 * 往返保真自检（P6 第 29 项，ADR 0033）：拿磁盘上的原文走一遍内核自己的
 * 「解析 → 序列化」，与原文逐行比对，如实报告「不改一个字保存会改写什么」。
 *
 * 只读不写 —— 它是给用户知情权的一个诊断动作，不是「修复」：
 * 报告里的差异是内核**故意保留**的规范化（表格重排之类），改不改由人决定。
 */
async function runRoundTripCheck(ctx: CommandContext): Promise<void> {
  const doc = activeDocument(ctx);
  if (doc === null) return;
  const handle = ctx.ui.editor();
  if (handle?.roundTrip === undefined) return;

  // 比对的基准是**磁盘那份**（savedText），不是当前编辑内容 ——
  // 用户改过的部分是「他的改动」，不属于「内核会改写什么」这个问题
  let roundTripped: string;
  try {
    roundTripped = await handle.roundTrip(doc.savedText);
  } catch (thrown) {
    const reason = thrown instanceof Error ? thrown.message : String(thrown);
    await ctx.ui.notify(t('core.cmd.roundTripFailed', { reason }), t('core.cmd.roundTripTitle'));
    return;
  }

  const report = describeRoundTrip(doc.savedText, roundTripped);
  await ctx.ui.notify(formatRoundTripReport(report), t('core.cmd.roundTripTitle'));
}

/**
 * 「提取为引用」的编排：把内核的文档操作与 provider 的文件操作接在一起。
 *
 * 依赖逐个注入而不是让 `block-ops.ts` 反过来 import 这里的 ctx ——
 * 这样编排逻辑可以被假依赖单测钉住（见 `block-ops.test.ts`）。
 */
async function runExtractReference(ctx: CommandContext): Promise<void> {
  const doc = activeDocument(ctx);
  const transform = ctx.ui.editor()?.blockTransform;
  if (doc === null || transform === undefined) return;
  await extractToReference({
    docPath: doc.path,
    selectedMarkdown: () => transform.selectedMarkdown(),
    replaceSelectionWith: (markdown) => transform.replaceSelectionWith(markdown),
    enclosingFence: () => transform.enclosingFence(),
    replaceEnclosingFenceWith: (markdown) => transform.replaceEnclosingFenceWith(markdown),
    readTextFile: async (path) => {
      try {
        return await ctx.provider.readTextFile(path);
      } catch {
        return null;
      }
    },
    writeTextFile: (path, text) => ctx.provider.writeTextFile({ path, text, create: true }).then(() => undefined),
    notify: (message, title) => ctx.ui.notify(message, title),
  });
}

/** 「拉平引用」的编排。与 `runExtractReference` 同一条分工。 */
async function runInlineReference(ctx: CommandContext): Promise<void> {
  const doc = activeDocument(ctx);
  const transform = ctx.ui.editor()?.blockTransform;
  if (doc === null || transform === undefined) return;
  await flattenReference({
    docPath: doc.path,
    selectedMarkdown: () => transform.selectedMarkdown(),
    replaceSelectionWith: (markdown) => transform.replaceSelectionWith(markdown),
    enclosingFence: () => transform.enclosingFence(),
    replaceEnclosingFenceWith: (markdown) => transform.replaceEnclosingFenceWith(markdown),
    readTextFile: async (path) => {
      try {
        return await ctx.provider.readTextFile(path);
      } catch {
        return null;
      }
    },
    writeTextFile: (path, text) => ctx.provider.writeTextFile({ path, text, create: true }).then(() => undefined),
    notify: (message, title) => ctx.ui.notify(message, title),
  });
}

/**
 * 「格式」与「段落」两组命令是**机械生成的**（每条只差一个格式名/块名），
 * 手写二十遍同样的 enabled 与 run 只会藏起共性。数组用显式类型标注 ——
 * `as const` 的异构数组解构不出可选的 `level`，标了类型反而直白；
 * id 刻意是**字面量联合**而不是 string：`t()` 的参数是精确的 key 联合，
 * 模板字面量在字面量联合上分发后恰好对得上（写错 key 是编译错误）。
 */
const INLINE_FORMATS = [
  // 命令 id 用「加粗」，内核的 InlineFormat 用 strong —— 两套词各归各的用户
  { id: 'bold', inline: 'strong' },
  { id: 'italic', inline: 'emphasis' },
  { id: 'strikethrough', inline: 'strikethrough' },
  { id: 'inlineCode', inline: 'inlineCode' },
] as const satisfies readonly { id: string; inline: InlineFormat }[];

const INLINE_FORMAT_COMMANDS: readonly Command[] = INLINE_FORMATS.map(({ id, inline }) => builtInCommand({
  id: `format.${id}`,
  titleKey: `command.format.${id}.title`,
  categoryKey: 'command.category.format',
  keywords: [id, 'geshi'],
  enabled: (ctx: CommandContext) => hasDocument(ctx) && ctx.ui.editor()?.formatting !== undefined,
  run: (ctx: CommandContext) => {
    ctx.ui.editor()?.formatting?.toggleInline(inline);
  },
}));

type ParagraphBlockId =
  | 'paragraph.plainText'
  | 'paragraph.heading1'
  | 'paragraph.heading2'
  | 'paragraph.heading3'
  | 'paragraph.heading4'
  | 'paragraph.heading5'
  | 'paragraph.heading6'
  | 'paragraph.blockquote'
  | 'paragraph.bulletList'
  | 'paragraph.orderedList'
  | 'paragraph.codeBlock';

const PARAGRAPH_BLOCK_COMMANDS: readonly Command[] = (
  [
    { id: 'paragraph.plainText', block: 'paragraph', level: undefined },
    { id: 'paragraph.heading1', block: 'heading', level: 1 },
    { id: 'paragraph.heading2', block: 'heading', level: 2 },
    { id: 'paragraph.heading3', block: 'heading', level: 3 },
    { id: 'paragraph.heading4', block: 'heading', level: 4 },
    { id: 'paragraph.heading5', block: 'heading', level: 5 },
    { id: 'paragraph.heading6', block: 'heading', level: 6 },
    { id: 'paragraph.blockquote', block: 'blockquote', level: undefined },
    { id: 'paragraph.bulletList', block: 'bulletList', level: undefined },
    { id: 'paragraph.orderedList', block: 'orderedList', level: undefined },
    { id: 'paragraph.codeBlock', block: 'codeBlock', level: undefined },
  ] as readonly { id: ParagraphBlockId; block: BlockFormat; level: number | undefined }[]
).map(({ id, block, level }) => builtInCommand({
  id,
  titleKey: `command.${id}.title`,
  categoryKey: 'command.category.paragraph',
  keywords: [id, 'duanluo'],
  enabled: (ctx: CommandContext) => hasDocument(ctx) && ctx.ui.editor()?.formatting !== undefined,
  run: (ctx: CommandContext) => {
    ctx.ui.editor()?.formatting?.toggleBlock(block, level);
  },
}));

const PARAGRAPH_INSERT_COMMANDS: readonly Command[] = (
  ['table', 'mathBlock', 'diagram', 'drawio', 'hr'] as const
).map((insertable) => builtInCommand({
  id: `paragraph.${insertable}`,
  titleKey: `command.paragraph.${insertable}.title`,
  categoryKey: 'command.category.paragraph',
  keywords: [insertable, 'charu', 'duanluo'],
  enabled: (ctx: CommandContext) => hasDocument(ctx) && ctx.ui.editor()?.formatting !== undefined,
  run: (ctx: CommandContext) => {
    ctx.ui.editor()?.formatting?.insertBlock(insertable);
  },
}));

const THEME_COMMANDS: readonly Command[] = (['system', 'paper', 'ink'] as const).map((preference) => builtInCommand({
  id: `theme.${preference}`,
  titleKey: `command.theme.${preference}.title`,
  categoryKey: 'command.category.theme',
  keywords: [preference, 'theme', 'zhuti'],
  run: (ctx: CommandContext) => {
    ctx.ui.setTheme(preference);
  },
}));

/**
 * 「编码」菜单的命令：一份编码一个「转为 …」。
 *
 * 标题直接用 `core/encodings.ts` 里的 `label` —— 那份数据本身就是界面文案
 * （状态栏的下拉显示的也是它），这里不再造第二个名字。可勾选状态由外壳
 * 按文档当前的编码判断（`menuChecked`），命令本身无状态。
 */
const ENCODING_COMMANDS: readonly Command[] = ENCODINGS.map((option) => builtInCommand({
  id: `encoding.set.${encodingSlug(option.encoding, option.bom)}`,
  titleFrom: () => encodingLabel(option),
  categoryKey: 'command.category.encoding',
  keywords: [option.encoding, 'bianma', 'encoding', '编码'],
  enabled: (ctx: CommandContext) => {
    const doc = activeDocument(ctx);
    return doc !== null && !doc.readonly;
  },
  run: (ctx: CommandContext) => {
    ctx.ui.setEncoding(option.encoding, option.bom);
  },
}));

/** 「编码」菜单要的命令 id 清单 —— 与命令定义同生共死，菜单不自己拼 id。 */
export const ENCODING_COMMAND_IDS: readonly string[] = ENCODING_COMMANDS.map((command) => command.id);

export const BUILT_IN_COMMANDS: readonly Command[] = [
  // ---- 文件 -------------------------------------------------------------
  builtInCommand({
    id: 'file.open',
    titleKey: 'command.file.open.title',
    categoryKey: 'command.category.file',
    keybinding: 'Mod+O',
    keywords: ['open', 'dakaishejian'],
    run: (ctx) => {
      void ctx.workspace.openViaDialog();
    },
  }),
  builtInCommand({
    id: 'file.openFolder',
    titleKey: 'command.file.openFolder.title',
    categoryKey: 'command.category.file',
    keybinding: 'Mod+Shift+O',
    keywords: ['open folder', 'workspace', 'gongzuoqu'],
    run: (ctx) => {
      void ctx.workspace.openFolder();
    },
  }),
  builtInCommand({
    /**
     * 新建文件：选位置、起名、落一个空文件、打开。
     *
     * 本产品**没有「无标题缓冲区」**——文档生命周期从磁盘上的一份文件开始
     * （打开/保存/监听/冲突检测全都以路径为主键），先造一个内存里的「新文档」
     * 再补一次另存为，等于在每条生命周期链路上都加一个「还没有路径」的状态。
     * 所以「新建」被实现成「直接在磁盘上起一个空文件」，语义反而更诚实。
     *
     * 绝不静默覆盖（硬约束 2）：另存为对话框允许指到一个已有文件上，
     * 新建撞上已有文件时明确说明并停手，而不是把它清空。
     */
    id: 'file.new',
    titleKey: 'command.file.new.title',
    categoryKey: 'command.category.file',
    keywords: ['new file', 'create', 'xinjian'],
    run: (ctx) => {
      void (async () => {
        const path = await ctx.provider.pickSavePath(t('core.cmd.untitledName'));
        if (path === null) return;
        await writeNewFileThenOpen(ctx, {
          path,
          // 内容由路径决定：选了 `.mindmap` 就落导图模板，其余落空文件
          // （`core/new-file-content.ts`）。这一条让「新建」本身也认得格式，
          // 不必非得找到那个专门的命令。
          text: newFileContent(path),
          title: t('core.cmd.newFileTitle'),
          existsNotice: t('core.cmd.newFileExists'),
          failedNotice: (reason) => t('core.cmd.newFileFailed', { reason }),
        });
      })();
    },
  }),
  builtInCommand({
    /**
     * 新建思维导图：在磁盘上落**一张真的导图**，打开即进画布（ADR 0048）。
     *
     * ## 为什么需要它，而不能让用户自己去改后缀
     *
     * `.mindmap` 的内核一直是一等公民：空 `.mindmap` 打开就会进画布。
     * 缺的是**入口**——此前菜单与命令面板里只有 `file.new`，
     * 用户得先知道「导图文件的后缀是 `.mindmap`」这件只有作者知道的事。
     *
     * ## 落模板而不是落空文件
     *
     * 空文件也合法，但画布上是一个**看不见内容的框**；落模板则给中心节点
     * 填上文件名（用户自己刚起的名字），打开就能点进去改。
     * 模板由 `core/mindmap/template.ts` 生成，永远等于当前格式。
     *
     * 已存在时同样停手并说明，与 `file.new` 走同一条 `writeNewFileThenOpen`。
     */
    id: 'file.newMindmap',
    titleKey: 'command.file.newMindmap.title',
    categoryKey: 'command.category.file',
    keywords: ['new mindmap', 'mind map', 'siweidaotu', '思维导图', '导图'],
    run: (ctx) => {
      void (async () => {
        const path = await ctx.provider.pickSavePath(t('core.cmd.untitledMindmapName'));
        if (path === null) return;
        await writeNewFileThenOpen(ctx, {
          path,
          text: newFileContent(path),
          title: t('core.cmd.newMindmapTitle'),
          existsNotice: t('core.cmd.newFileExists'),
          failedNotice: (reason) => t('core.cmd.newMindmapFailed', { reason }),
        });
      })();
    },
  }),
  builtInCommand({
    id: 'file.save',
    titleKey: 'command.file.save.title',
    categoryKey: 'command.category.file',
    keybinding: 'Mod+S',
    keywords: ['save', 'baocun'],
    enabled: (ctx) => {
      const snapshot = ctx.workspace.activeStore?.getSnapshot();
      if (snapshot === undefined || snapshot.doc === null) return false;
      if (snapshot.doc.readonly) return false;
      // 正在写盘时不再放行：连按 Ctrl+S 会排两次写，
      // 而第二次拿的 mtime 基准还是旧的，会误报一次「磁盘上的文件已被修改」
      return !snapshot.busy;
    },
    run: (ctx) => {
      void ctx.workspace.save();
    },
  }),
  builtInCommand({
    id: 'file.saveAs',
    titleKey: 'command.file.saveAs.title',
    categoryKey: 'command.category.file',
    keybinding: 'Mod+Shift+S',
    keywords: ['save as', 'lingcunwei'],
    enabled: hasDocument,
    run: (ctx) => {
      void ctx.workspace.saveAs();
    },
  }),
  builtInCommand({
    id: 'file.saveAll',
    titleKey: 'command.file.saveAll.title',
    categoryKey: 'command.category.file',
    keywords: ['save all', 'baocun', '保存全部'],
    // 一个脏标签页都没有时禁用 —— 与 file.save 的「不忙时才放行」同一原则：
    // 显示为禁用告诉用户「现在没有要保存的东西」，而不是点了没反应
    enabled: (ctx) => ctx.workspace.getSnapshot().tabs.some((tab) => tab.dirty),
    run: (ctx) => {
      void ctx.workspace.saveAll();
    },
  }),
  builtInCommand({
    id: 'file.reload',
    titleKey: 'command.file.reload.title',
    categoryKey: 'command.category.file',
    keywords: ['reload', 'chongxinjiazai'],
    enabled: hasDocument,
    run: (ctx) => {
      void ctx.workspace.reload();
    },
  }),
  builtInCommand({
    id: 'file.closeTab',
    titleKey: 'command.file.closeTab.title',
    categoryKey: 'command.category.file',
    keybinding: 'Mod+W',
    keywords: ['close', 'guanbi'],
    enabled: hasDocument,
    run: (ctx) => ctx.ui.closeActiveTab(),
  }),
  builtInCommand({
    id: 'file.exportHtml',
    titleKey: 'command.file.exportHtml.title',
    categoryKey: 'command.category.export',
    keybinding: 'Mod+Shift+E',
    keywords: ['export', 'html', 'daochu'],
    enabled: hasDocument,
    run: runExport,
  }),
  builtInCommand({
    id: 'file.exportPdf',
    titleKey: 'command.file.exportPdf.title',
    categoryKey: 'command.category.export',
    keywords: ['pdf', 'print', 'dayin', 'daochu'],
    // 浏览器预览没有打印子窗口，禁用比「点了报错」诚实；
    // printHtml 里还有一道明确拒绝，防的是别的入口直接调动作。
    enabled: (ctx) => hasDocument(ctx) && isDesktop(),
    run: runPrintExport,
  }),
  builtInCommand({
    id: 'file.exportPng',
    titleKey: 'command.file.exportPng.title',
    categoryKey: 'command.category.export',
    keywords: ['png', 'image', 'tupian', 'daochu'],
    // 不给快捷键：Mod+Shift+E 已归 HTML 导出；图片导出是低频动作，
    // 再排一个组合键只会平添一个谁也记不住的绑定。
    enabled: hasImageExport,
    run: runExportImage('png'),
  }),
  builtInCommand({
    id: 'file.exportSvg',
    titleKey: 'command.file.exportSvg.title',
    categoryKey: 'command.category.export',
    keywords: ['svg', 'image', 'tupian', 'daochu'],
    enabled: hasImageExport,
    run: runExportImage('svg'),
  }),

  // ---- 标签页 -----------------------------------------------------------
  builtInCommand({
    id: 'tab.next',
    titleKey: 'command.tab.next.title',
    categoryKey: 'command.category.tab',
    keybinding: 'Ctrl+Tab',
    keywords: ['next tab'],
    enabled: (ctx) => ctx.workspace.getSnapshot().tabs.length > 1,
    run: (ctx) => ctx.workspace.cycle(1),
  }),
  builtInCommand({
    id: 'tab.prev',
    titleKey: 'command.tab.prev.title',
    categoryKey: 'command.category.tab',
    keybinding: 'Ctrl+Shift+Tab',
    keywords: ['previous tab'],
    enabled: (ctx) => ctx.workspace.getSnapshot().tabs.length > 1,
    run: (ctx) => ctx.workspace.cycle(-1),
  }),

  // ---- 视图 -------------------------------------------------------------
  builtInCommand({
    id: 'view.toggleSidebar',
    titleKey: 'command.view.toggleSidebar.title',
    categoryKey: 'command.category.view',
    keybinding: 'Mod+B',
    keywords: ['sidebar', 'cebianlan'],
    run: (ctx) => ctx.ui.toggleSidebar(),
  }),
  builtInCommand({
    /**
     * 活动栏的「文件浏览器」。
     *
     * 它没有快捷键，也不是 `view.toggleSidebar` 的别名 —— 两者要的是两种行为：
     * `Ctrl+B` 收起 / 放出侧栏（**保留**正在看的那一栏），
     * 而这个按钮的语义是「我要看文件树」（侧栏正显示检索时应当换过去）。
     *
     * 仍然走命令而不是直接调 UI：活动栏上五个入口的可用性全部来自
     * `CommandRegistry.enabled`，这里是那套约定的第六个。
     */
    id: 'view.filesView',
    titleKey: 'command.view.filesView.title',
    categoryKey: 'command.category.view',
    keywords: ['files', 'explorer', 'wenjianliulanqi', '文件树'],
    run: (ctx) => ctx.ui.openFilesView(),
  }),
  builtInCommand({
    id: 'view.toggleOutline',
    titleKey: 'command.view.toggleOutline.title',
    categoryKey: 'command.category.view',
    keybinding: 'Mod+Shift+L',
    keywords: ['outline', 'dagang'],
    run: (ctx) => ctx.ui.toggleOutline(),
  }),
  builtInCommand({
    id: 'view.toggleSource',
    titleKey: 'command.view.toggleSource.title',
    categoryKey: 'command.category.view',
    /**
     * 与 horseMD 的「源码模式切换」同一个键位。
     *
     * `Ctrl+/` 在这里是安全的：`isAllowedInEditable` 放行带修饰键的快捷键，
     * 而 `/` 本身不带修饰键时在编辑器里就是普通输入，不会被抢走。
     */
    keybinding: 'Mod+/',
    keywords: ['source', 'markdown', 'yuanma', '源码', 'markdown source'],
    // 只有首选内核声明了 `sourceView` 的文档才可用（目前是 Markdown）。
    // 用禁用而不是隐藏：隐藏会让用户以为这个功能不存在。
    enabled: (ctx) => ctx.ui.canToggleSourceView(),
    run: (ctx) => ctx.ui.toggleSourceView(),
  }),
  builtInCommand({
    id: 'edit.extractReference',
    titleKey: 'command.edit.extractReference.title',
    categoryKey: 'command.category.edit',
    keywords: ['extract', 'reference', 'doc-block', 'tiqu', 'yinyong'],
    // 只有 Markdown 内核实现了 blockTransform（doc-block 是 Markdown 的语法）；
    // 其它内核禁用而不是隐藏 —— 与 view.toggleSource 同一条原则
    enabled: (ctx) => hasDocument(ctx) && ctx.ui.editor()?.blockTransform !== undefined,
    run: (ctx) => runExtractReference(ctx),
  }),
  builtInCommand({
    id: 'edit.inlineReference',
    titleKey: 'command.edit.inlineReference.title',
    categoryKey: 'command.category.edit',
    keywords: ['flatten', 'inline', 'doc-block', 'laping', 'zhankai'],
    enabled: (ctx) => hasDocument(ctx) && ctx.ui.editor()?.blockTransform !== undefined,
    run: (ctx) => runInlineReference(ctx),
  }),
  /**
   * 撤销 / 重做（菜单入口）。
   *
   * **刻意没有默认快捷键**：编辑器内部的 Ctrl+Z / Ctrl+Shift+Z 由内核自己的
   * keymap 处理（ProseMirror 的 history 插件、CodeMirror 的 historyKeymap），
   * 外壳若再注册一个全局 Mod+Z，`isAllowedInEditable` 放行带修饰键的绑定，
   * 两次撤销会在同一按键上叠加执行。所以这两个命令只走菜单 / 面板入口；
   * 用户确实想给它们自定义全局键时，可以在设置页录制（后果自负）。
   */
  builtInCommand({
    id: 'edit.undo',
    titleKey: 'command.edit.undo.title',
    categoryKey: 'command.category.edit',
    keywords: ['undo', 'chexiao'],
    enabled: (ctx) => hasDocument(ctx) && ctx.ui.editor()?.undo !== undefined,
    run: (ctx) => ctx.ui.editor()?.undo?.(),
  }),
  builtInCommand({
    id: 'edit.redo',
    titleKey: 'command.edit.redo.title',
    categoryKey: 'command.category.edit',
    keywords: ['redo', 'chongzuo'],
    enabled: (ctx) => hasDocument(ctx) && ctx.ui.editor()?.redo !== undefined,
    run: (ctx) => ctx.ui.editor()?.redo?.(),
  }),
  builtInCommand({
    /**
     * 复制为 HTML。
     *
     * `toHtml()` 返回 null（纯文本内核）或抛错时退回**复制原文**，并把这次
     * 降级说出来（硬约束 2）—— 用户得知道剪贴板里拿到的不是格式化结果。
     */
    id: 'edit.copyHtml',
    titleKey: 'command.edit.copyHtml.title',
    categoryKey: 'command.category.edit',
    keywords: ['copy html', 'fuzhi', '剪贴板'],
    enabled: hasDocument,
    run: (ctx) => {
      void (async () => {
        const doc = activeDocument(ctx);
        const handle = ctx.ui.editor();
        if (doc === null || handle === null) return;

        let html: string | null;
        try {
          html = await handle.toHtml();
        } catch {
          html = null;
        }

        try {
          await navigator.clipboard.writeText(html ?? doc.text);
        } catch {
          await ctx.ui.notify(t('core.cmd.clipCopyDenied'), t('command.edit.copyHtml.title'));
          return;
        }
        await ctx.ui.notify(
          html === null
            ? t('core.cmd.clipNoHtml')
            : t('core.cmd.clipHtmlOk'),
          t('command.edit.copyHtml.title'),
        );
      })();
    },
  }),
  /**
   * 三档视图的直接入口。
   *
   * ## 为什么是 `Ctrl+1/2/3` 而不是 `Ctrl+/` 那种循环
   *
   * 循环是**有状态的**：想从富文本切到分屏，得先知道「现在在哪一档、
   * 还要按几下」。三档之后这个心智负担明显变大（两档时循环很自然，
   * 三档就开始猜）。数字键是「直接跳过去」，与 VSCode 的编辑器组切换同一个思路。
   *
   * `Ctrl+/` 保留：它在两档之间来回仍然是最高频的动作，不该被迫多按一个键。
   *
   * ## `enabled` 用的是同一个判据
   *
   * 源码与分屏都要求首选内核声明了 `sourceView`（目前是 Markdown），
   * 所以三个命令里有两个共用 `canToggleSourceView()` —— 名字说的是
   * 「能不能切到源码那一侧」，而分屏正是「源码那一侧 + 预览」。
   * 富文本那一档永远可用（兜底内核也能渲染），所以它没有 `enabled`。
   */
  builtInCommand({
    id: 'view.rich',
    titleKey: 'command.view.rich.title',
    categoryKey: 'command.category.view',
    keybinding: 'Mod+1',
    keywords: ['rich', 'wysiwyg', 'fuwenben', '预览', '所见即所得'],
    run: (ctx) => ctx.ui.setViewMode('rich'),
  }),
  builtInCommand({
    id: 'view.source',
    titleKey: 'command.view.source.title',
    categoryKey: 'command.category.view',
    keybinding: 'Mod+2',
    keywords: ['source', 'yuanma', '源码', 'markdown'],
    enabled: (ctx) => ctx.ui.canToggleSourceView(),
    run: (ctx) => ctx.ui.setViewMode('source'),
  }),
  builtInCommand({
    id: 'view.split',
    titleKey: 'command.view.split.title',
    categoryKey: 'command.category.view',
    keybinding: 'Mod+3',
    keywords: ['split', 'side by side', 'bingshu', '分屏', '双栏', 'fenping'],
    enabled: (ctx) => ctx.ui.canToggleSourceView(),
    run: (ctx) => ctx.ui.setViewMode('split'),
  }),
  builtInCommand({
    id: 'view.cycleTheme',
    titleKey: 'command.view.cycleTheme.title',
    categoryKey: 'command.category.view',
    keybinding: 'Mod+Shift+T',
    keywords: ['theme', 'zhuti', 'dark', 'light'],
    run: (ctx) => ctx.ui.cycleTheme(),
  }),
  builtInCommand({
    id: 'view.toggleWordWrap',
    titleKey: 'command.view.toggleWordWrap.title',
    categoryKey: 'command.category.view',
    keywords: ['word wrap', 'huanhang', '换行', '软换行'],
    run: (ctx) => ctx.ui.toggleWordWrap(),
  }),
  builtInCommand({
    id: 'view.toggleCodeLineNumbers',
    titleKey: 'command.view.toggleCodeLineNumbers.title',
    categoryKey: 'command.category.view',
    keywords: ['line numbers', 'hanghao', '行号'],
    run: (ctx) => ctx.ui.toggleCodeLineNumbers(),
  }),
  builtInCommand({
    /**
     * 专注 / 打字机（Typora 的两个写作模式）。
     *
     * 与 `view.toggleWordWrap` 同属「设置层开关」：命令只翻偏好，
     * 哪些内核消费它由各内核决定（Markdown 两个都认；代码内核只认打字机；
     * 纯文本与导图忽略）。菜单项不按内核禁用 —— 与「行号」同一先例：
     * 偏好是全局的，当前文档用不上时切了也没害处。
     */
    id: 'view.toggleFocusMode',
    titleKey: 'command.view.toggleFocusMode.title',
    categoryKey: 'command.category.view',
    keywords: ['focus', 'zhuanzhu', '专注', '沉浸'],
    run: (ctx) => ctx.ui.toggleFocusMode(),
  }),
  builtInCommand({
    id: 'view.toggleTypewriterMode',
    titleKey: 'command.view.toggleTypewriterMode.title',
    categoryKey: 'command.category.view',
    keywords: ['typewriter', 'daziji', '打字机'],
    run: (ctx) => ctx.ui.toggleTypewriterMode(),
  }),
  builtInCommand({
    id: 'view.toggleFullscreen',
    titleKey: 'command.view.toggleFullscreen.title',
    categoryKey: 'command.category.view',
    // F11 是全屏的通用键位；功能键在输入框里不产生文本，isAllowedInEditable 放行
    keybinding: 'F11',
    keywords: ['fullscreen', 'quanping', '全屏'],
    // 把 Promise 交出去，不 `void` 掉：失败的出口是 `runCommandSafely`
    // 的兜底层（ADR 0041），自己 void 掉就等于把这次失败变成静默
    run: (ctx) => ctx.ui.toggleFullscreen(),
  }),

  // ---- 导航 -------------------------------------------------------------
  builtInCommand({
    id: 'find.open',
    titleKey: 'command.find.open.title',
    categoryKey: 'command.category.navigate',
    keybinding: 'Mod+F',
    keywords: ['find', 'search', 'chazhao'],
    enabled: hasDocument,
    run: (ctx) => ctx.ui.openFind(),
  }),
  builtInCommand({
    id: 'search.inFolder',
    titleKey: 'command.search.inFolder.title',
    categoryKey: 'command.category.navigate',
    /**
     * 与 VSCode 同一个键位。
     *
     * `Ctrl+Shift+F` 在这里是安全的：`isAllowedInEditable` 放行带修饰键的快捷键，
     * 而 `F` 不带修饰键时在编辑器里就是普通输入。
     */
    keybinding: 'Mod+Shift+F',
    keywords: ['search in folder', 'workspace search', 'grep', 'quanwen', 'sousuo', '全文检索', '在文件夹中查找'],
    // 没有文件夹就无处可查。用**禁用**而不是隐藏：隐藏会让用户以为功能不存在
    // （而它只是需要一个前置条件，命令面板里那行提示正好告诉他缺什么）
    enabled: (ctx) => ctx.workspace.getSnapshot().tree.root !== null,
    run: (ctx) => ctx.ui.openSearchFolder(),
  }),
  builtInCommand({
    id: 'palette.open',
    titleKey: 'command.palette.open.title',
    categoryKey: 'command.category.navigate',
    keybinding: 'Mod+P',
    keywords: ['command palette', 'minglingmianban'],
    run: (ctx) => ctx.ui.openPalette(),
  }),
  builtInCommand({
    id: 'app.settings',
    titleKey: 'command.app.settings.title',
    categoryKey: 'command.category.app',
    keybinding: 'Mod+,',
    keywords: ['settings', 'preferences', 'shezhi', 'pianhao'],
    run: (ctx) => ctx.ui.openSettings(),
  }),

  // ---- 诊断 ---------------------------------------------------------------
  builtInCommand({
    id: 'diagnostics.roundTrip',
    titleKey: 'command.diagnostics.roundTrip.title',
    categoryKey: 'command.category.diagnostics',
    keywords: ['roundtrip', 'fidelity', 'diagnostics', 'wangfan', 'baozhen'],
    // 内核没实现 roundTrip（导图内核）时禁用 —— 这是「没有这个能力」，
    // 不是「这份文档用不上」，但一致的呈现仍是禁用而非隐藏
    enabled: (ctx) => {
      const doc = activeDocument(ctx);
      return doc !== null && typeof ctx.ui.editor()?.roundTrip === 'function';
    },
    run: runRoundTripCheck,
  }),

  /**
   * 打开「本次运行的问题」清单（ADR 0046）。
   *
   * 刻意**不给快捷键**：它不是一个高频动作，而「键在哪」本身就是入口的一部分
   * （和专注 / 打字机模式同一条理由）。放在「帮助」菜单里 —— 「哪里出问题了」
   * 与「这是什么程序」是同一类问题，用户找得到。
   *
   * `enabled` 永远为真，哪怕清单是空的：空清单也是一个答案
   *（「我担心的那件事没有发生」），把入口藏起来就答不了这个问。
   */
  builtInCommand({
    id: 'diagnostics.issues',
    titleKey: 'command.diagnostics.issues.title',
    categoryKey: 'command.category.diagnostics',
    keywords: ['issue', 'problems', 'degrade', 'watch', 'problem', 'wenti', 'guanxi'],
    run: (ctx) => ctx.ui.openHealth(),
  }),
  // ---- 格式 ---------------------------------------------------------------
  // 四条都只是 `formatting` 契约的薄封装：能不能用问内核（Markdown 实现了它，
  // 导图/代码内核没有），格式切换本身由内核完成（含 USER_EDIT_META，与
  // 选中浮层工具条是同一条路径 —— 菜单只是多一个入口，不多一条语义）。
  ...INLINE_FORMAT_COMMANDS,

  // ---- 段落 ---------------------------------------------------------------
  // 块级格式走 toggleBlock（把**光标所在块**换过去，与 Typora 的段落菜单同一
  // 语义）；表格 / 数学块 / 图表 / 分隔线没有「所在块」可换，走 insertBlock
  // （斜杠菜单的同一入口，删 `/query` 那步对菜单场景无害 —— 不在触发态时它
  // 就等价于普通插入）。
  ...PARAGRAPH_BLOCK_COMMANDS,
  ...PARAGRAPH_INSERT_COMMANDS,

  // ---- 主题 ---------------------------------------------------------------
  // setTheme 与 cycleTheme 都是现成的 UiActions；三条直接入口给菜单用
  // （「我要暗黑」是两步循环里的第三步，循环键留着给快切）。
  ...THEME_COMMANDS,

  // ---- 编码 ---------------------------------------------------------------
  // 与状态栏下拉同一份候选清单（core/encodings.ts）；只改元数据，保存时生效
  ...ENCODING_COMMANDS,

  builtInCommand({
    id: 'app.about',
    titleKey: 'command.app.about.title',
    categoryKey: 'command.category.app',
    keywords: ['about', 'version', 'guanyu', 'banben'],
    // 构建期注入的常量必须写成 `import.meta.env.X`（硬约束 14）——
    // dev server 不替换裸标识符，写错的表现是「点一下关于整个应用就没了」
    run: (ctx) => {
      void ctx.ui.notify(t('core.cmd.aboutBody', { version: String(import.meta.env.LEVNOTE_VERSION ?? '') }), t('command.app.about.title'));
    },
  }),

  /**
   * 示例文档（B1）：往工作区根写入 `轻笺示例.md` 并打开。
   *
   * 与 `file.new` 同一条编排（stat → 写 → open），也守同一条红线：
   * **已存在就不再写** —— 用户可能已经改过这份示例，覆盖就是毁内容
   * （硬约束 2）。想重置就删掉文件再点一次，这与示例文档里写的话一致。
   */
  builtInCommand({
    id: 'help.sampleDocument',
    titleKey: 'command.help.sampleDocument.title',
    categoryKey: 'command.category.help',
    keywords: ['sample', 'example', 'demo', 'shili', '示例', '导览'],
    enabled: (ctx) => ctx.workspace.getSnapshot().tree.root !== null,
    run: (ctx) => {
      void (async () => {
        const root = ctx.workspace.getSnapshot().tree.root;
        if (root === null) return;
        const path = joinPath(root, SAMPLE_DOCUMENT_NAME);
        if (path === null) return;
        await writeNewFileThenOpen(ctx, {
          path,
          text: buildSampleDocument(),
          title: t('core.cmd.sampleTitle'),
          // 已存在就直接打开，不打扰用户：这份示例可能被改过，
          // 而「想重置就删掉再点一次」已经写在示例文档正文里。
          existsNotice: null,
          failedNotice: (reason) => t('core.cmd.sampleCreateFailed', { reason }),
        });
      })();
    },
  }),

  builtInCommand({
    id: 'ui.escape',
    titleKey: 'command.ui.escape.title',
    categoryKey: 'command.category.navigate',
    keybinding: 'Escape',
    keywords: ['cancel', 'close'],
    run: (ctx) => {
      ctx.ui.closePalette();
      ctx.ui.closeFind();
      ctx.ui.closeSettings();
    },
  }),
];

/** 「最近打开」进菜单的条数（持久化上界更大，见 core/recent.ts）。 */
export const RECENT_MENU_SLOTS = 10;

/**
 * 最近打开的文件（A5）→ 动态命令。
 *
 * 标题是**数据**：`文件名 — 所在目录`（不同目录里的同名文件要分得开），
 * 与编码命令用 `option.label` 当标题同一条先例。`dynamic: true` 让它们
 * 进菜单、不进命令面板与快捷键表 —— 那两处是能力清单，不是数据清单。
 *
 * id 带槽位号（`file.recent.0..9`）：列表顺序变了，命令**原位覆盖**，
 * 菜单项的位置不漂；尾部空出来的槽位由调用方注销。
 */
export function buildRecentFileCommands(paths: readonly string[], open: (path: string) => void): readonly Command[] {
  return paths.slice(0, RECENT_MENU_SLOTS).map((path, index) => {
    const name = basename(path);
    const dir = dirname(path);
    return builtInCommand({
      id: `file.recent.${index}`,
      titleFrom: () => (dir === null ? name : `${name} — ${dir}`),
      categoryKey: 'command.category.file',
      keywords: [name, 'recent', 'zuijin', '最近打开'],
      dynamic: true,
      run: () => open(path),
    });
  });
}

export function registerBuiltInCommands(registry: CommandRegistry): void {
  for (const command of BUILT_IN_COMMANDS) registry.register(command);
}
