# K05 · Agent Skills / 操作技能与交接

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #526](https://github.com/kaicreator-mm/tally/issues/526) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## Tally Workflow 选择与实际消费者边界

## 5. Tally 消费路径与跨模块接口

**主要 Workflow：** L3 Verification / Review / Delivery

| 工作环节 | 读取的当前事实 | 本模块的输出 | 原有权威/禁止推断 |
| --- | --- | --- | --- |
| L1/L2 | 接受要求、设计不变量和数据来源 | 可信 Oracle 与验收测试候选 | 未确认业务真相不写死期望 |
| Task design | 实际受影响代码/合同 | 各层最小测试和正反矩阵 | 不强制全套 E2E |
| Agent 执行 | 真实命令/环境/exact SHA | 测试发现/退出/失败明细 | exit 0 未必相关测试执行 |
| Review/Delivery | 源版本/CI/Host/用户路径证据 | 分层资格与遗漏/未知 | Test PASS 不等于 Release PASS |

**跨模块组合：** K01 供给业务预期、K02 供给系统接口/不变量、K03 交付源码与回归测试、K06 使用失败信息定位根因、K08 负责 CI/打包与实际交付。K09 的保留行为测试属于本模块的复用场景。

## 现有 Agent Skills 的输入输出与拒绝路径

## 6. Agent Skills（输入、输出和验证）

### SK-K05-01 · Test Matrix Designer

- **调用时机：** 需求已被接受且待验证行为明确
- **最低输入：** 业务来源、API/状态、实现目标与适用测试层
- **产出与格式：** 带来源的成功/失败/边界/不变量测试方案
- **自查及异常路线：** 故意反转实现应该触发至少一个失败断言
- **不能做：** 模仿新实现结果定义需求

### SK-K05-02 · Executed Test Evidence Reader

- **调用时机：** Builder 声称测试已通过
- **最低输入：** 真实测试命令/用例输出/exit code/环境/HEAD
- **产出与格式：** 实际发现测试、已测范围、未测/失败及 provenance
- **自查及异常路线：** 确认受影响用例被 runner 选择，失败源头可定位
- **不能做：** 将空套件或旧 SHA 的 green 认定当前成功

### SK-K05-03 · Selected Path Verifier

- **调用时机：** 涉及跨组件、UI/CLI、安装产物或外部服务
- **最低输入：** 用户入口、选定构建和必要真实依赖
- **产出与格式：** 入口→调用→结果的可重复收据，真实/Mock 资格区分
- **自查及异常路线：** 若还缺 Host/Provider 权限则保留明确 UNKNOWN
- **不能做：** 仅用 stub 的结果代替实际端到端行为

## 可直接作为单次 Agent 任务使用的提示词

> 注意：以下是**方法提示词**，不是写入许可、部署指令或平台强制格式；使用时必须先填写真实 Task/当前版本/输入来源。不得交给 Agent 无界持续执行。

```text
MODULE=K05
KNOWLEDGE_VERSION=TALLY_V2.1.2_SOURCE_CANDIDATE
ADS_EFFECTIVE_SOURCE=933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c
ROLE=SCOPED_ENGINEERING_ANALYST
TASK_SOURCE=<actual issue, current sha and approved scope>

你是测试设计和验证分析 Agent。输入已批准需求/接口/不变量、当前变更范围与测试环境。先设计可观察 Oracle，再列正常、禁止、边界、重复/超时案例并选择最小充分测试层；核查 Runner 真正发现/执行目标用例及当前 SHA/exit code。输出实际执行证据和遗漏/UNKNOWN。Mock 结果只限其模拟范围，不能自行设置新的 Validation Gate State 或发布 Release PASS。

输出时明确分隔 OBSERVED / SOURCE_BACKED / PROPOSED / UNKNOWN / NOT_RUN。
只要求本次问题所需内容；不自造 Product/L2/Claim/Review/Release 权威。
```

## 最小交接清单

- [ ] Task/Issue 与适用版本明确；信息来自可读的现行来源而不是聊天历史猜测。
- [ ] 具体问题、使用了哪些知识和适用条件明确，未加载整仓无关文档。
- [ ] 结果中的已观察事实、推断、建议、未知和测试未执行分开记录。
- [ ] 提交的 Artifact/测试/Review 指向准确 subject；没有合法执行环境时只给分析结果。
- [ ] 无界循环、未获权限的写入或虚构 PASS 被拒绝。