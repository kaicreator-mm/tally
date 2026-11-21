# K05.F3.3｜构建、Package 与目标平台的验证分层

> [K05 课程](systematic-syllabus.md) · [第二卷](foundations-deep-dive-02.md) · [K08 交付视图](../K08/foundations-deep-dive-02.md)
>
> **R4 ORIGINAL_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / NO_TARGET_PLATFORM_VALIDATION_EXECUTED。** 专业 Owner 为 K05（**检验目的与 Oracle**）；K08 拥有真实构建执行、产物/安装包来源。深化 K05.F3.3，不能将文档视为 Windows/Linux/macOS 任意真实安装验证。

## 1. Build、Package、Install、Launch 是四种不同断言

`compile` 证明特定工具链能产出中间/目标文件；`package` 证明所选平台/分发形式的打包步骤产出文件；`install` 证明目标宿主能完成依赖/权限/落盘与入口配置；`launch` 证明实际入口能够启动并完成必要最小使用路径。四者互相关联却不可相互替代。典型错误：JavaScript 单元测试全部通过，随后文档宣传 Windows 安装程序已生成且可运行。

| 验收问题 | 最小的可观察 Oracle | 不能宣称 |
| --- | --- | --- |
| Typecheck/Build | 当前 SHA + 实际目标工具链的类型/编译退出状态 | 业务功能完整 |
| Package | 目标 OS/arch 制品清单、格式与来源身份 | 已安装 |
| Install | 干净受支持 Host 的安装/初始化/卸载路径 | 任意 OS 的普适兼容 |
| First Launch | 实际入口运行、配置必要行为、安全缺参处理 | Mock 引导等于用户可用 |
| Core User Journey | 真实消费者走到被要求的可观察结果 | 导出函数被调用就等于界面正确 |

**为什么 Node 版本要求不能一刀切**：需要的是“产品选定的平台/Node 版本上真实使用的功能有对应保证”。若最低为 Node 22，应用并未使用 Node 24 独有特性，就不能仅因 Writer 的开发机使用 24 宣称项目必须 24；反过来，也不能用当前 Node 26 的测试发现策略证明老版 Node 22 的行为。编译目标、运行目标、测试发现策略必须明确版本。

## 2. 目标矩阵按产品支持合同而非理论全集

测试矩阵由 OS/arch、运行时与依赖 ABI、发行/安装方式、运行配置、环境约束和关键用户路径构成。不要因为理论上有 `Linux x64/arm64`、`Windows x64`、`macOS arm64` 就要求所有项目全矩阵；由**实际声明支持范围**和必要风险决定最小覆盖。缺乏真实 Host 而必须证明原生安装时，保持 NOT_RUN，不能用 Docker 中的 JS Mock 当实体系统收据。

例如跨平台 Node CLI 可以在跨 OS 上复用相同业务纯函数测试，但 Windows PowerShell 路径、Linux 可执行位/shebang、文件大小写、系统调用和本地二进制打包必须按材料性差异处理。

## 3. 测试发现与目标实际执行是独立事实

在 Node v22.16.0 的 `node:test` 运行器中，可通过适用版本的运行参数/文件发现规则选择 suite；exit code=0 不等于目标文件一定被发现。最小必要读回：目标测试文件、当前 SHA、有效命令/版本、发现数量、成功/失败/跳过和退出原因。

**负例**：CI 只匹配 `*.test.js`，新增材料测试 `contract.spec.mjs` 没有被发现，其他 100 个旧测试全绿；这只能报告“已运行的 100 个旧测试 PASS，必要合同测试 NOT_RUN”，不能以矩阵绿色替代。

## 4. 判断真实集成与平台测试的边界

测试层不等于环境层。单元测试可在真实平台执行，集成测试也可能只在 Mock 环境执行；“E2E”必须说明消费者调用范围、所用 Provider、真实外部 Effect 是否参与，以及哪些对象被 Mock。对第一启动路径进行模拟不代表系统服务、权限弹窗、持久化配置或最终用户可用性已证实。

**教学练习**：给定两个包（同一代码 SHA 的 Windows zip 和 Linux tar），分别记录制品 hash、目标依赖与首次配置行为。一端只完成编译，另一端确实启动。正确报告是两个平台**不同的限定结果**，不能合并成 `CROSS_PLATFORM_PASS`。

## 5. 失效边界与来源

不可获得目标 Host、无法确定目标版 Node Runner 的测试发现规则、Package 依赖清单不全、安装/配置在未授权阶段触发 Effect 时，输出相应 `NOT_RUN/UNKNOWN/STOP_FOR_RELEVANT_CLAIM`；不对无关文本变更重复跑全 OS 组合。

**公开来源线索**：[Node.js v22.16.0 Test Runner](https://nodejs.org/download/release/v22.16.0/docs/api/test.html)、[Microsoft Windows package/deploy](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/packaging/)、[Jest snapshot testing](https://jestjs.io/docs/snapshot-testing)。版本化官方文档支撑教学原理，不等于已经运行测试。
