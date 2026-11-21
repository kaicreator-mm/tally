# R8 专业知识来源→工程命题→失效边界的最小可核对矩阵

> [67+2 来源编目](external-source-catalog.md) · [165 点课程台账](foundation-concept-coverage-ledger.md) · [R8 综合演算 W13–W17](professional-reasoning-worked-traces-03.md)
>
> **SOURCE_REFERENCE_MAPPING_CANDIDATE / NO_ORIGINAL_CLAIM_QUALIFIED / NO_DOCUMENT_COPYRIGHT_LICENSE_ASSUMED。** 这里只验证“公开方法可定位、候选工程命题的适用假设明确”，不是将每项命题转换成合法生产 Rule。Axx 是历史编目编号，不是原条文号；引用表不代表所有原文已逐句核对。当前没有新建 DomainHarness Rule、Knowledge ABI 或真实 Host 运行记录。

## 一、权威等级和技术事实不是一个总分

- **正式规范**：ISO、IEEE、NASA 等各有特定实体/项目的适用范围。NASA 有权要求 NASA 项目达到某条款，不代表普通 GitHub 项目因此新增同样审批门槛；ISO 技术内容被引用也不表示已取得其受限全文复用许可。
- **技术合同/项目实践**：PostgreSQL 文档只对对应数据库版本和隔离语义有效；GitHub Required Checks 是否启用必须查询具体仓库，不能凭官方文档存在就推断已启用。
- **工程研究与度量**：DORA/SPACE 为经验研究框架，不能从行业层面的相关性直接推出某单次 Agent 并行策略提升了生产率。
- **教学中原创模型**：本文 W13–W17、K01/K02/K07/K08/K11 章节的公式/案例是合成问题，不是任何外部标准的逐字条文或真实 Agent 测量。

## 二、精确的“来源—命题—条件—反例”映射

| 知识 Owner 与原课程点 | 参考出处（已有编目） | 本轮可据以研究的命题/机制 | 失效边界或必须求证的反例 | R8 状态 |
| --- | --- | --- | --- | --- |
| K01.F1.2 / F2.1 | A05 / A12 / A13 需求与业务分析资料 | Stakeholder 观察与有权规范应分别建模，访谈/操作记录存在偏差 | 三个博客转述一个过期政策；无权销售自称最终决策者 | 原创知识候选，非业务 Policy |
| K02.F1.3 | A06 ISO 42010:2022 / A15 SEI ATAM | 用 Stakeholder Concern、假设、备选/后果组织 ADR 与回评 | 先验“微服务最好”，但无真实质量场景/吞吐基线 | ADR 方法候选 |
| K02.F2.3 | A06 / 具体 DomainHarness 版本合同待核实 | 依赖引用闭包、业务权限和 Host Effect 是不同边界 | 已声明 Extension 但未经授权访问外部 Host | 具体 BUILD/ABI 未资格 |
| K02.F3.3 | A32 AWS idempotent API / A51 PostgreSQL 18 isolation | 本地 DB COMMIT 不自动涵盖外部 Provider Effect | 远端已提交、ACK 丢失、本地仍 PENDING | Provider 合同 UNKNOWN |
| K02.F5.1 | A35 / A64 演进式 DB/Parallel Change | 版本共存下可评估迁移，明确完全重写时不必建旧 Shim | 旧消费者已退役却要求永久双写；或存在旧客户端却强删 | 真实受支持消费者 UNKNOWN |
| K07.F2.1 | A20/A21 工程协作与 Code Review | Writer/Reviewer 来源独立及 Diff/Exact SHA 绑定 | 同一 Builder 会话自称独立 Reviewer，或旧 HEAD Review 误覆盖新修改 | 文档审查候选 |
| K07.F3.3 | A29 Agent SDK / A40 SPACE | 选择够用的当前 Context，关联正确性与资源成本 | 上下文充满旧 FAQ，缺实际有权政策，答案仍强行 PASS | Agent 对照 NOT_RUN |
| K07.F5.2 | A40 SPACE / A54 ACM SIGSOFT | 完整分母、质量/成本/协作代价，避免活动指标冒充价值 | 只比较成功 Agent 的延迟或生成 Token，忽略超时/人工 | 真实效果 NOT_MEASURED |
| K08.F5.1 | A56 GitHub Required Checks / A68 SLSA Provenance v1.2 | SHA+Digest+可信 Builder+实际发布审批是不同证据 | 自写 JSON provenance、D1 通过但上传 D2 | SLSA 实施 NOT_RUN |
| K08.F5.2 | A35/A64 数据库渐进迁移 / A45 NASA Delivery context | 代码回滚与数据恢复、外部 Effect 补偿必须独立证明 | H2 镜像回滚 H1，但丢失 schema 或远端付款未恢复 | 真正 Host/Release NOT_RUN |
| K11.F2.1 | A04 生命周期 / A10 15939 度量 | 工时、依赖关键路径、计划承诺和真实结果分开 | 8h 理想关键路径误当 95% 工期 SLA | 项目分布 NOT_MEASURED |
| K11.F3.1 | A10 度量 / A31 SRE | 在可辩护模型下同时考虑概率、后果和信任边界 | 概率未知就按零计算，或拿期望成本替代业务禁令 | 合成风险比较 |
| K11.F5.1 | A33/A40/A69 DORA/SPACE/度量历史 | 改善交付同时保持合格质量和完整成本/失败记录 | “增加 Review 次数”直接等同 ROI 或可靠性提升 | 受控比较 NOT_RUN |

## 三、从学科研究真正进入 Tally 领域智能，需要额外什么

仅当确有实际消费目标时，再为**选定少量命题**确定 `Source URL/edition/Clause, Claim text in own words, applicability, counterexample, rights, qualified owner, actual consumer Contract, real Oracle receipt`。已有 69 项参考不是 69 个可直接编译到 DomainHarness 的 Rule；已审专业 Markdown 也没有自动生成软件工程 Domain Package。

若研究问题当前只有行业参考或合成案例，输出 `KNOWLEDGE_CANDIDATE`；如果业务标准权限未知，输出 `ASK_AUTHORIZED_OWNER`；如果真正的外部 Provider Effect 未读回，保留 `UNKNOWN`。不为每个基础课程点开独立审批 Issue，也不让来源搜索取代 GitHub 当前源码/正式 Issue 的事实来源。

**停止规则**：只把对当前真实问题有信息增益的来源带入 Agent 上下文；有理由排除的过期或不适用来源，不因引用次数多就提升权威。