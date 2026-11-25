# T25b-3 · 跨语言值、所有权、泛型约束与失败语义

#622 / R4-F1 / FM-05；K03 主责语言语义，K05 验证，K02 责任边界。复用 [T25b-1](T25b-1-language-value-and-text-semantics.md)、[T27a](T27a-detailed-design-representation.md)、[T27c-1](T27c-1-concurrency-task-lifetime.md)、[T16](../casebook/waves/T16-cross-language-scope-exit.md) 中 JS/Python 值/别名、frame、资源退出和真实 Java/C++/Rust 局部观察；新增 Go/Rust 跨语言具体选择，非全语言百科。以下代码为纸面教学，未编译/执行。

## 1. 业务责任不等于语言所有权

教学任务：解析只读输入 bytes、产生候选记录，随后校验和发布。输入由调用者拥有，解析器不得改变输入；若需跨异步任务保留，明确复制、所有权转移或有效共享；发布权限另由 domain/执行合同决定。TS readonly（已有 T25b-1）是静态访问约束，不表示深度冻结或其它别名无法变更；Go slice 值是底层数组的描述，复制 slice 可共享底层存储；Rust String/Vec 非 Copy 值通常 move，借用允许临时访问而不转移所有权。不能将一段 Go/JS 的“引用复制”机械翻译为 Rust move，或把 Rust 借用检查当远端业务合法性证明。

Go spec 教学：
```go
a := make([]int, 1, 2)
a[0] = 7
b := a
b = append(b, 9)
b[0] = 8
// a[0] == 8; len(a)==1, len(b)==2
c := make([]int, len(a))
copy(c, a)
c[0] = 10
// a[0] == 8
```
这里容量足够故 append 复用底层数组；若 cap=len 且追加一个，append 会分配新数组，之后写新 slice 不修改原数组。仅限制 cap 不阻止对已有元素的修改；copy 是元素拷贝，元素本身含引用时仍可能共享深层对象。并发访问另核同步，不能靠 GC 或 slice 类型保证无数据竞争。

Rust Book 教学：`let a = String::from("x"); let b = a;` 之后不能继续使用 moved a；若只是读取，应以 &str 或适当引用传入；需独立修改副本时显式 clone（不假定任意 Clone 深拷贝）；需要转移资源用拥有值的参数，明确返回/失败时是否移交。对同一数据同时存在冲突的活跃可变借用会被安全 Rust 拒绝；非重叠借用、内部可变性/unsafe/FFI 另有合同。语言能约束内存访问，不证明对文件/服务的合法写权。比较读取临时 buffer、跨任务保存、独立副本和独占更新，按真实生命周期选择，不因统一 interface 强制 clone 所有数据。

## 2. 泛型表达关系，而非业务等价

Go type parameter constraint 以类型集合限制可操作的值；如 `func First[T any](xs []T) (T, bool)` 需要定义空输入返回零值和 false，any 不允许任意算术或比较。若需排序，约束应与实际比较合同一致（注入 Comparator 允许业务 locale/单位语义）；类型有序不保证 NaN 或 locale 排序满足业务要求。Rust `fn pair<T: Display>(a: &T,b: &T)` 将二者约束为同一具体 T；两个独立 impl Trait 参数可允许不同满足 trait 的类型（Book 的 trait-bound 语义）。泛型共享的是真正相同的值操作合同，不是把订单/退款不同政策合并为 T。TS `as T` 不执行真实网络输入校验；任何语言的反序列化/FFI 输入都须按实际表示、域合同和错误验证。

## 3. 求值、异常、清理与跨边界

已有 T16 实际 Guava Closer 保留主异常、LLVM scope_exit 转移退出动作、Rust scopeguard 的 OnUnwind/OnSuccess 只是 panic 状态，不是业务成功。Go 普通 error 是返回值，panic 展开同 goroutine 的 defer，recover 只能在相应 deferred 调用中正确使用；不把另 goroutine 的 panic 当调用者正常返回。defer 参数求值与实际调用时机不同；JS finally/Promise 拒绝及 Python context manager 另按具体合同，不把同步退出等同远端撤销。若解析器只完成内存操作，纯 Result/error 适合显式业务拒绝；资源或程序异常应保留首错和实际清理结果，不把 panic/异常统一变成 success。跨语言边界明确编码、内存所有权、谁释放、可重入/线程、错误值及生命周期；不是所有错误都能作为另一语言异常直接跨 ABI。

教学：先获取临时文件，再申请独占发布资源；后者失败时只释放已取得资源且不删除调用者原文件。合法发布后 lost ACK 仍 UNKNOWN（T27a/CON）；作用域退出不能决定是否重复发布。若 Go 代码在 defer 闭包使用可变句柄变量，核实际捕获值与取得的资源；若将 Rust guard 移走，核清理责任随值转移。异常同时伴随关闭失败须按语言 API 记录首错及次错，不能为了统一 error 类型丢掉失败阶段。

## 4. 独立 Oracle（NOT_RUN）

| 实际候选 | 预期/区分性反例 |
|---|---|
| Go slice alias/append cap足够 vs不足 | 分别观察底层原元素、len/cap、深层引用；复制值不必隔离，实际副本保留原值 |
| Rust move/borrow | 固定工具链验证合法借用/非法 moved 使用和同时冲突借用；实际资源所有权与业务权限分别检查 |
| 泛型约束与输入 | 不合法类型编译拒绝；空集合、外部恶意/非法字段、不同单位/业务政策仍需运行时检查 |
| 部分获取/多失败 | 每个已获取资源恰当清理、未获取不关闭、借用资源未释放、主失败可诊断；取消/进程中断另验证 |
| 语言/ABI 迁移 | 相同真实输入/旧消费者、错误与可观察 frame，不仅类型检查；实际适用语言版本与工具链固定 |

实际资料 2026-11-26：[Go language spec](https://go.dev/ref/spec) 页面标记 go1.27（May 26, 2026），Slice types L514–522、Type constraints L1257–1259、append/copy L3245–3264、Handling panics L3427–3437、defer L3220–3223；[Rust Book Ownership](https://doc.rust-lang.org/book/ch04-01-what-is-ownership.html) L151–162、249–299；[References and Borrowing](https://doc.rust-lang.org/book/ch04-02-references-and-borrowing.html) L101–163；[Traits](https://doc.rust-lang.org/book/ch10-02-traits.html) L247–274，在线书未固定编译器发行号，不宣称具体所有 rustc 版本或执行。Go spec 为在线当前页面，不将新版特性强制到旧项目；实际目标 compiler 需另核。FM-05 限定目标候选，未新增真实案例。
