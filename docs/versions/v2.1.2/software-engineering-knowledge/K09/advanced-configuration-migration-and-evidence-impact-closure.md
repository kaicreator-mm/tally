# K09 专业深化：配置/数据语义迁移与回归证据的作用域传播

> [K09 系统课程](systematic-syllabus.md) · [R6 Contract 演进与退役](advanced-contract-evolution-migration-and-retirement.md) · [K08 Release 来源](../K08/advanced-release-provenance-permission-and-rollback.md)
>
> **R9 DOCUMENT_CANDIDATE / NO_REAL_USER_DATA_MIGRATED / NO_ACTUAL_ROLLBACK**。精确深化原课程 **K09.F3.3 配置和数据语义迁移**、**K09.F4.3 回归传播和证据复用** 两个 ID。相应客户端与运行机制需由实际产品版本合同决定，不能在知识库里擅自定义 Tally 的持久化目录、密钥布局或 Consumer ABI。

## 一、配置迁移首先是用户语义的连续性问题，不是复制文件

交互式 GUI 应用允许首次启动后配置（例如连接多个 GitHub 项目），其合法设计与需要 `ENV_FILE` 作为启动前唯一依赖的 CLI 后台服务不同。**应用能启动**、**用户已完成授权配置**、**可以执行受保护业务操作**三者是不同状态：
```text
BOOTABLE -> UNCONFIGURED -> CONFIGURED -> AUTHORIZED -> ACTIVE
```
这是教学状态序列而非现有 Tally 真正实现的 Schema。用户应能进入合理的首次配置界面；在没有权威配置/权限前，生产外部 Effect 必须受控拒绝；没有必要强制每种 CLI/容器后台服务提供 UI。

迁移内容可能包含：项目/仓库选择、多个受管项目的版本引用、Host Provider 选择、凭据关联、用户偏好、工作区地址与保存的 SourceSHA。将这些数据直接复制到新配置格式时，必须核查各字段**身份、所有者、加密/访问保护、失效期限与受权用途**。把一个旧项目固定 `TALLY_V2_REPOSITORY` 环境变量映射成“所有项目共用该 Repository”就可能导致多项目配置串扰。

## 二、有损迁移的反例必须先于上线承诺

假设旧数据：
```json
{"order_id":"O-17","full_name":"Prince","currency":"MMK",
 "state":"CANCELED","external_refund":"UNKNOWN"}
```
新 Schema 要求 `first_name` + `last_name`，并希望把 CANCELED 变成 REFUNDED。两个变换都不保真：单名 `Prince` 没有可从原记录确定的两字段划分；取消订单不意味着外部退款已提交。若程序为了“保证所有字段非空”给 last_name 填 `UNKNOWN`，只能把它当标识了数据缺口的显式占位，不能表示真实姓氏；若把 external_refund 直接改成功，则更可能违反财务事实。

**迁移前的最小验证对象**：数据版本、Owner、样本分布、缺失/异常值、不可逆字段、重放/幂等、恢复材料、是否有真实旧 Consumer 并存。若允许 Expand–Migrate–Contract，则按有权重叠期维持 Reader/Writer 兼容；若当前版本明确完全重写且无需旧版兼容，则不能为了“最佳实践”自动加旧 Runtime Shim，实际需要的**数据迁移**仍须分开审视。

## 三、配置与密钥的迁移需要特别防止危险“修复”

安全迁移原则包括：
- 凭据若由系统/Provider 管理，不应因为新配置 JSON 缺字段就默认复制旧 secrets 到所有项目；
- 迁移失败时输出可行动的非秘密错误和合法恢复入口，不把 Token 写入日志；
- 多项目身份要加入版本与仓库作用域，不允许只因同名 display title 而合并；
- 用户尚未配置/确认 Source 时，应允许打开 UI 配置中心，但拒绝没有授权的远端写操作；
- 本地可逆配置和远端已提交 Effect 分开：重置本地设置不能撤销真正的 GitHub Issue/财务操作。

这类设计要求可以作为专业质量参照，但具体配置存储位置/Windows 凭据管理/Node 最低版本都必须由受权产品合同与真实 Host 测试决定，不借本章制造新平台约束。

## 四、证据复用取决于输入与行为影响，不取决于文件是否改名

考虑 H1 通过的旧 Contract Test `T1`，H2 的 Diff 只改生成器配置 `schemaVersion=2` 但生成代码文件未在 Git Diff 中显示变化。可能已有构建缓存将旧代码返回给 H2。若 `T1` 消费新 Schema，则源与受检 Artifact 的实际构建输入闭包已变，旧证明不能自动继承。

可采用一个**受限影响图**：
```text
Changed source/Rule/Schema/Host/Provider version
  -> material generated/input artifacts
  -> specific owning component(s)
  -> actual Consumer contracts and negative Oracles
  -> selected build/package/target Host evidence
```

传播并不意味着沿图全量重跑一切节点。只需要对真正**受影响且不能证明等价**的边新增验证；对于已证明材料未变化的独立模块，旧 ExactSHA 证据在其有权范围内可以复用。若不知道动态调用、外部条件或生成代码是否参与输入闭包，应标 `IMPACT_UNKNOWN`，补最小辨别探针，而不是随意宣布“全部都需要跑/全部都可以跳过”。

## 五、两种典型证据错误

**假阳性复用**：新版本改了测试发现 glob、却看到 40/40 单测 PASS，于是复用旧 H1 结果宣称所有新 `*.spec.mjs` 合同测试已通过；目标测试甚至可能没有被选中，状态应是 `NOT_RUN`。

**假阴性全量作废**：H2 仅调整独立帮助文档的标点，没有改变工具链/生成器/业务输入，团队却强制下载和验证所有 Windows/Linux 安装包。这可能违反有权流程的经济性意图；无真实 Release/Provider 强制要求时，应优先以差异及受影响证据说明为何不重复 CI。

H1 的 Review 不能自动签发 H2 的新权限，即使 PR 编号相同；但 H1 对未变化专业正文的旧限定接受可以继续使用，不需要每次对所有 11 模块知识重新独审。

## 六、学科综合题与退出条件

给定交互式应用首次启动在 Windows 成功，但旧单项目配置迁移到多项目后两个 Tenant 共享凭据，同时 H2 的 CI 显示绿色。专业分析先核实权限与密钥作用域、数据变换可逆性和新 Context 是否真实触发 Contract Test。不要由“启动成功/测试部分 PASS”宣称有权多项目隔离及迁移完整。

**STOP**：真实数据样本不完整、敏感凭据不能合法读取、外部 Effect 状态未知或无受权 Consumer 合同的时候，不能声称迁移完全安全。已有权限条件下的只读结构研究/Mock 可继续。

**专业参考**：[A04 ISO 12207 生命周期](../external-source-catalog.md)、[A35/A64 Parallel Change](../external-source-catalog.md)、[A53 Reproducible Builds](../external-source-catalog.md)、[A57 Node Runner](../external-source-catalog.md)、[A60 Twelve-Factor Config](../external-source-catalog.md)、[A61 XDG 基础路径](../external-source-catalog.md)。研究方法不等于真实迁移或发布完成。