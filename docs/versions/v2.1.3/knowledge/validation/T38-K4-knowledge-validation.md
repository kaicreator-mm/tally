# T38-K4 · 业务决策、精确金额及复杂功能组合知识的最小核验

#644 / #563；冻结输入 8b4594b1a27852c294a379e345bf6f59451c7545；[Owner方向](https://github.com/kaicreator-mm/tally/issues/563#issuecomment-7190227558)；[K3独审](https://github.com/kaicreator-mm/tally/pull/643#issuecomment-7191421210)及[收尾](https://github.com/kaicreator-mm/tally/issues/642#issuecomment-7191709172)。K1/K2遗留均关闭，无新发现待修。当前 KNOWLEDGE_CONSTRUCTION；本批只验证有限知识命题，不建设业务应用或验证平台。以下四个原文均实际读自冻结基线，路径相对 fine-grained/，原字节不修改。

## 1. 最小实际观察 A–C

| ID | 原知识路径、Blob及范围 | 独立预期、观察与界限 |
|---|---|---|
| A | T24b-1-business-rule-decision-contracts.md fdb4bd53b58f5ab432d34253a75e562f0ae29df2 L7–35 | 合法已知三位域8项，政策仅111允许；按最早假条件划分4+2+1+1，恰一命中。实际逐项将表匹配结果与合取比较；添加C=1允许造成011两结果，删除R3造成110无结果，改R4为DENY即使仍恰一命中也与独立合同矛盾。有限补全中A=0全DENY，A=B=1时C未知同时有ALLOW/DENY；前者只有在适用性、已知禁止权威及非空合法补全确立时才能拒绝，后者必须ASK/UNKNOWN。None在脚本仅是规则通配符，实际输入皆0/1；未实现三值逻辑或DMN/FEEL。 |
| B | T25b-4-exact-numbers-locale-bidi.md b7cad18c0201941f038b51ceb66aee9a6dbe2ed9 L5–13 | 从精确有理数推导267.5分/−267.5分 half-even到268/−268，朝零到267/−267；三笔33.5分逐笔34分合102分，总100.5分half-even为100分。实际Decimal字符串验证上述固定结果，float 2.675转Decimal后舍入为2.67（本运行时的二进制输入，不是所需十进制2.675）；Inexact trap接受1.230量化1.23、拒绝1.231；3位有效精度(1000+1)−1000实际0，精确整数1。精确表示不替代舍入阶段、单位、异常、范围和业务政策；不是财务产品或全decimal验证。未运行Locale/UCA/BiDi。 |
| C | T35-4-complex-business-function-composition.md 903d9d7f170ae87f752d7b1025fab01d27fd703b L9–21；T35-2-business-function-knowledge-composition.md 02eba8247e89710357a2665ff40449f7aeba787f L20–37 | O1后可用X2 Y2，另一个原子订单要X1 Y3，联合合同拒绝后必须仍X2 Y2。实际内存SQLite已提交初始两行，同连接顺序条件更新X成功、Y不足影响0行、显式异常导致回滚，两行仍2；负例未传播拒绝而提交，X1 Y2，违反整单原子合同，即使每行非负。仅见证局部可用量两行，不建订单、预留、租户表或并发算法；不证明真实PG隔离、故障持久性、授权、完整联立状态或跨库。 |

合法反转：A 若获权政策明确允许多个命中且完整结果一致或获权汇总，重叠未必错误；First只有在政策授权顺序时适用。没有合法事实补全时报告冲突，不能利用空全称。B 若另一获权政策按总额舍入则1.00正确、逐笔政策1.02正确，不能由技术库决定；1.230与1.23数值相等不表示输入词法相同。C 若明确授权部分预留且有部分结果与后续取消合同，则X1 Y2可以是合法部分效果；不能用该变体放宽原整单合同。以上反转均纸面。

## 2. 组合知识的有限纸面推导 D–F（实际目标执行 NOT_RUN）

D · 业务动作、库存、退款及对账（T35-2 L20–46；T35-4 L9–46）。B1 两租户同名X仍不同身份；A总5，O1要3、O2要4，合法先提交者成功后另一不足，不能由测试固定赢家。通用changeStatus若允许先CANCELED再独立释放/退款，崩溃窗会产生取消但预留仍存在；同模块独立业务动作及共享精确值是候选，不是必须拆微服务或禁止复用。B3 教学O1 X3@1000、Y2@500，capture4000。预留后(X在手5,预留3,可用2)，(Y4,2,2)；发X1 Y1后(4,2,2),(3,1,2)；取消X1 Y1后(4,1,3),(3,0,3)；再取消X1后(4,0,4),(3,0,3)。取消不是增加在手或退货；X的3=0+1+2，Y的2=0+1+1。R1=1500 UNKNOWN仍占P1500，R2=1000新合法取消后P2500；R2确认时S1000 P1500，R1实际确认时S2500 P0，净确认1500。不由HTTP码、取消UI、未知状态释放额度；若仅看总额，两个同金额但不同取消身份/退款资源的对应互换仍可通过总和，逐键关联必须拒绝。额外外部退款500的来源未获权时差异500并冻结自动发起，不创造虚假账务。Stripe API v1在实际端点开始后缓存首次响应含500，清除旧key后重用可能新请求；没有保存结果的参数校验/并发冲突不等同已执行。必须核实际provider、身份和保留期及授权；未知或过期不得凭新key重发；实际外部资金、授权、并发、对账、Stripe API全部NOT_RUN。本例假定唯一退款发起权、固定金额和无税费折扣；跨仓独立权威或获权折扣分摊是重新澄清与选择机制的理由，不是把原事务/单价公式套用。

E · 规则版本、委派与批准（T24b-1 L27–35；T35-4 L52–60）。9件在6月30日受理、7月1日提交，指定提交时刻且V2最大8的政策下拒绝；若授权锁受理版本V1且其它条件成立才可允许，不由技术时间默认。B4 E以G在[10:00,12:00)被授予部门动作，金额9000≤10000、同租户、revision7且实际E非U：11:59:59提交可合格（仍核其他条件）；12:00才提交必须拒绝；已获权提交的历史不因到期回写取消。E兼财务角色也不能替F完成第二阶段，因为实际actor重复；若U=E则第一阶段即拒绝。若撤权先于提交或revision变8，旧许可/批准不能推进；金额11000超G范围，F不得沿用v7批准，保留历史。若E/D并发，仅一合法决定可推进，不能仅靠显示stage或决策ID字符串判断，需权威原子提交边界及身份/输入一致。若有权非实质修改不重启或单人审批可另选更简单合同；不是本例通用化。未建授权系统或测真实并发。

F · 快照、字段权限与流式交付（T35-4 §2.2 L64及后续）。获权S1 A的p1=9000,p2=3000两行12000，B的q1=7000排除；后续p2=4000不改变S1，混用得13000违反同快照；混租户得19000也错误。生成READY不等于获权披露。以本教学合同的chunk准入与撤权实际有序为前提，epoch4已准入p1在途可以晚到；epoch5撤销p2或金额字段后旧制品禁止新准入，即使READY；新授权只产生新身份、筛选和制品。只开始时验一次签名或不受控URL/Range路径不满足此合同，UI丢弃迟到响应也不能撤回已披露信息。若实际权威无法保证准入与撤权的顺序，不得声称满足，须另获权较弱合同或改变机制；不承诺物理字节瞬时回收。旧快照读取正确性和当前授权两个独立条件。短小查询若获权且有界，可不建后台Job；本例不是通用导出政策。未执行HTTP、PG快照、授权/缓存/流式传输或真实数据泄漏测试。

## 3. 外部资料核对与实际收据

本批定向复核 [Camunda 8.10 hit policy](https://docs.camunda.io/docs/components/modeler/dmn/decision-table-hit-policy/) L94–138（Unique可零命中，Any同结果，First按顺序，Rule order保持规则顺序），[Python 3.13.18 decimal](https://docs.python.org/3.13/library/decimal.html) L24–50（context、signals、构造），[Stripe API v1 idempotency](https://docs.stripe.com/api/idempotent_requests) L16–23及[refund create](https://docs.stripe.com/api/refunds/create) L10–15、61–62。仅相关文档语义，不重读全部外部来源、运行真实引擎或提升资格。

[T38-K4-witness.py](T38-K4-witness.py)：66行/3056字节，Python3.13.5 SQLite3.46.1 sys.flags.optimize=0；无外部依赖、网络或产品代码。首轮本地未提交初始化DML，C异常回滚同时撤销初始数据，断言失败，实际退出1；修复为显式db.commit()建立基线后重跑（这是真实见证构造错误，不是原文错误，进一步说明事务边界须明示）。最终 `python -m py_compile T38-K4-witness.py` exit0，`timeout 10 python T38-K4-witness.py` exit0；实际 Git blob fefb68209fb86b776c74250028598993df95bd3e，SHA256 f6bd28829d756b253006936d61e019d7b86e0a2ca914e92337c92b798490131e；GitHub读取同Blob。实际stdout：

```text
A eight known combinations; overlap, gap, wrong output; partial completions
B decimal rounding point, float conversion, Inexact and intermediate precision
C same-DB rollback preserves both quantities; partial commit violates all-or-nothing
```

未发现所选原命题实质错误；A–C固定机制实际观察，D–F独立条件推导，不能把脚本断言/纸面场景当实际业务证据或完整知识库验证。不新增Case/Claim/Rule或改变67/86、46/265、S51/D0/U5资格；不修改旧见证、原章、产品代码。独审仅本批正确性、合理性、边界与源反馈。#565 OPEN/HOLD #286 DEFERRED A24/T4 NOT_RUN/HOLD；不启动Package/DH/SDK/CI/Agent/LLM实验。后续按知识疑点定向核验或填补真正缺口，不要求无限测试。
