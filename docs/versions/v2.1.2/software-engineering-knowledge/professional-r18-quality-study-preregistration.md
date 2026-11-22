# R18 D06 · 软件工程知识质量增益的前瞻性对照实验公开预注册协议

> **`PUBLIC_PROTOCOL_CANDIDATE / D06_INDEPENDENT_REVIEW_PENDING / NOT_AN_EXPERIMENT_RESULT`** · [总控 Task DAG #551](https://github.com/kaicreator-mm/tally/issues/551) · [独立评价操作量表](professional-r18-quality-evaluator-rubric.md) · [已审 R18 风险选知](professional-quality-risk-knowledge-selection-r18.md) · [R10 通用知识评价](professional-knowledge-evaluation-protocol-r10.md)。
>
> 本文**只规定中立实验设计与证据**，不含未来候选项目代码、真实隐藏评分参考、冻结维护需求、QA 源码或答案。公开全文对所有组同等可见；只有受独立审核的 D07/G2 私有包才可以承载隔离的实例级材料。G1：PR #552 在 `bd371db6e5edd0bbef896fdd937587cf487e7b70` 独立 `ACCEPT_SCOPED` 后合并到 `main@8b1dc61ef398b7db1fc0818e460a87f4b141ce8f`。G1 是文档范围，不是生产知识或本实验的 QA/Agent PASS。

## 1. 研究问题、对照与禁止暗示

**Primary H0**：相同已发布 ADS、项目事实、质量目标、工具/权限/预算和模型条件下，追加经过 G1 审核的适用 K02/K03/K05/K09/K11 知识，不改善首次 L2 架构缺陷、陌生读者理解正确性或陌生维护者安全改动能力，或者额外成本不合理。

**Primary H1（仅候选，绝不预设成立）**：适时给出任务风险匹配的最小 Kxx 机制与反例，在不降低功能/隔离正确性的情况下，能在三个预先声明的质量终点至少一部分产生可观察的改善。**不设“知识组一定赢”**，亦不以名义 Review 分、代码长度、答题长度或多建 Package 数作替代终点。

- **A / BASELINE**：相同受权 Task、公开业务规则和质量属性、源代码、已发布 ADS、质量目标、时间/工具/上下文限制；不向 A 单独隐去设计要求或真实已有工程规范。不刻意令 A 难完成。
- **B / SUPPLEMENTED**：同一份 A 可获材料和规范、同模型/工具/权限/预算，**仅追加**先验固定的 R18 Kxx 最小适用知识片段，输入包带有与 A 对称的阶段封套；B 不接收未来评分答案/维护请求/隐藏测试/实验者定向提示。
- `KNOWLEDGE_DELIVERED`、`KNOWLEDGE_APPLIED`、`DESIGN_CHANGED`、`QUALITY_IMPROVED` 分别记录，前者绝不可自动推导后者。若工程事实缺失，两组均可 `ASK/UNKNOWN/STOP`，不得要求 A/B 猜测以制造分数差异。
- 不在本实验修改 ADS 规则、Tally Runtime、DomainHarness ABI/Domain Package 或生产 Source Claim。实验产生的是研究材料。

## 2. 抽样结构、明确预选与运行顺序

| 家族 | 公共任务类别（不是题目或答案） | 第一层观察 | 后续实施和维护 |
| --- | --- | --- | --- |
| **F1** | 模块/Package 边界、状态归属与扩展性架构 | 首次 L2 真实设计备选、职责/依赖图、可证伪质量场景 | **是**：进入预先选定的 S5 + 陌生维护者增量改动 |
| **F2** | 并发/持久化/故障可观察性的架构设计 | 首次 L2 的事实边界、原子性/错误归属、设计风险 | **否**：仅首次 L2；不得根据好坏事后晋升 S5 |
| **F3** | 既有代码的保留行为重构与 API 演进 | 首次 L2 的变化归属、消费者契约与演进风险 | **是**：进入预先选定的 S5 + 陌生维护者增量改动 |

**预注册样本数**：3 家族 × 2 独立重复 × 2 组 = **12 个首次冻结 L2**。预选 F1/F3 两家族进入实现和维护：2 × 2 × 2 = **8 个首次冻结 S5 和 8 个独立维护路径**（不许按 S2 成绩挑选“最佳”）。若 G2 前成本评估表明 F1/F3 某类不可执行，必须在启动任何候选前提出新版本预注册、独立审定并明确标旧版未运行；不得在看到结果后改分母。

**随机化/配对**：每家族两个重复，至少一对 `A→B`、另一对 `B→A`，顺序在 D07 中以不可事后编辑的 manifest 精确冻结；同一重复两组从同一个 H0/受权需求/Source SHA 派生，而不同重复应从相同代码快照、不同独立对话/Seed 出发；不共享对方日志、生成代码或 QA。全部 Attempt、超时、预算终止、模型拒绝和修复原样保留。**n=2/家族/组只能提供探索性观察**，不能以显著性、普适质量提升或固定 Token 收益宣传。

## 3. 有序证据冻结，禁止只看最终实现

`G1 knowledge exact corpus` → `D06 neutral public measurement rules accepted` → `D07 private case/task/hidden maintenance/Oracle and stage-manifest prospective freeze` → `G2 independent case validity and isolation review` → `S1 common requirements` → `S2 FIRST_L2_FREEZE` → `S4 implementation` → `S5 FIRST_S5_FREEZE and initial correctness` → `unfamiliar Reader + unrelated independent architecture review` → `pre-frozen unseen maintenance request to independent maintainer` → `maintained code and old/new Oracle` → `independent replay/adjudication`.

- **S2 freeze** 必须记录 `FIRST_L2_SHA`（或内容 digest）、原始 ADR/边界/异常/替代方案、提交和时间。代码出现前冻结。S4 发现和修复架构缺陷是可贵的后续事实，但**不能覆盖首次 L2**。无产出的 Attempt 记 `NO_SUBMISSION`，不得事后改起点。
- **S5 freeze** 必须保留首次实现 Git SHA、真实 typecheck/build/test discovery 和退出码。原始功能门槛/合同兼容、权限与隔离错误单列，不因 Reviewer 喜欢结构而抵消。若首次 S5 未满足硬功能底线，维护阶段记录 `MAINTENANCE_NOT_ELIGIBLE_OR_FAILURE` 并保留整体失败分母，**不偷偷排除或无限修复**。
- **理解评价**由不参与 Builder 的 Reader 对冻结 S5 执行相同知识边界、同样时间与工具的行为预测/符号导航任务，记录正确率、误读、耗时和必要提示。实现形式不同，题目应由既定“责任意图/公共接口行为”而非对某组私有文件名产生。
- **维护评价**在所有 S5 原始产物冻结后才解封预先封存的增量需求，独立陌生维护者按同一 Provider/工具/预算和同一旧/新行为合同进行。原 Builder 不看隐藏维护题，不参与修改；每个维护分支从真实原始 S5 SHA 创建。
- **任务有权限边界**：任何外部不可逆 Effect/真实凭据/敏感生产资料禁止进入试验；只能使用隔离本地 Mock/测试数据库/虚构项目。Mock 通过不升级为真实 Provider/Host E2E。

## 4. 正式评价结果及可比较性

**共同硬门槛**：真实公开要求、权限/租户隔离/数据安全、旧消费者保留行为（适用时），并真实发现和执行规定的正负与边界测试；失败或严重违反在报告中单列 `HARD_FAILURE`，不能以漂亮架构或低 Token 抵消。

**三个 Primary 终点**：
1. `P_ARCH_FIRST_L2`：每份 FIRST L2 **可独立证据支撑的** `MATERIAL` 架构违约（主要项）、`NONMATERIAL` 风险、`OVERDESIGN` 和明确的 `UNKNOWN`。Reviewer 判断必须附任务/质量场景、设计位置、反例和真实替代方案；不是质量形容词/ADR 字数。
2. `P_READ_COMPREHENSION`：陌生 Reader 在有界时间内对指定行为和代码位置的正确判断、偏差与提示成本；与代码审查发现的抽象不当、隐式状态/副作用分别报告，不把 LOC/函数长度/注释数变成质量分数。
3. `P_MAINTENANCE_CHANGE`：独立维护者在 **8 个预选 S5** 上完成预冻结增量需求且保留旧功能的正确率/回归/严重度，并列出阅读成本、真实变更影响、修复/审查成本。对不合格 S5 的维护缺失，需记录原因且纳入分析，不能仅分析维护成功者。

**Secondary**：首次 S5 功能 QA 实际通过率、架构后修复次数、测试与变异捕获、知识实际使用/误用、总模型输入输出 Token（区分 Provider 估算与账单）、人工时间、运行时间、工具/CI 成本、失败及超时。单次评审不能给未来维护效果作“已验证”预言；三类 Primary 各自独立报告，**不合成能互相抵消的综合赢家分**。

**拒绝给出明确胜负时的标准**：主要结果全部相同、Reviewer 置信/一致性不足、QA 无法发现关键变异、两组使用不同业务事实/模型/资源、隐藏材料泄漏或太多无法比较任务，结论写 `INCONCLUSIVE / NOT_COMPARABLE / NOT_TESTED` 并给精确原因。

## 5. 干扰变量、盲评与公平性

- 主导干扰：任务难度、模型随机性、A/B 执行顺序、同一协调者制定/评估 QA、读者/维护者经历与不同 IDE/缓存、B 额外 Token 量、公开教材例子与隐藏题同构、基础合同是否完整。D07 必须记录每一项控制/未控制的事实，不虚报完美因果识别。
- **统一评价规则**：两组都知道同样的业务合同与质量目标；编码者可读共同公开质量要求，不能只向 B 告知“高可维护性”；若 B 得到知识片段，A 也收到同结构的阶段封套（但不包含 B 的实际 Kxx）以缓解提示形式差异。额外知识本身造成 Token/上下文差异属于 treatment，不可伪称输入字节完全一致。
- **角色隔离**：原 Builder ≠ 独立架构 Reviewer ≠ Reader/维护者 ≠ QA 原作者；两名不抄对方判断的 Reviewer 最先形成发现，再预指定独立仲裁。不能通过换 Agent 昵称伪造不同主体，实际对话/文件/凭据边界须有证据。
- **盲性限定**：只要源码命名/设计痕迹揭示知识组、Reviewer 知道答案/预期或者同一协调者掌控所有评分，需分别报告 `MASKED_ID_ONLY`、`NOT_BLIND` 或 `NOT_INDEPENDENT`，而非一概“双盲”。
- **Oracle 可检测性**：在原始 Candidate 运行前用未实现/故障版负对照、具真实行为的阳性参照、刻意坏掉的关键变异，检查测试本身是否能发现设计差异；对不稳定/幸存变异写 `ORACLE_LIMITATION`。不把 R4 后置 QA v2 当 R4 原始成功资格。
- **公平成本**：不应只统计成功和最后一次 Attempt；额外知识检索与 QA Reviewer 费用、失败/修复/超时、陌生维护者阅读时间都单列，Token/人时/费用量纲不混加。

## 6. D06 提交物与 D07/G2 的严格隔离

本公开协议 + [评价操作量表](professional-r18-quality-evaluator-rubric.md) 是 D06 的**候选源码**，必须走另外一个对话非 Writer 的限量独立审核，检查任务类别是否公平、F1/F3 预选是否影响结论、评价盲性/Oracle 是否足够、缺失结果如何入分母、没有把知识文本与评价答案混为一谈。批准后冻结 **D06 exact Git Blob**，无需改变既有 236 篇知识正文。Reviewer 可明确指出偏差并要求 `REWORK`。

**D07/G2 必须后建、独立封存的私有数据（不放公开 docs/PR）**：
`CASE_F1/F2/F3_REQUEST_AND_H0`；`AUTHORITY_AND_PUBLIC_QUALITY_SCENARIOS`；`ASSIGNMENT_ORDER_AND_PROVIDER_PROFILE`；`SELECTED_Kxx_EXCERPT_BLOB_AND_DELIVERY_SCHEDULE`；`QA_SOURCE_AND_FAULT_MUTANT_SET`；`BLIND_REVIEW_REFERENCE_AND_ADJUDICATION_KEYS`；`READER_COMPREHENSION_EXPECTATIONS`；`SEALED_POST_S5_MAINTENANCE_CHANGE_AND_ORACLE`；`BUDGET_STOP`；`ACCESS_CONTROL_AND_ACTOR_MATRIX`；`SHA256_MANIFEST`；`G2_INDEPENDENT_REVIEW_RECEIPT`.

这些内容的 **公开 GitHub Issue 可以只记录文件名、完整 SHA256、范围/审查状态和权限边界**；未经认证的私有 ZIP、公开 issue 附件、仅文件名写“reviewer only”都不等于有效权限隔离。G2 审查必须由非 Writer 在实验开始前用真实包字节确认，发现泄露或不稳定 Oracle 需重冻全部相关材料且保留 superseded 历史。

本版不启动任何 A/B Agent、不运行 CI、不给出实验成效。不把 R18 项目误标为 Tally v2.1.2 Runtime 发布或 DomainHarness Package 激活。
