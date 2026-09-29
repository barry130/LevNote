import { describe, expect, it } from 'vitest';

import { AppError, asAppError, type AppErrorShape } from './types.js';

/**
 * `asAppError` 是**全应用错误的唯一收敛点**：UI 上显示的任何错误文案、
 * 任何 `code` 分支，都是先经过它才拿到形状的。
 *
 * 它之前一条测试都没有。这里补上的原因是它同时是「最后一层兜底」——
 * 上游抛什么都有可能，它自己绝不能崩，也不能把垃圾渲染到界面上。
 */

describe('AppError', () => {
  it('是真正的 Error，instanceof 两种都认', () => {
    const error = new AppError('NOT_FOUND', '没有这个文件');
    expect(error).toBeInstanceOf(Error);
    expect(error).toBeInstanceOf(AppError);
  });

  it('带堆栈 —— 这正是「继承 Error」而不是「抛裸对象字面量」的主要理由', () => {
    const error = new AppError('IO', '写盘失败');
    expect(typeof error.stack).toBe('string');
    expect(error.stack).not.toBe('');
    // name 决定 `String(error)` 的前缀，出错日志里靠它一眼认出是我们的错误
    expect(String(error)).toContain('AppError');
  });

  it('detail 缺省时是 null，不是 undefined', () => {
    expect(new AppError('IO', '写盘失败').detail).toBeNull();
    expect(new AppError('IO', '写盘失败', '磁盘满了').detail).toBe('磁盘满了');
  });

  it('形状与 Rust 的 AppError 对齐，所以 asAppError 原样认出来', () => {
    const shape = asAppError(new AppError('CONFLICT', '磁盘上已被改写', 'expected 1, actual 2'));
    expect(shape).toEqual({ code: 'CONFLICT', message: '磁盘上已被改写', detail: 'expected 1, actual 2' });
  });
});

describe('asAppError', () => {
  it('认来自 Tauri 的普通对象 —— 跨语言边界上就是这个形状', () => {
    const wire: AppErrorShape = { code: 'PERMISSION_DENIED', message: '只读文件', detail: null };
    expect(asAppError(wire)).toEqual({ code: 'PERMISSION_DENIED', message: '只读文件', detail: null });
  });

  it('线上对象没带 detail 时补成 null', () => {
    expect(asAppError({ code: 'NOT_FOUND', message: '没找到' })).toEqual({
      code: 'NOT_FOUND',
      message: '没找到',
      detail: null,
    });
  });

  it('detail 不是字符串时收敛成 null，而不是 String() 成 [object Object]', () => {
    // detail 的契约是 `string | null`。上游（比如某个 Rust 结构改了形状）
    // 多塞了个对象进来时，宁可显示「没有更多细节」，
    // 也不要把 '[object Object]' 当成一句人话显示给用户。
    expect(asAppError({ code: 'INTERNAL', message: '出错了', detail: { nested: true } }).detail).toBeNull();
  });

  it('普通 Error 走兜底，code 归为 INTERNAL', () => {
    expect(asAppError(new Error('炸了'))).toEqual({ code: 'INTERNAL', message: '炸了', detail: null });
  });

  it('抛字符串 / null / undefined 都不能让它自己崩', () => {
    expect(asAppError('随便抛了个字符串')).toEqual({
      code: 'INTERNAL',
      message: '随便抛了个字符串',
      detail: null,
    });
    expect(asAppError(null)).toEqual({ code: 'INTERNAL', message: 'null', detail: null });
    expect(asAppError(undefined)).toEqual({ code: 'INTERNAL', message: 'undefined', detail: null });
  });
});
