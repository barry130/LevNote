import { describe, expect, it } from 'vitest';

import { describeRoundTrip, formatRoundTripReport } from './fidelity.js';

describe('describeRoundTrip', () => {
  it('逐字节一致时报告零改写', () => {
    const text = '# 标题\n\n正文一段。\n';
    const report = describeRoundTrip(text, text);
    expect(report.same).toBe(true);
    expect(report.changed + report.added + report.removed).toBe(0);
    expect(report.totalLines).toBe(3);
    // 空文本也是「一致」，行数为 0
    expect(describeRoundTrip('', '').totalLines).toBe(0);
  });

  it('逐行比对：改了哪些行、样例给出旧 → 新', () => {
    const original = ['第一行', '旧行甲', '旧行乙', '第四行'].join('\n');
    const rewritten = ['第一行', '新行甲', '新行乙', '第四行'].join('\n');

    const report = describeRoundTrip(original, rewritten);

    expect(report.same).toBe(false);
    expect(report.changed).toBe(2);
    expect(report.added).toBe(0);
    expect(report.removed).toBe(0);
    expect(report.samples[0]).toContain('第 2 行');
    expect(report.samples[0]).toContain('旧行甲');
    expect(report.samples[0]).toContain('新行甲');
  });

  it('增删各归各类；连续的删+增优先折算成「修改」', () => {
    // 中间插一行
    const withAdd = describeRoundTrip(['甲', '乙'].join('\n'), ['甲', '新行', '乙'].join('\n'));
    expect(withAdd.added).toBe(1);
    expect(withAdd.removed).toBe(0);
    expect(withAdd.changed).toBe(0);

    // 中间删一行
    const withRemove = describeRoundTrip(['甲', '多余行', '乙'].join('\n'), ['甲', '乙'].join('\n'));
    expect(withRemove.removed).toBe(1);
    expect(withRemove.added).toBe(0);

    // 删一行加一行：作为一对「修改」报告，而不是「删了又加」
    const swap = describeRoundTrip(['甲', '旧', '乙'].join('\n'), ['甲', '新', '乙'].join('\n'));
    expect(swap.changed).toBe(1);
    expect(swap.added + swap.removed).toBe(0);
  });

  it('长行在样例里被截断，不让提示框被一行代码撑爆', () => {
    const long = 'x'.repeat(80);
    const report = describeRoundTrip(long, `${long}y`);
    expect(report.samples[0]).toContain('…');
    for (const sample of report.samples) {
      expect(sample.length).toBeLessThan(120);
    }
  });

  it('超大文档退化为近似比对：数量如实在、样例为空、approximate 标出来', () => {
    // 造一个超过 LCS_CELL_LIMIT 的输入：2001 × 2001 行
    const lines = (n: number, prefix: string) => Array.from({ length: n }, (_, index) => `${prefix}${index}`);
    const a = [...lines(2001, 'a')].join('\n');
    const b = [...lines(2001, 'b')].join('\n');

    const report = describeRoundTrip(a, b);

    expect(report.approximate).toBe(true);
    expect(report.samples).toEqual([]);
    // 全部行都不同：changed 折算出 2001
    expect(report.changed).toBe(2001);
    expect(report.added + report.removed).toBe(0);
  });
});

describe('formatRoundTripReport', () => {
  it('零改写给一句安心话', () => {
    const text = '一致\n';
    const message = formatRoundTripReport(describeRoundTrip(text, text));
    expect(message).toContain('不会改写任何内容');
    expect(message).toContain('1 行');
  });

  it('有改写时带统计与样例；近似比对要说明没逐行比', () => {
    const report = describeRoundTrip('甲\n旧\n乙', '甲\n新\n乙');
    const message = formatRoundTripReport(report);
    expect(message).toContain('会改写 1 行');
    expect(message).toContain('第 2 行');

    const approximate = formatRoundTripReport({
      same: false, changed: 5, added: 0, removed: 0, totalLines: 5000, samples: [], approximate: true,
    });
    expect(approximate).toContain('未逐行比对');
  });
});
