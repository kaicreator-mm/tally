<!-- tally-v212-public-software-engineering-sources-20261121 -->
# 外部软件工程学科知识与工程实践来源目录（2026-11-21）

> [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [K01–K11](README.md)。**SOURCE_DISCOVERY_AND_CROSSWALK_ONLY / NOT_CLAIM_QUALIFIED**。本目录只记录公开可定位的规范摘要、学科体系、企业工程手册、GitHub 工程项目及研究；并不复制它们的专有原文。当前采用来源优先，所有单独的 Knowledge Claim 仍需逐项确认内容、版本、适用性和允许使用方式。

## 来源等级与取舍

- **国际/IEEE 标准**：规范/术语的权威出版来源，但**标准存在**不意味着 Tally 已取得全文使用许可，也不意味着任何软件项目都要通过该标准审核；可依 ISO/IEEE 公告核对范围和版本，不能未经购买复制完整正文。
- **专业学科/资格体系**：SWEBOK、SEBoK、IREB、IIBA、ISTQB、CMMI 等组织公开知识体系或课程大纲。适用于定义知识覆盖与方法选择，不自动产生 ADS/Tally 的 MUST/Release 权限。
- **研究方法/实证论文**：CMU SEI ATAM/QAW、DORA、SPACE、AI 代码评估等。应保留样本、负结果与应用边界；不能因某研究观察到趋势，就默认某 Agent 完成任务一定更快。
- **真实企业实践**：Microsoft、GitLab、Google 和 AWS/SRE 公开流程。应提炼与目标任务相符的工程思路，而不是克隆企业所有治理/CI/审查细节。
- **GitHub 开源实现**：Testcontainers、Pact、OpenHands、SWE-agent、SWE-bench、Structurizr 等。仓库中的实用样例和可执行合同不等于通用的学科权威；选用代码或模型要按各自许可证与平台实际版本办理。

**权威优先级**：当前用户/任务的有权产品与工程约束、已生效 ADS 标准、合法 Tally Task/Host/Release 状态始终优先；外部标准定义可采纳的知识依据，专业技术指南给出方法选项，社区仓库是可验证实践例子。不能从官网链接或 GitHub Stars 推断“生产知识已合格”。

## 已核对的外部来源（标准、企业、开源、科研）

| ID | 参考来源 | 类别 | 版本/许可与限制 | K 模块 |
| --- | --- | --- | --- | --- |
| A01 | [IEEE Computer Society SWEBOK v4.0a (2025 revision)](https://www.computer.org/education/bodies-of-knowledge/software-engineering) | Academic BOK | IEEE CS consensus, 18 KAs; official download/TOC, not a reusable license to copy text | K01–K11 |
| A02 | [SWEBOK v4 topic index](https://www.computer.org/education/bodies-of-knowledge/software-engineering/topics) | Academic index | V4 chapter topic navigator; check v4.0a update against each claim | K01–K11 |
| A03 | [SEBoK v2.14 (May 2026)](https://sebokwiki.org/wiki/Guide_to_the_Systems_Engineering_Body_of_Knowledge_%28SEBoK%29) | Systems BOK | Interdisciplinary system/engineering management; not a software implementation prescription | K01 K02 K04 K09 K11 |
| A04 | [ISO/IEC/IEEE 12207:2026](https://www.iso.org/standard/90219.html) | International standard | Current software lifecycle processes; explicitly does not require one methodology; prior 2017 edition superseded | K04 K08 K09 K11 |
| A05 | [ISO/IEC/IEEE 29148:2018](https://www.iso.org/standard/72089.html) | International standard | Requirements engineering lifecycle; ISO page says current as of review 2024, successor draft not yet assumed published | K01 |
| A06 | [ISO/IEC/IEEE 42010:2022](https://www.iso.org/standard/74393.html) | International standard | Architecture description structure and stakeholders/concerns; not mandatory C4 or microservices | K02 |
| A07 | [ISO/IEC 25010:2023](https://www.iso.org/standard/78176.html) | International standard | Nine product-quality characteristics; select project relevant attributes, not all required checks | K01 K02 K05 K10 |
| A08 | [ISO/IEC/IEEE 29119-1:2022 / Part 2:2021](https://www.iso.org/standard/81291.html) | International standard | Software test concepts; optional process reference; paid full text not copied | K05 |
| A09 | [IEEE 1012-2024 V&V](https://standards.ieee.org/ieee/1012/7324/) | IEEE standard | Verification/Validation evidence distinctions; task-specific application; full text may be restricted | K05 K08 |
| A10 | [ISO/IEC/IEEE 15939:2017](https://www.iso.org/standard/71197.html) | International standard | Information-needs based software/system measurement; not universal numeric targets | K10 K11 |
| A11 | [ISO/IEC 25023:2016](https://www.iso.org/standard/35747.html) | International standard | Product quality measurement, status/replacement must be rechecked; ISO 25023 DIS underway | K05 K10 K11 |
| A12 | [IREB CPRE Foundation syllabus and handbook](https://cpre.ireb.org/en/downloads-and-resources/downloads) | Professional BOK | Requirements elicitation/analysis/validation; published training materials, rights to bulk copy not presumed | K01 |
| A13 | [IIBA Business Analysis Standard](https://www.iiba.org/knowledgehub/the-business-analysis-standard/) | Professional standard | Need, Stakeholder, Context, Solution, Change, Value; Business Analysis Core Concept Model | K01 K11 |
| A14 | [ISTQB CTFL v4.0.1 (2024)](https://istqb.org/certifications/certified-tester-foundation-level-ctfl-v4-0/) | Professional syllabus | Test analysis/design, techniques, review, risk-based test planning; syllabus content not recopied | K05 K06 |
| A15 | [CMU SEI ATAM and quality attribute tradeoffs](https://www.sei.cmu.edu/library/the-architecture-tradeoff-analysis-method/) | Research method | Evaluates architectural quality-attribute tradeoffs; formal workshop is optional for simple tasks | K02 |
| A16 | [CMU SEI Quality Attribute Workshop](https://www.sei.cmu.edu/library/quality-attribute-workshop-collection/) | Research method | Stakeholder scenarios for key quality concerns, useful before architecture options | K01 K02 K10 |
| A17 | [arc42 architecture template project](https://github.com/arc42/arc42-template) | Practice reference | Structured architecture communication; repo license classification unresolved by GitHub metadata—link, don't bulk copy | K02 |
| A18 | [MADR architectural decision record](https://github.com/adr/madr) | Practice reference | Decision context/options/consequences templates; repo license metadata unresolved—avoid template verbatim copying | K02 |
| A19 | [Simon Brown C4 Model](https://c4model.com/) | Author-led practice | Context/container/component/code architecture visualization; use selectively | K02 |
| A20 | [Microsoft Code-With Engineering Playbook](https://github.com/microsoft/code-with-engineering-playbook) | Enterprise practice | GitHub docs license CC BY 4.0; check separate source-code licenses and attribution if reusing content | K03 K04 K05 K07 K08 |
| A21 | [Google Engineering Practices (archived)](https://github.com/google/eng-practices) | Historical enterprise practice | Archived repository (2025); README states CC BY 3.0 for documents; not live Google policy | K03 K05 K07 |
| A22 | [Google Style Guides](https://github.com/google/styleguide) | Language practices | Language-specific examples; project toolchain/style owns current rule; license per files | K03 |
| A23 | [GitLab Engineering Workflow & Code Review](https://handbook.gitlab.com/handbook/engineering/workflow/) | Living enterprise practice | Public engineering process; not a binding rule for Tally's native GitHub/Host workflow | K04 K07 K08 |
| A24 | [GitLab modular code principle](https://handbook.gitlab.com/handbook/engineering/development/growth/modular_code/) | Enterprise practice | Vertical task slicing and unnecessary layer handoffs; adapt to real interfaces/write sets | K04 K07 |
| A25 | [Testcontainers for Java](https://github.com/testcontainers/testcontainers-java) | Executable practice | MIT license; shows disposable real dependencies; Java examples are not universal API for other languages | K05 |
| A26 | [Pact contract specification](https://github.com/pact-foundation/pact-specification) | Executable specification | MIT license; consumer-provider contract model; not full E2E proof | K02 K05 |
| A27 | [SWE-bench original and evaluation guide](https://github.com/SWE-bench/SWE-bench) | Agent evaluation research | MIT repo; benchmark is issue→patch→test, not product lifecycle/Release qualification | K05 K06 K11 |
| A28 | [SWE-agent](https://github.com/SWE-agent/SWE-agent) | Agent research/practice | MIT repo; useful Agent repair workflow and tool environment reference, not universal normative recipe | K06 K07 |
| A29 | [OpenHands Software Agent SDK](https://github.com/OpenHands/software-agent-sdk) | Agent implementation | MIT repo; event/workspace/tool/agent architecture as study, not Tally native API contract | K07 |
| A30 | [OpenHands benchmarks](https://github.com/OpenHands/benchmarks) | Agent evaluation practice | MIT repo; real evaluation harness patterns; selected benchmark may have separate data rights | K07 K11 |
| A31 | [Google SRE Books & Workbooks](https://sre.google/books/) | Open professional practice | Incident analysis, monitoring, SLO/SLI, change/recovery; not mandatory SRE team for all products | K06 K08 K10 |
| A32 | [AWS Builders' Library idempotent APIs](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/) | Engineering practice | Retry/duplicate-side-effect semantics; AWS system context, adapt to actual API contract | K03 K06 K10 |
| A33 | [DORA Research & capabilities](https://dora.dev/research/) | Empirical research | Software delivery flow, stability and sociotechnical conditions; observational metrics not proof of any single code change | K08 K10 K11 |
| A34 | [DORA 2025 AI-assisted development report](https://research.google/pubs/dora-2025-state-of-ai-assisted-software-development-report/) | Empirical research | AI can amplify existing conditions; findings not guaranteed individual productivity lift | K07 K11 |
| A35 | [Martin Fowler Evolutionary Database Design](https://martinfowler.com/articles/evodb.html) | Author-led practice | Schema evolution and incremental migration; no assumption all migration reversible | K09 |
| A36 | [OpenTelemetry signals](https://opentelemetry.io/docs/concepts/signals/) | CNCF project documentation | Metrics/logs/traces concepts; not mandate to install backend or new APM | K10 |
| A37 | [CMMI Model Quick Reference](https://cmmiinstitute.com/resource-files/public/cmmi-model-quick-reference-guide) | Process maturity reference | Capability areas and practice grouping; not an automatic audit or 11 required reviews | K04 K08 K11 |
| A38 | [Scrum Guide 2020 official](https://scrumguides.org/download.html) | Agile framework | Product/backlog/increment feedback, optional pattern; not universal Tally task model | K04 |
| A39 | [OMG Essence Kernel (v2.0 beta 2026)](https://www.omg.org/spec/Essence/About-Essence) | Formal and beta standards | Method vocabulary/kernel for comparison; current 2.0 beta not a mandatory adopted norm | K04 K11 |
| A40 | [SPACE developer productivity paper (2021)](https://www.microsoft.com/en-us/research/publication/the-space-of-developer-productivity-theres-more-to-it-than-you-think/) | Peer-reviewed research | Five-dimension productivity framework; cannot infer productivity from only token/PR/activity | K11 |
| A41 | [Software Engineering at Google book (2020)](https://abseil.io/resources/swe-book) | Book | Public online but CC BY-NC-ND 4.0 digital copy: reference/link only, no adapted republication | K03 K07 K09 K11 |
| A42 | [SEBoK Software Engineering related discipline](https://sebokwiki.org/wiki/Guide_to_the_Software_Engineering_Body_of_Knowledge_%28SWEBOK%29) | Related discipline | System/software boundary comparison; latest SEBoK site v2.14, underlying SWEBOK references may be historical | K02 K04 K09 |
| A43 | [Software Engineering: Standing on the Shoulders of Giants](https://www.swebook.org/) | Open educational project | Open textbook with attributed Creative Commons terms; check exact edition/license before reuse | K01 K02 K03 K05 K08 K09 K11 |

## 本轮追加的学科与实践来源（来源发现，非逐条 Claim 资格）

| ID | 来源 | 类别 | 适用限制 | 对应模块 |
| --- | --- | --- | --- | --- |
| A44 | [NASA Software Requirements Validation Handbook (SWE-055, Rev D)](https://swehb.nasa.gov/spaces/SWEHBVD/pages/102695440/SWE-055%2B-%2BRequirements%2BValidation) | Government engineering handbook | Published NASA project high-assurance context; use requirements verifiability/traceability insight, not universal Tally mandatory Gate | K01 K05 |
| A45 | [NASA Delivery Requirements Verification (SWE-194, Rev D)](https://swehb.nasa.gov/spaces/SWEHBVD/pages/102695529/SWE-194%2B-%2BDelivery%2BRequirements%2BVerification) | Government engineering handbook | NASA mission-specific delivery assurance reference; check current NASA handbook edition for exact clause; do not transplant certification process | K05 K08 |
| A46 | [CMU SEI A Playbook for Early Architecture Analysis](https://www.sei.cmu.edu/library/a-playbook-for-early-architecture-analysis/) | Research/practice method | Uses quality-attribute scenarios and lightweight architecture analysis, not universal requirement to run a formal ATAM workshop | K02 |
| A47 | [Martin Fowler Strangler Fig Application](https://martinfowler.com/bliki/StranglerFigApplication.html) | Author-led engineering practice | Incremental legacy modernization; careful about temporary routing and data coexistence; does not imply every refactor needs façade | K02 K09 |

## 版本与版权的关键异常

1. **IEEE SWEBOK**：组织官网列 18 个领域，2025-09-25 发布标为 **v4.0a** 的小修订；早期 v4.0 目录可能未与小修订完全同步。保存目录/概念映射，不冒充 IEEE 已批准的全文知识重发布。
2. **ISO/IEC/IEEE 12207**：正式 **2026 版** 已发布并取代 2017 版；同时 ISO 官方说明该标准不强制单一软件开发生命周期/方法。不要把旧的 2017 版标为最新，也不要据此要求 Tally 新建固定三阶段编排器。
3. **ISO/IEC/IEEE 29148**：可核对的正式 2018 版经 2024 复核仍有效，但官方提示新的 DIS 正在推进；不得把未发布 DIS 当正式版。
4. **SEBoK**：2026-05-18 **v2.14** 已发布，适合作为相关系统工程/复杂系统学科补充，而不将软硬件系统工程混为单一开发 Task 模型。
5. **Google Engineering Practices**：实际仓库已 Archive，README 明确文档采用 CC BY 3.0；可作为历史成熟 Code Review 做法，**不是**仍活跃更新的 Google 现行规范。**Software Engineering at Google** 在线版为 **CC BY-NC-ND 4.0**，只做链接与自写的学科思想映射，不复制或改编书中大量段落。
6. **Microsoft Engineering Playbook**：仓库文档 CC BY 4.0，源码另有许可说明；引用具体流程概念，不在本知识库复制完整企业手册。
7. **GitHub**：未识别许可证的 arc42、MADR 等仓库，只允许按链接研究已公开方法；不得把 `NOASSERTION` 误判为公共领域。实际用于源码/模板前继续核查对应文件明确授权。
8. **研究/学科书籍**：ISO/IEEE、IIBA、ISTQB、SEI、SWE-book 等各自有不同引用与复制条款；本知识库以准确链接、独立措辞、最小案例和原始主张定位为主。

## 知识接入时的最小来源记录

每项后续真实知识 Claim 只需对现有文档形成一行：`Module/Known Question`、`Source-ID + URL/Edition`、`Claim summary (original words)`、`Applicability/Counterexample`、`Normative ADS owner if any`、`License/reference-only`、`Qualification state`。**来源清单并不等于已对每项专业主张进行独立原文/Clause 审核**，引用不可访问或许可不明时保留 UNKNOWN。除确实需要被 Task 使用的选定小片段外，不批量建立新的 Claim、图谱实体或流程 Gate。

## 新发现的相关学科不增加第十二模块

除狭义编程和测试之外，当前发现的重要相关学科还有：**系统工程/复杂系统**（K01/K02/K04/K09）、**业务分析与需求工程**（K01）、**架构评估与质量建模**（K02/K10）、**过程/配置/生命周期**（K04/K08/K09）、**软件度量、实证研究与工程经济学**（K11）、**社会技术协作与人机交互式开发**（K07/K11）。软件安全/供应链和代码库逆向分析维持用户明确延后；SWEBOK 体系涉及这些部分并不表示 Tally 必须建设对应专项。

## 最小下一步

每个 `Kxx/` 的新增 `disciplinary-foundations.md` 提供**原创的学科概念/机制/反例**；`external-practices.md` 提供**来源可核验的工程做法与 Agent 可证伪任务案例**；既有 `sources.md` 记录本模块 Axx 来源和资格状态。上述均不自动引入生产 Package/Host 执行或新治理系统。
