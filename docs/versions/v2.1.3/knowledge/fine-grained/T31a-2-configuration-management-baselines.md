# T31a-2 · 软件配置管理：基线、变更控制、状态核算和合并审核

#598；K08 主责制品/SCM，K04 变更实施，K09 演进，K07 协作。复用 [K08 D1](../../../v2.1.2/software-engineering-knowledge/K08/foundations-deep-dive-01.md)、[制品来源与发布](../../../v2.1.2/software-engineering-knowledge/K08/advanced-release-provenance-permission-and-rollback.md) 与 [K09 合同演进](../../../v2.1.2/software-engineering-knowledge/K09/advanced-contract-evolution-migration-and-retirement.md)。本章是一般 SCM 机制与 Git 具体例，不定义任何项目强制流程；教学 NOT_RUN。

## 1. 配置项、基线与变更

软件配置管理不只是 Git 操作，也不是应用首次启动配置界面。配置项是需要在定义范围内识别与控制的材料，例如需求合同、源码、schema、依赖锁、生成配置、目标平台说明、制品及相关验证记录。基线是被指定用途接受并可重现定位的配置项集合；仅有分支名、tag 名或某次目录复制不是全部基线证据。它可能包含多个仓库/外部制品及其确定身份；monorepo 可让同一提交一致记录多个子模块，但外部依赖仍需记录。密钥不应直接作为版本库内容；记录受权引用、配置版本及必要约束，不暴露秘密。

| SCM 活动 | 实际问题 |
|---|---|
| 识别 | 哪些项、版本、依赖及 Owner 构成指定用途基线 |
| 变更控制 | 谁提出、为何改变、影响谁、接受何种风险、由谁决定 |
| 状态核算 | 计划、批准、已实施、已验证、已发布分别有何实际记录 |
| 配置审核 | 所需项是否齐、身份是否一致、批准范围与实际内容/结果是否匹配 |

教学：版本 v3 变更订单字段，受影响的不只有源码，还有 API 合同、迁移、旧消费者、测试及回退条件。先识别实际影响，选择最小受控变更；不必要求所有配置项统一更新。若源实现尚未改变，批准需求不等于实施完成；若源合入，发布记录仍独立。具体审批权、紧急变更及保留政策由项目有权方确定。

## 2. Git 对象与可变引用

Pro Git 第2版 10.2 将 blob、tree、commit 等解释为内容寻址对象：blob 描述内容，tree 记录路径名/模式与对象引用，commit 引用 tree 及父提交、作者/提交者等元数据。对象身份与文件名/分支不同；相同 blob 可以位于不同路径，不同 commit 可以指向相同 tree。分支/remote-tracking ref 会移动，本地 origin/main 仅是最近已取得的远端观察；tag 是引用，是否禁止移动由实际控制决定。不能把名字相同当内容相同；不能把同一 tree 当历史、签名或全部外部制品相同。

Git index 是拟提交快照；working tree 是当前可编辑文件。git status 分别报告 HEAD/index、index/working tree 及未跟踪项；脚本宜使用实际版本支持的 porcelain 格式和适当 NUL 分隔，不能解析给人看的颜色文本。干净状态不包含忽略项、所有外部依赖、环境变量或生成产物证明。子模块/gitlink 记录的是子仓库提交引用，不代表其工作目录已经检出正确内容或依赖可获取。采用多个仓库还是 monorepo 取决于真实交付边界；不要为目录组织重造版本身份。

## 3. 三方合并与语义冲突

教学历史：O 是共同祖先，A 分支改变库存拒绝条件，B 分支改变重试策略。若当前 HEAD 是目标提交祖先，可 fast-forward；否则常见双分支 true merge 以共同基线及双方差异生成新 tree/双父提交。多个最佳 merge base 可能存在，不能用时间最近提交代替祖先关系。Git 冲突时 index stage1/2/3 通常对应基线、当前、合入方的相关版本（某些删除路径没有全部三项）。解决冲突要依据业务合同、完整上下文及变更来源，不能无条件选 ours/theirs 或只删除冲突标记。

重要反例：A 在文件1将拒绝条件由金额>100改为>=100，B 在文件2遇拒绝时自动重试；文本无冲突，但边界100变为永久拒绝，B 仍重试，产生语义冲突。需要联合验证金额边界、错误分类与重试预算；合并器成功不证明兼容。rebase、cherry-pick、squash 可能产生新提交身份，即使部分文本差异相同，原审核/签名或测试适用性也需按真实影响核。revert 是通过新提交尝试逆转选定变更，不是删除历史；后续依赖、冲突及 merge revert 的父线选择均需核。源代码回退不能撤销实际业务数据或外部 Effect（K09）。

## 4. 状态核算与实际审核收据

以下教学账本不要求新系统；可由现有 Issue/PR/制品记录提供：

| 对象 | 记录 |
|---|---|
| Change C | 原因、受权范围、影响项、负责人、批准决定/未知 |
| Baseline B | 仓库与 commit/tree、实际依赖/配置引用、适用消费者 |
| Candidate H | 相对 B 的真实修改、未跟踪/生成输入影响、待核范围 |
| Verification | 实际 H、环境、选择/发现、观察及限制 |
| Integration M | 合入提交及父关系、审核 H 到 M 的实际差异、目标 ref 的回读 |
| Release | 实际制品摘要、支持版本/平台、发布权威和回退限制 |

若 H 审核后目标分支被另一个变更修改，不能仅凭 H 审核结论推导 M；需要检查 M 新增的变化、交叉影响及证据适用性。即使 H→M tree 零差异，也应核历史关系、受权目标及旧验证的环境条件；零差异不是无限制发布授权。远程操作超时可能已完成，应回读真实 ref、提交和操作收据，区分元数据暂时不一致与内容事实，不盲目重试合并。状态核算不是另一套并行事实源；链接到实际权威对象，保留 UNKNOWN 和纠正记录。

## 5. 独立验证与资料

| 教学反例（NOT_RUN） | Oracle |
|---|---|
| 分支同名移动、两个 commit 同 tree | 精确比较实际引用、对象身份与历史，分别报告 |
| index 只暂存一半、忽略文件污染生成 | 观察 index/working tree/实际生成输入，不用 clean 替代 |
| 两分支无文本冲突却改变拒绝/重试 | 联合业务边界和真实消费者回归 |
| 审核后 HEAD 变化、合并回执丢失 | 检查审核对象与目标 ref、实际合入身份/差异，未知不报完成 |
| 源 revert 后旧 schema 无法读新数据 | 保留数据事实、独立迁移回退判断，不报已恢复 |

2026-11-26 实读 [Pro Git 2nd ed. 10.2 Git Objects](https://git-scm.com/book/en/v2/Git-Internals-Git-Objects)，[git-merge](https://git-scm.com/docs/git-merge) FAST-FORWARD/TRUE MERGE 与冲突 stage，[git-status](https://git-scm.com/docs/git-status) OUTPUT/Porcelain，[git-merge-base](https://git-scm.com/docs/git-merge-base) DESCRIPTION，Git 在线文档（具体安装版本未实测）。SCM 四类活动、教学账本与例为专业综合，未核 ISO/IEEE SCM 标准全文，不宣称标准认证；未运行 Git 探针或新增 qualified Claim。后续单独深化变更审核、制品生命周期及历史 SCM 缺陷。
