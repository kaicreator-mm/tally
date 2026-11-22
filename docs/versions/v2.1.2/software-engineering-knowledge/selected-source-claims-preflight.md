# 首批有界知识来源核查与 Agent 消费前置（K05/K08/K07）

> 版本：Tally v2.1.2 [#521](https://github.com/kaicreator-mm/tally/issues/521) · 当前主体 [Draft PR #533](https://github.com/kaicreator-mm/tally/pull/533) · 未来消费者 [#286](https://github.com/kaicreator-mm/tally/issues/286)（仍为 **DEFERRED / NOT_FROZEN**）。
>
> **STATUS: SELECTED_SOURCE_ANCHOR_CHECKED / CLAIM_QUALIFICATION_PENDING / RUNTIME_CONSUMER_NOT_CONNECTED / HOST_NOT_RUN**。这是四条**优先审查的候选知识主张**，不是四条已经接受的 Rule、不是 SDK/Runtime 能力，也不是证明真实 Agent 使用了这些知识。已发布 ADS 基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。源文件按固定提交读取了对应实际条款；[Source catalog](external-source-catalog.md) 中的 47 项资料仍属更广泛的候选参考。

## 为什么只选择四个主张

K01–K11 已有 56 条问题路由和 26 篇深度 Topic。生产前不应要求把所有知识全文导入 Agent，也不能把所有 Topic 一次性重新验证。首批选择影响真实研发结果、容易产生假阳性、可由既有 ADS 和 GitHub 源证实且不用开发新的引擎的四个判定：**测试是否真的执行、公开集成是否真被调用、CI/制品的真实状态、Agent 迟到结果的有效性**。

## 1. 四个 Source→Claim→Counterexample 候选

### C-K05-TEST-DISCOVERY · [K05.FALSE_PASS](K05/topics/FALSE_PASS.md)

**核心主张（原创总结）**：当验证者声称一组材料性测试覆盖某个合同/恢复场景时，绿色测试命令和测试文件存在不等于目标用例**被发现、执行**。应核对 runner 的实际收集/运行结果，并把遗漏部分保留为 `NOT_RUN`；不按用例数设置统一阈值。

**实际发布规范定位：**
- [ADS TESTING_STANDARD.md §6.1, L64–68](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/TESTING_STANDARD.md#L64-L68)，文字明确使用 `SHOULD`，且条款标题仍含 `(v4.19 candidate)`。本知识将其作为**发布文件中的有界建议/候选条款**，不改写为全项目无条件 `MUST`。
- [ADS VALIDATION_STANDARD.md §1, L3–15](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/VALIDATION_STANDARD.md#L3-L15)，`PASS` 要求实际执行，未执行为 `NOT_RUN`，这是实际 Validation Owner 的现有状态规则。
- [ADS VALIDATION_STANDARD.md §6, L83–109](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/VALIDATION_STANDARD.md#L83-L109)：测试收据绑定真实 SHA，不能把旧 HEAD 的证据继承到新 HEAD。

**有界场景/负例**：已声明合同测试文件 `new-contract.test.mjs`，当前 Runner glob 忽略该路径，`node --test` exit 0 且目标用例收集为 0。正判断为**已运行步骤的实际状态 + 此材料目标的 NOT_RUN**，不能说 E2E/合同已证明。修复测试选择后只重跑受影响范围，原先通过的无关检查可按其适用来源保留。

**消费者输出**：`SOURCE_SHA / RUNNER / EXPECTED_SUITE / DISCOVERED_EXECUTED / OMITTED_SCOPE / VALIDATION_OWNER`。这是阅读时的字段提示，不是新增机器 Schema/Gate。K05 判断 Oracle/测试充分性；K08 读取真实 Provider step/Artifact。

**未合格边界**：上述规范段落已定位，但本次尚无独立 Claim 审核或真实 Tally Agent 选用/纠错实验。

### C-K05-PUBLIC-PATH · [K05.INTEGRATION](K05/topics/INTEGRATION.md)

**核心主张**：模块编译/导入/Mock 测试的成功只能证明各自实际执行的范围；涉及公开 SDK/插件/组件组合时，还必须核实对应公开消费者是否**真的选择、调用并观察到行为**，不能用“代码存在”代替“能力可用”。

**实际发布规范定位：**
- [ADS TESTING_STANDARD.md §6.2, L70–74](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/TESTING_STANDARD.md#L70-L74)：对**材料性公开集成**建议一个干净消费者的真实调用/响应和高影响负例，明确禁止 Mock-only/Loaded-only 被宣传为所需集成合格。条款也标为 `(v4.19 candidate)`，其建议与禁止误判应分别准确保留。
- [ADS IMPLEMENTATION_QUALITY_STANDARD.md §5, L37–45](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/IMPLEMENTATION_QUALITY_STANDARD.md#L37-L45)：外部可见接口及错误行为应明确，`SOURCE_PRESENT / EXPORTED_OR_LOADED` 是说明性证据类别，公开能力需要实际 selected-path invocation；不是新的 Workflow 状态。

**可证伪输入**：API 契约和真实 DB integration 通过，但桌面 UI 仍绑定旧端点。需要交付真实桌面 UI 时，最终 Journey 应失败/未测，而不是凭前两层绿色测试直接 `PASS`；如果当前任务仅变更一个 API 字段，则不自动要求完整桌面 E2E。

**消费者输出**：`TARGET_CONSUMER / PUBLIC_ENTRYPOINT / ACTUAL_SELECTED_PATH / OBSERVED_BEHAVIOR / UNTESTED_BOUNDARY`，标注哪些是真实运行结果。公开接口归 K02，测试 Oracle 归 K05，实际 CI/Host/制品归 K08。

### C-K08-STAGE-EVIDENCE · [K08.PIPELINE_STAGES](K08/topics/PIPELINE_STAGES.md)

**核心主张**：Dependency install、Build/Typecheck、测试、Packaging、用户首次安装/配置、Release 是不同的已执行主体与范围；早期步骤 PASS 不得自动晋升未执行阶段，Release 也不能因 PR Merge 而推定完成。

**实际发布规范定位：**
- [ADS CI_EXECUTION_STANDARD.md §10/10a, L193–249](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/CI_EXECUTION_STANDARD.md#L193-L249)：推荐廉价检查在前、Local-first 已等价的验证证据可复用，但真正有独立 Provider/OS/Host 要求时要补相应执行。属于**受条件约束的执行策略**，不是所有项目必须以同样顺序运行的命令模板。
- [ADS VALIDATION_STANDARD.md §1, L3–15](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/VALIDATION_STANDARD.md#L3-L15)：真实 PASS/FAIL/BLOCKED/NOT_RUN/NOT_APPLICABLE 的有效条件。
- [ADS RELEASE_STANDARD.md §6, L102–119](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/RELEASE_STANDARD.md#L102-L119)：发布 READY/CONDITIONAL/BLOCKED/FAIL 的真实决策属于现有 Release Owner；不得把 Component 验证跨层升级为 Release Verdict。

**可证伪输入**：`npm ci` SUCCESS，`tsc` TS2506 FAIL，后续 tests/package 没被执行。输出 `INSTALL_PASS/BUILD_FAIL/TEST_NOT_RUN/PACKAGE_NOT_RUN` 是**证据解释**而非新 Gate；不能报告“CI 全通过”。如果已经有有效同主体独立性/环境等价收据，不为徽章重复 CI。

**消费者输出**：阶段、已运行命令、Source/Config/Provider/Target、首个失败、实际 Artifact Identity、缺失的用户路径。K08 记录来源，K06 诊断首个源码失败，K05 判断行为测试充分性。

### C-K07-STALE-ATTEMPT · [K07.STALE_RESULT](K07/topics/STALE_RESULT.md)

**核心主张**：Agent 远程消息 `ACK/SUCCESS` 与 GitHub 原生 Task/Claim/Review/Release 事实不是同一事件；返回结果来自旧 `SOURCE_SHA` 或已过期 Attempt 时，不能凭 ACK 把当前 HEAD 或验证状态改为 PASS。

**实际发布规范定位：**
- [ADS GITHUB_AGENT_INTERACTION_PROTOCOL.md §8.1, L268–319](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/GITHUB_AGENT_INTERACTION_PROTOCOL.md#L268-L319)：Agent 意图/ACK 只是传输输入，合法的 canonical event/state mutation 需要符合既有 admission contract；工具 ACK 不是 Gate PASS。
- [ADS GITHUB_AGENT_INTERACTION_PROTOCOL.md §5, L178–218](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/GITHUB_AGENT_INTERACTION_PROTOCOL.md#L178-L218)：`transport_actor` 与逻辑 `operator_id/session_ref` 不是同一身份来源。
- [ADS VALIDATION_STANDARD.md §6, L83–109](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/VALIDATION_STANDARD.md#L83-L109)：请求 HEAD 与现行 PR HEAD 不符时，旧证据不可被安静地转换到新提交。

**可证伪输入**：旧 Worker 在 `aaa111` 工作并返回 `ACK/SUCCESS`，GitHub 当前 PR HEAD 为 `bbb222`。输出旧 Artifact/Attempt `STALE`，保持当前主体状态不变；如外部副作用未知，依据合法状态回读，不直接把 timeout 当未执行。

**消费者输出**：`TASK_REF / ATTEMPT / SOURCE_SHA / CURRENT_SHA / ARTIFACT_REF / MESSAGE_CLASS / CANONICAL_READBACK`。K07 负责交互/身份语义，K08 负责证据有效性，正式 Gate/Review 仍归已有 Owner。

## 2. 未来真实消费最小试验集（尚未运行）

| 测试情景（合成输入） | 应命中的候选 Topic | 正确回答至少要做到 | 失败模式 |
| --- | --- | --- | --- |
| CI 退出码 0，材料性目标 suite 被 Runner 排除 | K05.FALSE_PASS | 目标行为 `NOT_RUN`，提出选中测试最小补跑 | `TARGET_SUITE_PASS` |
| Mock 存在且模块被 Import，但实际 UI 指向旧 Endpoint | K05.INTEGRATION | 区分 Mock/Contract/真实用户路径 | `E2E_PASS` |
| `npm ci` 通过，tsc 失败，后续测试未运行 | K08.PIPELINE_STAGES | 分阶段报告和首个失败；交 K06 | `CI_SUCCESS` |
| 旧 HEAD Worker 返回 ACK，新 HEAD 已变化 | K07.STALE_RESULT | ACK 不改当前 canonical Task，保留旧结果 | `CURRENT_TASK_PASS` |
| 纯文档调整，无行为改变，也没有独立 Provider 要求 | K08.PIPELINE_STAGES + K11.ASSURANCE_FLOOR | 按真实要求选最少必要文档检查、不增重复 CI | 强制全量 CI，或虚报未运行检查 |

**实际 Agent A/B 评估另行进行**：未来合法消费者可用相同真实任务/版本/工具权限和同等预算对照无知识辅助与按 Topic 选知识的 Agent，使用独立评审、真实源码测试、用户结果及成本。当前资料只确定**预期判定/反例**，不声称已经执行这些测试，也不批准或要求 #286 的软件 Runtime 变更。

## 3. 资格推进的最小前置

1. 确认本文件及引用 Topic 的当前精确 SHA、ADS 有效来源、实际 Clause 适用条件与版权；不批量激活剩余 22 篇 Topic 或 47 项外部资料。
2. 由有权 Review/Knowledge Source Owner 对选定 Claim 的来源真值、反例和适用范围作真实独立审查，保留不一致/UNKNOWN；未完成时只叫 `SOURCE_ANCHOR_CHECKED`。
3. 由 **#286 后续真正获批的编号 Product/L2/Task** 决定是否在已有 Tally Engineering Business/Domain Package 接口选用这一小片知识；目前 #286 的最新声明仍为 `DEFERRED/NOT_FROZEN/NO_SOURCE_DISPATCH`，不得以本文件自动打开代码实施/Claim 权限。
4. 只有在真实消费和所需验证完成之后，才能评估可重复的工程问题选择效果；所有 `PASS/REWORK/NOT_RUN` 归各自原授权主体。

**不做：** 新知识编译器、知识生产平台、第二套权限/验证流程、全量 26 Topic 独立反复 Review、无关业务源码修改、运行期假成功声明或新安全/逆向专项。
