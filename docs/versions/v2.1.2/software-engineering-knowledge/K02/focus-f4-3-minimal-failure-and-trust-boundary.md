# K02.F4.3｜架构边界的最小威胁与失败模式分析

> [K02 课程](systematic-syllabus.md) · [第二卷](foundations-deep-dive-02.md) · [165 点台账](../foundation-concept-coverage-ledger.md)
>
> **R4 ORIGINAL_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / THREAT_ANALYSIS_NOT_PERFORMED_FOR_REAL_PRODUCT。** 仅深化 K02.F4.3；这是架构设计和现有基本防线的知识，不启动此前延后的独立安全专项/供应链审计，不声明真实应用通过安全审查。

## 1. 失败模式和威胁场景要分开

架构**失败模式**描述部件如何偏离预定功能（如缓存失效、数据库不可达、错误跨租户复用），与是否有恶意行为无必然关系；**威胁场景**还要明确资产/边界、潜在行为主体、进入路径、所需能力和有害后果。一个系统同时可能受到非恶意逻辑错误与恶意利用。最小分析不必凭空给每个系统造出复杂攻击树，但不能因为安全专项延期就不验证明显权限/租户隔离。

给定 Tenant A/B、Agent、共享 Worker Pool、数据库和外部插件，先画出**数据流、信任边界、状态所有者**。例如 Tenant ID 由不可信参数提供时，`cacheKey = taskId` 可能把不同租户的结果缓存到同一键；这既是架构隔离缺陷，也是潜在敏感数据泄漏路径。仅把 Tenant ID 写入日志并不能证明状态隔离。

## 2. 系统性推理：资产→边界→失效→后果→限制

| 对象与信任边界 | 最小失效/威胁 | 受影响不变量 | 候选防范与可观察反例 |
| --- | --- | --- | --- |
| Tenant → API → DB | 客户端伪造资源所有者 | 无权主体不能读写其他租户对象 | 服务端按有权身份验证归属；构造跨租户读/写反例 |
| Package → 声明的 Extension | 未声明依赖偷偷调用 Host | 组合边界依有权 BUILD 合同确定 | 构建侧检查声明闭包；负例为未声明 Import |
| Agent 消息 → 状态写入 | 旧 Attempt 交付迟到或重放 | 过期结果不能替代当前任务决议 | 关联 Attempt/SourceSHA，旧消息拒绝晋升 |
| 缓存/队列 → 不同消费者 | 不同租户或不同 SourceSHA 共用键 | 不能把 A 的结果当 B 的事实 | 区分有效输入与权限范围；断言隔离结果 |
| 运行进程 → 外部 Effect | 超时与重试导致重复写入 | 未证实幂等时不能重复不可逆效果 | 先通过权威 Effect ID 读回；未知保持 UNKNOWN |

表中措施是**候选模型**而非 Tally 已实现的事实。具体 DomainHarness BUILD/Init/Extension 合同只能引用对应已澄清版本，知识材料不自行制造运行时机制。

## 3. 与安全专项的界线

完整威胁建模可涉及多角色攻击者、可信计算基、加密协议、凭据泄露、供应链和系统级威胁分析；R4 仅提供**设计者最低限度的跨界失败意识**，按现有项目风险和权限做适用选择。OWASP 的方法强调资产、数据流和信任边界，但这是一种专业分析方法，不自动成为 ADS/Tally 的固定 Release Gate。若发现实际严重漏洞，走真实项目现行缺陷处理路线，不可因“知识范围延后”自动豁免。

**常见错误**：因为模块边界有接口或 TypeScript 类型，认定可信；因为用户从 UI 看不到其他租户就认定 API 无权访问；因为未开启安全专项就把跨租户数据污染写成 ACCEPTED。

## 4. 原理的反例与可以检验的教学任务

任务描述：两个租户恰好使用相同业务任务编号，一个后台 Worker 从共享缓存读取配置，再向一个远端 Agent 派发工作。要求读者交付：(1) 数据流/拥有者边界；(2) 键和值的身份组成；(3) 三个故障/错误授权序列；(4) 分别说明允许读取/执行/持久化所需的权威事实；(5) 两条可使错误实现失败的反例。不能用“加 JWT”或“加日志”代替边界分析。

**证伪案例**：两个租户各自的正向测试均通过，但先执行 A、后执行 B 时 B 读到 A 的缓存。该错误可能无法被单租户测试发现；需明确缓存 Scope 与隔离不变量并针对交错构造 Oracle。

**STOP/UNKNOWN**：缺少租户身份来源、边界 Owner、受权 Extension 合同或实际数据分区方式时，停止对相关隔离设计自报 PASS；其他独立安全范围之外的内容仍可继续。未真实运行的测试一律 NOT_RUN。

**公开来源线索**：[OWASP Threat Modeling Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Threat_Modeling_Cheat_Sheet.html)、[OWASP Developer Guide](https://devguide.owasp.org/en/04-design/01-threat-modeling/07-practical-threat-modeling/)，另参考 [系统外部来源目录](../external-source-catalog.md)。这些是原理参考，非版权转录或本项目已通过审计证据。
