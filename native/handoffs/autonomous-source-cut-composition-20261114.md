# Task46 · 源码剪切确认回调组合所有权 · 有限实现交接

## 固定输入与实现

- 唯一实现/focused owner：cut-composition；task-46 revision1→2 claim。固定 BASE/HEAD `5163e0538e96aa9953e569943dfbdb3a0e182d67`，无提交或推送。
- 已完整读取 [Spec](<F:/AIproject/LevNote/.localtools/source-cut-composition-spec-20261114.md>)，3316B/25行/SHA256 `2f1448e3309af6ceca0d95c2c6780b538cfcbdb82ec63f1e6da97567339c7eda`；[原生约束](<F:/AIproject/LevNote/native/AGENTS.md#L14-L43>)与[离线门禁](<F:/AIproject/LevNote/native/AGENTS.md#L567-L602>)已读。
- 已加载 tdd/code-review；procedure查重无匹配。公开 seam 已由 Spec约定；不再委派。实现自审不是 root独立 Standards/Spec接受。
- [editor.go](<F:/AIproject/LevNote/native/ui/editor/editor.go>)：新增 request.owner CompositionToken、在 compositionIssue之后/原身份检查之前比较owner；UnmarkText之后捕获owner。写错误优先、copy-only提前返回、原issue对象及 Buffer/version/selection/viewGeneration/readonly/ReleaseView守卫不变。
- 生产改动为8新增/1删除行（净+7，含2行why注释），没有refactor；[新测试](<F:/AIproject/LevNote/native/app/source_cut_composition_test.go>)394行/17642B，10顶层测试、无子测试/新Skip/旧断言变化。测试使用公开App初始化、Reveal(1,2)、Tester.Command/Type/Key/Frame、SetMarkedText/MarkedRange/Flush与Buffer公开状态及Undo/Redo。

## 原始RED和逐片控制

第一测试单独写出并执行，再最小修正、同名GREEN；每个后续测试单独追加并fresh focused；没有整批预写。所有原receipt/raw均create-only保留。

| label / 原result | 实际GoPID | UTC起止（2026-11-14） | exit / 测试结果 | stdout.raw SHA256 |
|---|---:|---|---|---|
| [red-01](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/red-01/result.json>) | 18752 | 12:20:45.520323–12:20:52.379580 | 1 / 1FAIL，真实产品RED | `429f6b820c4fde8c0ebf2f1245e1016a8b868eca9aa8be91eef308fe71ce604a` |
| [green-01](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/green-01/result.json>) | 24240 | 12:22:05.159662–12:22:12.440053 | 0 / 1PASS | `5d66e2f37789bf6e4668d1e5ae4d1d9f3a71506fb70d949c11641c15d699839f` |
| [control-01](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/control-01/result.json>) | 22088 | 12:23:01.143789–12:23:05.066413 | 1 / 0test，内置vet构建失败 | `ae794ab63623f8ed8021de1206d46ea650f6b1c6b707c8de735541446f94099f` |
| [control-02](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/control-02/result.json>) | 10228 | 12:23:52.657506–12:23:58.312197 | 0 / 1PASS，delayed owner/duplicate | `0b3e50a8f7aca9af44ead785a39fe506d57e1596e120e6ceb3adeace709313f9` |
| [control-03](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/control-03/result.json>) | 23024 | 12:24:54.463188–12:25:00.547240 | 0 / 1PASS，Copy正常cut首GREEN | `b3d139383a344efbd0ac44611a9247ebb3b4e082f3dfcbb873430ae4f84b2b09` |
| [control-04](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/control-04/result.json>) | 6792 | 12:25:58.883571–12:26:05.515338 | 0 / 1PASS，CopyThen正常cut首GREEN | `e24ae428a453c1e0de6de8d37836a65bcd90b2bd0c049f04043de214226c8ad9` |
| [control-05](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/control-05/result.json>) | 23496 | 12:27:30.210326–12:27:36.397443 | 0 / 1PASS，Copy copy-only首GREEN | `295f9b03250d1dd227f6d8cf64184d64517f40363a55f73113461638ca0bcf07` |
| [control-06](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/control-06/result.json>) | 24404 | 12:28:19.471233–12:28:25.281703 | 0 / 1PASS，CopyThen copy-only首GREEN | `df262fa2135393cf24000d7d2400296bd23752fab7b5d985618381b57f79f262` |
| [control-07](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/control-07/result.json>) | 20132 | 12:29:19.407567–12:29:25.363619 | 0 / 1PASS，原issue/恢复首GREEN | `2f7b87aab2ffb33c28d4ee1d8f93919e2857b07b3533f369b1738f4924b67f96` |
| [control-08](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/control-08/result.json>) | 22892 | 12:30:11.953688–12:30:18.055957 | 0 / 1PASS，Copy write-error优先首GREEN | `22068535bed6ade5c2f4c99c446488540bb802f482793b0930e84b741dc37c8d` |
| [control-09](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/control-09/result.json>) | 9204 | 12:31:10.419586–12:31:19.209247 | 1 / 1FAIL，测试缺少host Frame | `d262e1804194a45abbe2bfe92a2aeabf8a4f8ec6e6fa2cff0756b27f5d02c09f` |
| [control-10](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/control-10/result.json>) | 17068 | 12:32:22.000408–12:32:27.333329 | 0 / 1PASS，渲染后Escape恢复 | `24c9bb99d8f48204d4cb36925b7372e3dfc3bb1c016d22190b80f1946e921a6f` |
| [control-11](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/control-11/result.json>) | 1884 | 12:33:17.944342–12:33:23.590369 | 0 / 1PASS，readonly copy-only首GREEN | `9a71d6a812a9cf58c0c9201d8767bfbaae34a7102e929a2ca62c15ae47357c3b` |
| [focused-union-01](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/focused-union-01/result.json>) | 18816 | 12:35:02.698889–12:35:06.263121 | 0 / 10PASS、0FAIL、0SKIP | `b59d222bfd4450d224f2350f6636b4e2035ca3e0b3839d3823823126cccb4e49` |

- red-01公开前置断言通过：Copy收到起始字面`b`；同文本预编辑后原`abc\r\ndef\r\n`、version、History、非空Selection[1,2)保持，公开MarkedRange=[1,2) active。随后旧cut真实将文本改成`ac\r\ndef\r\n`（测试:30），不是静态推测。
- [第一测试原bytes](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/red-01/first-test.raw>)2566B/SHA `8a210d14c0595cab7d31b826e132c9305af5fe1b9fddc78dd778ebb65171ec93`，完整54行前缀保持；[原editor bytes](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/red-01/original-editor.raw>)26242B/978行/SHA `6e54c2366d8415536cad9783bbf2bc41d7532ce6475794bd52cc90ef09ba5f09`。
- control-01原因：`testing.Fatal arg done is a func value, not called`；仅改诊断参数为`done == nil`，条件不变，保留原失败label。control-09所有错误优先/issue/duplicate断言已过，仅Escape失败：Command已返回后新host issue尚未Frame，Escape shortcut尚未注册；追加公开tester.Frame后恢复，生产未再改。真实产品缺陷RED只有red-01；其余2个失败不冒充第二产品RED。
- 10测试名：`CopySameTextPreeditRetainsOwnership`、`CopyThenDelayedOwnerAndDuplicateDone`、`CopyNormalCutOwnsOneUndoRedo`、`CopyThenNormalCutOwnsOneUndoRedo`、`CopyOnlyKeepsNewOwnerWithoutIssue`、`CopyThenOnlyKeepsNewOwnerWithoutIssue`、`NewIssueRetainsOriginalErrorAndRecovery`、`CopyWriteErrorKeepsPriorityAndPendingIssue`、`CopyThenWriteErrorKeepsPriorityAndPendingIssue`、`ReadOnlyCutRemainsCopyOnly`；均带`TestBuiltinSourceCutComposition`前缀。

## 快照、守卫与停止写入

- [runner](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/run.py>)仅focused/freeze，首次runpy只读取Task40helper定义、不执行main；未执行Task43runner。Bash/Python/Go路径沿原值，Go1.27.3/offline vendor/telemetry-off/CGO0；完整argv、wrapperPID、gatePID、Go exe创建UTC/观察UTC、exit/raw哈希均在每个原result。
- 原1217 [冻结输入](<F:/AIproject/LevNote/.localtools/source-paste-composition-20261114/freeze-01/source.json>)rawSHA `6248b0dcfbfc19cbdbe86abee33ffaba3eec4ca42a21ed2a222a259c5fe56c46`、聚合`42ebbed2ea326603f0d25c3fe0ecc2649d393507daa6b0ba51a7266c67c71174`。全部14次开发run保存1218products before/after/source且inputsEqual=true；原1216其它产品、7pins、534excluded/CON仅metadata不变，八protectedformat/vendor/pins/CI未改。
- 唯一focused union产品聚合`de7753a02378bca509103a693dea528431267790e37e1989f97e2e8ed10f3287`；source.json229032B/rawSHA `8591837886bb93ded5efa38efd151207906be04caf6c45dac7c55eed02e12162`；before/after375796B/rawSHA `70c70d4a0cca10ec7252f38105190ddd88e5b3946e3ce665446a9dcde1583097`。所有stderr为0B/SHA`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`。
- [自审receipt](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/audit-01/result.json>)逐一实算14个result所钉5原流/清单哈希、前后守卫与第一前缀；editor diff读审及diff-check exit0。editor26571B/985行/rawSHA`4e252b48455f8d75d8a3907b3eaa286c0a032d9d4d27dd0dbd48ecfd88d71fac`/canonicalOID`7f509cc307d5c1c5b03aab2cf8dc6288c704013e`；test17642B/rawSHA`ad8e96041698e1f52a6e279a7ebd0457e543b9995ffeb31a2889d19050b7c631`/canonicalOID`4fc9b5b5c1c84026efcb44737684513ff391082e`。
- 最终三file原bytes与canonicalOID以 [freeze scope](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/freeze-01/scope.json>)为准（`hash-object --path=PATH -- PATH`），同处保存完整1218source、before/after、3file.raw与 [freeze result](<F:/AIproject/LevNote/.localtools/source-cut-composition-20261114/freeze-01/result.json>)。本handoff本身SHA/OID由freeze计算，避免自引用循环。
- 精确状态：`STOPPED_WRITING`（editor、新test、新handoff）；`FROZEN_NOT_ACCEPTANCE`。root独立Standards/Spec、唯一affected App/editor/modules、vet/build、精确提交/双main读回全部`PENDING`。只本有限实现task完成，不代表接受。

## 限制与后续

UI-thread同步重入/延迟done及Tester证据不覆盖物理IME/系统剪贴板、并发goroutine。CIinactive、应用NO-GO保持；fresh5PE/realworker/strongcleanup/ownedhost/physicalGUIIMEclipboard/pickers/resources独立PENDING，旧平台/guard/package失败未重复。没有gofmt-write、产品`./...`、旧helper/旧证据修改、GUI/权限重试或commit/push。
