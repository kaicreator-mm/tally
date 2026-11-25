# T11 · 真实软件演进与重构历史案例

[#567](https://github.com/kaicreator-mm/tally/issues/567)。六个真实版本差异案例，16 个新增来源文件，另复用 T08-S05、T05-S05。五个既有项目；历史 release/tag 及实际 Git Blob，观察限定于指定文件。六种变化分别是资源合同修正、并发取舍、已废止接口退役、输入收紧、API 迁移、路由结构与兼容机制替换。不是全部纯行为保持重构；不将旧设计笼统评为错误，也不从不同版本源码推断提交因果、维护收益或性能。原作者说明与本库推论分开；测试源码观察/迁移说明不是实际执行。所有验证 NOT_RUN。

## T11-C01 · Go slices：删除操作补足资源引用清理

**分类：** L6 · 资源与表示合同修正。**知识：** K03.F4.2 K09.F2.1 K05.F1.2。**来源：** T11-S01/S02/S03；T09-C02。

- **旧版本实际观察：** 1.21.13 的 Delete/ DeleteFunc 直接缩短 slice，注释允许尾部元素未改变，并建议调用方按需清零。
- **新版本实际观察：** 1.22.0 的 Delete 使用长度受限的三索引切片做边界检查、保存旧长度、原地移动并 clear 已废弃尾部；DeleteFunc 同样 clear。测试 TestDeleteClearTail 与 TestDeleteFuncClearTail 明确要求废弃引用清空且原 len 之外的元素不变。
- **原项目证据与动机边界：** 源注释说明清除旧引用供 GC；测试注明避免潜在内存泄漏。不能从这些证据推断真实内存下降量。
- **合同差异（分析）：** 返回有效元素的主要合同仍在，但底层数组尾部现在确定被清零；边界检查也改变了可接受索引范围，旧别名观察可能不同。不是纯重命名，也不能推广成所有删除都必须复制。
- **迁移参考、反例及验证意图：** 若 API 返回共享 backing storage 的视图，删除后的残留、别名和释放策略需要明确；如独立快照则用复制方案。用旧/新测试固定原长度内外边界、全删、无删、别名及非法索引。源测试已读取，未运行。

## T11-C02 · CPython cached_property：移除跨实例共享锁

**分类：** L6 · 并发合同与锁粒度变化。**知识：** K03.F3.1 K10.F3.2 K09.F3.1。**来源：** T11-S04/S05/S06。

- **旧版本实际观察：** 3.11.0 descriptor 持有 RLock；miss 后加锁、二次检查，再计算并缓存。
- **新版本实际观察：** 3.12.0 删除 descriptor 锁和二次检查，miss 后直接计算写入。文档说明多线程下同实例 getter 可能重复执行，后完成者可能覆盖缓存。
- **原项目证据与动机边界：** 项目文档明确旧锁是 per-property 而非 per-instance，会引起不可接受的锁竞争；这是作者说明，不是本库实测。
- **合同差异（分析）：** 公开访问形式不变，实际并发执行次数/等待改变；旧的未文档化锁保证不再存在。新调用者若依赖单次效果必须自行同步。
- **迁移参考、反例及验证意图：** 对照 T05-C06 原子加载与 T09-C08 Once：重复无害的计算可以允许竞争；有资源/业务副作用的初始化须明确等待和失败重试。检查同实例并发、不同实例并发、异常、重复 getter；不报告性能收益。

## T11-C03 · Django：退役 pytz 兼容路径和 is_dst 参数

**分类：** L6 · 已声明废止的兼容接口退役。**知识：** K09.F3.1 K09.F4.1 K01.F3.3。**来源：** T11-S07/S08/S09。

- **旧版本实际观察：** 4.2 make_aware 含 is_dst 参数及 deprecation warning；pytz 分支 localize，其他分支检查 naive 后 replace。
- **新版本实际观察：** 5.0 函数仅保留 value/timezone、naive 检查及 replace；release notes 明列 USE_DEPRECATED_PYTZ、pytz 支持、is_dst 参数移除。
- **原项目证据与动机边界：** 源码警告 is_dst 对 zoneinfo 无效；发布说明证明退役合同，不推断移除全部时区 bug 的动机。
- **合同差异（分析）：** 旧参数/旧后端不再受支持，DST 边界仍有源码警告；不是行为完全不变的重构。旧调用方不能只删参数就认定歧义时间语义等价。
- **迁移参考、反例及验证意图：** 明确废止支持范围后可去除旧分支；未授权废止则先兼容迁移。检查 aware/naive、普通时间、DST 重叠/不存在时间、旧调用参数，必要时独立确定歧义政策。未运行。

## T11-C04 · Redux 4→5：动作类型由建议变为运行时约束

**分类：** L6 · 输入合同收紧与生态兼容。**知识：** K01.F3.3 K03.F5.1 K09.F3.1。**来源：** T11-S10/S11；T08-S05。

- **旧版本实际观察：** 4.2.1 dispatch 在 plain object 检查后拒绝 undefined type，未在该入口强制 string。
- **新版本实际观察：** 5.0.1 同入口新增 typeof action.type !== 'string' 拒绝；迁移指南指出旧代码可能使用 Symbol，推荐字符串以支持可序列化与可读历史。T08-S05 已登记 5.0.1 src/createStore.ts Blob e5f3b6d7bad1208ee74b11dece2e3634b4d3acf8，本轮重读 L271–296。
- **原项目证据与动机边界：** 指南明确此前建议 string、现在强制；不将作者对影响比例的估计当本库证据。
- **合同差异（分析）：** 合法输入域缩小，原入口可能接收的 Symbol/数字类型现在被拒绝。单纯 TS 类型变化不代表完成运行时约束；实际代码新增检查。
- **迁移参考、反例及验证意图：** 对照 T05-C04/C05：版本收紧需真实消费者/迁移说明。检查数字、Symbol、字符串、undefined、middleware 转换及旧 reducer；不把 action 类型约束当业务授权。未运行。

## T11-C05 · Pydantic：API 迁移与验证/数据加载边界

**分类：** L6 · 公共接口演进及过渡 facade。**知识：** K02.F2.2 K03.F5.1 K09.F3.1。**来源：** T11-S12/S13；T05-S05。

- **旧版本实际观察：** 1.10.15 parse_obj 对非 dict 尝试 dict(obj)，失败包装 ValidationError，最后 cls(**obj)。
- **新版本实际观察：** 2.10.4 parse_obj 发出弃用警告并转发 model_validate；model_validate 支持 strict/from_attributes/context。迁移文档列重命名表、parse_raw/parse_file 退役方向及 from_attributes 替代 from_orm，并指出模型相等合同改变。T05-S05 同 Blob ca5a4529e9e398ccce2606acebfda1673f55185d，本轮重读 L603–629,1215–1223。
- **原项目证据与动机边界：** 迁移指南明确保留部分旧方法以帮助迁移，并要求非 JSON 数据先加载再验证。
- **合同差异（分析）：** 保留旧方法名不保证原语义完全相同；相等关系、数据输入域及错误需要逐一比较。数据加载与对象验证可分离，迁移 wrapper 有明确过渡用途。
- **迁移参考、反例及验证意图：** 真实受支持旧消费者需要窄兼容入口及迁移窗口；无旧消费者则直接新接口。检查旧对象/字典/属性来源、严格模式、警告、相等和错误；不宣称两个版本全部兼容。未运行。

## T11-C06 · Go ServeMux：路由表达能力扩展和旧行为隔离

**分类：** L6 · 算法/结构替换与启动级兼容开关。**知识：** K02.F2.1 K02.F2.2 K09.F4.1。**来源：** T11-S14/S15/S16。

- **旧版本实际观察：** 1.21.13 ServeMux 有 map、按长度排序的 entries 与 hosts 标记；Handler 处理 CONNECT、路径清理后匹配。
- **新版本实际观察：** 1.22.0 使用 routingNode/routingIndex/patterns，并保留 mux121；支持方法/host/path/wildcard，非法 pattern 可 panic，URL segment 解码合同改变。Handler 根据 use121 分派；servemux121.go 保留旧结构，init 只在启动时读取 httpmuxgo121。
- **原项目证据与动机边界：** 源码兼容说明明确旧行为恢复方式，并说明动态切换过于复杂易错；不是本库估计。
- **合同差异（分析）：** 新语法改变原 literal braces、无效 pattern、转义等输入合同；兼容开关是同一 API 的显式旧行为路径，而非自动语义等价。
- **迁移参考、反例及验证意图：** 有已存在路由消费者且允许升级窗口时隔离旧实现；无旧兼容义务的重写可直接新路由。检查字面花括号、GET/HEAD、无效路由、转义/冲突及启动切换；未检查完整 routing tree 或测试套件，NOT_RUN。

## 实际新增来源登记

- **T11-S01**：[golang/go/src/slices/slices.go](https://github.com/golang/go/blob/go1.21.13/src/slices/slices.go) · ref `go1.21.13` · Blob `afeed0afb522ceafe29a63aaa0194ae7b9e2d46b` · L212–246。
- **T11-S02**：[golang/go/src/slices/slices.go](https://github.com/golang/go/blob/go1.22.0/src/slices/slices.go) · ref `go1.22.0` · Blob `b0f048a6566014875edd80606d12cb4b82fce2ce` · L215–249。
- **T11-S03**：[golang/go/src/slices/slices_test.go](https://github.com/golang/go/blob/go1.22.0/src/slices/slices_test.go) · ref `go1.22.0` · Blob `31d59abe17b3734d9e35fa0de9e1813d4eec5acc` · L693–724。
- **T11-S04**：[python/cpython/Lib/functools.py](https://github.com/python/cpython/blob/v3.11.0/Lib/functools.py) · ref `v3.11.0` · Blob `43ead512e1ea4e63c2b82c4eb3f8db75d4e1eb92` · L965–971,995–1011。
- **T11-S05**：[python/cpython/Lib/functools.py](https://github.com/python/cpython/blob/v3.12.0/Lib/functools.py) · ref `v3.12.0` · Blob `2ae4290f98398296b8b5f98e49f355b6bccce205` · L964–970,993–1007。
- **T11-S06**：[python/cpython/Doc/library/functools.rst](https://github.com/python/cpython/blob/v3.12.0/Doc/library/functools.rst) · ref `v3.12.0` · Blob `f736eb0aca1ac5d1436397047b2cd0c79145aa2c` · L94–100,121–125。
- **T11-S07**：[django/django/django/utils/timezone.py](https://github.com/django/django/blob/4.2/django/utils/timezone.py) · ref `4.2` · Blob `f3eac0e7b263abd4c457a9afc9882e5032f641a1` · L272–291。
- **T11-S08**：[django/django/django/utils/timezone.py](https://github.com/django/django/blob/5.0/django/utils/timezone.py) · ref `5.0` · Blob `102562b254a3d2a35f6ab370b05250193eae3d12` · L237–245。
- **T11-S09**：[django/django/docs/releases/5.0.txt](https://github.com/django/django/blob/5.0/docs/releases/5.0.txt) · ref `5.0` · Blob `4c86337b7456e302a5bf67c99c97f202b3b410bb` · L675–691。
- **T11-S10**：[reduxjs/redux/src/createStore.js](https://github.com/reduxjs/redux/blob/v4.2.1/src/createStore.js) · ref `v4.2.1` · Blob `b588e3a04d98000e3c1d60d4047fd6ea354e6d9e` · L197–217。
- **T11-S11**：[reduxjs/redux/docs/usage/migrations/migrating-rtk-2.md](https://github.com/reduxjs/redux/blob/v5.0.1/docs/usage/migrations/migrating-rtk-2.md) · ref `v5.0.1` · Blob `3764058029d21bd8a6f6a3f95e2fc9b6401cbdce` · L69–83。
- **T11-S12**：[pydantic/pydantic/pydantic/main.py](https://github.com/pydantic/pydantic/blob/v1.10.15/pydantic/main.py) · ref `v1.10.15` · Blob `08b8af558ea15d0f4b26964677841826da8399e2` · L518–526。
- **T11-S13**：[pydantic/pydantic/docs/migration.md](https://github.com/pydantic/pydantic/blob/v2.10.4/docs/migration.md) · ref `v2.10.4` · Blob `1f4404e822730375e3edb1bbcab4cde0a2abce0c` · L131–159。
- **T11-S14**：[golang/go/src/net/http/server.go](https://github.com/golang/go/blob/go1.21.13/src/net/http/server.go) · ref `go1.21.13` · Blob `111adb0ecd6e9fd2a732a69705440d34e65a7945` · L2325–2335,2455–2474。
- **T11-S15**：[golang/go/src/net/http/server.go](https://github.com/golang/go/blob/go1.22.0/src/net/http/server.go) · ref `go1.22.0` · Blob `acac78bcd0c54b07b50ec9dd99e3e7e58bcb67ef` · L2322–2363,2412–2438,2500–2507。
- **T11-S16**：[golang/go/src/net/http/servemux121.go](https://github.com/golang/go/blob/go1.22.0/src/net/http/servemux121.go) · ref `go1.22.0` · Blob `c0a4b7701056095806f7582f0ad87331606f4f2b` · L21–39,100–119。

复用来源：[T08-S05](T08-platform-functions.md)、[T05-S05](T05-common-function-abstractions.md)，已按原 ref 重读，未重复登记。候选 Go doc/go1.22.html 与 Redux v4.2.1 src/createStore.ts 不存在；分别使用真实 test 文件和旧 JS 文件，未计不可读文件。

## 演进方法小结与交接

按 T10 D1/D2 先确定有权变化与兼容集合，再识别旧/新实际行为及源证据。六例均有前后文件与原项目测试或迁移/兼容说明；只有 C01 读取具体新测试源码，其余使用原项目文档或源码注释的迁移说明。未做跨版本运行、完整 commit 链或真实客户迁移；实际收益/未覆盖变更 UNKNOWN。T11 DELIVERED_SCOPED，待 T14 独立审查。