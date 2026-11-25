# T16 · 跨语言抽象：作用域退出、所有权与失败合同

[#570](https://github.com/kaicreator-mm/tally/issues/570) / #563；基线 724e83f9e0c1e515388164d949849aff233a0a21（T15 已合并）。回应 T09 多语言缺口，聚焦同一个机制族，而非按语言凑案例。复用 [T15-C02](T15-ts-js-abstraction-contrasts.md)、[T10 C1](T10-theory-cd.md) 的资源责任分析；不重定义 Kxx，不改 T13/T15 历史快照。三个真实来源 L5 案例；下方跨语言对照是综合分析，不另计案例。所有行为检查与外部项目执行 NOT_RUN。

## 增量目录

| ID | 机制及约束 | 关联与理由 | 来源 |
|---|---|---|---|
| T16-C01 Java 显式关闭与主次异常 | 多个成功登记的 Closeable、逆序、失败优先级 | K02.F2.2 机制关联：关闭异常合同；K03.F4.2 机制关联：登记后逆序尝试释放 I/O Closeable（非具体资源释放实证） | T16-S01 |
| T16-C02 C++ 可移动 scope guard | 析构执行回调、转移执行责任、解除 | K03.F2.1 机制关联：封装单一退出动作和执行责任；K03.F4.2 条件启发：回调可用于释放资源，但源未实现具体 I/O 关闭 | T16-S02 |
| T16-C03 Rust 拥有值的条件 guard | Drop、一次性闭包、panic 策略及取回值 | K03.F2.1 机制关联：值与退出行为封装；K03.F4.2 条件启发：用于资源或不变量恢复，未证明具体 I/O | T16-S03 |

## T16-C01 · Java：关闭动作可组合，异常优先级不能丢失

**真实观察：** Guava Closer 文档说明原意是近似 Java 7 try-with-resources 的旧环境兼容实现，并规定调用模式。register 将非 null Closeable 放在 deque 首位并返回原对象；rethrow 记录原异常再按类型重抛或包装；close 逐个取出尝试，保留已有主异常，否则记录第一次关闭异常，后续调用 suppressor。默认 suppressor 避免自抑制，尝试 addSuppressed，失败时尝试日志。主异常的保留依赖正确调用 rethrow，不能由普通 finally 自动获得；关闭本身仍可能失败，抑制记录不是无条件成功保证。

**选择（原创分析）：** 已知静态资源集合，优先语言自带的 try-with-resources；确有动态登记且同一作用域拥有关闭责任时，考虑集合式管理。借用资源不应因登记方便就关闭。不同事务的提交/回滚、异步 close 需要另外合同，不可只用 Closeable 名字归一化。没有动态资源或重复职责时，不新建通用生命周期框架。

**反例及 Oracle：** A 成功获取，B 获取抛错，A 仍须获得关闭尝试；业务异常 E 与两个关闭异常并存时，按上述规定使用 rethrow，E（或明确的包装异常）仍为主失败；没有业务失败时第一个关闭错误为主，检查逆序、重复登记和主次异常。源不保证重复登记同对象仅关闭一次，不保证 close 成功。NOT_RUN。

## T16-C02 · C++：转移的是退出动作责任，而非任意资源的所有权证明

**真实观察：** LLVM scope_exit 按值保存 callable 和 Engaged 标志；move 构造转移 callable、保留标志，随后 release 原对象；禁止 copy、move assignment、copy assignment；析构在 Engaged 时调用回调，release 仅取消该回调。make_scope_exit 返回 decayed callable 类型的 guard。源码没有异常聚合、事务状态、外部确认或资源引用有效性的检查。

**选择（原创分析）：** 资源本来有合适 RAII 类型时直接使用它；需恢复一个局部不变量、调用非 RAII API 且存在多退出路径时，可用 scope guard。仅在真实责任交接或合法完成后解除；移动 guard 不自动延长其捕获引用所指对象的寿命。回调必须有明确的析构安全合同，不能假定析构中抛错会像普通函数错误那样传出或被聚合；该片段不提供恢复机制。异步清理若必须等待，不能仅靠此同步析构回调。

**反例及 Oracle：** 移动后原 guard 不执行、目标 guard 正常析构时执行一次；release 后不执行；提前解除后实际提交失败导致残留。测试正常退出、提前返回、支持展开的异常环境、回调异常政策、捕获寿命。NOT_RUN，不证明进程终止/断电时执行。

## T16-C03 · Rust：panic 状态不是业务成功状态

**真实观察：** scopeguard 将值 T 与 FnOnce(T) 放入 ManuallyDrop；Drop 取出两者，按 Strategy::should_run 决定调用闭包，否则局部值与闭包按正常销毁语义处理；Always 在析构运行时返回 true，use_std 下 OnUnwind/OnSuccess 分别读取 thread::panicking() 及其否定。into_inner 消费 guard，取出值并销毁闭包而不调用该清理闭包。文档明确 abort、process exit 等析构不运行场景不覆盖。源文件含正常退出、catch_unwind 与策略的测试源码，未执行；不是整个 unsafe 实现的内存安全审计。

**选择（原创分析）：** 普通拥有类型已有 Drop 时优先直接使用；一个局部动作可用 defer/guard；需要保护中途不变量且真实有所有权交接时才引入持值 guard。对于返回 Result 的业务操作，Err 并不自动意味着线程正在 panic；OnSuccess 只是非 panicking 判定，不能触发仅业务提交成功才允许的外部动作。需要该业务判断就显式确认结果及责任转移，或在已确认成功时 defuse 默认清理动作。不能将 Drop 视为可等待的异步提交或持久化协议。

**反例及 Oracle：** 函数返回 Err 而没有 panic，OnSuccess 仍可能运行；OnUnwind 不会为普通 Err 自动补偿。检查正常返回/Err、unwind、into_inner、move 后仅在实际析构时一次调用及策略条件；panic=abort 只能在独立进程验证不能依赖清理。NOT_RUN。

## 同一个工程问题的跨语言选择（综合，不增加案例计数）

情境：临时输出资源先建立，后续解析、写入、发布可能失败。仅在有权发布确证之后才允许放弃临时资源清理；借用共享资源不得被关闭。若需求为断电后恢复，则另需可恢复的持久记录与实际存储语义。以下仅为设计参考，不是所列项目实际完成此业务。

| 决策维度 | Java Closer | C++ scope_exit | Rust scopeguard | TS/JS T15-C02 |
|---|---|---|---|---|
| 执行触发 | 显式 close，按约定 finally | Engaged guard 析构 | Drop 且策略允许 | 显式同步 dispose |
| 责任表达 | 登记 Closeable，LIFO | move 转移 guard，release 解除 | 持值/闭包，into_inner 取回值 | 组合句柄登记，借用另定 |
| 主次失败 | rethrow/close/suppressor 合同 | 未提供聚合 | 未提供 Result 型 Drop 返回 | 成员错误聚合；无主业务异常优先级 |
| 完成语义 | close 尝试不等于成功 | 解除不等于提交完成 | 非 panic 不等于业务成功 | 调用不等于异步结束 |
| 何时简化 | try-with-resources | 已有 RAII 类型 | 已有 Drop 类型 | 局部 try/finally |

选择顺序：①辨认真实 Owner、借用及获取失败；②明确合法结束与失败状态，是否需要等待、逆序、保留主错误；③选择语言已有的最小机制；④仅在真实重复责任或动态资源集合存在时组合；⑤以可观察失败、清理尝试/成功及业务提交分别验证。不存在统一的“ScopeGuard 就保证事务安全”规则。进程崩溃、断电和远端 UNKNOWN 需独立恢复合同，不能由语言局部控制流替代。

## 实际来源登记

- **T16-S01** [google/guava Closer.java](https://github.com/google/guava/blob/v33.3.1/guava/src/com/google/common/io/Closer.java) · v33.3.1 · Blob b947cd0994f8f8392ba711557f95f0634476af05 · L33–76,85–117,119–187,190–255（实际读全文件）。
- **T16-S02** [llvm/llvm-project ScopeExit.h](https://github.com/llvm/llvm-project/blob/llvmorg-19.1.0/llvm/include/llvm/ADT/ScopeExit.h) · llvmorg-19.1.0 · Blob 2f13fb65d34d809583b76944c06ab4e6cd77ce41 · L26–61（实际读全文件）。
- **T16-S03** [bluss/scopeguard src/lib.rs](https://github.com/bluss/scopeguard/blob/v1.2.0/src/lib.rs) · v1.2.0 · Blob b5522c7bbdc7f12288b076b047ad041866fdb952 · L4–6,200–250,291–380,382–430,465–478,493–549（实际读全文件；测试源码仅阅读）。尝试 ref 1.2.0 不存在，采用实际可读 v1.2.0。

原定义定位：[K02.F2.2 L48](https://github.com/kaicreator-mm/tally/blob/724e83f9e0c1e515388164d949849aff233a0a21/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L48)、[K03.F2.1 L40](https://github.com/kaicreator-mm/tally/blob/724e83f9e0c1e515388164d949849aff233a0a21/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L40)、[K03.F4.2 L102](https://github.com/kaicreator-mm/tally/blob/724e83f9e0c1e515388164d949849aff233a0a21/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L102)。

反向增量：K02.F2.2 → C01 机制；K03.F2.1 → C02/C03 机制；K03.F4.2 → C01 机制、C02/C03 条件。共六条边，四机制/二条件。原有 165 概念不变；三个新 source Blob 与前期 110 个来源独立；T16 并非其他语言全部抽象机制覆盖。更复杂真实业务函数、维护证据、异步及持久化恢复仍是独立缺口。待独立限定复核；不新建 Runtime、Agent 或规则。
