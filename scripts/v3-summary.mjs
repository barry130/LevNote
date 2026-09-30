/**
 * 把 `v3-run.mjs` 跑出来的 JSON 读数汇总成一张 Markdown 表。
 *
 * 为什么要有它：V3 的场景是「视图 × 落点 × 文档」的笛卡尔积，
 * 改一次内核就要重跑一遍。读数散在十几个 JSON 里没人愿意手工抄，
 * 而**手工抄一次就会错一次**（尤其是把 p50 抄成 p95）。
 *
 * ```bash
 * node scripts/v3-summary.mjs                 # 默认读 .v3-out/
 * node scripts/v3-summary.mjs --dir 别的目录
 * node scripts/v3-summary.mjs --notes         # 连每档的过程备注一起打
 * ```
 *
 * 表里刻意保留两列容易被忽略的东西：
 * - **刷新率**：`paint` 的物理下界就是「一帧」，60Hz 和 120Hz 的读数不可比；
 * - **样本数**：0 样本的档位会显示成 `0`，而 0 看起来像「极快」——
 *   所以样本数必须和百分位并排放，看到 `0 / 0 / 0` 要立刻警觉。
 */

import { readdirSync, readFileSync } from 'node:fs';
import { join, resolve } from 'node:path';

const argv = process.argv.slice(2);
const dirFlag = argv.indexOf('--dir');
const DIR = resolve(dirFlag === -1 ? '.v3-out' : argv[dirFlag + 1]);
const WITH_NOTES = argv.includes('--notes');

const files = readdirSync(DIR)
  .filter((name) => name.endsWith('.json'))
  .sort();

if (files.length === 0) {
  console.error(`${DIR} 里没有 JSON —— 先跑 scripts/v3-run.mjs --out …`);
  process.exit(1);
}

const rows = [];
for (const name of files) {
  const payload = JSON.parse(readFileSync(join(DIR, name), 'utf8'));
  const report = payload.报告 ?? {};
  const scene = payload.场景 ?? {};
  rows.push({ name, payload, report, scene });
}

const cell = (v) => (v === undefined || v === null ? '—' : String(v));

console.log('| 视图 | 落点 | 文档 | 刷新率 | 校准 | 样本 | **输入延迟** p50/p95/max | sync p50 | paint p50 | >50ms | 长任务 | 判定 |');
console.log('|---|---|---|---|---|---|---|---|---|---|---|---|');
for (const { report: r, scene: s, payload } of rows) {
  const hz = /(\d+)Hz/.exec(cell(r.刷新率))?.[1] ?? '?';
  const latency = r.输入延迟 ?? {};
  const sync = r.sync ?? {};
  const paint = r.paint ?? {};
  const over = cell(r['超过 50ms 的输入']).split('（')[0];
  const task = r.长任务 ?? {};
  const base = payload.校准基准ms ?? {};
  const verdict = (r.结论 ?? '').startsWith('✅') ? '✅ 达标' : (r.结论 ?? '').startsWith('❌') ? '❌ 超出' : '⚠️ 无样本';
  console.log(
    `| ${cell(s.视图)} | ${cell(s.落点)} | ${cell(s.文档)} | ${hz}Hz | ${cell(base.敲字前)}ms | ${cell(r.样本数)} | ` +
      `**${cell(latency.p50)} / ${cell(latency.p95)} / ${cell(latency.max)}** | ` +
      `${cell(sync.p50)} | ${cell(paint.p50)} | ` +
      `${over} | ${cell(task.条数)} 条（最长 ${cell(task.最长)}ms） | ${verdict} |`,
  );
}

if (WITH_NOTES) {
  console.log('');
  for (const { name, payload } of rows) {
    console.log(`### ${name}`);
    for (const note of payload.备注 ?? []) console.log(`- ${note}`);
    console.log('');
  }
}
