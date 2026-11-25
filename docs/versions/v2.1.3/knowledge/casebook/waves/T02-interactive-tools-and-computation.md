# T02 · 交互式软件、开发工具、科学计算架构案例

Task DAG [#567](https://github.com/kaicreator-mm/tally/issues/567)；T00 基线后首批实际交付。六种类型：移动 Android、桌面 CAD、游戏引擎、编译器、科学计算库、CLI 开发工具。8 个独立案例、10 个实际使用文件/6 个项目（文档及头文件/实现局部）。真实项目文档为 PROJECT_DOCUMENTED，声明/实现片段为 SOURCE_OBSERVED_SCOPED；参考架构及迁移建议是本库推论，全部验证意图 NOT_RUN。未编译/运行项目，无完整架构审计、性能或维护优胜证明；选定版本用于复核，不代表最新推荐版本。Android 固定实际 commit 并读回两文件，其他使用指定 release/tag 加实际 Blob；不复制原源码。项目名、功能族、粒度、质量/约束及已有 Kxx 明确，以供 T13 索引。

## T02-C01 · Now in Android：离线优先的移动应用架构

**类型/粒度：** 移动 Android / L1（项目文档视图）。**功能/机制：** 单向数据流、持久数据、同步、UI 状态。**知识：** K02.F3.1。**来源：** T02-S01、T02-S02（见下文）。**关系：** 机制关联；本地持久源与 UI 派生。验证意图另见正文，NOT_RUN。

- **来源实际支持：** 项目文档明确 UI、domain、data 三层；事件向下、数据向上。For You 屏幕启动后先等用户与新闻数据流，处于 Loading；WorkManager 同步任务通过 repository 和网络接口将远端数据更新到 Room，再由 DAO 数据流通知上层。用户兴趣数据来自本地 DataStore。文档明确该架构不是与 Clean Architecture 术语完全等价。
- **职责、依赖、状态/资源视图（分析）：** 视图：UI/ViewModel ← domain/use case ← repository ← 持久数据源；用户事件向下；后台 sync → network → repository → local DB → 数据流。Local 数据的实际写入与界面状态派生由不同职责负责，持久数据不应复制到多个 ViewModel 作为独立权威。
- **适用与不适用：** 适合需要离线读取、后台同步、多个屏幕消费同一数据的产品；如果只有在线单屏查询且无恢复要求，直接 API + 小状态对象可能更简单。
- **备选及反转条件：** 简单在线：API client + UI state；离线优先：repository + local database + sync；复杂跨设备多写：还需单独决定冲突策略和版本身份。不能从该项目说明推断任意业务的离线写入解决方案。
- **验证/反证建议（NOT_RUN）：** 网络失败、首次启动无本地数据、已有缓存后刷新失败、同步更新与用户兴趣变化、重复或旧同步结果。区分 Loading、无数据、失败及已缓存可用，不宣称实际已测试。

## T02-C02 · Now in Android：按功能分模块与导航 API

**类型/粒度：** 移动/多团队模块化 / L2–L3。**功能/机制：** app 组合、feature API/impl、core、依赖方向。**知识：** K02.F2.3。**来源：** T02-S02（见下文）。**关系：** 机制关联；feature API/impl 依赖方向。验证意图另见正文，NOT_RUN。

- **来源实际支持：** 项目模块化文档把 app 定义为组装和导航入口，feature 负责单一界面能力，api 子模块包含导航 key、impl 包含其余实现；feature impl 可依赖其他 feature api 而不是 impl；core 不依赖 feature/app。只有一个 feature 使用的类留在该模块。
- **职责、依赖、状态/资源视图（分析）：** 参考结构：app → feature-impl；feature-impl A → feature-api B；feature → 所需 core；core 只依赖下层 core。API 是真实跨模块导航合同，而不是把全部业务方法提前公共化。
- **适用与不适用：** 适用于多个功能独立变化、团队并行、编译或替换边界需要；单功能应用拆很多 gradle module 会增加构建/理解成本。
- **备选及反转条件：** 单模块按目录隔离、按业务 feature 模块化、真实消费者需要时增加窄 api；不要为了统一命名给每个模块都建立 api/impl。
- **验证/反证建议（NOT_RUN）：** 改变一个屏幕的实现不应迫使其他功能改变导航合同；禁止 core→feature；测试不同 app/flavor 接入真实导航 API。这里只核对文档，未审计全部 Gradle 依赖图。

## T02-C03 · FreeCAD：业务文档模型与多窗口视图

**类型/粒度：** 桌面/CAD / L2–L3。**功能/机制：** 文档模型、GUI 投影、重计算、事务与撤销。**知识：** K02.F3.1。**来源：** T02-S03、T02-S04（见下文）。**关系：** 机制关联；文档模型权威与视图。验证意图另见正文，NOT_RUN。

- **来源实际支持：** App::Document 声明对象添加、recompute、事务 open/commit/abort；Gui::Document 构造接收 App::Document，头文件说明 GUI 文档负责文档窗口及未保存关闭提示，并声明响应对象新增、属性变化、Undo/Redo、recompute 等信号的槽。
- **职责、依赖、状态/资源视图（分析）：** 参考：App Document（模型和事务）→ 模型变更信号 → GUI Document（视图协调）→ ViewProvider/窗口；GUI 用户操作通过模型合同改变文档，不让每个窗口拥有独立模型真值。这里只说明文档子系统声明，不推断整个 CAD 几何内核/实际 undo 原子性。
- **适用与不适用：** 适合复杂文档、多个视图、无 UI 模型操作、依赖重计算和撤销；简单一次性表单不必引入 CAD 文档对象体系。
- **备选及反转条件：** 普通 UI 状态+领域数据、共享文档模型+多个视图、独立进程计算服务；按模型复杂度、渲染/计算边界、撤销与持久要求选择。
- **验证/反证建议（NOT_RUN）：** 同一模型两视图、关闭一个窗口不误删除模型、模型变更传播、失败事务/撤销、重计算失败。观察真正模型状态和视图同步，不能只验证 UI。

## T02-C04 · Godot：场景树、节点组合与运行循环

**类型/粒度：** 游戏引擎 / L2–L3。**功能/机制：** 对象组合树、生命周期、Parent/Owner 区分。**知识：** K02.F2.1。**来源：** T02-S05、T02-S06（见下文）。**关系：** 机制关联；场景树组件与职责。验证意图另见正文，NOT_RUN。

- **来源实际支持：** Node 声明父子增删、查询、独立 set_owner/get_owner 和进入/退出、ready、物理/普通处理通知；SceneTree 继承 MainLoop，拥有根 Window，声明 initialize、physics_process、process、finalize。声明显示节点组合关系、Owner 及运行阶段分别表达，但不独自证明实际通知顺序或序列化所有权语义。
- **职责、依赖、状态/资源视图（分析）：** 参考：SceneTree 调度运行周期，Node 组合组织行为；Parent 表达结构，Owner 是另一种需明确的领域关系；生命周期由协调者与资源拥有者共同确定。
- **适用与不适用：** 适用于层级场景、组合组件和按帧运行；无层级的高密度同类数据批处理可能用扁平数据或 ECS 更合适。不得因 Godot 知名就将任何 App 模块组织为场景树。
- **备选及反转条件：** 直接对象组合、树结构、ECS/批数据处理；区分结构、资源拥有权、更新顺序和性能要求。
- **验证/反证建议（NOT_RUN）：** 节点移入/移出、重新挂载、暂停、重复激活和退出、资源释放；需要真实实现/测试确认时序，不由通知名称猜测。

## T02-C05 · LLVM：按 IR 粒度组合编译处理

**类型/粒度：** 编译器/代码分析 / L2–L3。**功能/机制：** Pass、IR 层级、适配、组合。**知识：** K03.F2.1。**来源：** T02-S07（见下文）。**关系：** 机制关联；IR pass 与适配职责。验证意图另见正文，NOT_RUN。

- **来源实际支持：** New Pass Manager 文档明确 Module → 可选 CGSCC → Function → Loop 层级；不同层级通过相应 adaptor 组合，一个特定粒度的 manager 本身可作为同粒度 pass。
- **职责、依赖、状态/资源视图（分析）：** 参考：输入 IR → 相应粒度 pass 或组合 pass manager → 输出或变更 IR。模块级和函数级的适用范围不同，适配器显式组织上下层，不把所有变换强制成无类型的通用 process(object)。
- **适用与不适用：** 适用于多轮分析/变换及明确中间表示；简单文本转换或固定验证步骤直接函数链即可。
- **备选及反转条件：** 普通函数序列、同类型 pipeline、具有嵌套 IR 范围的组合 pass；按上下文、状态影响及调度需求选择。
- **验证/反证建议（NOT_RUN）：** 错误粒度的 pass 不可随意执行；变换保留的语义、嵌套次序及空输入需检查。文档不证明任意 pass 都正确或可并行。

## T02-C06 · LLVM：分析缓存与跨层依赖的复杂度

**类型/粒度：** 编译器/静态分析 / L3–L5。**功能/机制：** 分析结果缓存、有效性、跨范围依赖。**知识：** K03.F1.1。**来源：** T02-S07（见下文）。**关系：** 机制关联；跨层分析缓存成本取舍。验证意图另见正文，NOT_RUN。

- **来源实际支持：** 文档描述分析 manager 按 IR 计算/缓存结果，有效时复用；跨层代理允许特定访问，内层不能随意重新计算外层分析。原作者解释可能导致二次复杂度及妨碍未来并行。
- **职责、依赖、状态/资源视图（分析）：** 分析结果以对象/范围及当前有效性为依据；修改 IR 后要确认结果仍适用，而不是用同名字复用旧结果。共享机制为缓存，失效条件仍归分析合同。
- **适用与不适用：** 当分析开销高且变更会影响结果时有价值；一次低成本纯计算无需复杂缓存。
- **备选及反转条件：** 直接重算、按对象局部缓存、显式失效/代理；只有真实变化/依赖要求时扩展。
- **验证/反证建议（NOT_RUN）：** 改动函数后依赖结果是否仍有效；不同 IR 不能串用结果；检查逐函数触发全模块分析的代价。未运行基准，不能声称优化倍数。

## T02-C07 · NumPy：数据缓冲与数组视图

**类型/粒度：** 科学计算/数组库 / L3–L5。**功能/机制：** 数据布局、元数据、别名、资源生命周期。**知识：** K03.F1.2。**来源：** T02-S08（见下文）。**关系：** 机制关联；数组数据表示与共享 buffer。验证意图另见正文，NOT_RUN。

- **来源实际支持：** 内部结构文档区分数据 buffer 与描述元素大小、偏移、shape、stride、byte order、只读和 dtype 等元数据；许多 slice/transpose 通过新视图共享底层数据，独立副本需要相应 copy，视图存在时底层缓冲仍保有引用。
- **职责、依赖、状态/资源视图（分析）：** 参考：Memory buffer 是实际数据资源，ArrayView 是解释与访问视图；形状/布局变化不必等同复制，但共享内存的可写性、生命周期、别名须有显式合同。
- **适用与不适用：** 适合大数组、零复制视图和跨算法布局；小数据或需要独立可修改快照时复制可能更清晰。并非所有 NumPy 操作都返回视图。
- **备选及反转条件：** 值拷贝、只读视图、共享可写视图；根据隔离、资源成本及消费者行为决定。
- **验证/反证建议（NOT_RUN）：** 视图修改是否影响原值、原对象释放后视图有效性、跨 stride 和非连续数据、只读限制；不能从对象不同推出数据独立。

## T02-C08 · ripgrep：CLI 入口与可复用搜索能力

**类型/粒度：** 开发工具/搜索 CLI / L2–L4。**功能/机制：** 入口适配、库 facade、模式分派与退出语义。**知识：** K02.F2.1。**来源：** T02-S09、T02-S10（见下文）。**关系：** 机制关联；CLI 入口与库职责。验证意图另见正文，NOT_RUN。

- **来源实际支持：** main 解析参数后进入 run，区分特殊命令、搜索/文件模式及单线程/并行，按匹配和错误决定退出码；BrokenPipe 有特定处理。grep library facade 实际重导出 matcher、printer、regex、searcher 等 crate。
- **职责、依赖、状态/资源视图（分析）：** 参考：CLI 参数/退出码/管道处理归入口；核心搜索能力按匹配、搜索、输出等真实库边界组合。CLI 错误处理不应成为所有库消费者的业务策略。
- **适用与不适用：** 适用于既有 CLI 又需嵌入其他应用的能力；只做一个短命令时无需抽完整 facade。
- **备选及反转条件：** 独立 CLI、共享库+CLI adapter、独立进程服务；实际其他消费者与语言边界决定是否增加公共接口。
- **验证/反证建议（NOT_RUN）：** 无匹配与执行失败不同退出码；下游管道关闭、单线程/并行结果合同、其他库消费者不被强制退出进程。未运行搜索或性能比较。

## 同类机制对照（Agent 选用提示）

| 真实问题 | 先参考 | 不应直接移植 |
|---|---|---|
| 多个屏幕共享数据，支持离线 | C01 数据源/同步及派生 UI 状态 | 将所有视图状态持久化为业务真值 |
| 多团队功能隔离与真实导航 API | C02 | 给所有私有对象都建公共 API |
| 多窗口共享可编辑业务模型 | C03 | 让每个窗口分别拥有同一业务状态 |
| 层级组合与周期/资源生命周期 | C04 | 用树结构替代一切数据/依赖模型 |
| 有明确粒度的多轮分析或转换 | C05/C06 | 无类型通用 pipeline、缓存无有效性 |
| 大型数据的低复制与独立副本 | C07 | 把共享视图当独立快照 |
| 核心能力被不同入口消费 | C08 | 将 CLI 退出策略塞入共享库 |

## 实际来源登记

- **T02-S01**：[android/nowinandroid / docs/ArchitectureLearningJourney.md](https://github.com/android/nowinandroid/blob/2982e63cb691ad49cd009aab43dc206fee6699c2/docs/ArchitectureLearningJourney.md) · ref `2982e63cb691ad49cd009aab43dc206fee6699c2` · Blob `d98dadf1e93ede12e0ed6d89dd8b818490a6433b` · L20–44, 65–134。项目架构文档：三层、单向数据流、同步及新闻展示例。
- **T02-S02**：[android/nowinandroid / docs/ModularizationLearningJourney.md](https://github.com/android/nowinandroid/blob/2982e63cb691ad49cd009aab43dc206fee6699c2/docs/ModularizationLearningJourney.md) · ref `2982e63cb691ad49cd009aab43dc206fee6699c2` · Blob `b50242776e797bdbf9069c2003be29c6ed18dbba` · L9–40, 66–88。项目模块化文档：app/core/feature 及 api/impl 的依赖约束。
- **T02-S03**：[FreeCAD/FreeCAD / src/App/Document.h](https://github.com/FreeCAD/FreeCAD/blob/1.0.0/src/App/Document.h) · ref `1.0.0` · Blob `c49434ff7b59831eb3170a25f483fb10d2ac4e00` · L61–73, 260–269, 351–360, 395–408。文档对象、重计算、事务操作声明。
- **T02-S04**：[FreeCAD/FreeCAD / src/Gui/Document.h](https://github.com/FreeCAD/FreeCAD/blob/1.0.0/src/Gui/Document.h) · ref `1.0.0` · Blob `42a3aae04c3cdba30998b8507ac6e9c8a8886333` · L61–111。GUI 文档持有 App 文档引用、视图职责及信号接线声明。
- **T02-S05**：[godotengine/godot / scene/main/node.h](https://github.com/godotengine/godot/blob/4.3-stable/scene/main/node.h) · ref `4.3-stable` · Blob `ee195ddef9655c64793365af5d1d9b3da5c45085` · L49–60, 366–375, 423–432, 474–480。Node 组合树、独立 Owner 接口、生命周期通知。
- **T02-S06**：[godotengine/godot / scene/main/scene_tree.h](https://github.com/godotengine/godot/blob/4.3-stable/scene/main/scene_tree.h) · ref `4.3-stable` · Blob `6f0a61ec51ed81220102b3354c05d09154a7f143` · L81–90, 123–132, 311–320。SceneTree 作为 MainLoop、根窗口及初始化/物理/普通处理/结束声明。
- **T02-S07**：[llvm/llvm-project / llvm/docs/NewPassManager.rst](https://github.com/llvm/llvm-project/blob/llvmorg-19.1.7/llvm/docs/NewPassManager.rst) · ref `llvmorg-19.1.7` · Blob `b75ab918fda0f3d5c1f8a06febe19edd8bad7a32` · L75–112, 191–248。项目开发文档：IR 粒度、适配、分析缓存与跨粒度计算限制及理由。
- **T02-S08**：[numpy/numpy / doc/source/dev/internals.rst](https://github.com/numpy/numpy/blob/v2.2.0/doc/source/dev/internals.rst) · ref `v2.2.0` · Blob `439645c374c49378e65e65b211e52c93165e1254` · L14–62。项目内部结构文档：数据缓冲、元数据、视图与复制/生命周期。
- **T02-S09**：[BurntSushi/ripgrep / crates/core/main.rs](https://github.com/BurntSushi/ripgrep/blob/14.1.1/crates/core/main.rs) · ref `14.1.1` · Blob `64f35cebb5ca7b0e35d66d3e1b4c669403429f26` · L43–106。真实入口：参数解析、模式分派、并行选择与退出码。
- **T02-S10**：[BurntSushi/ripgrep / crates/grep/src/lib.rs](https://github.com/BurntSushi/ripgrep/blob/14.1.1/crates/grep/src/lib.rs) · ref `14.1.1` · Blob `6c88bf7eb65243fdd2f2f1d0934f4dc1fba759fa` · L1–21。库 facade 的实际 re-export：CLI、matcher、printer、regex、searcher。

## 资格和待补

T02 的至少四类真实架构/子系统参考验收范围已满足；6 类全部有资料，移动跨平台框架及更广桌面业务系统、独立游戏具体应用、完整编译前端/后端、CAD 完整几何链均未系统覆盖，保留扩展缺口。一次 NumPy 候选路径 under-the-hood.rst 未取得（404），实际使用并读取 internals.rst；未将未读取路径计入来源。未取得额外全项目调用图；L1 仅 Android 有足够项目架构/模块整体文档，其他保持局部视图。当前仅 DELIVERED_SCOPED 文档候选，不代表 T14 独立接受。T13 统一索引时复核关系及引用适切性，后续 T09/T11/T12 可选取 C02、C03、C05–C08 作真实源对照，不能把此文档当已授权修改指令。
