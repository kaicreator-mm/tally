# T20 · 真实重构历史：简化、迁移与合同修复

[#578](https://github.com/kaicreator-mm/tally/issues/578) / #563；基线 ddef3026fa47000f60219e27c071f77523a929a7。三个 L4 历史源码案例、七个新版本文件 Blob、三个实际 Git commit 记录（提交及其 patch 不另计来源文件）。不是从现有实现猜测历史。区分维护者明确说明的动机、实际变更、工程推论、未验证收益。所有运行、回归测试、性能及维护经济性测量 NOT_RUN；测试 patch 仅为测试意图。

## 增量目录

| 案例 | 实际变化 | 关联与限定理由 | 来源 |
|---|---|---|---|
| T20-C01 / L4 | pytest 内部 helper → 直接边界判断 | K03.F2.3 机制：去除因异常控制流所需的额外函数和解释；K03.F2.1 条件：是否保留函数应依据真实独立职责 | T20-S01/S02, H01 |
| T20-C02 / L4 | HTTPX 旧 app shortcut → 显式 transport | K02.F2.2 机制：弃用、兼容窗口与后续接口删除；K02.F2.3 机制：明确既有 transport 选择扩展边界 | T20-S03–S05, H02 |
| T20-C03 / L4 | p-limit 包装函数补回 clearQueue | K02.F2.2 机制：运行时、类型、文档和测试合同同步；K03.F3.3 机制：未开始调用的队列清理与已运行任务区别 | T20-S06/S07, H03 |

## T20-C01 · 删除内部抽象：pytest

**真实历史：** H01 提交说明指出旧 getnextitem 通过 try/except IndexError 取下一测试项，单独函数用于避免 Python 2 中异常信息保留到测试调用（源码注释与说明）。维护者明确认为改用 if/else 可以减少该抽象及解释。父提交 S01 L135–153 定义 getnextitem 并在循环调用；子提交 S02 L135–144 将其换成 i+1 < len(session.items) 的条件表达式，末项返回 None；collectonly、hook 调用与 shouldstop 分支仍在。实际 compare 仅 AUTHORS +1、_pytest/main.py +1/-10，未见新增测试。此变更是控制流与内部结构简化，不是 Python 2 支持移除，也不证明任意对象实现下等价。

**选择与代价（推论）：** 若 helper 仅因当前实现技巧存在且没有独立业务责任，先考察语言已有的更直接控制流；本例实际删除一个函数定义及一处调用，降低局部阅读跳转（结构证据）。若下一项来自外部 iterator、特殊索引语义或需共享异常合同，长度判断可能不适用，不能将“短函数均应 inline”设为规则。未测维护时间、故障率或性能。

**反例/Oracle：** 在原支持环境核实 collectonly、空列表、单项、多项末尾 None、shouldstop 及 hook 参数；检查测试开始时异常状态符合原合同。若自定义序列的 len 和索引行为不同或集合在取值时变化，重新裁定，不宣称一般等价。NOT_RUN。

## T20-C02 · 删除公共便捷入口：HTTPX

**真实历史：** H02 #3050 提交明确将 app shortcut 弃用，推荐显式 WSGITransport/ASGITransport，并非立即删除；实际 _client.py patch 在同步/异步构造增加条件 warning，且还有异步 proxy 环境条件变化。实际版本源码：0.26.0 S03 同步和异步 Client 接收 app，_init_transport 优先已传 transport，否则 app 非 None 时构造相应 WSGI/ASGI transport，再走默认 HTTP transport。0.27.0 S04 保留该入口和分支，构造器在 if app（truthy 条件，非所有非 None）时发 DeprecationWarning。0.28.0 S05 同步和异步构造签名、_init_transport 均已无 app 参数/自动 app 分支，仍可接受显式 transport。这里用版本快照观察移除结果，不定位删除的单一提交，不把三版本间所有变更归因于 H02，也不声称代理、SSL 等其他合同完全等价。

**选择与代价（推论）：** 真实扩展点已存在且 shortcut 增加重复选择语义时，可优先统一到现有显式机制，而非再增加工厂/适配层。公共 API 不是内部 helper：删除导致使用旧 keyword 的调用者需要迁移，需考虑用户/依赖方、弃用提示、版本窗口和替代例子，不能仅凭减少参数判定整体维护成本下降。已观察的维护事实是旧入口先保留并警告，后在另一版本删除；尚无消费者数量、迁移耗时或真实回归成本记录。若公共 API 兼容必须永久维持且 shortcut 仍有真实用户价值，则保留或提供有限兼容层也合理。

**反例/Oracle：** 0.26/0.27 的 app 与 transport 优先级、0.27 truthy app 警告、0.28 不接受旧 keyword；迁移后分别以同步 WSGI 和异步 ASGI 显式 transport 请求同一应用，验证响应、异常和关闭行为，另核 proxy/trust_env，不假设所有组合等价。实际版本组合及消费者升级均 NOT_RUN。

## T20-C03 · 增加必要抽象能力：p-limit

**真实历史：** H03 #109 标题明确修复 limitFunction 的 queue clearing。父提交 S06 与子提交 S07 都有 pLimit 的 rejectOnClear 选项，默认 false；clearQueue 在 false 时清队列，在 true 时逐个 dequeue 并 reject AbortSignal.abort().reason（L3–14,77–90）。旧 limitFunction 创建私有 limit 仅返回调用闭包（L117–121），没有对外暴露 clearQueue；新实现保留闭包调用路径，增加该闭包的 clearQueue 属性指向已有 limit.clearQueue，再返回闭包（L117–126）。没有另建清理实现。H03 实际 patch 同时改变 index.d.ts 返回类型为函数与 Pick<LimitFunction,'clearQueue'> 的交叉类型，index.test-d.ts 增加调用的类型断言，readme 增加未开始调用的清理与 rejectOnClear 合同，test.js 新增运行一个任务、另一个待办、调用 wrapper.clearQueue、等待运行完成并断言待办拒绝 AbortError 的测试。实际 compare 五文件 +28/-2。测试源码存在不等于测试通过。这里是后续历史版本，不把新选项或 wrapper 修复归给此前 T15-C01 的 v6.2.0。

**选择与代价（推论）：** 包装层需保留真实使用者所需的操作能力，而不是仅复制最常用调用形态；如果队列属于包装对象且用户需要清理，直接委托已有能力比重新实现队列管理合理。仍需保留“不取消已经运行任务”和默认清理后 pending Promise 可能一直未完成的边界（README patch 明示）；增加方法不等于解决所有取消、背压或持久性问题。反过来，如果调用者合同仅允许提交且清理责任在其他 owner，盲目暴露底层全部方法可能破坏隔离；需要最小权限与职责判断。实际结构增加一个必要方法及类型/文档/测试变更，不提供维护耗时或性能收益实测。

**反例/Oracle：** 老 wrapper 无 clearQueue，新 wrapper 的方法可调用；两个 wrapper 之间不应误清彼此待办；rejectOnClear=true 时待办拒绝且已开始任务完成，false 时不把清理当作 Promise settlement；类型入口与实际运行行为一致；不取消运行中任务。NOT_RUN。

## 历史证据与决策准则（原创综合，不增加案例）

| 判断 | 可证实的变化 | 不能推出 |
|---|---|---|
| 删除内部 helper | pytest 一个函数及解释消失，直接条件判断 | 所有短函数都是错误抽象 |
| 收敛公共入口 | HTTPX 真实弃用及删除过程，调用者改传 transport | 兼容无损、全版本行为一致 |
| 恢复封装合同 | p-limit 新委托方法且类型/文档/测试同步 | 代码越少越好或所有底层 API 都应暴露 |

重构决策：先辨别真实变化需求/故障和现有责任，区分内部结构与公共契约，再选择删除、保留、合并、委托或分离；在受影响边界核实可观察行为，公共迁移须考虑使用者。记录实际代码差异及维护者理由只是历史证据；若要声称降低维护成本，应另有同类变更时间、涉及模块数、缺陷/回归及迁移成本等可比较证据（控制任务与环境），不能以 LOC、模式数量、提交接受或作者意见单独替代。T20 未获得这些量化资料，不能宣称重构收益已实证。

## 来源登记（七个实际源码版本）

- **T20-S01** [pytest-dev/pytest _pytest/main.py](https://github.com/pytest-dev/pytest/blob/66e66f61e8e3b53ef824e01679f50aa8a452241d/_pytest/main.py) · commit 66e66f61e8e3b53ef824e01679f50aa8a452241d · Blob 4a6c0877557c34cc55b23e5c0e62b43301cd596e · L135–153。
- **T20-S02** [pytest-dev/pytest _pytest/main.py](https://github.com/pytest-dev/pytest/blob/ab8b2e75a30d34799311bb2b0f85881842c410b9/_pytest/main.py) · commit ab8b2e75a30d34799311bb2b0f85881842c410b9 · Blob d8d6dc12e9d7f751fbee877f87816e8fc2ae9528 · L135–144。
- **T20-S03** [encode/httpx _client.py](https://github.com/encode/httpx/blob/0.26.0/httpx/_client.py) · ref 0.26.0 · Blob 2813a84f0108b48be7cdabf916057ab81cf7e585 · L630–746,1380–1480（完整文件实际读取）。
- **T20-S04** [encode/httpx _client.py](https://github.com/encode/httpx/blob/0.27.0/httpx/_client.py) · ref 0.27.0 · Blob e2c6702e0c7ecb0d9e12ae41d19a156daaed6981 · L630–743,1375–1490。
- **T20-S05** [encode/httpx _client.py](https://github.com/encode/httpx/blob/0.28.0/httpx/_client.py) · ref 0.28.0 · Blob 018d440c1732ad6bb32c0f96d27a05cec8d01d3c · L630–747,1360–1460（完整文件实际读取）。
- **T20-S06** [sindresorhus/p-limit index.js](https://github.com/sindresorhus/p-limit/blob/df476048d023ff868cd45b35ee47f5fb0ca2b25a/index.js) · commit df476048d023ff868cd45b35ee47f5fb0ca2b25a · Blob 5ecd39257c623b6dd4df204bbd7c94fbdf0b8d12 · L3–14,77–90,117–121。
- **T20-S07** [sindresorhus/p-limit index.js](https://github.com/sindresorhus/p-limit/blob/f3e7f9ba364a9357bd912d136367d06c46660917/index.js) · commit f3e7f9ba364a9357bd912d136367d06c46660917 · Blob b74d48ab0e37788e657e5ad12d90e281ef1f39d1 · L3–14,77–90,117–126。

实际 Git 历史记录（不重复计作 Blob）：
- H01 [pytest ab8b2e75a30d34799311bb2b0f85881842c410b9](https://github.com/pytest-dev/pytest/commit/ab8b2e75a30d34799311bb2b0f85881842c410b9) · commit message、实际两文件 patch、实际 compare 父提交 66e66f61e8e3b53ef824e01679f50aa8a452241d。
- H02 [HTTPX cabd1c095e52e2f67ab63ec17941d3a539d7c877](https://github.com/encode/httpx/commit/cabd1c095e52e2f67ab63ec17941d3a539d7c877) · commit message、实际 _client.py 弃用 patch；其他变更文件不作为已执行证据。
- H03 [p-limit f3e7f9ba364a9357bd912d136367d06c46660917](https://github.com/sindresorhus/p-limit/commit/f3e7f9ba364a9357bd912d136367d06c46660917) · commit message、实际五文件 patch、compare 父提交 df476048d023ff868cd45b35ee47f5fb0ca2b25a。

原知识定义：[K03.F2.1/F2.3](https://github.com/kaicreator-mm/tally/blob/ddef3026fa47000f60219e27c071f77523a929a7/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L40-L62)、[K02.F2.2/F2.3](https://github.com/kaicreator-mm/tally/blob/ddef3026fa47000f60219e27c071f77523a929a7/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L48-L62)、[K03.F3.3](https://github.com/kaicreator-mm/tally/blob/ddef3026fa47000f60219e27c071f77523a929a7/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L83-L89)。

反向增量：K03.F2.3→C01 机制、K03.F2.1→C01 条件；K02.F2.2→C02/C03 机制、K02.F2.3→C02 机制；K03.F3.3→C03 机制。五机制/一条件。原165概念不变。七个新版本 Blob 未在15份既有来源登记中出现；三个历史记录仅作为来源说明。没有真实工时/缺陷率/性能或消费者升级运行数据，不以此作为知识收益的因果证明。待独立审核。
