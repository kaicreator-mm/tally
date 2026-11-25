# T05 · 通用功能与模块抽象

[#567](https://github.com/kaicreator-mm/tally/issues/567)。10 个真实来源参考，11 个新登记文件；九功能族（配置、日志、错误、序列化、验证、缓存、查询、并发、资源）均有至少两个约束下的方案，复用既有 AB/DE/T04 作为对照，不重复计数。所有源为局部源码/接口文档观察；参考设计与验证建议为分析，NOT_RUN，无运行/性能保证。Kxx 保持原 Owner。

## T05-C01 · 配置解析：来源优先级不是配置激活

**分类：** 配置 / L3–L5。**知识：** K02.F3.1 K03.F2.1 K01.F3.3。**来源：** T05-S01。

- **实际观察：** Viper find 实际依次处理显式 Set、已变化的 flag、环境变量、配置文件、KV 和默认值；部分路径被上层遮蔽时停止。
- **职责及合同：** 读取来源→规范化键→按明确优先级求有效值→独立验证/应用。区分空值、缺失、显式覆盖和不合法值；持久配置与运行资源不共用 Owner。
- **对照与采用边界：** 固定启动参数可直接解析一次；多来源 CLI 适合确定优先级。对照 T04-C09、DE-006：需要交互首次配置、热切换的 App 还要有激活状态；不能把 Viper 查询结果当生效回执。
- **反例与检查（NOT_RUN）：** 同一键三来源、未改变 flag、父路径遮蔽、无效值、重载失败；不得泄露密钥，不推断该片段提供原子热更新。

## T05-C02 · 日志：提前过滤与输出职责

**分类：** 日志 / L3–L5。**知识：** K02.F2.3 K03.F2.1 K10.F2.1。**来源：** T05-S02。

- **实际观察：** slog.Handler 接口要求 Enabled 在参数处理前判断级别，并由 Logger 调用。
- **职责及合同：** 事件形成与级别/上下文过滤分开；输出端负责序列化和实际输出。对昂贵参数构造应注意调用点本身的求值，不把 Enabled 误认为能阻止所有调用者副作用。
- **对照与采用边界：** 对照 AB-006 CPython Handler 的记录过滤与锁内输出：早期抑制成本与输出端细粒度过滤可组合；单输出短 CLI 用普通记录足够。
- **反例与检查（NOT_RUN）：** 禁用级别、多个 Handler、输出失败、并发、敏感属性；日志不是业务授权或成功证据。

## T05-C03 · 错误：保留原因树与领域分类

**分类：** 错误 / L4–L5。**知识：** K03.F5.1 K06.F2.2 K02.F2.2。**来源：** T05-S03。

- **实际观察：** Go errors.Is 检查相等、自定义 Is 和单/多 Unwrap；As 对目标类型/非空指针有明确限制。
- **职责及合同：** 底层原因与公开错误分类分离；包装加上下文但保留可识别原因。聚合多失败时保存独立来源，不以字符串包含判断。
- **对照与采用边界：** 单层纯函数可直接返回窄错误；跨 I/O 层用包装原因；对照 AB-005，能否重试由具体任务政策决定，而不是通用异常分类。
- **反例与检查（NOT_RUN）：** 包装多层、多个独立原因、错误类型/同名消息、非法目标；异常传播不应暴露敏感数据。未检查循环错误树。

## T05-C04 · 序列化：流式解码和严格模式

**分类：** 序列化 / L4–L5。**知识：** K03.F1.2 K03.F5.1 K05.F1.2。**来源：** T05-S04。

- **实际观察：** Go Decoder 读取下一 JSON 值；DisallowUnknownFields 可针对 struct 选择拒绝未知键。
- **职责及合同：** 语法解析→形状/兼容政策→业务校验独立。读取一个值不等于验证整个流；消费端必须明确 EOF、尾随内容及限额合同。
- **对照与采用边界：** 一次小配置可整文档解码；大输入或连续消息按流。对照 T05-C05 的对象验证：解析合法不表示类型/业务合法；版本演进是否拒绝未知字段由消费者合同决定。
- **反例与检查（NOT_RUN）：** 多值输入、截断、未知键、尾随非法内容、内存/输入限额；只核对入口，不声称默认完整流/资源安全。

## T05-C05 · 验证：输入类型与业务政策

**分类：** 验证 / L4–L5。**知识：** K01.F3.3 K03.F5.1 K05.F1.2。**来源：** T05-S05。

- **实际观察：** Pydantic model_validate 将 strict/from_attributes/context 交给底层 validator。
- **职责及合同：** 验证边界明确转换还是严格拒绝，形状结果与受权业务不变量、当前资源状态分离。不能由客户端自报 context 赋权。
- **对照与采用边界：** 简单内部函数按已知输入域检查；外部结构化输入可模型校验；对照 AB-011 的 ERPNext 订单规则，类型通过仍需独立业务验证。
- **反例与检查（NOT_RUN）：** 字符串数字、未知字段、可空与缺失、模型合格但库存/权限不合格；源只支持参数和委托，不推断所有 coercion 规则。

## T05-C06 · 缓存：并发加载与失效合同

**分类：** 缓存 / L3–L5。**知识：** K03.F3.1 K02.F3.1 K10.F4.1。**来源：** T05-S06。

- **实际观察：** Caffeine Cache.get 文档要求调用原子执行、计算函数每键至多一次，计算宜短且不可更新其他 mapping；null/异常可不建立映射；加载中失效有限制。
- **职责及合同：** cache miss→受控加载→缓存记录；权威值仍归源。原子加载范围是单次本地缓存操作，不等于持久业务动作 exactly-once。
- **对照与采用边界：** 小缓存用直接 Map/重算；并发昂贵读取用成熟 loading cache；对照 AB-015 Redis 过期回收，缓存有效性、加载协调及资源回收是不同问题。
- **反例与检查（NOT_RUN）：** 同键并发、加载异常、递归更新、加载中失效、过期读；不把缓存 miss 当业务不存在。

## T05-C07 · 查询：连接与返回行的范围

**分类：** 查询 / L3–L4。**知识：** K02.F2.2 K03.F4.2 K05.F1.2。**来源：** T05-S08。

- **实际观察：** Go DB.QueryContext 经 retry 调用 query，返回 Rows/error；DB 表示连接池，而 Tx 绑定连接。
- **职责及合同：** 参数化查询、连接获取、结果迭代和关闭分别负责；查询返回句柄不代表全部结果已消费。错误/取消后要检查实际行和资源。
- **对照与采用边界：** 小内存数据直接过滤；数据库查询使用连接池/行迭代；对照 T04-C04 top-n 与全匹配收集、DE-007，结果范围与实际完整性必须显式区分。
- **反例与检查（NOT_RUN）：** 查询失败、迭代错误、部分结果、上下文取消及 Rows 释放（实际 Rows 实现未在本案审查）。不得宣称 retry 对任意写操作安全。

## T05-C08 · 并发：取消传播不是完成确认

**分类：** 并发 / L4–L5。**知识：** K03.F3.3 K03.F4.1 K10.F3.2。**来源：** T05-S09。

- **实际观察：** Context 有 Deadline/Done，WithCancel 连接父子取消；CancelFunc 不等待工作停止，多次调用无额外作用。
- **职责及合同：** 取消信号、工作完成、资源清理和已发生副作用分开记录。调用者负责及时释放 context 资源，被调用者合作式响应。
- **对照与采用边界：** 同步短工作可直接返回；并发/跨层工作传递取消与 deadline；对照 T04-C08 的线程调度，取消信号不等于内核强杀或实时完成。
- **反例与检查（NOT_RUN）：** 取消前已提交外部效果、子任务不响应、超时与完成竞争、重复取消；须读回真正终态。

## T05-C09 · 资源池：临时对象与稀缺连接

**分类：** 资源管理/并发 / L3–L5。**知识：** K03.F4.1 K03.F4.2 K10.F4.1。**来源：** T05-S07、T05-S08。

- **实际观察：** sync.Pool 文档允许任意时刻丢弃临时对象；database/sql DB 管理并发连接、空闲池、最大打开数，Tx 绑定单连接。
- **职责及合同：** 对象复用缓存不承诺保留项；连接池是稀缺资源和会话管理。二者不能因 get/put 形状相似而共享资源保证。
- **对照与采用边界：** 低频分配直接构造；临时 buffer 用可丢弃池；连接用明确借还、上限与失效策略。两种实际合同对照，不编造性能。
- **反例与检查（NOT_RUN）：** GC 丢对象、连接耗尽、事务内连接状态、重复归还；业务身份和租户数据不可残留在复用对象。

## T05-C10 · 资源清理：静态作用域与动态清理栈

**分类：** 资源管理 / L4–L5。**知识：** K03.F4.1 K03.F5.1 K10.F5.1。**来源：** T05-S10、T05-S11。

- **实际观察：** Go io.Closer 只保证第一次 Close 的抽象合同，重复行为由实现决定；io.Copy 可用可选快路径。Python ExitStack 说明动态注册的资源在后续获取失败时也关闭，退出时弹栈处理回调及异常。
- **职责及合同：** 成功获取→立即注册清理→按实际依赖逆序释放；主失败与清理失败有明确传播策略。复制/读取不自动拥有关闭权。
- **对照与采用边界：** 一个确定资源用语言作用域/defer；数量动态时使用退出栈；多个独立资源不强制统一生命周期框架。
- **反例与检查（NOT_RUN）：** 第二资源获取失败、第一个释放失败、重复 close、已转移所有权；不能从 ExitStack 推断所有资源自动事务回滚。

## 实际来源登记

- **T05-S01**：[spf13/viper/viper.go](https://github.com/spf13/viper/blob/v1.19.0/viper.go) · ref `v1.19.0` · Blob `da68d9944cc830bac38703fd5bb9424d6b4e30e7` · L902–916,1290–1405。多来源查询优先级及路径遮蔽。
- **T05-S02**：[golang/go/src/log/slog/handler.go](https://github.com/golang/go/blob/go1.23.4/src/log/slog/handler.go) · ref `go1.23.4` · Blob `2ff85b582ef32e6335fcd2d7d04ed7814cd7b5e7` · L28–44。Logger/Handler 和提前级别过滤的接口。
- **T05-S03**：[golang/go/src/errors/wrap.go](https://github.com/golang/go/blob/go1.23.4/src/errors/wrap.go) · ref `go1.23.4` · Blob `eec9591dae7b933132b442422cc74b3935b79f65` · L44–79,97–110。Is 的单/多错误展开与 As 目标限制。
- **T05-S04**：[golang/go/src/encoding/json/stream.go](https://github.com/golang/go/blob/go1.23.4/src/encoding/json/stream.go) · ref `go1.23.4` · Blob `cb61ea721207e82c656aac17ff236d543caf8893` · L39–61。流式解码下一值与未知字段策略。
- **T05-S05**：[pydantic/pydantic/pydantic/main.py](https://github.com/pydantic/pydantic/blob/v2.10.4/pydantic/main.py) · ref `v2.10.4` · Blob `ca5a4529e9e398ccce2606acebfda1673f55185d` · L603–629。model_validate 与 strict、context、attributes 参数。
- **T05-S06**：[ben-manes/caffeine/caffeine/src/main/java/com/github/benmanes/caffeine/cache/Cache.java](https://github.com/ben-manes/caffeine/blob/v3.1.8/caffeine/src/main/java/com/github/benmanes/caffeine/cache/Cache.java) · ref `v3.1.8` · Blob `251e4c8a4693d71a2881c985bdee42c7ad43a08b` · L46–81,158–166。getIfPresent、原子加载及失效限制。
- **T05-S07**：[golang/go/src/sync/pool.go](https://github.com/golang/go/blob/go1.23.4/src/sync/pool.go) · ref `go1.23.4` · Blob `0fa8f8cdaa028ded6f335170cf896599ed797c9b` · L14–27,48–63。临时对象池的自动丢弃合同。
- **T05-S08**：[golang/go/src/database/sql/sql.go](https://github.com/golang/go/blob/go1.23.4/src/database/sql/sql.go) · ref `go1.23.4` · Blob `c247a9b506bfabb38b03ed1f160116370fde0572` · L487–510,1021–1034,1729–1742,1861–1873,2279–2294。连接池、查询、事务及提交入口；同时供 T06 使用。
- **T05-S09**：[golang/go/src/context/context.go](https://github.com/golang/go/blob/go1.23.4/src/context/context.go) · ref `go1.23.4` · Blob `763d4f777ffb867379fd88699c54a4b7285ec63a` · L68–81,223–238。取消信号及资源责任。
- **T05-S10**：[golang/go/src/io/io.go](https://github.com/golang/go/blob/go1.23.4/src/io/io.go) · ref `go1.23.4` · Blob `00edcde763a55ae326a70f7c02a26a39d4f5261a` · L104–110,384–400。流复制、可选优化及 Close 重复调用边界。
- **T05-S11**：[python/cpython/Lib/contextlib.py](https://github.com/python/cpython/blob/v3.13.0/Lib/contextlib.py) · ref `v3.13.0` · Blob `5b646fabca0225b4bb2cecec065f6861e7ed12ae` · L557–572,595–609。ExitStack 的动态资源和逆序回调。

## 交接

T05 DELIVERED_SCOPED，等待 T14；同功能不同约束已明确，但多数是接口/局部源观察与设计备选，不等于两套完整真实实现已跑通。后续可追加跨语言序列化、查询资源、错误恢复的运行证据和第二真实实现。禁止将新案例当强制 Rule。