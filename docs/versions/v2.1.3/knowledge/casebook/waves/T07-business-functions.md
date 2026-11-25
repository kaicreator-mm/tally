# T07 · 业务功能与领域抽象案例

[#567](https://github.com/kaicreator-mm/tally/issues/567)。9 个新案例、9 个新登记文件；涵盖身份/会话、授权、多租户、订单/支付/对账、库存、表单/规则、报表、通知/审计。每条是真实局部源观察加专业推论，验证建议 NOT_RUN；不是业务规则授权或合规结论。T03-S04 同一来源复用用于不同功能决策（在途状态与价值），不重复登记。

## T07-C01 · 身份与会话：登录是安全状态转换

**分类：** 身份/会话 / L3–L5。**知识：** K03.F3.1。**来源：** T07-S01。**关系：** 机制关联；登录会话身份转换。验证意图另见正文，NOT_RUN。

- **实际观察：** Django login 检查既有 session 用户/认证 hash，必要时 flush；无已有认证 key 时 cycle_key，再写用户/后端/hash、更新 request.user、轮换 token；logout 发信号后 flush 并设置 AnonymousUser。
- **业务状态、职责与合同（分析）：** 身份认证结果、会话身份、当前请求身份和授权分开。登录不是仅写 userId；迁移匿名数据必须符合当前会话合同。会话切换也不是业务数据迁移。
- **对照、反转条件及验证建议（NOT_RUN）：** 本地单用户工具不需远程会话；浏览器登录需会话生命周期；API token 可有不同刷新/撤销机制。测试更换用户、旧 session、退出后再次访问、匿名数据保留边界；不宣称整个 Django 会话/CSRF 全量安全。

## T07-C02 · 权限：实际主体、操作和资源范围

**分类：** 授权 / L3–L5。**知识：** K02.F4.3。**来源：** T07-S02。**关系：** 机制关联；主体动作资源授权边界。验证意图另见正文，NOT_RUN。

- **实际观察：** Django BaseBackend.has_perm 接受 user、perm 和 obj，以 get_all_permissions 的返回判断；同文件 ModelBackend 有认证入口。这里只观察接口，不证明具体后端支持对象级授权。
- **业务状态、职责与合同（分析）：** 认证验证身份；授权须根据实际主体、动作与资源/环境。共用权限机制不应让普通请求自报真实角色；不能将模型级权限自动扩大到所有资源。
- **对照、反转条件及验证建议（NOT_RUN）：** 固定单角色工具可直接条件；多角色复用 RBAC；对象/租户/状态规则须按实际授权增加。对照 DE-005 与 T04-C03，测试同角色不同资源、失效权限、批量入口；不承诺 ModelBackend 对象权限。

## T07-C03 · 租户隔离：请求绑定与数据访问边界

**分类：** 多租户 / L2–L4。**知识：** K02.F4.3。**来源：** T07-S03。**关系：** 机制关联；租户选择和数据隔离威胁。验证意图另见正文，NOT_RUN。

- **实际观察：** django-tenants middleware 先把连接置于 public schema、查 hostname 对应租户，再设置 request.tenant/connection tenant/路由；未知租户可能按明确设置走 public 或报错。
- **业务状态、职责与合同（分析）：** 租户选择、认证授权、资源范围和连接生命周期是不同责任；所有后台任务/非 HTTP 入口也需独立租户绑定。未知租户公共路由只能暴露获权公共资源。
- **对照、反转条件及验证建议（NOT_RUN）：** 共享表 tenant key+强制过滤、schema-per-tenant、database-per-tenant；按隔离、迁移、成本和故障域选择。对照 T04-C03，测试同名 ID 跨租户、复用连接、未知 host、后台任务；未审查 schema 切换后 SQL/连接释放完整链。

## T07-C04 · 支付 webhook：接收真实性与业务记账

**分类：** 支付/订单 / L3–L5。**知识：** K02.F4.3。**来源：** T07-S04。**关系：** 机制关联；签名验证作为外部事件接收的威胁边界；业务一致性、重复及乱序处理仅为条件启发，源中并无该协议。验证意图另见正文，NOT_RUN。

- **实际观察：** Stripe Python Webhook.construct_event 在解析 JSON/构造 Event 前调用 verify_header，接收 payload/signature/secret/tolerance。
- **业务状态、职责与合同（分析）：** 外部事件验证是接收边界；订单归属、事件重复、顺序、金额币种及支付状态转换由业务负责。签名通过不表示钱已结算，也不表示事件适合当前订单。
- **对照、反转条件及验证建议（NOT_RUN）：** 无外部支付直接本地有权动作；外部支付采用签名验证+受权状态查询/幂等处理。对照 T03-C05，检查非法签名、重复事件、不同订单、迟到通知、失去响应后的实际状态；未执行签名测试，不推断 SDK 解决业务去重。

## T07-C05 · 付款与对账：余额差额、账务与逆向动作

**分类：** 支付/对账 / L3–L4。**知识：** K01.F3.3。**来源：** T07-S05。**关系：** 机制关联；付款差额及账务约束。验证意图另见正文，NOT_RUN。

- **实际观察：** ERPNext PaymentEntry.on_submit 拒绝非零 difference_amount，调用 GL、outstanding、advance paid、payment schedule 和 set_status；取消分支调用 GL cancel 等步骤。
- **业务状态、职责与合同（分析）：** 收款事实、分配到应收款、账务分录、剩余余额和状态是不同业务事实。取消应按有权财务政策形成逆向记录，而非简单 delete。
- **对照、反转条件及验证建议（NOT_RUN）：** 单笔收款记录、支付与应收分配、独立账务/银行对账；根据实际财务制度选择。对照 T03-C03/05，测试差额、部分付款、取消/退款、外部账单不匹配；未确认跨步骤事务保证或完整对账算法。

## T07-C06 · 库存：实物变化、在途与财务价值

**分类：** 库存/供应链 / L3–L4。**知识：** K01.F3.2。**来源：** T03-S04。**关系：** 机制关联；在途和入库状态。验证意图另见正文，NOT_RUN。

- **实际观察：** 重新读取 ERPNext StockEntry L195–241，同 T03-S04 Blob：Material Transfer 根据 add_to_transit/outgoing_stock_entry 调整 In Transit/Completed，取消时可能恢复 Not Started，并调用库存账、工单、GL。
- **业务状态、职责与合同（分析）：** 重点是库存功能的状态/价值区分，而非重复整体协调架构：来源库扣减、在途、目标入库、成本记录可能各有不同时间与条件。库存数量不是订单状态的同义字段。
- **对照、反转条件及验证建议（NOT_RUN）：** 单仓即时增减、双仓调拨带在途、批次/序列/成本追溯；按真实业务选最小模型。对照 T03-C03 与 DE-004，检查收货未完成、重复确认、取消/关联后续操作、数量和账务读回；未审查库存计算。

## T07-C07 · 表单：绑定状态、校验流水与业务规则

**分类：** 表单/规则 / L4–L5。**知识：** K03.F2.2。**来源：** T07-S06。**关系：** 机制关联；字段结构及跨字段校验。验证意图另见正文，NOT_RUN。

- **实际观察：** Django full_clean 先初始化错误；未绑定或允许空且未变更时停止；否则执行字段、表单、post-clean。
- **业务状态、职责与合同（分析）：** UI/输入形状、单字段约束、跨字段关系及持久化/业务校验需要分层。未绑定表单不等于合法业务命令；清洗不能静默替用户选择新业务含义。
- **对照、反转条件及验证建议（NOT_RUN）：** 简单单字段直接验证；复杂表单分字段/跨字段；跨记录/资源状态由领域执行点校验。对照 T05-C05、AB-011，测试缺字段、空表单、关联字段冲突、权限与并发状态变化。

## T07-C08 · 报表：查询权限、预生成与显示模型

**分类：** 报表 / L3–L4。**知识：** K02.F2.1。**来源：** T07-S07。**关系：** 机制关联；报表定义执行与显示职责。验证意图另见正文，NOT_RUN。

- **实际观察：** Frappe query_report.run 获取报表定义、检查 ref_doctype 的 report 权限，按设置取 prepared result 或实时 generate；结果构造含列、表/图/摘要及总计控制。
- **业务状态、职责与合同（分析）：** 定义与参数、授权查询、执行策略（实时/预生成）、结果投影和展示分别管理。预生成报告必须有适用的时间、用户/范围与版本，不能自动当最新数据。
- **对照、反转条件及验证建议（NOT_RUN）：** 小规模实时查询；昂贵报表有界缓存/预生成；单独分析平台仅在真实规模要求下采用。测试无权报表、缓存用户范围、过滤变化、旧快照、汇总与明细一致；仅观察入口，不证明所有行权限。

## T07-C09 · 通知和审计：发送与事实记录的差异

**分类：** 通知/审计 / L3–L5。**知识：** K02.F3.2。**来源：** T07-S08、T07-S09。**关系：** 条件启发；日志字段和邮件发送接口分别存在，若业务要求事件与状态一致性须另行设计；两源不证明共同状态、幂等或可靠投递协议。验证意图另见正文，NOT_RUN。

- **实际观察：** Django admin LogEntry 记录时间、用户、内容类型、对象 ID/表示、动作类型及变更说明；send_mail 是带收件人、认证/连接和 fail_silently 参数的发送包装器。
- **业务状态、职责与合同（分析）：** 审计是行为事实的记录模型；通知是向接收者产生外部效果。共享消息格式不意味着共享留存、权限、交付和错误语义。普通管理日志不自动具备防篡改/合规审计资格。
- **对照、反转条件及验证建议（NOT_RUN）：** 本地调试日志、业务审计、同步通知或可靠投递；按真实要求独立选择。对照 DE-008，检查邮件失败、重复发送、审计丢失、敏感字段和收件者可见性；不宣称可靠/不可变保证。

## 实际来源登记

- **T07-S01**：[django/django/django/contrib/auth/__init__.py](https://github.com/django/django/blob/5.2/django/contrib/auth/__init__.py) · ref `5.2` · Blob `8e359ec7ffe4cfa600c5217e300498d11e130329` · L153–169,175–197,243–258。登录 session 身份/哈希、轮换、登出 flush。
- **T07-S02**：[django/django/django/contrib/auth/backends.py](https://github.com/django/django/blob/5.2/django/contrib/auth/backends.py) · ref `5.2` · Blob `83525306655b6eec2b8ded319be41836ead60a28` · L47–62。权限接口与 ModelBackend 认证入口。
- **T07-S03**：[django-tenants/django-tenants/django_tenants/middleware/main.py](https://github.com/django-tenants/django-tenants/blob/v3.7.0/django_tenants/middleware/main.py) · ref `v3.7.0` · Blob `d958c50082670e10e347c69d8174105ed91e5528` · L31–60。从公共 schema 查租户，再切换连接/路由及未知租户分支。
- **T07-S04**：[stripe/stripe-python/stripe/_webhook.py](https://github.com/stripe/stripe-python/blob/v11.3.0/stripe/_webhook.py) · ref `v11.3.0` · Blob `6eaff0a2a9597b86c8fc810edc58a1ade5ce5c9f` · L19–35。先验证 webhook header，再反序列化 Event。
- **T07-S05**：[frappe/erpnext/erpnext/accounts/doctype/payment_entry/payment_entry.py](https://github.com/frappe/erpnext/blob/v15.0.0/erpnext/accounts/doctype/payment_entry/payment_entry.py) · ref `v15.0.0` · Blob `e6403fddefe3885c3c3217c648b8810e7a713db2` · L97–104,142–157。付款差额前置条件及提交/取消的账务操作。
- **T07-S06**：[django/django/django/forms/forms.py](https://github.com/django/django/blob/5.2/django/forms/forms.py) · ref `5.2` · Blob `d05bf4bb9e442b7723a77d81c22c46b349801976` · L324–339。表单绑定状态、字段/表单/后处理校验。
- **T07-S07**：[frappe/frappe/frappe/desk/query_report.py](https://github.com/frappe/frappe/blob/v15.0.0/frappe/desk/query_report.py) · ref `v15.0.0` · Blob `7ca483d806b19c1b69e7eb6b2c198f22a74f6d7b` · L72–88,187–229。报表权限、预生成/实时分支及结果列。
- **T07-S08**：[django/django/django/contrib/admin/models.py](https://github.com/django/django/blob/5.2/django/contrib/admin/models.py) · ref `5.2` · Blob `957c9e4877dc1defe3ed69821cd986df6c2dbbad` · L106–140。管理操作日志实体。
- **T07-S09**：[django/django/django/core/mail/__init__.py](https://github.com/django/django/blob/5.2/django/core/mail/__init__.py) · ref `5.2` · Blob `b179736b15ed1e367ff851cd092c28c1bf6bf4bd` · L59–75。邮件包装入口、收件者与连接参数。

跨分区 [T03-S04](T03-business-and-collaboration-architecture.md)。django-tenants 无 v 前缀 ref 404，实际使用 v3.7.0。T07 DELIVERED_SCOPED，未做支付/库存/租户端到端、合规及权限审计；全量业务政策须由具体产品授权，不能从现有软件移植成通用约束。