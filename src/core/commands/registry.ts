/**
 * 命令注册表。
 *
 * 只做三件事：登记、按 id 查、把键盘事件翻译成命令。
 * 真正的命令定义在 `definitions.ts`，与这张表分开 ——
 * 表是机制，定义是策略，两者的变更频率差一个数量级。
 */

import { isAllowedInEditable, matchesKeybinding, parseKeybinding, type Keybinding, type Platform } from './keybinding.js';
import type { Command } from './types.js';

export class CommandRegistry {
  private readonly commands = new Map<string, Command>();
  private readonly order: string[] = [];
  /** `平台:写法` → 解析结果。快捷键在每次按键时都要比对，不能每次重新解析。 */
  private readonly bindings = new Map<string, Keybinding | null>();

  /**
   * 注册一条命令。重复 id 会**覆盖**（并保留原位置）。
   *
   * 覆盖而不是抛错：模块热更新与测试里的重复注册都是正常场景。
   * 但保留原位置很重要 —— 否则覆盖一次就跳到列表末尾，命令面板的顺序
   * 会随开发过程漂移。
   */
  register(command: Command): void {
    if (!this.commands.has(command.id)) this.order.push(command.id);
    this.commands.set(command.id, command);
    // 换过定义的命令，缓存的解析结果作废
    for (const key of [...this.bindings.keys()]) {
      if (key.endsWith(`:${command.keybinding ?? ''}`)) this.bindings.delete(key);
    }
  }

  unregister(id: string): void {
    if (!this.commands.delete(id)) return;
    const at = this.order.indexOf(id);
    if (at >= 0) this.order.splice(at, 1);
  }

  /** 按注册顺序返回。顺序稳定，命令面板因此不会每次打开都换排列。 */
  list(): Command[] {
    const out: Command[] = [];
    for (const id of this.order) {
      const command = this.commands.get(id);
      if (command) out.push(command);
    }
    return out;
  }

  find(id: string): Command | undefined {
    return this.commands.get(id);
  }

  /**
   * 把快捷键写法解析成可比较的形式。
   *
   * 解析失败**只记录一次警告并当作没有绑定**，不抛错：
   * 一条命令的快捷键写错了，不该让整个应用（或整个测试套件）挂掉，
   * 只该让这条命令没有快捷键。
   */
  private bindingOf(spec: string, platform: Platform): Keybinding | null {
    const cacheKey = `${platform}:${spec}`;
    const cached = this.bindings.get(cacheKey);
    if (cached !== undefined) return cached;

    let parsed: Keybinding | null;
    try {
      parsed = parseKeybinding(spec, platform);
    } catch (thrown) {
      console.warn(`快捷键写法无效，已忽略：${spec}`, thrown);
      parsed = null;
    }
    this.bindings.set(cacheKey, parsed);
    return parsed;
  }

  /**
   * 找出与这次按键匹配的命令。
   *
   * `editable` 表示事件来自输入框 —— 此时按 `isAllowedInEditable` 收紧。
   * 返回 null 表示没有命令认领这次按键，调用方应**让事件继续传播**
   * （否则用户会在编辑器里打不出字）。
   */
  resolve(event: KeyboardEvent, platform: Platform, editable: boolean): Command | null {
    for (const id of this.order) {
      const command = this.commands.get(id);
      if (!command?.keybinding) continue;

      const binding = this.bindingOf(command.keybinding, platform);
      if (binding === null) continue;
      if (editable && !isAllowedInEditable(binding)) continue;
      if (!matchesKeybinding(binding, event)) continue;

      return command;
    }
    return null;
  }

  /** 供 UI 显示某个命令的快捷键。解析失败时返回 null。 */
  displayBinding(command: Command, platform: Platform, format: (binding: Keybinding, platform: Platform) => string): string | null {
    if (!command.keybinding) return null;
    const binding = this.bindingOf(command.keybinding, platform);
    return binding === null ? null : format(binding, platform);
  }

  clear(): void {
    this.commands.clear();
    this.order.length = 0;
    this.bindings.clear();
  }
}
