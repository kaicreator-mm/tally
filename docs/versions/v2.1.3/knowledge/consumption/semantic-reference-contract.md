# T36-A/B · 软件工程知识语义引用与关系合同（设计候选）

#626 / #563；输入 main 7f5160b3ec4b29138d4f21e9bcc6ff21459090a3；[R4-G 收尾](https://github.com/kaicreator-mm/tally/issues/624#issuecomment-7176986822)。46篇专业正文已限定接受，R3 历史 S40/D11/U5 与当前派生 S51/D0/U5 各保留。五个 U、未选变体、源资格与真实运行仍未完备。本文只设计文档和引用投影，不冻结 Product/L2、Consumer ABI、Rule 或执行权限。

## 1. 复用现有权威，而不是重新建库

| 已有资产 | 维持职责 | T36 的增量 |
|---|---|---|
| [K01–K11](../../../v2.1.2/software-engineering-knowledge/systematic-knowledge-architecture.md)、[165课程ID](../../../v2.1.2/software-engineering-knowledge/foundation-concept-coverage-ledger.md) | 学科语义和原身份 | 精确指向；原定义及165个ID不变 |
| [48概念关系](../../../v2.1.2/software-engineering-knowledge/cross-module-knowledge-relations.md)、[T23所有权](../T23-semantic-ownership.md) | 既有关系及分工 | 新的消费导航关系仅在其自身范围成立，不重数或覆盖原关系 |
| [T22入口](../casebook/waves/T22-current-knowledge-entry.md) 及其案例/反向索引/来源登记 | 142 Case、143 Blob、164机制/条件关联的原登记 | 定向读取实际案例及原资格，不把参考变为运行事实 |
| [46篇正文入口](../fine-grained/README.md)、[R3台账](../T23-R3-detail-reassessment.md) | 专业深度、适用问题、反例 | 小范围语义定位；不是全部正文已拆成知识单元 |
| [#564原投影](../../abstraction-knowledge-projection.json)、[原L1/L2](../../engineering-intelligence-l1-l2.md) | 16概念、3既有关系、两个DP与一个Oracle入口、历史graph crosswalk、rules=[] | 复用 DP-K02-R19-01、DP-K03-R19-01、K05.ORACLE 及原关联，不改变该投影、既有九维业务设计或 #565 的未决整改 |
| #299 历史 graph | 807节点、106 pending Claims 的原身份（本轮沿用 #564 已登记身份，不重数） | 只有原交叉映射指向参考；RELATED_CONTEXT_NOT_EQUIVALENCE 不是同一概念或 Claim 合格 |

一个确定含义只有一个原正文拥有者。K01 需求/业务规则；K02 架构与权威边界；K03 实现机制；K04 工程工作分解；K05 验证；K06 诊断；K07 协作；K08 制品；K09 演进；K10 运行；K11 经济评价。交叉问题保留各自语义责任。功能族、工程阶段、软件类型、语言、质量属性和读者是**发现视图**，不是平行内容所有者或强制 Workflow。以原 [T23合同](../T23-inventory-contract.md) 为准，不能把某主题所有技术都归一个文件。

## 2. 粒度与身份

选择可独立被引用且能影响一个具体判断的最小语义范围，而不是按 token 长度或段落机械切割。一个具体机制可能需要定义、条件、边界、正反例及来源一起返回；若单段无法支持完整含义，则引用多段或保留整节。学理归属不由检索相关性决定。以下类型是原体系的**引用视角**，不新增平行 ontology：
- Concept/Model：对象、关系、状态、假设和不变量；沿用原课程/概念ID，不能以标题相似认定等价。
- Principle/Mechanism：为什么或在何种条件下成立；区分一般机制与实际库/API/协议版本、具体失败模型。
- Method/DecisionProcedure：怎样比较选择、适用条件和何时停；已有 DP ID 优先，不能把方法建议变成唯一正确实现。
- Case/Counterexample/Oracle：真实局部观察、教学合成、预期与实际观察各分开。原 Source/Locator/Claim/Case/Observation 保留身份和资格。一个反例定义不是已执行测试。
- Topic/组合演练：导航与多机制应用，引用各自正文；不能因同一案例被多个阶段读取而复制知识本体。
- Profile/Rule/Skill/Workflow：项目应用与实际授权、阶段组织；Rule 必须独立有权且可适用，技能文案或知识候选不授予 Tool/Effect。

稳定语义身份与版本化定位分离。已有 ID 继续使用（包括已有 Topic 的 K05.ORACLE）；同一 ID 在不同资料的定位可能不同，不能借局部入口重新定义整个 Topic。对尚无原ID的细节，本批 pilot 只用 `P36-*` 局部导航键，非新增 Concept/Claim/Case ID、非全局持久知识身份；独审后若需全局ID，先查旧课程、Topic、#299 和原 Owner，分别裁定精确同义/细化/新含义，记录映射。不能以新 ID 逃避旧冲突或覆盖旧定义。

每个可消费引用绑定 repo、corpus commit、文件 path、实际 git blob、包含端点的行范围、定位目的及 Owner；行号随正文变更可能漂移，不能只凭同名文件或新路径继承。语义身份可稳定但 locator/适用性/资格分别版本化。改正文时检查影响的消费者和关系，记录实际变更；历史复现仍读旧冻结 SHA。变更原文/版本不自动失效所有相邻知识，也不自动承认新语义。若当前适用版本不能确认，保留 STALE/UNKNOWN；同内容跨 commit 仍须核是否适用于新任务，blob 一致仅说明字节一致。

## 3. 关系类型与推理强度

原48条课程关系保持原方向与含义（例如 CONTEXT_FOR 不是硬前提）。下表是设计时如何读不同关系，未修改原边；pilot 的 relation 只作选知提示，不是新已证明的图谱边。每条候选关系记录 source/target 的实际身份及方向、为何相关、条件、来源定位与资格；反向导航可由索引生成，不能当逆命题。

| 关系语义 | 方向与必要约束 | 消费结果 |
|---|---|---|
| IS_A / PART_OF / REFINED_BY | 分类/组成/细化；确实同一含义或明确父子范围 | 不继承所有规则，不把文档层级当业务依赖 |
| REQUIRES / PREREQUISITE | 本判断→必须确认的前提，明确失败/缺失后果 | 仅真实必要条件阻断受影响判断，不创建 GitHub Task DAG |
| CONTEXT_FOR / SUPPORTS / EXPLAINS | 依既有关系方向和原含义，必要时定向引用 | 可帮助解释，不能当充要条件或传递蕴含 |
| ALTERNATIVE_TO / TRADEOFF_WITH | 同一目标、实际可比较的候选与变化条件 | 对比，不自动选择一边；替代不等价 |
| APPLIES_WHEN / EXCLUDES_WHEN | 指向明确条件或有身份的反例；须核任务事实 | 条件未知不是 false；拒绝候选范围而非拒绝整个任务 |
| DERIVED_FROM / SUPPORTED_BY | 主张→原来源及实际读取范围 | 可追溯，不等于来源真实有效、已授权或 Claim 接受 |
| TESTED_BY / FALSIFIED_BY | 只在明确 Oracle/观察身份及适用范围存在时建立 | Oracle 定义与实际执行分别取证；不能以知识点当已执行测试 |
| CONTRADICTS / SUPERSEDES | 同一范围真实冲突，或有权版本替代 | 不因发布日期或最新检索排序自动解冲突 |
| IMPLEMENTS / CONSUMED_BY / OBSERVED_IN | 真实实现/消费/观察身份及版本（候选设计另标） | 只有实际证据才能宣称已实现/使用，不授权运行 |

本批不增加通用推理规则、自动闭包或 schema validator。pilot 用已有 CONTEXT_FOR/SUPPORTS 表达局部导航（全为 CANDIDATE_NAVIGATION），不修改历史48边；关联条件不是可执行表达式。关系访问预算、访问过的身份/版本及停止理由用于防循环；例如 K01↔K05 互相提供背景，并非循环阻塞。只有经确认的必需前提缺失才阻断该判断，不对所有邻接无条件递归加载。

## 4. 资格正交于内容及任务事实

| 层 | 独立记录 | 不允许的推断 |
|---|---|---|
| 内容 | 文档已接受范围、候选定位、原解释、条件/反例 | ACCEPT_SCOPED 文档不等于外部规范 Claim 合格 |
| 来源 | 原 Source/Claim/Case 资格，读到哪些版本/范围；当前来源是否足够 | 已读官方文档不证明实际系统用了该版本 |
| 任务权威 | 有权 Repo/Issue/合同/Owner、版本、Actor/资源/操作、真实输入与许可 | 引用/模型自报 CONFIRMED 不授予权利 |
| 工程事实 | 实际源码、环境、执行/Artifact、Observation 的真实绑定及观察范围 | 引用测试、代码或工具返回不等于全部合同 PASS |
| 评价 | 独立 Oracle、执行结果、覆盖/缺失/冲突及成本 | 知识被检索/注入不等于被正确采用，不等于效用提升 |

任务事实仍为 YES/NO/UNKNOWN/CONFLICT 等原语义（缺失与不适用独立，按实际合同确认），不建立第二权威账本。输入自报事实或证据链接必须由既有受权边界核对内容及范围；同 observationId 换内容不能继承确认，完整义务集合不能由待评价者删除。现有 #565 R5 的 omitted conflict、FAIL+NOT_RUN、prior contract version 三个已知问题仍 OPEN/HOLD，本设计不声称修正源代码或已有 Validator。

## 5. 本批实际产物及停止边界

[有界定位投影](pilot-references.json) 只为公开纸面选择路径提供指针、阅读目的、材料条件和排除提示；源均为冻结输入的实际定位，所有记录 ADVISORY_REFERENCE、不是 Rule、不是合格 Claim。它不是 DIS/SDK ABI、索引服务或另一套持久图谱。没有复制专业正文、全库语义切片、自动索引或真实消费执行。后续扩展优先根据实际工程缺口逐组查旧定义，不以新单元/边数量评价知识质量。正式生成、接入或实验须另行获权；不变更 #565 HOLD / #286 DEFERRED / A24,T4 NOT_RUN/HOLD。
