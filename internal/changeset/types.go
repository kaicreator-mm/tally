// Package changeset 定义不可变的候选变更快照(ChangeSet)。
//
// ChangeSet 是被验证、评审与集成的固定对象(PRD §6.4.2、§15.3,INV-CHANGE-02):
// CodegenRun 是过程,ChangeSet 是快照;一个 Run may produce 0..1 个候选快照,
// 没有固定 candidate 的 Run 不产生 ChangeSet。
//
// 不变性约定:构造之后 API 层面不可变 —— 字段全部不导出、无 setter,
// 切片取值一律返回副本。候选身份由内容推导(identity.go),
// 可执行代码一变,身份必变。取代关系是独立的 append-only 事实
// (SupersededFact),绝不通过修改快照本体表达(PRD §15.3)。
package changeset

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"tally/internal/core"
)

// 哨兵错误:调用方用 errors.Is 判因,不做字符串匹配。
var (
	// ErrNoProject 缺少项目边界。
	ErrNoProject = errors.New("ChangeSet 缺少项目")
	// ErrNoBaseRevision 缺少固定的 base revision。
	ErrNoBaseRevision = errors.New("ChangeSet 缺少 base revision")
	// ErrNoCandidateIdentity 既没有 candidate revision 也没有可推导 patch 身份的改动文件。
	ErrNoCandidateIdentity = errors.New("ChangeSet 缺少 candidate revision 或 patch 身份")
	// ErrNoProducerRun 缺少产生本快照的 CodegenRun(追溯必须完整)。
	ErrNoProducerRun = errors.New("ChangeSet 缺少产生它的 CodegenRun")
	// ErrInvalidFile 改动文件条目不合法(如空路径)。
	ErrInvalidFile = errors.New("ChangeSet 含无效的文件条目")
	// ErrBadSupersede 取代事实不合法。
	ErrBadSupersede = errors.New("无效的 ChangeSet 取代事实")
)

// ChangeSetID 标识一个不可变候选快照。
// 由 project + base + candidate/patch 身份 + producer run 推导,可重算、确定。
type ChangeSetID string

// SupersedeReason 是取代原因,取值固定于 PRD §15.3。
type SupersedeReason string

const (
	// SupersedeEscalation runtime/model 升级终止旧 Run 后重做。
	SupersedeEscalation SupersedeReason = "escalation"
	// SupersedeRework 返工重做。
	SupersedeRework SupersedeReason = "rework"
	// SupersedeSpecDefect 规格缺陷导致的重做。
	SupersedeSpecDefect SupersedeReason = "spec_defect"
)

// Hunk 是一个文件内一段改动的行坐标(统一 diff 风格,0 值表示不涉及)。
// 纯值类型:不含内容,只描述改动落点。
type Hunk struct {
	OldStart int
	OldLines int
	NewStart int
	NewLines int
}

// ChangedFile 是候选快照里的一个改动文件:路径 + 改后完整内容 + 是否可执行代码 + hunk 落点。
// 与 internal/core.FileChange 同理用完整内容而不是 patch 引擎;
// 值类型使外部拿到的是副本,改不动快照内部。
type ChangedFile struct {
	Path       string
	Content    string
	Executable bool
	Hunks      []Hunk
}

// Spec 是 New 的输入。New 不保留对 Spec 及其内部切片的引用。
type Spec struct {
	// Project 项目边界。
	Project core.ProjectID
	// BaseRevision 固定的 base revision(如 commit SHA)。
	BaseRevision string
	// CandidateRevision 固定的 candidate revision(如 commit SHA);
	// 与 Files 至少提供一个 —— PRD §6.4.2「candidate_commit 或 patch hash」。
	CandidateRevision string
	// Files 改动文件。非空时由内容推导 patch 身份(见 identity.go)。
	Files []ChangedFile
	// Entities 受影响的 SoftwareEntity(PRD §6.4.2,可推导时;可为空)。
	Entities []string
	// ProducerRun 产生本快照的 CodegenRun。
	ProducerRun string
	// CreatedAt 快照固定时刻;零值时取当下(UTC)。
	CreatedAt time.Time
}

// ChangeSet 是不可变的候选变更快照(INV-CHANGE-02:验证、评审与集成的固定单元)。
// 固定:base revision、candidate revision/patch 身份、改动文件/hunks、created_at、producer run(PRD §15.3)。
// 字段全部不导出:构造之后唯一入口是取值方法,快照不会在验证/Review 中原地变形。
// 取代关系不在本类型上:用独立的 SupersededFact 表达,UI 可派生 superseded_by。
type ChangeSet struct {
	id          ChangeSetID
	project     core.ProjectID
	base        string
	candidate   string // 可为空:只有 patch 身份时
	patchID     string // Files 非空时由内容推导;否则为空
	files       []ChangedFile
	entities    []string
	producerRun string
	createdAt   time.Time
}

// New 构造并固定一个候选快照。不满足必填项时返回哨兵错误。
//
// 必填:project、base revision、producer run,以及 candidate revision 与
// 改动文件二者至少其一(候选身份)。改动文件按路径排序归一,输入切片被复制。
func New(spec Spec) (*ChangeSet, error) {
	if spec.Project == "" {
		return nil, ErrNoProject
	}
	if strings.TrimSpace(spec.BaseRevision) == "" {
		return nil, ErrNoBaseRevision
	}
	if strings.TrimSpace(spec.ProducerRun) == "" {
		return nil, ErrNoProducerRun
	}

	files := make([]ChangedFile, 0, len(spec.Files))
	for _, f := range spec.Files {
		if strings.TrimSpace(f.Path) == "" {
			return nil, fmt.Errorf("%w: 文件路径为空", ErrInvalidFile)
		}
		cf := ChangedFile{Path: f.Path, Content: f.Content, Executable: f.Executable}
		if len(f.Hunks) > 0 {
			cf.Hunks = append([]Hunk(nil), f.Hunks...)
		}
		files = append(files, cf)
	}
	// 文件顺序不是语义,归一成按路径排序:身份与取值都确定。
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	patchID := ""
	if len(files) > 0 {
		patchID = patchIdentity(files)
	}
	candidate := spec.CandidateRevision
	if strings.TrimSpace(candidate) == "" && patchID == "" {
		return nil, ErrNoCandidateIdentity
	}

	createdAt := spec.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	return &ChangeSet{
		id:          deriveID(spec.Project, spec.BaseRevision, candidate, patchID, spec.ProducerRun),
		project:     spec.Project,
		base:        spec.BaseRevision,
		candidate:   candidate,
		patchID:     patchID,
		files:       files,
		entities:    append([]string(nil), spec.Entities...),
		producerRun: spec.ProducerRun,
		createdAt:   createdAt.UTC(),
	}, nil
}

// ID 是快照自身身份,由不可变输入推导,可重算。
func (c *ChangeSet) ID() ChangeSetID { return c.id }

// Project 是项目边界。
func (c *ChangeSet) Project() core.ProjectID { return c.project }

// BaseRevision 是固定的 base revision。
func (c *ChangeSet) BaseRevision() string { return c.base }

// CandidateRevision 是固定的 candidate revision;只有 patch 身份时为空。
func (c *ChangeSet) CandidateRevision() string { return c.candidate }

// PatchIdentity 是由改动文件内容推导的 patch 身份;没有改动文件时为空。
func (c *ChangeSet) PatchIdentity() string { return c.patchID }

// Files 返回改动文件(按路径排序)。返回副本:修改返回值不影响快照本体。
func (c *ChangeSet) Files() []ChangedFile {
	out := make([]ChangedFile, len(c.files))
	for i, f := range c.files {
		cf := f
		if len(f.Hunks) > 0 {
			cf.Hunks = append([]Hunk(nil), f.Hunks...)
		}
		out[i] = cf
	}
	return out
}

// Entities 返回受影响的 SoftwareEntity。返回副本。
func (c *ChangeSet) Entities() []string {
	return append([]string(nil), c.entities...)
}

// ProducerRun 是产生本快照的 CodegenRun。
func (c *ChangeSet) ProducerRun() string { return c.producerRun }

// CreatedAt 是快照固定时刻(UTC)。
func (c *ChangeSet) CreatedAt() time.Time { return c.createdAt }

// SupersededFact 记录 old 被 new 取代的 append-only 事实(PRD §15.3)。
// 它是独立事实值,不是 ChangeSet 上的可变字段:取代不改快照本体,
// 后续重做产生新的 ChangeSet,取代关系只追加新事实。
type SupersededFact struct {
	Old    ChangeSetID
	New    ChangeSetID
	Reason SupersedeReason
	At     time.Time
}

// NewSupersededFact 构造一条取代事实。old/new 必须非空且不同,原因必须是固定取值之一。
func NewSupersededFact(oldID, newID ChangeSetID, reason SupersedeReason, at time.Time) (SupersededFact, error) {
	if oldID == "" || newID == "" {
		return SupersededFact{}, fmt.Errorf("%w: old/new 不能为空", ErrBadSupersede)
	}
	if oldID == newID {
		return SupersededFact{}, fmt.Errorf("%w: old 与 new 相同", ErrBadSupersede)
	}
	switch reason {
	case SupersedeEscalation, SupersedeRework, SupersedeSpecDefect:
	default:
		return SupersededFact{}, fmt.Errorf("%w: 未知原因 %q", ErrBadSupersede, reason)
	}
	if at.IsZero() {
		at = time.Now()
	}
	return SupersededFact{Old: oldID, New: newID, Reason: reason, At: at.UTC()}, nil
}
