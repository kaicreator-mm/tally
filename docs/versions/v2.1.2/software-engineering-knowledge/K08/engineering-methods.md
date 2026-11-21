# K08 · Engineering Methods / 工程方法与决策程序

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #529](https://github.com/kaicreator-mm/tally/issues/529) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## 既有核心方法

## 3. 知识正文：分析方法与工程权衡

### 3.1 区分安装、构建、测试和发布

依赖安装检验 package resolution，TypeScript/编译器检验语法/类型，测试检验行为，构建/packaging 检验产物，真实 Host/用户路径检验运行；各自回答不同问题。一个 pipeline 可能安装全部成功却在 tsc 阶段失败，说明该 CI 没有完成 Build。不要省略真实失败点，只报告总 exit code。

### 3.2 工具链与版本适配

Node/Python/Go 等版本依据项目当前 manifest、构建特性与实际已验证平台确定，不能以 Agent 主机的工具链当作产品硬要求。按现有锁文件、包管理器和目标平台复现依赖；若没有用到某个新 Runtime 特性，不能无故把版本下限抬高。可缓存第三方依赖，但必须区分缓存正确性与可复现证据。

### 3.3 CI 最小充分设计

将工作流拆成必要的语法/类型、受影响测试、真实接口/组合、制品发布等层，只有真实差异/风险要求才额外构建复杂矩阵。一次 clone/checkout exact SHA 在同一 pipeline 中尽量复用，在真正需要隔离的环境才单独获取源码。稀疏检出/部分 clone 要遵守相关文件和构建脚本的依赖闭包，不能因文档无关就误删测试必须的数据。

### 3.4 集成合同与真实入口

从已确认的模块 API/Schema/状态 Owner 检查新旧版本与真实消费者是否一致。合并多个 PR 时不应假定各自已通过单元测试即集成成功；当前集成 HEAD 的真实 UI/API/Service/DB/CLI/安装包所使用的入口必须调用新实现。对改变共享接口的任务，选择必要的合同回归和最小真实集成路径。

### 3.5 制品、安装与配置的实际行为

可交付目标可能是 desktop binary/installer、容器镜像、Web bundle 或 CLI 分发包，必须明确目标平台、版本及完整起步路径。适合交互的应用应允许启动后在应用中配置，CLI/后台服务/容器可能使用环境或启动参数；不能对所有产品机械要求同样初始化 UI。未获得真实安装/运行环境时保留对应资格 NOT_RUN。

### 3.6 变更/合并/发布的权威分离

代码候选有 Git SHA；执行 Build/Test 有对应 Tested SHA；CI Provider 对其真实命令负责；独立 Review 审核准确 Diff；Release Owner 根据适用证据选择发布。某项 PASS 不可自动提升其他层资格。发布后若发现失败，事件与修复、回滚和新 Candidate 都应可追溯，不能修改历史 PASS 的语义。

### 3.7 交付效率与失败归因

无关格式/文档变更不应强制重跑昂贵 Host/E2E，但真正受影响的行为和发布平台仍要验。失败需要记录首个实际异常步骤、命令、退出码、具体文件/行号及可操作诊断；区分 Provider 网络拉取、安装依赖、代码编译和用户安装错误，避免每次失败都完整重新 clone 并重做旧验证。

## 方法论进一步细化

开始前按项目实际 manifest/lock/scripts 和目标平台选择工具链，不用开发机新版本强制修改产品要求。改动代码应在提交/CI 前经过相应语法/类型、受影响测试；真正要求 Provider/OS/安装证据时执行对应环境验证。在同一 pipeline 内能安全共享 exact-SHA checkout 时减少重复 clone，但必须尊重 Woodpecker workflow 容器隔离语义。单个模块、真实集成入口与打包产物各有独立收据，不能由 install SUCCESS 推导 build 成功。

## 操作决策表

| 任务/环境条件 | 选择或调查路径 | 应交付证据 | 需要排除的错误 |
| --- | --- | --- | --- |
| npm install 成功而 tsc 失败 | 定位 tsc 首个失败类型和当前源码 | BUILD_FAIL 与根因信息 | 不能继续汇报全 CI SUCCESS |
| 多个 workflow 重复 clone 同一 commit | 按真实 Provider 工作区/缓存机制判断能否复用 | 网络/墙钟证据与最小优化 | 不假设所有容器共享工作区 |
| Unit PASS 但运行入口未加载修改模块 | K05 指定 public-path 检查，K08 安排真实集成 | 适用 Integration/Host 收据 | 不能发布实际功能成功 |
| 发布制品没有存在/摘要/安装验证 | 将 Release Candidate 与可下载正式包分开 | artifact UNKNOWN/NOT_FOUND | 不虚构 Release 链接 |

## 原始 Decision Procedures：输入、步骤、输出、停止

## 4. Decision Procedures（Agent 可执行的决策方法）

### DP-K08-01：最小有效构建/CI 路线

**输入：** 项目真实 package/build 配置、合法源码 SHA、目标环境和改动类型

**步骤：**
1. 确认 manifest/lock/toolchain 和实际构建入口，选择有效目标平台。
2. 判断受影响源文件、测试和公共合同的依赖闭包。
3. 先安排语法/类型/受影响测试，确有必要时执行集成/真实 Host/打包。
4. 审查 CI workflow 能否复用单次 checkout/缓存且不会漏依赖。
5. 对所选命令记录准确执行环境、SHA、exit code、首个失败点。

**输出：** 实际运行矩阵、跳过/复用理由与后续交付条件

**停机与边界：** 没有运行的 Provider/Host 证据保留 NOT_RUN，不因本地命令成功写 SUCCESS。

### DP-K08-02：多模块集成与变更收敛

**输入：** 目标 base/PR HEAD、合同/写集/依赖和原始测试收据

**步骤：**
1. 核对目标基础版本和最新候选，识别互相冲突的接口或生成产物。
2. 把正确已通过的相同源码/测试证据复用到受影响范围。
3. 对新组合必须测试的公共 seam 选择真正的 Consumer/Host。
4. 确认合并后当前源和生成产物版本与测试/安装版本一致。

**输出：** 集成版本、相应验证计划、缺口及需要修复的范围

**停机与边界：** 有实际兼容冲突/目标环境缺失时不合并为“已验证交付”。

### DP-K08-03：发布和首次启动验收

**输入：** 可打包的精确源码、目标平台、安装配置和授权 Release 规则

**步骤：**
1. 生成可定位的安装包/镜像/CLI 制品，绑定来源、平台与版本。
2. 按用户实际首次运行流程验证配置、异常和必要功能。
3. 核对所需 QA、审查、CI/Host 结果及发布元数据。
4. 交给正确 Release Owner 形成真实发行事实，追踪残余未验证范围。

**输出：** 交付资格清单与实际可用制品证据

**停机与边界：** Candidate 包存在不等于 GitHub Release 创建或用户安装成功。

这些步骤为可引用的决策方法，不是第二个调度器、权限授权或 Review/Release PASS。

## 方法适用性限制

这些 Decision Procedures 是可解释的候选工程知识，不自动创建新增 Runtime Workflow；遇到无权 Product/L2/Task/Host/Release 决策时应输出上游 Owner、缺失原始证据与最小下一步。已有真实证据可复用，不因模型上下文长或文档多而重复重审。