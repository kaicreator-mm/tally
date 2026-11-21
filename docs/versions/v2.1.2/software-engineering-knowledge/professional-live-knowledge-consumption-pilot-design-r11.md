# R11 有界真实知识消费试验：受权 Agent A/B + 真实代码与独立 Oracle

> [R10 选择手册](professional-knowledge-selection-playbook-r10.md) · [R10 评价协议](professional-knowledge-evaluation-protocol-r10.md) · [R10 变形题](professional-knowledge-evaluation-fixtures-r10.md) · [R10 原始出处命题档案](professional-primary-source-claim-dossiers-r10.md) · [工程 B/Domain Data #276](https://github.com/kaicreator-mm/tally/issues/276) · [前置知识实验 #299](https://github.com/kaicreator-mm/tally/issues/299) · [未来产品消费边界 #286](https://github.com/kaicreator-mm/tally/issues/286)

**R11 DOCUMENTED_PILOT_DESIGN_ONLY / NO_REAL_AGENT_RUN / NO_SOURCE_DISPATCH / NO_PRODUCT_OR_L2_FREEZE / QUALIFIED_CLAIMS=0。** 本设计针对 v2.1.2 K01–K11 的选定知识在真实编程任务中的作用，不定义新 Kernel、通用 Knowledge Server、独立 Workflow Engine 或机器 ABI；首轮仍为仅文档准备。运行阶段需要当前真实 Github Task/Owner、准许的仓库和写集、真实模型/执行收据与独立 QA。任何纸面教学评价不能冒充真实结果。

## 1. 目标和分层证据

三个问题必须分别回答：
1. **送达**：真实 Agent 的一次实际 Attempt，是否真的读取了精确 SHA/Blob 固定的 Kxx 原理/反例/来源，而不是只引用文章标题？
2. **工程正确性**：Agent 是否在真实源码固定输入下，产出被实际运行的受影响目标测试和有权 Oracle 接受的有限修复？仅有 Job exit 0 不足以证明目标被发现执行。
3. **比较效用**：在严格相同的源码、任务、合法业务资料、模型、工具权限和时间/Token 上限下，知识增强是否改善了该任务的正确性、反例识别和总成本？

**一个任务的 A/B 结果只能证明单实例差异、接线与评估流程可运转，不能推出普遍因果提效百分比。** 想报告可泛化的提升，需要额外样本、预定分母、随机化/盲评与不确定性报告。

## 2. 首轮只选一个真实且安全的代码任务

优先选 **Node22 代码库中真实测试选择或真实 Consumer 断言被遗漏而虚假 GREEN** 的受限修复任务：K03.F5.3、K05 Test Oracle、K08 Build/Test Receipt 与 K09 影响传播都具有直接用途。

- 有权 Owner 在一次实际 GitHub Issue 中选出**当前可访问**的测试仓库、精确 H0 Commit、需求、实际 Runtime/Runner、最小可执行修复目标、准许的分支/路径/效果及完成条件。此处**未选定真实 repo/H0**，不得沿用历史 test repo 名字冒充当前可访问实例。若仓库/权限缺失，明确 BLOCKED/NOT_RUN。
- QA 在任何 Agent 看题之前固定目标行为、真实受影响 testId/负例、独立 Expected 与对应真实命令/Runner，记录受权源和版本；独立 QA 的隐藏负例不能泄漏到 Agent Prompt、知识文章或另一个执行组。
- 任务在合法隔离仓库里允许实际修改受限代码/测试/Runner 配置，但禁止 main merge、GitHub Release、真实支付/敏感数据/生产 Provider 操作。可以是**受权的 seeded defect**，但必须记录人工注入，不能计为真实线上事故。
- 与 R10 EV10-01/W19 避免逐字同题：使用实际仓库不同文件/实际 Runner 选择机制与未暴露负例；仅测试“知道 EV10 的答案”不属于真实代码工程泛化。
- Node22 是环境候选（以目标仓库真实工具链版本为准）；无需求不能强制升级 Node24。无额外需要不重复 Woodpecker CI。

如果没有可运行源码、真实 Owner/Host 许可或独立可观察 Oracle，E1 STOP，允许继续研究但**不能伪报已经实验**。首轮完成后，可再考虑不同学科的 Mock 双项目 Credential 引用串扰作为第二独立任务，但不预先扩张首轮写集。

## 3. 最小闭环：一个 H0、两个隔离执行、独立 Reviewer

~~~text
GitHub Native Issue / authorized task + fixed repo source SHA H0
                         |
       QA freezes actual selected test IDs + negative Oracle
                         |
            two isolated worktrees from identical H0
              /                              \
A: task baseline                    B: knowledge-assisted
same official Task/Owner/Tools      same official Task/Owner/Tools
same model, budget, permissions     same model, budget, permissions
no selected Kxx injected           plus minimal pinned Kxx/context
              \                              /
        H1A / H1B changes + real runner Discovery
                + Execution + Oracle results
                         |
      independent blinded QA/Reviewer checks true evidence
                         |
      scope-limited finding + actual cost/Attempt evidence
      (no automatic Domain Package or Release qualification)
~~~

两组都必须获得**同样的合法任务/业务规则、真实源码、现有 GitHub 权限和工具**。不能通过剥夺 A 组必要的权威 Source 来人为制造性能差距。知识实验组 B 仅多一项：限定、版次固定并带失效反例的专业**学术方法**；不是给 B 更多运行权限、不同模型或隐藏答案。

两组不得共享工作树、旧 Context、代码修改、缓存中的生成结果或对方输出。可随机化先后顺序、另用独立 Reviewer 盲评分；所有失败、NOT_RUN、超时、人工接管都纳入真实分母。R10 B1 Topic-only 与 B2 Full corpus 暂不运行，首轮只做 B0 vs B3。

## 4. 两级消费，不把 Prompt 注入冒充 Tally Domain Package

**Stage E1：真实 Agent + 受限知识输入的 Shadow Pilot。** 无需编译机器 Package：已获权 Agent 在实际隔离代码任务开始前，直接从版本固定的 K01–K11 Markdown 读取选定片段。用一份文档型 Selection Receipt 列出每条内容的 Corpus Commit SHA、Git Blob SHA、真实文件路径、Kxx Owner、为何相关、适用条件与排除原因。Agent 必须对真实当前源码和 Runner 重新求证，不能用教学案例推断项目已成功。E1 的真实收据可证明 Agent **确实收到/引用了选定知识并完成真实代码任务**，但不能声称 Tally App 已自动消费生产 Domain Package。

**Stage E2：Tally 正式消费者。** 仅当另一个**编号 v2.1.x Child 的 Product/L2/DAG** 明确冻结、#286 被合法重新裁决且选定真正当前的 DomainHarness 0.11.x/Engineering B 消费 ABI 后，再复用 E1 已验证的最小知识选择/评估逻辑接入。先读回 #276 真实当前源码与 public getPolicy()/Domain Data/Workflow 消费能力，避免把旧 v0.10.3 参考或 #299 PRE_PACKAGE 实验误称新 Runtime ABI。现有 Tally Engineering B 优先，不另造通用服务器、第二套 Host/Task/Claim/Release Authority 或要求当前候选必须经历史外部 Producer 先生成。

这里的真实规范来源依旧是**当前有效 ADS 标准、受权业务需求、GitHub Native Task/Exact SHA、实际 Provider/Runner 合同**。Kxx 教学/学科方法和五项 R10 档案虽可作为专家建议，不会因注入上下文成为有权业务 Rule 或取得原文复制许可。#286 顶部最新 Owner 方向明确 ADS→Tally，历史 mandatory outside DomainForge producer 仅作 superseded 参考。

## 5. 必需的真实证据清单

| 真实收据 | 事实来源/最小字段 | 缺失时的结论 |
| --- | --- | --- |
| Task 权限 | GitHub 原生 Issue、Owner、可写仓库/ref/路径、合法 Effect、真实任务验收要求 | 无权写入，STOP |
| 固定源码 | H0、实际 Git 依赖/Runner/Lockfile、每组独立 H1A/H1B、Diff | 不能宣称 Exact-SHA 实验 |
| 知识选择 | Corpus Commit/各文件 Git Blob、Kxx 概念 ID、具体片段/失效条件、实际提供给 Agent 的输入清单 | 不能宣称真正消费了特定知识 |
| Agent 真实运行 | 模型/Provider 版本、权限/预算/工具、Attempt 和实际输出时间/成本 | Agent NOT_RUN |
| Test Discovery/Execution | 真实目标 testId 列表、Selected/Executed/Skipped、Assert/EXIT、受检版本 | 目标 NOT_RUN 不是 PASS |
| 独立 Oracle | 任务开始前冻结的合法 Expected、负例、真实执行/Reader 结果、独立 Reviewer 范围判断 | ORACLE_NOT_VERIFIED |
| 完整成本 | 全部分派任务、失败、超时、人工审查和修复、Token/工具/真实 CI/Host 成本 | NOT_MEASURED，不报告提效 |

表中状态词只为教学说明，不定义 Tally 或 DomainHarness Wire Enum。不得将 Token、真实客户身份/私有数据、敏感 Provider 响应原样写入公开日志。

## 6. 四项必须可执行的 Oracle

- **O1 Scope**：两组均从同一真实 H0 启动且操作仅限合法路径；不得出现越权生产效果。
- **O2 Discovery**：QA 目标测试身份在真实当前 Runner Discovery 与 Execution 中分别有据可查；不能拿 40 条无关测试 GREEN 宣称新目标 PASS。
- **O3 Behavior**：真正受影响的 Consumer/负例有当前独立 Oracle，对错误期望（例如 Writer 自行更新 Snapshot）应拒绝；不能只更名测试文件绕过有材料的行为验证。
- **O4 Comparative record**：盲评比较真正源/范围的正确性、违法放行、过度拒绝、错误根因和全成本；单样本只报观察，不报通用统计显著性。

必要 O1/O2/O3 缺一则不能宣称当前代码任务通过；即使四项都通过，也只代表受限真实任务，不自动推动 #286、L2、Release 或 Package。

## 7. 分段准入及验收

| 阶段 | 本阶段实际交付 | 进入下一阶段之前必须成立 |
| --- | --- | --- |
| **D0：本次 R11 设计** | 本文、R10 P2 精确修复、GitHub 独立文档审查 | 不能冒充真实 Agent 运行；#286 继续 Deferred |
| **E1：真实受限 Shadow A/B** | 单个受权实验 Issue、H0、两次隔离执行与真实 Node Runner、独立 QA 和完整证据 | Repo/Owner/Task/Branch/Allowed paths/Oracle/Provider 权限明确；无影响生产 Host |
| **E2：Tally 正式知识消费者** | 在真实选定 Engineering B/DomainHarness 版本内经合法 ABI 消费与重复真实任务验证 | 新编号 v2.1.x Product/L2/DAG、#286 决议和真实编译/运行授权 |
| **E3：经验候选回流** | Source/Actor/范围明确的纠错、候选知识更新与失效案例 | 不把一次 Agent 自述或一次 A/B 差异自动升格强制 Rule 或定时自动化 |

**推荐路线：D0 → 一个真正允许本地写与运行的 E1 → 评估价值 → 再决定是否投入 E2。** 不为证实一次知识选择假设预先实施整个新 Knowledge Server 或 Kernel。

## 8. E1 未来授权任务的精确字段模板（非当前触发词）

~~~ini
ROLE=OWNER_AUTHORIZED_BOUNDED_AGENT_EVALUATION
REPO=actual-accessible-github-repo
TASK_ISSUE=real-native-issue
SOURCE_H0_SHA=real-immutable-base
ALLOWED_PATHS=exact-task-code-test-runner-files
ALLOWED_BRANCHES=two-disjoint-experiment-branches
ALLOWED_EFFECTS=READ_SOURCE,LOCAL_TEST,WRITE_EXPERIMENT_BRANCH
FORBIDDEN_EFFECTS=MAIN_MERGE,RELEASE,REAL_FINANCIAL_EFFECT,LEAK_SECRETS
ACTUAL_RUNNER_VERSION=confirmed-from-repo
BASELINE_AND_ASSISTED_MODEL=identical-version-settings-budget
CORPUS_SHA=reviewed-K01-K11-document-snapshot
INDEPENDENT_QA_ORACLE=precommitted-negative-tests-and-expected
OUTPUT_LABEL=one-time-bounded-real-agent-evaluation
~~~

本字段清单只陈述进入 E1 时需要的**真实事实**，不能填上示意值就启动真实任务。若运行需要本地 Agent，只在实际仓库/Issue/H0 已明确之后另行给有限的一次性触发词；禁止后台无限推进。E1 未完成前保留 **REAL_AGENT_RUNS=0、PRODUCTION_ORACLES=0、PRODUCT_CONSUMER_NOT_FROZEN**。
