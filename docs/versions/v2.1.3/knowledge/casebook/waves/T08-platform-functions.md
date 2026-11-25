# T08 · 平台功能与模块抽象

[#567](https://github.com/kaicreator-mm/tally/issues/567)。7 个新案例、7 个新登记文件（另复用 T04-S08 同 Blob 新位置），涵盖任务调度、插件、IPC、UI、生命周期、设备、安装升级七族。源观察与分析分离，测试建议 NOT_RUN；不把 SDK/框架接口当项目必须采用的设计。

## T08-C01 · 周期任务：到期调度与任务执行分离

**分类：** 任务/调度 / L3–L5。**知识：** K03.F3.3。**来源：** T08-S01。**关系：** 机制关联；到期调度及任务执行分离。验证意图另见正文，NOT_RUN。

- **实际观察：** Celery Beat tick 更新 schedule heap，判断堆顶是否到期，reserve 后 apply_entry、重新入堆，返回下一次调用等待；每 tick 处理一个到期任务。
- **职责及合同（分析）：** 调度器拥有计划与到期计算，Worker 负责执行；计划推进、任务提交、收到结果是不同事实，不可因 tick 返回 0 推断业务完成。
- **对照、反转条件及验证建议（NOT_RUN）：** 少量单进程定时器、独立周期调度+Worker、依赖驱动 DAG（AR-009）；实际持久性与并发决定方案。对照 AB-005 任务 ACK。检查时间变化、延迟、重复触发、调度重启、发布失败；不证明单实例或 exactly-once。

## T08-C02 · 插件发现与加载：声明、解析、权限分离

**分类：** 插件/模块组合 / L3–L5。**知识：** K02.F2.3。**来源：** T08-S02。**关系：** 机制关联；插件发现与实际扩展点。验证意图另见正文，NOT_RUN。

- **实际观察：** CPython EntryPoint 按包装约定记录 name/group/value，load 根据描述 import module，再逐级获取属性。
- **职责及合同（分析）：** 发现候选、验证兼容/授权、加载并绑定功能是不同阶段。元数据是声明，不是插件可信证明；import 可能执行代码，不能靠发现过程保证隔离。
- **对照、反转条件及验证建议（NOT_RUN）：** 固定模块直接 import；有独立发布消费者才用 entry-point 发现；不可信/异构插件需实际进程/权限边界。对照 AR-005、DE-009，检查版本缺失、加载异常、非法能力与资源释放。

## T08-C03 · IPC：命令调用与事件订阅是不同合同

**分类：** 进程/IPC / L3–L5。**知识：** K03.F4.2。**来源：** T08-S03、T08-S04。**关系：** 机制关联；进程通信及监听资源。验证意图另见正文，NOT_RUN。

- **实际观察：** VS Code IChannel/IServerChannel 分别声明 call/listen，服务端带 context，命令带取消 token；请求类型区分 Promise、取消、监听、释放。Electron 项目文档分别展示单向 send/on 与双向 invoke/handle。
- **职责及合同（分析）：** 传输封装请求相关性、取消与订阅生命周期；应用命令定义合法动作和错误、实际授权。事件表示通知，不自动有响应或成功确认。服务端 context 不天然可信。
- **对照、反转条件及验证建议（NOT_RUN）：** 同进程直接函数/回调、跨进程命令应答、跨进程事件订阅；仅在真实边界时付序列化成本。检查重复消息、断线、未授权命令、取消与 late reply、释放 listener；不宣称两套实现全部安全。

## T08-C04 · UI 状态：状态更新与观察通知

**分类：** UI 状态/组件 / L3–L5。**知识：** K03.F3.1。**来源：** T08-S05。**关系：** 机制关联；状态转移和通知。验证意图另见正文，NOT_RUN。

- **实际观察：** Redux dispatch 检查 action，拒绝 reducer 内 dispatch；在 try/finally 中执行 reducer 更新 currentState，之后通知 listeners；subscribe 在 reducer 执行时拒绝。
- **职责及合同（分析）：** 命令/事件意图→状态转移→通知消费者。共享 store 不等于领域权威账本；组件局部编辑和服务器业务事实的所有者不同。
- **对照、反转条件及验证建议（NOT_RUN）：** 局部 UI 用组件状态；多个视图共享派生状态用 store；对照 T02-C01 单向数据流与 T02-C03 文档模型。检查非法 action、reducer 失败、重复订阅、过期响应、UI 关闭；不把 Redux 当持久事务引擎。

## T08-C05 · 生命周期：配置存在、加载、卸载和资源所有权

**分类：** 生命周期 / L3–L4。**知识：** K03.F3.1。**来源：** T04-S08。**关系：** 机制关联；激活与卸载状态。验证意图另见正文，NOT_RUN。

- **实际观察：** 同 T04-S08 的 Home Assistant config_entries.py Blob，重新读取 async_unload L802–817：忽略条目设置 NOT_LOADED 返回 true，NOT_LOADED 直接成功；其他分支寻找 integration。原 T04-S08 已观察 setup 锁、状态和能力。
- **职责及合同（分析）：** 配置记录、加载状态、资源与卸载结果独立。是否允许重复初始化/结束由具体合同决定；不能由上述 early return 推断所有已加载卸载幂等。
- **对照、反转条件及验证建议（NOT_RUN）：** 静态单资源作用域、显式多阶段对象、动态集成状态机；对照 T05-C10 退出栈。检查未激活卸载、重复调用、半初始化、卸载失败、其他消费者仍引用；未读完整卸载与资源释放链。

## T08-C06 · 设备协议：共享总线与设备级语义

**分类：** 设备协议 / L3–L5。**知识：** K02.F2.2。**来源：** T08-S06。**关系：** 机制关联；总线与专有协议合同。验证意图另见正文，NOT_RUN。

- **实际观察：** ESPHome UARTDevice 持有 UARTComponent parent，write_byte/write_array 等直接委托；不同 array 形式进入共同字节传输路径。
- **职责及合同（分析）：** 物理端口/共享总线负责字节传输，设备驱动负责帧、命令和设备状态；不以共享 UART 抽象不同设备协议。父资源和设备生命周期需要明确归属。
- **对照、反转条件及验证建议（NOT_RUN）：** 单设备直接串口；多设备复用总线 adapter；各协议专有 parser/状态机。对照 DE-010 设备反馈。检查空/失效 parent、并发总线、短帧、超时、错误响应；头文件不支持具体锁或串口时序保证。

## T08-C07 · 安装升级：目标模型、迁移计划与登记

**分类：** 安装升级/数据库迁移 / L2–L4。**知识：** K09.F3.2。**来源：** T08-S07。**关系：** 机制关联；数据库状态迁移及登记。验证意图另见正文，NOT_RUN。

- **实际观察：** Django migrate 文档描述先构建项目状态，再执行数据库操作；有迁移记录表；apply_migration 根据 fake/fake_initial 与 detect_soft_applied 决定是否执行 schema_editor，atomic 取自 migration。
- **职责及合同（分析）：** 当前版本与目标合同→差异计划→实际变更/验证→登记；登记、真实数据库状态和应用可运行性三者独立。fake 只在适用的有权维护条件下使用，不是实际执行证明。
- **对照、反转条件及验证建议（NOT_RUN）：** 无状态程序可原子替换制品；持久数据系统需有版本迁移及备份/恢复；大规模在线升级额外考虑跨版本兼容。检查重复升级、实际 schema 与登记不符、部分失败、旧消费者、回滚；未验证 Django 所有后端事务性。

## 实际来源登记

- **T08-S01**：[celery/celery/celery/beat.py](https://github.com/celery/celery/blob/v5.5.3/celery/beat.py) · ref `v5.5.3` · Blob `86ad837f0d5333169e7c566ff8fbfc1ba9618d1a` · L218–231,325–364。周期任务 tick、堆和下次调度。
- **T08-S02**：[python/cpython/Lib/importlib/metadata/__init__.py](https://github.com/python/cpython/blob/v3.13.0/Lib/importlib/metadata/__init__.py) · ref `v3.13.0` · Blob `8ce62dd864fc2735dc4c863909b0622b4f5f4548` · L126–140,173–186。EntryPoint 声明及动态 import/属性查找。
- **T08-S03**：[microsoft/vscode/src/vs/base/parts/ipc/common/ipc.ts](https://github.com/microsoft/vscode/blob/1.96.0/src/vs/base/parts/ipc/common/ipc.ts) · ref `1.96.0` · Blob `5b51cd2ae90d00342590e5d5ebc4345da9deff6b` · L25–45。客户端/服务端通道、取消和订阅消息类型。
- **T08-S04**：[electron/electron/docs/tutorial/ipc.md](https://github.com/electron/electron/blob/v33.0.0/docs/tutorial/ipc.md) · ref `v33.0.0` · Blob `b8b0bad4212f6291a37a99cf455ff1ff46b44221` · L34–46,153–165。项目文档单向 send/on 与双向 invoke/handle。
- **T08-S05**：[reduxjs/redux/src/createStore.ts](https://github.com/reduxjs/redux/blob/v5.0.1/src/createStore.ts) · ref `v5.0.1` · Blob `e5f3b6d7bad1208ee74b11dece2e3634b4d3acf8` · L202–217,271–319。dispatch 检查、reducer 调用和订阅通知。
- **T08-S06**：[esphome/esphome/esphome/components/uart/uart.h](https://github.com/esphome/esphome/blob/2024.12.0/esphome/components/uart/uart.h) · ref `2024.12.0` · Blob `dc6962fbae45775ea333c576d2c0f40b0d711a8e` · L12–27。UARTDevice 持有 parent，写操作委托。
- **T08-S07**：[django/django/django/db/migrations/executor.py](https://github.com/django/django/blob/5.2/django/db/migrations/executor.py) · ref `5.2` · Blob `13afa5988f68eecb54d08e7c3cf087a33e13eb83` · L94–110,239–255。目标状态、计划、迁移登记、fake 和 schema_editor。

跨分区 [T04-S08](T04-intelligence-and-infrastructure-architecture.md) 实际追加位置 [L802–817](https://github.com/home-assistant/core/blob/2025.1.0/homeassistant/config_entries.py#L802-L817)，同 Blob ade4cd855cac21c9e7f484adae4384f4aa4aaa04。Alembic 候选 ref rel_1_14_0 404，改用真实 Django migration executor；未计入不可读文件。T08 DELIVERED_SCOPED，独立接受待 T14；完整插件兼容/权限、部署升级、设备硬件行为未运行。