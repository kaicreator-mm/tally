# T03 · Web、ERP、电商、协同与 CMS 架构参考

[#567](https://github.com/kaicreator-mm/tally/issues/567)；8 条参考、9 个实际文件、7 个项目。五类：Web 框架、企业管理、电子商务、团队/文件协同、CMS。L2–L4 皆为限定子系统/模块视图，不冒称 L1 全量产品审计。真实观察与本库参考结构、替代方案分别表达；所有验证建议 NOT_RUN。部署/分布式、完整业务事务及权限未从局部源外推；未读取范围明确 UNKNOWN。不复制项目代码。

## T03-C01 · Rails：MVC 与持久化模型的关系

**分类：** Web 框架 / L2 / PROJECT_DOCUMENTED。**功能/机制：** 路由、Controller、View、Model。**知识：** K02.F2.1 K03.F2.1 K02.F2.2。**来源：** T03-S01、T03-S02。

- **实际观察：** 指南描述路由、动作、控制器、视图和模型的职责；Active Record 将数据访问与对象行为结合，Active Model/普通对象也可属于模型层，不要求一切模型对应数据库表。
- **职责与数据/控制视图（分析）：** HTTP 路由→Controller/action→模型行为→View/响应；持久模型由 ORM 连接数据库。领域模型不等于数据库访问接口；同一进程内即可分离职责。
- **适用边界：** 常规 Web 表单/CRUD 与有实际领域对象的业务；复杂聚合、远端多系统副作用不应仅因框架惯例塞进单个 ORM 模型。
- **备选与反转：** 直接 CRUD、Active Record、独立领域对象+repository；按复杂度、事务和独立变化决定，不先引入微服务。
- **验证/反例（NOT_RUN）：** 非法输入、真实路由/控制器、事务失败和回显；没有读取完整框架调用链，部署细节未知。

## T03-C02 · Frappe：文档动作与公共生命周期

**分类：** 业务框架/ERP / L2–L4 / SOURCE_OBSERVED_SCOPED。**功能/机制：** 保存、提交、取消、生命周期分派。**知识：** K01.F3.3 K02.F2.1 K03.F5.1 K08.F4.1。**来源：** T03-S03。

- **实际观察：** Document 在 before-save 阶段区分 save、submit、cancel、update-after-submit，各分支调用不同生命周期方法；存在 ignore_validate 路径，旧文档读回使用 for_update。
- **职责与数据/控制视图（分析）：** 框架承担动作分派与文档持久化契约；业务控制器拥有特有校验和副作用。动作与可见业务状态须分别核对；不能把任意内部跳过校验当正常用户权限。
- **适用边界：** 多个文档类型有真实共用动作；单一数据表/纯配置无需引入通用文档生命周期。
- **备选与反转：** 直接 CRUD、显式命令/状态机、公共文档 lifecycle；共享阶段机制不共享具体业务政策。
- **验证/反例（NOT_RUN）：** 保存未提交、取消与修改已提交、绕过标志、旧状态读取失败；源仅为局部方法，不证明全部权限/事务。

## T03-C03 · ERPNext：库存单据作为跨业务操作的协调边界

**分类：** ERP/生产/库存 / L2–L3 / SOURCE_OBSERVED_SCOPED。**功能/机制：** 库存账、工单、财务凭证、取消。**知识：** K01.F3.3 K02.F3.1 K03.F3.1 K09.F2.1。**来源：** T03-S03、T03-S04。

- **实际观察：** StockEntry.on_submit 调用库存账、工单、分包/拣货、GL、项目成本及质量等更新；on_cancel 有独立逆向步骤、取消 GL、批次/关联单据清理与状态调整。
- **职责与数据/控制视图（分析）：** 库存动作→领域协调→库存账/工单/财务等各自职责；取消不是删行。共享生命周期来自框架，但实际业务步骤和顺序属于库存控制器。
- **适用边界：** 有库存、成本及制造联动的 ERP；简易库存计数不必复制整套总账协调。
- **备选与反转：** 单数据库事务内协调、领域服务组合、跨系统时明确可恢复消息与补偿；事务与部署边界须按项目确认。
- **验证/反例（NOT_RUN）：** 中间步骤失败、重复提交、取消与后继业务冲突、库存与财务读回；调用序列不证明各操作原子性，未读完整事务/总账实现。

## T03-C04 · Saleor：Checkout 与已成立 Order 的模型边界

**分类：** 电商 / L2–L3 / SOURCE_OBSERVED_SCOPED。**功能/机制：** 结账、订单、行项目、身份。**知识：** K01.F3.3 K02.F2.1 K02.F3.1 K03.F2.1。**来源：** T03-S05、T03-S06。

- **实际观察：** Checkout 有 token、channel、用户/邮箱、账单/收货地址、运输/仓库选择；Order 有独立 UUID、订单号和状态。CheckoutLine 允许相同变体但不同 data 的多行；OrderLine 绑定 Order。
- **职责与数据/控制视图（分析）：** 候选购物/结账模型与已成立订单分别建模，转换时确认实际规则；稳定身份与渠道、快照、行等事实不能由相似字段自动等同。
- **适用边界：** 订单成立后有独立生命周期、审计/履约及价格合同；简单购物清单无需提前构建结账/订单两套复杂服务。
- **备选与反转：** 共享金额/地址纯机制，独立结账/订单政策；或简单单状态实体（仅有权语义确实相同）。
- **验证/反例（NOT_RUN）：** 相同 SKU 不同选项、渠道变化、结账重试、订单身份、历史数据；当前只读模型声明，不推断实际转换、库存扣减或快照实现。

## T03-C05 · Saleor：订单、授权与扣款的正交状态

**分类：** 电商/支付 / L3–L4 / SOURCE_OBSERVED_SCOPED。**功能/机制：** 订单状态与支付状态分离。**知识：** K01.F3.3 K02.F3.1 K03.F3.2 K05.F2.2。**来源：** T03-S06。

- **实际观察：** Order 分别声明 status、authorize_status、charge_status，后两者各有独立默认值及 choices。
- **职责与数据/控制视图（分析）：** 不要用单一 success 或枚举排列代替所有业务维度；订单履约与支付授权/扣款是不同观察。聚合展示由有权业务规则决定，不改变实际 Owner 状态。
- **适用边界：** 有预授权、部分扣款、履约与退款等独立变化；无支付的单步骤操作可以单状态。
- **备选与反转：** 独立状态维度、明确受权状态组合、必要时专门聚合视图；避免每个笛卡尔组合预建巨型枚举。
- **验证/反例（NOT_RUN）：** 有授权未扣款、扣款失败与履约未知、重复回调；模型声明不证明允许哪些具体组合或财务行为，需读取政策/执行。

## T03-C06 · Mattermost：频道业务变更与实时事件分离

**分类：** 团队协作/聊天 / L2–L4 / SOURCE_OBSERVED_SCOPED。**功能/机制：** 团队/频道、业务动作、WebSocket 通知。**知识：** K01.F3.3 K02.F3.1 K03.F3.2 K10.F3.2。**来源：** T03-S07。

- **实际观察：** CreateChannelWithUser 拒绝 direct/group 或缺 TeamId，检查团队频道上限，设置 CreatorId，调用 CreateChannel，随后查用户、发加入消息和发布频道创建事件；RenameChannel 拒绝 direct/group 类型。
- **职责与数据/控制视图（分析）：** App 业务操作→频道创建→后续用户/消息与事件发布。实时事件是通知，不自动成为存储真值；步骤间失败需明确返回合同和读回。
- **适用边界：** 协作系统有多用户实时视图、类型/配额差异；本地单用户项目不用为每个保存部署消息总线。
- **备选与反转：** 同步服务+实时通知、可靠事件/outbox（有可靠投递要求时）、直接查询；不能由此片段断言 Mattermost 已采用 outbox 或事务。
- **验证/反例（NOT_RUN）：** 创建后查用户失败、频道上限并发、重复事件、重命名类型限制；实际数据是否已提交及权限/一致性未验证。

## T03-C07 · Nextcloud：协同文件的节点合同与存储边界

**分类：** 文件协同/云盘 / L3–L4 / SOURCE_OBSERVED_SCOPED。**功能/机制：** 文件节点、移动/复制、后端、失败。**知识：** K02.F2.2 K03.F4.2 K03.F5.1。**来源：** T03-S08。

- **实际观察：** Node extends FileInfo，声明 move/delete/copy/touch/getStorage/getParent 等；move 注明路径、缺失、权限及锁相关异常。
- **职责与数据/控制视图（分析）：** 调用者依赖文件语义合同，节点关联具体存储资源；路径合法性、权限、锁与对象是否存在是不同失败。共享路径处理不等同共享所有后端的提交保证。
- **适用边界：** 多个消费者需统一文件/目录操作；单纯本地路径工具可直接使用平台 API。
- **备选与反转：** 文件系统 facade、面向对象节点、能力分离 adapter（对照 AB-009）；按真实后端和副作用条件决定。
- **验证/反例（NOT_RUN）：** 权限拒绝、锁冲突、移动失败后的读回、跨后端操作、旧路径；接口声明不证明所有后端支持原子移动，也不覆盖全 Nextcloud。

## T03-C08 · WordPress：内容类型、可见性与路由生成

**分类：** CMS / L2–L3 / SOURCE_OBSERVED_SCOPED。**功能/机制：** 内容模型、独立能力标志、路由。**知识：** K02.F2.3 K03.F2.1 K01.F3.3 K09.F3.1。**来源：** T03-S09。

- **实际观察：** WP_Post_Type 的 public、hierarchical 等属性表达不同意图；注释明确 search/query/UI/nav 的默认可由 public 推导但分别控制。add_rewrite_rules 按 query_var、viewable、rewrite 等条件添加规则。
- **职责与数据/控制视图（分析）：** 类型描述→各消费功能（后台、查询、导航、rewrite）。默认推导和最终授权分开；共享类型元数据不代表可见意图完全相同。
- **适用边界：** 扩展 CMS 的内容类型和各入口差异；固定单类型项目不需动态类型 registry。
- **备选与反转：** 直接字段、显式独立能力配置、稳定扩展点；仅在真实共同语义时共享单一 public 标记。
- **验证/反例（NOT_RUN）：** 公开查询但后台隐藏、排除搜索、层级路径、默认与显式覆盖；不把 UI 可见性当访问授权，未运行路由或完整权限测试。

## 实际来源登记

- **T03-S01**：[rails/rails / guides/source/getting_started.md](https://github.com/rails/rails/blob/v7.2.2/guides/source/getting_started.md) · ref `v7.2.2` · Blob `ecfa818aad26f171c5b45c4b7dc982d86d485a1b` · L349–380。项目教学文档 MVC/路由/模型。
- **T03-S02**：[rails/rails / guides/source/active_record_basics.md](https://github.com/rails/rails/blob/v7.2.2/guides/source/active_record_basics.md) · ref `v7.2.2` · Blob `4def6d157f8ad3a78dbe0916ad7b2067ed399f4a` · L29–80。项目文档 Active Record/Active Model/ORM。
- **T03-S03**：[frappe/frappe / frappe/model/document.py](https://github.com/frappe/frappe/blob/v15.0.0/frappe/model/document.py) · ref `v15.0.0` · Blob `1b7a97cc9711b8973ea9f5811b1086948ece9413` · L87–97, 336–345, 1061–1103。文档控制器保存/提交/取消前的实际分派。
- **T03-S04**：[frappe/erpnext / erpnext/stock/doctype/stock_entry/stock_entry.py](https://github.com/frappe/erpnext/blob/v15.0.0/erpnext/stock/doctype/stock_entry/stock_entry.py) · ref `v15.0.0` · Blob `a2cae7ff8d9e068bca989a6fee1c22175c0cc40e` · L76–85, 195–241。库存业务提交/取消的实际操作序列。
- **T03-S05**：[saleor/saleor / saleor/checkout/models.py](https://github.com/saleor/saleor/blob/3.20.0/saleor/checkout/models.py) · ref `3.20.0` · Blob `e034d7a0643ff24bc857d92a8a90f794f6480983` · L35–89, 277–287。Checkout 与 CheckoutLine 声明。
- **T03-S06**：[saleor/saleor / saleor/order/models.py](https://github.com/saleor/saleor/blob/3.20.0/saleor/order/models.py) · ref `3.20.0` · Blob `76f1cb1a8e921e091c6ace0dbb10110cd3b0f28e` · L108–150, 523–533。Order 独立身份、订单/授权/扣款状态、行关系。
- **T03-S07**：[mattermost/mattermost / server/channels/app/channel.go](https://github.com/mattermost/mattermost/blob/v9.11.0/server/channels/app/channel.go) · ref `v9.11.0` · Blob `05aa3f4fa3fb3ed2ee2f65ac3692d735389d62c6` · L157–215。创建/重命名频道、用户查找、发布 WebSocket 消息。
- **T03-S08**：[nextcloud/server / lib/public/Files/Node.php](https://github.com/nextcloud/server/blob/v30.0.0/lib/public/Files/Node.php) · ref `v30.0.0` · Blob `e3641a82df3f52a78bf8b794ec20547d40cfd125` · L21–82, 209–217。公共文件节点/存储与错误合同。
- **T03-S09**：[WordPress/WordPress / wp-includes/class-wp-post-type.php](https://github.com/WordPress/WordPress/blob/6.7/wp-includes/class-wp-post-type.php) · ref `6.7` · Blob `5336d3e1cf8f32dcce1f1f5023ec9e31a3b0ba34` · L25–95, 698–707。内容类型元数据、可见意图与 rewrite 接线。

## 交接与缺口

本批满足至少四类真实业务架构视图的限定验收；直接源支持职责与数据/调用边界，部署与事务保证仍未知。Mattermost server/README.md 未取得，实际使用 channel.go，不计入缺失文件。SaaS 多租户隔离、CRM 完整系统、整体部署架构与真实支付/库存事务需 T07 或未来更深源读取，未假定已覆盖。T03 DELIVERED_SCOPED，独立资格待 T14；T12 可比较真实模型/生命周期与简单 CRUD、通知/存储、不同文件后端等场景。