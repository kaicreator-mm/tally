package verification

// 本文件实现 PRD §8.7 Verification Execution Integrity(P1-26):把「一次验证
// verdict 到底绑定什么」与「同 candidate 的重复执行何时才算同一 protocol」
// 建成机械可判定的类型,使 rerun-until-pass 与 environment-shopping 无法把
// 多次尝试伪装成一个 verdict。
//
// 规则来源:
//   - §8.7.1:information-independent verdict 绑定四元组 —— immutable candidate
//     ChangeSet + verification binding revision + frozen execution protocol +
//     verification environment identity。ExecutionContext 是该四元组的值层投影,
//     Identity() 给出其稳定内容哈希;
//   - §8.7.2:同 candidate 的重复执行只有在 repetitions、stopping rule、
//     aggregation 在首次执行前冻结且与 observed outcome 无关时,才属于同一次
//     verification protocol。随机 seed 是冻结 protocol 的一部分;改变语义相关的
//     verification environment 不是「同一次 rerun」。ExecutionSession 在类型层面
//     强制:停止规则只能随构造冻结(无修改路径),每个 repetition 序号只能记录
//     一次(append-only),超出冻结次数、换 seed、换语义环境的执行一律拒绝进入
//     聚合(不静默合并);
//   - §8.7.2 禁止 FAIL → rerun → FAIL → rerun → PASS → cherry-pick PASS:
//     aggregation 只支持 all_pass(P1),任一 repetition FAIL 即 verdict FAIL。
//
// 边界(P1):
//   - 本文件不执行命令、不持久化 run 事实、不推导 coverage/effective
//     independence:命令执行归 runner/execution 任务,事实持久化归 evidence
//     存储任务,覆盖推导见 coverage_functional.go(§8.8.3:功能覆盖与
//     independence 覆盖是两个独立维度);
//   - 「停止规则先冻结后执行」由类型结构保证:session 只能从已冻结的 protocol
//     构造,构造后没有任何修改路径;跨 session 的历史 FAIL 是 append-only 事实,
//     由 §8.8.1 的 coverage 推导与 evidence 存储保证不被抹掉,本文件不重复该机制。

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// 哨兵错误:调用方用 errors.Is 分支,不做错误字符串匹配。均表示「该执行观察
// 不能计入当前冻结 protocol 的 verdict」,而非输入形态非法(形态非法用普通
// error 表达)。
var (
	// ErrRunBeyondStoppingRule — repetition 序号超出冻结次数,或全部 repetition
	// 已记录后仍试图追加执行(§8.7.2:禁止首次执行后扩大停止规则 rerun-until-pass)。
	ErrRunBeyondStoppingRule = errors.New("verification: 执行超出冻结停止规则的 repetition 次数")
	// ErrRunIndexAlreadyRecorded — 同一 repetition 序号已有 append-only 记录,
	// 禁止同序号重跑覆盖已观察事实(§8.7.2:禁止 cherry-pick)。
	ErrRunIndexAlreadyRecorded = errors.New("verification: 该 repetition 序号已有记录")
	// ErrSeedMismatch — 执行使用的随机 seed 与冻结 protocol 声明不一致
	// (§8.7.2:随机 seed 必须是冻结 protocol 的一部分)。
	ErrSeedMismatch = errors.New("verification: 执行 seed 与冻结 protocol 不一致")
	// ErrEnvironmentMismatch — 执行所在环境的语义身份与冻结 execution context
	// 不一致(§8.7.2:语义环境变化不是同一次 rerun;§8.7.1:verdict 绑定环境身份)。
	ErrEnvironmentMismatch = errors.New("verification: 执行环境身份与冻结 execution context 不一致")
	// ErrStoppingRuleIncomplete — 冻结停止规则尚未满足(还有未记录的 repetition),
	// 不能产生 verdict。
	ErrStoppingRuleIncomplete = errors.New("verification: 冻结停止规则未满足,不能产生 verdict")
)

// ToolVersion — 语义相关工具及其版本(参与环境身份哈希)。
type ToolVersion struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// EnvironmentIdentity — verification environment identity(PRD §8.7.1)。
//
// 语义字段(OS/Arch/Toolchain/Image/Tools/Env)全部参与身份哈希:任一语义字段
// 变化即产生不同的环境身份,旧 verdict 不能移植到新环境(§8.7.2)。Annotations
// 是非语义标注(如 runner 主机名、时间戳备注),供人读,不参与身份哈希 —— 改
// 标注不构成环境变化。
type EnvironmentIdentity struct {
	OS          string            `yaml:"os,omitempty"`
	Arch        string            `yaml:"arch,omitempty"`
	Toolchain   string            `yaml:"toolchain,omitempty"` // 语言/工具链版本,如 go1.23.4
	Image       string            `yaml:"image,omitempty"`     // 容器镜像引用
	Tools       []ToolVersion     `yaml:"tools,omitempty"`     // 语义相关工具及版本
	Env         []string          `yaml:"env,omitempty"`       // 语义相关环境变量,KEY=VALUE
	Annotations map[string]string `yaml:"-"`                   // 非语义标注,不参与身份哈希
}

// Validate 校验环境身份的确定性形态:至少一项语义字段;tools 无重复 name 且
// name/version 非空;env 项均为 KEY=VALUE 且 KEY 不重复。
func (e EnvironmentIdentity) Validate() error {
	if e.OS == "" && e.Arch == "" && e.Toolchain == "" && e.Image == "" && len(e.Tools) == 0 && len(e.Env) == 0 {
		return fmt.Errorf("verification: 环境身份缺少全部语义字段(os/arch/toolchain/image/tools/env 至少声明一项;§8.7.1:verdict 绑定环境身份)")
	}
	names := make(map[string]bool, len(e.Tools))
	for _, t := range e.Tools {
		if strings.TrimSpace(t.Name) == "" || strings.TrimSpace(t.Version) == "" {
			return fmt.Errorf("verification: 环境身份 tools 项缺少 name/version")
		}
		if names[t.Name] {
			return fmt.Errorf("verification: 环境身份 tools 项 name 重复: %s", t.Name)
		}
		names[t.Name] = true
	}
	keys := make(map[string]bool, len(e.Env))
	for _, kv := range e.Env {
		k, ok := envKey(kv)
		if !ok {
			return fmt.Errorf("verification: 环境身份 env 项必须是 KEY=VALUE,当前 %q", kv)
		}
		if keys[k] {
			return fmt.Errorf("verification: 环境身份 env 项 KEY 重复: %s", k)
		}
		keys[k] = true
	}
	return nil
}

// envKey 拆出 KEY=VALUE 的 KEY;KEY 非空才合法(值允许为空)。
func envKey(kv string) (string, bool) {
	i := strings.Index(kv, "=")
	if i <= 0 {
		return "", false
	}
	return kv[:i], true
}

// Hash 返回环境身份的规范化内容哈希,形如 sha256:<hex>:语义字段按结构体字段
// 顺序序列化,tools 按 name 排序、env 按字典序排序,因此与输入顺序无关;
// Annotations 不参与。任何语义字段变化都会改变哈希(AC3)。
func (e EnvironmentIdentity) Hash() (string, error) {
	if err := e.Validate(); err != nil {
		return "", err
	}
	data, err := yaml.Marshal(&canonicalEnvironment{
		OS:        e.OS,
		Arch:      e.Arch,
		Toolchain: e.Toolchain,
		Image:     e.Image,
		Tools:     sortedTools(e.Tools),
		Env:       sortedStrings(e.Env),
	})
	if err != nil {
		return "", fmt.Errorf("verification: 环境身份规范化序列化: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// canonicalEnvironment — 环境身份哈希专用的有序投影(无 map、无 Annotation)。
type canonicalEnvironment struct {
	OS        string        `yaml:"os,omitempty"`
	Arch      string        `yaml:"arch,omitempty"`
	Toolchain string        `yaml:"toolchain,omitempty"`
	Image     string        `yaml:"image,omitempty"`
	Tools     []ToolVersion `yaml:"tools,omitempty"`
	Env       []string      `yaml:"env,omitempty"`
}

func sortedTools(tools []ToolVersion) []ToolVersion {
	if len(tools) == 0 {
		return nil
	}
	cp := append([]ToolVersion(nil), tools...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].Name < cp[j].Name })
	return cp
}

func sortedStrings(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	cp := append([]string(nil), s...)
	sort.Strings(cp)
	return cp
}

// clone 返回深拷贝;session 冻结环境身份时用它与调用方对象脱钩。
func (e EnvironmentIdentity) clone() EnvironmentIdentity {
	e.Tools = append([]ToolVersion(nil), e.Tools...)
	e.Env = append([]string(nil), e.Env...)
	if e.Annotations != nil {
		a := make(map[string]string, len(e.Annotations))
		for k, v := range e.Annotations {
			a[k] = v
		}
		e.Annotations = a
	}
	return e
}

// ExecutionContext — §8.7.1 verdict 绑定四元组的值层投影:
// verification binding revision(ContractRevision+ContractHash+BindingID)、
// immutable candidate(CandidateID)、frozen execution protocol(Protocol)、
// verification environment identity(Environment)。Identity() 返回四元组的
// 稳定内容哈希;两个 execution context 身份哈希相同 ⇔ 协议与环境完全相同,
// 这是「能否并入同一次 verdict」的机械判据(AC4)。
type ExecutionContext struct {
	ContractRevision int
	ContractHash     string            // 冻结 VerificationContract 内容哈希(sha256:<hex>)
	BindingID        string            // 契约中的 binding id
	CandidateID      string            // immutable candidate ChangeSet id
	Protocol         ExecutionProtocol // 冻结执行协议(重复次数/聚合/seed/命令)
	Environment      EnvironmentIdentity
}

// Validate 校验 execution context:契约 revision/哈希形态、binding 与 candidate
// 非空、执行协议已冻结(§8.7.1:没有冻结 protocol 就没有可绑定的 verdict)、
// 环境身份形态合法。
func (x ExecutionContext) Validate() error {
	if !sha256HashRe.MatchString(x.ContractHash) {
		return fmt.Errorf("verification: contract_hash 必须形如 sha256:<64 位十六进制>(§8.7.1:verdict 绑定冻结契约 revision),当前 %q", x.ContractHash)
	}
	if x.ContractRevision < 1 {
		return fmt.Errorf("verification: contract_revision 必须 >= 1(当前 %d)", x.ContractRevision)
	}
	if strings.TrimSpace(x.BindingID) == "" {
		return fmt.Errorf("verification: binding_id 不能为空(verdict 绑定具体 verification binding revision)")
	}
	if strings.TrimSpace(x.CandidateID) == "" {
		return fmt.Errorf("verification: candidate_id 不能为空(§8.7.1:verdict 绑定 immutable candidate ChangeSet)")
	}
	if x.Protocol.Repetitions < 1 {
		return fmt.Errorf("verification: execution_protocol.repetitions 必须 >= 1(§8.7.2:repetitions 必须在首次执行前冻结)")
	}
	if x.Protocol.Aggregation != AggregationAllPass {
		return fmt.Errorf("verification: execution_protocol.aggregation 只支持 all_pass(P1,§8.7.2),当前 %q", x.Protocol.Aggregation)
	}
	if err := x.Protocol.Command.Validate(); err != nil {
		return fmt.Errorf("verification: execution_protocol: %w", err)
	}
	return x.Environment.Validate()
}

// Identity 返回 execution context 的稳定内容哈希(sha256:<hex>)。冻结 protocol
// 与环境身份以各自内容哈希参与,因此 seed 或任一语义环境字段变化都会改变
// execution context 身份(AC2/AC3)。
func (x ExecutionContext) Identity() (string, error) {
	if err := x.Validate(); err != nil {
		return "", err
	}
	protoHash, err := protocolHash(&x.Protocol)
	if err != nil {
		return "", err
	}
	envHash, err := x.Environment.Hash()
	if err != nil {
		return "", err
	}
	data, err := yaml.Marshal(&canonicalExecutionContext{
		ContractRevision: x.ContractRevision,
		ContractHash:     x.ContractHash,
		BindingID:        x.BindingID,
		CandidateID:      x.CandidateID,
		ProtocolHash:     protoHash,
		EnvironmentHash:  envHash,
	})
	if err != nil {
		return "", fmt.Errorf("verification: execution context 规范化序列化: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// canonicalExecutionContext — execution context 哈希专用的有序投影。
type canonicalExecutionContext struct {
	ContractRevision int    `yaml:"contract_revision"`
	ContractHash     string `yaml:"contract_hash"`
	BindingID        string `yaml:"binding_id"`
	CandidateID      string `yaml:"candidate_id"`
	ProtocolHash     string `yaml:"protocol_hash"`
	EnvironmentHash  string `yaml:"environment_hash"`
}

// protocolHash 返回冻结执行协议的规范化内容哈希(§8.7.1)。Seed 是协议字段,
// 改 seed 即改变协议身份(AC2);Env/Argv 保持声明顺序(语义上有序)。
func protocolHash(p *ExecutionProtocol) (string, error) {
	data, err := yaml.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("verification: 执行协议规范化序列化: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// clone 返回深拷贝,使冻结后的 context 与调用方对象彻底脱钩。
func (x ExecutionContext) clone() ExecutionContext {
	x.Protocol.Command = cloneCommandSpec(x.Protocol.Command)
	x.Environment = x.Environment.clone()
	return x
}

// StoppingRule — 冻结停止规则的只读投影(§8.7.2):固定 repetition 次数、
// 聚合方式与随机 seed。它与 observed outcome 无关:只能随 ExecutionSession
// 构造冻结,之后没有任何修改路径。
type StoppingRule struct {
	Repetitions int
	Aggregation Aggregation
	Seed        string // 空 = protocol 声明无随机性;一旦冻结,执行的 seed 必须与之精确一致
}

// RunObservation — 单次 repetition 执行的观察事实(§8.7.2)。Status 复用
// EvidenceStatus(pass|fail);Seed 与 Environment 是该次执行实际使用的随机源
// 与实际所在环境,由 session 与冻结 protocol 精确比对。
type RunObservation struct {
	Index       int                 // 1 起始的 repetition 序号,必须落在冻结停止规则内
	Status      EvidenceStatus      // pass | fail
	Seed        string              // 本次执行实际使用的 seed;必须等于冻结 seed
	Environment EnvironmentIdentity // 本次执行实际所在的验证环境
}

func (r RunObservation) validate() error {
	switch r.Status {
	case EvidenceStatusPass, EvidenceStatusFail:
	default:
		return fmt.Errorf("verification: repetition %d status 非法: %q(只支持 pass|fail)", r.Index, r.Status)
	}
	return nil
}

// ExecutionVerdict — 完整执行冻结停止规则后聚合出的单次验证判定(§8.7.1/
// §8.7.2)。它携带完整四元组身份哈希,使 verdict 与 immutable candidate、
// binding revision、冻结 protocol、环境身份可机械互相追溯;不满足停止规则就
// 不存在 verdict。
type ExecutionVerdict struct {
	ContractRevision int
	ContractHash     string
	BindingID        string
	CandidateID      string
	ProtocolHash     string
	EnvironmentHash  string
	ContextHash      string // 四元组 execution context 身份(ExecutionContext.Identity)

	Aggregation Aggregation    // 冻结的聚合方式(P1 仅 all_pass)
	Status      EvidenceStatus // 聚合后的总判定:任一 repetition FAIL 即 FAIL
	Repetitions int            // 冻结的 repetition 次数
	Failures    int            // FAIL 的 repetition 数
	RunStatuses []EvidenceStatus
}

// ExecutionSession — 首次执行前构造、之后只读的停止规则载体与 run 记录器。
//
// 构造时深拷贝并冻结 execution context(含 protocol 与环境身份),据此机械
// 强制(§8.7.2):
//   - 每个 repetition 序号至多记录一次,记录总数不能超过冻结次数 —— 停止规则
//     不可能因 observed outcome 而扩大,同序号重跑覆盖 FAIL 被拒绝;
//   - 每次 run 的 seed 必须与冻结 seed 精确一致 —— 随机性只能来自冻结 protocol;
//   - 每次 run 的环境身份哈希必须与冻结环境身份一致 —— 语义环境不同的执行
//     被显式拒绝,绝不静默并入聚合(AC4);
//   - aggregation 为冻结的 all_pass:任一 FAIL 即 verdict FAIL,cherry-pick
//     无路可走。
type ExecutionSession struct {
	ctx       ExecutionContext // 构造时冻结的深拷贝
	idHash    string
	protoHash string
	envHash   string
	runs      map[int]RunObservation // repetition 序号 → append-only 观察
}

// NewExecutionSession 冻结 execution context 并创建 session。这是停止规则的
// 唯一入口:session 存在后规则不可变,run 只能记录进已构造的 session,因此
// 「先冻结、后执行」是类型结构保证,不是约定。校验失败不产生 session。
func NewExecutionSession(ctx ExecutionContext) (*ExecutionSession, error) {
	if err := ctx.Validate(); err != nil {
		return nil, err
	}
	frozen := ctx.clone()
	id, err := frozen.Identity()
	if err != nil {
		return nil, fmt.Errorf("verification: 计算 execution context 身份: %w", err)
	}
	protoHash, err := protocolHash(&frozen.Protocol)
	if err != nil {
		return nil, fmt.Errorf("verification: 计算执行协议哈希: %w", err)
	}
	envHash, err := frozen.Environment.Hash()
	if err != nil {
		return nil, fmt.Errorf("verification: 计算环境身份哈希: %w", err)
	}
	return &ExecutionSession{
		ctx:       frozen,
		idHash:    id,
		protoHash: protoHash,
		envHash:   envHash,
		runs:      make(map[int]RunObservation, frozen.Protocol.Repetitions),
	}, nil
}

// StoppingRule 返回构造时冻结的停止规则(§8.7.2)。
func (s *ExecutionSession) StoppingRule() StoppingRule {
	return StoppingRule{
		Repetitions: s.ctx.Protocol.Repetitions,
		Aggregation: s.ctx.Protocol.Aggregation,
		Seed:        s.ctx.Protocol.Seed,
	}
}

// ContextHash 返回冻结 execution context 的身份哈希(§8.7.1 四元组)。
func (s *ExecutionSession) ContextHash() string { return s.idHash }

// Complete 报告冻结停止规则是否已被完整执行(全部 repetition 均已记录)。
func (s *ExecutionSession) Complete() bool {
	return len(s.runs) == s.ctx.Protocol.Repetitions
}

// RecordedRuns 返回已记录的 run 观察(按 repetition 序号排序的深拷贝投影)。
func (s *ExecutionSession) RecordedRuns() []RunObservation {
	idx := make([]int, 0, len(s.runs))
	for i := range s.runs {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	out := make([]RunObservation, 0, len(idx))
	for _, i := range idx {
		r := s.runs[i]
		r.Environment = r.Environment.clone()
		out = append(out, r)
	}
	return out
}

// Record 记录一次 repetition 的执行观察。观察与冻结 execution context 不一致
// (序号越界、序号重复、seed 不一致、语义环境不一致)时返回哨兵错误且不产生
// 任何记录 —— 外来执行不能静默并入聚合(AC4)。合法记录 append-only。
func (s *ExecutionSession) Record(run RunObservation) error {
	if err := run.validate(); err != nil {
		return err
	}
	total := s.ctx.Protocol.Repetitions
	if run.Index < 1 || run.Index > total {
		return fmt.Errorf("%w: repetition %d 超出冻结的 %d 次(§8.7.2:禁止首次执行后扩大停止规则)", ErrRunBeyondStoppingRule, run.Index, total)
	}
	if _, ok := s.runs[run.Index]; ok {
		return fmt.Errorf("%w: repetition %d 已有记录(append-only;同序号重跑覆盖观察事实即 cherry-pick,§8.7.2)", ErrRunIndexAlreadyRecorded, run.Index)
	}
	if run.Seed != s.ctx.Protocol.Seed {
		return fmt.Errorf("%w: 冻结 seed %q,观察 %q(§8.7.2:随机 seed 必须属于冻结 protocol)", ErrSeedMismatch, s.ctx.Protocol.Seed, run.Seed)
	}
	envHash, err := run.Environment.Hash()
	if err != nil {
		return err
	}
	if envHash != s.envHash {
		return fmt.Errorf("%w: 冻结环境身份 %s,观察 %s(§8.7.2:语义环境变化不是同一次 rerun)", ErrEnvironmentMismatch, s.envHash, envHash)
	}
	run.Environment = run.Environment.clone()
	s.runs[run.Index] = run
	return nil
}

// Verdict 在冻结停止规则被完整执行后聚合出判定。P1 仅 all_pass:任一
// repetition FAIL 即 verdict FAIL(§8.7.2 禁止 cherry-pick PASS);未满足停止
// 规则(存在未记录的 repetition)返回 ErrStoppingRuleIncomplete —— 部分执行
// 没有可声明的 verdict。
func (s *ExecutionSession) Verdict() (ExecutionVerdict, error) {
	total := s.ctx.Protocol.Repetitions
	if len(s.runs) < total {
		return ExecutionVerdict{}, fmt.Errorf("%w: 已记录 %d/%d 次 repetition(§8.7.2:verdict 只能来自对冻结停止规则的完整执行)", ErrStoppingRuleIncomplete, len(s.runs), total)
	}
	statuses := make([]EvidenceStatus, 0, total)
	failures := 0
	for i := 1; i <= total; i++ {
		st := s.runs[i].Status
		if st == EvidenceStatusFail {
			failures++
		}
		statuses = append(statuses, st)
	}
	status := EvidenceStatusPass
	if failures > 0 {
		status = EvidenceStatusFail
	}
	return ExecutionVerdict{
		ContractRevision: s.ctx.ContractRevision,
		ContractHash:     s.ctx.ContractHash,
		BindingID:        s.ctx.BindingID,
		CandidateID:      s.ctx.CandidateID,
		ProtocolHash:     s.protoHash,
		EnvironmentHash:  s.envHash,
		ContextHash:      s.idHash,
		Aggregation:      s.ctx.Protocol.Aggregation,
		Status:           status,
		Repetitions:      total,
		Failures:         failures,
		RunStatuses:      statuses,
	}, nil
}
