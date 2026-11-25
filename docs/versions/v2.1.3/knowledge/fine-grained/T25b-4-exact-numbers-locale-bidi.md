# T25b-4 · 精确数值运算、Locale 排序与双向文本

#622 / R4-F2 / FD-02；K03 数值/文本机制，K01 有权业务单位和舍入政策，K05 验证，K02 数据身份。复用 [T25b-1](T25b-1-language-value-and-text-semantics.md) Number/BigInt/定点基础、Unicode 表示/规范化与 [T25b-2](T25b-2-time-clock-calendar-semantics.md) 时间、[ABS R19-F2](../../../v2.1.2/software-engineering-knowledge/K03/advanced-abstraction-level-selection-and-semantic-reuse-r19.md) 中间安全整数反例。新增实际 Decimal、locale 排序及 BiDi，不改变原身份、政策或语言规范。示例均原创教学 NOT_RUN。

## 1. 精确表示仍须定义运算及舍入合同

先确定单位、比例、输入域、范围、结果比例、允许舍入的阶段/模式、溢出/异常、跨边界保真和独立对账。整数最小单位适于已知固定比例且加减居多；任意精度整数仍有资源/输出范围限制；Decimal 有十进制表示与明确 context，但除法/乘法/中间舍入仍可能不精确；物理测量可按有权误差模型用浮点。不可只说“使用 decimal 即准确”。

Python 3.13.18 decimal 文档：从字符串构造保留所给十进制数，从 float 构造则准确转换其实际二进制浮点值而不是恢复期望十进制；context 控制运算有效位精度、指数范围、舍入和 signals/traps，构造位数不直接受算术 precision 截断；flags 需显式清理后观察；quantize 可设定目标指数，Inexact trap 可拒绝需要丢失非零位的输入，但不能仅凭此拒绝多余零或非规范原始字符串。NaN/Infinity/负零按实际业务合同明确拒绝或处理。构造混合 float 可用 FloatOperation trap 辅助，不能将它当所有转换/比较的统一禁令。

教学假定金额合法精度分，输入字符串“2.675”“-2.675”，context 足够，明确 ROUND_HALF_EVEN 得 2.68、-2.68；若正负均朝零截断则 2.67、-2.67。另一例三个 0.335：逐笔 half-even 得 0.34×3=1.02；精确和1.005再 half-even 得1.00。两者相差0.02，必须由业务方选择合法舍入点，不宣称其中一种是行业规则。若政策要求逐笔与总额对账，明确有权分摊余数规则、稳定顺序及负数/退款反转。Decimal context=3 个有效位时 (1000+1)-1000 可在第一次运算舍入为1000，最终0；先做精确整数或足够精度中间值则1。仅最后 quantize 无法修复已经丢失的中间信息。

定点教学：整数 cents=9007199254740991 (JS 最大安全整数)，再加2 的精确结果9007199254740993超出安全整数，不能用 Number 完成并再转 BigInt。须在输入保真时转换为 BigInt/精确表示，并验证中间乘积及目标域；乘法、除法、负数及取整必须实现获权舍入合同，不把语言整数除法当通用财务策略。适合明确单位的纯金额 helper；若两个领域的比例/舍入/异常或授权不同，则只共享实际相同的机制而非共用政策。

## 2. Locale 排序不是身份相等

T25b-1 已区分 NFC/NFKC 和 grapheme，本节仅补比较与显示。UTS #10 rev53 / Unicode17 的 collation 使用按等级的比较规则及表，可按 locale tailoring 改变顺序；规范等价序列应有同一排序；收缩/扩展意味着不能将每个字符独立按 code point 排序替代。显示排序可忽略某级差异（依选定 locale/strength），但排序比较相等不意味着原始 ID、授权资源、业务键或密码相等。持久分页必须在同一实际数据版本、locale/排序参数/引擎版本下使用稳定排序与唯一 tie-breaker，不能按旧排序 cursor 接新规则。

教学：两个不同获权记录 id=101 与102，名称分别为 U+00C5 与 U+0041 U+030A；按 UCA 规范等价排序相同，若业务只按 id 定义身份，两条均保留；以 (collationKey,name-independent unique id) 排序，第一页大小1返回101后第二页仍应有102（假设两名称相同 collation key、id升序）。错误实现按 name compare==0 去重或下一页仅查 collationKey>last，会丢102。若业务明确按 NFC 名称作为唯一键，则两者冲突应由业务合同裁定，而非自动保留或合并。变更 locale/collation strength 或引擎/CLDR 版本前，检查实际排序、唯一约束和索引/cursor 重建；不要假定所有实现具体 locale 一致。排序权重或 sort key 可依实现/版本，不应作为长期跨版本身份。

## 3. 双向文本：逻辑顺序与显示顺序

UAX #9 rev51 / Unicode17 规定内存中逻辑顺序，双向算法改变呈现顺序而不改变逻辑解释；LRI/RLI/FSI 与 PDI 的 isolate 控制嵌入片段及其范围。多语言 UI 应按实际渲染框架/文本方向支持隔离显示，不能手工倒置字符串或从视觉位置重建标识。例：逻辑数据由 label、一个 RTL 名称、ASCII id“INV-12”及独立金额构成；显示层应隔离名称方向，id 和金额仍从原字段取，不通过抓取视觉字符串解析提交。实际渲染取决于段落方向/字符类型及实现；本章不猜测所有字符在所有布局的具体位置。对于可信机器 ID 与日志字段，可按获权语法拒绝/转义不允许的方向控制字符；不把所有合法自然语言控制字符一概删除。复制/搜索、光标/选择与实际逻辑值的合同按真实 UI 验证，未检查平台专项仍 U。

## 4. 独立验证（NOT_RUN）

| 测试 | 独立预期 |
|---|---|
| 精确字符串 vs float 中转 | 原始数值与实际运算类型，输入规范/特殊值/范围；不能由同一被测转换生成 Expected |
| 逐笔/汇总、正负及中间舍入 | 独立精确整数/有理数纸面推导，2.68/-2.68、1.02 vs1.00、context3 中间丢失；只接受已获权政策 |
| 边界与跨语言序列化 | 超安全整数、乘积溢出、单位错误、NaN/Infinity、负零；拒绝丢精度或保真传递 |
| Locale 相等但身份不同 | 同观察快照下 101/102 均返回，实际排序参数、tie-breaker 和 cursor；规则变化需明确失效/重新查询 |
| RTL 名称+ASCII id | 实际界面方向与隔离、复制/原始值、存储/授权/日志；不以截图正常证明逻辑 id 或权限正确 |
| 真实消费者变更 | 固定库/Unicode/locale/运行时版本及获权输入语义，旧合法消费者和负例；真实工具未运行不宣称支持 |

实际资料 2026-11-26：[Python 3.13.18 decimal](https://docs.python.org/3.13/library/decimal.html) L26–43、43–77、168–189、663–667、1433–1465；[UTS #10 rev53](https://www.unicode.org/reports/tr10/tr10-53.html) Unicode17 L151–206、1055–1074；[UAX #9 rev51](https://www.unicode.org/reports/tr9/tr9-51.html) Unicode17 L73–82、122–140。实际阅读上述机制；不声称特定库完整 UCA/BiDi/CLDR 合规或具体 runtime 实测。数值推导和 UI/cursor 例为原创有限教学，FD-02 扩展目标候选；所有 Oracle NOT_RUN，无新 qualified Claim/Case。
