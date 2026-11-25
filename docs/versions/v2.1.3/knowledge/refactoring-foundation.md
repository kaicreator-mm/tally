# 软件工程领域知识 · 面向后续重构的知识基础

> R6 / #563。用户已明确将本阶段改为知识优先：整理软件工程知识，为未来 Tally / DomainHarness 重构奠基；可以只读调用既有研究 Package 验证真实代码行为，但不开发新代码、不依赖当前 SDK 的实现/版本、不展开新 Agent 实验。本文为 v2.1.2 K01–K11 的跨模块知识导航和专业补充，既不是新规范、机器 Schema，也不是重构实施任务或生产授权。
>
> 来源基线：Tally main@c6d2c8404072f38f2bf946a7f77ce0c712621516；原知识在 [v2.1.2 系统知识架构](../../v2.1.2/software-engineering-knowledge/systematic-knowledge-architecture.md)、[课程覆盖台账](../../v2.1.2/software-engineering-knowledge/foundation-concept-coverage-ledger.md)、[概念关系](../../v2.1.2/software-engineering-knowledge/cross-module-knowledge-relations.md)。165 原课程知识点、48 原概念关系、原 Source/Claim 状态均不改变。当前新增的专业推论属于 DOCUMENT_CANDIDATE / ADVISORY，不是新增合格 Claim 或 ADS 强制 Rule。

## 案例库入口

[Software Engineering Casebook](casebook/README.md) 是同一软件工程知识体系的按问题索引视图。[T22 当前统一入口](casebook/waves/T22-current-knowledge-entry.md) 汇总已合入 T00–T21 的 142 个案例、143 个唯一来源文件 Blob、165 个原知识点与164条案例知识关系，并提供按类型/功能/质量约束选择的目录、反向索引及实际资格。T22 已在 [#582 closeout](https://github.com/kaicreator-mm/tally/issues/582#issuecomment-7164517356) 限定接受并合入（历史入口保留当时的候选措辞）；原定义及 Source/Claim 资格不变，源观察、原创分析、运行证据与拟议验证分开。原40条初始案例及后续增量保持历史身份。所有外部项目运行和新综合推演的 Oracle NOT_RUN；不编写新的业务或 DomainHarness 代码。

## 第三批细粒度知识建设（T23-R1 候选）

用户要求持续扩展知识；T15–T22 仅完成第二批限定验收，不是全域建设终止。[盘点合同](T23-inventory-contract.md) · [56组初始细节目录](T23-detail-inventory.md) · [语义归属及重复处理](T23-semantic-ownership.md) · [阅读路径与建设子流](T23-reading-paths-and-waves.md)。基线 383358955d117d155e2b3b2fc4cd7886e77f71b9；[#584](https://github.com/kaicreator-mm/tally/issues/584) 实际有界盘点候选，待独立审核，未冻结最终分类或派发 T24–T36，不把候选组计入既有知识点或案例。旧 T22 入口仍是实际已接受的知识索引，129机制/35条件共164关系不变。

## 1. 稳定知识与易变实现分离

| 层 | 主要内容 | 所有权与变化原因 |
|---|---|---|
| 学科知识 | 概念、模型、原则、机制、假设、适用条件与反例 | K01–K11 原语义 Owner；依据学科与实践证据演进 |
| 决策方法 | 明确问题、已知事实、选择条件、备选方案、判断、UNKNOWN、反证和停止 | 各专业模块拥有；不固定运行时函数、类型或 ABI |
| 项目应用 | 有权需求、实际代码、依赖/资源 Owner、版本、行为合同及质量目标 | 具体项目的当前有权 Source；通用知识不替它创设义务 |
| 验证与经验 | Oracle、正反例、实测观察、局限、反事实、维护成本 | K05/K06/K08/K09/K11 的分工；经验不自动升格为规则 |
| 实施映射 | 未来 Package/Element/Function、具体类型、工具链、Workflow | 仅待重构后的实际规范和项目设计确定；当前不生产代码 |

知识应在不安装 Tally、DomainHarness 或任何 SDK 时仍能阅读和使用。用现有文档体系的 Identity / Meaning / Applicability / Relations / Provenance / Procedure / Falsifier / Qualification 表达知识即可；不建立平行图谱、重复记录定义、固定未来程序接口或统一评分公式。需要实际消费时才选择具体实现形态；普通专业判断同样可由人或 LLM 依现有来源完成。

## 2. 以真实重构问题重新组织 11 个模块的知识导航

下表的原 ID 均来自各模块 systematic-syllabus.md，使用相对链接回原语义 Owner，不是建立第十二个模块或缩小学科覆盖。此视图按重构决策使用顺序组织；K04、K07、K11 为贯穿性专业能力，不是固定阶段。

| Owner / 知识入口 | 关键问题与应掌握的专业机制 | 面向后续重构的产物与反证 |
|---|---|---|
| [K01 需求工程](../../v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md) F2.2/F3.3/F4.1/F4.2/F5.1/F5.3 | 事实/假设/未知、功能合同、业务不变量、版本权威和同条件冲突 | 合同与保留/改变行为清单；不从现有 bug 或旧接口自行生成新需求 |
| [K02 架构与设计](../../v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md) F2.1/F2.2/F2.3/F3.1/F4.1/F4.2/F5.1 | 职责内聚、信息隐藏、真实变化轴、接口/错误、资源/状态 Owner、质量场景 | 备选边界、依赖方向与可观察失败场景；不以目录、类数或所谓统一接口作为充分证据 |
| [K03 实现与构造](../../v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md) F1.3/F2.1/F2.2/F2.3/F4.2/F5.1/F5.3 | 算法/类型边界、语义复用、抽象粒度、资源、错误、可读性 | 内联/纯 helper/模块/窄接口/Strategy 的取舍及精确输入输出；同语法不同语义为反例 |
| [K04 工程过程](../../v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md) F1.3/F2.2/F3.1/F3.2/F4.2/F5.3 | 因果依赖、分解/组合、共享写集、变化影响、风险驱动和停止 | 最小必要任务划分、先基础后依赖；不强制每个小改动走复杂 DAG |
| [K05 质量验证](../../v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md) F1.1/F1.2/F1.3/F2.1/F2.2/F2.3/F4.1/F5.3 | 有权 Oracle、正反边界、实际执行、范围和证据聚合 | 保留合同与授权变化的独立观察；测试全绿不代表全部义务被验证 |
| [K06 调试与修复](../../v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md) F1.1/F1.2/F1.3/F2.1/F2.2/F4.1/F4.2/F4.3 | 现象与根因分离、第一失败点、版本身份、最小复现和反事实 | 当前 exact source 的可复现反例，区分设计缺陷、实现缺陷与环境缺陷；不凭症状决定大重写 |
| [K07 协作与 Agent](../../v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md) F1.3/F2.2/F3.1/F3.3/F4.1/F5.1 | 权责、会话/版本、消息与事实、知识选择、独立写集和可证伪审核 | 区分建议、授权、执行证据；不把模型多次声明当独立实测 |
| [K08 构建与交付](../../v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md) F1.1/F1.2/F2.1/F2.2/F2.3/F3.3/F4.1/F4.2 | 源码、依赖、编译、运行制品与环境身份，公共入口集成 | 源到真实行为的证据链；不同版本、缓存、mock、历史环境不得冒充当前产物 |
| [K09 维护与演进](../../v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md) F1.1/F2.1/F2.2/F2.3/F3.1/F3.2/F4.1/F4.3/F5.1/F5.2/F5.3 | 重构/修复/增强/重写区别、兼容性、变化隔离、技术债、退役 | 明确有权保留和放弃的合同；只有真实后续变更才能观察维护收益；不强制保留明确废止的旧 ABI |
| [K10 运行质量](../../v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md) F1.1/F1.2/F2.1/F3.1/F3.2/F3.3/F5.1/F5.2 | 生命周期、失败/未知副作用、恢复、隔离、可观测性、首次运行 | 真实故障模型、资源释放、失败后状态、恢复与性能目标；不因正常路径成功而证明可靠性 |
| [K11 经济与风险决策](../../v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md) F1.2/F1.3/F2.2/F2.3/F3.2/F4.1/F5.1/F5.3 | 总成本、可逆性、复杂度、信息价值、质量下限、停止与知识维护 | 比较必要复杂度、阅读/变更/测试/迁移成本；没有实测不编造总分或提效比例 |

## 3. 核心专业决策方法：从问题到合适抽象

复用 [DP-K02-R19-01](../../v2.1.2/software-engineering-knowledge/K02/advanced-engineering-abstraction-boundaries-and-variation-r19.md)、[DP-K03-R19-01](../../v2.1.2/software-engineering-knowledge/K03/advanced-abstraction-level-selection-and-semantic-reuse-r19.md) 与 [DP-K09-R19-01](../../v2.1.2/software-engineering-knowledge/K09/advanced-abstraction-evolution-and-change-locality-r19.md)。这些是条件化专业方法，不是每次任务强制填写的模板。

1. 先确定改动性质：保持外部可观察行为的重构、纠正已授权合同的 Bug 修复、增加/改变行为的增强、明确新合同的重写。多种性质可以同时存在，但必须分开验收。明确已废止接口不是待保留义务；保留行为也不等同保留旧代码。
2. 从真实任务提取需要保持的合同、约束、资源/状态/信息所有者、当前可观察失败和获权变化轴。事实未知时不按常识补齐。检查谁拥有真值，谁负责读、写、确认及聚合。
3. 识别共性是语法、纯机制、不变量还是共同业务语义。对前置、结果、不变量、错误、授权/副作用、状态、清理、消费者版本和变化轴比较实际差异。只共享已证明等价的机制，保留必须独立的业务语义。
4. 选择最小合适候选：直接保持或局部内联；共享纯函数/私有机制；内聚职责模块；有真实消费者需要的窄公共接口；有真实获权替换需求的 Strategy。不是每次都逐级增加层次，也不凭不存在的未来扩展需求引入 Registry 或框架。
5. 比较实际代价及风险：变更传播、依赖方向、导航和认知成本、失败语义、兼容范围、验证难度、迁移与可逆性。至少比较真正可行的方案；已确认违约的方案应先排除，其他方案不必强行选唯一最优。
6. 设计能推翻判断的观察：同一真值下只改变一个材料事实/输入（省略、替换、冲突、边界版本等）；检查未受影响范围仍成立；随后以真实有权变更和有效旧/新 Oracle 观察维护效果，而非文件数或一次测试通过率。没有需要改变的事实或收益即停止。

## 4. 未来重构优先深化的四组知识

这些是本轮整理顺序的建议，不改变 K01–K11 完整学科目录，不绑定当前 DomainHarness 的实现。

- **A：语义抽象与边界（K01/K02/K03/K09）**：数据与控制、事实与权威、机制与政策、状态与引用所有权、模块内聚和依赖方向、信息隐藏、稳定/变化边界、兼容义务与明确退役。先整理真正专业知识及正反例，再谈 Package 或类。
- **B：行为合同与证据逻辑（K01/K03/K05/K06/K08）**：输入合法域、不变量、错误优先级、覆盖与 Oracle、证据身份/集合完备性、局部冲突与独立反例、未知/过期状态、边界/变异检验。此次真实 Package 只读案例见 [证据边界案例](evidence-boundary-case-r6.md)。
- **C：资源、可靠性及生命周期（K02/K03/K08/K09/K10）**：初始化/退出的责任与时序、资源独占与共享、错误传播和兜底、并发/事务/幂等、外部结果未知、恢复可观察性、性能/容量与故障隔离。保留学科机制与实际失败模型，不把当前 Boot/Core/SDK 类名当成知识定义。
- **D：变化经济性与交付（K04/K07/K08/K09/K11）**：依赖分解和写集、变更局部性、兼容与迁移、阅读与维护成本、反事实、独立审核与最小验证、停止/回退条件。不要重写现有 ADS；从学科机制解释何时应采取具体实践。

下一轮先逐组核对已有基础/深文是否已包含定义、机制、适用与不适用、可检验反例、来源和专业限制；有真实缺口才补正文并链接原 Owner。优先补 A/B 的材料性空白，之后 C/D，再用学科覆盖台账完成其他模块。现有课程已经具备覆盖和深入导航，不以更多知识文章或节点计数表示质量。保留源资格差距与已有研究结果。除非用户重新开启实施，不为知识整理新增代码、运行时框架或实验 DAG，也不启动 #286 正式接线。

## 5. 对重构的直接应用边界

未来 Tally 和 DomainHarness 应各自从这套知识选取与当前有权合同有关的原理，检查真实源码和行为；只有完成相应项目的新规范与设计后才能决定具体 Package 结构、语言、运行时、公开接口、依赖、验证工具和迁移方案。知识可提出反例及设计替代，不能越权取代项目规范。当前 #565 实现存在 R5 REWORK，保留为历史可复现实验样本，不合并或修正；无需用它来证明所有知识已验证。正式 T4/A24 在原实现验收合同下保持 NOT_RUN/HOLD；本阶段只读知识案例另列，不计为合格产品验收、质量因果增益或生产迁移。
