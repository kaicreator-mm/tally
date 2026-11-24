# K05 · R18 进阶：架构/代码质量的独立审查、评分校准与评价工具自检

> `DOCUMENT_CANDIDATE / REVIEW_CALIBRATION_METHOD / NOT_A_NEW_ADS_GATE` · [K05](README.md) · [Oracle 有效性/R4 实验限制](advanced-test-fixture-and-oracle-validity-r15.md) · [质量知识选择](../professional-quality-risk-knowledge-selection-r18.md) · [#551](https://github.com/kaicreator-mm/tally/issues/551)。
>
> 参考：[Google Code Review Guidelines](https://google.github.io/eng-practices/review/reviewer/looking-for.html) 将设计、功能、复杂度、测试、命名、文档等方面分别审查；[SEI ATAM](https://www.sei.cmu.edu/library/architecture-tradeoff-analysis-method-collection/) 用有权质量属性场景评价架构；[ISO/IEC 25010:2023](https://www.iso.org/standard/78176.html) 提供质量特性分类。**现有来源不能提供一个通用的“代码可维护性 95/100”真值**；本篇提出限定的 Reviewer 可复核性方法。

## 1. 四层证据不互相代替

| 层 | 审核问题 | 限定结论 |
| --- | --- | --- |
| **功能资格** | 已授权要求是否被真实测试发现、执行和区分？ | 只支持已运行的入口、平台、数据与时间窗 |
| **架构质量** | 首次 L2 的职责/状态 Owner/质量属性方案是否能承受指定失败或变化场景？ | 需针对**第一次 L2**的实际依据与替代方案，不由修复后的 S5 覆盖 |
| **代码理解** | 陌生 Reviewer 能否在固定条件下正确预测行为/找到代码路径？ | 需要实际盲读、理解题与耗时/误读，不能单用 lint/LOC |
| **修改可维护性** | 另一维护者对冻结的实现作预注册后续改变时，旧合同与新需求是否均成立？ | 需真正的修改分支、旧/新行为 Oracle、所有成本和失败 |

即使功能全绿，架构仍可能过度复杂、API 难理解或未来难改；即使模块边界优美，严重越权/数据丢失仍是硬失败。报告四个维度的分布和具体证据，**不将它们合成可抵消关键违约的总分**。

## 2. Reviewer 质量的最低准备

1. **优先冻结评审问题，再看候选。** 收集当次确实有权的目标、性能/可靠性/兼容性边界、失效模型；Reviewer 不应知道 A/B 知识组、作者、哪个输出更长或哪个 Candidate 测试更多。无法去除组别时 `NOT_BLIND`，不能虚称随机或双盲。
2. **按可复算的等级分类具体缺陷。** `MATERIAL` 需给需求/不变量、入口/状态、真实失败轨迹或可执行反例；`NONMATERIAL` 给可说明的风险和替代成本；`STYLE_ONLY` 不当作架构重大缺陷；`UNKNOWN` 需要缺失事实；`NOT_TESTED` 指被声明的性质未被 Oracle 触达。
3. **至少两名互不抄答的非 Writer Reviewer**，先各自判定再汇合。记录独立 `Finding-ID`、证据、严重度、是否有关、置信限制。若分类不同，第三方或事先指定裁定人按原先的质量属性场景、源/代码/日志裁决；原样保留两份原始意见，不能“协调后都是一致”。
4. **Reviewer 一致性与歧义分开计算。** 可报告共同发现/不同发现、严重度一致率、裁决后剩余 `UNKNOWN`；有足够样本才计算合适的一致性统计。不能让长答案、文档数量或某个架构模板的贴合度隐性加分。
5. **阴性、阳性与变异校准**：事先给 Reviewer 一个无额外问题的中性示例，以及有真实错误的边界/过度设计/旧行为破坏示例；保证 Reviewer 能区分“确实有违约”与“只是另一种实现风格”。校准材料不应包含正式实验隐藏答案，不能让同一 Reviewer 自己写题、写答案、给自己实验评分。
6. **记录来源和实际工作**：Reviewer Git SHA、模型/人类身份、工具、是否看过原 Builder 消息、时间/成本、可访问的上下文以及独立性程度。模拟 Reviewer 或同一协调者只换角色名，不算独立审查。

## 3. 对质量 Oracle 本身的可证伪试验

**功能 Oracle**：H0 空实现应失败；一个获授权的正确参考应满足；至少一项刻意破坏的具体不变量/兼容行为应触发相应断言。若参考只是受测候选实现，标 `NO_INDEPENDENT_GOLDEN`。

**架构 Oracle**：对同一个有权质量场景，让 Reviewer 判断“拥有真实提交事实的边界”与“跨包双写产生不可恢复中间态”的明确差异；错将两者一律标 CLEAN 的评分器不合格。这里是**公开方法层例子**，不可复制成 R18 隐藏题。

**可读性 Oracle**：预先确定外部行为和目标代码路径，评陌生 Reader 的正确解释、信息缺失与理解成本；文件少/注释多不是自动阳性。

**维护 Oracle**：后续维护变更先于原始实现结果冻结；对每份源检查原合同与新增合同、公共接口/数据影响；变异控制应能检测“新功能看似实现，但旧行为被破坏”。只测新功能不是有效的可维护性评价。

**时间窗/并发缺口**：回读仅在进程退出后发生时，不能声称覆盖成功回复到进程退出间状态不变；并发锁移除只在 1/3 运行被发现，不能自动称其稳定。保留 R4 真实弱 Oracle 的全部原始缺陷、幸存变异和七项限制，**R4 后置 v2 不能晋升原始冻结 QA**。不要求每轮强制模拟断电，真实故障模型仍需项目授权和实际测试环境。

## 4. 漏检与误报同样需要计入

| Reviewer 行为 | 怎样记录 | 禁止的解释 |
| --- | --- | --- |
| 未发现预先存在、可证的架构缺陷 | `FALSE_NEGATIVE_WITH_REFERENCE`，只对拥有独立参考的示例成立 | 没报告 ≠ 没缺陷 |
| 将正常替代方案/个人风格说成重大缺陷 | `FALSE_POSITIVE_OR_STYLE_BIAS` | 不得奖励 Reviewer 提更多 Issue |
| 缺少业务合同导致无法判断 | `EXPECTED_UNKNOWN` | 不把保守合理停止算错 |
| 发现新的等价、正确实现 | 在解盲之前由独立裁定人按同一准则决定 A/B 皆应如何评分 | 不可挑有利的一组重写 Oracle |
| Reviewer 看到了组别或参考答案 | `BLINDING_BREACH`，需要降级/隔离该样本 | 不隐去漏检证据继续宣传强盲 |

**最小结论结构**：`Task/Scenario/Code_SHA/Reviewer_ID/BlindFlag/Finding_ID/ExpectedSource/Falsifier/ActualEvidence/Verdict/Disagreement/NotTested/Cost`。评分器不变，独立性和错误维度必须可重算；缺乏经授权的源真值就不能凭多数投票直接创造 Source Claim。

## R19-F2 反馈候选：先确认 Oracle 有权，再统计 Reviewer 发现率

[R19-F2 真实代码测量](../professional-abstraction-empirical-feedback-r19f2.md) 显示，隐藏断言如果要求明文任务未约定的入参所有权或输入合法域，就会把**合理但不同的合同解读**误当组间模型差异。研究开始前需建立 `SOURCE -> INPUT_DOMAIN -> OBSERVABLE_CONTRACT -> ORACLE` 映射，分 `DECLARED / DERIVED / ENGINEERING_HYPOTHESIS`；最后一种不能决定主要 PASS/FAIL。原始结果应保留，事后只能做有标注的敏感性分析。

在另一个数值极值反例上，同根缺陷出现于多份真实代码，但多数 Reviewer 给出 `JUST_ENOUGH`。发现了新反例后应在**不改原始首判**的前提下，对全部相关匿名候选做对等重放，单列 `POST_REVIEW_FALSIFIER`、`FALSE_NEGATIVE` 与其来源限制，不能用仲裁后的一个总标签声称没有其它缺陷。

## 5. 对过度评估的限制

一个文档链接修正或已明确的局部 bug 不必启用双 Reviewer/复杂 ATAM。只有真正研究**知识增量对软件质量的影响**且 Reviewer 判断成为实验主要终点时，才需要上述独立与校准方案。所有实验组应有相同公开质量目标、预算和 Reviewer 标准；只改变额外 Kxx 知识内容。本篇没有自动启动本地 Agent 或新增 ADS 强制审核门。
