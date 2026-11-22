# R18 D06 · 软件架构、代码理解与维护能力的中立评价操作量表

> `PUBLIC_RUBRIC_CANDIDATE / D06_INDEPENDENT_REVIEW_PENDING / NO_QA_EXECUTED` · [公开预注册实验协议](professional-r18-quality-study-preregistration.md) · [K02 架构场景](K02/advanced-scenario-driven-architecture-quality-r18.md) · [K03 代码理解](K03/advanced-code-comprehension-and-abstraction-quality-r18.md) · [K05 独立评审](K05/advanced-quality-reviewer-calibration-r18.md) · [K09 后续维护](K09/advanced-stranger-maintainer-change-assay-r18.md) · [K11 总成本](K11/advanced-total-quality-cost-and-maintenance-economics-r18.md) · [#551](https://github.com/kaicreator-mm/tally/issues/551)。
>
> 此量表是给**独立实验评估方**使用的通用判断原则，不是提供隐藏任务的评分答案。不应奖励 Kxx 教材用语、ADR 模板字段数、更多模块/测试/注释。必须在 **D07 冻结实例级真实验收来源与反例**后才能审真实候选。公开协议供所有组同等可见。

## 1. 统一输入与不可跳过的事实门槛

Reviewer 读取：匿名 `Case_ID`、已授权实际需求与质量属性场景、每个 Candidate 的 `FIRST_L2_SHA`、`FIRST_S5_SHA`、可用源码、真实环境、测试发现+执行日志及 `NOT_TESTED`；维护 Reviewer 另需前置冻结的旧行为和新增维护要求。Reviewer 不得读候选 Prompt 的知识组别、Agent 与协调者评价、自报“设计 PASS”或对方审查结果。

任何重要判断必须给 **`REQUIREMENT_OR_QUALITY_SCENARIO → CODE_OR_DESIGN_LOCATION → MECHANISM → FALSIFIER_OR_TEST → FINDING_CLASS`**。无明确权威来源时应说明 `EXPECTED_UNKNOWN`，不因 Reviewer 熟悉某架构模式就代替受权要求。

**严重性锚点**（不随组别变化）：

| Class | 证据标准 | 报告 |
| --- | --- | --- |
| `MATERIAL` | 能由已授权的合同/质量属性场景与当前版本设计形成具体违约或危险路径；至少提供可执行的失败刺激或可检验的机制链。不能仅以“不使用我喜欢的架构”成立 | 原始场景 ID、完整触发、受影响状态/接口、位置、预计错误结果、需要的最小证伪 |
| `NONMATERIAL` | 有可信的成本/复杂度/变更传播风险，但当前有权需求未被证明失败 | 风险、备选、何种未来/当前事实会升级为 MATERIAL |
| `OVERDESIGN` | 有实际不必要间接层/模式/可配置性，缺少当前有权需求收益，并造成明显理解/变更/验证成本 | 真实导航或维护代价、删除候选、替代成本；不可仅计类/层数 |
| `STYLE_ONLY` | 合同允许的命名/布局风格差异，没有可展示的误解/行为/维护负担 | 仅旁注，不作为 Primary 缺陷 |
| `UNKNOWN` | 需要的业务权限、质量阈值、源码或真实执行证据缺失 | 缺失事实与最小核查，避免虚构 BAD/PASS |
| `NOT_TESTED` | 宣称的性质没有被当前固定 Oracle/环境/时间窗触及 | 具体未测要求/环境，不合成通过 |
| `NOT_COMPARABLE` | A/B 不是同一公开输入/工具/预算、QA 失效、内容泄漏或环境不匹配 | 不计算该指标的组间增益，但记录失败样本与原因 |

`MATERIAL` 项必须由两位 Reviewer 各自指出/裁定；多数投票不能变出原本不存在的事实。`MATERIAL` 与 `NONMATERIAL` 不应在没有新证据时互换。

## 2. 首次 L2 架构评价（Primary A）

### 判定次序

1. **Quality scenario**：受权的刺激/环境/受影响对象/响应/观察量是否齐全；不存在有效阈值时应保持 UNKNOWN，不为“方案很好”造指标。
2. **真实状态/职责/依赖**：每个具有权威事实的模块的读写 Owner 是谁？有没有隐藏共享写者、循环/越权依赖、半提交、错误责任漂移？
3. **契约与演进影响**：接口错误/隔离/兼容约束是否由现有来源支撑？新增变化会触及不相关模块是实际证据还是猜测？
4. **备选与反例**：存在更简单但同样满足质量场景的选择时，是否说明为什么没有选？抽象多不加分；必要隔离也不能因文件多扣分。
5. **首次性**：只读 `FIRST_L2` 原始提交。S4 改进/修复单独记 `POST_L2_CHANGE`，不能追改原评分。

**原始报告**：`FIRST_L2_SUBMITTED`、`MATERIAL_FINDINGS`（distinct ID/severity）、`NONMATERIAL`、`OVERDESIGN`、`UNKNOWN`、`QUALITY_SCENARIO_COVERED`、`REVIEWER_DISAGREEMENT`。若架构缺陷不是当前任务质量驱动则不把“潜在未来灾难”当 MATERIAL。

**阳性/阴性校准**：D07 私有参考必须含至少一个被授权质量场景明确触发的结构违约，和一个架构形式不同但满足相同合同的合法替代；Reviewer 应能指出具体失败和合法替代，不应偏爱某模式。校准题与正式隐藏任务不得同构照抄 K02 教学例。

## 3. 代码理解与可读性评价（Primary B）

**Reviewer 的正确性优先于偏好**。针对冻结 S5 的公开接口/行为，使用独立、预注册、与实现无关的理解题：

- `BEHAVIOR_PREDICTION`：给合法/非法输入场景，Reader 说明 API 结果、状态变化和可能错误的真实链路（若缺运行事实写 UNKNOWN）。
- `NAVIGATE_SYMBOL_PATH`：从公开 API 找到真实状态写入/读取或错误决策的符号；允许工具跳转，但记录实际使用。
- `SIDE_EFFECT_BOUNDS`：辨别异步/持久化/外部调用的位置与状态可观察边界，不能猜测未公开 Provider。
- `CHANGE_IMPACT_EXPLANATION`：基于当前源码说明变更会触及的职责、哪些推断仍需测试。

**测量字段**：`QUESTIONS_PRE_FROZEN`、`ANSWERS_CORRECT/INCORRECT/UNKNOWN`、`TIME_TO_FIRST_CORRECT_EXPLANATION`、`CODE_LOCATIONS_VISITED`、`WRONG_INFERENCES`、`HINTS`、`READER_MODEL_PROFILE`、`BLINDNESS`。应说明不答的原因；不能把不知来源却猜对结果当完美理解。

**公平性**：不要拿某候选原函数名作题，另一候选需要阅读五层；问相同**公共职责/实际行为**，若没有跨实现可比映射，标 `ITEM_NOT_COMPARABLE`。同一 Reader 先看 A 再 B 可能携带答案，需盲化/平衡顺序或使用独立 Reader，每个尝试和缓存均记录。

**代码审查补充维度**：可见职责边界、错误信息可行动程度、异常退出路径、资源释放、抽象能否隔离真实变化、代码导航绕行、重复行为差异。代码审查者须给 path/symbol + 真实成本/误解证据；`LOC`、方法长度、圈复杂度、lint/format、注释多少只是辅证，不直接作为赢家分。

## 4. 后续陌生维护者真实增量修改（Primary C）

仅对 D06 已**预先选择**的 F1/F3 两家族/共 8 份首次 S5 进行。冻结维护题在看原始实现前完成，原 Builder 不得看到未来维护要求，维护 Agent 不得参与原 Builder。维护条件/环境/profile 对两组相同。

**最小正确性分级**：

| 结果 | 判定所需观察 |
| --- | --- |
| `MAINTAINED_OLD_AND_NEW_PASS` | 同时真实执行已冻结旧行为回归 + 增量新要求正反 Oracle，按真实 runner 收据接受 |
| `NEW_PASS_OLD_REGRESSION` | 新需求部分达标但已支持旧用户/数据/接口合同被破坏；不能算维护成功 |
| `NEW_FAIL` | 新行为不满足预冻结合同，即使旧合同仍有效也不能算成功 |
| `INITIAL_S5_INELIGIBLE` | 原候选在开始维护前已有严重功能违约，不可以靠仅统计可维护样本隐藏它 |
| `NOT_TESTED/ORACLE_INVALID` | 缺真实测试/环境或 QA 自身缺陷；需修 Oracle 并标后置版本，不能改原始分母 |

**记录变更影响而非文件数竞赛**：`NECESSARY_OWNER_TOUCHES`、`INCIDENTAL_TOUCHES`、`PUBLIC_API_COMPATIBILITY`、`OLD_NEW_QA_RESULTS`、`MAINTAINER_READ_TIME`、`CHANGE_ATTEMPTS`、`REPAIR/REVIEW_TIME`、`TOOL_CALLS`、`MODEL_USAGE`、`HUMAN_HINTS`、`MAINTENANCE_HEAD`。多改一个隔离 Adapter 可有益；把多个责任塞进一个巨大文件虽然触及少仍可能有害，需证据说明。

## 5. 独立性、校准和分歧裁定

- **角色**：Case/Oracle 作者不能是受测代码 Builder 或 Reviewer；至少两名与 Builder 分离、尚未知道 A/B 组别的 Reviewer 在 FIRST_L2 上**分别先提交原始发现**；两个 Reviewer 的发现、未测、否定判断都保留，之后第三位有权裁定者按冻结规则裁决。Reader 与维护者不是原 Builder；若真实上使用同一 Operator 但不同 Agent 名称，应声明 `OPERATOR_SHARED`，不宣称完全独立。
- **校准**：正例/负例/变异与正式案例分离；Reviewer 必须能识别一个确切架构违反和一个不同风格但合法方案；QA 能在空实现负对照 FAIL、可信阳性 PASS、关键破坏 Mutant FAIL。对隐式历史状态或依赖真实运行的判定，只给真实可观察性，不凭后置数据改原先结果。
- **分歧**：`REVIEWER_A_FINDINGS`、`REVIEWER_B_FINDINGS`、`MATCHED_ITEMS`、`CONFLICT_ITEMS`、`ARBITRATION_RATIONALE`、`REMAINING_UNKNOWN`。严重度分歧保留原始证据，`CLEAN` 不代表没有错误，只代表未提出合格发现。
- **盲性**：匿名 ID ≠ 强双盲；看过样本/来源/组别时写 `NOT_BLIND`。评估者不可以知道谁读 Kxx 或曾在原实验中哪组获胜；若设计特征泄漏可能组别，记录 `BLINDNESS_LIMITED_BY_CODE`。
- **反例裁定**：若找到原先未设想的等价合法设计，先在不看组别下裁定，然后用一致方法同时复核 A/B。不能只给一个组改 Expected；Oracle 污染必须留旧 SHA + 明确新版本。

## 6. 样本表、证据完整性与结论语言

每一个 Attempt 用相同 schema 记录，不允许只看成功或漂亮样本：

```text
CASE_ID / FAMILY / REPLICATE / ASSIGNMENT_CODE / H0_SHA
PUBLIC_REQUIREMENT_SHA / ADS_SHA / SELECTED_KNOWLEDGE_MANIFEST_SHA
FIRST_L2_SHA / FIRST_L2_TIME / ARCH_REVIEW_A / ARCH_REVIEW_B / ARBITRATION
FIRST_S5_SHA / ACTUAL_DISCOVERED_TEST_IDS / RUN_RESULTS / HARD_FAILURES
READER_QUESTION_SET_DIGEST / READER_CODE_COMPREHENSION / READ_COST
SEALED_MAINTENANCE_SPEC_DIGEST / MAINTAINER_ID / MAINTENANCE_HEAD
OLD_QA / NEW_QA / MAINTENANCE_COST / TOOL_PROVIDER / BLINDNESS
ORACLE_VERSION / POSITIVE_NEGATIVE_MUTANT_PROBES / NOT_TESTED
KNOWLEDGE_DELIVERY_DIGEST / APPLIED_DECISION_TRACE / COST_UNITS
STATUS / CAUSE_HYPOTHESIS / COMPETING_EXPLANATIONS / EXIT_REASON
```

**报告按 3 个 Primary 分别输出每样本的数据、未测、Review 分歧和环境差异**。不引入单一“代码质量总分”作为赢家。n=2/家族/组不宣称统计显著，不输出放大到 Tally 全项目的通用质量收益百分比；仅在明确实验实际观察范围内写 `OBSERVED_ASSOCIATION`、`PLAUSIBLE_MECHANISM`、`NOT_ESTABLISHED`、`INCONCLUSIVE`。

## 7. D06 独立审核界面 / D07 边界

**可以在公开 PR 中审核**本文定义、未定量的标准及无隐藏题的概念例；**不可以公开** D07 的实例级 CASE、正确答案、负面代码变异、QA 脚本、维护变更、Reviewer keys 或凭据。仅因本量表文档通过审查，仍不能认为实验 Oracle 校准成功。

本量表与 [公共预注册协议](professional-r18-quality-study-preregistration.md) 应由**另一个对话的非 Writer 独立审核者**在真实 Exact HEAD 上复核偏差、权威性、可比性、缺失分母、成本计量、反事实限制和准入边界；`ACCEPT_SCOPED` 后才可冻结 D06 SHA 并另行推进 D07 真实封包。没有 G2 审核通过，**禁止 Agent D08 开始**。本文不构成新 ADS Gate、DomainHarness SDK API、Tally 生产质量评分器或 Source Claim 资格。
