# T22 · 软件工程知识体系当前入口与阶段验收候选

[#582](https://github.com/kaicreator-mm/tally/issues/582) / [#563](https://github.com/kaicreator-mm/tally/issues/563)。冻结输入 405548998f1bd4dbf2b2b8c32587fcdabb605fc6；T15–T21 已按独立审核及 exact-head 合并，T22 本身待独立知识验收及合并。知识优先，不修改软件实现、SDK、原 ADS 合同、原 Kxx 定义或原 Source/Claim 权威。此处是统一阅读/决策入口，不是运行时知识引擎、生产规则、Task 状态权威或产品 Release。

## 1. 按需要选择阅读入口

| 决策需求 | 当前材料 |
|---|---|
| 了解原专业定义及学科组织 | [K01–K11 原学科知识架构](../../../../v2.1.2/software-engineering-knowledge/systematic-knowledge-architecture.md)；[重构知识导航](../../refactoring-foundation.md) |
| 按软件类型、功能、模块/代码抽象、质量目标选案例 | [142 条当前案例目录](T22-current-case-catalog.md) |
| 按专业知识点找已有实际机制与条件启发 | [165 条当前反向索引](T22-current-knowledge-index.md) |
| 追溯真实版本、源码片段和读取资格 | [143 个来源 Blob 登记](T22-current-source-ledger.md) |
| 深化专业机制、边界和反例 | [T01 六项理论](T01-theory-ab.md)、[T10 六项理论](T10-theory-cd.md) |
| 阅读实际选知过程及跨学科综合 | [T13 八组只读选知](T13-selection-walkthroughs.md)、[T21 三组综合推演](T21-cross-subject-judgment.md) |
| 内部实际研究证据（单独保留） | [R6 已有研究 Package 案例](../../evidence-boundary-case-r6.md)；不混计外部案例或 T21 执行 |

知识原语义仍由各 Kxx 原文拥有；新索引不复制原定义或引入新的知识类型。实际问题先读有权合同、环境、已选入口和现有代码；再根据需要从上述入口选择少量相关知识与候选案例，核对案例资格/版本、真实变化轴、失败与成本，拒绝不等价的抽象。L1 明确需求和 Oracle；L2 比较职责、依赖、状态/资源、事务、失败域及质量；L3 具体化函数/数据/接口/异常；实施、验证、维护按真实风险执行。简单任务可合并阶段。若真实合同已明确、所需机制及反例足够，停止额外研究；若缺失材料事实会改变选择，定向补证或保留 UNKNOWN，不全量注入知识库或机械使用所有 Kxx。知识冲突以适用条件、版本和真实权威裁定。

## 2. 实际重数与资格

| 对象 | 数量 | 限定 |
|---|---:|---|
| 原专业知识点 | 165（11×15） | 保持原定义 |
| 当前案例 | 142 | 首期125 + T15–20新增17；118真实来源参考/24原创 |
| 固定来源文件版本 | 143 | 首期106 + 第二批37；16份登记中4次重复不计；同文件不同Blob分计 |
| 案例到知识关系 | 164 | 原127 + 37；129机制/35条件，不是原概念之间的48条关系 |
| 有机制关联的原知识点 | 27 | 仅有条件启发6，无直接案例标引132；不代表专业知识缺失或完整率 |
| 理论补深 | 12 | T01/T10，不另计案例 |
| 只读选知及综合推演 | 8 + 3 | T13/T21；T21引用59个不同原知识点，适用性独审通过，不新增机制关系 |
| 登记来源有列名独立读取记录 | 84 | 首期47 + 新37；另59无列名记录，非全源重新独审 |
| 外部项目执行及行为Oracle | NOT_RUN | 无外部运行资格；无性能、维护成本或因果收益实测 |

上述计数由当前目录、反向索引、来源登记实际重数（首期正反向关系127/127一致，新增37条）及既有独立报告分开得出。首期47/59独立复读仅集合汇总，逐文件身份以原报告为准，本轮未重新分配或审计这些来源。T21评审额外读取的 p-limit test.js 不在登记范围，不能将总数加一。

## 3. 审核与合入链

| 批次 | 任务及收尾记录 | PR | 审核HEAD | 合入SHA | 资格 |
|---|---|---|---|---|---|
| T00–T14 | [#567](https://github.com/kaicreator-mm/tally/issues/567#issuecomment-7157423397) | [#566](https://github.com/kaicreator-mm/tally/pull/566) | `aca4afcb03c442ea61501a5b6a6138450c5f8314` | `772eb6819a30a4fb248c6c60d24e08bb02a00c19` | ACCEPT_SCOPED / MERGED |
| T15 | [#568](https://github.com/kaicreator-mm/tally/issues/568#issuecomment-7158258711) | [#569](https://github.com/kaicreator-mm/tally/pull/569) | `80f28e78afa59f0a88d3f88feca1578d73cd6253` | `724e83f9e0c1e515388164d949849aff233a0a21` | ACCEPT_SCOPED / MERGED |
| T16 | [#570](https://github.com/kaicreator-mm/tally/issues/570#issuecomment-7159662209) | [#571](https://github.com/kaicreator-mm/tally/pull/571) | `4a9446924f4f8f719e390fc709dcc5df81ab18f9` | `ece526072ee323d7e68bc8feb9623fe3bc545fde` | ACCEPT_SCOPED / MERGED |
| T17 | [#572](https://github.com/kaicreator-mm/tally/issues/572#issuecomment-7160095757) | [#573](https://github.com/kaicreator-mm/tally/pull/573) | `c83a0540f91fe0baa8b519868eef8f39df08872f` | `0fff9c230c86f7e421b332b7d9815ad77825c367` | ACCEPT_SCOPED / MERGED |
| T18 | [#574](https://github.com/kaicreator-mm/tally/issues/574#issuecomment-7160519238) | [#575](https://github.com/kaicreator-mm/tally/pull/575) | `dbdc63a70bb40401a2d97bd91db2e08a4c2ff994` | `0516b64b17da66f3029dbd34664b1328bf254a54` | ACCEPT_SCOPED / MERGED |
| T19 | [#576](https://github.com/kaicreator-mm/tally/issues/576#issuecomment-7160962254) | [#577](https://github.com/kaicreator-mm/tally/pull/577) | `62881abfc46127a6a2c0fc3605a0a88a010530db` | `ddef3026fa47000f60219e27c071f77523a929a7` | ACCEPT_SCOPED / MERGED |
| T20 | [#578](https://github.com/kaicreator-mm/tally/issues/578#issuecomment-7161383125) | [#579](https://github.com/kaicreator-mm/tally/pull/579) | `aee25d18f3bc93063686084f1c4bb3e1e9c9fec7` | `eddfc89124e9aed7355b33187b63379e893c165f` | ACCEPT_SCOPED / MERGED |
| T21 | [#580](https://github.com/kaicreator-mm/tally/issues/580#issuecomment-7161588425) | [#581](https://github.com/kaicreator-mm/tally/pull/581) | `ff2dbb6f20eeaa9811a49113a8beef54f3219d7c` | `405548998f1bd4dbf2b2b8c32587fcdabb605fc6` | ACCEPT_SCOPED / MERGED |

P2-T18-01（可靠重放 vs 允许丢失）在 T19 独审关闭；P2-T19-01（正阈值、rows.Err 与部分列表）在 T20 独审关闭；P2-T20-01（仍排队 vs 已出队许可）在 T21 独审关闭。T21 P0/P1/P2=0/0/0。原历史文档及冻结版本仍可访问，T22 不重审所有过去案例或增加源资格。T22 的候选状态不是最终关闭事实；验收以 #582/#563 和实际 PR 当前性为准。

## 4. 本批停止与后续真实任务的知识缺口

- TS/JS、Java/C++/Rust、真实业务订单、同功能架构、完整订阅刷新子系统、实际重构历史已有限补深；不是所有语言/类型/业务/产品都完整覆盖。T19 为子系统局部调用链，未审完整产品、鉴权、启动周期和所有依赖。后续遇到 GUI/移动、AI/嵌入式、跨平台、完整产品或深业务合同才定向补缺口。
- 132个原概念无直接案例标引，仍是原专业知识；不以补齐数值为任务。59个历史登记来源无列名独立读取记录；如真实决策依赖该来源，定向重新读取并核实版本、限制。外部项目未实际执行；已核源码、维护者理由、测试代码都不能替代行为证据。
- 实际维护工时、缺陷率、迁移代价、真实消费效果及知识因果收益 NOT_MEASURED / NOT_RUN；既有实验和 R6 证据仅在原限定范围使用。没有新 A/B 实验、自动检索资格、可运行 Package、生产集成或 SDK 验收。
- 完成本轮限定知识独审及合入后，结束 T15–T22 有界批次。#565 仍 HOLD、#286 DEFERRED、T4/A24 NOT_RUN/HOLD；#563 保持产品后续决议入口，不以知识验收自动启动实施或发布。未来真实项目消费必须先取得其有权需求和源码，再按相关知识决定抽象，保留必要验证及反例；不为证明知识价值而开启无界实验或重复整库审查。

验收范围：当前索引完整性、关系一致性、来源计数与身份、资格区分、可阅读入口、真实历史接受/修正链及缺口；明确非软件产品/执行验证/生产发布资格。待独立审核。
