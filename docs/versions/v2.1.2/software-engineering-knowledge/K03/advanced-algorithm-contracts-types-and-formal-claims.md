# K03 专业基础深化：算法不变量、类型语义与形式证明的适用边界

> [K03 系统课程](systematic-syllabus.md) · [R5 并发/幂等深文](advanced-concurrency-idempotency-proof-sketch.md) · [现有学科覆盖](../foundation-concept-coverage-ledger.md)
>
> **R7 ORIGINAL_ACADEMIC_REASONING / CODE_NOT_VERIFIED / NO_PRODUCTION_TOOL_EXECUTION。** 深入现有 K03.F1.1、F1.2、F1.3、F2.2、F4.3 五项课程，并借 K05 的不同 Oracle 思想形成明确边界。下面的伪代码及数学推理不代表真实 Dafny/Z3/Lean 形式证明已经运行。

## 一、算法正确性不是“用对数据结构”

算法应满足**相对给定前置条件的后置条件**，以及需要时的终止、成本和故障边界。一个接口 `lookup(list, key)` 如果声称使用二分搜索，必须有排序前提；否则即使多数随机样例能找到 key，也不能推断算法正确。

规范候选（教学）：
```text
Input: items[0..n-1], key
Pre: items 按全序 <= 非降序排列，n 是有效有限长度
Post: 若返回 index >= 0，则 0<=index<n 且 items[index]=key
      若返回 -1，则对所有合法 index: items[index] != key
Termination: 搜索区间长度严格减少
```

以下迭代实现可以作为验证对象，`lo` 含、`hi` 不含：
```text
lo=0; hi=n
while lo < hi:
  mid = lo + floor((hi-lo)/2)
  if items[mid] < key: lo = mid+1
  else: hi = mid
if lo < n and items[lo] == key: return lo
return -1
```

此算法返回的是**第一个满足 `items[i]>=key`** 的位置，存在 key 时也是第一个重复项的索引。正确性的关键是循环不变量，例如 `0<=lo<=hi<=n`、所有 `i<lo` 元素小于 key、所有 `i>=hi` 元素大于等于 key，并在每步保持。这是一个纸面推理示例，**不是 Dafny/Lean 实际机器检查**。

**失败反例**：输入 `[3,1,2]` 未排序，查询 3，算法可能返回 -1；这不必是算法实现有 bug，而是调用方违反了 Pre。另一个危险是大整数语言中的 `(lo+hi)/2` 上溢；需要依据目标语言数值范围、索引长度以及实际内存限制判断，不能机械把 64 位固定宽度问题套到任意精度整数语言。

## 二、部分正确性、完全正确性、终止与资源成本要分别证明

- **部分正确性**：若算法结束，结果满足 Post；不保证它结束；
- **终止性**：循环的某个适当度量（variant）在有限的良基集合上严格下降；
- **完全正确性**：部分正确性 + 终止；
- **复杂度**：在指定成本模型下，迭代二分搜索时间 O(log n)，但随机访问必须是合理成本；链表每次定位中点本身是 O(n)；
- **内存与数字正确性**：栈/数组/整型溢出及浮点 NaN 需要单独定义前置条件。

不能在 B-tree 索引、网络跳表或被动读取远端数据库时仅用抽象的 `O(log n)` 就承诺尾延迟；存储层 I/O、缓存命中率和请求分布由 K10/K11 进一步评估。正确性模型和性能模型是不同且相关的两个问题。

## 三、类型只能证明它真的表达的约束

```ts
type PaymentState = "UNKNOWN" | "PAID" | "REJECTED";
type OrderId = string;
```

这样的联合类型帮助排除拼写以外的状态，但不能证明 `OrderId` 属于当前租户，或 `PAID` 的真实外部扣款已完成。把类型命名为 `AuthorizedEffect` 也不能在运行时自动授权 Actor、Tenant 和 Host。

常见设计选项：
1. 使用区分联合类型降低非法状态组合，例如区分 `PaymentUnknown` 和 `PaymentConfirmed` 的必要字段；
2. 在服务端边界进行真正的运行时数据解析/授权，不能信任不可信 JSON 已满足 TS 类型；
3. 对关键变换建立前置/后置条件，必要时建立可执行性质测试或形式化规格；
4. 对真实外部 Effect 的事实读回采用授权 Provider 合同，而非依赖本地变量赋值。

**强反例**：函数参数经 `as AuthorizedPayment` 强制断言后类型检查通过，但来源是伪造的客户端 JSON。业务安全与正确性并未改变。第二个反例是依赖的生成代码来自旧 Schema，Typecheck 可以对彼此一致但过期的一组类型通过，真实 Consumer 却不接受当前契约（K08 的输入闭包问题）。

## 四、规范与证明的边界必须可被反驳

假设想要证明 `uniquePayment(order)`。本地数据结构不变量“订单行只有一个 `PAID` 标签”不能推出真实第三方银行只有一笔扣款。形式证明只能说明在其**已建模的状态与动作**和被证明的前提下不变量成立；如果模型省略了消息重放/外部扣款或没有真实远端合同，就无法推出端到端 exactly-once。

[官方 Dafny 参考 A66](../external-source-catalog.md) 介绍 `requires/ensures/invariant/decreases` 等工具语义；但本章没有运行 Dafny，也没有提交机器验证证明。工具选择要基于真实风险和可证明需求，不必让每次普通 TypeScript 修改都进入形式化证明工具链。

**反例模型**：
```text
A: local state=UNKNOWN -> send request
Remote: commits payment
A: crashes before writing PAID
B: replays same order without provider-enforced idempotency
Remote: commits payment second time
```
若真实 Provider 用可靠业务幂等键去重，则轨迹第二笔提交可能不成立；因此这是**有条件的可证伪反例**，不是所有支付 API 必然重复收费的定理。K02 管理架构与 State Owner，K03 负责实现和形式前提，K05 负责独立 Oracle，K10 负责真实 Effect 的 UNKNOWN 恢复。

## 五、生成代码与 Build 工具的输入也属于正确性前提

算法的源代码正文可能未改，但变更了 Schema、生成器、Compiler Flags、Node Runtime 或运行环境后，旧的编译验证不能自动证明新消费者契约。若编译通过却未发现新测试，同样不能称为“全部测试通过”。专业软件构造至少区分：

`Source/Contract Pre -> Transform/Generator -> Code/Type Check -> Actual Tests -> Selected Real Consumer`。

每个箭头均需精确输入版本和合格范围。生成器的 `--cache` 可以合法复用，但 Cache Key 不包含有材料影响的 Schema 时，它不能保证语义新鲜。不要把“编译/静态分析 PASS”和“当前系统需求满足”混为一谈。

## 六、完整演算与误用识别

给定一个未排序数组、`as AuthorizedPayment` 的 TS 类型断言、以及一次远端支付超时，错误的答复会说“二分查找 O(log n)，TS 编译通过，订单是 PAID，所以端到端正确”。专业回答必须：
- 检查预置排序合同与实际输入；
- 分开算法结果、终止、成本模型与类型系统可保证的范围；
- 对跨进程付款保留 UNKNOWN，除非有合法外部结果/幂等合同；
- 指定最低的性质/边界反例和真正需要的 Consumer/Effect Oracle；
- 只在相应事实当前可核查时标限定 PASS。

**STOP**：模型中不存在的外部动作/权限不能凭形式符号“证明”；不为普通无关计算强制全流程数学验证。引用 [A01 SWEBOK v4 计算/数学基础](../external-source-catalog.md)、[A50 TLA+](../external-source-catalog.md)、[A66 官方 Dafny 文档](../external-source-catalog.md)。所有算法、TS 类型、状态轨迹均为本章原创教学候选。
