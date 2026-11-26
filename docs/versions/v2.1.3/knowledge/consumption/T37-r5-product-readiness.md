# T37-R5 · 既有产品 L1/L2 差距及下一阶段准入候选

#636 / #563；评估输入 edbe97e099c52f88144f07e5747fe2888a314593。[累计盘点](T37-r5-cumulative-assessment.md)、[原L1/L2](../../engineering-intelligence-l1-l2.md)。本篇为有界只读设计差距核对，不改 #564 已审核历史设计，不修 #565、不冻结新产品设计、不运行或派发消费者。必须区分本篇 T37-R5（知识阶段）与原 #565 R5（实际实施REWORK）。

## 1. 真实产品基线及阻塞

- #563 A1–A7 的目标仍是用真实有权合同、源码事实、候选知识和验证输出工程决策，而非只返回文档/模式；优先首个抽象纵向能力，非全Kxx一次转换。当前知识阶段由 [R6 Owner方向](https://github.com/kaicreator-mm/tally/pull/565#issuecomment-7146968848) 约束：KNOWLEDGE_FIRST / NO_NEW_SOURCE_IMPLEMENTATION；NO_NEW_LOCAL_AGENT_EXPERIMENT=TRUE。#286 DEFERRED，A24/T4 NOT_RUN/HOLD。
- #564 经 [原R3设计复审](https://github.com/kaicreator-mm/tally/issues/563#issuecomment-7146502900) 限定接受，reviewed 06a2d33caad619b7d6e4839b80f0079612b8f697，merge c6d2c8404072f38f2bf946a7f77ce0c712621516。现 main 实际设计Blob 2d045d09c199da2c82d59b2a0ea96518185cae1b、原projection f9657c07a72c134e6d21e6aa7523b9982ffb327f、acceptance-cases bc44c8b88bdc61d5f5593ffcac7dedc9f12f6222；原16概念/3既有关系/3 advisory入口、44个设计用例，不被67/86投影覆盖。原3入口的实际DP-K02 L17–28、DP-K03 L7–38、K05.ORACLE L51–55 与T36的三个原ID引用同源Blob/范围；原 projection 的 sourceSha 3c9241d7a08a39b8a9865b7b7d221339a0f145e3 不因后续文档增长而替换。
- #565 OPEN/unmerged exact 55ea00ba3cdf7936b73c14aaec89f5387030d61f；实际 original-base compare 六新增研究文件+358/-0。对本评估 main 的实际 compare diverged、ahead7/behind259（只针对上述输入时点），PR diff列出同六研究文件，不能以差异少推断可直接合并；任何未来修正先核新main及真实分支、保留原review身份和受审设计，冲突/漂移按真实变化处理。
- 实际只读取原代码 [engineering.ts](https://github.com/kaicreator-mm/tally/blob/55ea00ba3cdf7936b73c14aaec89f5387030d61f/research/v213/engineering-decision/engineering.ts) Blob62f92ddb0808a52bde77a5079d27b435bb4265b7 与 [model.ts](https://github.com/kaicreator-mm/tally/blob/55ea00ba3cdf7936b73c14aaec89f5387030d61f/research/v213/engineering-decision/model.ts) Blob83a2cba461960a62f00532cfc0c20f0376be02eb。engineering.ts L9/16–18/84–96 固定原projection Blob及根有界query，查两个DP或Oracle及conceptRefs；L20–56普通领域判断，L58–76证据判定；L131–142单根Workflow调用。model.ts L4–24定义九维、事实/候选/证据与可信外部上下文；L69–72确认内容比较。实际不是67引用检索器、关系遍历器或LLM工具；不能用文档增长宣称其已消费。保留普通函数与原生SDK、避免另建引擎/通用Rule VM/graph DB/vector/运行时。旧v08-business-abi/1不迁移，不加shim。
- [原实现独审 R5](https://github.com/kaicreator-mm/tally/issues/563#issuecomment-7146842399) 正式 REWORK 2P1/2P2；[原PR Review](https://github.com/kaicreator-mm/tally/pull/565#pullrequestreview-6048711491)；R6实际受控只读全Package探针重现三项，不代表新A24任务。作者57/57及3 mutation是历史执行收据，独审只执行原函数切片而非完整包/SDK重验。本次未执行任何源码/SDK或检索/LLM。

## 2. A1–A7 逐项差距与最小可行路径

| 原验收目标 | 当前知识能提供的选择证据 | 产品差距与裁定 |
|---|---|---|
| A1 有权/冲突/过期 | POLICY、ELICIT、BOUNDARIES、K05.ORACLE 与T36材料身份、缺事实/范围条件 | #565 P1-01、P2-01 OPEN；缺有权真实确认不能从普通输入自认；不因新知识已接受判PASS |
| A2 同语法不同语义 | 两DP、COHESION、INPUT-OWNERSHIP（研究反馈候选）、ERROR-RESOURCE、REPRESENTATION；按实际前后置/状态/权限/错误/资源/消费者/变化选择 | 原九维应保留，不默认增加第十枚举；所有权/别名根据真实合同在适用维度内或补明确证据，先核缺口；代码现有区别仅限定受控观察，不能遗漏已核对冲突 |
| A3 私有共享与保留 | SMELL、两DP、P36-MAINT、HISTORY-CASES；真实多维护位置与共同不变量/Owner/变化，不以LOC/未来猜测 | 现有有限类别及判断结构可复用；需实际源码和反证，未验证新任务 |
| A4 公共合同/Strategy与权衡 | VARIATION、STYLE、ADR、COST、REPLACEMENT、INSTALLED；真实消费者/替换与成本 | P2-02 OPEN，A14无实质成本对照；保留明确已核成本与具体候选反证，不新增通用评分 |
| A5 合格证据与独立Oracle | K05.ORACLE、TESTMODEL、RESPONSIBILITY、INSTALLED；真实有权义务全集/执行/范围；可定向读T28b-1/T28c-1 | P1-02/P2-01 OPEN，R20历史反例保留；不能以测试数/自报/缺执行证明判全通过 |
| A6 真实来源/适用性/反证及有界决定 | T36语义/阶段合同、46/265发现、67/86参考、R4组合的条件/反转；原DP可直接用 | 原产品只查固定projection，尚无实际新投影发现/选择/消费收据；P2-02及A24未决；首个任务可先人工受权提取并固定必要引用，不需全图或自动检索器 |
| A7 真实SDK单根调用 | 原设计与已接受内部0.11.2制品合同、DEPENDENCY/INSTALLED及旧历史SDK smoke | 已有原始收据非当前重验，实际执行前再确认SDK和环境；不得Mock或换新版本冒充。无当前生产集成/发布 |

以上为知识支持与未实施/未证差距，不给 A1–A7 行为PASS；不将完整课程覆盖、实际模型召回、自动图谱推理当原A1–A7必要前提。原设计限定通过不自动解除现行Owner限制；若新需求改变已冻结语义、输入/输出或适用范围，提交明确L1/L2限定增量并独审，而非悄改旧合同。

## 3. 以实际 #565 代码作有界的知识选择核对（只读）

当前真实工程问题是如何修正现有四项已报告发现，而非为抽象追求统一架构。本次重新读实际工程及model源码与独审原记录，引用先前已经发生的 R6 观察，不宣称新运行。原 confirm() 的准确函数名是 model.ts 的 confirmed()；它比较实际已核记录与证据内容，适用于确证内容相同这一纯机制，不替代业务权威、适用事实全集或证据Oracle。原R6对保留此纯机制的限定 SUPPORTED 不意味着所有抽象已正确。下列四项为现行 OPEN，不新增或关闭缺陷：

| 原发现/真实位置 | 新知识与实际最小修正候选（未实施） | 独立反证及正例 |
|---|---|---|
| P1-01 engineering.ts L20–29/L103–110：普通请求省略已确认冲突可SUPPORTED | DP-K02/K03、BOUNDARIES、COHESION、POLICY；保持 trusted 已核适用事实权威，不用普通输入子集重建判定全集；不把通用内容比较与业务判定合并 | 同一受权候选/可信上下文，完整、删除、替换冲突事实均不得提升为SUPPORTED；独立不受影响候选仍可支持，真实不适用或已核冲突解决可反转 |
| P1-02 L68–74：同一执行FAIL+NOT_RUN先报FAIL | K05.ORACLE、TESTMODEL、REPRESENTATION；按实际义务/执行身份/target/env/覆盖范围先消歧，保留独立有效失败；非全局冲突优先 | 同run同cell FAIL+NOT_RUN→CONFLICT；FAIL_ONLY→OBSERVED_FAIL；PASS+NOT_RUN及PASS+FAIL冲突；部分覆盖冲突与另一独立有效反例仍FAIL。记录真实覆盖，不把UNVERIFIED当同级反证 |
| P2-01 L62–76：已确认旧Oracle版本被滤掉后UNKNOWN | K05.ORACLE、ADR、REPLACEMENT、T36来源身份；确认同主体旧合同/版本后单列STALE，不排序任意opaque版本 | 仅确认旧版本且无当前合格观察→STALE、聚合INCOMPLETE；未知版本/无资格→UNKNOWN，当前合格观察按真实范围判定，旧SHA反例保持 |
| P2-02 L27/L45–56/L111–115及model keys：成本差异与反证泛化 | DP-K03、COST、SMELL、MAINT、TESTMODEL；最小保留实际已核可比较成本/证据与候选专属反证，未知不造数；必要时限定L2变更审查 | 两个已证可行方案、真实不同维护/间接性成本；TRADEOFF及实际理由/证据绑定、可执行区分检查；未知成本仍未知，不以数量/权重造唯一最优 |

反证均为原发现派生的前瞻限定测试意图，未执行；当前具体证据以原review及R6真实记录为准，不视为新隐藏题。上述选择不要求把67引用全部加载至Package，也不要求改原投影Hash检查或增加固定图谱API。若首个真实用例需别名/语言/静态检查细节，可定向读取原T25b-3/T28b-1等并确认实际语言合同；尚非新运行时硬需求。先解决现存实际缺陷，比单纯换知识数据格式更重要。

## 4. 下一阶段准入顺序（规划候选，当前不派发）

| 门 | 可观察通过条件 | 当前状态 |
|---|---|---|
| G0 知识阶段综合独审 | 实际本批盘点、范围重叠/覆盖/选择挑战、A1–A7差距与后续准入独立确认；发现的问题限定修正 | REVIEW_PENDING |
| G1 恢复原实现修正权限 | Owner明确解除R6 NO_NEW_SOURCE_IMPLEMENTATION 的**限定范围**，只允许 #565 现有4项修正及必要测试；确认现main/旧HEAD和真实SDK制品可得、评审计划；如需改变原L2先小设计增量独审 | HOLD_OWNER_DIRECTION |
| G2 修正/实际验证/独审 | 同Package最小改动，不引入新引擎/旧ABI或生产接线；当前可验证环境实际编译/语法/完整相关测试及四项区分探针，SDK根入口及根预算；复审绑定新Exact HEAD，4项各CLOSED_SCOPED且无阻断发现，再按当前main核合并 | NOT_RUN/HOLD |
| G3 A24/T4 有权新源码消费 | 单一真实新代码设计任务，预先绑定repo/issue/baseSHA/真实源及支持合同、参与者权限、实际工具链/预算、独立义务全集与Oracle；不是本篇公开例、#565旧反例或R20回放。至少一个实际候选选择与代码行为证据（正例及有实质区分的负例）可独立复核；是否改源仅按该任务权限。未验证范围不得宣称通过 | NOT_RUN/HOLD |
| G4 独立裁定及后续产品设计 | 保留全Attempt分母、实际选知/使用/行为及未决，区分知识定位/选择/推理/实现/Oracle错误；依据证据决定下一专业功能或有界补深。#286生产接线另行明确授权及迁移设计独审，禁止自动Release | DEFERRED |

G3 采用最小有限真实消费者方案：优先在当前可实际访问仓库和真实SDK环境内完成，不要求本地Agent、Windows/CI代办或新一轮A/B；若环境不可获得则 BLOCKED/NOT_RUN，不能Mock。实际任务、模型/参与人、确切预算、源码身份、预期/负例及独立证据由获权后确定，不提前虚构。独立评审者在执行前冻结当前有权义务及独立Oracle，必要时隔离未公开测试输入；不以公开G/B/W36作隐藏题。读者可使用原知识路径而非固定键；实际收据至少记录分配任务与原SHA、真实合同/事实来源、读到的具体范围及拒绝/未访问、候选与适用/反转、实际调用/命令/结果/失败、真实制品及独立验证范围。公开静态核验不能作为A24执行收据。单一真实行为成功只支持限定可行性，不支持因果收益、准确率或节约统计；维护评价若后来获权，再按真实合法维护变化、实际工时/回归与原分母观察；因果才另立独立研究。

G0可与知识设计验收一起完成，不自动使G1–G4通过。#563 不关闭，#565全部2P1/2P2 OPEN/HOLD、#286 DEFERRED、A24/T4 NOT_RUN/HOLD，不改变内部DH阶段或生产权限；不改旧设计/代码、原知识源、历史研究、SDK或Rule，不创建新合格Claim/Case。
