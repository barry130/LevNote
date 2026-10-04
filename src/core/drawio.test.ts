import { describe, expect, it } from 'vitest';

import { missingStencilLibs, stencilLibsOf, VENDORED_STENCIL_LIBS } from './drawio.js';

/** 一段只含关键结构的样本：stencil 引用全在 style 属性里。 */
function cell(style: string): string {
  return `<mxCell style="${style}" vertex="1"><mxGeometry/></mxCell>`;
}

describe('stencilLibsOf', () => {
  it('从 style 里提取 mxgraph.<库名>. 的库名', () => {
    const xml = cell('shape=mxgraph.bpmn.task;whiteSpace=wrap;');
    expect(stencilLibsOf(xml)).toEqual(['bpmn']);
  });

  it('同一库引用多次只算一次，按出现顺序去重', () => {
    const xml = [cell('shape=mxgraph.bpmn.task;'), cell('shape=mxgraph.flowchart.decimal;'), cell('shape=mxgraph.bpmn.event;')].join('\n');
    expect(stencilLibsOf(xml)).toEqual(['bpmn', 'flowchart']);
  });

  it('`mxgraph.` 前缀之外的 shape 引用不收 —— 那是内置形状，不需要 stencil 文件', () => {
    const xml = cell('shape=cylinder3;') + cell('rhombus;');
    expect(stencilLibsOf(xml)).toEqual([]);
  });

  it('库名允许数字、下划线、连字符', () => {
    const xml = cell('shape=mxgraph.azure-2.function_app;') + cell('shape=mxgraph.veeam.2d_3d;');
    expect(stencilLibsOf(xml)).toEqual(['azure-2', 'veeam']);
  });

  it('没有图（空串 / 纯矩形）就是空列表', () => {
    expect(stencilLibsOf('')).toEqual([]);
    expect(stencilLibsOf(cell('rounded=0;'))).toEqual([]);
  });

  it('大小写不敏感地匹配（XML 生成器有写大写的），返回的库名保留原文', () => {
    const xml = cell('shape=mxgraph.BPMN.task;');
    expect(stencilLibsOf(xml)).toEqual(['BPMN']);
  });
});

describe('missingStencilLibs', () => {
  it('全在内置清单里 → 空数组（渲染不会缺形状）', () => {
    const xml = cell('shape=mxgraph.bpmn.task;') + cell('shape=mxgraph.flowchart.database;');
    expect(missingStencilLibs(xml)).toEqual([]);
  });

  it('引用了没内置的库 → 列出库名（这份差异就是图区警示条的内容）', () => {
    const xml = cell('shape=mxgraph.bpmn.task;') + cell('shape=mxgraph.aws4.lambda_function;');
    expect(missingStencilLibs(xml)).toEqual(['aws4']);
  });

  it('内置清单只认库层面，不认形状层面：库里随便哪个形状都算「有」', () => {
    // `mxgraph.bpmn.` 后面接一个清单里不存在的形状名，也按「库内置了」算
    const xml = cell('shape=mxgraph.bpmn.totally_missing_shape;');
    expect(missingStencilLibs(xml)).toEqual([]);
  });

  it('内置清单本身是非空的', () => {
    // 守住「清单里列的名字与 public/drawio/stencils/ 的文件名一致」这一环：
    // 若有人改了目录名而忘了改这里，这条会提醒去对照
    expect(VENDORED_STENCIL_LIBS).toContain('bpmn');
    expect(VENDORED_STENCIL_LIBS).toContain('flowchart');
  });
});
