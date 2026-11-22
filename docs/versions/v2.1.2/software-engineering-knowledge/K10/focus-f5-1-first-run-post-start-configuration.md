# K10.F5.1｜首次启动、运行后配置与缺参安全边界

> [K10 课程](systematic-syllabus.md) · [第二卷：运营与容量](foundations-deep-dive-02.md) · [K05 真实平台验证](../K05/focus-f3-3-build-package-platform-verification.md)
>
> **R4 ORIGINAL_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / NO_PRODUCT_BOOT_FLOW_IMPLEMENTED。** 仅深化 K10.F5.1 的软件可运维性与用户可用性专业知识。任何应用启动协议仍以其有权 Product 要求为准，不把桌面首配需求错误外推到 headless 服务/CLI。

## 1. 可启动不代表已获业务操作资格

桌面应用可以设计为：进程启动 → 本地受控 Shell/UI 就绪 → 提醒用户选择项目/连接信息 → 验证与保存配置 → 进入可工作状态。**未配置**应是一个可解释的正常状态，而不是应用崩溃；但它也不得允许操作意外写入默认仓库或远端生产资源。

建议教学状态：
`UNCONFIGURED -> CONFIGURING -> VALIDATING -> READY`，另有 `INVALID/UNAUTHORIZED/RECONNECT_REQUIRED`，实际允许路径必须由产品合同确定。这是**解释模型**，不是对 DomainHarness Package 增加新的 Kernel 状态机或公开 ABI。

合法“未配置”时可显示项目选择和 Help，禁止需要权限的 Effect；配置完成前不应把只读帮助页误判为“生产系统 READY”。用户主动撤销授权后，已有本地缓存是否还能显示、正在运行的 Job 如何处理、敏感 Token 如何清除，都要按照实际产品约束回答。

## 2. 不同应用形态应该采用不同的配置策略

| 运行形态 | 可行的首次配置方式 | 关键错误边界 |
| --- | --- | --- |
| 桌面 GUI | 启动后向导、项目管理、授权连接 | 禁止在选择目标之前默默发起业务写入 |
| 本地 CLI | 首次子命令提示、交互式向导、明确命令参数 | 非交互脚本不得永久等待输入 |
| Headless Service | 环境/Secret Manager/启动 API/管理端等有权配置手段 | 无必需凭据时必须 fail closed 且给出可行动错误 |
| 临时开发工具 | 交互输入或受控默认本地路径 | 不能将开发示例配置误用生产 |
| 多项目桌面工具 | 保存多个受权项目 Profile，切换时隔离状态和权限 | 不能把 `REPOSITORY` 进程变量当成永久唯一项目身份 |

Twelve-Factor App 倡导将服务配置与代码分离，常以环境变量承载；适用对象主要是服务交付实践，**不是一条命令所有桌面程序必须启动前设置 Environment**。用户明确要求应用启动后配置时，需要实现真实 UI/操作路径，而不是只改 README 文案。与此同时，非交互服务可能更适合外部配置，不要为“统一”强制构建 GUI。

## 3. 配置数据、运行状态和秘密必须分层

- **Profile**：用户可命名的逻辑项目/工作区标识、非敏感偏好、目标仓库；支持列出、选择、切换、删除/更新。
- **Secret**：凭据/Token、刷新身份与权限；应由相应 OS 安全存储或获准 Secret Provider 保管；配置日志/导出应遮蔽秘密。
- **Runtime State**：当前连接健康、Session/Attempt、进度、未决 Effect；不能盲目认为进程重启后仍可复用有效授权。
- **Product/Release Identity**：应用版本、数据 Schema 版本和升级迁移状态，不应混入某一个用户 Profile 的配置值。

XDG Base Directory Specification 规定了类 Unix 桌面环境配置/数据目录的标准约定，Windows/macOS 有各自原生约定；“配置文件可持久化”与“启动必须依赖手工编辑静态配置文件”是两回事。不要把未加密的明文 Token 当作正常的 Profile 字段。

## 4. 健康检查与失败可行动信息

应区分进程/GUI 已启动、配置可编辑、连接可用、实际工作流可成功四个指标。一个 API Server 返回 200 的 `/health` 只能说明其定义范围内的健康；依赖外部仓库授权的业务操作可能仍被拒绝。错误消息至少让使用者知道：当前模块/阶段、可操作原因、相关配置项或恢复动作、可供调试的请求/日志 ID，同时不泄露 Token 和敏感载荷。

**跨平台反例**：Windows 桌面应用启动成功，Linux 终端在未配置条件下因不可见的 GUI 初始化而挂住。若宣称“全平台支持”，至少应定义和测量对无图形环境的预期（报错退出/使用 CLI/显式不支持），不能把一个平台的正向 Demo 外推。

## 5. 故障、恢复与真实验收

测试教学序列：
1. 全新用户目录、无 Profile/Token 时能否进入合法配置界面/路径，不发生未授权 Effect？
2. 配错 URL/授权失效时是否有可行动错误、可恢复输入和安全日志？
3. 添加项目 A 和 B 后，项目切换是否隔离当前 Workspace、SourceSHA 和凭据？
4. 重启后配置是否按用户产品承诺恢复、是否验证版本/迁移？
5. Headless 未配置时是否明确非零退出或管理端就绪、且不会无限等待输入？

K05 负责给这些目标独立测试 Oracle，K08 负责真实 Host/安装收据；本章仅写出需求与技术机制，不声明上述测试已经执行。

**停止条件**：无法确定受权项目或 Secret、已发生未知远端副作用、迁移遇到非法旧数据等情况，受影响业务路径保持 `NOT_READY/UNKNOWN/STOP`；UI/帮助/重新配置等安全路径可继续。不能为了“正常启动”偷偷默认成固定 GitHub 项目。

**公开来源线索**：[Twelve-Factor Config](https://www.12factor.net/config)、[XDG Base Directory Specification](https://specifications.freedesktop.org/basedir/)、[Windows App Packaging/Deployment](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/packaging/)。这些并非现行 Tally UI 真实实现证明。
