# T25c-3 · 协议版本、能力协商与混合消费者

#620 / R4-B / FM-07；K03 主责协议实现，K02 消费者与授权边界，K09 演进，K01 业务语义，K05 验证。承接 [T25c-2](T25c-2-network-transport-and-protocol.md) 分帧/应用协议、[T32b-1](T32b-1-data-migration-and-mixed-consumers.md) Proto3 未知字段与迁移、[T28a-2](T28a-2-consumer-contracts-installed-artifacts.md) 实际消费者合同。不重复基础编码，不把所有协议都假设有协商。新增 Kafka 4.1 实际 API 协商机制及独立的应用消息演进演练；所有教学与 Oracle NOT_RUN，非真实产品或新 qualified Case/Claim。

## 1. 五种版本与能力不能互相替代

| 事实 | 实际决定什么 |
|---|---|
| 运行制品/客户端/服务端版本 | 具体实现及支持范围，不是单一协议版本 |
| 连接上的 API key + version | 请求/响应线格式及实际可调用 API；某 API 可有独立版本 |
| 具体功能能力 | 即使 API 可解析，具体语义仍可能依服务配置、功能启用及资源约束 |
| 业务消息 schemaVersion | 应用负载的编码、字段含义及迁移策略，不等于 broker API 版本 |
| 当前授权及业务政策版本 | 是否有权执行；握手、解析或兼容成功不授予权利 |

兼容应说明实际消费者/提供者、方向、版本、是否经中转、是否保真、是否允许功能降级。客户端版本大于服务端版本并不必然失败，也不必然支持新功能。不能只用 semver 大小、HTTP 200、解析成功或某 API 存在宣布整个系统兼容。

## 2. 实际 Kafka 4.1 协议协商

官方协议 Compatibility / Retrieving Supported API versions：API key 与 version 共同确定请求格式。客户端建立连接（TLS 启用时先完成 TLS）后可发 ApiVersionsRequest；返回各 API 的支持范围。对具体 API 选双方共同支持的最高版本；没有共同版本报告错误，而非猜测另一种编码。一般响应按请求版本；查询版本本身有特例。若 ApiVersionsRequest 版本不被支持，且 broker ≥2.4.0，按文档返回 v0 响应及 UNSUPPORTED_VERSION，并提供该 API 的支持范围，客户端再选适用版本重试。早于 0.10.0.0 的 broker 不支持查询，可能忽略或关闭连接；这段历史协议说明不是 Java client 4.1.2 支持这些 broker 的承诺。

同一连接获得的版本表只对该连接有效；断开后再获取，不能把 broker A 的缓存套给 B，也不能在升级/降级后继续用旧表。SASL listener 上可能在 SASL 认证前取得版本范围，TLS listener 仍先满足 TLS 握手；ApiVersions 成功不是认证或资源授权。文档还说明 flexible protocol 的 tagged fields 可不提升版本加入，未知 tag 接收方会忽略；要求接收方理解并执行的新语义不应只放未知 tag。实际是否支持 flexible 版本及具体 field 必须查该 API 的 schema；不能把任意 payload 扩展都当 Kafka tagged field。Java producer 4.1.2 API 说明调用不支持的 API 会有 UnsupportedVersionException；不擅自把必需操作降级为更弱动作。

示意（**假设整数区间，不是真实任何 ApiKey 版本表**）：
- 客户端 X 支持 API A [2,5]，broker B1 广告 [0,3]，交集 [2,3]，选3；另一个 broker B2 [4,6] 则选5。连接迁移后重算。
- X 与 B3 [0,1] 无交集，停止依赖 A 的路径、提供可行动错误；未获权不能回退到会改变业务结果的 API。
- 另一个 API C 需要版本≥4 才表达必需的教学功能 F（此为假设），双方共同最高3：A 的兼容与 C 的格式可解析都不能证明 F 能用；若 F 是硬合同则拒绝受理或等待实际支持，若 F 可选且获权降级，明确报告未启用并按旧合同运行。

能力协商是对端声明与实际条件的选择，不是任意客户端输入可改变安全/业务策略。一次认证后仍须确认当前资源授权；会话/版本/许可变化必须按适用合同重查。无协商的固定文件格式可直接用 magic/version 明确选择 parser，未知必需版本拒绝；无可信版本声明不得尝试解析到“看起来成功”为止。

## 3. 应用消息演进：把 broker 与 payload 分开

复用 [T27c-3](T27c-3-reliable-messaging-outbox-offsets.md) 的 D0 文档事件。假定旧应用 v1 支持完整快照事件，字段 tenant/docId/version/eventId/body；新应用 v2 可增加可选 displayHint（只影响合法展示）及新的受限可见性 visibilityPolicy（必需决定披露资格）。以下不是 Kafka wire schema，而是教学应用合同。所有事件必须有权威来源、稳定身份及确定的字段语义；旧消费者即使解析 v2 bytes 也未必理解新政策。

| 变化 | 可选处理与必要条件 | 必须避免 |
|---|---|---|
| 添加可选展示提示 | 若旧读者忽略不改变业务，且中转不需要保留提示，可有权降级 | 将可选提示误作权限 |
| 添加必需 visibilityPolicy | 只给能够理解且真正落实此政策的消费者；未知必需版本/功能拒绝或隔离；可用独立版本 topic/路由或显式应用能力注册并验证；不能以 Kafka ApiVersions 代替 | v1 忽略字段后披露内容 |
| 旧字段重新解释（如 version 从文档版本变为时间戳） | 新字段/新语义版本，显式转换和受权迁移；不能证明等价就拒绝 | 字段类型仍是整数就认为兼容 |
| 旧代理转发 v2 消息 | 若要求保真，验证原始二进制转发或实际保留未知字段机制；ProtoJSON/逐字段重建可能丢失未知字段（见 T32b-1） | 仅验证旧代理能 parse |
| 恢复保留期内旧记录 | 旧 schema/语义仍须有实际 reader 或有权迁移，按旧合法合同解释；核当前执行权，不将旧授权当现行授权 | 只按当前最新 schema 或只靠旧签名判定 |

应用能力若需协商，应定义可信 consumer identity、可支持 schema/必需能力、连接或注册的有效期及拒绝条件；注册成功不是实际支持的证明，须由版本化消费者合同测试和真实运行入口验证。消息日志是持久存储，不像单一请求响应连接；发布前知道当前在线 v2 并不能证明所有离线/重放的 v1 已退役。发布者若不能确保 v1 不读取敏感内容，不能把新 policy 附带在仍可被 v1 消费的同一不安全路径。授权系统另维护当前可读范围；消息 payload/索引不能替代它。

## 4. 兼容方向、转换损失与可反驳演算

教学原始 v1 事件 E1={version:7, body:B}（其余身份省略）；v2 E2={version:8, body:B2, visibilityPolicy:restricted}。v1 → v2 读者只有在有权旧缺省政策明确时才能解释历史 v1，不默认 public。v2 → v1 读者对可选展示提示可忽略，但 restricted 若旧消费者不能落实，必须阻断该旧路径；不能写一个转换器删除 policy 并声称兼容。若新 v2 消费者经过 v1 中转，要求 E2 的必需策略和 eventId 原义保持；丢字段即失败，即使末端 v2 能解析。若真正存在无损转换，记录源/目标 schema、前置、字段语义、稳定身份和可观察逆向验证；若仅投影转换且有损获权，则区分原始权威记录与目标视图，明确不能用于逆向恢复。

升级选择：先扩展读者/路由、独立验证旧/新/中转组合，再允许新必需语义；仍有旧消费者时隔离其能力不足路径或继续旧合法业务。若业务要求立即生效，先阻断不安全旧路径，而不是等待所有升级同时仍披露。回滚程序并不能撤销新语义已产生的状态/效果；有真实旧消费者复活风险时保留限制或拒绝该版本。退役根据实际支持合同、消费者登记、在途及可重放旧数据，而非“近日没访问”。这属于 [T32a-1](T32a-1-maintenance-refactoring-replacement.md) 的消费者退役及 T32b-1 的演进边界，不创建统一版本发布政策。

## 5. 独立区分性 Oracle（NOT_RUN）

| 测试 | 独立预期 |
|---|---|
| 新/旧 broker、连接断开重连或不同 broker | 实际捕获所选 API key/version 与该连接真实广告；共同版本最高、无交集拒绝，不能用前连接缓存；版本查询特例的 v0 UNSUPPORTED_VERSION 路径按实际版本核 |
| TLS/SASL 与版本查询 | 合法协商不替代认证；无权资源操作实际拒绝；失败不因重试自动换弱认证 |
| 必需能力 F 不在共同版本 | 阻止依赖 F 的业务提交，不把解析成功、低版本返回成功当等价；可选降级须有权且输出明确 |
| v1→v2、v2→v1、v2→v1 中转→v2 | 独立预期字段/政策/身份/合法行为；缺失必需策略拒绝，未知可选字段按合同；验证保真或明确获权损失，不以字节相等作为所有语义的唯一判断 |
| 旧消费者离线后恢复、保留历史重放 | 旧版不得读/产生未获权效果，旧 schema 正确迁移或拒绝；不以当前在线列表证明已退役 |
| 不支持消息 schema 与 offset | 不进入目标有效业务事务；按照 T27c-3 合法未决或持久隔离规则推进，不因解码错误静默跳 offset |

测试须固定具体客户端/服务端/编解码库/实际制品、配置及能力版本，实际真实消费者执行；Mock 仅证明 Mock 范围。通信格式、业务资格、数据保真、持久效果、恢复各自独立观察。所有示意区间/字段为原创教学，未运行真实混合部署。

## 6. 实际资料与限制

2026-11-26 实际读 [Kafka 4.1 Protocol](https://kafka.apache.org/41/design/protocol/) Compatibility / Retrieving Supported API versions / SASL Authentication Sequence L235–266；[KafkaProducer 4.1.2 API](https://kafka.apache.org/41/javadoc/org/apache/kafka/clients/producer/KafkaProducer.html) L44–50、78；[KafkaConsumer 4.1.2 API](https://kafka.apache.org/41/javadoc/org/apache/kafka/clients/consumer/KafkaConsumer.html) L25–51、78–123。只证明该版官方文档的明确机制；未读取全部 API schema、实现源码或完整版本兼容矩阵。Proto3 未知字段限定复用已接受 T32b-1 的既有资料，不声称本次重新验证原规范及所有 runtime。应用消息方案、数值及失败例原创教学 NOT_RUN。FM-07 扩展目标候选待独审，不宣称所有协议/平台覆盖；无新合格 Claim/Case。
