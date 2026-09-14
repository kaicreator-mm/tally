// Package failure 诊断失败并验证修复。
//
// 干什么:查知识库 → 未命中则调 llm 提案 → 四道闸验证 → 归类 → 记账。
//
// 四道闸(执行顺序与编号不同,各有讲究):
//
//	G4 修复只改白名单内的文件(静态判定,零成本;改验证文件直接拒绝,必须先于执行)
//	G1 连跑 5 次全红,确认稳定复现(4 红 1 绿算 flaky)
//	G2 应用修复后连跑 5 次全绿
//	G3 其余构建目标不受影响
//
// G4 特别规定:改到应用源码 → 暂停等人确认;改到验证用的文件 → 直接拒绝,
// 返回 ErrFixTouchesTests(不然 LLM 最省事的过关方式就是把检查放松)。
//
// 五种出口,必须显式归类:fixed / flaky / upstream / infra / spec_defect。
// 判定永远在确定性代码(gates/classify)里,LLM 只提候选(铁律 1):
// 一轮诊断至多一次 LLM 调用;spec_defect 提议还要过「条款确实存在」的确定性闸门。
//
// 预算:诊断三轮不过归档「暂无自动修复」,同指纹再命中直接返回该结论,不再调 LLM。
//
// 缺复现上下文(Event.Repro 为空)时跳过 G1/G2:本轮不提候选、不验证、零 LLM,
// 次数照计(预算仍三轮归档);知识库精确命中在入口短路,不受影响。不假装验证过。
//
// 依赖全部注入,业务包互不导入、协作由上层(web/jobs)编排:
// airuntime.Executor(通用推理只提案,经公共 AI Runtime 边界,P0-11)、
// Runner(执行器 port,真实实现是 runner 包)、
// KnowledgeBase(knowledge.Store 满足)、StatStore(包装 fingerprint_stat)、
// SpecDefectSink(requirement 回写,触发失效传播)。
//
// 允许导入:core、airuntime、logpipe、store。不导入 knowledge / requirement(互不导入)。
package failure
