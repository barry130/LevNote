#!/usr/bin/env python3
"""Create-only Task11 report and small fixed opaque evidence archive; no Go."""
from pathlib import Path
import datetime
import difflib
import hashlib
import io
import json
import os
import tarfile
import sys

LANE = Path('/home/user/levnote-dev-reliability-20261112-01')
OUT = LANE / 'delivery/document-fingerprint-review-02'
SOURCE = LANE / 'inputs/document-final-core-tests-01/native'
REPORT = LANE / 'work/document-fingerprint-01/native/handoffs/autonomous-document-fingerprint-20261112.md'
BASELINE = Path('/home/user/LevNote/native')
HEAD = '8897244ffa4f00f5ec51f95545e7a3ca27d37d97'
PINS = {
 'core/document/document.go': (14511, '44da6396913741a18ed5ebd1d0c668853d28ad399eba9b13abb02aa08f76a3fe'),
 'core/document/atomic.go': (4873, '641601d61fe3da0ec83954c6b199cac46864099ffa78f09e2a309180491fec83'),
 'core/document/utf8_save_test.go': (8238, '7c02ad0c795cadda2b93c443d6816d742ff13bf0e9ae26facb3e8149f31b3ec0'),
 'core/document/fingerprint_boundary_test.go': (3399, 'f0d3204bf2b7430eb1a95ca183ade789816faf7752433cad06c2e6a74f434aef'),
 'core/document/fingerprint_public_test.go': (2957, '3c94db4b1ac0604f03ebb3cc26fdf62e661dadfd266ad74b06d6d9fe587955d5'),
 'core/document/fingerprint_conflict_public_test.go': (6993, 'c0556277c6d7ae7c7cf4a72d75d3901a23aedcedec94aa00711dc5ac0ab20b69'),
 'core/document/fingerprint_encoding_public_test.go': (4870, '40212ed1a6c5319b16e2e107ae1aaa8530b3d381fe3dac85d64266436f974edc'),
 'core/document/fingerprint_directory_sync_public_linux_test.go': (7142, '8f68c732927b6375edeb8b589facef356a360f944512b4564968afaa6fe51951'),
}
STAGES = (
 'document-fingerprint-red-01', 'document-fingerprint-format-01',
 'document-fingerprint-green-01', 'document-fingerprint-format-02',
 'document-fingerprint-green-02', 'document-public-mtime-format-01',
 'document-public-mtime-positive-01', 'document-public-conflict-format-01',
 'document-public-conflict-positive-01', 'document-public-encoding-format-01',
 'document-public-encoding-positive-01', 'document-public-directory-sync-format-01',
 'document-public-directory-sync-positive-01', 'document-final-core-tests-01',
 'document-final-core-vet-01',
)
CLOSED = (
 'document-fingerprint-red-01', 'document-fingerprint-green-02',
 'document-public-mtime-positive-01', 'document-public-conflict-positive-01',
 'document-public-encoding-positive-01', 'document-public-directory-sync-positive-01',
 'document-final-core-tests-01', 'document-final-core-vet-01',
)
EVIDENCE_NAMES = (
 'invocation.json', '01-clone-head.stdout.bin', '01-clone-head.json',
 '03-go-version.stdout.bin', '03-go-version.json', '04-go-env.stdout.bin',
 '04-go-env.json', '05-product-command.command.json',
 '05-product-command.started.json', '05-product-command.json',
 '05-product-command.stdout.bin', '05-product-command.stderr.bin',
 'result.json', 'source-before.json', 'source-after.json', 'snapshot-after.json',
 '06-clone-head-after.stdout.bin', '06-clone-head-after.json',
)
os.umask(0o077)
sys.dont_write_bytecode = True

def utc():
 return datetime.datetime.now(datetime.timezone.utc).isoformat()

def identity(data):
 return {'bytes': len(data), 'sha256': hashlib.sha256(data).hexdigest()}

def read_json(path):
 return json.loads(path.read_bytes())

def json_bytes(value):
 return (json.dumps(value, ensure_ascii=False, indent=2) + '\n').encode()

def put(path, data):
 path.parent.mkdir(parents=True, exist_ok=True)
 with path.open('xb') as stream:
  stream.write(data)
 assert path.read_bytes() == data

started = utc()
assert OUT.resolve() == OUT and REPORT.resolve() == REPORT and not REPORT.exists()
assert set(p.name for p in OUT.iterdir()) == {'prepare-report-and-fixed-archive-02.py', 'derive-receipt.json'}
base_manifest = read_json(LANE / 'inputs/reopen-red-01/expected-manifest.json')
base_files = dict(base_manifest['files'])
del base_files['app/autonomous_reliability_test.go']
final_manifest = read_json(SOURCE.parent / 'expected-manifest.json')
assert base_manifest['head'] == final_manifest['head'] == HEAD
assert set(base_files).issubset(final_manifest['files'])
assert {p for p in final_manifest['files'] if final_manifest['files'][p] != base_files.get(p)} == set(PINS)
assert len(final_manifest['files']) == 1196
assert sum(v['bytes'] for v in final_manifest['files'].values()) == 16996849
files = {}
origins = {}
references = {}
REFERENCE_ONLY = {'expected-manifest.json', 'source-before.json', 'source-after.json', 'snapshot-after.json'}

def add(name, path):
 assert name not in references and path.is_file() and not path.is_symlink()
 raw = path.read_bytes()
 copied = path.name not in REFERENCE_ONLY
 references[name] = {'path': str(path), **identity(raw), 'body_in_primary': copied}
 if copied:
  assert name not in files
  files[name] = raw
  origins[name] = str(path)

for label in STAGES:
 evidence = LANE / 'runs' / label / 'evidence'
 for path in sorted(evidence.iterdir()):
  if path.is_file() and not path.is_symlink() and path.name not in EVIDENCE_NAMES:
   references['runs/' + label + '/evidence/' + path.name] = {'path': str(path), **identity(path.read_bytes()), 'body_in_primary': False}

for name, (size, digest) in PINS.items():
 raw = (SOURCE / name).read_bytes()
 assert identity(raw) == final_manifest['files'][name] == {'bytes': size, 'sha256': digest}
 assert (LANE / 'inputs/document-final-core-vet-01/native' / name).read_bytes() == raw
 add('source/' + name, SOURCE / name)
for name in ('core/document/document.go', 'core/document/atomic.go', 'core/document/utf8_save_test.go'):
 before = (BASELINE / name).read_bytes()
 after = (SOURCE / name).read_bytes()
 assert identity(before) == base_files[name]
 add('baseline/' + name, BASELINE / name)
 diff = ''.join(difflib.unified_diff(before.decode().splitlines(True), after.decode().splitlines(True), fromfile='baseline/' + name, tofile='final/' + name)).encode()
 files['diff/' + name + '.diff'] = diff
 origins['diff/' + name + '.diff'] = 'metadata unified diff of the two pinned actual files'
wrong_before = files['baseline/core/document/utf8_save_test.go']
wrong_after = files['source/core/document/utf8_save_test.go']
assert wrong_before.replace(b'doc.didSave(wrong)', b'doc.didSave(wrong, int64(len("baseline")), beforeDisk.info)') == wrong_after

rows = []
checks = {}
for label in STAGES:
 evidence = LANE / 'runs' / label / 'evidence'
 manifest_path = LANE / 'inputs' / label / 'expected-manifest.json'
 if manifest_path.is_file():
  add('inputs/' + label + '/expected-manifest.json', manifest_path)
 for name in EVIDENCE_NAMES:
  path = evidence / name
  if path.is_file():
   add('runs/' + label + '/evidence/' + name, path)
 if label == 'document-fingerprint-green-01':
  assert not (evidence / '05-product-command.json').exists() and not (evidence / 'result.json').exists()
  assert not (evidence / 'source-after.json').exists() and not (evidence / 'snapshot-after.json').exists()
  add('classification/document-fingerprint-green-01-interruption-01/result.json', LANE / 'evidence/document-fingerprint-green-01-interruption-01/result.json')
  checks[label] = {'classification': 'INTERRUPTED_WRAPPER_FINAL_MISSING_NOT_PRODUCT_PASS', 'numeric_exit': None, 'wrapper_exit': None}
  continue
 command = read_json(evidence / '05-product-command.json')
 final = read_json(evidence / 'result.json')
 intent = read_json(evidence / '05-product-command.command.json')
 pid = read_json(evidence / '05-product-command.started.json')
 expected = read_json(manifest_path)
 assert command == final['product_command']
 assert command['argv'] == intent['argv'] and command['cwd'] == intent['cwd']
 assert pid['pid'] == command['pid'] == command['owned_pgid']
 assert not command['timed_out'] and not command['owned_cleanup'] and not command['owned_group_remaining_after_wait']
 assert command.get('owned_cleanup_error') is None and command.get('started_receipt_error') is None
 for stream in ('stdout', 'stderr'):
  assert identity(Path(command[stream]['path']).read_bytes()) == {k: command[stream][k] for k in ('bytes', 'sha256')}
 before = read_json(evidence / 'source-before.json')
 after = read_json(evidence / 'source-after.json')
 snapshot = read_json(evidence / 'snapshot-after.json')
 assert before['files'] == expected['files'] and before['roots'] == expected['roots']
 if final['mode'] == 'format':
  assert after == before and snapshot['roots'] == before['roots'] and set(snapshot['files']) == set(before['files'])
  changed = [p for p in before['files'] if snapshot['files'][p] != before['files'][p]]
  assert set(changed).issubset(command['argv'][2:])
 else:
  assert before == after == snapshot
  changed = []
 assert command['exit_code'] == (1 if label == 'document-fingerprint-red-01' else 0)
 assert final['exit_code'] == (2 if label == 'document-fingerprint-red-01' else 0)
 checks[label] = {'pid': command['pid'], 'pgid': command['owned_pgid'], 'numeric_exit': command['exit_code'], 'wrapper_exit': final['exit_code'], 'started_at': command['started_at'], 'launched_at': command['launched_at'], 'ended_at': command['ended_at'], 'elapsed_s': command['elapsed_s'], 'argv': command['argv'], 'cwd': command['cwd'], 'stdout': command['stdout'], 'stderr': command['stderr'], 'expected': identity(manifest_path.read_bytes()), 'three_maps_before': identity((evidence / 'source-before.json').read_bytes()), 'input_count': len(before['files']), 'input_bytes': sum(v['bytes'] for v in before['files'].values()), 'numeric_receipt': identity((evidence / '05-product-command.json').read_bytes()), 'final_receipt': identity((evidence / 'result.json').read_bytes()), 'formatter_changed_files': changed}
 if label in CLOSED:
  c = checks[label]
  rows.append(f"| {label} | {c['pid']}/{c['pgid']} | {c['numeric_exit']}/{c['wrapper_exit']} | {c['input_count']}/{c['input_bytes']} | {c['elapsed_s']} |")

# Only the exact freeze/derive result records, not work trees or caches.
for label in STAGES:
 for prefix in ('freeze-', 'derive-'):
  root = LANE / 'evidence' / (prefix + label)
  if root.is_dir():
   for path in sorted(root.iterdir()):
    if path.is_file() and not path.is_symlink():
     name = 'provenance/' + prefix + label + '/' + path.name
     if path.name == 'result.json':
      add(name, path)
     else:
      references[name] = {'path': str(path), **identity(path.read_bytes()), 'body_in_primary': False}
for name in ('document.go.diff', 'atomic.go.diff', 'utf8_save_test.go.diff'):
 add('historical-semantic-diff-02/' + name, LANE / 'evidence/document-fingerprint-semantics-02' / name)
for name in ('dev-remote-validation-20261112-03.py',):
 add('helper/' + name, LANE / 'tools' / name)
for name in ('freeze-document-input-01.py', 'capture-document-formatted-input-01.py', 'capture-document-public-formatted-input-01.py', 'capture-document-conflict-formatted-input-01.py', 'capture-document-encoding-formatted-input-01.py', 'capture-document-final-formatted-input-01.py'):
 add('helper/' + name, LANE / name)
add('helper/prepare-report-and-fixed-archive-02.py', Path(__file__))
add('preparation/document-fingerprint-review-01/preparation-failed-result.json', LANE / 'delivery/document-fingerprint-review-01/preparation-failed-result.json')
add('preparation/document-fingerprint-review-02/derive-receipt.json', OUT / 'derive-receipt.json')
old_preparation = LANE / 'delivery/document-fingerprint-review-01/prepare-report-and-fixed-archive-01.py'
references['preparation/document-fingerprint-review-01/prepare-report-and-fixed-archive-01.py'] = {'path': str(old_preparation), **identity(old_preparation.read_bytes()), 'body_in_primary': False}

package_lines = files['runs/document-final-core-tests-01/evidence/05-product-command.stdout.bin'].decode().splitlines()
top_runs = sum(s.startswith('=== RUN   ') and '/' not in s[len('=== RUN   '):] for s in package_lines)
top_pass = sum(s.startswith('--- PASS: ') for s in package_lines)
all_runs = sum(s.lstrip().startswith('=== RUN   ') for s in package_lines)
all_pass = sum(s.lstrip().startswith('--- PASS: ') for s in package_lines)
assert (top_runs, top_pass, all_runs - top_runs, all_pass - top_pass) == (57, 57, 31, 31)
assert not any(s.lstrip().startswith(('--- FAIL:', '--- SKIP:')) for s in package_lines)
assert package_lines[-2] == 'PASS' and package_lines[-1].startswith('ok')
assert files['runs/document-final-core-vet-01/evidence/05-product-command.stdout.bin'] == files['runs/document-final-core-vet-01/evidence/05-product-command.stderr.bin'] == b''

intro = '''# 原生文档指纹精确编码字节边界 · Task11

日期：2026-11-13。任务：task-11；实现及全部新增 Go/gofmt 执行者：native-reliability（当前 a753a04c-1383-4870-877c-e70639eb0422）。

## 1. 结论与验收边界

type:progress

最终八份源码的实际编码字节身份已冻结。原始内部增长 oracle 的真实 RED → 最小边界修复 → 原 oracle GREEN 完成；另有四项正常公开 API 的兼容/安全 **positive**。最后完整 `go test -v -count=1 -timeout=240s ./core/document` 达到 57 个主测试与 31 个子测试全部 PASS、0 FAIL、0 SKIP；`go vet ./core/document` 实际退出 0、stdout/stderr 均空。

Lead 于 2026-11-13 已独立读取最终测试 raw178/numeric62/final138、vet numeric59/final132，实际核验全部 1196 input/run 文件、三份清单和先前 directory-sync positive 的同源身份；这两项是 **core/document 有限源码与 Linux 包门禁**，不是整应用或物理 Windows 验收。最终报告/原字节镜像仍由 Lead 审核；本报告不能替代 task 状态或发布决定。

以下不作通过宣称：
- 公开 API 的 deterministic didSave 验证 I/O 失败，以及 Save nil-disk guard 的 preceding public RED；当前无合法确定性业务 seam。
- 文件系统 CAS、任意并发外部写入完全排除、Windows ReplaceFileW/MoveFileEx/ACL/IME/WM_CLOSE/Job/PE/物理显示后端。
- 整应用、旧 app panic 后未达到的 selectors，旧 Linux case-sensitive reopen 失败，以及各独立任务尚未完成的能力。
- 旧 GREEN01 的 wrapper 中断不能凭 raw PASS 补作成功；原记录与缺失状态独立保留。

当前已接受的可运行物理 Windows 产物仍是 [.29](<F:/AIproject/LevNote/bin/levnote-native-worker-20261110.29/levnote-native.exe>)；本工作未建立 .30 或替代其验收。

## 2. 固定范围、输入与执行环境

type:state

- 克隆/输入 HEAD：`8897244ffa4f00f5ec51f95545e7a3ca27d37d97`；产品对照基线：`77454291ce1aad7c5a617ab19d507e192154baf6`。CI 的后续 source-only 暂停提交未被推定已更新 box 普通克隆。
- 授权 box：`cb-698f697fed7c`；所有语义源码、测试、gofmt、Go 门禁均在 `/home/user/levnote-dev-reliability-20261112-01` 隔离 lane。没有本机 Go、复跑旧标签、替代编译器、下载、网络依赖、遥测或配置变更。
- 八份最终源码来自 [最终 immutable input](</home/user/levnote-dev-reliability-20261112-01/inputs/document-final-core-tests-01/native>)，不是未格式化 mutable work；vet immutable input 的对应八文件完全相同。
- 最终 1196 files / 16996849 B；变化集相对克隆基线恰好是下表八文件（3 份存量调整，5 份新增测试），其余基线文件原字节保留。
- Go1.27.3；`GOTOOLCHAIN=local CGO_ENABLED=0 GOWORK=off GOVCS=*:off GOFLAGS=-mod=vendor GOPROXY=off GOSUMDB=off GOENV=off GOTELEMETRY=off GOOS=linux GOARCH=amd64`。HOME/XDG/tmp/cache 为 owned lane；telemetry 在 Go 启动前关闭。有效 Go 环境的 GOENV 文件名为空。
- freeze、format、capture、test/vet 分开调用；每个 Go 命令均有原始 `.command.json` prelaunch intent、`.started.json` PID/PGID、数字退出记录、final、实际 streams SHA/bytes、timeout/cleanup/remaining 状态及三份输入清单。文件名不是不存在的 `.intent.json`。
- 旧 runner 中 `owner=native-reliability/task-5` 是原 provenance 原文，未为 Task11 重标。

## 3. 原始问题与实现

type:goal

### 3.1 指纹只读取实际 expected encoded length + 1

旧 fingerprint 通过 `io.Copy(h, f)` 可将 before-Stat 后增长文件读到 EOF，工作量没有精确编码字节边界。内部真实文件 oracle 使用独立 12-byte UTF-8 BOM + `甲🙂\\r\\n` 字面量；实际 append 在 production before-Stat 后、first Read 前发生。其 non-embedded `*os.File` 包装避免继承 `WriteTo` 绕过计数。真实 RED28258 delivered79、allowed13，在 byte-work assertion 失败；它不是编译失败或 public-business RED。

最终内部 `fingerprintOpened(path, Reader+Stat, expectedLength, expectedInfo)` 保留 production 实际 `os.File`、真实当前路径 Stat，不暴露业务 setter/getter。采用 `io.Copy(h, io.LimitReader(f, expectedLength+1))`；拒绝负长度/MaxInt64、short/extra byte、增长、Stat/I/O/identity drift，失败 identity 为零。GREEN02 原 3399-byte oracle 不改，确实 delivered13、ErrConflict、zero identity。

### 3.2 SHA256 + 实际编码 Size + SameFile；不比基线旧 mtime

Save/Check 使用合法 disk baseline 的 actual encoded Size 与 `os.SameFile`，而不是 decoded UTF-8 bytes、UTF-16 units 或 editable cap。真实 size/SameFile mismatch 可在 hash 前拒绝；before/after/current-path 均要求 regular、size、in-read mtime 与 identity 一致，实际 delivered count 恰好 expected length 后才返回 hash identity。

mtime-only 同 inode/同 bytes 不因旧 baseline timestamp 漂移产生冲突。`CheckExternalChange` 把 missing/ErrConflict（含 size/SameFile mismatch）映射 changed，其他 I/O 可见；nil disk baseline 返回 changed。Save check 的必要 nil-disk guard 返回具路径的 ErrConflict，避免 nil dereference；Lead 明确保留，但 **没有 preceding public-RED 证明**，不把此 guard 伪装成公开业务 TDD 修复。

### 3.3 发布验证来自实际 encoded writer/新文件身份

UTF-8 的 `encodedCountingWriter` 在真实 accepted-byte 输出端计数，包含 BOM；保留 `streamUTF8` 原 32 KiB bufio/MultiWriter、Snapshot immutable piece accepted-byte/short-write 语义，不增加公开 Snapshot-length API。其他编码用既有 encode 返回的实际字节 slice：`len(bytes)` 与 SHA256。

私有 `atomicOutcome.info` 是 intended temp 的真实 FileInfo。新 `temp.Stat()` 位于 `temp.Sync()` 成功后、`temp.Close()` 前，属于 **新增可失败的 prepublication I/O 边界**，并非原子层零语义变化：shared atomic wrapper 与 BinaryCopy 调用链也会到达这里。Stat 错误关闭/清理 temp 并在 publish 前返回，不凭 post-publication path Stat 接受另一个写者的 identity。Unix rename/hardlink/directory-Sync 与 Windows 发布 primitive、已存在 wrappers 的签名保持；`Committed` 仍以真实发布点为界。

`didSave` 验证的路径仍为 `d.Path`，参数只是 digest、encodedLength、expectedInfo，不偷偷增加 path adoption。fingerprint 或 digest 验证失败不 advance saved/disk/hash/baseline/history/Dirty；成功才建立实际验证后的新基线。验证不能把已发布文件回滚成“没发生”。

### 3.4 原 SaveAs metadata adoption 顺序明确保留

两种编码分支均为：atomic 成功返回 → `d.Path = path; d.IsNew = false; d.ReadOnly = false` → `didSave(...)`。不是“didSave 成功后才切 Path”。新 non-UTF8 分支仅 flatten 原 else，不能因此改变顺序。

Directory-Sync 失败则 atomic 带 Committed 事实返回 error，SaveAs 在 metadata assignment 和 didSave **之前**返回。因此原 metadata/基线保留、target 却可能已存在；公开 directory-sync oracle 证明这一边界，不声称 didSave 已执行或验证 I/O 已失败。

### 3.5 旧 wrong-digest 全部断言保留及格式身份

既有 `TestUTF8StreamingSaveRejectsWrongDigestWithoutStateChange` 唯一变更是必要内部签名适配：`doc.didSave(wrong, int64(len("baseline")), beforeDisk.info)`；旧 Error/metadata/baseline/Dirty/saved/disk/hash/SameFile/DiskToken/Buffer 断言原字节保留，元数据差异生成器逐字比较确认。

Mutable work 中 atomic.go 为 4867 B / `680e7f00aca4ba5a5079ee8a78644a69bdb2a8e75daad63cd1157d2209ba9c01`；matching snapshot 的 gofmt 仅 6 个 whitespace bytes 形成已接受 4873 B / `641601d61fe3da0ec83954c6b199cac46864099ffa78f09e2a309180491fec83`。不得称 mutable work 自始至终已经等于 formatted final。最终 test/vet capture 复用已经闭合 directory-sync formatted snapshot，不新增 gofmt/replay。

## 4. 四项正常公开 API positive

四项均是 `package document_test`，正常 Open → actual public Buffer edit → Check/Save/SaveAs → 真实 OS bytes/Stat。无私有 baseline 注入、fake FileInfo、hook、getter、race、sleep 或生产修复；第一次 PASS 保持 positive，绝不虚构先前 RED。

1. **mtime-only（1757）**：actual SameFile/12 encoded bytes/同 mode，只有 mtime touch；Check false、untouched Save 不 replace 或改变 touched timestamp；实际编辑乙后 Save 保 BOM/CRLF、new clean usable baseline。
2. **same-size/same-mtime 外部真实改写（4163）**：独立 12-byte literals 原甲/外丙/稿乙，same actual file、size/mode/恢复原 mtime。Check true、Save ErrConflict，不改 external bytes/receipt 或 draft owner/version/caret/history/metadata/Dirty/baseline；Undo 回原 clean/caret0:1 且 Check 仍 true，Redo 回 dirty draft。实际恢复原 bytes 后 Check false，Save retry 成功，新 clean baseline/保 history/caret，未 Reload。
3. **UTF-16LE 实际 encoded length（5015）**：独立 BOM literals 14 encoded bytes（不是 decoded UTF8=10 或 UTF16 units=6）；实际 Edit 乙 → Save14，尾 Z 改 ZZ → SaveAs16（decoded11/units7），原已保存文件仍为14；真实 bytes/BOM/CRLF/原 Save permissions/public owner/version/caret/history/new baseline/Check false 均保持。Save caret1:1/history1，SaveAs caret7:7/history2。
4. **Linux 真目录 postpublication I/O（5361）**：正常 Open 的 UTF8 BOM12 原文件0600，实际 edit乙，valid old DiskToken/dirty/history1/caret1:1。新 child 0300（w+x/no read）实际 Stat；non-root 要求是 Fatal 而非 skip，defer0700 在 cleanup 前恢复。SaveAs 真实 publish 后 `syncDirectory` 的目录 Open EACCES（PathError Op=open、Path=child）；target 已是独立稿12、regular0600，原 metadata/baseline/draft/version/caret/history/Dirty/source bytes 不变。Undo 回原 clean，Redo 回 dirty；显式恢复0700，同 target retry ErrExists 且 SameFile/bytes/size/mode/mtime 不变；unused new name SaveAs 成功、fresh clean baseline/Check false，原 source/已发布 target 不变。它 **不是** didSave verify I/O、nil guard、CAS 或 Windows 证据。

### 公共验证限制

同步 Save/SaveAs 没有 publication → didSave 之间的中立 public callback。precall 权限会影响更早 I/O；inotify/chmod race 不能成为 deterministic proof。未扩 seam、未开 private backdoor、未修改 production 以便强造测试；Root 选择真实 directory-Sync EACCES 替代并要求保留限制。

## 5. 原始失败、后继与最终门禁

type:dead-end

- 内部 RED28258：`encoded-byte work boundary crossed: delivered=79 allowed=13 grew=true`，numeric1/wrapper2。之后只有已批准边界实现及 necessary signatures；原 oracle identity 不变。
- GREEN01：connector 外层 90 s timeout/reset。实际 Go29841 raw PASS 但 numeric/final/source-after/snapshot-after 缺失；后续只读观察确认 group 已不存在。永久 `INTERRUPTED_WRAPPER_FINAL_MISSING_NOT_PRODUCT_PASS`；旧 raw/intent/started/invocation及缺失记录均保留，不补 final、不采纳、不重标签。
- GREEN02：fresh format/capture/source 后 Go30119 numeric0/wrapper0 原 oracle delivered13/ErrConflict/zero identity；限定内部工作量证明，不冒充 public business proof。
- 四项 public positive 独立补充；最后全 `./core/document` 跑原 binary sharedAtomic、SaveAs/commit/cancellation/receipt/adoption/directory_sync/changed_before_verify/prepublication cleanup、各编码和 wrong-digest 家族，没有仅挑新测试证明整包。
- 原 app `TestReopenClosedWindowCloseCancelsRunningImage` panic（index out of range，447.038s）、case-sensitive reopen `reclose did not dedup with a new ID`、旧 experience05 minimum-input failures 不因本 Task11 结果消失；原失败包/未到达 selectors 保持各自事实，不在此重新执行或修改。

## 6. 最终八份源码 pins

以下 SHA 与 bytes 来自实际最终 input 与 vet input，不是工作根名或手抄 inference。原始三份 baseline 和 exact metadata diff 一并进入固定归档；scope 不包含 worker/BinaryFile/public text API/其他模块修改。

| 文件 | bytes | SHA256 |
|---|---:|---|
'''
pin_lines = []
for name, (size, digest) in PINS.items():
 pin_lines.append(f'| [{Path(name).name}](<F:/AIproject/LevNote/native/{name}>) | {size} | `{digest}` |')
report = intro + '\n'.join(pin_lines) + '\n\n## 7. 数字退出与固定 input receipts\n\n数字列为 Go exit / wrapper exit；wrapper0 只表示命令有限范围 PASS。raw/numeric/final 与完整 SHA 详见后表及归档，所有 closed records timeout=false、cleanup=[]、remaining=false。\n\n| stage | PID/PGID | exit | files/bytes | elapsed s |\n|---|---:|---:|---:|---:|\n' + '\n'.join(rows) + '\n\n'
for label in CLOSED:
 c = checks[label]
 report += f"### {label}\n\n- [真实证据](<{LANE}/runs/{label}/evidence>)；实际 cwd `{c['cwd']}`。\n- argv：`{' '.join(c['argv'])}`。\n- started `{c['started_at']}`；launched `{c['launched_at']}`；ended `{c['ended_at']}`。\n"
 for key, title in (('stdout','raw stdout'), ('stderr','raw stderr'), ('numeric_receipt','numeric'), ('final_receipt','final'), ('expected','expected manifest'), ('three_maps_before','三份 maps')):
  v=c[key];report += f"- {title}: {v['bytes']} B / `{v['sha256']}`。\n"
 report += '\n'
report += '''## 8. 固定归档、原字节镜像与后续

type:progress

小型 primary 归档只包含：上述八份最终 source、三份 actual baseline + exact diff、本报告、上述 15 个精确 document stage 的必要原始 command/start/raw/numeric/final/environment/head、相应 freeze/derive result、GREEN01 interruption classification、7 个既有 metadata runner/capture helpers 和本次报告 metadata helper/准备记录。报告准备01仅 RUN 分类器错误（88 条 unindented RUN 实际57主/31子）而 exit1，无报告/归档/Go；旧失败原保留，02以名字中的 `/` 分类，未改产品输出。重复的 expected/input/maps **不复制正文**，而是完整不可变原引用；它不复制 native 全树/vendor、Go工具链、cache、env HOME、其他业务任务或旧 guard，不启动 Go 或补写 consumed evidence。

`EvidenceReferences.json` 列出这些 stage 的完整既有 evidence、manifest 及 freeze/derive 文件，每项 actual full box path + 完整 bytes/SHA256 + body_in_primary；未复制正文仍在既有 box 原位置保存，**不是完整备份宣称**。`bundle-manifest.json` 给每个归档条目完整 bytes/SHA256 和 actual origin，含 closed_receipts 核对值、final八pins 与 GREEN01 missing records。Archive 为 create-only tar.xz，要求 <90000 B；tar 条目只是 regular relative files，不含 symlink/绝对/.. 路径。创建回执提供 archive exact bytes/SHA256、固定成员数及解压总 bytes，供最多10段已有 opaque 通道传输；这不是新增 product execution。

Lead 先读本报告/最后八文件与 manifest 后再批准/执行原字节镜像。host 仅将固定 source 八项、报告与证据档案进行 opaque import；每个已存在目标需先读、记录 old exact identity、复核 write 前未漂移并保留原内容；每项写后实读比对 final pin。没有重新手写/重格式化 host product 源码。Root 最终差异和后续受影响测试/物理 Windows 统一验收由 Root 按全局收口安排。

可直接执行的下一步：Root 读取 box 的本报告，以及 `delivery/document-fingerprint-review-02/bundle-manifest.json` / `archive-receipt.json`，独核 8 pins、diff、guard/SaveAs adoption/temp.Stat 边界和 small fixed archive；认可后按已有 opaque 流程镜像，不重新执行 document Go/format。Task11 必须等 Root 报告+镜像审核通过再 finite-close。
'''
report_raw = report.encode()
put(REPORT, report_raw)
files['report/autonomous-document-fingerprint-20261112.md'] = report_raw
origins['report/autonomous-document-fingerprint-20261112.md'] = str(REPORT)
references['report/autonomous-document-fingerprint-20261112.md'] = {'path': str(REPORT), **identity(report_raw), 'body_in_primary': True}
reference_raw = json_bytes({'metadata_only': True, 'product_executed': False, 'not_a_complete_backup': True, 'reference_count': len(references), 'body_in_primary_count': sum(v['body_in_primary'] for v in references.values()), 'files': dict(sorted(references.items()))})
files['EvidenceReferences.json'] = reference_raw
origins['EvidenceReferences.json'] = 'actual box file identity records captured by this metadata-only script'
manifest = {'metadata_only': True, 'product_executed': False, 'label': 'document-fingerprint-review-02', 'head': HEAD, 'task': 'task-11', 'source_origin': str(SOURCE), 'source_pins': {p: {'bytes': v[0], 'sha256': v[1]} for p,v in PINS.items()}, 'source_total_bytes': sum(v[0] for v in PINS.values()), 'closed_receipts': checks, 'final_full_package': {'top_run': top_runs, 'top_pass': top_pass, 'nested_run': all_runs-top_runs, 'nested_pass': all_pass-top_pass, 'fail': 0, 'skip': 0}, 'not_whole_application_or_windows_acceptance': True, 'interrupted_green01_missing': ['05-product-command.json','result.json','source-after.json','snapshot-after.json'], 'fixed_stages': list(STAGES), 'files': {k: identity(v) for k,v in sorted(files.items())}, 'origins': origins, 'report': {'path': str(REPORT), **identity(report_raw)}, 'host_scope': 'opaque mirror only exact eight native/core/document files, named own report and own fixed evidence archive; host no product or semantic changes'}
manifest_raw = json_bytes(manifest)
put(OUT / 'bundle-manifest.json', manifest_raw)
files['bundle-manifest.json'] = manifest_raw
archive = OUT / 'document-fingerprint-review-02.tar.xz'
with archive.open('xb') as target:
 with tarfile.open(fileobj=target, mode='w:xz', preset=9) as bundle:
  for name, data in sorted(files.items()):
   member = tarfile.TarInfo(name)
   member.size = len(data); member.mode=0o600; member.mtime=0; member.uid=0; member.gid=0
   bundle.addfile(member, io.BytesIO(data))
with tarfile.open(archive, 'r:xz') as bundle:
 actual = {m.name: bundle.extractfile(m).read() for m in bundle.getmembers() if m.isfile()}
 assert actual == files
assert archive.stat().st_size < 90000, 'primary archive exceeds Root-approved small-transport budget'
receipt = {'metadata_only': True, 'product_executed': False, 'started_at': started, 'ended_at': utc(), 'pid': os.getpid(), 'argv': sys.argv, 'cwd': str(Path.cwd()), 'script': {'path': str(Path(__file__).resolve()), **identity(Path(__file__).read_bytes())}, 'archive': {'path': str(archive), **identity(archive.read_bytes())}, 'manifest': {'path': str(OUT / 'bundle-manifest.json'), **identity(manifest_raw)}, 'report': {'path': str(REPORT), **identity(report_raw)}, 'fixed_file_count': len(files), 'fixed_file_bytes': sum(len(v) for v in files.values()), 'source_file_count': len(PINS), 'source_bytes': sum(v[0] for v in PINS.values()), 'all_members_readback_exact': True, 'exit_code': 0, 'not_product_or_whole_application_or_windows_acceptance': True}
put(OUT / 'archive-receipt.json', json_bytes(receipt))
print(json.dumps(receipt, ensure_ascii=False, indent=2))
print('TASK11_REPORT_FIXED_ARCHIVE_READY_FOR_ROOT_REVIEW_NO_GO')
