# T44 · HC-04 有限平台知识收敛与 T43 深度反馈

#660/#563，冻结 b9b162da52e683f40bebfb58ddda2288cd5a365f。实际读 T24c 任务/心理模型、输入/焦点和请求、T25b-1 字节/字符/规范化、T08-C04/C05 实际状态通知和资源所有权、T35-3 G4 请求/缓存/授权；复用已有 K01/K02/K03 等 Owner，不重复写核心 HCI 定义。

新增 [R1 Web事件/布局/几何](fine-grained/T44-R1-web-events-layout-geometry.md) 和 [R2 应用生命周期/状态恢复](fine-grained/T44-R2-platform-lifecycle-state-restoration.md)：前者补实际 DOM 传播与取消、动态列表身份、监听释放、Flex 分配和约束冻结、布局几何与命中反例；后者补 Android 配置变化/系统进程回收/用户退出的不同恢复、轻量与持久状态、业务受理/回执丢失与资源代际。真实官方资料有限实读（DOM Living Standard、CSS Flexbox CRD、CSSOM View WD、Android 文档），非浏览器/Android/产品实测。HC-04 主行 SPECIALIST U 保持，核心已限定 S；未选各移动/桌面框架、渲染内部/图形算法及完整 BiDi/collation 保持 UNKNOWN，未来由真实问题触发，不为专项全量覆盖增加无限章节。

T43-RV-01 非阻断问题来自 [#659审核](https://github.com/kaicreator-mm/tally/pull/659#issuecomment-7197569726)，[合入收据](https://github.com/kaicreator-mm/tally/issues/658#issuecomment-7197851084)。T43-R1 §3 仅一段加法说明 t 原始 pop 与 k 去重输出成本区别及 4流3身份反例，原文本保持，待本轮独立关闭。T42-RV-01/02 已接受合入关闭，不复审。

按有限收敛建议下一步 T45 多约束动态工作流及算法，T46 真实完整产品知识案例、T47 关系/组合收敛、T48 最终盘点；不是冻结工作量，按已存在知识收缩。DA-08 overall U，#653/#655 仍缺原独立回执；本批不改共享台账/索引。58正式接受正文、50/284正式发现、67/86、S51/D0/U5历史不变，新增2篇待审核，不新增合格 Case/Claim/Rule。#565 HOLD、#286 DEFERRED、A24/T4 NOT_RUN/HOLD；KNOWLEDGE_CONSTRUCTION，无产品/Package/DH/SDK/CI/Agent/LLM或实际目标执行。
