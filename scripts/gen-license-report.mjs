// 生成 docs/dependency-licenses.md —— 发版杂项 P7-34 的清单与再生成工具。
//
// 为什么是脚本而不是手工清单：手工的那份一定会漂（升一次依赖漂一次），
// 漂了之后表现是「清单说没有 GPL，实际有」—— 比没有清单更糟。
// 唯一来源分别是 node_modules（实际装了什么）与 Cargo.lock + 本机 registry
// 源码目录（实际编了什么），脚本只做转述。
//
// 运行：node scripts/gen-license-report.mjs（纯 Node 内置模块，无依赖，离线可跑）
//   - npm 侧：从 package.json 的 dependencies 出发走实装的 node_modules，
//     只列**生产依赖**（devDependencies 不进打包产物，与用户无关）；
//   - Rust 侧：读 src-tauri/Cargo.lock 的 registry 包，许可证从
//     ~/.cargo/registry/src/*/ 的 crate 清单里取 —— 不联网。

import { readFileSync, writeFileSync, readdirSync, existsSync, realpathSync } from 'node:fs';
import { join, dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { homedir } from 'node:os';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');

// ---------------------------------------------------------------- npm 侧

function readPackageJson(dir) {
  try {
    return JSON.parse(readFileSync(join(dir, 'package.json'), 'utf8'));
  } catch {
    return null;
  }
}

/**
 * 从 fromDir 向上解析 node_modules —— 与 Node 自己的解析规则同构。
 *
 * pnpm 的 node_modules 里全是**符号链接**（真身在 `node_modules/.pnpm/…`）。
 * 解析命中后必须折算成 realpath 再继续走：Node 解析模块时默认就解析符号链接
 * （preserve-symlinks 关闭），pnpm 的目录布局正是围绕这个行为搭的 ——
 * 某个包的依赖就摆在它真身旁边（`.pnpm/foo@1/node_modules/<dep>`）。
 * 拿着链接路径往上走会一路走到根 node_modules，那里只有直接依赖，
 * 于是传递依赖全部「解析不到」—— npm 的扁平布局下没有链接，所以从来没暴露过。
 */
function resolvePackage(name, fromDir) {
  let dir = fromDir;
  while (true) {
    const candidate = join(dir, 'node_modules', name);
    if (existsSync(join(candidate, 'package.json'))) return realpathSync(candidate);
    const parent = dirname(dir);
    if (parent === dir) return null;
    dir = parent;
  }
}

function licenseOf(pkg) {
  const raw = pkg.license ?? pkg.licenses ?? '未知';
  if (typeof raw === 'string') return raw;
  if (Array.isArray(raw)) return raw.map((item) => item.type ?? '?').join(', ');
  return raw.type ?? '未知';
}

function collectNpm() {
  const manifest = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'));
  const seen = new Map(); // name@version -> { name, version, license, homepage }
  const queue = Object.keys(manifest.dependencies ?? {}).map((name) => ({
    name,
    from: root,
  }));
  while (queue.length > 0) {
    const { name, from } = queue.shift();
    const dir = resolvePackage(name, from);
    if (dir === null) {
      console.warn(`[npm] 解析不到 ${name}（从 ${from}）—— 依赖装全了吗？`);
      continue;
    }
    const pkg = readPackageJson(dir);
    if (pkg === null) continue;
    const key = `${pkg.name}@${pkg.version}`;
    if (seen.has(key)) continue;
    seen.set(key, {
      name: pkg.name,
      version: pkg.version,
      license: licenseOf(pkg),
      homepage: pkg.homepage ?? pkg.repository?.url ?? '',
    });
    for (const dep of Object.keys(pkg.dependencies ?? {})) {
      queue.push({ name: dep, from: dir });
    }
  }
  return [...seen.values()].sort((a, b) => a.name.localeCompare(b.name));
}

// ---------------------------------------------------------------- Rust 侧

function collectCargo() {
  const lockText = readFileSync(join(root, 'src-tauri', 'Cargo.lock'), 'utf8');
  // Cargo.lock 是宽松 TOML，这里只解析 [[package]] 块的四个字段，不引 TOML 库
  const packages = [];
  const blocks = lockText.split('[[package]]').slice(1);
  for (const block of blocks) {
    const name = block.match(/^name = "(.+)"$/m)?.[1];
    const version = block.match(/^version = "(.+)"$/m)?.[1];
    const source = block.match(/^source = "(.+)"$/m)?.[1] ?? '';
    if (name && version) packages.push({ name, version, source });
  }

  const registryRoots = [];
  const cargoSrc = join(homedir(), '.cargo', 'registry', 'src');
  if (existsSync(cargoSrc)) {
    for (const entry of readdirSync(cargoSrc)) {
      if (entry.startsWith('index.crates.io-')) registryRoots.push(join(cargoSrc, entry));
    }
  }

  const out = [];
  for (const pkg of packages) {
    if (pkg.name === 'qingmo') continue;
    // git 依赖与路径依赖不在 registry 里；本项目没有，遇到了如实标注
    if (!pkg.source.startsWith('registry+')) {
      out.push({ name: pkg.name, version: pkg.version, license: '非 registry 来源，请人工核对' });
      continue;
    }
    let license = null;
    for (const rootDir of registryRoots) {
      const crateDir = join(rootDir, `${pkg.name}-${pkg.version}`);
      const manifestPath = join(crateDir, 'Cargo.toml');
      if (!existsSync(manifestPath)) continue;
      const text = readFileSync(manifestPath, 'utf8');
      const m = text.match(/^\s*license\s*=\s*"([^"]+)"/m);
      license = m?.[1] ?? (text.includes('license-file') ? '见 license-file（未声明 license 字段）' : null);
      break;
    }
    out.push({ name: pkg.name, version: pkg.version, license: license ?? '本地 registry 缺源码，跑一次 cargo fetch 后重试' });
  }
  out.sort((a, b) => a.name.localeCompare(b.name));
  return out;
}

// ---------------------------------------------------------------- 审阅与输出

/**
 * 严格的/copyleft 的许可证片段。
 *
 * 判定逻辑：**所有**顶层 OR 选项都含严格条款才列出来 —— `Unlicense OR MIT`
 * 这种双许可里用户（我们）可以选宽松的那一支，不构成风险；而
 * `ISC AND OpenSSL` 这种 AND 组合里严格条款躲不掉，必须列。
 * 列出来只代表「需要人看一眼」，不做法律判断。
 */
function needsReview(license) {
  const STRICT = /(GPL|AGPL|LGPL|SSPL|EUPL|OSL|CDDL|OpenSSL)/i;
  const PERMISSIVE_SINGLE = /^(MIT|Apache-[\d.]+|BSD-\d-Clause|BSD\d?|ISC|Zlib|CC0-[\d.]+|CC-BY-[\d.-]+|Unlicense|BSL-[\d.]+|MulanPSL-[\d.]+|WTFPL|MS-PL|CDLA-Permissive-[\d.]+|Unicode-[\w.-]+|BlueOak-[\d.]+)$/i;
  const options = license.split(/\s+OR\s+|\s*\/\s*/i);
  if (options.length === 0) return true;
  return options.every((option) => {
    if (PERMISSIVE_SINGLE.test(option.trim())) return false;
    return STRICT.test(option);
  });
}

function groupByLicense(items) {
  const groups = new Map();
  for (const item of items) {
    const list = groups.get(item.license) ?? [];
    list.push(item);
    groups.set(item.license, list);
  }
  return [...groups.entries()].sort((a, b) => b[1].length - a[1].length || a[0].localeCompare(b[0]));
}

function mdEscapeCell(text) {
  return text.replaceAll('|', '\\|');
}

const npm = collectNpm();
const cargo = collectCargo();

const lines = [];
lines.push('# 依赖许可证清单', '', '> 本文件由 `node scripts/gen-license-report.mjs` 生成（P7-34），**不要手改** ——', '> 升级依赖后重跑一次。npm 侧只列生产依赖；Rust 侧来自 `Cargo.lock` + 本机 registry。', '');
lines.push('## npm（打进前端产物）', '', `共 ${npm.length} 个包（含传递依赖）。`, '', '| 包 | 版本 | 许可证 |', '|---|---|---|');
for (const item of npm) {
  lines.push(`| ${mdEscapeCell(item.name)} | ${mdEscapeCell(item.version)} | ${mdEscapeCell(item.license)} |`);
}
lines.push('', '## Rust crates（编译进二进制）', '', `共 ${cargo.length} 个 crate。按许可证分组：`, '');
for (const [license, crates] of groupByLicense(cargo)) {
  const names = crates.map((c) => `${c.name} ${c.version}`).join(', ');
  lines.push(`### ${license}（${crates.length}）`, '', names, '');
}

const watch = [
  ...npm.map((item) => ({ side: 'npm', ...item })),
  ...cargo.map((item) => ({ side: 'cargo', ...item })),
].filter((item) => needsReview(item.license));
lines.push('## 需要人工复核的条目', '');
if (watch.length === 0) {
  lines.push('无 —— 所有依赖都存在可选择的宽松许可（或本身即宽松/公有领域）。');
} else {
  lines.push(
    '下列条目的**每一个**许可选项都含严格条款（双许可里可选宽松支的不列）。',
    '本项目规则：不引入 GPL 系依赖，见 AGENTS.md「明确不要做的事」。逐条核对再决定：',
    '',
  );
  for (const item of watch) {
    lines.push(`- [${item.side}] ${item.name} ${item.version} — ${item.license}`);
  }
}
lines.push('');

const outPath = join(root, 'docs', 'dependency-licenses.md');
writeFileSync(outPath, lines.join('\n'), 'utf8');
console.log(`已生成 ${outPath}`);
console.log(`npm 生产依赖 ${npm.length} 个；Rust crates ${cargo.length} 个；待复核 ${watch.length} 条。`);
