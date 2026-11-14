# Task50 · Markdown 草稿恢复回调所有权交接

## 任务状态（type:state）

实现者：`recovery-identity`；固定 BASE：`ce58c44b2e7bac439883de0a780b8fe7073dad3a`。
生产修改只涉及 [draft.go](<F:/AIproject/LevNote/native/modules/markdownview/draft.go#L249-L280>) 的 `RecoverDraft`，净增加9行；新增 [外部包回归](<F:/AIproject/LevNote/native/modules/markdownview/recovery_identity_test.go>) 和本交接。
7项最终公开回归通过：Go10496，2026-11-14T14:49:07.029561Z–14:49:09.557462Z，exit0，7 RUN/7 PASS/0 FAIL/0 SKIP。
本篇及两源文件交给唯一 `freeze-01`；[冻结范围](<F:/AIproject/LevNote/.localtools/markdown-recovery-identity-20261114/freeze-01/scope.json>) 与 [全源快照](<F:/AIproject/LevNote/.localtools/markdown-recovery-identity-20261114/freeze-01/source.json>) 是冻结后字节身份的权威回执。
独立双轴审阅、最终 affected suite/vet/build、Git及应用/平台验收均仍由lead负责；本交接不代表应用或新包验收。

## 目标与约定（type:goal）

遵循不可变 [原固定spec](<F:/AIproject/LevNote/.localtools/markdown-recovery-identity-spec-20261114.md>)（SHA256 `9ca7fc0fb6a391161d9455be683280d44175acb13429bae992767aea87d5f425`）及并列 [运行时修订01](<F:/AIproject/LevNote/.localtools/markdown-recovery-identity-amendment-20261114-01.md>)（SHA256 `bcc34e65c248b90f407f361751ff7be71570ca2a93c47f3a294df58bbd7dacb8`）。
公开seam为 Adapter/Create/Host恢复与Report回调、Draw、RecoverDraft/DiscardDraft/CompositionActive/Warnings、真实Tester可访问cell/field指针及原生文本/组合输入、公开text.Buffer观察；无私有状态、反射、unsafe或toolkit树检查。
恢复只复制发起草稿已接受的字面量；成功只清理仍相同owner、版本未变且无新的active composition的草稿。
替换owner、sameowner后输入及same-text新组合保留；旧回调错误原样返回并报告，但不绑定替换草稿的问题状态。
主CRLF字节、history、定向selection、版本和内容准入上限维持权威；未接受的incoming输入不成为恢复payload。

## 已试方案与失败原因（type:dead-end）

原无guard→replacement owner第二恢复丢失（真实首RED）；pointer-only→sameowner新文本丢失（真实control RED）；pointer+version→same-text新组合丢失（真实control RED）。
旧owner错误无条件setDraftError→新owner出现旧问题标签与Warnings（真实control RED）；按当前owner设置问题或仅报告旧错误后同oracle通过。
control03期待原error出现在HasText→UI实际为问题标签加Warnings/tooltip，属fixture观察错误；只纠正公开观察，生产不变，原流保留。
control06假定Editor.Flush调用Changed→实际`changes=0 copies=1`，属错误fixture前提；lead修订01撤下该test/helper，不Skip、不弱化断言、不增预想Flushguard。
所有消耗过的label及原始流保持；未执行原Windows失败重试、ACL/UAC/替代shell修复、Computer Use、formatter写入、最终suite/vet/build或Git。

## 进度与下一步（type:progress）

1. lead先核对 [freeze-01](<F:/AIproject/LevNote/.localtools/markdown-recovery-identity-20261114/freeze-01/result.json>) 的scope/full-source与本篇记录，并按BASE对 [生产diff](<F:/AIproject/LevNote/.localtools/markdown-recovery-identity-20261114/production-diff-01.txt>) 和完整新增测试作Standards/Spec独立审阅。
2. 不重跑已完成focused label；审阅后的修正须单独范围/证据，随后由lead执行一次最终affected suite/vet/build。
3. lead对三路径执行canonical path-filtered `git hash-object --path=PATH -- PATH`，完整staged diff及原流审计后正常双main交付。
4. 原Task46/全部历史包和失败证据、其余1217原product、7pins、534excluded metadata、CON metadata及CIinactive不变。
5. 最终物理Windows IME/真实clipboard/GUI/PE/WM_CLOSE/包装及全应用可交付条件另行验收；本次source-focused不建立这些结论。

## 最终实现与回归覆盖

- Flush之前捕获发起draft指针；Flush之后读取其已接受字面量与buffer version。
- Host恢复成功之后先比较owner，再比较version，再检查当前composition；任一新输入证据成立即保留当前draft，不执行旧成功clear。
- Host恢复错误且owner相同：沿用既有setDraftError/可见问题出口；owner已更换：report原error而不污染替换owner；两支均返回原error。
- 原无草稿quiet no-op、默认/路径copy title、普通成功clear、缺callback错误出口均保持。
- [replacement owner](<F:/AIproject/LevNote/native/modules/markdownview/recovery_identity_test.go#L50-L95>) 保留原首oracle，第二次复制独立新字面量且0主Changed，定向选区保持。
- 其余6项分别覆盖sameowner新文本、正常success/no-draft、原错误与重试、same-text新composition及`最终𠮷🙂文字`原生最终提交、旧错误不污染replacement、未接受ErrUTF8 incoming仍只复制`已接受𠮷🙂`。
- accepted-byte control使用真实native `Compose("未接受\xff🙂", 4)`，观测原ErrUTF8 Report后恢复已接受literal；不是私有draft大小限制注入，也不声称跑过128MiB oversized IME压力。

## 首oracle和运行时修订的证据保真

[首test.raw](<F:/AIproject/LevNote/.localtools/markdown-recovery-identity-20261114/red-01/first-test.raw>)：92行、3363B、SHA256 `ffce11ad9b4619aed349e446dbdf63c0490839a90bd415d9cbd82cdef2f43e7a`。
[原production.raw](<F:/AIproject/LevNote/.localtools/markdown-recovery-identity-20261114/red-01/original-draft.raw>)：307行、8926B、SHA256 `e02a59da5fc713f5ea9db2b8fa656f00b92edf39daed137f29291c8cfffecb22`。
原constants/helpers/comment/首test构成的完整projection逐字节不变，位置仍在后来controls之前。新增errors/slices/utf8集中imports改变whole-file prefix；**whole-byte-prefix不相等，不作相等声明**。
[control06原fixture完整快照](<F:/AIproject/LevNote/.localtools/markdown-recovery-identity-20261114/control-06-fixture-hypothesis.raw>)：11449B，SHA256 `710f130a41a08b9dcafd30ef896cf745207f90bde55a7132f726f84c00fb94ac`，经原before记录比对恢复并create-only保存。
control06原诊断使用PRODUCT字样，但callback前提未发生；按修订01计fixture hypothesis failure，不算真实产品RED，不留shipping suite，不计该假设已通过。
[Editor.Flush/UnmarkText](<F:/AIproject/LevNote/native/ui/editor/editor.go#L429-L455>) 实际成功只Invalidate不changed，issue提前返回；RecoverDraft调用具体Editor.Flush，不是会syncDraft的Markdown View.Flush。后续若引入该回调路径，需新的公开回归再判owner。

## 执行来源与全输入守卫

runner：[run.py](<F:/AIproject/LevNote/.localtools/markdown-recovery-identity-20261114/run.py>)，185行、9548B，SHA256 `fbc99e1380a6fffd2854ff1de41d21f971fb104a140b4fb5a3ad3785ac3ed51c`。
实际launcher模式：bundled Python `-I -B .localtools/markdown-recovery-identity-20261114/run.py LABEL focused '^TestMarkdownRecoveryIdentityNAME$'`；最终union模式为 `'^TestMarkdownRecoveryIdentity.*$'`。
实际runner Popen是已钉Bash加 `.localtools/gate.sh test -json -count=1 -timeout=90s -run PATTERN ./modules/markdownview/`，ROOT cwd为`F:/AIproject/LevNote`。
[gate](<F:/AIproject/LevNote/.localtools/gate.sh>) 自cd至native，实际Go路径为`../.localtools/go1.27.3/bin/go.exe`；offline vendor/telemetryoff/本地cache与tmp保持。
receipt键`actualGoArgv`为**gate-derived expected argv**，非OS command-line capture；其首root-relative路径不能理解成在native cwd的原始调用路径。独立捕获Go PID、parent、image、创建与观测UTC、wrapper/gate PID、真实exit和原流；原receipt不重写。
freeze的`rawBlobOID`仅原字节Git blob framing SHA1，不是canonical Git path/filter OID；由lead另核canonical OID。
原Task46 full1218 snapshot SHA256 `8591837886bb93ded5efa38efd151207906be04caf6c45dac7c55eed02e12162`，aggregate `de7753a02378bca509103a693dea528431267790e37e1989f97e2e8ed10f3287`。
本轮全1219 product aggregate为`3f0aefcaf9f8f2ed88ea82bae11c6ab8f98625981864550af83814a6a6252c5c`；DOC另列scope/guard。每次run均前后inputsEqual，原1217受保护product、7pins、534excluded、CON与Task46闭合全证据守卫通过。
最终源身份：draft9147B SHA256 `7d76f090f0b3af2d4fb383b9f416e4301acfe626848310f3e890ffcd4d5be9f0`；test13266B SHA256 `8b773e93ba46d0a2f2eaf59238c9574072f0fb109ab2f9f00e05c20135b897cf`。

## 逐slice原流台账

下表时间均为2026-11-14 UTC；每项focused均一个top-level RUN，zeroSKIP；union为7top-level。没有build/setup失败冒充RED。
完整Go image/创建UTC/parent/wrapper/gate/argv、before/after/source摘要及result SHA列于 [独立读取摘要](<F:/AIproject/LevNote/.localtools/markdown-recovery-identity-20261114/evidence-summary-01.json>)；摘要由create-only只读脚本逐项核对原流hash生成，不替代原receipt。

| Label | UTC start–end | Go PID | Exit / 定性 | 原stdout SHA256 |
|---|---|---:|---|---|
| red-01 | 14:15:35.093587–14:15:38.075247 |17636|1 / replacement真实RED|`a4e8457e989040af252504f89ad57955127b5e4a4f1f730aa52e127ee7e898e9`|
| green-01 |14:17:01.787290–14:17:05.809410|22840|0 / 首oracle不变GREEN|`e053cfa3ef826826dca31b71c999e7526b303723a825dfd19b8117e791909b3e`|
| control-01 |14:18:06.355428–14:18:09.286349|20948|1 / sameowner真实RED|`74e1732117485cfccbb3c47f244e3fdda4c2bdc6637c38081848d52e3da8a8d5`|
| green-02 |14:19:31.637253–14:19:35.071230|23244|0 / 同oracleGREEN|`4c48fb368f29ac93e7b98289602a87b2eeea8e19951fbd972c80f89732a43756`|
| control-02 |14:21:01.846083–14:21:20.573452|9412|0 / normal/no-draft|`9e5aa70973f2594fa02bb52d30ce8fde6d07a012f30bdf7edeefe1e015c5f546`|
| control-03 |14:22:11.959008–14:22:14.902266|14412|1 / fixture HasText观察错误|`e343cf88c16ec711870da143d63cc604fa73a271123587d34fefb8e848ad3d53`|
| control-04 |14:23:55.066369–14:23:57.657276|300|0 / 公开label+Warnings纠正|`dedf47a08209023e0a72fe32cda5c3e7ab8e4d88232666788ea3dff51f4be6b6`|
| control-05 |14:28:11.989614–14:28:14.651578|8608|1 / 新same-text组合真实RED|`21045cb3db06ff2e28301e0310b1995b8080868af898dd20e8f232e92f468faa`|
| green-03 |14:29:31.637429–14:29:34.978432|4740|0 / 同oracleGREEN|`11fc265a6d9aa7d5837b802a739ab3cc9641cfd3744385e5c0c32b93034fc88a`|
| control-06 |14:32:15.153431–14:32:18.878157|22472|1 / fixture callback假设失败|`da99ddf11dfcea5010d92820199d3dbd33fe434d4b5eb8d59f48444b8f6adbc6`|
| control-07 |14:38:15.942324–14:38:19.134874|25844|1 / replacement旧错污染真实RED|`636e6979cc7252345052d84fc48196d649575a7e3c7e524effcbd93bd02e75e7`|
| green-04 |14:40:27.846673–14:40:31.701247|25420|0 / 同oracleGREEN|`5a44245a31cd0df1212db72dce3a6b0d7b5183d31018600c50e9e875c3b7f2dd`|
| control-08 |14:47:41.606207–14:47:44.327224|8388|0 / 已接受byte恢复|`31fb32b0cefc5201379c897c5d1de9b95265ef88845371558894d9d57751ac7a`|
| focused-union-01 |14:49:07.029561–14:49:09.557462|10496|0 / 7PASS 0FAIL 0SKIP|`63678dd55a6dabcceeafbb59fd2cf852bbd2ea7201d3bab1aaa0452cf3536a60`|

4个真实产品失败均先于对应窄修复；2个fixture失败分别修正观察/撤回不成立的假设。14次focused执行已闭合，无仍在运行的产品job。
[union原receipt](<F:/AIproject/LevNote/.localtools/markdown-recovery-identity-20261114/focused-union-01/result.json>) SHA256 `8259e084d4ebda2374aa1ed4cc3a9902afc8438c202c2c4432de41857bdc6b12`；[union原stdout](<F:/AIproject/LevNote/.localtools/markdown-recovery-identity-20261114/focused-union-01/stdout.raw>)7044B。
union before/after473288B均SHA256 `ccf20ebe6baab8fd7d9c03e5cec12ed947f2aeda10f364a916383af50b840126`；source229221B SHA256 `585b7c71dbe62bfee7d25309c1fc7eafba5de2bfaca751e9333777589ee5c1ee`。
