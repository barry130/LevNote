# flowix 主题移植批次根验收（2026-11-16）

## 任务
用户要求以 text2future/flowix 项目的 UI 风格完善 LevNote 原生应用界面。

## 实现
- 来源分析：flowix 主题单一事实源 styles/theme/{light,dark}.css（shadcn CSS 变量体系），亮色中性轴 hue 255、暗色 hue 265；亮色品牌浅紫、暗色品牌金黄 oklch(0.816 0.142 91.7)；圆角 8px。
- 移植：native/app/workbench_theme.go 重写为 flowix 设计令牌（冷蓝灰中性轴分层表面、紫/金双品牌锚点、Radius 8、FocusRing 2+1）。OKLCH→sRGB 经 OKLab 转换并按 AA 自适应：浅色文本类色值对最亮面 #e6e9ec 降 L，暗色文本类色值对最亮面 #31353e 升 L，语义色全部由对比度约束收敛。
- 终值：浅色 Accent #4c5ad3（Hover #3f4bc2/Pressed #363eb5）、Danger #ba393f、Warning #8c5f0e、Success #337538、TextMuted #64686d；暗色 Danger #ff734f（其余暗色语义色维持 flowix 原值且已达标）。

## 验证证据
- 对比度测试：native/app/workbench_theme_test.go 全绿（文本/静音/焦点/危险/警告/成功在九个表面上均 ≥4.5）。
- 六根正式门禁：test ./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/... exit0；vet exit0。
- 批次校验器：test-02 exit0、vet-02 exit0；final-audit-02：removed=0/missing=0/regressed=0/newSkips=0，liveTestInputsEqual=true，changedProductFiles 22（21 个 drawio 批次 + workbench_theme.go），3424 旧测试身份零退化、199 新增。
- 截图：.localtools/flowix-restyle-20261116/images/ 共 11 张（深浅 × 源码/阅读/欢迎/窄窗 + 状态栏悬停/按下），人工检视确认浅色紫锚点、深色金锚点、冷灰分层与 8px 圆角。

## 交付
- ZIP：native/downloads/LevNote-Flowix-preview-20261116.5-windows-x64.zip，11,744,105 字节，SHA256 c365e312858b5d8c6cff4318c515acee2789394dc52e9fb164665da33c4607a8；版本 native-flowix.20261116.5-preview。
- 五 GUI PE、26 文件、八引擎路径、全部清单哈希与构建/测试输入同源（436 项）由 .localtools/flowix-restyle-preview-final-verify-20261116.py 独立核验 PASS（回执 .localtools/flowix-restyle-preview-final-verify-20261116.json）。
- 预览构建：未做 Windows 物理输入法/剪贴板/正常关闭的完整发行验收；正式接受包仍为 native-worker.20261110.29。

## 修复史
- 首跑对比度 FAIL（浅色语义色按白底自适应，在 pressed #e6e9ec 上仅 2.92-3.82）→ 浅色按最亮面降 L 收敛；暗色 Danger 按 #31353e 升 L 至 #ff734f。
- 截图采集两次失败：MSYS $PWD（/f/...）被 Windows 进程当盘符；修正为硬编码 F:/ 路径后采集成功。
- 证据链首轮误用不存在的脚本名 drawio-multiselect-20261115.py（真名 ...-validate-...py）；audit-02 目录被误跑占位，残留移位为 -02-stray-1340 后重跑通过。
- 构建脚本变体首轮 zip 自检失败：字符串替换漏改自检环内第二处包目录名；修复后重跑成功。

## 边界
- 未提交推送；.3/.4 体验包与全部历史证据不可变保留；主题仅改设计令牌，无行为变化。
