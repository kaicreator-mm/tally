<!-- tally-v212-bok-and-related-disciplines-map-20261121 -->
# SWEBOK 18 知识领域与 Tally K01–K11 的学科覆盖映射

> [Tally v2.1.2](README.md) · [外部来源目录](external-source-catalog.md) · [IEEE SWEBOK 正式章节](https://www.computer.org/education/bodies-of-knowledge/software-engineering)（v4.0a；2025-09-25 修订）。**ACADEMIC_COVERAGE_MAP / NOT_KNOWLEDGE_QUALIFIED**。本映射是 Tally 适用范围设计，**不是** IEEE 官方对 Tally 的认证或 SWEBOK 全文改写。

## 一、SWEBOK 十八个 KA 的当前归属

| SWEBOK 章节 | 官方领域名称 | Tally 的知识归属 | 本版本适用性与边界 |
| --- | --- | --- | --- |
| 1 | Software Requirements | K01（主），K02/K05 | 需求获取/规格/验收及冲突；不自动制造 Product Authority |
| 2 | Software Architecture | K02（主），K04/K10 | 架构驱动/接口/状态与质量权衡；ATAM/QAW 仅适用时研究 |
| 3 | Software Design | K02/K03 | 详细设计、状态/异常/结构方案；不与正式 L2/L3 责任混淆 |
| 4 | Software Construction | K03（主），K05/K06 | 算法、API、实现、类型检查与错误/回归 |
| 5 | Software Testing | K05（主），K06/K08 | Oracle、测试选择、受影响执行与 Mock/真实区分 |
| 6 | Software Engineering Operations | K08/K10，K06 | 构建、安装、配置、观测、恢复与运行质量 |
| 7 | Software Maintenance | K09（主），K06/K10 | 重构、兼容、迁移、修复、退役与现有消费者 |
| 8 | Software Configuration Management | K04/K08/K09 | 版本/构建/来源/变更与受控交付；不另建任务权威 |
| 9 | Software Engineering Management | K04/K07/K11 | 工作组织、资源、协作与真实成本/交付目标 |
| 10 | Software Engineering Process | K04/K08/K11 | 生命周期适配、过程反馈；2026 ISO 12207 不指定单一方法 |
| 11 | Software Engineering Models and Methods | K01/K02/K03/K04 | 模型、不变量/条件/分析与过程形态，统一三 DAG 候选不是已发布规则 |
| 12 | Software Quality | K05/K10/K11，K02 | 质量属性、V&V、必要风险与可观测结果 |
| 13 | Software Security | **用户明确暂缓单独建设** | 仅复用当前实际工程已生效的最小权限/安全/保密底线，不扩本专项 |
| 14 | Software Engineering Professional Practice | K07/K03/K11 | 独立审查、协作/沟通、责任边界和反馈 |
| 15 | Software Engineering Economics | K11（主），K04/K08 | 全工程成本、方案取舍、实证/机会成本 |
| 16 | Computing Foundations | K03/K02/K08（按任务选择） | 数据结构、算法、操作系统/网络/数据库等只取真实问题所需 |
| 17 | Mathematical Foundations | K02/K05/K11（条件性） | 逻辑/不变量、统计、复杂度/估计；不强制高级形式化证明 |
| 18 | Engineering Foundations | K02/K06/K10/K11 | 实验、采样、根因、测量、风险和系统思维 |

**现状判断**：Tally K01–K11 按 Agent 实际工程问题组织，故一个 SWEBOK KA 可由多个模块承担，不会形成“一个 KA=一个 Tally Issue”的强制一一映射。**SWEBOK 章节覆盖本身仅证明找到了学科主题归属，不证明该模块知识主张已全部通过来源资格**。K01–K08 优先覆盖实际软件开发最小路径，K09–K11 完成知识文档但条件性消费。

## 二、相关学科补充（不增加新的知识模块）

| 相关学科 | 可信参考 | 对 Tally Agent 决策的补充 | 不能据此推断 |
| --- | --- | --- | --- |
| Business Analysis / Domain Modeling | [IIBA Business Analysis Standard](https://www.iiba.org/knowledgehub/the-business-analysis-standard/)、[IREB CPRE](https://cpre.ireb.org/en/downloads-and-resources/downloads) | K01 的 Need/Value/Stakeholder/规则与 UNKNOWN | 业务分析方法自己就获得 Product Freeze |
| Systems Engineering / Systems Thinking | [SEBoK v2.14](https://sebokwiki.org/wiki/Guide_to_the_Systems_Engineering_Body_of_Knowledge_%28SEBoK%29) | K01/K02/K09 的系统范围、跨组件生命周期与外部环境 | 所有软件项目都要采用大型系统工程文书 |
| Software/System Architecture Research | [ISO 42010](https://www.iso.org/standard/74393.html)、[CMU SEI ATAM](https://www.sei.cmu.edu/library/the-architecture-tradeoff-analysis-method/) | K02 的 stakeholders/concerns、质量冲突和可证伪 Demo | 画图或单个 Demo 就代表架构/Release 资格 |
| Empirical Methods & Measurement | [ISO 15939](https://www.iso.org/standard/71197.html)、[SPACE](https://www.microsoft.com/en-us/research/publication/the-space-of-developer-productivity-theres-more-to-it-than-you-think/)、[DORA](https://dora.dev/research/) | K11/K10 的测量目标、基线、样本、变量控制和成本/质量 | PR 数或一个问答得分就是开发生产率 |
| Software & Systems Quality | [ISO 25010](https://www.iso.org/standard/78176.html)、[IEEE 1012](https://standards.ieee.org/ieee/1012/7324/)、[ISTQB](https://istqb.org/certifications/certified-tester-foundation-level-ctfl-v4-0/) | K05/K10 的质量特征、测试与 V&V 条件 | 质量标准标题等同自动测试已运行 |
| Development Process / Project Management | [ISO 12207:2026](https://www.iso.org/standard/90219.html)、[Scrum Guide 2020](https://scrumguides.org/download.html)、[CMMI](https://cmmiinstitute.com/resource-files/public/cmmi-model-quick-reference-guide) | K04/K08/K11 的有界阶段选择、增量交付与过程改进 | 需要在 Tally 再建 Scrum/CMMI 状态机 |
| Human Factors / Socio-technical Coordination | [GitLab Engineering](https://handbook.gitlab.com/handbook/engineering/)、[Google Engineering Practices archive](https://github.com/google/eng-practices) | K07 的角色/独立审查/反馈与 K11 协作成本 | 在 GitHub 收到评论就算 Agent 真的完成执行 |
| Computing / Math / Engineering Foundations | [SWEBOK 16–18](https://www.computer.org/education/bodies-of-knowledge/software-engineering/topics)、[SEBoK](https://sebokwiki.org/wiki/Main_Page) | 仅在 K03 算法/并发、K05 逻辑/统计测试、K10 量化诊断、K11 工程经济决策实际需要时选择 | 所有 Task 必须做完整形式化证明/统计模型 |

## 三、跨学科问题如何选择知识，而不膨胀工作流

**示例：用户要新增“异步批量订单导出”，过程中不能丢进度，要求错误时可恢复。**

1. **Business Analysis / K01**：确定业务角色、允许执行的订单状态、输出格式、取消/部分失败语义；未批准的批量退款/取消政策保持 UNKNOWN。
2. **Architecture / K02**：根据恢复/状态所有权/部署约束，比较同步、内存队列、持久化 Job，并明确真正的验证场景；测试吞吐的 Demo 不能证明重启恢复。
3. **Process / K04**：有一个确实会改变架构的前置问题时使用已有 L2 Research/Demo；接口稳定后才组织实际 implementation Task Dependency；候选 DAG-A 不被自动提升为正式工作图。
4. **Construction / K03**：根据已批准合同选择幂等键/事务边界/错误回复，并在允许的源码范围内实现。
5. **Testing / K05**：对正确导出、非法状态拒绝、重复请求、重启/部分失败设计不同测试 Oracle，真正确认目标 tests 被运行。
6. **Operations / K08/K10**：运行适用 Build/CI/真实运行与错误恢复检查，保持源码/环境/制品的当前性。
7. **Empirical Evaluation / K11**：只有有对照任务和真实成本/正确性收据，才说知识辅助是否有收益；使用用户期望的质量作为主结果而不是单 Agent token 节省。

**学科知识选择**不等于必须完整执行上述七步的文档工作。已明确的小改动可以直接按当前 ADS Fast Path 完成，只有真正材料性风险才增加研究/验证，严禁以新知识库制造第二套 Product/Review/Host/Release 体系。

## 四、软件工程知识丰富度下一步的判断标准

从“文件中出现专业名词”推进到“Agent 真的能决策”，至少需要：
- 学科主张绑定**准确出版方/版本/适用条件和反例**；
- 对应 Decision Procedure 可从真实输入得出可操作的备选方案/停止条件；
- 至少一条错误输入会在测试/业务结果中暴露可观察失败；
- 真实任务的 Agent 结果与相同条件下基线相比有可观察差异；
- 适用的 ADS、GitHub/Host、产品设计和知识引用有清晰且不重复的权限归属。

本映射不是承诺全覆盖 SWEBOK 或相关学科教材；如项目不需要某计算/数学/工程基础，允许只保存外部来源与简单适用标记，不消耗 Agent 的工作上下文和 CI。
