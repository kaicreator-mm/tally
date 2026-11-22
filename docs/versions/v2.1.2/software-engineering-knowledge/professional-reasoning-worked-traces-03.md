# 软件工程领域智能进阶演算第三卷：W13–W17

> [第一卷 W01–W05](professional-reasoning-worked-traces.md) · [第二卷 W06–W12](professional-reasoning-worked-traces-02.md) · [R8 专业学科来源与限定适用关系](professional-source-applicability-r8.md)
>
> **R8 SYNTHETIC_TEACHING_ONLY / FIVE_CASES_NOT_AGENT_EXECUTED / NO_NEW_NATIVE_TASK_OR_HOST_EFFECT。** 五条演算属于原创教学输入，不能被记为真实 Agent 性能评估、正式标准 Claim、真实 CI/E2E、付款或发布资格。仅展示在已知事实下如何从11模块中选择必要学科、提出可证伪判断与合法停止条件。

## W13：两位 Stakeholder 声称不同付款规则，谁决定？

**输入（合成）**：产品手册 v1 允许 PAID/PACKING 普通用户取消；一名客服说最近已经禁用；Owner 起草 Tenant B 从 11 月 1 日起禁止 PACKING 取消但尚未有签发证据；Tenant A 日志显示 11 月 5 日界面“取消成功”。

**推理路由**：[K01 来源/三角核验](K01/advanced-stakeholder-authority-and-triangulated-evidence.md) 分离观测/规范/生效/租户；[K05 Oracle](K05/advanced-verification-validation-and-oracle-qualification.md) 区分 UI 成功提示与服务端 Commit；[K02 边界](K02/advanced-extension-authority-state-and-consumer-boundaries.md) 分离本地状态与远端退款 Effect。

**选项**：A 采用“客服更近”直接修改业务；B 仅根据 UI 日志把远端退款标为完成；C 查询有权签发、Tenant 范围和订单/远端真实状态，再将未知部分 ASK/UNKNOWN。**正确教学选项：C。** A 忽视有权政策与有效区间，B 混淆 UI/财务 Effect。

**反事实与 STOP**：若发现确已签发且仅 Tenant B 生效的规则，可以限定判断该租户相关交易；若只找到 Draft 与客服描述，不能向所有 Tenant 发放统一生产 Rule。本文未查真实付款。

## W14：声明 Extension 就能操作 Host 吗？

**输入（合成）**：Package A 声明只依赖 B，B 依赖 C；开发者让 A 在代码生成中直接用 C，并因静态编译成功而推断 A 能直接调用未经授权的外部 Host 操作。数据库订单 Intent 成功入库，远端 Effect ACK 丢失。

**推理路由**：[K02 的四种边界](K02/advanced-extension-authority-state-and-consumer-boundaries.md) 划分直接依赖/业务权限/状态 Owner/版本合同；[K03](K03/advanced-algorithm-contracts-types-and-formal-claims.md) 分清类型检查与授权；[K10](K10/advanced-reliability-slo-effect-recovery-and-measurement.md) 维护 UNKNOWN Effect。

**方案比较**：让 A 显式声明合法的 C 或通过 B 的允许接口，取决于**实际有权 BUILD 合同**；再另行检查操作 Actor/Scope/HostPort 是否有权；对 ACK 丢失的请求依据 Provider 真实幂等与查询契约，不要自动二次提交。

**失败 Oracle**：静态 BUILD 接受未声明直接依赖（若合同明确禁止，应拒绝）；Host 接受未授权付款；本地 Commit 自动推断远端成功。三项问题互相独立，不能用单一 Test PASS 证明。**停止**：若 ABI/权限未给出，只设计候选测试，不能宣称真实 Kernel/Provider 已实现。

## W15：为什么两个 Agent 和更大的上下文反而可能降低工程价值？

**输入（合成）**：20 个预先分配任务，单 Agent 完成 16 个合格任务、总资源 100；三 Agent 完成 18 个合格任务、总资源 150；某轮研究把所有 203 篇知识文档都加入上下文，却遗漏最新授权的业务 Rule。

**数学比较**：单 Agent 合格率 16/20=80%，合格/资源=0.16；三 Agent 合格率 18/20=90%，合格/资源=0.12。**质量完成比例上升不等于资源效率上升**。如果两组任务难度或人工接管记录不同，连这两个比率都不能当真实因果证据。

**推理路由**：[K07 角色/Context](K07/advanced-agent-role-context-and-outcome-evaluation.md) 要求当前任务充分的最小知识集；[K11 整体工程经济](K11/advanced-decision-risk-value-of-information-and-process-cost.md) 要求 all-assigned-task 真实分母、质量门槛及成本；[K01](K01/advanced-stakeholder-authority-and-triangulated-evidence.md) 要求 Source 权威优先于重复引用次数。

**可证伪实验**：固定任务、源版本和业务 Oracle，比较含正确 Owner Rule 的定向上下文、删除必要 Rule 的上下文以及大量无关材料的上下文；观察正确 ASK/UNKNOWN、合格交付、墙钟、人工复核和资源成本。**当前实验未运行，不能宣称多 Agent 更快或最小 Context 必胜。**

## W16：CI 绿色、Provenance 完整，是否意味着可以正式 Release？

**输入（合成）**：构建 H1 得到制品 D1；CI 单测全部通过；有人手写一份“D1 由 H1 构建”的 JSON provenance。正式 Release 上传的 Installer 为 D2（不同 digest），但没有目标平台首次启动的验收。

**推理路由**：[K08 Release/Provenance/回滚](K08/advanced-release-provenance-permission-and-rollback.md) 将 Source SHA、Builder、Artifact Digest、签名/信任根、Test Discovery、Host 安装与 Release 签发逐层分开；[A68 SLSA](external-source-catalog.md) 是来源结构的研究参考，不是已验证的项目评级。

**反例**：JSON 字段正确不能证明签发 Builder 可信；D1 的 Test PASS 不能自动用于 D2；某个 GitHub Check SUCCESS 不等于正式 Release 权威已执行；回滚旧镜像不能恢复已迁移丢失的数据与外部付款。

**Oracle 候选**：真实下载 D2 校验 digest，追溯受信 Builder/源码材料，核对必要 Check/Reviewer，选定目标 OS 安装、启动后配置和业务能力权限。任何一项不存在，应准确写为 NOT_RUN/NOT_VERIFIED，不能自行制造 PASS。

## W17：为了降低风险，是否应每次都增加 Review/CI？

**输入（合成）**：一个导航链接断裂的概率假设 20%、损失 2 单位，期望损失 0.4；另一个远端金融 Effect 错误概率假设 1%、损失 500，期望损失 5。对后一问题有一次可信来源调查成本 8，假定能将误判概率从 30% 降到 10%，误判成本 100。

**计算结果**：[K11 风险/信息价值](K11/advanced-decision-risk-value-of-information-and-process-cost.md) 在严格合成假设下给出调查前期望误判损失 30、调查后总成本 18、差值 12；只能说明该条件下进一步核实可能值得做，**没有测得真实概率，也不能把未知财务授权用期望成本抵消**。

**选择**：对无材料影响的文档导航，执行必要的局部链接/审查足够，不必重跑所有 Host；对远端资金，必须满足有权 Source、Host/Provider 和必要恢复 Oracle；若真实调查并不能可靠降低错误概率，不能凭上述数字宣称投资回报。

**错误答案**：“二者都必须固定 3 次 CI + 4 位 Reviewer”“都只要模型判 PASS”。正确做法是在合法质量/授权硬约束下比较信息增益与机会成本，同时记住 SLO/Release 的真正 Owner。

## 归纳与合法边界

W13–W17 五个输入均为 synthetic。W01–W12 历史内容和 48 条已接受 Concept 关系不因本卷新增而重新赋予来源/运行权限。研究问题可以复用现有 Kxx 专业学科与真实出版方资料；在 DomainHarness Consumer ABI 未获真实资格、Tally #286 仍 Deferred 的条件下，任何一条完整 W 也不代表真实 Agent 已运行该工作或某个 Task 已被发布。