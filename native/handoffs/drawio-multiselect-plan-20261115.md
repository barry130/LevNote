# draw.io 多选与对齐：当前实施分工

日期2026-11-15。用户持续自主开发指令下的新批次；[.3验收](<F:/AIproject/LevNote/native/handoffs/feature-parity-root-acceptance-20261115.md>)与所有旧包保持原字节。

基线：[429文件清单](<F:/AIproject/LevNote/.localtools/drawio-multiselect-20261115-baseline/inputs.json>)；前版3424测试身份保存在同目录。详细语义采用[已审实施合同](<F:/AIproject/LevNote/native/handoffs/feature-parity-drawio-next-20261115.md>)。

## type:goal 用户可见闭环

Ctrl多选图形，检查器显示选中数量和六种对齐；Shift框选与整组移动随后接入，同批收敛。六对齐使用集合外包框、不吸附；整组移动仅吸附共同位移。一次几何动作一个撤销事务，原有标签草稿Flush保留为独立事务。对齐只改需要变化的x/y，尺寸词法和未知XML原字节保留。

## type:state 唯一写域

- gap-markdown：native/formats/drawio/positions.go、positions_test.go；必要时窄改同目录parse.go/types.go/edit.go与新增position测试。另写handoffs/drawio-multiselect-format-20261115.md。
- gap-graph：native/modules/graphview的drawio_selection/alignment/gesture实现、对应测试，以及adapter/input/draw/sync/view_ui/draft/geometry/edit的最小接线。只设这一名共享view写者。另写handoffs/drawio-multiselect-view-20261115.md。
- gap-editor：仅native/app/drawio_alignment_app_test.go及同前缀新增app测试；另写handoffs/drawio-multiselect-app-20261115.md。无app生产或graphview改动。
- lead：计划、证据包装器、全部Go执行、独审整合、产品文档与包。child不得运行Go测试/vet/build或派生child，不改共享STATUS/README/记忆/旧证据。

## 固定接口与联调

格式层公共接口：PositionUpdate{CellID string; X,Y float64}；Alignment uint8与AlignLeft/AlignCenterX/AlignRight/AlignTop/AlignCenterY/AlignBottom；Document.ValidatePositionTargets(pageIndex int,cellIDs []string) error；Document.PlanAlignment(pageIndex int,cellIDs []string,alignment Alignment)([]PositionUpdate,error)；Document.UpdatePositions(pageIndex int,updates []PositionUpdate) error。

ValidatePositionTargets接受一个以上目标供拖动复用；PlanAlignment要求至少两个唯一有效顶点。格式owner先发布接口冻结，view可按此合同准备。资格从实际XML核对，canonical父容器/绝对叶顶点/合法有限几何，非支持的变换或结构给具体原因，不基于fallback shape猜测。

六动作Key统一drawio.align.left、drawio.align.center-x、drawio.align.right、drawio.align.top、drawio.align.center-y、drawio.align.bottom；Label依次左对齐、水平居中、右对齐、顶对齐、垂直居中、底对齐。多选Count文案“已选择 N 个图形”。app通过公开Host/真实Tester事件验证，不添加生产测试钩子。

## type:progress 验收顺序

先独审格式原子性与XML字节oracle，再收集view/app同源联调；实现期间按小范围看实际失败，最终冻结后一次六根全量+vet。测试身份与.3比较，旧PASS/SKIP不删不弱化；至少四张深浅/窄宽实渲图实际读取。成功后构建新Windows预览并独立验包。

完整合同中的owner/buffer/version/pageEpoch/selectionEpoch/field receipt/回调重入/readonly/IME/growth守卫适用于所有新入口；无操作保持历史与草稿。Ctrl多选时单节点删除/连接/属性写入须明确停用，避免只改primary。切页/换buffer/Release清理选择与捕获动作，迟到事件无写入。复制、分组与批量删除留在后续批次。
