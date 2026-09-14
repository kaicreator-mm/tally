package agentruntime

// ObservedIdentity 是 runtime 对一个已开启会话实际执行身份的自报观察值
// (P0-01;Decision Lock 15:实际身份 != 期望配置,两者必须分开携带、分开落库)。
//
// 语义边界(改动前必读):
//
//   - 全部字段都是 runtime 实际报告的观察事实;零值表示「未观察」,
//     不是「空身份」。调用方绝不能用期望配置(StartRequest.ModelProfile、
//     RuntimeDescriptor 等)补齐未观察字段 —— 把期望伪装成实际即伪造观察。
//   - 与 RuntimeDescriptor 的分工:Descriptor 是构造期自述(适配器版本 +
//     声明能力,本身是 claim);ObservedIdentity 是会话建立后 runtime 现场
//     报告的实际身份(如实际 CLI 版本、实际生效模型、实际会话句柄)。
//     适配器版本与 CLI 版本是两个事实,后者才属于这里。
//   - 本类型只定义观察形状,不定义如何获得;获得经 IdentityReporter 可选
//     接口(见下)。它不进协议核心六件事(PRD §10.2),也不新增 capability
//     字符串 —— PRD §10.3 的能力词汇表是契约,不得扩充。
type ObservedIdentity struct {
	// RuntimeName 是观察到的 runtime 自报标识;未观察为空。
	RuntimeName string
	// RuntimeVersion 是观察到的 runtime/CLI 实际版本;未观察为空。
	RuntimeVersion string
	// Model 是本会话实际生效的模型;未观察为空。它与请求的
	// StartRequest.ModelProfile 是不同的值,禁止互相回填。
	Model string
	// SessionID 是 runtime 协议层实际给出的会话句柄;未观察为空。
	SessionID SessionID
}

// Observed 报告是否观察到任何身份事实。全零值 = 完全未观察(合法状态:
// runtime 不支持自报身份,或事实时刻尚未观察到该项)。
func (i ObservedIdentity) Observed() bool {
	return i.RuntimeName != "" || i.RuntimeVersion != "" || i.Model != "" || i.SessionID != ""
}

// IdentityReporter 是 Session 的可选扩展:能自报实际会话身份的 runtime 在其
// Session 实现上额外实现本接口。调用方(如 workflow)用类型断言获取:
//
//	if rep, ok := session.(IdentityReporter); ok { id = rep.Identity() }
//
// 能力安全(P0-01 AC3):不实现 = 完全未观察,调用方走零值降级路径,
// 不报错、不补齐 —— FakeRuntime 无需实现本接口。这里刻意不引入新的
// capability 协商字符串(PRD §10.3 词汇表冻结),可选性由 Go 接口断言表达。
type IdentityReporter interface {
	// Identity 返回该会话当前已观察到的实际身份快照。只读、非阻塞、可并发
	// 调用;尚未观察到的字段为零值。实现不得在调用时发起新的观察动作
	//(发现进程/网络请求等),只返回已建立的观察。
	Identity() ObservedIdentity
}
