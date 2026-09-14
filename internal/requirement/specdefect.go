package requirement

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"tally/internal/core"
)

// SpecDefect 出口(PRD §20、INV-VER-04):Agent 在诊断中认为「需求本身有问题」时,
// 不得改 hard gate 绕过验证,只能产出 SpecDefectProposal 进 review。
//
// 本文件是 failure.SpecDefectSink 的 requirement 侧实现。两边不互相导入
// (业务包互不导入),靠同形方法守住形状 —— 上层(exec/worker 接线)把它
// 注入 failure.Deps.SpecSink:
//
//	ReportSpecDefect(ctx, projectID, clauseID, ambiguity string) error
//
// 语义与 failure/ports.go 的接口注释一一对应:
//   - 确定性闸门(铁律 1):先校验条款真的存在于该项目的需求文档,不存在 →
//     返回错误,提案不成立 —— LLM 编不出不存在的条款;
//   - 记录是确定性信号:提案钩在提案时条款的 ContentHash 上(INV-4 挂钩),
//     以 append-only 方式留痕进 git(同 interview.Reject 的驳回留痕机制);
//   - 失效传播不引入新机制:owner 决策(specification update / verification
//     update / reject)产出需求新 revision 后,内容哈希对不上,挂钩经
//     CheckReference / ClassifyReferences 自动显示「依据已变更」;
//   - 记录的只是提案:状态恒为 NEEDS_REVIEW,决定权在 owner。这里不写任何
//     VERIFIED / 生命周期状态(failure 侧的 spec_defect 归类是它自己的索引面)。

// SpecDefectFile 是 spec-defect 提案留痕文件(在项目 git 仓库里,append-only)。
const SpecDefectFile = ".tally/specdefects.md"

// SpecDefectNeedsReview 是提案进入 review 的状态(PRD §20 流程)。本包只写
// 这个状态;决策(spec 更新 / 验证更新 / 驳回)由 owner 做,不经这里。
const SpecDefectNeedsReview = "NEEDS_REVIEW"

// SpecDefectRecord 是一条已留痕的 spec-defect 提案。
type SpecDefectRecord struct {
	Project    string          // 提案所属项目
	ClauseID   string          // 指向哪条条款(如 REQ-1)
	Hash       core.ClauseHash // 提案时该条款的内容哈希(INV-4 挂钩)
	Ambiguity  string          // Agent 声称的模糊点 —— 只是提案内容,不是事实
	Status     string          // 目前恒为 SpecDefectNeedsReview
	ReportedAt time.Time       // 提案时间(留痕时刻)
}

// Reference 返回这条提案的挂钩,直接喂给 ClassifyReferences / CheckReference
// 对照需求当前版本:哈希失配 = 依据已变更(失效传播的全部机制)。
func (r SpecDefectRecord) Reference() Reference {
	return Reference{ClauseID: r.ClauseID, Hash: r.Hash}
}

// SpecDefectSink 把 requirement 的「条款校验 + 提案留痕 + 失效挂钩」能力
// 收成一个可注入 failure 的实现。RepoPath 是项目 git 仓库根;Identity 是
// 留痕提交身份(同签署身份的注入方式,测试不依赖真实 git 配置)。
type SpecDefectSink struct {
	RepoPath string
	Identity Identity
}

// ReportSpecDefect 校验条款存在后,把 spec-defect 提案以 append-only 方式
// 记进 git 留痕文件并提交。
//
// 条款不存在(或项目没有需求文档)→ 返回错误 —— 这是确定性闸门:LLM 只能
// 提案,编不出不存在的条款(铁律 1)。同一条款同一内容已有待审提案时不重复
// 追加(同一输入 → 同一留痕结果,确定性幂等)。
//
// 注意:本方法只留痕,不改需求正文、不写任何验证/生命周期状态;LLM 的指控
// 原样记录为「待审提案」,成立与否由 owner 决策(NEEDS_REVIEW 之后是人的路)。
func (s SpecDefectSink) ReportSpecDefect(ctx context.Context, projectID, clauseID, ambiguity string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if projectID == "" {
		return fmt.Errorf("spec-defect 提案:缺少项目标识")
	}
	if clauseID == "" {
		return fmt.Errorf("spec-defect 提案:缺少条款编号")
	}
	if s.Identity.Name == "" || s.Identity.Email == "" {
		return fmt.Errorf("spec-defect 提案:缺少提交身份(Name/Email)")
	}

	// 确定性闸门:条款必须真实存在(铁律 1 的落点)。
	clause, err := s.lookupClause(projectID, clauseID)
	if err != nil {
		return err
	}
	hash := ContentHash(clause) // 挂钩:指向提案时的这一版条款(INV-4)

	existing, err := LoadSpecDefects(s.RepoPath)
	if err != nil {
		return err
	}
	for _, r := range existing {
		if r.Status == SpecDefectNeedsReview && r.Project == projectID &&
			r.ClauseID == clauseID && r.Hash == hash {
			return nil // 同一缺陷已有人提案待审:不重复追加
		}
	}

	when := s.Identity.When
	if when.IsZero() {
		when = time.Now()
	}

	fsPath := filepath.Join(s.RepoPath, filepath.FromSlash(SpecDefectFile))
	prev, err := os.ReadFile(fsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("读 spec-defect 留痕: %w", err)
	}
	var b strings.Builder
	b.Write(prev)
	if len(prev) > 0 && !strings.HasSuffix(string(prev), "\n") {
		b.WriteString("\n")
	}
	if len(prev) == 0 {
		b.WriteString("# Spec defect 提案归档\n\n" +
			"Agent 在诊断中认为「需求本身有问题」时的正式提案(PRD §20,INV-VER-04)。\n" +
			"LLM 只提案;条款存在与否已过确定性闸门。每条提案钩在提案时条款的内容哈希上\n" +
			"(INV-4):owner 决策产出需求新 revision 后,哈希失配即显示「依据已变更」。\n" +
			"本文件只追加;状态恒为 NEEDS_REVIEW,决定权在 owner,不写任何验证/生命周期状态。\n\n")
	}
	fmt.Fprintf(&b, "## 提案 %s\n", when.Format(time.RFC3339))
	fmt.Fprintf(&b, "- 状态: %s\n", SpecDefectNeedsReview)
	fmt.Fprintf(&b, "- 项目: %s\n", projectID)
	fmt.Fprintf(&b, "- 条款: %s\n", clauseID)
	fmt.Fprintf(&b, "clause-hash: %s %s\n", clauseID, hash)
	fmt.Fprintf(&b, "- 模糊点: %s\n\n", strings.TrimSpace(ambiguity))

	msg := fmt.Sprintf("specdefect: 提案 %s 需求缺陷待审(%s)\n\nclause-hash: %s %s\n", clauseID, projectID, clauseID, hash)
	if _, err := commitFile(s.RepoPath, SpecDefectFile, []byte(b.String()), msg, s.Identity, nil); err != nil {
		return fmt.Errorf("提交 spec-defect 留痕: %w", err)
	}
	return nil
}

// lookupClause 在项目需求文档(.tally/requirements/<projectID>/*.md)中定位
// 条款。项目没有需求文档或条款不存在 → 错误(提案不成立)。
// 文件按名字序、条款按文档序取第一个命中:编号重复属 L1 缺口,这里确定性取一。
func (s SpecDefectSink) lookupClause(projectID, clauseID string) (Clause, error) {
	dir := filepath.Join(s.RepoPath, ".tally", "requirements", projectID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return Clause{}, fmt.Errorf("spec-defect 提案:项目 %s 没有需求文档,条款 %s 无从校验", projectID, clauseID)
		}
		return Clause{}, fmt.Errorf("spec-defect 提案:读需求目录 %s: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		doc, err := ParseFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return Clause{}, fmt.Errorf("spec-defect 提案: %w", err)
		}
		for _, c := range doc.Clauses {
			if c.ID == clauseID {
				return c, nil
			}
		}
	}
	return Clause{}, fmt.Errorf("spec-defect 提案:条款 %s 不存在于项目 %s 的需求文档中,提案不成立", clauseID, projectID)
}

// LoadSpecDefects 读回留痕文件里的全部提案(按记录顺序)。
// 文件不存在 = 还没有任何提案,返回空切片不报错;审查面(以及失效对照)
// 用 Record.Reference() 喂 ClassifyReferences 即得「依据是否已变更」。
func LoadSpecDefects(repoPath string) ([]SpecDefectRecord, error) {
	b, err := os.ReadFile(filepath.Join(repoPath, filepath.FromSlash(SpecDefectFile)))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读 spec-defect 留痕: %w", err)
	}
	return parseSpecDefects(string(b)), nil
}

// 留痕文件各字段的识别规则。宽容解析:不认识的行忽略,格式问题不影响已记录内容。
var (
	reSDHeading   = regexp.MustCompile(`^##\s*提案\s*(\S*)\s*$`)
	reSDStatus    = regexp.MustCompile(`^[-*]\s*状态\s*[:：]\s*(.+?)\s*$`)
	reSDProject   = regexp.MustCompile(`^[-*]\s*项目\s*[:：]\s*(.+?)\s*$`)
	reSDClause    = regexp.MustCompile(`^[-*]\s*条款\s*[:：]\s*(.+?)\s*$`)
	reSDHash      = regexp.MustCompile(`^clause-hash:\s*(\S+)\s+(\S+)\s*$`)
	reSDAmbiguity = regexp.MustCompile(`^[-*]\s*模糊点\s*[:：]\s*(.*)$`)
)

// parseSpecDefects 解析留痕 markdown:每个 `## 提案` 标题开启一条记录,
// clause-hash 行给出挂钩(条款编号 + 内容哈希,同签署提交信息的约定)。
func parseSpecDefects(src string) []SpecDefectRecord {
	var out []SpecDefectRecord
	var cur *SpecDefectRecord
	flush := func() {
		if cur != nil {
			out = append(out, *cur)
			cur = nil
		}
	}
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case reSDHeading.MatchString(trimmed):
			flush()
			cur = &SpecDefectRecord{}
			if m := reSDHeading.FindStringSubmatch(trimmed); m != nil && m[1] != "" {
				if when, err := time.Parse(time.RFC3339, m[1]); err == nil {
					cur.ReportedAt = when
				}
			}
		case cur == nil:
			continue
		case reSDHash.MatchString(trimmed):
			m := reSDHash.FindStringSubmatch(trimmed)
			cur.ClauseID = m[1]
			cur.Hash = core.ClauseHash(m[2])
		case reSDStatus.MatchString(trimmed):
			cur.Status = reSDStatus.FindStringSubmatch(trimmed)[1]
		case reSDProject.MatchString(trimmed):
			cur.Project = reSDProject.FindStringSubmatch(trimmed)[1]
		case reSDClause.MatchString(trimmed):
			if cur.ClauseID == "" {
				cur.ClauseID = reSDClause.FindStringSubmatch(trimmed)[1]
			}
		case reSDAmbiguity.MatchString(trimmed):
			cur.Ambiguity = reSDAmbiguity.FindStringSubmatch(trimmed)[1]
		}
	}
	flush()
	return out
}
