# T17 · 真实业务功能抽象：订单状态、政策与模块扩展

[#572](https://github.com/kaicreator-mm/tally/issues/572) / #563；基线 ece526072ee323d7e68bc8feb9623fe3bc545fde。承接 T03/T07 的业务架构与功能，研究实际业务动作而非通用 CRUD。三个限定源码参考案例，四个新来源及一个已有来源复读；下方综合比较不另计案例。案例不是订单领域通用规范；实际授权、税务、退款、库存、事务隔离和外部效果需按本项目合同验证。所有运行/Oracle 建议 NOT_RUN。

## 增量目录

| ID / 粒度 | 实际机制及选择轴 | 原知识关系与理由 | 来源 |
|---|---|---|---|
| T17-C01 / L4 | Oscar 订单状态转移、配置式级联 | K01.F3.2 机制：合法后继与变更事件；K03.F3.1 机制：显式检查当前状态到目标状态的允许关系，不代表并发持久条件已建立 | T17-S01 |
| T17-C02 / L4 | Saleor 取消资格谓词与业务操作 | K01.F3.2 机制：状态和履约条件；K02.F3.1 机制：取消操作内集中更新订单状态及事务调用位置，不代表全部写路径受控 | T03-S06（复读）、T17-S02 |
| T17-C03 / L3 | Odoo 确认动作与库存扩展 | K02.F2.3 机制：基类显式扩展点与库存模块覆盖；K03.F2.1 机制：确认动作、局部资格检查与扩展职责分开 | T17-S03/S04 |

## T17-C01 · Oscar：配置允许转移，不代表配置已表达全部业务合同

**实际观察：** AbstractOrder.status 是字段；pipeline 从 OSCAR_ORDER_STATUS_PIPELINE 设置取得，cascade 从另一配置取得。available_statuses 返回当前状态允许后继；set_status 对相同目标直接返回，非法目标抛 InvalidOrderStatus，合法时更新内存订单状态；若有 cascade，对每个订单行仅在该行允许目标状态时保存，之后保存订单，再发送包含 old/new status 的信号（T17-S01 L109–179）。源片段未显示包围整个操作的事务、锁/版本检查、权限判断或信号可靠投递；不能据此断言整个应用不存在这些机制。

**抽象选择（分析）：** 固定几个状态且条件简单，普通业务函数加明确条件即可；若多个部署确有不同合法转移且可表达为同一张有限关系表，再抽取 pipeline。订单行与订单有独立合法状态时保留各自谓词；级联允许跳过不合格行是这里的实际机制，不是所有业务的正确策略。若合同要求全部订单行与订单状态一起变化，必须设计全量前置检查与原子持久更新，不能复用这里的跳过语义。表的配置来源必须受权，未知状态不得猜测。

**反例/Oracle（原创）：** 一个订单有两行，L1 允许 CLOSED、L2 不允许。若需求为 all-or-nothing，不能接受仅 L1 关闭而订单 CLOSED；若需求明确允许跳过，反过来不应强制全量失败。非法转移拒绝、同状态返回是否允许、状态和事件一致性、并发两个请求基于同一旧状态，分别验证。重复调用同状态直接返回并不证明并发幂等或所有外部效果只发生一次。NOT_RUN。

## T17-C02 · Saleor：可取消谓词与取消命令是不同责任

**实际观察：** 已登记的 T03-S06 Order 模型，本次扩读 L448–480：is_open 与 can_cancel 含不同条件；can_cancel 排除 CANCELED/DRAFT/EXPIRED，且检查是否有状态不在五种允许集合内的 fulfillment。can_capture 使用付款可 capture 条件与另一组订单状态门槛。actions.py L178–200 的 cancel_order 在 traced_atomic_transaction 上下文中登记取消事件、调用库存释放、设订单 CANCELED 并 save 指定字段，随后调用通知/插件相关函数。该函数片段未调用 can_cancel；真实调用链、权限、锁、事务 wrapper 的具体行为、call_event 的时点和远端结果未核对。不能由函数名证明已授权、已退款、已释放所有库存或保证消息投递。

**抽象选择（分析）：** UI 展示资格或其他只读需求确实存在时，独立可取消谓词具有意义；业务动作仍需在有权执行边界检查材料条件。谓词可能使用数据库查询，不能假设纯函数。只有一处动作且无复用需求，可把条件直接放在动作附近；多个动作只共享确实等价的条件（例如已证同义的状态排除），不把 can_capture/can_cancel 因都返回 bool 合并为相同业务政策。若执行中存在状态竞争，需在实际写边界复核/使用版本条件或锁；本案例不声称 Saleor 已提供或缺少全局保护。业务操作可组织跨模块调用，但不能仅靠通用 repository.save 替代库存和事件责任。

**反例/Oracle（原创）：** 订单最初符合取消资格，检查后新增不允许状态的履约，再提交取消。必须由当前项目合同决定拒绝、协调或合法补偿，不能使用陈旧布尔值冒充许可。再测试库存释放异常、取消事件与状态持久化的失败关系、重复请求及未知外部结果，明确实际副作用的完成证据；只有执行成功且有相应证据才能宣称释放。NOT_RUN。

## T17-C03 · Odoo：业务动作保留主合同，扩展点对应真实附加模块

**实际观察：** 固定 commit a08aa7340922e36c0303d80b87e3de14d3978714，sale_order.py L1145–1222：action_confirm 逐订单执行 _confirmation_error_message，验证分析分配，订阅消息，write 确认值，清除两个上下文 key，再调用 _action_confirm，按条件锁定、可选邮件。局部资格方法限制 draft/sent，并检查缺商品的非展示/非预付款订单行；_prepare_confirmation_values 设置 sale 与日期。基类 _action_confirm 空实现且文档明示额外单据扩展点。相同提交 sale_stock/models/sale_order.py L180–182 实际覆盖：先调用订单行 _action_launch_stock_rule，再调用 super。可证两个模块的真实扩展关系，不是仅根据名字推测。库存规则内部、具体安装组合/方法解析、事务提交、邮件与远端效果及 ACL 未核，不推断所有确认都创建同一类单据。

**抽象选择（分析）：** 单一部署、固定确认步骤时直接调用领域函数比无需求的插件注册中心简单。确有独立安装的库存模块与基线销售职责时，可采用显式扩展点、组合函数或框架既有 override 机制；明确何时进入扩展、数据状态、super/调用顺序、错误传播及依赖。不要让扩展任意绕过确认的合法性，也不要把“已写入 sale”当作事务已经提交。不能仅由实际存在扩展点推导继承普遍优于组合。

**反例/Oracle（原创）：** 混合订单集合中一个订单缺少商品（按规定非展示非预付款行），验证拒绝和是否产生前置副作用；在允许确认且安装库存模块时，记录实际扩展调用和 stock rule 失败对状态/事务的结果；无库存扩展时只验证已定义基础合同。重复确认、并发和邮件回执另测，不把 action_confirm 返回 True 当所有业务效果已完成。NOT_RUN。

## 同领域不同抽象的选取（原创综合，不计第四个案例）

| 问题 | 优先选择 | 不应外推 |
|---|---|---|
| 仅固定状态允许关系 | 局部状态谓词/有限表 | 未经授权配置即真实政策 |
| 不同部署确有变化的转移规则 | 受权的 pipeline，并保留各实体状态条件 | 表驱动自动获得跨实体原子性 |
| 同一动作需检查履约、库存等状态 | 独立可复用资格判断 + 有权执行操作 | 前置检查结果就是持续有效授权 |
| 多个业务模块需要附加动作 | 明确扩展点或组合，按真实模块变化轴 | 所有业务动作都需要插件/继承 |
| 真实事务或远端效果 | 明确状态所有者、事务与回执；未知结果保留 UNKNOWN | ORM 保存/信号/钩子名证明已提交或可靠投递 |

设计时先确定用户动作含义、合法状态及禁止状态、跨实体不变量、真实权限和错误后果；再确定谁修改哪些状态、事务与外部效果边界；最后选择最小抽象并写独立反例。不同产品的 CANCEL/CONFIRM 即使同名，也可能具有不同的状态、支付、库存及补偿合同，不宜直接合并为通用状态服务。

## 来源登记

- **T17-S01** [django-oscar/django-oscar abstract_models.py](https://github.com/django-oscar/django-oscar/blob/3.2.5/src/oscar/apps/order/abstract_models.py) · ref 3.2.5 · Blob b974b2a441b8ec501b37c076ddc8fa7a260a9405 · L109–179。
- **T17-S02** [saleor/saleor actions.py](https://github.com/saleor/saleor/blob/3.20.0/saleor/order/actions.py) · ref 3.20.0 · Blob 82f3071d50583cce26e399405fb6b0a7904ab3ca · L178–200。
- **T17-S03** [odoo/odoo sale_order.py](https://github.com/odoo/odoo/blob/a08aa7340922e36c0303d80b87e3de14d3978714/addons/sale/models/sale_order.py) · commit a08aa7340922e36c0303d80b87e3de14d3978714 · Blob ca5e4f69597b219ba189c5654c81539b48333261 · L1145–1222。18.0 分支解析为该 commit，已重新按 commit 读取并确认同 Blob。
- **T17-S04** [odoo/odoo sale_stock sale_order.py](https://github.com/odoo/odoo/blob/a08aa7340922e36c0303d80b87e3de14d3978714/addons/sale_stock/models/sale_order.py) · 同一 commit · Blob 6ef8f49cd233df9527469aa5e0fc39d28b949365 · L180–182（读取 L1–300）。

复用 [T03-S06](T03-business-and-collaboration-architecture.md) Saleor Order 模型 [3.20.0](https://github.com/saleor/saleor/blob/3.20.0/saleor/order/models.py) Blob 76f1cb1a8e921e091c6ace0dbb10110cd3b0f28e，扩读 L448–480，不新增来源计数。Spree 初始候选路径不可读，未收录。

原知识：[K01.F3.2](https://github.com/kaicreator-mm/tally/blob/ece526072ee323d7e68bc8feb9623fe3bc545fde/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L75)、[K02.F2.3](https://github.com/kaicreator-mm/tally/blob/ece526072ee323d7e68bc8feb9623fe3bc545fde/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L56)、[K02.F3.1](https://github.com/kaicreator-mm/tally/blob/ece526072ee323d7e68bc8feb9623fe3bc545fde/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L67)、[K03.F2.1](https://github.com/kaicreator-mm/tally/blob/ece526072ee323d7e68bc8feb9623fe3bc545fde/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L40)、[K03.F3.1](https://github.com/kaicreator-mm/tally/blob/ece526072ee323d7e68bc8feb9623fe3bc545fde/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L67)。

反向增量：K01.F3.2→C01/C02；K03.F3.1→C01；K02.F3.1→C02；K02.F2.3→C03；K03.F2.1→C03；六条机制关系均限于上述实际观察，不等于完整专业概念或合同已验证。旧 165 概念不变。其他业务领域、真实事务/授权/调用链与持久效果仍需具体任务补深，不为覆盖数量扩张本任务。待独立限定审核；无业务源码、运行时或新实验。
