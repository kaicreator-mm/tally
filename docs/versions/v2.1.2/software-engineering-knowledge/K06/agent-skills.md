# K06 · Agent Skills / 操作技能与交接

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #527](https://github.com/kaicreator-mm/tally/issues/527) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## Tally Workflow 选择与实际消费者边界

## 5. Tally 消费路径与跨模块接口

**主要 Workflow：** L3 Repair / Incident / Maintenance

| 工作环节 | 读取的当前事实 | 本模块的输出 | 原有权威/禁止推断 |
| --- | --- | --- | --- |
| 失败观察 | 来源/测试/Runner 的真实输出 | 故障分类、首个失败点 | 不能先宣告源码错误 |
| 诊断 | 候选解释与复现 | 最小区分测试与根因定位 | UNKNOWN 不能冒充 RootCause PASS |
| 修复 | 受控 Task/Owner/写集 | 最小 Patch、必要测试 | 修改必须遵守原有授权 |
| 交付/学习 | 新 HEAD 与真实结果 | 回归收据、可推广/限制案例 | 不能自动 Release 或推广硬规则 |

**跨模块组合：** K01 对业务规则真伪作权威判断；K02 处理需回溯的结构/接口；K03 执行具体 Patch；K05 生成回归 Oracle；K07 管理 Repairer/Verifier 的独立性；K08 区分源码与 CI/Host 的失败事实。K11 帮助确定修复是否已经失去边际收益。

## 现有 Agent Skills 的输入输出与拒绝路径

## 6. Agent Skills（输入、输出和验证）

### SK-K06-01 · Failure Triage

- **调用时机：** 收到 CI/TypeScript/测试/Host 错误
- **最低输入：** 实际日志、命令/阶段、SHA、上下文与已成功的步骤
- **产出与格式：** 症状、最早失效位置、竞争解释和最小验证建议
- **自查及异常路线：** 写清直接观察 vs 推断 vs UNKNOWN
- **不能做：** 从最终 exit code 自动生成代码修改

### SK-K06-02 · Bounded Repairer

- **调用时机：** 已确认可在当前合法 Task 修改范围内修复
- **最低输入：** 失败复现、Task 写集/当前 HEAD、相关测试
- **产出与格式：** 最小 diff/修复理由/新 HEAD/受影响测试收据
- **自查及异常路线：** 比较新旧失败行为并保护旧功能
- **不能做：** 无界运行直到发布或编辑越权文件

### SK-K06-03 · Incident Learning Extractor

- **调用时机：** 一次真实故障已得到可信结果或保持阻塞
- **最低输入：** 确切问题、修复/测试、环境/前置条件
- **产出与格式：** 经验 Case 的原因链、适用条件、反例和未知
- **自查及异常路线：** 判断是否真的跨任务可复用并避免泛化
- **不能做：** 将一次失败的个别修复当作全局 MUST 规则

## 可直接作为单次 Agent 任务使用的提示词

> 注意：以下是**方法提示词**，不是写入许可、部署指令或平台强制格式；使用时必须先填写真实 Task/当前版本/输入来源。不得交给 Agent 无界持续执行。

```text
MODULE=K06
KNOWLEDGE_VERSION=TALLY_V2.1.2_SOURCE_CANDIDATE
ADS_EFFECTIVE_SOURCE=933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c
ROLE=SCOPED_ENGINEERING_ANALYST
TASK_SOURCE=<actual issue, current sha and approved scope>

你是受控故障诊断 Agent。读取当前源/测试 SHA、完整执行阶段、原始日志与最近变更。先定位 FIRST_FAILED_STEP，区分观察事实与竞争假设，设计一个可证伪的最小探针；只在合法 Writer/写集内提议根因修复，运行受影响正反回归并记录结果。连续无新证据的重复尝试必须停止。禁止将 CI 最终退出码或历史问题直接当成当前根因。

输出时明确分隔 OBSERVED / SOURCE_BACKED / PROPOSED / UNKNOWN / NOT_RUN。
只要求本次问题所需内容；不自造 Product/L2/Claim/Review/Release 权威。
```

## 最小交接清单

- [ ] Task/Issue 与适用版本明确；信息来自可读的现行来源而不是聊天历史猜测。
- [ ] 具体问题、使用了哪些知识和适用条件明确，未加载整仓无关文档。
- [ ] 结果中的已观察事实、推断、建议、未知和测试未执行分开记录。
- [ ] 提交的 Artifact/测试/Review 指向准确 subject；没有合法执行环境时只给分析结果。
- [ ] 无界循环、未获权限的写入或虚构 PASS 被拒绝。