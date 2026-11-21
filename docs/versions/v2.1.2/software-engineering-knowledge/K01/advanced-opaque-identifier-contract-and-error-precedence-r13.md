# K01：公开标识符语义、错误优先级与需求内部一致性（R13 候选）

> [系统课程](systematic-syllabus.md) · [需求追溯](advanced-requirements-traceability-and-change-propagation.md) · [K03 错误实现](../K03/advanced-error-semantics-observability-and-validation-proof.md) · [知识增量 #537](https://github.com/kaicreator-mm/tally/issues/537)
>
> **R13 REPORT_DERIVED_KNOWLEDGE_CANDIDATE / NOT_RUNTIME_API_CONTRACT / NO_SOURCE_CLAIM_QUALIFICATION。** 主要归属 K01.F4.1 规格与 K01.F5.1 冲突判定、跨 K03.F5.1 错误实现。#534 R2 补充报告指出一个需求内部自相矛盾的现象，但 #535 尚未独立复算。此章教学案例完全合成；不在此规定真实 Tally API 错误码。

## 1. 谁拥有标识符的格式？

**Opaque identifier（不透明 ID）** 是对外承诺「这是一个用来引用资源的令牌」而**不承诺其内部编码、UUID/Hash 格式、长度、结构或可拆分字段**的合同。服务内部采用 UUID、base64、自增或复合键，都只是实现选择，**除非有权公开合同显式规定**，不得以内部格式解析器去拒绝公开可接受的 ID。

相反，若公开合同明确规定 `tenantId` 是限制长度和字符集的结构化标识或路径分量，则可以在权限边界先验证该字段。二者**必须分别判定**，不能把「所有 ID 都先用 UUID regex」当通用最佳实践。

关键前提：`INVALID_INPUT`、`NOT_FOUND`、`FORBIDDEN`、`UNKNOWN` 等是**具体产品合同的输出语义**；没有行业通用的「所有未知 ID 一律 NOT_FOUND」规则。外部暴露什么状态必须考虑已验证的 Actor、租户范围、资源存在性是否敏感、当前版本与接口公开承诺。

## 2. 四种不同错误，不能随意互相改名

| 输入/事实 | 可能的公开处置（**须由合同/安全策略决定**） | 为什么 |
| --- | --- | --- |
| 非字符串、空值、超允许大小、非法路径分隔符、恶意控制字符 | `INVALID_INPUT` 或有权定义的拒绝 | 外层类型/大小/路径安全，属于对外输入边界而非内部生成格式 |
| 语法合法且当前项目内没有目标记录的不透明令牌 | `NOT_FOUND`（若合同采用此分类） | 不能让内部存储/生成约定凌驾于公开 opaque 合同 |
| 目标记录存在于**另一**项目、但当前调用者无权查看 | `NOT_FOUND` 或同形拒绝（由安全策略决定），不得泄漏存在性 | 先按调用者有权作用域查找，不能跨租户查到后再返回有差异的明细 |
| 指向已提交操作但远端状态未知的引用 | `UNKNOWN` / 受界回读（由 Effect 合同决定） | 查找失败和外部 Effect 不确定不是一个事实 |

**特别注意**：不能把对象不可猜测当访问授权；每次通过标识符访问对象都应施加对象级授权。错误状态的差异也可能成为枚举侧信道：不要把跨租户存在性以不同错误内容或时延作为不受控观察输出。

## 3. L1 必须自洽：从有权语句到唯一的错误表

一个公开的**教学**接口 `getReport(projectId, reportRef)`：`projectId` 有明确格式限制；`reportRef` 只有最大长度要求，为 opaque token。公开规则是「在当前已授权项目内找不到报告则返回 NOT_FOUND」。

| 规范部位 | 必须相互一致 |
| --- | --- |
| FR / User Case | 哪一类输入、Actor 和资源被调用者接受？ |
| 输入合同 | 哪些字段真的公开了结构化格式？哪些是 opaque？ |
| 错误分类表 | 何时 INVALID_INPUT、NOT_FOUND、无权访问、UNKNOWN？ |
| AC/负例 | 是否包含任意非空但未知的 `reportRef`、不存在项目、跨项目匹配、内部生成格式变化？ |
| L2 设计 | 状态/身份 Owner 和查找路径是否能实现并证明这些语义？ |
| L3 实现/测试 | 真实公开入口是否引用**同一个最终决议**而不是从内部生成函数反向推出规则？ |

**可操作一致性检查**：对每个输入类把 `(Actor, scope, input validity, existence, effect state, policy version)` 展成一条决策行；找出两份文档/表对同一行给出相反 Expected 的情况；无有权 Owner 裁定时标 **CONFLICT/UNKNOWN**，不要静默以更晚的文件或更像规范的那段文字为准。

## 4. 可反驳试验

- 公开 opaque `reportRef` 的编码在内部发生改变，既有客户端对当前合法合同令牌的处理不能因内部迁移而误报 INVALID_INPUT。
- 一个外形不符合内部 UUID 的非空未知令牌，如果合同允许且限制条件满足，应按合同走作用域查询而不是内部解析异常。
- 两项目存在同一引用值时，互相隔离；在未授权项目中查询该值不得泄漏另一个项目的信息。
- L1 用例与错误码表故意放入相反规则，跨表检查必须报告冲突；不能让 CI PASS 掩盖规格自相矛盾。

## 5. 权限、来源与适用边界

- [OWASP API1:2023 Broken Object Level Authorization](https://api-security.owasp.org/editions/2023/en/0xa1-broken-object-level-authorization/)：所有基于对象 ID 的访问仍需对象级授权。
- [OWASP IDOR Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Insecure_Direct_Object_Reference_Prevention_Cheat_Sheet.html)：作用域内检查与不存在/不可访问的公开响应选择；不是「必须都用 404」的跨产品普适规定。
- [OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)：权限应由有效策略及资源上下文控制，不可只依赖难猜 ID。

**STOP**：如果任务没有明确 ID 公开合同或未知错误码的有权 Owner，则先在 L1 形成选择/澄清和反例，不得由本章替 Tally、第三方 API 或用户产品制定规范。
