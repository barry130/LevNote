/**
 * 命令执行的兜底层（ADR 0041）。
 *
 * ## 这个文件为什么存在
 *
 * 命令有四个入口：菜单、工具栏 / 活动栏、全局快捷键、命令面板。前三个走
 * `App.tsx` 的 `runCommand` 与快捷键 `onKeyDown`，第四个走 `CommandPalette`，
 * 而四个入口原先都长这样：
 *
 * ```ts
 * void Promise.resolve(command.run(context)).catch((thrown: unknown) => {
 *   console.warn(`命令 ${id} 执行失败`, thrown);
 * });
 * ```
 *
 * `console.warn` 是**开发者看的**，不是给用户看的 —— 用户那一侧的表现就是
 * 「点了一下，什么都没发生」。这正是硬约束 2 要拦的东西：命令的 `run` 抛了
 * （`view.toggleFullscreen` 在浏览器端会被 Fullscreen API 拒绝，就是现成的一例），
 * 而这次失败**没有出口**。
 *
 * 把兜底抽成一个纯函数，是为了让它能被单测钉住：三个调用点各自只剩一行
 * `reportCommandFailure(...)`，而「失败怎么变成一句话」只在这里定义一次。
 *
 * ## 它为什么是最后一层，而不是唯一一层
 *
 * 已经有七处命令**自己**处理了预期内的失败（导出失败、新建文件重名、
 * 复制 HTML 被拒、示例文档写不进去 …），它们各自 `ctx.ui.notify` 一句说清原因。
 * 这里管的是**它们没覆盖到的那部分**：意外的 throw。
 *
 * 命令已经给过提示时，这里再说一遍确实是两句（外部无法判断「这个 throw
 * 是不是已经被报告过」），所以本层的取舍是「宁可多一句，不可没有一句」——
 * 重了至多是两条提示，漏了是用户以为应用坏了。想让某条命令不落到这里，
 * 正确做法是**在命令内部把失败接住并说明**，不是在这里加白名单。
 *
 * ## 通知本身也可能失败
 *
 * 平台对话框（桌面是原生对话框，浏览器是 `window.alert`）在极端情况下也会抛。
 * 那一次失败**留在本层**，不冒泡出去、也不再找第三个出口 —— 否则就是
 * 「为了报告失败而报告失败」的无限递归。此时 `console.warn` 是唯一剩下的痕迹，
 * 这也是它**不删**的理由：兜底层自己要有兜底，而给开发者看的日志就是那一层。
 */

import { asAppError } from '../document/types.js';
import { t } from '../i18n/index.js';
import type { Command, CommandContext } from './types.js';

/**
 * 把一个命令的失败变成用户看得到的一句话。
 *
 * 收 `Command` 而不是只收 id，是因为提示里要说出**哪个**命令 ——
 * 而 `command.title` 是惰性求值的 getter（ADR 0031），现读才是对的，
 * 这里不需要知道注册表在哪。
 *
 * `run` 是同步抛还是返回 reject 的 Promise，在这里没有区别 ——
 * 所以调用点统一写 `Promise.resolve(...).catch(...)`，不必分两种写法。
 */
export async function reportCommandFailure(command: Command, thrown: unknown, ctx: CommandContext): Promise<void> {
  const error = asAppError(thrown);
  // `asAppError` 保证 message 永远是一句人话（不会是 `[object Object]`），
  // 所以 reason 可以直接进文案；detail 只有开发者用得上，落进控制台那条。
  try {
    await ctx.ui.notify(
      t('core.cmd.commandFailedBody', { title: command.title, reason: error.message }),
      t('core.cmd.commandFailedTitle'),
    );
  } catch (notifyError) {
    // 提示框自己失败了：这里绝不能再抛（否则调用方的 catch 会再次进来，
    // 表现为「点了没反应」外加一条未处理 Promise）
    console.warn(`命令 ${command.id} 执行失败，且失败提示也没能弹出`, thrown, notifyError);
  }
}

/**
 * 执行一条命令，并把意外的失败交给 {@link reportCommandFailure}。
 *
 * 三个入口共用它，是为了让「命令怎么被执行」只有一份写法 ——
 * 上面那三个调用点原先各自抄了一遍 `Promise.resolve(...).catch(console.warn)`，
 * 抄三遍的结果是修一处漏两处（正是这次的起因）。
 */
export function runCommandSafely(command: Command, ctx: CommandContext): void {
  // 同步抛必须在这里接住：`Promise.resolve(command.run(ctx))` 这种写法里，
  // `run` 是在**参数求值时**跑的，它抛出来的东西根本到不了后面的 `.catch` ——
  // 三个调用点原先都这么写，所以「同步 throw」那一路一直是漏的
  // （快捷键那一路会直接冒到 React 的事件处理里）。
  let result: void | Promise<void>;
  try {
    result = command.run(ctx);
  } catch (thrown) {
    void reportCommandFailure(command, thrown, ctx);
    return;
  }
  void Promise.resolve(result).catch((thrown: unknown) => {
    void reportCommandFailure(command, thrown, ctx);
  });
}
