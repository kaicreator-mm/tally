# T28a-2 · 真实消费者、接口合同与安装制品验证

#602；K05 主责，K08 制品、K03 实现、K02 接口。深化 QA02/03、OP02；复用 [T28a-1](T28a-1-test-design-stateful-isolation.md) 的独立模型/Fixture、[T31a-1](T31a-1-compilation-linking-abi.md) 的链接/入口身份、[T32b-1](T32b-1-data-migration-and-mixed-consumers.md) 的混合消费者、[K05 V&V](../../../v2.1.2/software-engineering-knowledge/K05/advanced-verification-validation-and-oracle-qualification.md)。不以工具名决定测试层；以下全部教学例和 Oracle NOT_RUN。

## 1. 按失败机制分配验证责任

| 验证对象 | 必要观察 | 不能由此推出 |
|---|---|---|
| 纯函数/域组件 | 输入、结果、状态转移与独立业务 Oracle | 网络、部署或安装正确 |
| 消费者 adapter + 协议替身 | 真实 adapter 序列化、请求、状态码处理、领域输出 | provider 真能响应或持久化 |
| provider 合同验证 | 独立准备前置状态；实际请求经过真实服务入口，核响应 | 所有消费者都已覆盖、业务 Effect 正确 |
| provider 功能/集成测试 | 权威数据、状态变化、事务及禁止 Effect | 真实消费者在全部环境都能使用 |
| 已安装制品的真实消费者 | 干净外部环境、公开入口、实际依赖、行为及负例 | 完整生产拓扑、吞吐、安全认证 |
| 组合端到端 | 实际支持链路、业务观察点及恢复 | 未测版本、错误空间与任意并发交错 |

选择能区分缺陷且成本合理的最小层，按风险增加少量跨边界验证；非固定测试比例。一个测试可以覆盖多个责任，但必须明确观察依据，不能仅把测试名称算覆盖。接口 schema 约束数据形状，交互例限制具体请求/响应，业务性质约束允许历史；三者不是替代。

## 2. 消费者驱动交互合同

以 Pact 的公开说明为例，消费者真实 client 对协议替身发请求并断言其业务可见解析；产出交互合同。提供者准备合同声明的独立前置状态，回放请求，对照响应。不同交互独立，不依赖前一个测试先创建状态。不要用调用替身的手写请求代替真实 client，不要在验证中 mock 被测 provider 的路由/序列化逻辑。若外部依赖被替换，须记录被替换范围及缺少的真实验证。

教学库存 API：消费者以 GET /items/A?warehouse=W1 获取数量；已知存在返回 200 和 {id:"A", available:7}，已知不存在返回 404 和合同错误码 NOT_FOUND。消费者将 404 映射为不存在；网络超时或 503 映射 UNKNOWN，不能把错误当零。Pact 交互覆盖真实请求路径、查询、必要 header、响应和解析；库存数的实际计算、同一货品跨仓隔离、权限、数据库故障等由相应真实功能/集成用例验证。provider state “W1/A 有7件”须建立能被实际路由读取的状态，不能把预期 JSON 直接从 handler 返回来通过测试。

选择 matcher 的原则：固定由合同赋予语义的值（错误码、请求身份、必要单位），放宽合法变化（未知字段允许时的附加字段、真实随机 ID 的已声明格式）；数量的整数类型并不保证非负或业务正确。Pact Matching 文档以 Ruby API 示范类型/格式匹配，实际工具语言/规范版本需另核。若消费者必须拒绝未知字段，不能仅依据宽松响应 matcher；另用真实消费者对额外字段做负例。数组空、非空及顺序/重复依合同；一次非空匹配不推出空列表行为。消费者与 provider 的实际版本及合同内容身份必须一起记录。

## 3. 演进方向、组合和未覆盖状态

教学版本矩阵：C1 读取 available:number，C2 读取 available:number 与 unit:"piece"；P1 只有 available，P2 同时提供两个字段，P3 只有 quantity/unit（不支持旧字段）。假设旧消费者容许附加字段且无其它变化：
| 组合 | 纸面判断（非已测） | 原因 |
|---|---|---|
| C1/P1 | 候选兼容 | 原字段存在 |
| C1/P2 | 候选兼容 | 新字段可忽略 |
| C2/P1 | 不兼容 | 缺少必需 unit |
| C2/P2 | 候选兼容 | 两字段均存在 |
| C1/P3 | 不兼容 | 原字段被删除 |
| C2/P3 | 不兼容 | available 被删除 |

不能由 C2/P2 通过推出 C2/P1 或 C1/P3；若部署允许新旧混合则先验证实际允许组合，再安排顺序或拒绝升级。C1/P2 若解析器拒绝附加字段，表格前提不成立。除形状外，字段单位、默认值、分页、错误码、幂等身份、授权范围的语义变化也可能破坏兼容，即使 schema 验证通过。合同没有对应交互则 NOT_COVERED，而非不可能。组合测试需明确客户端与服务器版本、真实配置及支持策略，不要求不受支持的历史版本兼容。

## 4. 安装制品与源码测试的边界

教学 Node.js ESM 包 public root 导出 readStock，package.json 的 exports 指向 ./dist/index.js。工作区源码测试通过，仍可能缺 dist、漏传运行时依赖、类型声明指向不存在文件、使用未导出的内部路径，或被 workspace hoist/链接意外满足依赖。Node.js v22.23.3 packages 文档规定 exports 的公开子路径；未公开的包子路径可被拒绝（ERR_PACKAGE_PATH_NOT_EXPORTED），绝对路径访问并非强安全隔离。按实际支持的 Node/OS/module 条件选择验证，不要求兼容未承诺的 CJS。仅能 import 不表示业务可用。

有限制品验收步骤：
1. 在受权环境得到真实待发布包字节，记录来源提交、制品 hash、manifest 与文件清单；独立核必要文件和依赖声明。
2. 在不属于生产者 workspace 的新目录，以真实公开安装方式安装同一字节，固定实际 runtime 和依赖解析；记录安装后的包路径/版本、真实解析的依赖及条件。避免从源码目录、全局模块、NODE_PATH 或 symlink 偶然补足缺文件。隔离措施按目标工具确认。
3. 真实消费者仅通过支持的 public specifier 调用 readStock；如果 TypeScript 声明是公开合同，另以真实消费者 typecheck，再执行编译产物，不能用 typecheck 代替执行。按实际产品 API 验证成功、失败和禁止行为。合法负例 import 私有子路径应拒绝（仅适用于所设 exports 合同）。
4. 验证已安装内容与同一待交付字节的对应，不让测试改写包内文件后再报 PASS。保留安装/执行日志、失败阶段及独立观察点；环境不可用明确 NOT_RUN。没有声明所有包必须采用 tarball、单入口或 Node。

## 5. 区分性测试设计及结果资格

| 教学变异（NOT_RUN） | 首个合理检测点及 Oracle |
|---|---|
| client 丢失 warehouse 查询参数 | 真实 adapter 对协议替身发出的请求不匹配 |
| provider 将 NOT_FOUND 改为 200 空对象 | provider 回放不匹配；真实 client 负例避免误报库存为0 |
| provider 返回7但实际记录属于 W2 | 仅类型匹配可通过；独立权威数据按 W1/W2 对照，未经授权无读写 |
| create 返回成功但数据库未提交 | provider 功能/真实集成读取提交状态，响应匹配不能证明 Effect |
| dist 未被打包，工作区测试引用 src | 干净消费者真实安装及公开入口加载失败；不得 fallback |
| P2 增加字段，C1 拒绝额外字段 | C1 实际解析新响应失败，即使宽松 provider 验证通过 |
| 依赖只在生产者开发环境存在 | 外部安装解析或业务调用失败，记录真实缺项 |

不以被测实现生成预期值；可以使用独立业务规格、受权 fixture 数据和前后观察。真实外部效果不可见时保留 UNKNOWN。结果至少区别未发现/已发现未运行/已运行通过/失败/未覆盖及不支持，不因重试绿丢失首个失败。发布决策仍需真实授权和产品质量范围，合同成功不是生产发布许可。

实读 2026-11-26 Pact [How Pact works](https://docs.pact.io/getting_started/how_pact_works) L63–91、[Contract Tests vs Functional Tests](https://docs.pact.io/consumer/contract_tests_not_functional_tests) L23–109、[Matching](https://docs.pact.io/getting_started/matching) L69–103/179–192；在线说明非固定 SDK 版本；[Node.js v22.23.3 Packages](https://nodejs.org/download/release/v22.23.3/docs/api/packages.html) L273–351，公开入口与封装边界。制品验收步骤、版本矩阵和库存例为原创教学设计；未安装包、未执行 Pact、未验证真实消费者、未重新裁定历史 Case。DOCUMENT_CANDIDATE，无新合格 Claim。
