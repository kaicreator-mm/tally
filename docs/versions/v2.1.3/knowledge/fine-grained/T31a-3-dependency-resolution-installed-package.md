# T31a-3 · 依赖约束、解析图与真实安装制品

#622 / R4-D1 / OP-02；K08 主责生成/交付，K03 语言与加载，K09 消费者演进，K05 验证。复用 [T31a-1](T31a-1-compilation-linking-abi.md)、[T31a-2](T31a-2-configuration-management-baselines.md)、[T28a-2](T28a-2-consumer-contracts-installed-artifacts.md) 已有 ABI、基线、真实 consumer，不再解释编译与链接基础。本章限定 npm CLI v11；示例均教学 NOT_RUN。

## 1. 区分声明图、解析图、安装位置和实际加载图

一个依赖边包含调用者/包身份、允许版本集合、类型（运行、开发、peer、optional）、目标平台条件及所需能力；解析选择的是带身份的版本与放置位置，不只是每个名称一个版本。两个普通依赖可通过嵌套安装不同版本满足，但同一插件宿主位置的 peer 合同不能据此假定满足。锁文件记录确定的解析树及内容信息，不替代实际安装/加载检查。具体 npm CLI v11 文档：peer 用于声明与 host 的兼容，自 v7 默认安装 peer；冲突可能使解析失败；optional peer 不自动安装，optionalDependencies 安装失败可以继续，因此调用者须实际处理缺失。不要把忽略 peer 冲突当修复。

教学 registry（仅以下版本存在）：app 声明 host ^2.0.0，pluginA peer host ^2.1.0，pluginB peer host ^3.0.0；host 可用 2.0.0、2.2.0、3.1.0。假定插件必须与 app 使用同一 host 实例。app+A 的交集含 2.2.0；加入 B 无交集。即使在某处单独装 3.1.0 也不证明 B 能参与同一 host。可选择支持2.x的 B 版本、迁移整组到3.x并复核 A、隔离不同 host 消费者，或拒绝该组合。不是声称 npm 对任意复杂树一定以固定顺序选择这些版本；本例为有限约束推理。若独立工具仅各自使用 host，不共用接口/实例，则可允许不同版本；不能机械要求 monorepo 一切依赖同版。

## 2. Workspace 链接与实际包消费者

有多个协作子模块时可选择 monorepo：根 workspace 描述 packages/app 与 packages/pluginA；pluginA 声明 host peer，app 声明实际 host 与 pluginA。npm v11 workspaces 机制会将识别到的本地包链接入 node_modules，并记录正常版本约束；安装位置可能由 hoist 决定。若 pluginA 实际 import codec，但只在 app 或根 devDependencies 声明 codec，workspace 测试可能被提升路径掩盖；真实装包消费时缺失。直接消费依赖由实际消费包声明，不能凭传递包路径或工作区碰巧可解析就认定合同成立。不要为单个功能拆多个包；依据实际共享语义、独立消费者和变更需求选择。

npm v11 package-lock 格式中 lockfileVersion=3 为 npm v9+ 的格式；package.json 与锁不一致时 npm ci 报错而不是修锁，存在 node_modules 则先移除。应记录实际 npm 版本、影响树的配置（如 peer 解析及链接选项）、目标 OS/CPU、Node、registry 与 lock 内容；不把相同锁解释为各平台全部可选产物相同。npm ci 只说明安装路径按实际命令运行，不证明生成正确、无安装脚本或安全。若选择禁止生命周期脚本，明确项目合同和命令；ignore-scripts 不表示显式 npm run 不运行目标脚本。某 native optional 包在 Windows 构建失败仍安装成功的情况下，应用必须有真实合法 fallback 或明确拒绝所需功能，不允许缺能力却声称成功。

## 3. 制品选择和身份闭包

npm v11 files 控制 pack 内容（另有必含/排除规则）；package-lock.json 默认不能随包发布；若需要发布锁策略须研究实际 npm-shrinkwrap 适用合同。教学插件 exports 指向 dist/index.js 与 dist/index.d.ts；若 files 仅列 src，则实际 tarball 可能缺 dist；若编译使用 workspace 相邻 src，即使 dist 存在也可能错误。明确生成输入和输出、生产依赖、入口、类型及兼容策略。npm pack 针对包产生 tgz；观察实际内容与 digest，不以 dry-run 或源码目录代替真实制品。

| 独立验证（NOT_RUN） | 预期/反例 |
|---|---|
| 声明/锁不一致 | 实际 ci 拒绝且不修锁；固定命令、版本和配置；不得将修改锁后的成功冒充原输入 |
| peer 冲突 | 在同一 host 合同下无解；检查实际安装图和加载 host 身份，不用 --force/legacy-peer-deps 掩盖 |
| optional 缺失或目标平台不同 | 依有权能力决定 fallback/拒绝；分别观测 actual resolve 和业务结果 |
| 源码测试正常但 tarball 缺 dist/codec | 隔离消费者不访问 workspace/全局路径；实际安装 tgz 后按公开 exports、类型及业务正负例验证 |
| 新旧 host/plugin 版本组合 | 固定消费者矩阵、真实加载版本、行为与错误；不得以 semver 本身证明可用 |
| 同名 tgz/锁被替换 | 核输入/工具/实际文件和 digest；同名不等同同内容 |

实际资料 2026-11-26：npm CLI v11 [package.json](https://docs.npmjs.com/cli/v11/configuring-npm/package-json/) files L541–585、peer/peerMeta L1009–1039、optional L1089–1124；[package-lock](https://docs.npmjs.com/cli/v11/configuring-npm/package-lock-json/) L288–310；[npm ci](https://docs.npmjs.com/cli/v11/commands/npm-ci/) L284–287、382–389；[workspaces](https://docs.npmjs.com/cli/v11/using-npm/workspaces/) L333–345；[npm pack](https://docs.npmjs.com/cli/v11/commands/npm-pack/) L330–336。未运行 npm、验证全部生态或证明真实树；教学约束与 oracle NOT_RUN；OP-02 限定目标候选。
