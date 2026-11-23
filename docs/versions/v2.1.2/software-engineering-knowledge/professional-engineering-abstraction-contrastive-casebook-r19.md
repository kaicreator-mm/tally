# R19 · 工程抽象正反对照案例：学习决定条件，不模仿架构答案

> DOCUMENT_CANDIDATE / ORIGINAL_CASEBOOK / GA2_REVIEW_PENDING / NOT_EXPERIMENT_ORACLE。关联 [#554](https://github.com/kaicreator-mm/tally/issues/554)、[R19 选知与三组隔离](professional-engineering-abstraction-selection-r19.md)、[K02 信息隐藏](K02/advanced-engineering-abstraction-boundaries-and-variation-r19.md)、[K03 抽象粒度](K03/advanced-abstraction-level-selection-and-semantic-reuse-r19.md)、[K09 变更检验](K09/advanced-abstraction-evolution-and-change-locality-r19.md)。
>
> 以下均为原创虚构教学案例，代码是展示最小语义差异的 TypeScript 片段，不是可部署实现。案例故意覆盖 *合理抽象、不抽象、错误共性、过度设计*；所有判断取决于已授权合同和真正的变化方向。不含 R18 或 R19 的隐藏测试、实现参考、维护需求。不得将案例直接改名或做同构复刻后用作实验试题。

## 读案例的方法

比较两个外观相似的决策，记录真正导致不同结论的事实：稳定不变量、授权的变化轴、State Owner、错误/权限/资源语义、真实消费者与维护成本。代码行数、模块数量、设计模式名称不是正确性证据。缺少合同事实时应 ASK/UNKNOWN/STOP。

### P1：何时 DRY，何时同语法不同语义

**EX-01 · 相同业务不变量应局部复用（UNDER → JUST_ENOUGH）。** 假设绘图工具的画布与图层标题受相同规范约束：Unicode NFC、删除首尾空白、空值报相同错误，两者未来规则同步变更。

~~~ts
function normalizeLabel(raw: string): string {
  const value = raw.normalize("NFC").trim();
  if (!value) throw new Error("EMPTY_LABEL");
  return value;
}
~~~

两个 UI 入口分别复制上述行为会让修复漂移。可以在合法共同 Owner 中复用这一小函数，而让各自 UI 行为保持独立；不需要 LabelServiceRegistry。**反转条件**：若画布允许空标题或图层必须保留空白，Source/错误语义不再等同，该完整 Helper 就可能成为错误抽象。独立测试必须核对各自当前合同。

**EX-02 · 相同代码形状不能证明应合并（WRONG）。** 假设画面备注可以折叠空白，但可追溯标识必须逐字节保留；两者都由 string 输入。

~~~ts
const displayNote = (s: string) => s.replace(/\s+/g, " ").trim();
const preserveTraceId = (s: string) => s;
~~~

为了去重把二者合并成 normalize(s, mode)，并在共同层默认压缩空白，会破坏身份合同。小范围代码重复可以比错误共性更安全。**反转条件**：只有权威事实证明两者具有相同字符、错误与变更规则时才重新考虑共享；复用更底层的无业务假设纯函数是另一回事。

### P2：一个固定实现与真实变化轴

**EX-03 · 单个静态格式不需要插件注册表（OVER）。** 离线小工具只将当前已验证摘要输出为 JSON，只有一种格式、一个本地消费者，没有动态替换或独立发布要求。

~~~ts
function encodeSummary(summary: Readonly<Summary>): string {
  return JSON.stringify(summary);
}
~~~

为它增加工厂、Adapter、动态 Registry、配置 DSL 和注入容器，会制造初始化、错误定位、测试成本，而未隐藏真实变化。直接函数是合理工程选择。**反转条件**：当受权需求新增真正可替换格式、插件 ABI 或异构消费者时，抽象收益可能出现；不得永远拒绝抽象。

**EX-04 · 真实多实现且合同一致时可定义窄边界（JUST_ENOUGH）。** 假设一个 ChartSnapshot 同时需要 CSV 与 JSON 导出，三个真实消费者都使用相同输出编码合同，格式可以运行时选择。

~~~ts
interface SnapshotEncoder {
  encode(snapshot: Readonly<ChartSnapshot>): string;
}
const encoders: Record<"csv" | "json", SnapshotEncoder> = {
  csv: csvEncoder,
  json: jsonEncoder,
};
~~~

如果各消费者复制格式分支，新格式会传播到无关模块。窄编码接口可以隔离格式变化，但不能把异构错误吞为成功；**反转条件**：如果 CSV 和 JSON 并不共享输出/权限/错误合同，则需要收窄共同接口，而不是维护“所有格式都一样”的虚构抽象。

### P3：State Owner 内聚与派生读模型可拆分

**EX-05 · 一个原子状态变化应由单个 Owner 负责（错误拆分）。** 假设绘图会话中，当前画布与撤销历史必须在同一次动作里一致推进；UI 只能看见提交后的状态。

~~~ts
class DrawingSession {
  private current: Drawing;
  private history: Drawing[] = [];
  apply(next: Drawing) {
    this.history.push(this.current);
    this.current = next;
  }
}
~~~

示意代码省略构造与失败回滚，不可直接用作原子提交实现。若为了“单一职责”把 current 和 history 放进两个可被 UI 随意写入的全局 Store，就会产生半提交和撤销不一致。**反转条件**：有权需求引入独立可恢复事务日志时，实现可以分层，但依旧必须证明一个权威提交/恢复协议；不是“越少模块越好”。

**EX-06 · 可重算的只读投影可以独立（合理拆分）。** 假设图库的最近颜色视图只来自已提交、不可变的颜色变更事件，投影丢失可以重算，不允许改写真正编辑器状态。

~~~ts
function recentColors(events: readonly ColorChanged[]): readonly string[] {
  return [...new Set(events.map(e => e.color))].slice(-5);
}
~~~

把只读投影放独立模块有实际益处；但允许它反向修改编辑器权威色表就是 Owner 泄漏。**反转条件**：如果用户能独立编辑/持久化最近颜色，它就不再只是投影，需要单独明确写入 Owner。对照 EX-05 的差别是权威状态与派生视图，而非同一个文件还是两个文件。

### P4：错误共性与后续真实变化

**EX-07 · 不同错误语义被强行统一会破坏兼容（WRONG）。** 用户取消绘制是一个合法结果；编码失败需要向调用者提供可操作的错误。初期两个流程都有 try/catch，但语义不等同。

~~~ts
type DrawResult = { kind: "applied" } | { kind: "cancelled" };
type EncodeResult = { kind: "ok"; text: string } | { kind: "error"; cause: string };
~~~

使用 execute(fn): boolean 吞掉取消和错误，可能初始正例通过，但未来增加错误提示时无法恢复原语义、影响多个消费者。保留不同结果合同，可以只复用真正相同的底层错误格式化机制。**反转条件**：若有权消费者确实只关心可继续/不可继续，且原始错误已经在各自 Owner 内明确处理，较窄统一结果可能合理。

**EX-08 · 共享真正相同的纯算法，但不统一表现层（UNDER 与 OVER 的双向反例）。** 画布网格与标尺来自同一几何刻度规范，展示与交互策略不同，未来纯几何算法修复应该一起生效。

~~~ts
function tickPositions(min: number, max: number, step: number): number[] {
  if (!(step > 0)) throw new Error("INVALID_STEP");
  const ticks: number[] = [];
  for (let x = min; x <= max; x += step) ticks.push(x);
  return ticks;
}
~~~

以上代码不处理浮点误差或极端输入，仅是讲解抽象边界的示意，不能作为几何正确实现。两个 UI 各复制相同刻度算法会造成维护漂移；但把刻度计算、标签国际化、网格绘制、交互策略统统塞进 UniversalTickEngine 同样是过度抽象。共享纯位置计算、展示逻辑各自保留是更小的边界。**反转条件**：若其中一方改用非线性刻度，先重新核对共同算法不变量，不要因为已抽出 Helper 就强求一致。

## 对照汇总与抽象决定的反转依据

| 对照 | 允许共享/分离的核心事实 | 不成立时的判断 |
| --- | --- | --- |
| EX-01 vs EX-02 | 语义和变更方向相同，才能共享完整规范 | 表面相似但合同不同，保留差异 |
| EX-03 vs EX-04 | 真实替换合同与多个消费者，才值得接口成本 | 无变动证据时保持直接实现 |
| EX-05 vs EX-06 | 权威状态/事务集中，派生只读投影可以独立 | 缺失真实 Owner 时停止拆分 |
| EX-07 vs EX-08 | 错误语义不同则分离；相同纯机制可复用 | 纯算法被表现层变化污染时重评 |

## 对 #554 A/B/C 的严格输入边界

- A 只有共同 ADS、业务与质量需求、项目 H0 和合法基础知识；**不额外送达** R19 本案例库。
- B 收到 A + 已预先选定的**概念性知识切片和本案例库中固定的对照案例**，不送达 K02/K03/K09 的编号决策步骤。
- C 收到 B **完全相同的概念与案例字节**，另增加经前置冻结的显式 K02/K03/K09 决策步骤。C 的额外 Token 计入全成本。
- 必须在任何正式候选执行之前冻结 CASE ID、选择依据、真实 UTF-8 SHA256、Git Blob、送达阶段/顺序、角色访问控制；不可看到实验任务/输出后动态挑选案例。G-B 必须核查 A/B/C 实际收到的字节与访问授权。
- **案例与考题不许同构**：R19 正式任务的领域、关键变量、失败语义、后续维护变化不能直接复刻 EX-01 至 EX-08；不得把这些材料变成 Reviewer 隐藏评分标准。近似改名、泄漏或评估只奖励案例术语时，标 LEAKED/NOT_COMPARABLE。
- A→B 只能代表“概念 + 案例知识包”增益；B→C 才代表在相同知识/案例上的显式决策方法增益。**这一设计不能单独推断案例的净效应**；如需那一因果问题，必须独立增加“概念无案例”组并重新预登记样本。
- 不得更改既有 #551 R18 A/B 的语料、QA、知识时间表及实验对象。所有真实质量效益需由独立冻结的 FIRST L2、FIRST S5、陌生 Reader、后续变更和旧/新行为 Oracle 验证。

**有限研究依据**：[Evaluating Instruction-Tuned Large Language Models on Code Comprehension and Generation](https://arxiv.org/abs/2308.01240) 和 [What Makes Good In-Context Demonstrations for Code Intelligence Tasks with LLMs?](https://arxiv.org/abs/2304.07575) 研究指出 few-shot 示例可能帮助特定代码任务，但效果和示例选择有关且有失败反例；不能因此宣称这八个案例已提升 LLM 工程抽象能力。
