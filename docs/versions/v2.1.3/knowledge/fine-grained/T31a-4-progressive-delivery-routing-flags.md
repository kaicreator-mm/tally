# T31a-4 · 渐进部署、版本路由、Feature Flag 与恢复

#622 / R4-D2 / OP-04；K08 发布与制品，K09 兼容迁移，K10 运行，K05 独立验证。复用 [REL](../../../v2.1.2/software-engineering-knowledge/K08/advanced-release-provenance-permission-and-rollback.md)、[T32a-1](T32a-1-maintenance-refactoring-replacement.md)、[T32b-1](T32b-1-data-migration-and-mixed-consumers.md)、[T33c-1](T33c-1-operations-incident-service-support.md)。原代码回滚不等于数据/Effect 回滚已足够；新增真实渐进部署与 flag 合同，不规定所有应用需容器或在线部署。

## 1. 四类独立变化

制品部署（实际 digest/实例）、流量分配（实际请求命中版本）、功能选择（flag/context/策略版本）、数据演进（权威 schema/状态与消费者）分别有 Owner 和证据。一个容器更新不证明流量已切换；flag=true 不证明新制品已运行，也不授予业务权限。离线 CLI/桌面可按分发与安装版本管理，不必引入网络路由。

Kubernetes v1.34 Deployment 的 RollingUpdate 调整新旧 ReplicaSet，maxUnavailable 百分比向下取整、maxSurge 向上取整；二者不能同时0。教学 replicas=5，两者25%，控制器计划允许不可用1、额外2，名义目标至少4可用、最多7个非终止期目标副本；不能当任意故障下保证至少4可用，终止中的 Pod 仍可占资源，实际负载/配额另核。progressDeadlineSeconds 超过后报告 ProgressDeadlineExceeded，控制器不会因此自动回滚，仍重试；暂停不计算该进度期限。版本回滚控制的是 Pod template，不是权威数据恢复。官方 canary 建议多个 Deployment；真实用户流量比例另由 Service/路由/客户端及实际观测决定，不以 Pod 数量等同请求比例。

## 2. 教学部署合同

两个租户共用权威 DB，旧 v1、新 v2 都必须读取同一合法商品状态，v2 引入新可选排序视图（不改变权限及已确认写入合同）。实际镜像身份 D1/D2 需验证；初始 v1 100%，v2 无流量。先扩展兼容 DB 与旧/新真实消费者验证，部署 v2 小组，独立检查 readiness、健康、权限负例及实际路由。假设获权 canary 目标5% 的 eligible 请求，固定 cohort 标识及路由版本；不可变用户/租户身份和当前授权不由客户端 flag 值决定。会话、长任务及重试按实际合同决定是否固定版本；恢复时不得让旧 worker 重放已失效操作。若 v2 引入旧版不能理解的必需权限字段，须先拒绝/隔离 v1 路径；不让 canary 成为权限例外（参 [T25c-3](T25c-3-protocol-capability-negotiation.md)）。

OpenFeature 在线 Flag Evaluation API 区分 typed value/default/context 与 detailed evaluation 的 variant/reason/error；异常求值按默认值返回，详情可能只具 best-effort 信息。这里选择排序 flag 默认 false（保持有权旧功能）；如果默认旧路径会违反必需新政策，则不能靠 false 安全回退，应拒绝/隔离。实际 provider readiness、规则更新、缓存/失联与回退按 SDK/平台验证，不把 flag 服务当权限数据库。

## 3. 观察与停止条件

假定同一窗口 v1 1900次合法请求，失败19（1%）；v2 100次，失败8（8%），合计27/2000=1.35%。若全局阈值2% 单看总量会通过；v2 cohort 8% 超过本例获权上限3%，应暂停扩大并调查，不能将本小样本解释为因果或长期SLA。负载/租户构成不同须控制；无流量或监控失联=UNKNOWN，不是0失败。任何实际越权、数据不一致、关键回归即停止推广，不由总体比率抵销。指标、阈值、最小窗口/样本、观测责任、可回退条件须预先获权；以上数字仅教学。

| 具体失败/恢复（NOT_RUN） | 独立 Expected |
|---|---|
| 实际部署 D2 但路由仍 D1 | 分别核 digest、请求命中/比例、实际功能；不得标 canary 已验证 |
| rollout stalled | 观察条件、quota/实例与用户结果；按有权规则停止/回退，非自动 rollback 假设 |
| Flag provider ERROR 或缓存陈旧 | 实际求值 reason/error/default 及功能状态；默认值仍不得绕过授权 |
| v2 数据变更使 v1 无法解释 | 不放行 v1 回退；若合法兼容恢复/迁移需独立验证；已发生外部 Effect 不随版本撤销 |
| 旧实例停止时仍持有任务 | 分开排空/撤销许可/实际终止、待办恢复与完成；核当前 fence 而非 Pod 删除请求 |
| 回退路由或制品后 | 核真实 v1 请求、合法数据、投影及持久待办、无新错效果、恢复目标；不以进程重启通过 |

如没有旧/新并存需求，停机升级或原位替换可能更简单；选择 rolling、蓝绿、canary、flag 的不同变化面而非强制组合。实际部署、路由、状态和用户路径证据应关联版本、时间、授权及真实观测；未实测不声称发布成功。

实际资料 2026-11-26：[Kubernetes v1.34 Deployment](https://v1-34.docs.kubernetes.io/docs/concepts/workloads/controllers/deployment/) L1602–1615、1708–1710、1756–1767、1850–1851；[OpenFeature Flag Evaluation API](https://openfeature.dev/specification/sections/flag-evaluation/) 在线规范 §1.3/1.4 L199–280、285–366、§1.7 L452–488，未冻结具体 SDK 或整个规范版本。未验证部署/控制器源码/路由实现，数值和方案原创教学 NOT_RUN。OP-04 限定目标候选。
