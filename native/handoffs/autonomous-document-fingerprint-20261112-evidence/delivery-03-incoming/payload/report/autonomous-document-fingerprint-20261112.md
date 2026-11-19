# 原生文档指纹精确编码字节边界 · Task11

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

当前已接受的 runnable reference 仍是 [.29](<F:/AIproject/LevNote/bin/levnote-native-worker-20261110.29/levnote-native.exe>)；本工作未建立 .30 或替代其验收。

## 2. 固定范围、输入与执行环境

type:state

- 克隆/输入 HEAD：`8897244ffa4f00f5ec51f95545e7a3ca27d37d97`；产品对照基线：`77454291ce1aad7c5a617ab19d507e192154baf6`。CI 的后续 source-only 暂停提交未被推定已更新 box 普通克隆。
- 授权 box：`cb-698f697fed7c`；所有语义源码、测试、gofmt、Go 门禁均在 `/home/user/levnote-dev-reliability-20261112-01` 隔离 lane。没有本机 Go、复跑旧标签、替代编译器、下载、网络依赖、遥测或配置变更。
- 八份最终源码来自 [最终 immutable input](</home/user/levnote-dev-reliability-20261112-01/inputs/document-final-core-tests-01/native>)，不是未格式化 mutable work；vet immutable input 的对应八文件完全相同。
- 最终 1196 files / 16996849 B；变化集相对克隆基线恰好是下表八文件（3 份存量调整，5 份新增测试），其余基线文件原字节保留。
- Go1.27.3；`GOTOOLCHAIN=local CGO_ENABLED=0 GOWORK=off GOVCS=*:off GOFLAGS=-mod=vendor GOPROXY=off GOSUMDB=off GOENV=off GOTELEMETRY=off GOOS=linux GOARCH=amd64`。HOME/XDG/tmp/cache 为 owned lane；telemetry 在 Go 启动前关闭。有效 Go 环境的 GOENV 文件名为空。
- freeze、format、capture、test/vet 分开调用；每个已闭合 Go 命令均有原始 `.command.json` prelaunch intent、`.started.json` PID/PGID、数字退出记录、final、实际 streams SHA/bytes、timeout/cleanup/remaining 状态及三份输入清单。文件名不是不存在的 `.intent.json`；中断 GREEN01 的数字/final/后清单缺失例外见第5节。
- 旧 runner 中 `owner=native-reliability/task-5` 是原 provenance 原文，未为 Task11 重标。

## 3. 原始问题与实现

type:goal

### 3.1 指纹只读取实际 expected encoded length + 1

旧 fingerprint 通过 `io.Copy(h, f)` 可将 before-Stat 后增长文件读到 EOF，工作量没有精确编码字节边界。内部真实文件 oracle 使用独立 12-byte UTF-8 BOM + `甲🙂\r\n` 字面量；实际 append 在 production before-Stat 后、first Read 前发生。其 non-embedded `*os.File` 包装避免继承 `WriteTo` 绕过计数。真实 RED28258 delivered79、allowed13，在 byte-work assertion 失败；它不是编译失败或 public-business RED。

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
| [document.go](<F:/AIproject/LevNote/native/core/document/document.go>) | 14511 | `44da6396913741a18ed5ebd1d0c668853d28ad399eba9b13abb02aa08f76a3fe` |
| [atomic.go](<F:/AIproject/LevNote/native/core/document/atomic.go>) | 4873 | `641601d61fe3da0ec83954c6b199cac46864099ffa78f09e2a309180491fec83` |
| [utf8_save_test.go](<F:/AIproject/LevNote/native/core/document/utf8_save_test.go>) | 8238 | `7c02ad0c795cadda2b93c443d6816d742ff13bf0e9ae26facb3e8149f31b3ec0` |
| [fingerprint_boundary_test.go](<F:/AIproject/LevNote/native/core/document/fingerprint_boundary_test.go>) | 3399 | `f0d3204bf2b7430eb1a95ca183ade789816faf7752433cad06c2e6a74f434aef` |
| [fingerprint_public_test.go](<F:/AIproject/LevNote/native/core/document/fingerprint_public_test.go>) | 2957 | `3c94db4b1ac0604f03ebb3cc26fdf62e661dadfd266ad74b06d6d9fe587955d5` |
| [fingerprint_conflict_public_test.go](<F:/AIproject/LevNote/native/core/document/fingerprint_conflict_public_test.go>) | 6993 | `c0556277c6d7ae7c7cf4a72d75d3901a23aedcedec94aa00711dc5ac0ab20b69` |
| [fingerprint_encoding_public_test.go](<F:/AIproject/LevNote/native/core/document/fingerprint_encoding_public_test.go>) | 4870 | `40212ed1a6c5319b16e2e107ae1aaa8530b3d381fe3dac85d64266436f974edc` |
| [fingerprint_directory_sync_public_linux_test.go](<F:/AIproject/LevNote/native/core/document/fingerprint_directory_sync_public_linux_test.go>) | 7142 | `8f68c732927b6375edeb8b589facef356a360f944512b4564968afaa6fe51951` |

## 7. 数字退出与固定 input receipts

数字列为 Go exit / wrapper exit；wrapper0 只表示命令有限范围 PASS。raw/numeric/final 与完整 SHA 详见后表及归档，所有 closed records timeout=false、cleanup=[]、remaining=false。

| stage | PID/PGID | exit | files/bytes | elapsed s |
|---|---:|---:|---:|---:|
| document-fingerprint-red-01 | 28258/28258 | 1/2 | 1192/16971991 | 2.0337316209916025 |
| document-fingerprint-green-02 | 30119/30119 | 0/0 | 1192/16974887 | 5.794274664018303 |
| document-public-mtime-positive-01 | 1757/1757 | 0/0 | 1193/16977844 | 5.558381884002301 |
| document-public-conflict-positive-01 | 4163/4163 | 0/0 | 1194/16984837 | 2.4344512710013078 |
| document-public-encoding-positive-01 | 5015/5015 | 0/0 | 1195/16989707 | 1.5295129930018447 |
| document-public-directory-sync-positive-01 | 5361/5361 | 0/0 | 1196/16996849 | 1.627826804004144 |
| document-final-core-tests-01 | 5691/5691 | 0/0 | 1196/16996849 | 2.1367834670018055 |
| document-final-core-vet-01 | 5868/5868 | 0/0 | 1196/16996849 | 1.5796118939979351 |

### document-fingerprint-red-01

- [真实证据](</home/user/levnote-dev-reliability-20261112-01/runs/document-fingerprint-red-01/evidence>)；实际 cwd `/home/user/levnote-dev-reliability-20261112-01/runs/document-fingerprint-red-01/source/native`。
- argv：`/home/user/go/bin/go test -v -count=1 -timeout=120s -run ^TestFingerprintGrowthReadsAtMostExpectedEncodedBytesPlusOne$ ./core/document`。
- started `2026-11-12T11:00:09.818737+00:00`；launched `2026-11-12T11:00:09.821144+00:00`；ended `2026-11-12T11:00:11.852468+00:00`。
- raw stdout: 379 B / `f8a8815361c9b205c4c3463210375acdee905a201abf1b4703a31415c871de8b`。
- raw stderr: 0 B / `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`。
- numeric: 2675 B / `f62aba2e27d102a6d288b2dceaeac5d35bc570d1309ae3d7ebcbf0c71fccc220`。
- final: 5742 B / `32d54fe4286544d136cb7d34bd4f723d7fe8da1501c053c12c3e2d3caea93ead`。
- expected manifest: 200297 B / `fb9886ba6395e952542d422853863a9eb06e78c117c1f409d57956a3496b7b74`。
- 三份 maps: 199418 B / `a49db7a25cb7a9c72759486300b83080d3fc901268d46cc7beefd43e53d7658f`。

### document-fingerprint-green-02

- [真实证据](</home/user/levnote-dev-reliability-20261112-01/runs/document-fingerprint-green-02/evidence>)；实际 cwd `/home/user/levnote-dev-reliability-20261112-01/runs/document-fingerprint-green-02/source/native`。
- argv：`/home/user/go/bin/go test -v -count=1 -timeout=120s -run ^TestFingerprintGrowthReadsAtMostExpectedEncodedBytesPlusOne$ ./core/document`。
- started `2026-11-12T12:01:14.840193+00:00`；launched `2026-11-12T12:01:14.845638+00:00`；ended `2026-11-12T12:01:20.634466+00:00`。
- raw stdout: 200 B / `cf1da4913834f12e352612064651d40795441e14b893f8481306408eefc8a11d`。
- raw stderr: 0 B / `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`。
- numeric: 2694 B / `8fdfaaabb9e729ce49c6880d8129c58808b006be9409827eef0fdf8fa6c66794`。
- final: 5789 B / `65caefabcbf733ce56b4f36b8b2abc1b9f99f3e37f8de5c1e5a572c42c8fb140`。
- expected manifest: 201085 B / `2e7ea5dd5cd385dbfad43aa2793e84e19711df8e8702dc8a397192165be307e1`。
- 三份 maps: 199418 B / `53d921d870f059a6779832f32f64545f2ca840bf5dcbca7762e454565e346890`。

### document-public-mtime-positive-01

- [真实证据](</home/user/levnote-dev-reliability-20261112-01/runs/document-public-mtime-positive-01/evidence>)；实际 cwd `/home/user/levnote-dev-reliability-20261112-01/runs/document-public-mtime-positive-01/source/native`。
- argv：`/home/user/go/bin/go test -v -count=1 -timeout=120s -run ^TestDocumentMtimeOnlyTouchKeepsCheckAndSaveAvailable$ ./core/document`。
- started `2026-11-13T02:04:08.112075+00:00`；launched `2026-11-13T02:04:08.114849+00:00`；ended `2026-11-13T02:04:13.670453+00:00`。
- raw stdout: 186 B / `e5127baedbc0e65268bb742230a8fa47a6f79cccbe7b0e2efa34eca52541e5bb`。
- raw stderr: 0 B / `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`。
- numeric: 2725 B / `b21d70bae2717605345092c90e411638bc3a01a07e35b143249500faa0e8f5ff`。
- final: 5841 B / `44fd49377477f88f427360c9bcfc21a3394bad456ee3e057bebe075b65aa9444`。
- expected manifest: 201480 B / `bfa22cc9d13bf66f9aca1974a699ded4846e46b81be1d27269433f14e28158b0`。
- 三份 maps: 199579 B / `3ce7b33473c52e5bb9d051c459d28a4d7a22b002bc66e85679e9bb8af0de0d8f`。

### document-public-conflict-positive-01

- [真实证据](</home/user/levnote-dev-reliability-20261112-01/runs/document-public-conflict-positive-01/evidence>)；实际 cwd `/home/user/levnote-dev-reliability-20261112-01/runs/document-public-conflict-positive-01/source/native`。
- argv：`/home/user/go/bin/go test -v -count=1 -timeout=120s -run ^TestDocumentSameSizeSameMtimeConflictPreservesUndoAndRetry$ ./core/document`。
- started `2026-11-13T02:46:33.034227+00:00`；launched `2026-11-13T02:46:33.035601+00:00`；ended `2026-11-13T02:46:35.468673+00:00`。
- raw stdout: 198 B / `e064e2a8f57cf1e58ac14945dc6167a0d2623480161a006f239dc6576e3eb4a6`。
- raw stderr: 0 B / `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`。
- numeric: 2762 B / `36cc79caf5be2b951ce04da8bbace52798ec34813a16136df92c2001387030c2`。
- final: 5905 B / `3bb3f3f550f750d280928308df5c04ff53047c9084c53080ef7a9a64536cbab8`。
- expected manifest: 201896 B / `233c958e66cc514ec0a13f433d6f4d2f341b9977b572c3f87a97899dfa60bd95`。
- 三份 maps: 199749 B / `b418892185e824d14a641bef2a17600c3c4ecb78681ee0628378e19ee16d09a9`。

### document-public-encoding-positive-01

- [真实证据](</home/user/levnote-dev-reliability-20261112-01/runs/document-public-encoding-positive-01/evidence>)；实际 cwd `/home/user/levnote-dev-reliability-20261112-01/runs/document-public-encoding-positive-01/source/native`。
- argv：`/home/user/go/bin/go test -v -count=1 -timeout=120s -run ^TestDocumentUTF16SaveAndSaveAsUseActualEncodedLength$ ./core/document`。
- started `2026-11-13T03:10:11.618852+00:00`；launched `2026-11-13T03:10:11.620349+00:00`；ended `2026-11-13T03:10:13.148388+00:00`。
- raw stdout: 186 B / `849472e4ce0e3744a88a639e1dd9b381290e7a8fa14a894ec00f6615f92520cf`。
- raw stderr: 0 B / `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`。
- numeric: 2756 B / `c87fb004fe33ce158a7c0bb718470b710694ec3a7bc0f33f190c45777f01b3fe`。
- final: 5893 B / `69bddf94316e40cc4827e301f35e9139b1fee19ae00b09def5a2dd67485082ff`。
- expected manifest: 202303 B / `60c1d49723e1da75034dd6c554338e941b95742da297f99914213216b69c0f11`。
- 三份 maps: 199919 B / `30b02b38901a360aa8f6b3ed4a4522a1922454d86cc5edfaa7001f62b071952a`。

### document-public-directory-sync-positive-01

- [真实证据](</home/user/levnote-dev-reliability-20261112-01/runs/document-public-directory-sync-positive-01/evidence>)；实际 cwd `/home/user/levnote-dev-reliability-20261112-01/runs/document-public-directory-sync-positive-01/source/native`。
- argv：`/home/user/go/bin/go test -v -count=1 -timeout=120s -run ^TestDocumentSaveAsDirectorySyncErrorKeepsPublishedBytesAndRetry$ ./core/document`。
- started `2026-11-13T03:26:58.073857+00:00`；launched `2026-11-13T03:26:58.075027+00:00`；ended `2026-11-13T03:26:59.701718+00:00`。
- raw stdout: 208 B / `51a3509f21dda7f692d4948567d6f59a7a4405cdbe0ac40c13cc187e7680d483`。
- raw stderr: 0 B / `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`。
- numeric: 2826 B / `396d0d63d5350610c93d9bc4b0e37100a115c13a34e5e246fcb0593767d0a0cc`。
- final: 6016 B / `514031a71e5510d5604bd57f2a9e979cb6769700b24c7ac8cb79aba98c419d21`。
- expected manifest: 202752 B / `934bbbf502d3f3e65137794984e0ad5f436922dd6a9c0f2ecc8dbcc6c684aac8`。
- 三份 maps: 200101 B / `7c8e44f1984b2c44beace46757652e53258f6bc034fb94714b17855ae43726ee`。

### document-final-core-tests-01

- [真实证据](</home/user/levnote-dev-reliability-20261112-01/runs/document-final-core-tests-01/evidence>)；实际 cwd `/home/user/levnote-dev-reliability-20261112-01/runs/document-final-core-tests-01/source/native`。
- argv：`/home/user/go/bin/go test -v -count=1 -timeout=240s ./core/document`。
- started `2026-11-13T03:40:49.264430+00:00`；launched `2026-11-13T03:40:49.265495+00:00`；ended `2026-11-13T03:40:51.401208+00:00`。
- raw stdout: 12384 B / `b324a2895022082d43b6d0612c792d47bc8aead1d1f669e069db84b65a4b1ff1`。
- raw stderr: 0 B / `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`。
- numeric: 2604 B / `d775bac2152ecc794602f0aef25235b2cdfc7340d08300f3d07f3f62a7dab58e`。
- final: 5607 B / `7f276ff9552068bce7182790a8c310a59c30639e4b5aac22fdf9feae11cb76a5`。
- expected manifest: 202738 B / `bcb3119e0144d0b0509dfb8f8ec4d859ed0e273a8a1b9c790f3268308fdb544a`。
- 三份 maps: 200101 B / `7c8e44f1984b2c44beace46757652e53258f6bc034fb94714b17855ae43726ee`。

### document-final-core-vet-01

- [真实证据](</home/user/levnote-dev-reliability-20261112-01/runs/document-final-core-vet-01/evidence>)；实际 cwd `/home/user/levnote-dev-reliability-20261112-01/runs/document-final-core-vet-01/source/native`。
- argv：`/home/user/go/bin/go vet ./core/document`。
- started `2026-11-13T03:41:41.063686+00:00`；launched `2026-11-13T03:41:41.067032+00:00`；ended `2026-11-13T03:41:42.643290+00:00`。
- raw stdout: 0 B / `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`。
- raw stderr: 0 B / `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`。
- numeric: 2532 B / `bacb93a87d18e790a219978bdaddf3b49a8538c1f9db2849d0a7eb0f228257a4`。
- final: 5466 B / `3c783f57eafeaa4e8b69019a18424e20fa605917663674557c9e2e07a80b4bb8`。
- expected manifest: 202736 B / `cb330730eb9449402a1b4eb9957987f36a2eabfa9e90700c58da0869e3d7cb45`。
- 三份 maps: 200101 B / `7c8e44f1984b2c44beace46757652e53258f6bc034fb94714b17855ae43726ee`。

## 8. 固定归档、原字节镜像与后续

type:progress

小型 primary 归档只包含：上述八份最终 source、三份 actual baseline + exact diff、本报告、上述 15 个精确 document stage 的必要原始 command/start/raw/numeric/final/environment/head、相应 freeze/derive result、GREEN01 interruption classification、7 个既有 metadata runner/capture helpers 和本次报告 metadata helper/准备记录。报告准备01仅 RUN 分类器错误（88 条 unindented RUN 实际57主/31子）而 exit1，无报告/归档/Go；旧失败原保留，02以名字中的 `/` 分类，未改产品输出。重复的 expected/input/maps **不复制正文**，而是完整不可变原引用；它不复制 native 全树/vendor、Go工具链、cache、env HOME、其他业务任务或旧 guard，不启动 Go 或补写 consumed evidence。

`EvidenceReferences.json` 列出这些 stage 的完整既有 evidence、manifest 及 freeze/derive 文件，每项 actual full box path + 完整 bytes/SHA256 + body_in_primary；未复制正文仍在既有 box 原位置保存，**不是完整备份宣称**。`bundle-manifest.json` 给每个归档条目完整 bytes/SHA256 和 actual origin，含 closed_receipts 核对值、final八pins 与 GREEN01 missing records。Archive 为 create-only tar.xz，要求 <90000 B；tar 条目只是 regular relative files，不含 symlink/绝对/.. 路径。创建回执提供 archive exact bytes/SHA256、固定成员数及解压总 bytes，供最多10段已有 opaque 通道传输；这不是新增 product execution。

Lead 先读本报告/最后八文件与 manifest 后再批准/执行原字节镜像。host 仅将固定 source 八项、报告与证据档案进行 opaque import；每个已存在目标需先读、记录 old exact identity、复核 write 前未漂移并保留原内容；每项写后实读比对 final pin。没有重新手写/重格式化 host product 源码。Root 最终差异和后续受影响测试/物理 Windows 统一验收由 Root 按全局收口安排。

可直接执行的下一步：Root 读取 box 的本报告，以及 `delivery/document-fingerprint-review-03/bundle-manifest.json` / `archive-receipt.json`，独核 8 pins、diff、guard/SaveAs adoption/temp.Stat 边界和 small fixed archive；认可后按已有 opaque 流程镜像，不重新执行 document Go/format。Task11 必须等 Root 报告+镜像审核通过再 finite-close。
