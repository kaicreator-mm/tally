# T15 · TypeScript/JavaScript 抽象合同与选择对照

[#568](https://github.com/kaicreator-mm/tally/issues/568) / #563；首期固定基线 [#567 closeout](https://github.com/kaicreator-mm/tally/issues/567#issuecomment-7157423397) @ 772eb6819a30a4fb248c6c60d24e08bb02a00c19。回应 T13 的 T09 多语言缺口；四条独立 L5 源码观察，不是四个完整产品审计。原 T13 目录与反向台账保持历史快照；本文件为增量目录、关系和来源登记，原 Kxx Owner 不变。所有原创反例和验证意图 NOT_RUN；不声明实际维护收益、运行正确性或生产 Rule。

## 增量选择目录

| ID / 功能 | 决策条件 | Kxx 关系与理由 | 源 |
|---|---|---|---|
| T15-C01 异步任务限流 | 并行上限、待办及完成语义 | K03.F3.3 机制关联：独立队列和活跃任务计数限制并发启动，而非保证队列容量 | T15-S01 |
| T15-C02 组合资源释放 | 多资源同寿命、失败时仍尝试释放 | K03.F4.2 条件启发：通用同步 dispose 机制可用于文件/网络/进程资源，但这些实际资源合同未在本片段验证；K02.F2.2 机制关联：组合释放的错误合同 | T15-S02 |
| T15-C03 类型与解析错误合同 | 不可信数据、同步/异步、错误传播 | K03.F2.2 机制关联：unknown 运行时解析与类型化结果；K02.F2.2 机制关联：异常式与结果式公开合同 | T15-S03 |
| T15-C04 状态函数组合 | 多状态分区、身份保持、跨分区不变量 | K03.F2.1 机制关联：同一调用合同下独立 reducer 与组合函数职责；K02.F3.1 仅条件启发：状态分片不证明业务事务 | T15-S04 |

## T15-C01 · 共享并发许可，不能将限流冒充背压或取消

**实际观察（T15-S01）：** p-limit 工厂持有 queue、activeCount 与 concurrency；resumeNext 在有许可且有待办时启动并计数。run 将任务 promise 交给调用者，然后等待它 settle（拒绝也会进入 next）；next 释放计数再尝试队列。clearQueue 只清队列；该片段没有给每个被清除的待办显式 resolve/reject 的操作。concurrency setter 校验新值并在微任务尝试启动待办。

**条件比较（分析）：**
- A：已知只有少量任务且允许串行，用普通逐项 await，不需要额外生命周期对象。
- B：同一进程的多个消费者确有共享并发额度且允许排队，用共享 limiter；不可按请求新建实例后声称全局上限。该许可覆盖任务 promise 的实际 settle；若函数提前返回、效果仍在途，额度不覆盖真实效果。
- C：输入流不可无限缓冲、任务有超时/取消/每条必须终止回执，另定义队列容量、拒绝/等待策略及取消合同。不能因存在 activeCount 就推导内存有界，降低上限也不代表已运行任务立即取消。
- 原创反例：两个各自上限为 2 的实例不保证共享资源最多 2 个操作；提前 resolve 的任务可能仍消耗实际连接。检查同一实例峰值、拒绝后后续启动、降额后不新超额启动、清队列后各 promise 是否终止。NOT_RUN；无实测吞吐或完整公平性承诺。

## T15-C02 · 组合释放的核心是合同与责任，不是把一切包装成对象

**实际观察（T15-S02）：** VS Code 定义同步 IDisposable.dispose(): void；dispose 对 iterable 中每个非空成员尝试释放，收集成员抛出的错误，一个错误原样抛出、多个错误用 AggregateError；combinedDisposable 将组合释放封装为一个 disposable，并登记 parent。此处不证明迭代器本身抛错时继续遍历，不证明每个成员成功、异步 close 已完成或任意重复 dispose 都安全。

**条件比较（分析）：**
- A：单一局部资源、明确作用域结束，用 try/finally，最小抽象。
- B：多个同所有者、同生命周期的同步释放责任，用组合句柄；成员失败不能掩盖其他成员的清理尝试和实际异常。借用的共享对象不可因被引用就登记为拥有的资源。
- C：事务提交/回滚、异步 close、严格依赖逆序释放等具有不同合同，需要明确的阶段/顺序/等待/失败政策；不可把同步 dispose 改名为 async 并假定调用者会等待，或默认任意顺序正确。
- 原创反例：第一个资源释放抛错，第二个仍应获得释放尝试；两个错误不丢失；共享资源仍服务另一消费者；异步关闭未完成不能宣告 quiescent。检查成员调用、失败聚合、借用关系、异步结束；NOT_RUN。不是 VS Code 全系统资源正确性结论。

## T15-C03 · 相同解析机制可以复用，不同失败合同仍需显式表达

**实际观察（T15-S03）：** ZodType 有 Input/Output 类型参数和 abstract _parse。parse 调用 safeParse 并对失败抛错；safeParse 从 unknown 数据建上下文，调用 _parseSync 再 handleResult；后者返回 success/data 或 success/error；_parseSync 遇异步结果抛错；_parseAsync 将结果转换为 promise。handleResult 在无 issues 的非有效结果时也抛错。因此 safeParse 不是对任意内部异常的全吞噬，也不证明验证器副作用或授权。

**条件比较（分析）：**
- A：内部已建立且当前仍有效的小类型合同，普通函数/类型足够；不要为了统一风格再引入通用 schema 工厂。
- B：外部数据、转换与约束需要运行时检查，复用解析机制，按调用者需要选择显式结果式或异常式包装；明确输入类型和输出类型可以不同。
- C：授权、并发前提和业务状态裁定属于业务合同；不能把 shape validation 成功转换为业务准入成功。异步检查须有真正等待与失败边界；类型断言不是运行时检查。
- 原创反例：合法结构但无权访问的数据；内部解析器抛错不等同于格式错误；同步入口遇异步工作不应返回成功。检查输入边界、失败分类、转换后值、异步入口、未授权请求；NOT_RUN。未审全部 schema 实现。

## T15-C04 · 函数组合的结构共享不等于事务一致性

**实际观察（T15-S04）：** Redux combineReducers 从映射构成函数集合，探测初始化和未知动作不得返回 undefined；实际 combination 对各分片调用 reducer，禁止返回 undefined，按分片引用变化及 key 数量决定返回新 state 或原 state。函数组合共享遍历和形状合同；并非深比较，也未在本函数中检查业务跨片事务约束。

**条件比较（分析）：**
- A：小状态且单一业务不变量直接使用一个 reducer/普通函数。
- B：分片有稳定职责、共享 action 调用合同且能独立计算，用组合机制；保留未变引用有利于引用式消费者（具体性能 NOT_RUN）。
- C：多分片需一起维护库存与账务等不变量，由有权业务状态转换定义整体合法性、持久化与失败语义；不能把广播动作到所有分片视为数据库事务，或在返回原引用时偷偷原地修改数据。
- 原创反例：相同对象引用内字段已变，引用判断无法证明未变；一片拒绝而另一片产生业务变化不自动满足跨片约束。检查未知 action、undefined、引用保持、跨片合法状态、持久化失败；NOT_RUN。该源只支持 reducer 组合机制，跨片合同为原创分析。

## 来源登记（固定版本 ref 与实际读取 Blob）

- **T15-S01** [sindresorhus/p-limit/index.js](https://github.com/sindresorhus/p-limit/blob/v6.2.0/index.js) · v6.2.0 · Blob 22b1e8717d7509cf2c1f56c95bc53beb3b94ec7b · L1–104（实际完整读取）；队列、许可、settle、公开属性。
- **T15-S02** [microsoft/vscode/src/vs/base/common/lifecycle.ts](https://github.com/microsoft/vscode/blob/1.96.0/src/vs/base/common/lifecycle.ts) · 1.96.0 · Blob 4cc2d172bf4f61b6f00d06b761551adf7a0ec8a3 · L269–339（实际读取范围内） IDisposable、dispose、combinedDisposable；不推断其他成员实现。
- **T15-S03** [colinhacks/zod/src/types.ts](https://github.com/colinhacks/zod/blob/v3.24.1/src/types.ts) · v3.24.1 · Blob 98281ff2f08c8ceff026e8ed1733cf07e083589f · L92–115、154–185、228–266：handleResult、SafeParseReturnType、ZodType._parseSync/_parseAsync/parse/safeParse。
- **T15-S04** [reduxjs/redux/src/combineReducers.ts](https://github.com/reduxjs/redux/blob/v5.0.1/src/combineReducers.ts) · v5.0.1 · Blob 5d99a8b585591506b7adc5d5aa2d860d80d7c91f · L62–201：assertReducerShape、combineReducers/combination。

专业定义锚点（原 Kxx 未修改）：[K02.F2.2 L48 / F3.1 L67](https://github.com/kaicreator-mm/tally/blob/772eb6819a30a4fb248c6c60d24e08bb02a00c19/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L48) · [K03.F2.1 L40 / F2.2 L48 / F3.3 L83 / F4.2 L102](https://github.com/kaicreator-mm/tally/blob/772eb6819a30a4fb248c6c60d24e08bb02a00c19/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L40)。

## 反向补充与停止条件

K02.F2.2 → C02/C03 机制；K02.F3.1 → C04 条件；K03.F2.1 → C04 机制；K03.F2.2 → C03 机制；K03.F3.3 → C01 机制；K03.F4.2 → C02 条件。先判断真实需求/语义差异，再查候选；不能凭语言或 API 名称选模式。原 T13 仍是首期 125 条快照，本增量新增 4 条来源参考、4 个来源文件；没有新增理论概念或把待验证建议变成实际证明。仍缺真实业务函数、多语言对照（Java/C++/Rust）、端到端证据和维护经济性；不无限扩充。独立复核本增量后停止。
